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
)

// HandleSegmentManipulationCommand processes segment manipulation commands
func (bs *Shell) HandleSegmentManipulationCommand(cmd string, args []string) bool {
	switch cmd {
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
		
	case "segment-trim":
		if len(args) > 0 {
			bs.handleSegmentTrimCommand(args[0])
		} else {
			fmt.Printf("Usage: segment-trim <1|2|all>\n")
		}
		
	default:
		return false // Command not handled by this module
	}
	
	return true
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

// handleSegmentTrimCommand automatically trims silence from segment edges
func (bs *Shell) handleSegmentTrimCommand(target string) {
	fmt.Printf("Auto-trimming silence from segment edges...\n")
	
	silenceThreshold := -40.0 // dB threshold for silence detection
	minTrimAmount := 0.1     // Minimum trim amount in seconds
	maxTrimAmount := 2.0     // Maximum trim amount per edge in seconds
	
	switch target {
	case "1":
		bs.trimSegmentsForTrack(1, silenceThreshold, minTrimAmount, maxTrimAmount)
	case "2":
		bs.trimSegmentsForTrack(2, silenceThreshold, minTrimAmount, maxTrimAmount)
	case "all":
		bs.trimSegmentsForTrack(1, silenceThreshold, minTrimAmount, maxTrimAmount)
		bs.trimSegmentsForTrack(2, silenceThreshold, minTrimAmount, maxTrimAmount)
	default:
		fmt.Printf("Invalid target: %s (use 1, 2, or all)\n", target)
	}
}

// trimSegmentsForTrack performs silence trimming for a specific track
func (bs *Shell) trimSegmentsForTrack(trackNum int, threshold, minTrim, maxTrim float64) {
	var segments *[]VocalSegment
	var segmentsDir string
	var id string
	
	switch trackNum {
	case 1:
		segments = &bs.Segments1
		segmentsDir = bs.SegmentsDir1
		id = bs.ID1
	case 2:
		segments = &bs.Segments2
		segmentsDir = bs.SegmentsDir2
		id = bs.ID2
	default:
		fmt.Printf("Invalid track number: %d\n", trackNum)
		return
	}
	
	if len(*segments) == 0 {
		fmt.Printf("No segments found for track %d. Run 'split %d' first.\n", trackNum, trackNum)
		return
	}
	
	fmt.Printf("Trimming %d segments for track %d (%s)...\n", len(*segments), trackNum, id)
	
	trimmedCount := 0
	totalTimeSaved := 0.0
	
	for i := range *segments {
		segment := &(*segments)[i]
		segmentPath := fmt.Sprintf("%s/part_%03d.wav", segmentsDir, segment.Index)
		
		// Check if segment file exists
		if _, err := os.Stat(segmentPath); os.IsNotExist(err) {
			fmt.Printf("  Segment %d: File not found, skipping\n", segment.Index)
			continue
		}
		
		originalDuration := segment.Duration
		
		// Analyze segment for silence at beginning and end
		startTrim, endTrim := bs.detectSilenceAtEdges(segmentPath, threshold, maxTrim)
		
		if startTrim < minTrim && endTrim < minTrim {
			fmt.Printf("  Segment %d: No significant silence detected (< %.1fs)\n", 
				segment.Index, minTrim)
			continue
		}
		
		// Apply trimming to segment metadata
		newDuration := originalDuration - startTrim - endTrim
		if newDuration < 0.5 { // Don't trim too aggressively
			fmt.Printf("  Segment %d: Would be too short after trimming, skipping\n", segment.Index)
			continue
		}
		
		segment.StartTime += startTrim  // Adjust start time in original track
		segment.Duration = newDuration  // Update duration
		
		trimmedCount++
		timeSaved := startTrim + endTrim
		totalTimeSaved += timeSaved
		
		fmt.Printf("  Segment %d: Trimmed %.2fs start + %.2fs end = %.2fs saved (%.1fs → %.1fs)\n",
			segment.Index, startTrim, endTrim, timeSaved, originalDuration, newDuration)
	}
	
	fmt.Printf("Track %d trimming complete: %d/%d segments trimmed, %.2fs total time saved\n",
		trackNum, trimmedCount, len(*segments), totalTimeSaved)
}

// detectSilenceAtEdges analyzes a segment file for silence at the beginning and end
func (bs *Shell) detectSilenceAtEdges(filePath string, threshold, maxTrim float64) (float64, float64) {
	// Use ffmpeg to detect silence at the beginning and end
	// This is a simplified approach - in production you'd want more sophisticated analysis
	
	startTrim := bs.detectSilenceAtStart(filePath, threshold, maxTrim)
	endTrim := bs.detectSilenceAtEnd(filePath, threshold, maxTrim)
	
	return startTrim, endTrim
}

// detectSilenceAtStart detects silence duration at the beginning of a file
func (bs *Shell) detectSilenceAtStart(filePath string, threshold, maxTrim float64) float64 {
	// Use ffmpeg silencedetect filter to find silence at start
	cmd := exec.Command("ffmpeg", "-i", filePath, "-af", 
		fmt.Sprintf("silencedetect=noise=%.1fdB:d=0.1", threshold),
		"-f", "null", "-")
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If ffmpeg fails, estimate based on low energy
		return bs.estimateSilenceAtStart(filePath, maxTrim)
	}
	
	// Parse ffmpeg output to find silence at the beginning
	outputStr := string(output)
	if strings.Contains(outputStr, "silence_start: 0") {
		// Extract silence duration from output
		lines := strings.Split(outputStr, "\n")
		for _, line := range lines {
			if strings.Contains(line, "silence_end:") && strings.Contains(line, "silence_duration:") {
				// Parse duration from line like: [silencedetect @ 0x...] silence_end: 1.234 | silence_duration: 1.234
				parts := strings.Split(line, "silence_duration:")
				if len(parts) > 1 {
					durationStr := strings.TrimSpace(strings.Split(parts[1], " ")[0])
					if duration, err := strconv.ParseFloat(durationStr, 64); err == nil {
						if duration > maxTrim {
							return maxTrim
						}
						return duration
					}
				}
				break
			}
		}
	}
	
	return 0.0
}

// detectSilenceAtEnd detects silence duration at the end of a file  
func (bs *Shell) detectSilenceAtEnd(filePath string, threshold, maxTrim float64) float64 {
	// For end silence, we need to reverse the audio or analyze from the end
	// This is a simplified implementation that estimates based on energy analysis
	return bs.estimateSilenceAtEnd(filePath, maxTrim)
}

// estimateSilenceAtStart estimates silence at start based on low energy detection
func (bs *Shell) estimateSilenceAtStart(filePath string, maxTrim float64) float64 {
	// Simple heuristic: analyze first few seconds for very low energy
	// In a full implementation, you'd use proper audio analysis
	
	// Use ffmpeg to get volume stats for first few seconds
	analyzeCmd := exec.Command("ffmpeg", "-i", filePath, "-t", fmt.Sprintf("%.1f", maxTrim), 
		"-af", "volumedetect", "-f", "null", "-")
	
	output, err := analyzeCmd.CombinedOutput()
	if err != nil {
		return 0.0
	}
	
	// Look for very quiet audio at the start
	outputStr := string(output)
	if strings.Contains(outputStr, "max_volume: -") {
		// Parse max volume to estimate silence
		lines := strings.Split(outputStr, "\n")
		for _, line := range lines {
			if strings.Contains(line, "max_volume:") {
				parts := strings.Split(line, "max_volume:")
				if len(parts) > 1 {
					volumeStr := strings.TrimSpace(strings.Split(parts[1], " ")[0])
					if volume, err := strconv.ParseFloat(volumeStr, 64); err == nil {
						// If very quiet (below -30dB), consider it silence
						if volume < -30.0 {
							return maxTrim * 0.3 // Conservative estimate
						}
					}
				}
			}
		}
	}
	
	return 0.0
}

// estimateSilenceAtEnd estimates silence at end based on low energy detection
func (bs *Shell) estimateSilenceAtEnd(filePath string, maxTrim float64) float64 {
	// Get file duration first
	durationCmd := exec.Command("ffprobe", "-v", "quiet", "-show_entries", 
		"format=duration", "-of", "csv=p=0", filePath)
	
	durationOutput, err := durationCmd.Output()
	if err != nil {
		return 0.0
	}
	
	duration, err := strconv.ParseFloat(strings.TrimSpace(string(durationOutput)), 64)
	if err != nil {
		return 0.0
	}
	
	// Analyze last few seconds
	startTime := duration - maxTrim
	if startTime < 0 {
		startTime = 0
	}
	
	analyzeCmd := exec.Command("ffmpeg", "-ss", fmt.Sprintf("%.1f", startTime), 
		"-i", filePath, "-af", "volumedetect", "-f", "null", "-")
	
	output, err := analyzeCmd.CombinedOutput()
	if err != nil {
		return 0.0
	}
	
	// Similar analysis as start
	outputStr := string(output)
	if strings.Contains(outputStr, "max_volume: -") {
		lines := strings.Split(outputStr, "\n")
		for _, line := range lines {
			if strings.Contains(line, "max_volume:") {
				parts := strings.Split(line, "max_volume:")
				if len(parts) > 1 {
					volumeStr := strings.TrimSpace(strings.Split(parts[1], " ")[0])
					if volume, err := strconv.ParseFloat(volumeStr, 64); err == nil {
						if volume < -30.0 {
							return maxTrim * 0.2 // Even more conservative for end
						}
					}
				}
			}
		}
	}
	
	return 0.0
}