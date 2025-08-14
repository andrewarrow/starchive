package blend

import (
	"fmt"
	"os"
	"os/exec"
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