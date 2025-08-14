package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strings"
)

func (bs *BlendShell) recordBlendBasic(ctx context.Context, startPosition1, startPosition2, maxAvailableDuration float64, outputFile string) {
	// Build ffmpeg command to mix two tracks and output to file
	ffmpegArgs := []string{
		"-y", // Overwrite output file
		"-ss", fmt.Sprintf("%.1f", startPosition1),
		"-i", bs.inputPath1,
		"-ss", fmt.Sprintf("%.1f", startPosition2), 
		"-i", bs.inputPath2,
		"-t", fmt.Sprintf("%.1f", maxAvailableDuration),
	}
	
	// Build filter complex for mixing with effects
	var filterComplex []string
	
	// Process track 1
	filter1 := "[0:a]"
	if bs.tempo1 != 0 {
		tempoMultiplier := 1.0 + (bs.tempo1 / 100.0)
		if tempoMultiplier > 0.5 && tempoMultiplier <= 2.0 {
			filter1 += fmt.Sprintf("atempo=%.6f,", tempoMultiplier)
		}
	}
	if bs.pitch1 != 0 {
		pitchSemitones := float64(bs.pitch1)
		filter1 += fmt.Sprintf("asetrate=44100*%.6f,aresample=44100,atempo=%.6f,", 
			math.Pow(2, pitchSemitones/12.0), 1.0/math.Pow(2, pitchSemitones/12.0))
	}
	if bs.volume1 != 100 {
		volumeMultiplier := bs.volume1 / 100.0
		filter1 += fmt.Sprintf("volume=%.6f,", volumeMultiplier)
	}
	// Remove trailing comma
	if strings.HasSuffix(filter1, ",") {
		filter1 = filter1[:len(filter1)-1]
	}
	filter1 += "[a1]"
	
	// Process track 2
	filter2 := "[1:a]"
	if bs.tempo2 != 0 {
		tempoMultiplier := 1.0 + (bs.tempo2 / 100.0)
		if tempoMultiplier > 0.5 && tempoMultiplier <= 2.0 {
			filter2 += fmt.Sprintf("atempo=%.6f,", tempoMultiplier)
		}
	}
	if bs.pitch2 != 0 {
		pitchSemitones := float64(bs.pitch2)
		filter2 += fmt.Sprintf("asetrate=44100*%.6f,aresample=44100,atempo=%.6f,", 
			math.Pow(2, pitchSemitones/12.0), 1.0/math.Pow(2, pitchSemitones/12.0))
	}
	if bs.volume2 != 100 {
		volumeMultiplier := bs.volume2 / 100.0
		filter2 += fmt.Sprintf("volume=%.6f,", volumeMultiplier)
	}
	// Remove trailing comma
	if strings.HasSuffix(filter2, ",") {
		filter2 = filter2[:len(filter2)-1]
	}
	filter2 += "[a2]"
	
	// Mix both processed tracks
	filterComplex = append(filterComplex, filter1)
	filterComplex = append(filterComplex, filter2) 
	filterComplex = append(filterComplex, "[a1][a2]amix=inputs=2[out]")
	
	ffmpegArgs = append(ffmpegArgs, "-filter_complex", strings.Join(filterComplex, ";"))
	ffmpegArgs = append(ffmpegArgs, "-map", "[out]")
	ffmpegArgs = append(ffmpegArgs, outputFile)
	
	cmd := exec.CommandContext(ctx, "ffmpeg", ffmpegArgs...)
	cmd.Run()
}

func (bs *BlendShell) recordBlendWithSegments(ctx context.Context, startPosition1, startPosition2, maxAvailableDuration float64, outputFile string) {
	// Build complex ffmpeg command that includes all active segments
	ffmpegArgs := []string{"-y"} // Overwrite output file
	
	// Add base tracks as inputs
	ffmpegArgs = append(ffmpegArgs, 
		"-ss", fmt.Sprintf("%.1f", startPosition1), "-i", bs.inputPath1,
		"-ss", fmt.Sprintf("%.1f", startPosition2), "-i", bs.inputPath2)
	
	inputIndex := 2
	var segmentFilters []string
	
	// Add active segments from track 1 as inputs
	for _, seg := range bs.segments1 {
		if !seg.Active {
			continue
		}
		
		// Check if segment should play during our playback window
		segmentStart := seg.Placement
		segmentEnd := seg.Placement + seg.Duration
		playbackEnd := startPosition1 + maxAvailableDuration
		
		if segmentEnd < startPosition1 || segmentStart > playbackEnd {
			continue
		}
		
		segmentPath := fmt.Sprintf("%s/part_%03d.wav", bs.segmentsDir1, seg.Index)
		if _, err := os.Stat(segmentPath); os.IsNotExist(err) {
			continue
		}
		
		// Calculate delay and seek
		var delay float64 = segmentStart - startPosition1
		var seekTime float64 = 0
		
		if delay < 0 {
			seekTime = -delay
			delay = 0
		}
		
		// Add segment as input
		if seekTime > 0 {
			ffmpegArgs = append(ffmpegArgs, "-ss", fmt.Sprintf("%.1f", seekTime))
		}
		ffmpegArgs = append(ffmpegArgs, "-i", segmentPath)
		
		// Create filter for this segment with delay
		segmentFilter := fmt.Sprintf("[%d:a]adelay=%d|%d[seg%d]", 
			inputIndex, int(delay*1000), int(delay*1000), inputIndex)
		segmentFilters = append(segmentFilters, segmentFilter)
		inputIndex++
	}
	
	// Add active segments from track 2 as inputs
	for _, seg := range bs.segments2 {
		if !seg.Active {
			continue
		}
		
		// Check if segment should play during our playback window
		segmentStart := seg.Placement
		segmentEnd := seg.Placement + seg.Duration
		playbackEnd := startPosition2 + maxAvailableDuration
		
		if segmentEnd < startPosition2 || segmentStart > playbackEnd {
			continue
		}
		
		segmentPath := fmt.Sprintf("%s/part_%03d.wav", bs.segmentsDir2, seg.Index)
		if _, err := os.Stat(segmentPath); os.IsNotExist(err) {
			continue
		}
		
		// Calculate delay and seek
		var delay float64 = segmentStart - startPosition2
		var seekTime float64 = 0
		
		if delay < 0 {
			seekTime = -delay
			delay = 0
		}
		
		// Add segment as input
		if seekTime > 0 {
			ffmpegArgs = append(ffmpegArgs, "-ss", fmt.Sprintf("%.1f", seekTime))
		}
		ffmpegArgs = append(ffmpegArgs, "-i", segmentPath)
		
		// Create filter for this segment with delay
		segmentFilter := fmt.Sprintf("[%d:a]adelay=%d|%d[seg%d]", 
			inputIndex, int(delay*1000), int(delay*1000), inputIndex)
		segmentFilters = append(segmentFilters, segmentFilter)
		inputIndex++
	}
	
	// Set duration
	ffmpegArgs = append(ffmpegArgs, "-t", fmt.Sprintf("%.1f", maxAvailableDuration))
	
	// Build filter complex
	var filterComplex []string
	
	// Process base track 1
	filter1 := "[0:a]"
	if bs.tempo1 != 0 {
		tempoMultiplier := 1.0 + (bs.tempo1 / 100.0)
		if tempoMultiplier > 0.5 && tempoMultiplier <= 2.0 {
			filter1 += fmt.Sprintf("atempo=%.6f,", tempoMultiplier)
		}
	}
	if bs.pitch1 != 0 {
		pitchSemitones := float64(bs.pitch1)
		filter1 += fmt.Sprintf("asetrate=44100*%.6f,aresample=44100,atempo=%.6f,", 
			math.Pow(2, pitchSemitones/12.0), 1.0/math.Pow(2, pitchSemitones/12.0))
	}
	if bs.volume1 != 100 {
		volumeMultiplier := bs.volume1 / 100.0
		filter1 += fmt.Sprintf("volume=%.6f,", volumeMultiplier)
	}
	if strings.HasSuffix(filter1, ",") {
		filter1 = filter1[:len(filter1)-1]
	}
	filter1 += "[a1]"
	
	// Process base track 2
	filter2 := "[1:a]"
	if bs.tempo2 != 0 {
		tempoMultiplier := 1.0 + (bs.tempo2 / 100.0)
		if tempoMultiplier > 0.5 && tempoMultiplier <= 2.0 {
			filter2 += fmt.Sprintf("atempo=%.6f,", tempoMultiplier)
		}
	}
	if bs.pitch2 != 0 {
		pitchSemitones := float64(bs.pitch2)
		filter2 += fmt.Sprintf("asetrate=44100*%.6f,aresample=44100,atempo=%.6f,", 
			math.Pow(2, pitchSemitones/12.0), 1.0/math.Pow(2, pitchSemitones/12.0))
	}
	if bs.volume2 != 100 {
		volumeMultiplier := bs.volume2 / 100.0
		filter2 += fmt.Sprintf("volume=%.6f,", volumeMultiplier)
	}
	if strings.HasSuffix(filter2, ",") {
		filter2 = filter2[:len(filter2)-1]
	}
	filter2 += "[a2]"
	
	// Add all filters
	filterComplex = append(filterComplex, filter1)
	filterComplex = append(filterComplex, filter2)
	filterComplex = append(filterComplex, segmentFilters...)
	
	// Create mix command
	mixInputs := "[a1][a2]"
	for i := 2; i < inputIndex; i++ {
		mixInputs += fmt.Sprintf("[seg%d]", i)
	}
	mixCommand := fmt.Sprintf("%samix=inputs=%d[out]", mixInputs, inputIndex)
	filterComplex = append(filterComplex, mixCommand)
	
	ffmpegArgs = append(ffmpegArgs, "-filter_complex", strings.Join(filterComplex, ";"))
	ffmpegArgs = append(ffmpegArgs, "-map", "[out]")
	ffmpegArgs = append(ffmpegArgs, outputFile)
	
	cmd := exec.CommandContext(ctx, "ffmpeg", ffmpegArgs...)
	cmd.Run()
}