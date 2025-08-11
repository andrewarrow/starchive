#!/usr/bin/env python3
import argparse
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
    print("[DEBUG] Starting BPM estimation...")
    print(f"[DEBUG] hop_length={hop_length}, frame_duration={hop_length/sr:.4f}s")
    onset_env = librosa.onset.onset_strength(y=y, sr=sr, hop_length=hop_length)
    print(f"[DEBUG] Onset envelope shape: {onset_env.shape}, first 10: {np.round(onset_env[:10],3)}")

    tempo, beat_frames = librosa.beat.beat_track(
        y=y, sr=sr, onset_envelope=onset_env, hop_length=hop_length
    )
    tempo = _to_float(tempo)
    beat_times = librosa.frames_to_time(beat_frames, sr=sr, hop_length=hop_length)

    print(f"[DEBUG] Beat frame indices: {beat_frames[:20]}{'...' if len(beat_frames)>20 else ''}")
    print(f"[DEBUG] Beat times (s): {np.round(beat_times[:10], 3)}{'...' if len(beat_times)>10 else ''}")
    print(f"[DEBUG] Raw tempo from beat_track: {tempo}")

    # Fallback if tracker didn’t find anything sensible
    if not np.isfinite(tempo) or tempo <= 0:
        print("[DEBUG] beat_track returned invalid tempo; falling back to beat.tempo()")
        # aggregate can return scalar; ensure float
        t_arr = librosa.beat.tempo(onset_envelope=onset_env, sr=sr, hop_length=hop_length, aggregate='mean')
        tempo = _to_float(t_arr)
        print(f"[DEBUG] Fallback tempo (mean): {tempo}")

    return tempo

def estimate_key(y, sr):
    print("[DEBUG] Starting key estimation...")
    chroma = librosa.feature.chroma_cqt(y=y, sr=sr)
    print(f"[DEBUG] Chroma matrix shape: {chroma.shape}")
    chroma_avg = chroma.mean(axis=1)
    print(f"[DEBUG] Average chroma vector: {np.round(chroma_avg, 3)}")
    if np.allclose(chroma_avg.sum(), 0.0):
        print("[DEBUG] No harmonic content detected — returning Unknown")
        return "Unknown"

    chroma_norm = (chroma_avg - chroma_avg.mean()) / (np.linalg.norm(chroma_avg) + 1e-12)
    print(f"[DEBUG] Normalized chroma vector: {np.round(chroma_norm, 3)}")

    best_key = None
    best_corr = -np.inf
    for shift in range(12):
        maj = np.roll(MAJOR_PROFILE, shift)
        minr = np.roll(MINOR_PROFILE, shift)
        maj_n = (maj - maj.mean()) / (np.linalg.norm(maj) + 1e-12)
        min_n = (minr - minr.mean()) / (np.linalg.norm(minr) + 1e-12)

        corr_maj = float(np.dot(chroma_norm, maj_n))
        corr_min = float(np.dot(chroma_norm, min_n))

        print(f"[DEBUG] Shift {shift:2d}: {PITCH_CLASSES[shift]} major corr={corr_maj:.4f}, "
              f"{PITCH_CLASSES[shift]} minor corr={corr_min:.4f}")

        if corr_maj > best_corr:
            best_corr = corr_maj
            best_key = f"{PITCH_CLASSES[shift]} major"
        if corr_min > best_corr:
            best_corr = corr_min
            best_key = f"{PITCH_CLASSES[shift]} minor"

    print(f"[DEBUG] Best correlation: {best_corr:.4f} → Key = {best_key}")
    return best_key

def main():
    ap = argparse.ArgumentParser(description="Estimate BPM and musical key from a WAV file.")
    ap.add_argument("wav_path", help="Path to .wav file")
    ap.add_argument("--sr", type=int, default=None, help="Target sample rate (default: use file's native)")
    ap.add_argument("--mono", action="store_true", help="Force mono (recommended)")
    args = ap.parse_args()

    print(f"[DEBUG] Loading {args.wav_path} (sr={args.sr}, mono={args.mono or True})...")
    y, sr = librosa.load(args.wav_path, sr=args.sr, mono=args.mono or True)
    print(f"[DEBUG] Audio loaded: {len(y)} samples at {sr} Hz ({len(y)/sr:.2f} seconds)")

    if len(y) < sr * 1.0:
        print("[WARNING] Very short file (<1s). Results may be unreliable.")

    bpm = estimate_bpm(y, sr)
    key = estimate_key(y, sr)

    print("\n=== Final Results ===")
    print(f"BPM: {bpm:.1f}")
    print(f"Key: {key}")

if __name__ == "__main__":
    main()

