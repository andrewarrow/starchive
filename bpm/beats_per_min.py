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
    onset_env = librosa.onset.onset_strength(y=y, sr=sr, hop_length=hop_length)
    
    tempo, beat_frames = librosa.beat.beat_track(
        y=y, sr=sr, onset_envelope=onset_env, hop_length=hop_length
    )
    tempo = _to_float(tempo)
    
    # Fallback if tracker didn't find anything sensible
    if not np.isfinite(tempo) or tempo <= 0:
        t_arr = librosa.beat.tempo(onset_envelope=onset_env, sr=sr, hop_length=hop_length, aggregate='mean')
        tempo = _to_float(t_arr)

    return tempo

def estimate_key(y, sr):
    chroma = librosa.feature.chroma_cqt(y=y, sr=sr)
    chroma_avg = chroma.mean(axis=1)
    if np.allclose(chroma_avg.sum(), 0.0):
        return "Unknown"

    chroma_norm = (chroma_avg - chroma_avg.mean()) / (np.linalg.norm(chroma_avg) + 1e-12)

    best_key = None
    best_corr = -np.inf
    for shift in range(12):
        maj = np.roll(MAJOR_PROFILE, shift)
        minr = np.roll(MINOR_PROFILE, shift)
        maj_n = (maj - maj.mean()) / (np.linalg.norm(maj) + 1e-12)
        min_n = (minr - minr.mean()) / (np.linalg.norm(minr) + 1e-12)

        corr_maj = float(np.dot(chroma_norm, maj_n))
        corr_min = float(np.dot(chroma_norm, min_n))

        if corr_maj > best_corr:
            best_corr = corr_maj
            best_key = f"{PITCH_CLASSES[shift]} major"
        if corr_min > best_corr:
            best_corr = corr_min
            best_key = f"{PITCH_CLASSES[shift]} minor"

    return best_key

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