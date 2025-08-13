package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func handleBlendCommand() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: starchive blend <id1> <id2>")
		fmt.Println("Example: starchive blend OIduTH7NYA8 EbD7lfrsY2s")
		fmt.Println("Enters an interactive blend shell with real-time controls.")
		os.Exit(1)
	}

	id1 := os.Args[2]
	id2 := os.Args[3]
	
	blendShell := newBlendShell(id1, id2)
	blendShell.run()
}

func handleBlendClearCommand() {
	if len(os.Args) == 2 {
		clearAllBlendMetadata()
	} else if len(os.Args) == 4 {
		id1 := os.Args[2]
		id2 := os.Args[3]
		clearSpecificBlendMetadata(id1, id2)
	} else {
		fmt.Println("Usage: starchive blend-clear [id1 id2]")
		fmt.Println("  starchive blend-clear          Clear all blend metadata")
		fmt.Println("  starchive blend-clear id1 id2  Clear metadata for specific track pair")
		os.Exit(1)
	}
}

func clearAllBlendMetadata() {
	tmpDir := "/tmp"
	pattern := "starchive_blend_*.tmp"
	
	cmd := exec.Command("find", tmpDir, "-name", pattern, "-type", "f", "-delete")
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error clearing blend metadata: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Println("All blend metadata cleared.")
}

func clearSpecificBlendMetadata(id1, id2 string) {
	file1 := "/tmp/starchive_blend_" + id1 + "_" + id2 + ".tmp"
	file2 := "/tmp/starchive_blend_" + id2 + "_" + id1 + ".tmp"
	
	removed := false
	
	if _, err := os.Stat(file1); err == nil {
		os.Remove(file1)
		removed = true
	}
	
	if _, err := os.Stat(file2); err == nil {
		os.Remove(file2)
		removed = true
	}
	
	if removed {
		fmt.Printf("Blend metadata cleared for tracks %s and %s.\n", id1, id2)
	} else {
		fmt.Printf("No blend metadata found for tracks %s and %s.\n", id1, id2)
	}
}

// Key and BPM calculation utilities
var keyMap = map[string]int{
	"C major": 0, "G major": 7, "D major": 2, "A major": 9, "E major": 4, "B major": 11,
	"F# major": 6, "Db major": 1, "Ab major": 8, "Eb major": 3, "Bb major": 10, "F major": 5,
	"C# major": 1, "G# major": 8,
	"A minor": 9, "E minor": 4, "B minor": 11, "F# minor": 6, "C# minor": 1, "G# minor": 8,
	"Eb minor": 3, "Bb minor": 10, "F minor": 5, "C minor": 0, "G minor": 7, "D minor": 2,
}

func calculateKeyDifference(key1, key2 string) int {
	val1, exists1 := keyMap[key1]
	val2, exists2 := keyMap[key2]
	
	if !exists1 || !exists2 {
		return 0
	}
	
	diff := val2 - val1
	if diff > 6 {
		diff -= 12
	} else if diff < -6 {
		diff += 12
	}
	
	return diff
}

func calculateEffectiveBPM(originalBPM float64, tempoAdjustment float64) float64 {
	multiplier := 1.0 + (tempoAdjustment / 100.0)
	return originalBPM * multiplier
}

func calculateEffectiveKey(originalKey string, pitchAdjustment int) string {
	if pitchAdjustment == 0 {
		return originalKey
	}
	
	reverseKeyMap := make(map[int]string)
	isMinor := strings.Contains(originalKey, "minor")
	
	for key, value := range keyMap {
		if (isMinor && strings.Contains(key, "minor")) || (!isMinor && strings.Contains(key, "major")) {
			reverseKeyMap[value] = key
		}
	}
	
	originalValue, exists := keyMap[originalKey]
	if !exists {
		return originalKey
	}
	
	newValue := (originalValue + pitchAdjustment) % 12
	if newValue < 0 {
		newValue += 12
	}
	
	if newKey, exists := reverseKeyMap[newValue]; exists {
		return newKey
	}
	
	return originalKey
}

// Utility functions
func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func clampFloat(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func detectTrackTypes(id1, id2 string) (string, string) {
	id1HasVocal := hasVocalFile(id1)
	id1HasInstrumental := hasInstrumentalFile(id1)
	id2HasVocal := hasVocalFile(id2)
	id2HasInstrumental := hasInstrumentalFile(id2)

	var type1, type2 string

	// If one track only has instrumental, make the other vocal (if possible)
	if !id2HasVocal && id2HasInstrumental {
		// Track 2 is instrumental-only, prefer vocal for track 1
		if id1HasVocal {
			type1 = "V"
		} else {
			type1 = "I"
		}
		type2 = "I"
	} else if !id1HasVocal && id1HasInstrumental {
		// Track 1 is instrumental-only, prefer vocal for track 2
		type1 = "I"
		if id2HasVocal {
			type2 = "V"
		} else {
			type2 = "I"
		}
	} else {
		// Both tracks have options, choose complementary types
		if id1HasVocal && !id1HasInstrumental {
			type1 = "V"
		} else if !id1HasVocal && id1HasInstrumental {
			type1 = "I"
		} else if id1HasVocal && id1HasInstrumental {
			type1 = "V"  // Default to vocal for track 1
		} else {
			type1 = "I"  // Fallback
		}

		if id2HasVocal && !id2HasInstrumental {
			type2 = "V"
		} else if !id2HasVocal && id2HasInstrumental {
			type2 = "I"
		} else if id2HasVocal && id2HasInstrumental {
			// Choose opposite of track 1
			if type1 == "V" {
				type2 = "I"
			} else {
				type2 = "V"
			}
		} else {
			type2 = "I"  // Fallback
		}
	}

	return type1, type2
}