package blend

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	
	"starchive/audio"
)

// HandleMatchingCommand processes track matching commands (match, type)
func (bs *Shell) HandleMatchingCommand(cmd string, args []string) bool {
	switch cmd {
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
		
	case "invert":
		bs.handleInvertCommand()
		
	case "auto-match":
		bs.handleAutoMatchCommand()
		
	default:
		return false // Command not handled by this module
	}
	
	return true
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

// handleInvertCommand intelligently matches tracks
func (bs *Shell) handleInvertCommand() {
	fmt.Printf("Inverting current match state...\n")
	
	// Save current state to determine what was matched
	stateFile := fmt.Sprintf("/tmp/starchive_invert_%s_%s.tmp", bs.ID1, bs.ID2)
	
	// Check if we have a previous state to invert from
	if bs.loadInvertState(stateFile) {
		// We have previous state, so invert it
		bs.applyInvertedState()
	} else {
		// No previous state, save current and apply default invert
		bs.saveInvertState(stateFile)
		bs.ResetAdjustments()
		// Default: match bpm2to1 and key2to1
		bs.handleMatchCommand("bpm2to1")
		bs.handleMatchCommand("key2to1")
	}
}

// saveInvertState saves current match state for inversion
func (bs *Shell) saveInvertState(stateFile string) {
	// Determine current match state based on adjustments
	var bmpMatch, keyMatch string
	
	if bs.Tempo1 != 0 && bs.Tempo2 == 0 {
		bmpMatch = "bmp1to2"
	} else if bs.Tempo2 != 0 && bs.Tempo1 == 0 {
		bmpMatch = "bmp2to1"
	} else {
		bmpMatch = "none"
	}
	
	if bs.Pitch1 != 0 && bs.Pitch2 == 0 {
		keyMatch = "key1to2"
	} else if bs.Pitch2 != 0 && bs.Pitch1 == 0 {
		keyMatch = "key2to1"
	} else {
		keyMatch = "none"
	}
	
	content := fmt.Sprintf("%s,%s", bmpMatch, keyMatch)
	ioutil.WriteFile(stateFile, []byte(content), 0644)
}

// loadInvertState loads previous match state
func (bs *Shell) loadInvertState(stateFile string) bool {
	data, err := ioutil.ReadFile(stateFile)
	if err != nil {
		return false
	}
	
	parts := strings.Split(string(data), ",")
	if len(parts) != 2 {
		return false
	}
	
	bs.PreviousBPMMatch = parts[0]
	bs.PreviousKeyMatch = parts[1]
	return true
}

// applyInvertedState applies the inverted state
func (bs *Shell) applyInvertedState() {
	bs.ResetAdjustments()
	
	// Invert BPM match
	switch bs.PreviousBPMMatch {
	case "bpm1to2":
		bs.handleMatchCommand("bpm2to1")
	case "bpm2to1":
		bs.handleMatchCommand("bpm1to2")
	}
	
	// Invert key match
	switch bs.PreviousKeyMatch {
	case "key1to2":
		bs.handleMatchCommand("key2to1")
	case "key2to1":
		bs.handleMatchCommand("key1to2")
	}
	
	// Clear the state file so next invert toggles back
	stateFile := fmt.Sprintf("/tmp/starchive_invert_%s_%s.tmp", bs.ID1, bs.ID2)
	os.Remove(stateFile)
}

// handleAutoMatchCommand intelligently determines best BPM/key matching direction
func (bs *Shell) handleAutoMatchCommand() {
	if bs.Metadata1 == nil || bs.Metadata2 == nil {
		fmt.Printf("Metadata not available for auto-matching\n")
		return
	}

	fmt.Printf("Analyzing tracks for optimal matching...\n")
	
	// Reset current adjustments
	bs.ResetAdjustments()
	
	// Determine BPM matching direction
	var bpmDirection string
	if bs.Metadata1.BPM != nil && bs.Metadata2.BPM != nil {
		bpm1 := *bs.Metadata1.BPM
		bpm2 := *bs.Metadata2.BPM
		
		// Calculate ratios for both directions
		ratio1to2 := bpm2 / bpm1
		ratio2to1 := bpm1 / bpm2
		
		// Choose direction with smaller adjustment (closer to 1.0)
		diff1to2 := abs(ratio1to2 - 1.0)
		diff2to1 := abs(ratio2to1 - 1.0)
		
		if diff1to2 <= diff2to1 {
			bpmDirection = "bpm1to2"
			fmt.Printf("  BPM: %.1f -> %.1f (ratio: %.2fx, %.1f%% change)\n", 
				bpm1, bpm2, ratio1to2, (ratio1to2-1.0)*100)
		} else {
			bpmDirection = "bpm2to1"
			fmt.Printf("  BPM: %.1f -> %.1f (ratio: %.2fx, %.1f%% change)\n", 
				bpm2, bpm1, ratio2to1, (ratio2to1-1.0)*100)
		}
	} else {
		fmt.Printf("  BPM: No BPM data available\n")
	}
	
	// Determine key matching direction
	var keyDirection string
	if bs.Metadata1.Key != nil && bs.Metadata2.Key != nil {
		key1 := *bs.Metadata1.Key
		key2 := *bs.Metadata2.Key
		
		// Calculate key differences for both directions
		diff1to2 := audio.CalculateKeyDifference(key1, key2)
		diff2to1 := audio.CalculateKeyDifference(key2, key1)
		
		// Choose direction with smaller semitone adjustment
		if abs(float64(diff1to2)) <= abs(float64(diff2to1)) {
			keyDirection = "key1to2"
			fmt.Printf("  Key: %s -> %s (%+d semitones)\n", key1, key2, diff1to2)
		} else {
			keyDirection = "key2to1"
			fmt.Printf("  Key: %s -> %s (%+d semitones)\n", key2, key1, diff2to1)
		}
	} else {
		fmt.Printf("  Key: No key data available\n")
	}
	
	// Apply the chosen matching
	if bpmDirection != "" {
		bs.handleMatchCommand(bpmDirection)
	}
	if keyDirection != "" {
		bs.handleMatchCommand(keyDirection)
	}
	
	fmt.Printf("Auto-match complete!\n")
}

// abs returns absolute value of float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}