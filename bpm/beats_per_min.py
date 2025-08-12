#!/usr/bin/env python3
import argparse
import json
import numpy as np
import librosa

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
    """Improved BPM estimation using multiple methods and tempo disambiguation."""
    
    # Use multiple onset detection methods
    onset_env_complex = librosa.onset.onset_strength(y=y, sr=sr, hop_length=hop_length, feature=librosa.feature.spectral_centroid)
    onset_env_simple = librosa.onset.onset_strength(y=y, sr=sr, hop_length=hop_length)
    
    # Method 1: Beat tracking with complex onset
    try:
        tempo1, _ = librosa.beat.beat_track(
            y=y, sr=sr, onset_envelope=onset_env_complex, hop_length=hop_length
        )
        tempo1 = _to_float(tempo1)
    except:
        tempo1 = np.nan
    
    # Method 2: Beat tracking with simple onset  
    try:
        tempo2, _ = librosa.beat.beat_track(
            y=y, sr=sr, onset_envelope=onset_env_simple, hop_length=hop_length
        )
        tempo2 = _to_float(tempo2)
    except:
        tempo2 = np.nan
    
    # Method 3: Direct tempo estimation with wider range
    try:
        tempo3_arr = librosa.beat.tempo(
            onset_envelope=onset_env_simple, sr=sr, hop_length=hop_length,
            start_bpm=60, max_tempo=200, aggregate='mean'
        )
        tempo3 = _to_float(tempo3_arr)
    except:
        tempo3 = np.nan
    
    # Method 4: Autocorrelation-based tempo
    try:
        tempo4_arr = librosa.beat.tempo(
            onset_envelope=onset_env_complex, sr=sr, hop_length=hop_length,
            start_bpm=80, max_tempo=180, aggregate='harmonic_mean'
        )
        tempo4 = _to_float(tempo4_arr)
    except:
        tempo4 = np.nan
    
    # Collect all valid tempos
    tempos = [t for t in [tempo1, tempo2, tempo3, tempo4] if np.isfinite(t) and t > 0]
    
    if not tempos:
        return 120.0  # Default fallback
    
    # Handle tempo disambiguation (half-time/double-time)
    expanded_tempos = []
    for t in tempos:
        expanded_tempos.extend([t/2, t, t*2])
    
    # Find the most common tempo range
    tempo_ranges = {}
    for t in expanded_tempos:
        if 60 <= t <= 200:  # Reasonable tempo range
            key = round(t / 5) * 5  # Group into 5 BPM buckets
            if key not in tempo_ranges:
                tempo_ranges[key] = []
            tempo_ranges[key].append(t)
    
    if not tempo_ranges:
        return np.median(tempos)
    
    # Return the median of the most populated range
    best_range = max(tempo_ranges.keys(), key=lambda k: len(tempo_ranges[k]))
    return np.median(tempo_ranges[best_range])

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