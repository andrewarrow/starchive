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