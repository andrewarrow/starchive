package blend

import (
	"fmt"
	"strconv"
	"strings"
	
	"starchive/audio"
)

// HandleCommand processes user commands in the blend shell
// Returns false if the command indicates exit
func (bs *Shell) HandleCommand(input string) bool {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return true
	}
	
	cmd := parts[0]
	// Remove leading slash if present for backward compatibility
	if strings.HasPrefix(cmd, "/") {
		cmd = cmd[1:]
	}
	args := parts[1:]
	
	switch cmd {
	case "exit", "quit", "q":
		fmt.Println("Exiting blend shell...")
		return false
		
	case "help", "h":
		bs.ShowHelp()
		
	case "status", "s":
		bs.ShowStatus()
		
	case "reset", "r":
		bs.ResetAdjustments()
		
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
		
	case "match":
		if len(args) > 0 {
			bs.handleMatchCommand(args[0])
		} else {
			fmt.Printf("Usage: match <bpm1to2|bpm2to1|key1to2|key2to1>\n")
		}
		
	case "type1":
		if len(args) > 0 {
			bs.handleTypeCommand("1", args[0])
		} else {
			fmt.Printf("Usage: type1 <vocal|instrumental>\n")
		}
		
	case "type2":
		if len(args) > 0 {
			bs.handleTypeCommand("2", args[0])
		} else {
			fmt.Printf("Usage: type2 <vocal|instrumental>\n")
		}
		
	case "play", "p":
		if len(args) > 0 {
			if startPos, err := strconv.ParseFloat(args[0], 64); err == nil {
				bs.handlePlayCommand(startPos)
			} else {
				fmt.Printf("Invalid start position: %s\n", args[0])
			}
		} else {
			bs.handlePlayCommand(-1) // -1 means use default (middle)
		}
		
	case "invert":
		bs.handleInvertCommand()
		
	default:
		fmt.Printf("Unknown command: %s. Type 'help' for available commands.\n", cmd)
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

// handleMatchCommand handles BPM and key matching between tracks
func (bs *Shell) handleMatchCommand(matchType string) {
	if bs.Metadata1 == nil || bs.Metadata2 == nil {
		fmt.Printf("Metadata not available for matching\n")
		return
	}

	switch matchType {
	case "bpm1to2":
		if bs.Metadata1.BPM != nil && bs.Metadata2.BPM != nil {
			targetBPM := *bs.Metadata2.BPM
			currentBPM := *bs.Metadata1.BPM
			tempoChange := ((targetBPM / currentBPM) - 1.0) * 100.0
			bs.Tempo1 = clampFloat(tempoChange, -50.0, 100.0)
			fmt.Printf("Matched track 1 BPM to track 2: %.1f -> %.1f (tempo %+.1f%%)\n", 
				currentBPM, targetBPM, bs.Tempo1)
		} else {
			fmt.Printf("BPM data not available for matching\n")
		}
		
	case "bpm2to1":
		if bs.Metadata1.BPM != nil && bs.Metadata2.BPM != nil {
			targetBPM := *bs.Metadata1.BPM
			currentBPM := *bs.Metadata2.BPM
			tempoChange := ((targetBPM / currentBPM) - 1.0) * 100.0
			bs.Tempo2 = clampFloat(tempoChange, -50.0, 100.0)
			fmt.Printf("Matched track 2 BPM to track 1: %.1f -> %.1f (tempo %+.1f%%)\n", 
				currentBPM, targetBPM, bs.Tempo2)
		} else {
			fmt.Printf("BPM data not available for matching\n")
		}
		
	case "key1to2":
		if bs.Metadata1.Key != nil && bs.Metadata2.Key != nil {
			pitchChange := audio.CalculateKeyDifference(*bs.Metadata1.Key, *bs.Metadata2.Key)
			bs.Pitch1 = clamp(pitchChange, -12, 12)
			fmt.Printf("Matched track 1 key to track 2: %s -> %s (pitch %+d)\n", 
				*bs.Metadata1.Key, *bs.Metadata2.Key, bs.Pitch1)
		} else {
			fmt.Printf("Key data not available for matching\n")
		}
		
	case "key2to1":
		if bs.Metadata1.Key != nil && bs.Metadata2.Key != nil {
			pitchChange := audio.CalculateKeyDifference(*bs.Metadata2.Key, *bs.Metadata1.Key)
			bs.Pitch2 = clamp(pitchChange, -12, 12)
			fmt.Printf("Matched track 2 key to track 1: %s -> %s (pitch %+d)\n", 
				*bs.Metadata2.Key, *bs.Metadata1.Key, bs.Pitch2)
		} else {
			fmt.Printf("Key data not available for matching\n")
		}
		
	default:
		fmt.Printf("Unknown match type: %s\n", matchType)
		fmt.Printf("Usage: match <bpm1to2|bpm2to1|key1to2|key2to1>\n")
	}
}

// handleTypeCommand changes track types
func (bs *Shell) handleTypeCommand(track, trackType string) {
	switch strings.ToLower(trackType) {
	case "vocal", "vocals", "v":
		if track == "1" {
			bs.Type1 = "V"
			bs.InputPath1 = audio.GetAudioFilename(bs.ID1, "V")
			fmt.Printf("Track 1 set to vocal\n")
		} else {
			bs.Type2 = "V"
			bs.InputPath2 = audio.GetAudioFilename(bs.ID2, "V")
			fmt.Printf("Track 2 set to vocal\n")
		}
	case "instrumental", "instrumentals", "i":
		if track == "1" {
			bs.Type1 = "I"
			bs.InputPath1 = audio.GetAudioFilename(bs.ID1, "I")
			fmt.Printf("Track 1 set to instrumental\n")
		} else {
			bs.Type2 = "I"
			bs.InputPath2 = audio.GetAudioFilename(bs.ID2, "I")
			fmt.Printf("Track 2 set to instrumental\n")
		}
	default:
		fmt.Printf("Invalid track type: %s (use vocal or instrumental)\n", trackType)
	}
}

// handlePlayCommand plays the blend
func (bs *Shell) handlePlayCommand(startFrom float64) {
	fmt.Printf("Play functionality not yet implemented in new package structure\n")
	fmt.Printf("TODO: Implement playback in blend package\n")
}

// handleInvertCommand intelligently matches tracks
func (bs *Shell) handleInvertCommand() {
	fmt.Printf("Invert functionality not yet implemented in new package structure\n")
	fmt.Printf("TODO: Implement intelligent matching in blend package\n")
}