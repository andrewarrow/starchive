package blend

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
	
	"starchive/audio"
)

// HandleSegmentCommand processes segment-related commands
func (bs *Shell) HandleSegmentCommand(cmd string, args []string) bool {
	switch cmd {
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
		
	default:
		return false // Command not handled by this module
	}
	
	return true
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