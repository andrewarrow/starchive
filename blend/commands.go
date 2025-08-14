package blend

import (
	"context"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
	
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
			bs.handleSegmentsCommand("") // List both tracks
		}
		
	case "random":
		if len(args) > 0 {
			bs.handleRandomCommand(args[0])
		} else {
			fmt.Printf("Usage: random <1|2>\n")
		}
		
	case "place":
		bs.handlePlaceCommand(args)
		
	case "shift":
		bs.handleShiftCommand(args)
		
	case "toggle":
		if len(args) > 0 {
			bs.handleToggleCommand(args[0])
		} else {
			fmt.Printf("Usage: toggle <track:segment> (e.g. 1:3)\n")
		}
		
	case "preview":
		if len(args) > 0 {
			bs.handlePreviewCommand(args[0])
		} else {
			fmt.Printf("Usage: preview <track:segment> (e.g. 1:3)\n")
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
	var startPosition1, startPosition2 float64
	
	if startFrom >= 0 {
		// Use specified start position
		startPosition1 = startFrom
		startPosition2 = startFrom
	} else {
		// Use middle position with window offsets
		startPosition1 = (bs.Duration1 / 2) + bs.Window1
		startPosition2 = (bs.Duration2 / 2) + bs.Window2
	}
	
	// Ensure valid start positions
	if startPosition1 < 0 {
		startPosition1 = 0
	}
	if startPosition2 < 0 {
		startPosition2 = 0
	}
	
	if startPosition1 >= bs.Duration1 {
		startPosition1 = bs.Duration1 - 1
	}
	if startPosition2 >= bs.Duration2 {
		startPosition2 = bs.Duration2 - 1
	}

	// Calculate maximum available play duration for both tracks
	remainingDuration1 := bs.Duration1 - startPosition1
	remainingDuration2 := bs.Duration2 - startPosition2
	maxAvailableDuration := remainingDuration1
	if remainingDuration2 < maxAvailableDuration {
		maxAvailableDuration = remainingDuration2
	}

	fmt.Printf("Playing blend... Press any key to stop.\n")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ffplayArgs1 := bs.buildFFplayArgs(bs.InputPath1, startPosition1, bs.Pitch1, bs.Tempo1, bs.Volume1, maxAvailableDuration)
	ffplayArgs2 := bs.buildFFplayArgs(bs.InputPath2, startPosition2, bs.Pitch2, bs.Tempo2, bs.Volume2, maxAvailableDuration)

	go func() {
		cmd1 := exec.CommandContext(ctx, "ffplay", ffplayArgs1...)
		cmd1.Run()
	}()

	go func() {
		cmd2 := exec.CommandContext(ctx, "ffplay", ffplayArgs2...)
		cmd2.Run()
	}()

	// Wait for any key press
	go func() {
		var input string
		fmt.Scanf("%s", &input)
		cancel()
	}()

	<-ctx.Done()
	fmt.Printf("Playback stopped.\n")
}

// buildFFplayArgs constructs ffplay arguments with audio effects
func (bs *Shell) buildFFplayArgs(inputPath string, startPos float64, pitch int, tempo float64, volume float64, playDuration float64) []string {
	args := []string{
		"-ss", fmt.Sprintf("%.1f", startPos),
		"-t", fmt.Sprintf("%.1f", playDuration),
		"-autoexit",
		"-nodisp",
		"-loglevel", "quiet",
	}

	if pitch != 0 || tempo != 0 || volume != 100 {
		var filters []string
		
		if tempo != 0 {
			tempoMultiplier := 1.0 + (tempo / 100.0)
			if tempoMultiplier > 0.5 && tempoMultiplier <= 2.0 {
				filters = append(filters, fmt.Sprintf("atempo=%.6f", tempoMultiplier))
			}
		}
		
		if pitch != 0 {
			pitchSemitones := float64(pitch)
			filters = append(filters, fmt.Sprintf("asetrate=44100*%.6f,aresample=44100,atempo=%.6f", 
				math.Pow(2, pitchSemitones/12.0), 1.0/math.Pow(2, pitchSemitones/12.0)))
		}
		
		if volume != 100 {
			volumeMultiplier := volume / 100.0
			filters = append(filters, fmt.Sprintf("volume=%.6f", volumeMultiplier))
		}
		
		if len(filters) > 0 {
			filter := strings.Join(filters, ",")
			args = append(args, "-af", filter)
		}
	}

	args = append(args, inputPath)
	return args
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
		bmpMatch = "bpm1to2"
	} else if bs.Tempo2 != 0 && bs.Tempo1 == 0 {
		bmpMatch = "bpm2to1"
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

// handleSplitCommand splits a track into vocal segments
func (bs *Shell) handleSplitCommand(trackNum string) {
	var id, inputPath string
	var segments *[]VocalSegment
	var segmentsDir string
	
	switch trackNum {
	case "1":
		id = bs.ID1
		inputPath = bs.InputPath1
		segments = &bs.Segments1
		segmentsDir = bs.SegmentsDir1
	case "2":
		id = bs.ID2
		inputPath = bs.InputPath2
		segments = &bs.Segments2
		segmentsDir = bs.SegmentsDir2
	default:
		fmt.Printf("Invalid track number: %s (use 1 or 2)\n", trackNum)
		return
	}
	
	// Only split vocal tracks
	trackType := bs.Type1
	if trackNum == "2" {
		trackType = bs.Type2
	}
	if trackType != "V" {
		fmt.Printf("Track %s is not vocal type. Switch to vocal first using 'type%s vocal'\n", trackNum, trackNum)
		return
	}
	
	fmt.Printf("Splitting track %s (%s) into vocal segments...\n", trackNum, id)
	
	err := os.MkdirAll(segmentsDir, 0755)
	if err != nil {
		fmt.Printf("Error creating segments directory: %v\n", err)
		return
	}
	
	// Run silence detection
	silenceCmd := exec.Command("ffmpeg", "-hide_banner", "-i", inputPath,
		"-af", "silencedetect=noise=-35dB:d=0.5", "-f", "null", "-")
	
	silenceOutput, err := silenceCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error detecting silence: %v\n", err)
		return
	}
	
	// Extract timestamps
	sedCmd := exec.Command("sed", "-n", "s/.*silence_end: \\([0-9.]*\\).*/\\1/p")
	sedCmd.Stdin = strings.NewReader(string(silenceOutput))
	sedOutput, err := sedCmd.Output()
	if err != nil {
		fmt.Printf("Error extracting timestamps: %v\n", err)
		return
	}
	
	timestamps := strings.TrimSpace(string(sedOutput))
	timestamps = strings.ReplaceAll(timestamps, "\n", ",")
	if timestamps == "" {
		fmt.Printf("No silence detected in track %s\n", trackNum)
		return
	}
	
	// Split the file
	outputPattern := fmt.Sprintf("%s/part_%%03d.wav", segmentsDir)
	splitCmd := exec.Command("ffmpeg", "-hide_banner", "-y", "-i", inputPath,
		"-c", "copy", "-f", "segment", "-segment_times", timestamps, outputPattern)
	
	err = splitCmd.Run()
	if err != nil {
		fmt.Printf("Error splitting file: %v\n", err)
		return
	}
	
	// Analyze created segments
	bs.loadSegments(trackNum)
	fmt.Printf("Successfully split track %s into %d segments\n", trackNum, len(*segments))
}

// handleSegmentsCommand lists available segments for a track
func (bs *Shell) handleSegmentsCommand(track string) {
	if track == "" {
		// List segments for both tracks
		fmt.Printf("Track 1 segments: %d total\n", len(bs.Segments1))
		for i, seg := range bs.Segments1 {
			status := "inactive"
			if seg.Active {
				status = "active"
			}
			endTime := seg.StartTime + seg.Duration
			fmt.Printf("  1:%d - %.2fs to %.2fs (%s)\n", i+1, seg.StartTime, endTime, status)
		}
		fmt.Printf("Track 2 segments: %d total\n", len(bs.Segments2))
		for i, seg := range bs.Segments2 {
			status := "inactive"
			if seg.Active {
				status = "active"
			}
			endTime := seg.StartTime + seg.Duration
			fmt.Printf("  2:%d - %.2fs to %.2fs (%s)\n", i+1, seg.StartTime, endTime, status)
		}
	} else if track == "1" {
		fmt.Printf("Track 1 segments: %d total\n", len(bs.Segments1))
		for i, seg := range bs.Segments1 {
			status := "inactive"
			if seg.Active {
				status = "active"
			}
			endTime := seg.StartTime + seg.Duration
			fmt.Printf("  1:%d - %.2fs to %.2fs (%s)\n", i+1, seg.StartTime, endTime, status)
		}
	} else if track == "2" {
		fmt.Printf("Track 2 segments: %d total\n", len(bs.Segments2))
		for i, seg := range bs.Segments2 {
			status := "inactive"
			if seg.Active {
				status = "active"
			}
			endTime := seg.StartTime + seg.Duration
			fmt.Printf("  2:%d - %.2fs to %.2fs (%s)\n", i+1, seg.StartTime, endTime, status)
		}
	} else {
		fmt.Printf("Invalid track: %s (use 1 or 2)\n", track)
	}
}

// handleRandomCommand randomly places segments from a track
func (bs *Shell) handleRandomCommand(trackNum string) {
	var segments *[]VocalSegment
	var targetDuration float64
	var id string
	
	switch trackNum {
	case "1":
		segments = &bs.Segments1
		targetDuration = bs.Duration2  // Place track 1 segments across track 2
		id = bs.ID1
	case "2":
		segments = &bs.Segments2
		targetDuration = bs.Duration1  // Place track 2 segments across track 1
		id = bs.ID2
	default:
		fmt.Printf("Invalid track number: %s (use 1 or 2)\n", trackNum)
		return
	}
	
	if len(*segments) == 0 {
		fmt.Printf("No segments found for track %s. Run 'split %s' first.\n", trackNum, trackNum)
		return
	}
	
	fmt.Printf("Randomly placing %d segments from track %s (%s) across %.1fs...\n", 
		len(*segments), trackNum, id, targetDuration)
	
	// Generate random placements, ensuring no overlaps
	rand.Seed(time.Now().UnixNano())
	
	for i := range *segments {
		// Place randomly in first 80% of target track to avoid cutting off
		maxPlacement := targetDuration * 0.8
		placement := rand.Float64() * maxPlacement
		
		(*segments)[i].Placement = placement
		(*segments)[i].Active = true
		
		fmt.Printf("  %s:%d placed at %.1fs\n", trackNum, (*segments)[i].Index, placement)
	}
	
	fmt.Printf("Random placement completed for track %s\n", trackNum)
}

// handlePlaceCommand places a segment at a specific time
func (bs *Shell) handlePlaceCommand(args []string) {
	if len(args) < 3 || args[1] != "at" {
		fmt.Printf("Usage: place <track:segment> at <time>\n")
		fmt.Printf("Example: place 1:3 at 45.2\n")
		return
	}
	
	segmentRef := args[0]
	timeStr := args[2]
	
	trackNum, segNum, ok := bs.parseSegmentRef(segmentRef)
	if !ok {
		fmt.Printf("Invalid segment reference: %s (use format track:segment, e.g., 1:3)\n", segmentRef)
		return
	}
	
	placement, err := strconv.ParseFloat(timeStr, 64)
	if err != nil {
		fmt.Printf("Invalid time: %s\n", timeStr)
		return
	}
	
	var segments *[]VocalSegment
	switch trackNum {
	case 1:
		segments = &bs.Segments1
	case 2:
		segments = &bs.Segments2
	default:
		fmt.Printf("Invalid track number: %d\n", trackNum)
		return
	}
	
	if len(*segments) == 0 {
		fmt.Printf("No segments found for track %d. Run 'split %d' first.\n", trackNum, trackNum)
		return
	}
	
	if segNum < 1 || segNum > len(*segments) {
		fmt.Printf("Segment %d not found. Track %d has %d segments.\n", segNum, trackNum, len(*segments))
		return
	}
	
	// Update segment placement
	segment := &(*segments)[segNum-1] // Convert 1-based to 0-based index
	segment.Placement = placement
	segment.Active = true // Placing a segment activates it
	
	fmt.Printf("Segment %d:%d placed at %.2fs and activated\n", trackNum, segNum, placement)
}

// handleShiftCommand shifts a segment timing
func (bs *Shell) handleShiftCommand(args []string) {
	if len(args) < 2 {
		fmt.Printf("Usage: shift <track:segment> <+/-time>\n")
		fmt.Printf("Example: shift 1:3 +2.5 (shift forward by 2.5 seconds)\n")
		fmt.Printf("Example: shift 1:3 -1.0 (shift backward by 1.0 seconds)\n")
		return
	}
	
	segmentRef := args[0]
	shiftStr := args[1]
	
	trackNum, segNum, ok := bs.parseSegmentRef(segmentRef)
	if !ok {
		fmt.Printf("Invalid segment reference: %s (use format track:segment, e.g., 1:3)\n", segmentRef)
		return
	}
	
	shift, err := strconv.ParseFloat(shiftStr, 64)
	if err != nil {
		fmt.Printf("Invalid shift amount: %s\n", shiftStr)
		return
	}
	
	var segments *[]VocalSegment
	switch trackNum {
	case 1:
		segments = &bs.Segments1
	case 2:
		segments = &bs.Segments2
	default:
		fmt.Printf("Invalid track number: %d\n", trackNum)
		return
	}
	
	if len(*segments) == 0 {
		fmt.Printf("No segments found for track %d. Run 'split %d' first.\n", trackNum, trackNum)
		return
	}
	
	if segNum < 1 || segNum > len(*segments) {
		fmt.Printf("Segment %d not found. Track %d has %d segments.\n", segNum, trackNum, len(*segments))
		return
	}
	
	// Update segment placement
	segment := &(*segments)[segNum-1] // Convert 1-based to 0-based index
	oldPlacement := segment.Placement
	segment.Placement += shift
	
	// Prevent negative placement
	if segment.Placement < 0 {
		segment.Placement = 0
	}
	
	fmt.Printf("Segment %d:%d shifted from %.2fs to %.2fs (%+.2fs)\n", 
		trackNum, segNum, oldPlacement, segment.Placement, shift)
}

// handleToggleCommand toggles a segment on/off
func (bs *Shell) handleToggleCommand(segmentRef string) {
	trackNum, segNum, ok := bs.parseSegmentRef(segmentRef)
	if !ok {
		fmt.Printf("Invalid segment reference: %s (use format track:segment, e.g., 1:3)\n", segmentRef)
		return
	}
	
	var segments *[]VocalSegment
	switch trackNum {
	case 1:
		segments = &bs.Segments1
	case 2:
		segments = &bs.Segments2
	default:
		fmt.Printf("Invalid track number: %d\n", trackNum)
		return
	}
	
	if len(*segments) == 0 {
		fmt.Printf("No segments found for track %d. Run 'split %d' first.\n", trackNum, trackNum)
		return
	}
	
	if segNum < 1 || segNum > len(*segments) {
		fmt.Printf("Segment %d not found. Track %d has %d segments.\n", segNum, trackNum, len(*segments))
		return
	}
	
	// Toggle the segment
	segment := &(*segments)[segNum-1] // Convert 1-based to 0-based index
	segment.Active = !segment.Active
	
	status := "inactive"
	if segment.Active {
		status = "active"
	}
	
	fmt.Printf("Segment %d:%d is now %s\n", trackNum, segNum, status)
}

// parseSegmentRef parses segment references like "1:3" 
func (bs *Shell) parseSegmentRef(segRef string) (int, int, bool) {
	parts := strings.Split(segRef, ":")
	if len(parts) != 2 {
		return 0, 0, false
	}
	
	trackNum, err := strconv.Atoi(parts[0])
	if err != nil || (trackNum != 1 && trackNum != 2) {
		return 0, 0, false
	}
	
	segNum, err := strconv.Atoi(parts[1])
	if err != nil || segNum < 1 {
		return 0, 0, false
	}
	
	return trackNum, segNum, true
}

// handlePreviewCommand previews a specific segment
func (bs *Shell) handlePreviewCommand(segRef string) {
	trackNum, segNum, ok := bs.parseSegmentRef(segRef)
	if !ok {
		fmt.Printf("Invalid segment reference: %s (use format track:segment like 1:3)\n", segRef)
		return
	}
	
	var segments []VocalSegment
	var segmentsDir string
	switch trackNum {
	case 1:
		segments = bs.Segments1
		segmentsDir = bs.SegmentsDir1
	case 2:
		segments = bs.Segments2
		segmentsDir = bs.SegmentsDir2
	default:
		fmt.Printf("Invalid track number: %d\n", trackNum)
		return
	}
	
	if len(segments) == 0 {
		fmt.Printf("No segments found for track %d. Run 'split %d' first.\n", trackNum, trackNum)
		return
	}
	
	if segNum < 1 || segNum > len(segments) {
		fmt.Printf("Segment %d not found for track %d (has %d segments)\n", segNum, trackNum, len(segments))
		return
	}
	
	segment := segments[segNum-1]
	segmentPath := fmt.Sprintf("%s/part_%03d.wav", segmentsDir, segment.Index)
	
	if _, err := os.Stat(segmentPath); os.IsNotExist(err) {
		fmt.Printf("Segment file not found: %s\n", segmentPath)
		return
	}
	
	fmt.Printf("Previewing segment %s (%.1fs duration)...\n", segRef, segment.Duration)
	fmt.Println("Press Ctrl+C to stop...")
	
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(segment.Duration*1000)*time.Millisecond)
	defer cancel()
	
	playCmd := exec.CommandContext(ctx, "ffplay", "-autoexit", "-nodisp", "-loglevel", "quiet", segmentPath)
	playCmd.Run()
	
	fmt.Printf("Preview completed.\n")
}

// loadSegments loads and analyzes segment files for a track
func (bs *Shell) loadSegments(trackNum string) {
	var segments *[]VocalSegment
	var segmentsDir string
	
	switch trackNum {
	case "1":
		segments = &bs.Segments1
		segmentsDir = bs.SegmentsDir1
	case "2":
		segments = &bs.Segments2
		segmentsDir = bs.SegmentsDir2
	default:
		return
	}
	
	entries, err := os.ReadDir(segmentsDir)
	if err != nil {
		return
	}
	
	*segments = []VocalSegment{}
	startTime := 0.0
	for i, entry := range entries {
		if strings.HasPrefix(entry.Name(), "part_") && strings.HasSuffix(entry.Name(), ".wav") {
			segmentPath := fmt.Sprintf("%s/%s", segmentsDir, entry.Name())
			duration, err := audio.GetAudioDuration(segmentPath)
			if err != nil {
				continue
			}
			
			segment := VocalSegment{
				Index:     i + 1,
				StartTime: startTime,
				Duration:  duration,
				Placement: startTime, // Default placement at original position
				Active:    false,     // Segments start inactive
			}
			
			*segments = append(*segments, segment)
			startTime += duration
		}
	}
}