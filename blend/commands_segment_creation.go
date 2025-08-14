package blend

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	
	"starchive/audio"
)

// HandleSegmentCreationCommand processes segment creation and listing commands
func (bs *Shell) HandleSegmentCreationCommand(cmd string, args []string) bool {
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
		
	case "analyze-segments":
		if len(args) > 0 {
			bs.handleAnalyzeSegmentsCommand(args[0])
		} else {
			fmt.Printf("Usage: analyze-segments <1|2>\n")
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
			energyInfo := ""
			if seg.EnergyCategory != "" {
				energyInfo = fmt.Sprintf(" [%s energy: %.3f RMS]", seg.EnergyCategory, seg.RMSEnergy)
			}
			fmt.Printf("  1:%d - %.2fs to %.2fs (%s)%s\n", i+1, seg.StartTime, endTime, status, energyInfo)
		}
		fmt.Printf("Track 2 segments: %d total\n", len(bs.Segments2))
		for i, seg := range bs.Segments2 {
			status := "inactive"
			if seg.Active {
				status = "active"
			}
			endTime := seg.StartTime + seg.Duration
			energyInfo := ""
			if seg.EnergyCategory != "" {
				energyInfo = fmt.Sprintf(" [%s energy: %.3f RMS]", seg.EnergyCategory, seg.RMSEnergy)
			}
			fmt.Printf("  2:%d - %.2fs to %.2fs (%s)%s\n", i+1, seg.StartTime, endTime, status, energyInfo)
		}
	} else if track == "1" {
		fmt.Printf("Track 1 segments: %d total\n", len(bs.Segments1))
		for i, seg := range bs.Segments1 {
			status := "inactive"
			if seg.Active {
				status = "active"
			}
			endTime := seg.StartTime + seg.Duration
			energyInfo := ""
			if seg.EnergyCategory != "" {
				energyInfo = fmt.Sprintf(" [%s energy: %.3f RMS]", seg.EnergyCategory, seg.RMSEnergy)
			}
			fmt.Printf("  1:%d - %.2fs to %.2fs (%s)%s\n", i+1, seg.StartTime, endTime, status, energyInfo)
		}
	} else if track == "2" {
		fmt.Printf("Track 2 segments: %d total\n", len(bs.Segments2))
		for i, seg := range bs.Segments2 {
			status := "inactive"
			if seg.Active {
				status = "active"
			}
			endTime := seg.StartTime + seg.Duration
			energyInfo := ""
			if seg.EnergyCategory != "" {
				energyInfo = fmt.Sprintf(" [%s energy: %.3f RMS]", seg.EnergyCategory, seg.RMSEnergy)
			}
			fmt.Printf("  2:%d - %.2fs to %.2fs (%s)%s\n", i+1, seg.StartTime, endTime, status, energyInfo)
		}
	} else {
		fmt.Printf("Invalid track: %s (use 1 or 2)\n", track)
	}
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
				RMSEnergy: 0.0,       // Will be populated by analyze-segments
				PeakLevel: 0.0,       // Will be populated by analyze-segments
				EnergyCategory: "",   // Will be populated by analyze-segments
			}
			
			*segments = append(*segments, segment)
			startTime += duration
		}
	}
}

// handleAnalyzeSegmentsCommand analyzes energy levels of segments
func (bs *Shell) handleAnalyzeSegmentsCommand(trackNum string) {
	var segments *[]VocalSegment
	var segmentsDir string
	var id string
	
	switch trackNum {
	case "1":
		segments = &bs.Segments1
		segmentsDir = bs.SegmentsDir1
		id = bs.ID1
	case "2":
		segments = &bs.Segments2
		segmentsDir = bs.SegmentsDir2
		id = bs.ID2
	default:
		fmt.Printf("Invalid track number: %s (use 1 or 2)\n", trackNum)
		return
	}
	
	if len(*segments) == 0 {
		fmt.Printf("No segments found for track %s. Run 'split %s' first.\n", trackNum, trackNum)
		return
	}
	
	fmt.Printf("Analyzing energy levels for %d segments in track %s (%s)...\n", len(*segments), trackNum, id)
	
	// Collect all RMS and peak values to determine thresholds
	var rmsValues, peakValues []float64
	
	for i := range *segments {
		segment := &(*segments)[i]
		segmentPath := fmt.Sprintf("%s/part_%03d.wav", segmentsDir, segment.Index)
		
		if _, err := os.Stat(segmentPath); os.IsNotExist(err) {
			fmt.Printf("  Segment %d: file not found\n", segment.Index)
			continue
		}
		
		// Use ffprobe to get audio statistics
		rms, peak, err := bs.getAudioStatistics(segmentPath)
		if err != nil {
			fmt.Printf("  Segment %d: analysis failed - %v\n", segment.Index, err)
			continue
		}
		
		segment.RMSEnergy = rms
		segment.PeakLevel = peak
		rmsValues = append(rmsValues, rms)
		peakValues = append(peakValues, peak)
		
		fmt.Printf("  Segment %d: RMS=%.3f, Peak=%.3f\n", segment.Index, rms, peak)
	}
	
	// Calculate thresholds for categorization (tertiles)
	if len(rmsValues) > 0 {
		rmsThresholds := bs.calculateThresholds(rmsValues)
		
		// Categorize segments based on RMS energy
		for i := range *segments {
			segment := &(*segments)[i]
			if segment.RMSEnergy <= rmsThresholds[0] {
				segment.EnergyCategory = "low"
			} else if segment.RMSEnergy <= rmsThresholds[1] {
				segment.EnergyCategory = "medium"
			} else {
				segment.EnergyCategory = "high"
			}
		}
		
		// Show summary
		low, medium, high := 0, 0, 0
		for _, segment := range *segments {
			switch segment.EnergyCategory {
			case "low":
				low++
			case "medium":
				medium++
			case "high":
				high++
			}
		}
		
		fmt.Printf("Energy analysis complete: %d low, %d medium, %d high energy segments\n", low, medium, high)
	}
}

// getAudioStatistics uses ffprobe to get RMS and peak levels
func (bs *Shell) getAudioStatistics(filePath string) (float64, float64, error) {
	// Use ffprobe with astats filter to get audio statistics
	cmd := exec.Command("ffprobe", "-hide_banner", "-v", "quiet", 
		"-f", "lavfi", "-i", fmt.Sprintf("amovie=%s,astats=metadata=1:reset=1", filePath),
		"-show_entries", "frame_tags=lavfi.astats.Overall.RMS_level,lavfi.astats.Overall.Peak_level",
		"-of", "json")
	
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}
	
	// Parse JSON output
	var result struct {
		Frames []struct {
			Tags struct {
				RMSLevel  string `json:"lavfi.astats.Overall.RMS_level"`
				PeakLevel string `json:"lavfi.astats.Overall.Peak_level"`
			} `json:"tags"`
		} `json:"frames"`
	}
	
	if err := json.Unmarshal(output, &result); err != nil {
		return 0, 0, err
	}
	
	if len(result.Frames) == 0 {
		return 0, 0, fmt.Errorf("no audio statistics found")
	}
	
	// Get the last frame's statistics (overall)
	lastFrame := result.Frames[len(result.Frames)-1]
	
	rmsStr := lastFrame.Tags.RMSLevel
	peakStr := lastFrame.Tags.PeakLevel
	
	// Convert from dB to linear scale (0.0-1.0)
	rms, err := strconv.ParseFloat(rmsStr, 64)
	if err != nil {
		return 0, 0, err
	}
	
	peak, err := strconv.ParseFloat(peakStr, 64)
	if err != nil {
		return 0, 0, err
	}
	
	// Convert dB to linear (dB values are negative, convert to 0-1 range)
	rmsLinear := math.Pow(10, rms/20)
	peakLinear := math.Pow(10, peak/20)
	
	// Clamp to 0-1 range
	if rmsLinear > 1.0 {
		rmsLinear = 1.0
	}
	if peakLinear > 1.0 {
		peakLinear = 1.0
	}
	
	return rmsLinear, peakLinear, nil
}

// calculateThresholds calculates tertile thresholds for categorization
func (bs *Shell) calculateThresholds(values []float64) [2]float64 {
	if len(values) < 3 {
		// Not enough data for tertiles, use simple thresholds
		return [2]float64{0.33, 0.66}
	}
	
	// Sort values
	sorted := make([]float64, len(values))
	copy(sorted, values)
	
	// Simple bubble sort for small arrays
	for i := 0; i < len(sorted); i++ {
		for j := 0; j < len(sorted)-1-i; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}
	
	// Calculate tertile boundaries
	third := len(sorted) / 3
	twoThirds := (len(sorted) * 2) / 3
	
	return [2]float64{sorted[third], sorted[twoThirds]}
}