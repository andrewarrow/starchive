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

def estimate_fundamental_frequency(y, sr):
    """Estimate the fundamental frequency using multiple methods."""
    
    frequencies = []
    
    # Method 1: Pitch estimation using piptrack
    try:
        pitches, magnitudes = librosa.piptrack(y=y, sr=sr, threshold=0.1, fmin=80, fmax=400)
        pitch_values = []
        
        for t in range(pitches.shape[1]):
            index = magnitudes[:, t].argmax()
            pitch = pitches[index, t]
            if pitch > 0:
                pitch_values.append(pitch)
        
        if pitch_values:
            freq = np.median(pitch_values)
            if np.isfinite(freq) and freq > 0:
                frequencies.append(freq)
    except:
        pass
    
    # Method 2: Zero crossing rate based frequency estimation
    try:
        # Focus on the first 30 seconds for better accuracy
        segment_length = min(len(y), int(30 * sr))
        y_segment = y[:segment_length]
        
        # Apply low-pass filter to focus on fundamental frequency
        y_filtered = librosa.effects.preemphasis(y_segment)
        
        # Compute zero crossing rate
        zcr = librosa.feature.zero_crossing_rate(y_filtered, frame_length=2048, hop_length=512)[0]
        mean_zcr = np.mean(zcr)
        
        # Convert ZCR to frequency (rough approximation)
        if mean_zcr > 0:
            estimated_freq = mean_zcr * sr / 2
            if 80 <= estimated_freq <= 400:  # Reasonable range for fundamental frequency
                frequencies.append(estimated_freq)
    except:
        pass
    
    # Method 3: Spectral analysis for peak frequency
    try:
        # Compute FFT and find peak frequency
        fft = np.fft.rfft(y)
        magnitude = np.abs(fft)
        freqs = np.fft.rfftfreq(len(y), 1/sr)
        
        # Focus on frequency range where fundamental is likely to be
        low_freq_idx = np.where(freqs >= 80)[0]
        high_freq_idx = np.where(freqs <= 400)[0]
        
        if len(low_freq_idx) > 0 and len(high_freq_idx) > 0:
            start_idx = low_freq_idx[0]
            end_idx = high_freq_idx[-1]
            
            if start_idx < end_idx:
                peak_idx = np.argmax(magnitude[start_idx:end_idx]) + start_idx
                peak_freq = freqs[peak_idx]
                if peak_freq > 0:
                    frequencies.append(peak_freq)
    except:
        pass
    
    if not frequencies:
        return None
    
    # Remove outliers and return median
    if len(frequencies) > 2:
        q25 = np.percentile(frequencies, 25)
        q75 = np.percentile(frequencies, 75)
        iqr = q75 - q25
        lower_bound = q25 - 1.5 * iqr
        upper_bound = q75 + 1.5 * iqr
        frequencies = [f for f in frequencies if lower_bound <= f <= upper_bound]
    
    if not frequencies:
        return None
    
    return np.median(frequencies)

def estimate_peak_frequency(y, sr):
    """Estimate the peak frequency in the spectrum."""
    try:
        # Compute the magnitude spectrum
        fft = np.fft.rfft(y)
        magnitude = np.abs(fft)
        freqs = np.fft.rfftfreq(len(y), 1/sr)
        
        # Find the frequency with maximum magnitude
        peak_idx = np.argmax(magnitude)
        peak_freq = freqs[peak_idx]
        
        return peak_freq if peak_freq > 0 else None
    except:
        return None

def estimate_centroid_frequency(y, sr):
    """Estimate spectral centroid as a frequency measure."""
    try:
        centroid = librosa.feature.spectral_centroid(y=y, sr=sr)[0]
        mean_centroid = np.mean(centroid)
        return mean_centroid if np.isfinite(mean_centroid) and mean_centroid > 0 else None
    except:
        return None

def main():
    ap = argparse.ArgumentParser(description="Estimate frequency characteristics from a WAV file.")
    ap.add_argument("wav_path", help="Path to .wav file")
    ap.add_argument("--sr", type=int, default=None, help="Target sample rate (default: use file's native)")
    ap.add_argument("--mono", action="store_true", help="Force mono (recommended)")
    args = ap.parse_args()

    y, sr = librosa.load(args.wav_path, sr=args.sr, mono=args.mono or True)

    fundamental_freq = estimate_fundamental_frequency(y, sr)
    peak_freq = estimate_peak_frequency(y, sr)
    centroid_freq = estimate_centroid_frequency(y, sr)
    
    result = {
        "fundamental_frequency": round(fundamental_freq, 1) if fundamental_freq else None,
        "peak_frequency": round(peak_freq, 1) if peak_freq else None,
        "spectral_centroid": round(centroid_freq, 1) if centroid_freq else None
    }
    
    print(json.dumps(result))

if __name__ == "__main__":
    main()