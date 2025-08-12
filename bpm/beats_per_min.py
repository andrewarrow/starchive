#!/usr/bin/env python3
import argparse
import json
import numpy as np
import librosa
import warnings
import sys

# Suppress librosa warnings to stderr so they don't interfere with JSON output
warnings.filterwarnings('ignore', category=FutureWarning, module='librosa')
warnings.filterwarnings('ignore', category=UserWarning, module='librosa')

PITCH_CLASSES = ["C","C#","D","D#","E","F","F#","G","G#","A","A#","B"]

MAJOR_PROFILE = np.array([6.35,2.23,3.48,2.33,4.38,4.09,2.52,5.19,2.39,3.66,2.29,2.88], dtype=float)
MINOR_PROFILE = np.array([6.33,2.68,3.52,5.38,2.60,3.53,2.54,4.75,3.98,2.69,3.34,3.17], dtype=float)

def _to_float(x, default=np.nan):
    """Coerce tempo to a Python float (handles ndarray/0-d arrays)."""
    try:
        return float(np.atleast_1d(x)[0])
    except Exception:
        return float(default)


def estimate_bpm(y, sr, hop_length=512):
    """Improved BPM estimation with better onset detection and tempo analysis."""
    
    # Try multiple onset detection approaches
    onset_methods = []
    
    # Standard onset strength
    try:
        onset1 = librosa.onset.onset_strength(y=y, sr=sr, hop_length=hop_length)
        onset_methods.append(('standard', onset1))
    except:
        pass
    
    # Onset with percussive component
    try:
        y_percussive = librosa.effects.percussive(y)
        onset2 = librosa.onset.onset_strength(y=y_percussive, sr=sr, hop_length=hop_length)
        onset_methods.append(('percussive', onset2))
    except:
        pass
    
    # Complex domain onset
    try:
        onset3 = librosa.onset.onset_strength(y=y, sr=sr, hop_length=hop_length, feature=librosa.feature.spectral_centroid)
        onset_methods.append(('complex', onset3))
    except:
        pass
    
    if not onset_methods:
        return 120.0
    
    # Try beat tracking with each onset method
    all_tempos = []
    
    for method_name, onset_env in onset_methods:
        # Method 1: Standard beat tracking
        try:
            tempo, _ = librosa.beat.beat_track(y=y, sr=sr, onset_envelope=onset_env, hop_length=hop_length)
            tempo = _to_float(tempo)
            if np.isfinite(tempo) and tempo > 0:
                all_tempos.append(tempo)
        except:
            pass
        
        # Method 2: Tempo estimation with different aggregation
        try:
            tempo_arr = librosa.feature.rhythm.tempo(
                onset_envelope=onset_env, sr=sr, hop_length=hop_length,
                start_bpm=60, max_tempo=200, aggregate='mean'
            )
            tempo = _to_float(tempo_arr)
            if np.isfinite(tempo) and tempo > 0:
                all_tempos.append(tempo)
        except:
            pass
    
    if not all_tempos:
        return 120.0
    
    # Remove outliers using IQR method
    if len(all_tempos) > 2:
        q25 = np.percentile(all_tempos, 25)
        q75 = np.percentile(all_tempos, 75)
        iqr = q75 - q25
        lower_bound = q25 - 1.5 * iqr
        upper_bound = q75 + 1.5 * iqr
        all_tempos = [t for t in all_tempos if lower_bound <= t <= upper_bound]
    
    if not all_tempos:
        return 120.0
    
    # Handle common tempo multiples/divisions
    candidate_tempos = []
    for tempo in all_tempos:
        candidate_tempos.extend([tempo/2, tempo, tempo*2])
    
    # Filter to reasonable range
    candidate_tempos = [t for t in candidate_tempos if 60 <= t <= 200]
    
    if not candidate_tempos:
        return np.median(all_tempos)
    
    # Use mode-finding approach with bins
    hist, bins = np.histogram(candidate_tempos, bins=28, range=(60, 200))  # 5 BPM bins
    max_bin_idx = np.argmax(hist)
    bin_center = (bins[max_bin_idx] + bins[max_bin_idx + 1]) / 2
    
    # Find all tempos close to the most common bin
    tolerance = 7.0  # BPM
    close_tempos = [t for t in candidate_tempos if abs(t - bin_center) <= tolerance]
    
    return np.median(close_tempos) if close_tempos else np.median(all_tempos)

def estimate_key(y, sr):
    """Improved key estimation using multiple methods and harmonic analysis."""
    
    # Method 1: Standard chroma-based key detection
    chroma1 = librosa.feature.chroma_cqt(y=y, sr=sr, hop_length=512)
    chroma1_avg = chroma1.mean(axis=1)
    
    # Method 2: STFT-based chroma for comparison
    chroma2 = librosa.feature.chroma_stft(y=y, sr=sr, hop_length=512)
    chroma2_avg = chroma2.mean(axis=1)
    
    # Method 3: CQT chroma with different parameters
    chroma3 = librosa.feature.chroma_cqt(y=y, sr=sr, hop_length=1024, fmin=librosa.note_to_hz('C2'))
    chroma3_avg = chroma3.mean(axis=1)
    
    # Use weighted average of the three methods, with more weight on the first
    if (np.allclose(chroma1_avg.sum(), 0.0) and 
        np.allclose(chroma2_avg.sum(), 0.0) and 
        np.allclose(chroma3_avg.sum(), 0.0)):
        return "Unknown"
    
    # Normalize each chroma vector
    chroma_vectors = []
    weights = [0.5, 0.3, 0.2]  # Weight the first method more heavily
    
    for chroma_avg, weight in zip([chroma1_avg, chroma2_avg, chroma3_avg], weights):
        if not np.allclose(chroma_avg.sum(), 0.0):
            chroma_norm = chroma_avg / (np.linalg.norm(chroma_avg) + 1e-12)
            chroma_vectors.append((chroma_norm, weight))
    
    if not chroma_vectors:
        return "Unknown"
    
    # Compute weighted average chroma
    final_chroma = np.zeros(12)
    total_weight = 0
    for chroma_norm, weight in chroma_vectors:
        final_chroma += chroma_norm * weight
        total_weight += weight
    
    final_chroma /= total_weight
    final_chroma_norm = (final_chroma - final_chroma.mean()) / (np.linalg.norm(final_chroma) + 1e-12)

    # Enhanced key-finding with better profiles
    # Using Krumhansl-Schmuckler key profiles (more accurate)
    MAJOR_KS = np.array([6.35, 2.23, 3.48, 2.33, 4.38, 4.09, 2.52, 5.19, 2.39, 3.66, 2.29, 2.88])
    MINOR_KS = np.array([6.33, 2.68, 3.52, 5.38, 2.60, 3.53, 2.54, 4.75, 3.98, 2.69, 3.34, 3.17])
    
    best_keys = []
    correlations = []
    
    for shift in range(12):
        maj = np.roll(MAJOR_KS, shift)
        minr = np.roll(MINOR_KS, shift)
        maj_n = (maj - maj.mean()) / (np.linalg.norm(maj) + 1e-12)
        min_n = (minr - minr.mean()) / (np.linalg.norm(minr) + 1e-12)

        corr_maj = float(np.dot(final_chroma_norm, maj_n))
        corr_min = float(np.dot(final_chroma_norm, min_n))

        best_keys.append((f"{PITCH_CLASSES[shift]} major", corr_maj))
        best_keys.append((f"{PITCH_CLASSES[shift]} minor", corr_min))
        correlations.extend([corr_maj, corr_min])

    # Find the best key
    best_key, best_corr = max(best_keys, key=lambda x: x[1])
    
    # Additional validation: check if the correlation is strong enough
    correlation_threshold = np.mean(correlations) + 0.5 * np.std(correlations)
    if best_corr < correlation_threshold:
        # Try alternative approach using harmonic content
        return _estimate_key_harmonic_fallback(y, sr)
    
    return best_key

def _estimate_key_harmonic_fallback(y, sr):
    """Fallback key estimation using harmonic content analysis."""
    try:
        # Extract harmonic content
        y_harmonic = librosa.effects.harmonic(y)
        chroma = librosa.feature.chroma_cqt(y=y_harmonic, sr=sr)
        
        # Focus on stronger harmonic content
        chroma_strong = np.where(chroma > np.percentile(chroma, 75), chroma, 0)
        chroma_avg = chroma_strong.mean(axis=1)
        
        if np.allclose(chroma_avg.sum(), 0.0):
            return "C major"  # Default fallback
        
        # Find the strongest pitch class
        strongest_pitch = np.argmax(chroma_avg)
        
        # Simple heuristic: if the strongest pitch has strong perfect fifth, likely major
        fifth_pitch = (strongest_pitch + 7) % 12
        if chroma_avg[fifth_pitch] > 0.6 * chroma_avg[strongest_pitch]:
            return f"{PITCH_CLASSES[strongest_pitch]} major"
        else:
            return f"{PITCH_CLASSES[strongest_pitch]} minor"
            
    except:
        return "C major"

def main():
    ap = argparse.ArgumentParser(description="Estimate BPM and musical key from a WAV file.")
    ap.add_argument("wav_path", help="Path to .wav file")
    ap.add_argument("--sr", type=int, default=None, help="Target sample rate (default: use file's native)")
    ap.add_argument("--mono", action="store_true", help="Force mono (recommended)")
    args = ap.parse_args()

    y, sr = librosa.load(args.wav_path, sr=args.sr, mono=args.mono or True)

    bpm = estimate_bpm(y, sr)
    key = estimate_key(y, sr)

    result = {
        "bpm": round(bpm, 1),
        "key": key
    }
    
    print(json.dumps(result))

if __name__ == "__main__":
    main()