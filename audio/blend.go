package audio

import ()

// Key and BPM calculation utilities
var keyMap = map[string]int{
	"C major": 0, "G major": 7, "D major": 2, "A major": 9, "E major": 4, "B major": 11,
	"F# major": 6, "Db major": 1, "Ab major": 8, "Eb major": 3, "Bb major": 10, "F major": 5,
	"C# major": 1, "G# major": 8,
	"A minor": 9, "E minor": 4, "B minor": 11, "F# minor": 6, "C# minor": 1, "G# minor": 8,
	"Eb minor": 3, "Bb minor": 10, "F minor": 5, "C minor": 0, "G minor": 7, "D minor": 2,
}

// CalculateKeyDifference calculates the semitone difference between two keys
func CalculateKeyDifference(key1, key2 string) int {
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


// DetectTrackTypes determines optimal track types (vocal/instrumental) for blending
func DetectTrackTypes(id1, id2 string) (string, string) {
	id1HasVocal := HasVocalFile(id1)
	id1HasInstrumental := HasInstrumentalFile(id1)
	id2HasVocal := HasVocalFile(id2)
	id2HasInstrumental := HasInstrumentalFile(id2)

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