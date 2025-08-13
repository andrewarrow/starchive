package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func (bs *BlendShell) handleCommand(input string) bool {
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
		bs.showHelp()
		
	case "play", "p":
		if len(args) > 0 {
			if startPos, err := strconv.ParseFloat(args[0], 64); err == nil {
				bs.playBlendWithStart(startPos)
			} else {
				fmt.Printf("Invalid start position: %s\n", args[0])
			}
		} else {
			bs.playBlendWithStart(-1) // -1 means use default (middle)
		}
		
	case "status", "s":
		bs.showStatus()
		
	case "reset", "r":
		bs.resetAdjustments()
		
	case "pitch1":
		if len(args) > 0 {
			if val, err := strconv.Atoi(args[0]); err == nil {
				bs.pitch1 = clamp(val, -12, 12)
				fmt.Printf("Track 1 pitch: %+d semitones\n", bs.pitch1)
			} else {
				fmt.Printf("Invalid pitch value: %s\n", args[0])
			}
		} else {
			fmt.Printf("Current track 1 pitch: %+d semitones\n", bs.pitch1)
		}
		
	case "pitch2":
		if len(args) > 0 {
			if val, err := strconv.Atoi(args[0]); err == nil {
				bs.pitch2 = clamp(val, -12, 12)
				fmt.Printf("Track 2 pitch: %+d semitones\n", bs.pitch2)
			} else {
				fmt.Printf("Invalid pitch value: %s\n", args[0])
			}
		} else {
			fmt.Printf("Current track 2 pitch: %+d semitones\n", bs.pitch2)
		}
		
	case "tempo1":
		if len(args) > 0 {
			if val, err := strconv.ParseFloat(args[0], 64); err == nil {
				bs.tempo1 = clampFloat(val, -50.0, 100.0)
				fmt.Printf("Track 1 tempo: %+.1f%%\n", bs.tempo1)
			} else {
				fmt.Printf("Invalid tempo value: %s\n", args[0])
			}
		} else {
			fmt.Printf("Current track 1 tempo: %+.1f%%\n", bs.tempo1)
		}
		
	case "tempo2":
		if len(args) > 0 {
			if val, err := strconv.ParseFloat(args[0], 64); err == nil {
				bs.tempo2 = clampFloat(val, -50.0, 100.0)
				fmt.Printf("Track 2 tempo: %+.1f%%\n", bs.tempo2)
			} else {
				fmt.Printf("Invalid tempo value: %s\n", args[0])
			}
		} else {
			fmt.Printf("Current track 2 tempo: %+.1f%%\n", bs.tempo2)
		}
		
	case "volume1":
		if len(args) > 0 {
			if val, err := strconv.ParseFloat(args[0], 64); err == nil {
				bs.volume1 = clampFloat(val, 0.0, 200.0)
				fmt.Printf("Track 1 volume: %.0f%%\n", bs.volume1)
			} else {
				fmt.Printf("Invalid volume value: %s\n", args[0])
			}
		} else {
			fmt.Printf("Current track 1 volume: %.0f%%\n", bs.volume1)
		}
		
	case "volume2":
		if len(args) > 0 {
			if val, err := strconv.ParseFloat(args[0], 64); err == nil {
				bs.volume2 = clampFloat(val, 0.0, 200.0)
				fmt.Printf("Track 2 volume: %.0f%%\n", bs.volume2)
			} else {
				fmt.Printf("Invalid volume value: %s\n", args[0])
			}
		} else {
			fmt.Printf("Current track 2 volume: %.0f%%\n", bs.volume2)
		}
		
	case "type1":
		if len(args) > 0 {
			if args[0] == "vocal" || args[0] == "v" {
				bs.type1 = "V"
				bs.inputPath1 = getAudioFilename(bs.id1, bs.type1)
				fmt.Printf("Track 1 type: vocal\n")
			} else if args[0] == "instrumental" || args[0] == "i" {
				bs.type1 = "I"
				bs.inputPath1 = getAudioFilename(bs.id1, bs.type1)
				fmt.Printf("Track 1 type: instrumental\n")
			} else {
				fmt.Printf("Invalid type: %s (use 'vocal' or 'instrumental')\n", args[0])
			}
		} else {
			fmt.Printf("Current track 1 type: %s\n", bs.getTrackTypeDesc(bs.type1))
		}
		
	case "type2":
		if len(args) > 0 {
			if args[0] == "vocal" || args[0] == "v" {
				bs.type2 = "V"
				bs.inputPath2 = getAudioFilename(bs.id2, bs.type2)
				fmt.Printf("Track 2 type: vocal\n")
			} else if args[0] == "instrumental" || args[0] == "i" {
				bs.type2 = "I"
				bs.inputPath2 = getAudioFilename(bs.id2, bs.type2)
				fmt.Printf("Track 2 type: instrumental\n")
			} else {
				fmt.Printf("Invalid type: %s (use 'vocal' or 'instrumental')\n", args[0])
			}
		} else {
			fmt.Printf("Current track 2 type: %s\n", bs.getTrackTypeDesc(bs.type2))
		}
		
	case "window":
		if len(args) >= 2 {
			if val1, err1 := strconv.ParseFloat(args[0], 64); err1 == nil {
				if val2, err2 := strconv.ParseFloat(args[1], 64); err2 == nil {
					bs.window1 = val1
					bs.window2 = val2
					fmt.Printf("Window offsets: track 1 %+.1fs, track 2 %+.1fs\n", bs.window1, bs.window2)
				} else {
					fmt.Printf("Invalid window offset for track 2: %s\n", args[1])
				}
			} else {
				fmt.Printf("Invalid window offset for track 1: %s\n", args[0])
			}
		} else {
			fmt.Printf("Usage: window <track1_offset> <track2_offset>\n")
			fmt.Printf("Current window offsets: track 1 %+.1fs, track 2 %+.1fs\n", bs.window1, bs.window2)
		}
		
	case "match":
		if len(args) > 0 {
			bs.handleMatchCommand(args[0])
		} else {
			fmt.Printf("Usage: match <bpm1to2|bpm2to1|key1to2|key2to1>\n")
		}
		
	case "invert":
		bs.handleInvertCommand()
		
	case "split":
		if len(args) > 0 {
			bs.handleSplitCommand(args[0])
		} else {
			fmt.Printf("Usage: split <1|2>\n")
		}
		
	case "segments":
		if len(args) > 0 {
			bs.handleSegmentsCommand(args[0])
		} else {
			bs.handleSegmentsCommand("")
		}
		
	case "place":
		if len(args) >= 3 && args[1] == "at" {
			bs.handlePlaceCommand(args[0], args[2])
		} else {
			fmt.Printf("Usage: place <track:seg> at <time>\n")
			fmt.Printf("Example: place 1:3 at 45.2\n")
		}
		
	case "shift":
		if len(args) >= 2 {
			bs.handleShiftCommand(args[0], args[1])
		} else {
			fmt.Printf("Usage: shift <track:seg> <+/-time>\n")
			fmt.Printf("Example: shift 1:3 +2.5\n")
		}
		
	case "toggle":
		if len(args) > 0 {
			bs.handleToggleCommand(args[0])
		} else {
			fmt.Printf("Usage: toggle <track:seg>\n")
			fmt.Printf("Example: toggle 1:3\n")
		}
		
	case "preview":
		if len(args) > 0 {
			bs.handlePreviewCommand(args[0])
		} else {
			fmt.Printf("Usage: preview <track:seg>\n")
			fmt.Printf("Example: preview 1:3\n")
		}
		
	case "random":
		if len(args) > 0 {
			bs.handleRandomCommand(args[0])
		} else {
			fmt.Printf("Usage: random <1|2>\n")
			fmt.Printf("Example: random 1\n")
		}
		
	default:
		fmt.Printf("Unknown command: %s (type help for commands)\n", cmd)
	}
	
	// Auto-show status after commands that modify state
	switch cmd {
	case "exit", "quit", "q", "help", "h", "status", "s", "play", "p":
		// Don't show status after these commands
	default:
		bs.showStatus()
	}
	
	return true
}

func (bs *BlendShell) handleMatchCommand(matchType string) {
	if bs.metadata1 == nil || bs.metadata2 == nil {
		fmt.Printf("Cannot match - missing metadata\n")
		return
	}
	
	switch matchType {
	case "bpm1to2":
		if bs.metadata1.BPM != nil && bs.metadata2.BPM != nil {
			bpm1 := *bs.metadata1.BPM
			bpm2 := *bs.metadata2.BPM
			requiredRatio := bpm2 / bpm1
			bs.tempo1 = (requiredRatio - 1.0) * 100.0
			bs.tempo2 = 0.0
			fmt.Printf("Matched track 1 BPM (%.1f) to track 2 BPM (%.1f)\n", bpm1, bpm2)
			fmt.Printf("Track 1 tempo: %+.1f%%\n", bs.tempo1)
		} else {
			fmt.Printf("BPM data not available\n")
		}
		
	case "bpm2to1":
		if bs.metadata1.BPM != nil && bs.metadata2.BPM != nil {
			bpm1 := *bs.metadata1.BPM
			bpm2 := *bs.metadata2.BPM
			requiredRatio := bpm1 / bpm2
			bs.tempo2 = (requiredRatio - 1.0) * 100.0
			bs.tempo1 = 0.0
			fmt.Printf("Matched track 2 BPM (%.1f) to track 1 BPM (%.1f)\n", bpm2, bpm1)
			fmt.Printf("Track 2 tempo: %+.1f%%\n", bs.tempo2)
		} else {
			fmt.Printf("BPM data not available\n")
		}
		
	case "key1to2":
		if bs.metadata1.Key != nil && bs.metadata2.Key != nil {
			key1 := *bs.metadata1.Key
			key2 := *bs.metadata2.Key
			bs.pitch1 = calculateKeyDifference(key1, key2)
			bs.pitch2 = 0
			fmt.Printf("Matched track 1 key (%s) to track 2 key (%s)\n", key1, key2)
			fmt.Printf("Track 1 pitch: %+d semitones\n", bs.pitch1)
		} else {
			fmt.Printf("Key data not available\n")
		}
		
	case "key2to1":
		if bs.metadata1.Key != nil && bs.metadata2.Key != nil {
			key1 := *bs.metadata1.Key
			key2 := *bs.metadata2.Key
			bs.pitch2 = calculateKeyDifference(key2, key1)
			bs.pitch1 = 0
			fmt.Printf("Matched track 2 key (%s) to track 1 key (%s)\n", key2, key1)
			fmt.Printf("Track 2 pitch: %+d semitones\n", bs.pitch2)
		} else {
			fmt.Printf("Key data not available\n")
		}
		
	default:
		fmt.Printf("Unknown match type: %s\n", matchType)
		fmt.Printf("Available: bpm1to2, bpm2to1, key1to2, key2to1\n")
	}
}

func (bs *BlendShell) handleInvertCommand() {
	if bs.metadata1 == nil || bs.metadata2 == nil {
		fmt.Printf("Cannot invert - missing metadata\n")
		return
	}
	
	fmt.Printf("Inverting current match state...\n")
	
	// Save current state to determine what was matched
	stateFile := fmt.Sprintf("/tmp/starchive_invert_%s_%s.tmp", bs.id1, bs.id2)
	
	// Check if we have a previous state to invert from
	if bs.loadInvertState(stateFile) {
		// We have previous state, so invert it
		bs.applyInvertedState()
	} else {
		// No previous state, save current and apply default invert
		bs.saveInvertState(stateFile)
		bs.resetAdjustments()
		// Default: match bpm2to1 and key2to1
		bs.handleMatchCommand("bpm2to1")
		bs.handleMatchCommand("key2to1")
	}
}

func (bs *BlendShell) saveInvertState(stateFile string) {
	// Determine current match state based on adjustments
	var bpmMatch, keyMatch string
	
	if bs.tempo1 != 0 && bs.tempo2 == 0 {
		bpmMatch = "bpm1to2"
	} else if bs.tempo2 != 0 && bs.tempo1 == 0 {
		bpmMatch = "bpm2to1"
	} else {
		bpmMatch = "none"
	}
	
	if bs.pitch1 != 0 && bs.pitch2 == 0 {
		keyMatch = "key1to2"
	} else if bs.pitch2 != 0 && bs.pitch1 == 0 {
		keyMatch = "key2to1"
	} else {
		keyMatch = "none"
	}
	
	content := fmt.Sprintf("%s,%s", bpmMatch, keyMatch)
	os.WriteFile(stateFile, []byte(content), 0644)
}

func (bs *BlendShell) loadInvertState(stateFile string) bool {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return false
	}
	
	parts := strings.Split(string(data), ",")
	if len(parts) != 2 {
		return false
	}
	
	bs.previousBPMMatch = parts[0]
	bs.previousKeyMatch = parts[1]
	return true
}

func (bs *BlendShell) applyInvertedState() {
	bs.resetAdjustments()
	
	// Invert BPM match
	switch bs.previousBPMMatch {
	case "bpm1to2":
		bs.handleMatchCommand("bpm2to1")
	case "bpm2to1":
		bs.handleMatchCommand("bpm1to2")
	}
	
	// Invert key match
	switch bs.previousKeyMatch {
	case "key1to2":
		bs.handleMatchCommand("key2to1")
	case "key2to1":
		bs.handleMatchCommand("key1to2")
	}
	
	// Clear the state file so next invert toggles back
	stateFile := fmt.Sprintf("/tmp/starchive_invert_%s_%s.tmp", bs.id1, bs.id2)
	os.Remove(stateFile)
}