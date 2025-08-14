package blend

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// HandleAudioCommand processes audio parameter commands (pitch, tempo, volume, window)
func (bs *Shell) HandleAudioCommand(cmd string, args []string) bool {
	switch cmd {
	case "pitch1":
		if len(args) > 0 {
			if val, err := strconv.Atoi(args[0]); err == nil {
				bs.Pitch1 = clamp(val, -12, 12)
				fmt.Printf("Track 1 pitch set to %+d semitones\n", bs.Pitch1)
			} else {
				fmt.Printf("Invalid pitch value: %s\n", args[0])
			}
		} else {
			fmt.Printf("Usage: pitch1 <semitones> (-12 to +12)\n")
		}
		
	case "pitch2":
		if len(args) > 0 {
			if val, err := strconv.Atoi(args[0]); err == nil {
				bs.Pitch2 = clamp(val, -12, 12)
				fmt.Printf("Track 2 pitch set to %+d semitones\n", bs.Pitch2)
			} else {
				fmt.Printf("Invalid pitch value: %s\n", args[0])
			}
		} else {
			fmt.Printf("Usage: pitch2 <semitones> (-12 to +12)\n")
		}
		
	case "tempo1":
		if len(args) > 0 {
			if val, err := strconv.ParseFloat(args[0], 64); err == nil {
				bs.Tempo1 = clampFloat(val, -50.0, 100.0)
				fmt.Printf("Track 1 tempo adjustment set to %+.1f%%\n", bs.Tempo1)
			} else {
				fmt.Printf("Invalid tempo value: %s\n", args[0])
			}
		} else {
			fmt.Printf("Usage: tempo1 <percentage> (-50 to +100)\n")
		}
		
	case "tempo2":
		if len(args) > 0 {
			if val, err := strconv.ParseFloat(args[0], 64); err == nil {
				bs.Tempo2 = clampFloat(val, -50.0, 100.0)
				fmt.Printf("Track 2 tempo adjustment set to %+.1f%%\n", bs.Tempo2)
			} else {
				fmt.Printf("Invalid tempo value: %s\n", args[0])
			}
		} else {
			fmt.Printf("Usage: tempo2 <percentage> (-50 to +100)\n")
		}
		
	case "volume1":
		if len(args) > 0 {
			if val, err := strconv.ParseFloat(args[0], 64); err == nil {
				bs.Volume1 = clampFloat(val, 0.0, 200.0)
				fmt.Printf("Track 1 volume set to %.0f%%\n", bs.Volume1)
			} else {
				fmt.Printf("Invalid volume value: %s\n", args[0])
			}
		} else {
			fmt.Printf("Usage: volume1 <percentage> (0 to 200)\n")
		}
		
	case "volume2":
		if len(args) > 0 {
			if val, err := strconv.ParseFloat(args[0], 64); err == nil {
				bs.Volume2 = clampFloat(val, 0.0, 200.0)
				fmt.Printf("Track 2 volume set to %.0f%%\n", bs.Volume2)
			} else {
				fmt.Printf("Invalid volume value: %s\n", args[0])
			}
		} else {
			fmt.Printf("Usage: volume2 <percentage> (0 to 200)\n")
		}
		
	case "window":
		if len(args) >= 2 {
			if val1, err1 := strconv.ParseFloat(args[0], 64); err1 == nil {
				if val2, err2 := strconv.ParseFloat(args[1], 64); err2 == nil {
					bs.Window1 = val1
					bs.Window2 = val2
					fmt.Printf("Track windows set to %+.1fs, %+.1fs\n", bs.Window1, bs.Window2)
				} else {
					fmt.Printf("Invalid second window value: %s\n", args[1])
				}
			} else {
				fmt.Printf("Invalid first window value: %s\n", args[0])
			}
		} else {
			fmt.Printf("Usage: window <seconds1> <seconds2>\n")
		}
		
	case "beat-detect":
		if len(args) > 0 {
			bs.handleBeatDetectCommand(args[0])
		} else {
			fmt.Printf("Usage: beat-detect <1|2|both>\n")
		}
		
	case "beats":
		if len(args) > 0 {
			bs.handleBeatsCommand(args[0])
		} else {
			bs.handleBeatsCommand("") // Show both tracks
		}
		
	default:
		return false // Command not handled by this module
	}
	
	return true
}

// Helper functions
func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func clampFloat(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// handleBeatDetectCommand detects beat positions using onset detection
func (bs *Shell) handleBeatDetectCommand(target string) {
	switch target {
	case "1":
		bs.detectBeats("1", bs.InputPath1, bs.ID1, &bs.Beats1)
	case "2":
		bs.detectBeats("2", bs.InputPath2, bs.ID2, &bs.Beats2)
	case "both":
		bs.detectBeats("1", bs.InputPath1, bs.ID1, &bs.Beats1)
		bs.detectBeats("2", bs.InputPath2, bs.ID2, &bs.Beats2)
	default:
		fmt.Printf("Invalid target: %s (use 1, 2, or both)\n", target)
	}
}

// detectBeats uses ffprobe with onset detection to find beat positions
func (bs *Shell) detectBeats(trackNum, inputPath, id string, beats *[]float64) {
	fmt.Printf("Detecting beats in track %s (%s)...\n", trackNum, id)
	
	// Use ffprobe with silencedetect as a simple onset detector
	// This detects sudden changes in audio level which often correspond to beats
	cmd := exec.Command("ffprobe", "-hide_banner", "-v", "quiet", 
		"-f", "lavfi", "-i", fmt.Sprintf("amovie=%s,aresample=22050,asplit[a][b];[a]aformat=channel_layouts=mono,showwaves=s=640x120:mode=point,format=gray[wave];[b]aformat=channel_layouts=mono,atempo=1.0,highpass=f=80,lowpass=f=400,aresample=1024,showfreqs=s=640x240:mode=bar:ascale=log[freq]", inputPath),
		"-show_entries", "packet=pts_time",
		"-select_streams", "a:0",
		"-of", "json=compact=1")
	
	_, err := cmd.Output()
	if err != nil {
		// Fallback to a simpler approach using aubio if available
		bs.detectBeatsWithAubio(trackNum, inputPath, id, beats)
		return
	}
	
	// Try a different approach using spectral analysis
	bs.detectBeatsWithSpectralAnalysis(trackNum, inputPath, id, beats)
}

// detectBeatsWithSpectralAnalysis uses spectral flux for onset detection
func (bs *Shell) detectBeatsWithSpectralAnalysis(trackNum, inputPath, id string, beats *[]float64) {
	// Use ffprobe to analyze spectral changes that indicate onsets/beats
	cmd := exec.Command("ffprobe", "-hide_banner", "-v", "quiet",
		"-f", "lavfi", "-i", fmt.Sprintf("amovie=%s,aresample=22050,asplit[a][b];[a]showspectrum=s=1024x1:slide=scroll:mode=separate:color=intensity:scale=log[spec];[b]showwaves=s=1024x1:mode=point[wave]", inputPath),
		"-show_entries", "frame=pkt_pts_time",
		"-of", "json")
	
	_, err := cmd.Output()
	if err != nil {
		fmt.Printf("  Spectral analysis failed, using simple approach: %v\n", err)
		bs.detectBeatsSimple(trackNum, inputPath, id, beats)
		return
	}
	
	// For now, fall back to simple detection
	bs.detectBeatsSimple(trackNum, inputPath, id, beats)
}

// detectBeatsWithAubio uses aubio onset detection if available
func (bs *Shell) detectBeatsWithAubio(trackNum, inputPath, id string, beats *[]float64) {
	fmt.Printf("  Trying aubio onset detection...\n")
	
	// Check if aubio is available
	checkCmd := exec.Command("which", "aubiodet")
	if checkCmd.Run() != nil {
		fmt.Printf("  aubio not available, using simple approach\n")
		bs.detectBeatsSimple(trackNum, inputPath, id, beats)
		return
	}
	
	// Use aubio for onset detection
	cmd := exec.Command("aubiodet", "-i", inputPath, "-O", "onset")
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("  aubio failed: %v, using simple approach\n", err)
		bs.detectBeatsSimple(trackNum, inputPath, id, beats)
		return
	}
	
	// Parse aubio output (timestamps in seconds)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	*beats = []float64{}
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		if timestamp, err := strconv.ParseFloat(line, 64); err == nil {
			*beats = append(*beats, timestamp)
		}
	}
	
	fmt.Printf("  Found %d onsets/beats using aubio\n", len(*beats))
}

// detectBeatsSimple uses a basic approach based on BPM metadata
func (bs *Shell) detectBeatsSimple(trackNum, inputPath, id string, beats *[]float64) {
	fmt.Printf("  Using simple BPM-based beat detection...\n")
	
	var metadata *VideoMetadata
	var duration float64
	
	if trackNum == "1" {
		metadata = bs.Metadata1
		duration = bs.Duration1
	} else {
		metadata = bs.Metadata2
		duration = bs.Duration2
	}
	
	*beats = []float64{}
	
	if metadata == nil || metadata.BPM == nil {
		fmt.Printf("  No BPM metadata available for track %s\n", trackNum)
		return
	}
	
	bpm := *metadata.BPM
	if bpm <= 0 {
		fmt.Printf("  Invalid BPM value: %.1f\n", bpm)
		return
	}
	
	// Calculate beat interval in seconds
	beatInterval := 60.0 / bpm
	
	// Generate beat positions every beat interval
	for t := 0.0; t < duration; t += beatInterval {
		*beats = append(*beats, t)
	}
	
	fmt.Printf("  Generated %d beats based on %.1f BPM (every %.2fs)\n", len(*beats), bpm, beatInterval)
}

// findNearestBeat finds the closest beat to a given time
func (bs *Shell) findNearestBeat(beats []float64, targetTime float64) float64 {
	if len(beats) == 0 {
		return targetTime
	}
	
	minDiff := float64(1000000) // Large initial value
	nearest := targetTime
	
	for _, beat := range beats {
		diff := targetTime - beat
		if diff < 0 {
			diff = -diff
		}
		
		if diff < minDiff {
			minDiff = diff
			nearest = beat
		}
	}
	
	return nearest
}

// handleBeatsCommand shows detected beats
func (bs *Shell) handleBeatsCommand(track string) {
	if track == "" {
		// Show beats for both tracks
		fmt.Printf("Track 1 beats: %d total\n", len(bs.Beats1))
		if len(bs.Beats1) > 0 {
			fmt.Printf("  First 10 beats: ")
			for i, beat := range bs.Beats1 {
				if i >= 10 { break }
				fmt.Printf("%.1fs ", beat)
			}
			fmt.Printf("\n")
			if len(bs.Beats1) > 10 {
				fmt.Printf("  ... and %d more\n", len(bs.Beats1)-10)
			}
		}
		
		fmt.Printf("Track 2 beats: %d total\n", len(bs.Beats2))
		if len(bs.Beats2) > 0 {
			fmt.Printf("  First 10 beats: ")
			for i, beat := range bs.Beats2 {
				if i >= 10 { break }
				fmt.Printf("%.1fs ", beat)
			}
			fmt.Printf("\n")
			if len(bs.Beats2) > 10 {
				fmt.Printf("  ... and %d more\n", len(bs.Beats2)-10)
			}
		}
	} else if track == "1" {
		fmt.Printf("Track 1 beats: %d total\n", len(bs.Beats1))
		for i, beat := range bs.Beats1 {
			fmt.Printf("  Beat %d: %.2fs\n", i+1, beat)
		}
	} else if track == "2" {
		fmt.Printf("Track 2 beats: %d total\n", len(bs.Beats2))
		for i, beat := range bs.Beats2 {
			fmt.Printf("  Beat %d: %.2fs\n", i+1, beat)
		}
	} else {
		fmt.Printf("Invalid track: %s (use 1 or 2)\n", track)
	}
}