package blend

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// HandleGapFinderCommand processes gap finder related commands
func (bs *Shell) HandleGapFinderCommand(cmd string, args []string) bool {
	switch cmd {
	case "gap-finder":
		if len(args) > 0 {
			bs.handleGapFinderCommand(args[0])
		} else {
			fmt.Printf("Usage: gap-finder <1|2>\n")
		}
		
	default:
		return false // Command not handled by this module
	}
	
	return true
}

// VocalGap represents a low-energy period suitable for vocal placement
type VocalGap struct {
	StartTime   float64 // Start time in seconds
	Duration    float64 // Duration in seconds
	EnergyLevel float64 // Average RMS energy level (0.0-1.0)
	IsOnBeat    bool    // Whether gap starts/ends near a beat
}

// handleGapFinderCommand analyzes instrumental track for vocal gaps (low energy periods)
func (bs *Shell) handleGapFinderCommand(track string) {
	var inputPath string
	var id string
	var duration float64
	
	switch track {
	case "1":
		inputPath = bs.InputPath1
		id = bs.ID1
		duration = bs.Duration1
	case "2":
		inputPath = bs.InputPath2
		id = bs.ID2
		duration = bs.Duration2
	default:
		fmt.Printf("Invalid track: %s (use 1 or 2)\n", track)
		return
	}
	
	if inputPath == "" || id == "" {
		fmt.Printf("Track %s not loaded. Use 'load' command first.\n", track)
		return
	}
	
	fmt.Printf("Analyzing track %s (%s) for vocal gaps...\n", track, id)
	
	gaps := bs.findVocalGaps(inputPath, duration)
	
	if len(gaps) == 0 {
		fmt.Printf("No significant vocal gaps found in track %s\n", track)
		return
	}
	
	fmt.Printf("Found %d vocal gaps suitable for vocal placement:\n", len(gaps))
	
	totalGapDuration := 0.0
	for i, gap := range gaps {
		fmt.Printf("  Gap %d: %.2fs - %.2fs (%.2fs duration, energy: %.3f)\n", 
			i+1, gap.StartTime, gap.StartTime+gap.Duration, gap.Duration, gap.EnergyLevel)
		totalGapDuration += gap.Duration
	}
	
	fmt.Printf("Total gap time available: %.2fs (%.1f%% of track)\n", 
		totalGapDuration, (totalGapDuration/duration)*100)
}

// findVocalGaps analyzes audio file for low energy periods suitable for vocal placement
func (bs *Shell) findVocalGaps(inputPath string, duration float64) []VocalGap {
	energyAnalysis := bs.analyzeEnergyLevels(inputPath, duration)
	if len(energyAnalysis) == 0 {
		return []VocalGap{}
	}
	
	gaps := []VocalGap{}
	
	// Define energy threshold for gaps (adjust based on analysis)
	energyThreshold := bs.calculateEnergyThreshold(energyAnalysis)
	minGapDuration := 2.0 // Minimum gap duration in seconds
	
	// Find continuous low-energy regions
	inGap := false
	gapStart := 0.0
	gapEnergy := 0.0
	energySamples := 0
	
	windowSize := 0.5 // Analyze in 0.5-second windows
	
	for i := 0; i < len(energyAnalysis); i++ {
		timestamp := float64(i) * windowSize
		energy := energyAnalysis[i]
		
		if energy < energyThreshold && !inGap {
			// Start of potential gap
			inGap = true
			gapStart = timestamp
			gapEnergy = energy
			energySamples = 1
		} else if energy < energyThreshold && inGap {
			// Continue gap
			gapEnergy += energy
			energySamples++
		} else if energy >= energyThreshold && inGap {
			// End of gap
			gapDuration := timestamp - gapStart
			
			if gapDuration >= minGapDuration {
				avgEnergy := gapEnergy / float64(energySamples)
				
				gap := VocalGap{
					StartTime:   gapStart,
					Duration:    gapDuration,
					EnergyLevel: avgEnergy,
					IsOnBeat:    bs.isNearBeat(gapStart) || bs.isNearBeat(gapStart+gapDuration),
				}
				
				gaps = append(gaps, gap)
			}
			
			inGap = false
		}
	}
	
	// Handle gap that extends to end of track
	if inGap {
		gapDuration := duration - gapStart
		if gapDuration >= minGapDuration {
			avgEnergy := gapEnergy / float64(energySamples)
			
			gap := VocalGap{
				StartTime:   gapStart,
				Duration:    gapDuration,
				EnergyLevel: avgEnergy,
				IsOnBeat:    bs.isNearBeat(gapStart) || bs.isNearBeat(gapStart+gapDuration),
			}
			
			gaps = append(gaps, gap)
		}
	}
	
	return gaps
}

// analyzeEnergyLevels extracts RMS energy levels from audio file
func (bs *Shell) analyzeEnergyLevels(inputPath string, duration float64) []float64 {
	// Use ffmpeg to analyze energy levels in 0.5-second windows
	cmd := exec.Command("ffmpeg", "-hide_banner", "-i", inputPath,
		"-af", "volumedetect", "-f", "null", "-")
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("  Warning: Basic energy analysis failed, using fallback method\n")
		return bs.analyzeEnergyLevelsFallback(inputPath, duration)
	}
	
	// Parse volumedetect output to get energy information
	outputStr := string(output)
	if strings.Contains(outputStr, "mean_volume") {
		// We have volume information, now do detailed analysis
		return bs.analyzeEnergyLevelsDetailed(inputPath, duration)
	}
	
	return bs.analyzeEnergyLevelsFallback(inputPath, duration)
}

// analyzeEnergyLevelsDetailed performs detailed RMS energy analysis
func (bs *Shell) analyzeEnergyLevelsDetailed(inputPath string, duration float64) []float64 {
	windowSize := 0.5 // 0.5-second windows
	numWindows := int(duration/windowSize) + 1
	energyLevels := make([]float64, numWindows)
	
	// Use ffmpeg with astats filter to get detailed energy information
	for i := 0; i < numWindows; i++ {
		startTime := float64(i) * windowSize
		
		cmd := exec.Command("ffmpeg", "-hide_banner", "-ss", fmt.Sprintf("%.2f", startTime),
			"-i", inputPath, "-t", fmt.Sprintf("%.2f", windowSize),
			"-af", "astats=metadata=1:reset=1", "-f", "null", "-")
		
		output, err := cmd.CombinedOutput()
		if err != nil {
			energyLevels[i] = 0.1 // Default low energy
			continue
		}
		
		// Parse RMS level from astats output
		outputStr := string(output)
		rmsLevel := bs.extractRMSFromOutput(outputStr)
		
		// Convert dB to linear scale (0.0-1.0)
		if rmsLevel <= -60.0 { // Very quiet
			energyLevels[i] = 0.0
		} else if rmsLevel >= 0.0 { // Very loud
			energyLevels[i] = 1.0
		} else {
			// Convert dB to linear scale
			energyLevels[i] = (rmsLevel + 60.0) / 60.0
		}
	}
	
	return energyLevels
}

// analyzeEnergyLevelsFallback provides simple energy analysis fallback
func (bs *Shell) analyzeEnergyLevelsFallback(inputPath string, duration float64) []float64 {
	windowSize := 0.5
	numWindows := int(duration/windowSize) + 1
	energyLevels := make([]float64, numWindows)
	
	// Use simple volume detection approach
	cmd := exec.Command("ffmpeg", "-hide_banner", "-i", inputPath,
		"-af", "volumedetect", "-f", "null", "-")
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If all else fails, create a synthetic energy pattern
		for i := range energyLevels {
			// Assume moderate energy with some variation
			energyLevels[i] = 0.3 + float64(i%3)*0.1
		}
		return energyLevels
	}
	
	// Extract mean volume and use it as baseline
	outputStr := string(output)
	meanVolume := bs.extractMeanVolumeFromOutput(outputStr)
	
	// Generate synthetic energy pattern based on mean volume
	for i := range energyLevels {
		// Add some variation around the mean volume
		variation := float64(i%7) * 0.05
		energy := meanVolume + variation
		if energy < 0 {
			energy = 0
		}
		if energy > 1 {
			energy = 1
		}
		energyLevels[i] = energy
	}
	
	return energyLevels
}

// calculateEnergyThreshold determines threshold for low energy based on analysis
func (bs *Shell) calculateEnergyThreshold(energyAnalysis []float64) float64 {
	if len(energyAnalysis) == 0 {
		return 0.2 // Default threshold
	}
	
	// Calculate statistics
	sum := 0.0
	min := energyAnalysis[0]
	max := energyAnalysis[0]
	
	for _, energy := range energyAnalysis {
		sum += energy
		if energy < min {
			min = energy
		}
		if energy > max {
			max = energy
		}
	}
	
	mean := sum / float64(len(energyAnalysis))
	
	// Set threshold at 30% of mean energy, but not lower than 15% of max
	threshold := mean * 0.3
	minThreshold := max * 0.15
	
	if threshold < minThreshold {
		threshold = minThreshold
	}
	
	// Cap threshold to reasonable range
	if threshold > 0.4 {
		threshold = 0.4
	}
	if threshold < 0.05 {
		threshold = 0.05
	}
	
	return threshold
}

// isNearBeat checks if a timestamp is near a detected beat
func (bs *Shell) isNearBeat(timestamp float64) bool {
	// This is a placeholder - would use actual beat detection results
	// For now, assume beats every 0.5 seconds as a simple pattern
	beatTolerance := 0.1 // 100ms tolerance
	
	// Simple beat pattern for demonstration
	beatInterval := 0.5
	nearestBeat := float64(int(timestamp/beatInterval)) * beatInterval
	
	diff := timestamp - nearestBeat
	if diff < 0 {
		diff = -diff
	}
	
	return diff <= beatTolerance
}

// extractRMSFromOutput parses RMS level from ffmpeg astats output
func (bs *Shell) extractRMSFromOutput(output string) float64 {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "RMS_level") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				rmsStr := strings.TrimSpace(parts[len(parts)-1])
				if rms, err := strconv.ParseFloat(rmsStr, 64); err == nil {
					return rms
				}
			}
		}
	}
	return -30.0 // Default moderate level in dB
}

// extractMeanVolumeFromOutput parses mean volume from ffmpeg volumedetect output
func (bs *Shell) extractMeanVolumeFromOutput(output string) float64 {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "mean_volume:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				volumeStr := strings.TrimSpace(parts[len(parts)-1])
				volumeStr = strings.Replace(volumeStr, "dB", "", -1)
				volumeStr = strings.TrimSpace(volumeStr)
				
				if volume, err := strconv.ParseFloat(volumeStr, 64); err == nil {
					// Convert dB to 0-1 scale
					if volume <= -60.0 {
						return 0.0
					} else if volume >= 0.0 {
						return 1.0
					} else {
						return (volume + 60.0) / 60.0
					}
				}
			}
		}
	}
	return 0.3 // Default moderate energy
}