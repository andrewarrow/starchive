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

// HandleSegmentAdvancedCommand processes advanced segment manipulation commands
func (bs *Shell) HandleSegmentAdvancedCommand(cmd string, args []string) bool {
	switch cmd {
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
		
	case "smart-random":
		if len(args) > 0 {
			bs.handleSmartRandomCommand(args[0])
		} else {
			fmt.Printf("Usage: smart-random <1|2>\n")
		}
		
	default:
		return false // Command not handled by this module
	}
	
	return true
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

// handleSmartRandomCommand intelligently places segments with beat alignment and collision avoidance
func (bs *Shell) handleSmartRandomCommand(trackNum string) {
	var segments *[]VocalSegment
	var beats []float64
	var targetDuration float64
	var otherSegments *[]VocalSegment
	var id string
	
	switch trackNum {
	case "1":
		segments = &bs.Segments1
		beats = bs.Beats2  // Align to beats of target track (track 2)
		targetDuration = bs.Duration2
		otherSegments = &bs.Segments2  // Check for collisions with track 2
		id = bs.ID1
	case "2":
		segments = &bs.Segments2
		beats = bs.Beats1  // Align to beats of target track (track 1)
		targetDuration = bs.Duration1
		otherSegments = &bs.Segments1  // Check for collisions with track 1
		id = bs.ID2
	default:
		fmt.Printf("Invalid track number: %s (use 1 or 2)\n", trackNum)
		return
	}
	
	if len(*segments) == 0 {
		fmt.Printf("No segments found for track %s. Run 'split %s' first.\n", trackNum, trackNum)
		return
	}
	
	if len(beats) == 0 {
		fmt.Printf("No beats detected for target track. Run 'beat-detect' first.\n")
		return
	}
	
	fmt.Printf("Smart-placing %d segments from track %s (%s) with beat alignment and collision avoidance...\n", 
		len(*segments), trackNum, id)
	
	// Reset all segments to inactive first
	for i := range *segments {
		(*segments)[i].Active = false
		(*segments)[i].Placement = 0
	}
	
	// Filter beats to usable range (first 80% of track to avoid cutoffs)
	maxTime := targetDuration * 0.8
	usableBeats := make([]float64, 0)
	for _, beat := range beats {
		if beat <= maxTime {
			usableBeats = append(usableBeats, beat)
		}
	}
	
	if len(usableBeats) == 0 {
		fmt.Printf("No usable beats found in target time range\n")
		return
	}
	
	fmt.Printf("Found %d usable beats in %.1fs timeframe\n", len(usableBeats), maxTime)
	
	// Smart placement algorithm
	placedCount := 0
	maxAttempts := len(*segments) * 10 // Allow multiple attempts per segment
	
	// Seed random generator
	rand.Seed(time.Now().UnixNano())
	
	// Try to place each segment
	for i := range *segments {
		segment := &(*segments)[i]
		placed := false
		attempts := 0
		
		// Try to find a good placement for this segment
		for attempts < maxAttempts && !placed {
			// Pick a random beat position
			beatIdx := rand.Intn(len(usableBeats))
			candidateTime := usableBeats[beatIdx]
			
			// Check if this placement would cause conflicts
			if bs.wouldCauseConflict(segment, candidateTime, segments, otherSegments) {
				attempts++
				continue
			}
			
			// Good placement found!
			segment.Placement = candidateTime
			segment.Active = true
			placed = true
			placedCount++
			
			fmt.Printf("  %s:%d placed at beat %.1fs (beat %d/%d)\n", 
				trackNum, segment.Index, candidateTime, beatIdx+1, len(usableBeats))
		}
		
		if !placed {
			fmt.Printf("  %s:%d could not be placed without conflicts (tried %d positions)\n", 
				trackNum, segment.Index, attempts)
		}
	}
	
	fmt.Printf("Smart-random placement complete: %d/%d segments placed successfully\n", 
		placedCount, len(*segments))
	
	// Show collision summary
	if placedCount < len(*segments) {
		fmt.Printf("💡 Tip: Use 'gap-finder' to find better placement opportunities\n")
	}
}

// wouldCauseConflict checks if placing a segment at a given time would cause conflicts
func (bs *Shell) wouldCauseConflict(segment *VocalSegment, placementTime float64, 
	sameTrackSegments, otherTrackSegments *[]VocalSegment) bool {
	
	segmentStart := placementTime
	segmentEnd := placementTime + segment.Duration
	
	// Check conflicts with other segments on the same track
	for _, otherSeg := range *sameTrackSegments {
		if !otherSeg.Active || otherSeg.Index == segment.Index {
			continue
		}
		
		otherStart := otherSeg.Placement
		otherEnd := otherSeg.Placement + otherSeg.Duration
		
		// Check for overlap
		if !(segmentEnd <= otherStart || segmentStart >= otherEnd) {
			return true // Conflict detected
		}
	}
	
	// Check conflicts with segments on the other track (if both are vocal tracks)
	if bs.Type1 == "V" && bs.Type2 == "V" {
		for _, otherSeg := range *otherTrackSegments {
			if !otherSeg.Active {
				continue
			}
			
			otherStart := otherSeg.Placement
			otherEnd := otherSeg.Placement + otherSeg.Duration
			
			// Check for overlap (more strict for vocal-vocal conflicts)
			minGap := 0.5 // Require at least 0.5s gap between vocals
			if !(segmentEnd + minGap <= otherStart || segmentStart >= otherEnd + minGap) {
				return true // Vocal conflict detected
			}
		}
	}
	
	return false // No conflicts
}