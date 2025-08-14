package blend

// Helper functions for audio operations

// clamp constrains an integer value to be within the specified range
func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// clampFloat constrains a float64 value to be within the specified range
func clampFloat(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// findNearestBeat finds the closest beat to a given time
func (bs *Shell) findNearestBeat(beats []float64, targetTime float64) float64 {
	if len(beats) == 0 {
		return targetTime
	}
	
	minDiff := float64(1000000) // Large initial value
	nearest := targetTime
	
	for _, beat := range beats {
		diff := targetTime - beat
		if diff < 0 {
			diff = -diff
		}
		
		if diff < minDiff {
			minDiff = diff
			nearest = beat
		}
	}
	
	return nearest
}