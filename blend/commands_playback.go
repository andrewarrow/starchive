package blend

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// HandlePlaybackCommand processes playback-related commands
func (bs *Shell) HandlePlaybackCommand(cmd string, args []string) bool {
	switch cmd {
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
		
	default:
		return false // Command not handled by this module
	}
	
	return true
}

// handlePlayCommand plays the blend
func (bs *Shell) handlePlayCommand(startFrom float64) {
	var startPosition1, startPosition2 float64
	
	if startFrom < 0 {
		// Use middle + window offsets (default behavior)
		startPosition1 = (bs.Duration1 / 2) + bs.Window1
		startPosition2 = (bs.Duration2 / 2) + bs.Window2
	} else {
		// Use specified position + window offsets
		startPosition1 = startFrom + bs.Window1
		startPosition2 = startFrom + bs.Window2
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

	// Check for active segments
	activeSegments1 := 0
	activeSegments2 := 0
	for _, seg := range bs.Segments1 {
		if seg.Active { activeSegments1++ }
	}
	for _, seg := range bs.Segments2 {
		if seg.Active { activeSegments2++ }
	}

	if activeSegments1 > 0 || activeSegments2 > 0 {
		fmt.Printf("Playing blend with %d+%d active segments... Press any key to stop.\n", activeSegments1, activeSegments2)
		bs.playBlendWithSegments(startPosition1, startPosition2, maxAvailableDuration)
	} else {
		fmt.Printf("Playing blend... Press any key to stop.\n")
		bs.playBlendBasic(startPosition1, startPosition2, maxAvailableDuration)
	}
}

func (bs *Shell) playBlendBasic(startPosition1, startPosition2, maxAvailableDuration float64) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Generate output filename with timestamp
	outputFile := fmt.Sprintf("./data/blend_%s_%s_%d.wav", bs.ID1, bs.ID2, time.Now().Unix())
	
	// Start recording the mix to file
	go bs.recordBlendBasic(ctx, startPosition1, startPosition2, maxAvailableDuration, outputFile)

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
	fmt.Printf("Playback stopped. Mix saved to %s\n", outputFile)
}

func (bs *Shell) playBlendWithSegments(startPosition1, startPosition2, maxAvailableDuration float64) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Generate output filename with timestamp
	outputFile := fmt.Sprintf("./data/blend_%s_%s_%d.wav", bs.ID1, bs.ID2, time.Now().Unix())
	
	// Start recording the mix to file
	go bs.recordBlendWithSegments(ctx, startPosition1, startPosition2, maxAvailableDuration, outputFile)

	// Play base tracks
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
	
	// Play active segments
	bs.playActiveSegments(ctx, startPosition1, startPosition2, maxAvailableDuration)

	// Wait for any key press
	go func() {
		var input string
		fmt.Scanf("%s", &input)
		cancel()
	}()

	<-ctx.Done()
	fmt.Printf("Playback stopped. Mix saved to %s\n", outputFile)
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

// playActiveSegments plays vocal segments at their designated placement times
func (bs *Shell) playActiveSegments(ctx context.Context, startPosition1, startPosition2, maxAvailableDuration float64) {
	
	// Play active segments from track 1
	for _, seg := range bs.Segments1 {
		if !seg.Active {
			continue
		}
		
		// Check if segment should play during our playback window
		segmentStart := seg.Placement
		segmentEnd := seg.Placement + seg.Duration
		playbackEnd := startPosition1 + maxAvailableDuration
		
		// Skip if segment is completely outside our playback window
		if segmentEnd < startPosition1 || segmentStart > playbackEnd {
			continue
		}
		
		// Calculate delay and duration for this segment
		var delay float64
		var segmentDuration float64 = seg.Duration
		
		if segmentStart >= startPosition1 {
			delay = segmentStart - startPosition1
		} else {
			// Segment started before our window, need to seek into it
			delay = 0
			segmentDuration = segmentEnd - startPosition1
		}
		
		// Launch segment playback with delay
		go func(segment VocalSegment, delaySeconds float64, duration float64) {
			if delaySeconds > 0 {
				select {
				case <-time.After(time.Duration(delaySeconds * 1000) * time.Millisecond):
				case <-ctx.Done():
					return
				}
			}
			
			segmentPath := fmt.Sprintf("%s/part_%03d.wav", bs.SegmentsDir1, segment.Index)
			if _, err := os.Stat(segmentPath); os.IsNotExist(err) {
				return
			}
			
			cmd := exec.CommandContext(ctx, "ffplay", 
				"-t", fmt.Sprintf("%.1f", duration),
				"-autoexit", "-nodisp", "-loglevel", "quiet", 
				segmentPath)
			cmd.Run()
		}(seg, delay, segmentDuration)
	}
	
	// Play active segments from track 2
	for _, seg := range bs.Segments2 {
		if !seg.Active {
			continue
		}
		
		// Check if segment should play during our playback window
		segmentStart := seg.Placement
		segmentEnd := seg.Placement + seg.Duration
		playbackEnd := startPosition2 + maxAvailableDuration
		
		// Skip if segment is completely outside our playback window
		if segmentEnd < startPosition2 || segmentStart > playbackEnd {
			continue
		}
		
		// Calculate delay and duration for this segment
		var delay float64
		var segmentDuration float64 = seg.Duration
		
		if segmentStart >= startPosition2 {
			delay = segmentStart - startPosition2
		} else {
			// Segment started before our window, need to seek into it
			delay = 0
			segmentDuration = segmentEnd - startPosition2
		}
		
		// Launch segment playback with delay
		go func(segment VocalSegment, delaySeconds float64, duration float64) {
			if delaySeconds > 0 {
				select {
				case <-time.After(time.Duration(delaySeconds * 1000) * time.Millisecond):
				case <-ctx.Done():
					return
				}
			}
			
			segmentPath := fmt.Sprintf("%s/part_%03d.wav", bs.SegmentsDir2, segment.Index)
			if _, err := os.Stat(segmentPath); os.IsNotExist(err) {
				return
			}
			
			cmd := exec.CommandContext(ctx, "ffplay", 
				"-t", fmt.Sprintf("%.1f", duration),
				"-autoexit", "-nodisp", "-loglevel", "quiet", 
				segmentPath)
			cmd.Run()
		}(seg, delay, segmentDuration)
	}
}