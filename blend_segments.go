package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func (bs *BlendShell) handleSplitCommand(trackNum string) {
	var id, inputPath string
	var segments *[]VocalSegment
	var segmentsDir string
	
	switch trackNum {
	case "1":
		id = bs.id1
		inputPath = bs.inputPath1
		segments = &bs.segments1
		segmentsDir = bs.segmentsDir1
	case "2":
		id = bs.id2
		inputPath = bs.inputPath2
		segments = &bs.segments2
		segmentsDir = bs.segmentsDir2
	default:
		fmt.Printf("Invalid track number: %s (use 1 or 2)\n", trackNum)
		return
	}
	
	// Only split vocal tracks
	trackType := bs.type1
	if trackNum == "2" {
		trackType = bs.type2
	}
	if trackType != "V" {
		fmt.Printf("Track %s is not vocal type. Switch to vocal first using 'type%s vocal'\n", trackNum, trackNum)
		return
	}
	
	fmt.Printf("Splitting track %s (%s) into vocal segments...\n", trackNum, id)
	
	if err := os.MkdirAll(segmentsDir, 0755); err != nil {
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
		fmt.Printf("No silence detected. Creating single segment.\n")
		outputPath := fmt.Sprintf("%s/part_001.wav", segmentsDir)
		copyCmd := exec.Command("cp", inputPath, outputPath)
		if err := copyCmd.Run(); err != nil {
			fmt.Printf("Error copying file: %v\n", err)
			return
		}
		
		duration, _ := getAudioDuration(inputPath)
		*segments = []VocalSegment{{
			Index: 1, StartTime: 0, Duration: duration,
			Placement: 0, Active: false,
		}}
		fmt.Printf("Created 1 segment\n")
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

func (bs *BlendShell) loadSegments(trackNum string) {
	var segments *[]VocalSegment
	var segmentsDir string
	
	switch trackNum {
	case "1":
		segments = &bs.segments1
		segmentsDir = bs.segmentsDir1
	case "2":
		segments = &bs.segments2
		segmentsDir = bs.segmentsDir2
	default:
		return
	}
	
	entries, err := os.ReadDir(segmentsDir)
	if err != nil {
		return
	}
	
	*segments = []VocalSegment{}
	for i, entry := range entries {
		if strings.HasPrefix(entry.Name(), "part_") && strings.HasSuffix(entry.Name(), ".wav") {
			segmentPath := fmt.Sprintf("%s/%s", segmentsDir, entry.Name())
			duration, err := getAudioDuration(segmentPath)
			if err != nil {
				continue
			}
			
			*segments = append(*segments, VocalSegment{
				Index: i + 1, StartTime: 0, Duration: duration,
				Placement: 0, Active: false,
			})
		}
	}
}

func (bs *BlendShell) handleSegmentsCommand(trackNum string) {
	if trackNum == "" {
		// Show both tracks
		fmt.Printf("--- Segments ---\n")
		if len(bs.segments1) > 0 {
			fmt.Printf("Track 1 (%s): %d segments\n", bs.id1, len(bs.segments1))
			for _, seg := range bs.segments1 {
				status := "off"
				if seg.Active { status = "on" }
				fmt.Printf("  1:%d - %.1fs duration, placed at %.1fs [%s]\n", 
					seg.Index, seg.Duration, seg.Placement, status)
			}
		} else {
			fmt.Printf("Track 1 (%s): no segments (use 'split 1')\n", bs.id1)
		}
		
		if len(bs.segments2) > 0 {
			fmt.Printf("Track 2 (%s): %d segments\n", bs.id2, len(bs.segments2))
			for _, seg := range bs.segments2 {
				status := "off"
				if seg.Active { status = "on" }
				fmt.Printf("  2:%d - %.1fs duration, placed at %.1fs [%s]\n", 
					seg.Index, seg.Duration, seg.Placement, status)
			}
		} else {
			fmt.Printf("Track 2 (%s): no segments (use 'split 2')\n", bs.id2)
		}
		return
	}
	
	// Show specific track
	var segments []VocalSegment
	var id string
	
	switch trackNum {
	case "1":
		segments = bs.segments1
		id = bs.id1
	case "2":
		segments = bs.segments2
		id = bs.id2
	default:
		fmt.Printf("Invalid track number: %s (use 1 or 2)\n", trackNum)
		return
	}
	
	if len(segments) == 0 {
		fmt.Printf("Track %s (%s): no segments (use 'split %s')\n", trackNum, id, trackNum)
		return
	}
	
	fmt.Printf("Track %s (%s): %d segments\n", trackNum, id, len(segments))
	for _, seg := range segments {
		status := "off"
		if seg.Active { status = "on" }
		fmt.Printf("  %s:%d - %.1fs duration, placed at %.1fs [%s]\n", 
			trackNum, seg.Index, seg.Duration, seg.Placement, status)
	}
}

func (bs *BlendShell) parseSegmentRef(segRef string) (int, int, bool) {
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

func (bs *BlendShell) handlePlaceCommand(segRef, timeStr string) {
	trackNum, segNum, ok := bs.parseSegmentRef(segRef)
	if !ok {
		fmt.Printf("Invalid segment reference: %s (use format track:segment like 1:3)\n", segRef)
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
		segments = &bs.segments1
	case 2:
		segments = &bs.segments2
	}
	
	if segNum > len(*segments) {
		fmt.Printf("Segment %d not found for track %d\n", segNum, trackNum)
		return
	}
	
	(*segments)[segNum-1].Placement = placement
	(*segments)[segNum-1].Active = true
	fmt.Printf("Placed segment %s at %.1fs\n", segRef, placement)
}

func (bs *BlendShell) handleShiftCommand(segRef, shiftStr string) {
	trackNum, segNum, ok := bs.parseSegmentRef(segRef)
	if !ok {
		fmt.Printf("Invalid segment reference: %s (use format track:segment like 1:3)\n", segRef)
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
		segments = &bs.segments1
	case 2:
		segments = &bs.segments2
	}
	
	if segNum > len(*segments) {
		fmt.Printf("Segment %d not found for track %d\n", segNum, trackNum)
		return
	}
	
	(*segments)[segNum-1].Placement += shift
	fmt.Printf("Shifted segment %s by %+.1fs to %.1fs\n", 
		segRef, shift, (*segments)[segNum-1].Placement)
}

func (bs *BlendShell) handleToggleCommand(segRef string) {
	trackNum, segNum, ok := bs.parseSegmentRef(segRef)
	if !ok {
		fmt.Printf("Invalid segment reference: %s (use format track:segment like 1:3)\n", segRef)
		return
	}
	
	var segments *[]VocalSegment
	switch trackNum {
	case 1:
		segments = &bs.segments1
	case 2:
		segments = &bs.segments2
	}
	
	if segNum > len(*segments) {
		fmt.Printf("Segment %d not found for track %d\n", segNum, trackNum)
		return
	}
	
	(*segments)[segNum-1].Active = !(*segments)[segNum-1].Active
	status := "disabled"
	if (*segments)[segNum-1].Active {
		status = "enabled"
	}
	fmt.Printf("Segment %s is now %s\n", segRef, status)
}

func (bs *BlendShell) handlePreviewCommand(segRef string) {
	trackNum, segNum, ok := bs.parseSegmentRef(segRef)
	if !ok {
		fmt.Printf("Invalid segment reference: %s (use format track:segment like 1:3)\n", segRef)
		return
	}
	
	var segments []VocalSegment
	var segmentsDir string
	switch trackNum {
	case 1:
		segments = bs.segments1
		segmentsDir = bs.segmentsDir1
	case 2:
		segments = bs.segments2
		segmentsDir = bs.segmentsDir2
	}
	
	if segNum > len(segments) {
		fmt.Printf("Segment %d not found for track %d\n", segNum, trackNum)
		return
	}
	
	segment := segments[segNum-1]
	segmentPath := fmt.Sprintf("%s/part_%03d.wav", segmentsDir, segment.Index)
	
	if _, err := os.Stat(segmentPath); os.IsNotExist(err) {
		fmt.Printf("Segment file not found: %s\n", segmentPath)
		return
	}
	
	fmt.Printf("Previewing segment %s (%.1fs duration)...\n", segRef, segment.Duration)
	
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(segment.Duration*1000)*time.Millisecond)
	defer cancel()
	
	playCmd := exec.CommandContext(ctx, "ffplay", "-autoexit", "-nodisp", "-loglevel", "quiet", segmentPath)
	playCmd.Run()
	
	fmt.Printf("Preview completed.\n")
}

func (bs *BlendShell) handleRandomCommand(trackStr string) {
	trackNum, err := strconv.Atoi(trackStr)
	if err != nil || (trackNum != 1 && trackNum != 2) {
		fmt.Printf("Invalid track number: %s (use 1 or 2)\n", trackStr)
		return
	}
	
	var segments *[]VocalSegment
	var targetDuration float64
	var id string
	
	switch trackNum {
	case 1:
		segments = &bs.segments1
		targetDuration = bs.duration2
		id = bs.id1
	case 2:
		segments = &bs.segments2 
		targetDuration = bs.duration1
		id = bs.id2
	}
	
	if len(*segments) == 0 {
		fmt.Printf("No segments found for track %d. Run 'split %d' first.\n", trackNum, trackNum)
		return
	}
	
	fmt.Printf("Randomly placing %d segments from track %d (%s) across %.1fs...\n", 
		len(*segments), trackNum, id, targetDuration)
	
	// Generate random placements, ensuring no overlaps
	rand.Seed(time.Now().UnixNano())
	
	for i := range *segments {
		// Place randomly in first 80% of target track to avoid cutting off
		maxPlacement := targetDuration * 0.8
		placement := rand.Float64() * maxPlacement
		
		(*segments)[i].Placement = placement
		(*segments)[i].Active = true
		
		fmt.Printf("  %d:%d placed at %.1fs\n", trackNum, (*segments)[i].Index, placement)
	}
	
	fmt.Printf("Random placement completed for track %d\n", trackNum)
}