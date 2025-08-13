package main

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func handlePlayCommand() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: starchive play <id> [I|V]")
		fmt.Println("Example: starchive play NdYWuo9OFAw")
		fmt.Println("         starchive play NdYWuo9OFAw I  (instrumental)")
		fmt.Println("         starchive play NdYWuo9OFAw V  (vocals)")
		fmt.Println("Plays the wav file starting from the middle. Press any key to stop.")
		os.Exit(1)
	}

	id := os.Args[2]
	
	audioType := ""
	if len(os.Args) > 3 {
		audioType = os.Args[3]
	}
	
	inputPath := getAudioFilename(id, audioType)

	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		fmt.Printf("Error: Input file %s does not exist\n", inputPath)
		os.Exit(1)
	}

	duration, err := getAudioDuration(inputPath)
	if err != nil {
		fmt.Printf("Error getting audio duration: %v\n", err)
		os.Exit(1)
	}

	startPosition := duration / 2
	
	trackDesc := "main track"
	switch audioType {
	case "I", "instrumental", "instrumentals":
		trackDesc = "instrumental track"
	case "V", "vocal", "vocals":
		trackDesc = "vocal track"
	}
	
	fmt.Printf("Playing %s (%s) from position %.1fs (middle of %.1fs total)\n", id, trackDesc, startPosition, duration)
	fmt.Println("Press any key to stop playback...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		cmd := exec.CommandContext(ctx, "ffplay", 
			"-ss", fmt.Sprintf("%.1f", startPosition),
			"-autoexit",
			"-nodisp",
			"-loglevel", "quiet",
			inputPath)
		
		cmd.Run()
	}()

	waitForKeyPress()
	cancel()
	fmt.Println("\nPlayback stopped.")
}

func handleBlendCommand() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: starchive blend <id1> <id2> [volume1] [volume2] [adjustments...]")
		fmt.Println("Example: starchive blend OIduTH7NYA8 EbD7lfrsY2s")
		fmt.Println("         starchive blend OIduTH7NYA8 EbD7lfrsY2s 100 50")
		fmt.Println("         starchive blend OIduTH7NYA8 EbD7lfrsY2s 100 50 I+tempo V-pitch")
		fmt.Println("         starchive blend OIduTH7NYA8 EbD7lfrsY2s 80 120 bpm2to1 key1to2")
		fmt.Println("Volume: 0-200 (default: 100 for both tracks)")
		fmt.Println("Adjustments: I+tempo, I-tempo, V+tempo, V-tempo, I+pitch, I-pitch, V+pitch, V-pitch")
		fmt.Println("Matching:")
		fmt.Println("  bpm1to2, bpm2to1 (sync BPM between tracks)")
		fmt.Println("  key1to2, key2to1 (sync key between tracks)")
		fmt.Println("Plays two tracks simultaneously with intelligent BPM/key-based adjustments for 10 seconds")
		os.Exit(1)
	}

	id1 := os.Args[2]
	id2 := os.Args[3]
	
	// Parse volume parameters if provided
	volume1 := 100.0
	volume2 := 100.0
	adjustments := os.Args[4:]
	
	// Check if first two arguments after the IDs are volume levels
	if len(os.Args) >= 6 {
		if vol1, err := strconv.Atoi(os.Args[4]); err == nil && vol1 >= 0 && vol1 <= 200 {
			if vol2, err := strconv.Atoi(os.Args[5]); err == nil && vol2 >= 0 && vol2 <= 200 {
				volume1 = float64(vol1)
				volume2 = float64(vol2)
				adjustments = os.Args[6:]
			}
		}
	}

	db, err := initDatabase()
	if err != nil {
		fmt.Printf("Error initializing database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	metadata1, found1 := getCachedMetadata(db, id1)
	metadata2, found2 := getCachedMetadata(db, id2)

	rand.Seed(time.Now().UnixNano())

	var pitch1, pitch2 int
	var tempo1, tempo2 float64
	var type1, type2 string

	if found1 && found2 && metadata1.BPM != nil && metadata2.BPM != nil && metadata1.Key != nil && metadata2.Key != nil {
		bpm1 := *metadata1.BPM
		bpm2 := *metadata2.BPM
		key1 := *metadata1.Key
		key2 := *metadata2.Key
		
		fmt.Printf("Using intelligent blending:\n")
		fmt.Printf("  Track 1 (%s): %.1f BPM, %s\n", id1, bpm1, key1)
		fmt.Printf("  Track 2 (%s): %.1f BPM, %s\n", id2, bpm2, key2)

		type1, type2 = getOrAssignTrackTypes(id1, id2)
		
		pitch1, tempo1 = calculateIntelligentAdjustments(bpm1, key1, bpm2, key2)
		pitch2, tempo2 = calculateIntelligentAdjustments(bpm2, key2, bpm1, key1)
		
		pitch1, tempo1, pitch2, tempo2 = applyAndSaveAdjustments(id1, id2, pitch1, tempo1, pitch2, tempo2, type1, type2, adjustments, bpm1, bpm2, metadata1, metadata2)
		
		effectiveBPM1 := calculateEffectiveBPM(bpm1, tempo1)
		effectiveBPM2 := calculateEffectiveBPM(bpm2, tempo2)
		effectiveKey1 := calculateEffectiveKey(key1, pitch1)
		effectiveKey2 := calculateEffectiveKey(key2, pitch2)
		
		fmt.Printf("Effective values:\n")
		fmt.Printf("  Track 1 (%s): %.1f BPM, %s (was %.1f BPM, %s)\n", id1, effectiveBPM1, effectiveKey1, bpm1, key1)
		fmt.Printf("  Track 2 (%s): %.1f BPM, %s (was %.1f BPM, %s)\n", id2, effectiveBPM2, effectiveKey2, bpm2, key2)
	} else {
		fmt.Printf("BPM/key data not available, using random adjustments\n")
		
		// Use smart file detection instead of pure random
		type1, type2 = getOrAssignTrackTypes(id1, id2)

		pitchRange := []int{-8, -6, -4, -2, 0, 2, 4, 6, 8}
		tempoRange := []float64{-20.0, -15.0, -10.0, -5.0, 0.0, 5.0, 10.0, 15.0, 20.0}

		pitch1 = pitchRange[rand.Intn(len(pitchRange))]
		tempo1 = tempoRange[rand.Intn(len(tempoRange))]
		pitch2 = pitchRange[rand.Intn(len(pitchRange))]
		tempo2 = tempoRange[rand.Intn(len(tempoRange))]
	}

	inputPath1 := getAudioFilename(id1, type1)
	inputPath2 := getAudioFilename(id2, type2)

	if _, err := os.Stat(inputPath1); os.IsNotExist(err) {
		fmt.Printf("Error: Input file %s does not exist\n", inputPath1)
		os.Exit(1)
	}
	if _, err := os.Stat(inputPath2); os.IsNotExist(err) {
		fmt.Printf("Error: Input file %s does not exist\n", inputPath2)
		os.Exit(1)
	}

	duration1, err := getAudioDuration(inputPath1)
	if err != nil {
		fmt.Printf("Error getting audio duration for %s: %v\n", inputPath1, err)
		os.Exit(1)
	}
	duration2, err := getAudioDuration(inputPath2)
	if err != nil {
		fmt.Printf("Error getting audio duration for %s: %v\n", inputPath2, err)
		os.Exit(1)
	}

	startPosition1 := duration1 / 2
	startPosition2 := duration2 / 2

	trackDesc1 := "instrumental"
	if type1 == "V" {
		trackDesc1 = "vocal"
	}
	trackDesc2 := "instrumental"
	if type2 == "V" {
		trackDesc2 = "vocal"
	}

	fmt.Printf("Blending tracks:\n")
	fmt.Printf("  %s (%s): pitch %+d semitones, tempo %+.1f%%, volume %.0f%%\n", id1, trackDesc1, pitch1, tempo1, volume1)
	fmt.Printf("  %s (%s): pitch %+d semitones, tempo %+.1f%%, volume %.0f%%\n", id2, trackDesc2, pitch2, tempo2, volume2)
	fmt.Printf("Playing for 10 seconds...\n")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ffplayArgs1 := []string{
		"-ss", fmt.Sprintf("%.1f", startPosition1),
		"-t", "10",
		"-autoexit",
		"-nodisp",
		"-loglevel", "quiet",
	}
	
	if pitch1 != 0 || tempo1 != 0 || volume1 != 100 {
		var filters []string
		if tempo1 != 0 {
			tempoMultiplier := 1.0 + (tempo1 / 100.0)
			if tempoMultiplier > 0.5 && tempoMultiplier <= 2.0 {
				filters = append(filters, fmt.Sprintf("atempo=%.6f", tempoMultiplier))
			}
		}
		if pitch1 != 0 {
			// Use rubberband-style pitch shifting that preserves tempo
			pitchSemitones := float64(pitch1)
			filters = append(filters, fmt.Sprintf("asetrate=44100*%.6f,aresample=44100,atempo=%.6f", 
				math.Pow(2, pitchSemitones/12.0), 1.0/math.Pow(2, pitchSemitones/12.0)))
		}
		if volume1 != 100 {
			volumeMultiplier := volume1 / 100.0
			filters = append(filters, fmt.Sprintf("volume=%.6f", volumeMultiplier))
		}
		if len(filters) > 0 {
			filter := strings.Join(filters, ",")
			ffplayArgs1 = append(ffplayArgs1, "-af", filter)
		}
	}
	
	ffplayArgs1 = append(ffplayArgs1, inputPath1)

	ffplayArgs2 := []string{
		"-ss", fmt.Sprintf("%.1f", startPosition2),
		"-t", "10",
		"-autoexit",
		"-nodisp",
		"-loglevel", "quiet",
	}
	
	if pitch2 != 0 || tempo2 != 0 || volume2 != 100 {
		var filters []string
		if tempo2 != 0 {
			tempoMultiplier := 1.0 + (tempo2 / 100.0)
			if tempoMultiplier > 0.5 && tempoMultiplier <= 2.0 {
				filters = append(filters, fmt.Sprintf("atempo=%.6f", tempoMultiplier))
			}
		}
		if pitch2 != 0 {
			// Use rubberband-style pitch shifting that preserves tempo
			pitchSemitones := float64(pitch2)
			filters = append(filters, fmt.Sprintf("asetrate=44100*%.6f,aresample=44100,atempo=%.6f", 
				math.Pow(2, pitchSemitones/12.0), 1.0/math.Pow(2, pitchSemitones/12.0)))
		}
		if volume2 != 100 {
			volumeMultiplier := volume2 / 100.0
			filters = append(filters, fmt.Sprintf("volume=%.6f", volumeMultiplier))
		}
		if len(filters) > 0 {
			filter := strings.Join(filters, ",")
			ffplayArgs2 = append(ffplayArgs2, "-af", filter)
		}
	}
	
	ffplayArgs2 = append(ffplayArgs2, inputPath2)

	go func() {
		cmd1 := exec.CommandContext(ctx, "ffplay", ffplayArgs1...)
		cmd1.Run()
	}()

	go func() {
		cmd2 := exec.CommandContext(ctx, "ffplay", ffplayArgs2...)
		cmd2.Run()
	}()

	<-ctx.Done()
	fmt.Println("Blend playback completed.")
}

func handleBlendClearCommand() {
	if len(os.Args) == 2 {
		// Clear all blend metadata
		clearAllBlendMetadata()
	} else if len(os.Args) == 4 {
		// Clear specific blend metadata for two IDs
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
	// Try both possible filename combinations
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

func playTempFile(filePath string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		cmd := exec.CommandContext(ctx, "ffplay", 
			"-autoexit",
			"-nodisp",
			"-loglevel", "quiet",
			filePath)
		
		cmd.Run()
	}()

	waitForKeyPress()
	cancel()
	fmt.Println("Preview stopped.")
}

func calculateIntelligentAdjustments(sourceBPM float64, sourceKey string, targetBPM float64, targetKey string) (int, float64) {
	pitch := calculateKeyDifference(sourceKey, targetKey)
	tempo := 0.0
	
	return pitch, tempo
}

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

func getOrAssignTrackTypes(id1, id2 string) (string, string) {
	tmpFile := "/tmp/starchive_blend_" + id1 + "_" + id2 + ".tmp"
	
	if data, err := os.ReadFile(tmpFile); err == nil {
		parts := strings.Split(strings.TrimSpace(string(data)), ",")
		if len(parts) >= 2 {
			return parts[0], parts[1]
		}
	}
	
	// Check which files exist for each ID
	id1HasVocal := hasVocalFile(id1)
	id1HasInstrumental := hasInstrumentalFile(id1)
	id2HasVocal := hasVocalFile(id2)
	id2HasInstrumental := hasInstrumentalFile(id2)
	
	var type1, type2 string
	
	// Determine type1 (for id1)
	if !id1HasVocal && id1HasInstrumental {
		type1 = "I" // Only instrumental available
	} else if id1HasVocal && !id1HasInstrumental {
		type1 = "V" // Only vocal available
	} else if id1HasVocal && id1HasInstrumental {
		// Both available, choose randomly
		if rand.Intn(2) == 0 {
			type1 = "V"
		} else {
			type1 = "I"
		}
	} else {
		// Neither available - this shouldn't happen, but default to "I"
		type1 = "I"
	}
	
	// Determine type2 (for id2)
	if !id2HasVocal && id2HasInstrumental {
		type2 = "I" // Only instrumental available
	} else if id2HasVocal && !id2HasInstrumental {
		type2 = "V" // Only vocal available
	} else if id2HasVocal && id2HasInstrumental {
		// Both available, choose randomly
		if rand.Intn(2) == 0 {
			type2 = "V"
		} else {
			type2 = "I"
		}
	} else {
		// Neither available - this shouldn't happen, but default to "I"
		type2 = "I"
	}
	
	// Format: type1,type2,pitch1,tempo1,pitch2,tempo2
	os.WriteFile(tmpFile, []byte(fmt.Sprintf("%s,%s,0,0,0,0", type1, type2)), 0644)
	return type1, type2
}

func applyAndSaveAdjustments(id1, id2 string, basePitch1 int, baseTempo1 float64, basePitch2 int, baseTempo2 float64, type1, type2 string, adjustments []string, originalBPM1, originalBPM2 float64, metadata1, metadata2 *VideoMetadata) (int, float64, int, float64) {
	tmpFile := "/tmp/starchive_blend_" + id1 + "_" + id2 + ".tmp"
	
	// Load existing adjustments
	pitch1Adj, tempo1Adj, pitch2Adj, tempo2Adj := loadAccumulatedAdjustments(tmpFile)
	
	// Check for BPM and key matching first
	for _, adj := range adjustments {
		if adj == "bpm1to2" || adj == "match1to2" {
			// Make track 1 BPM match track 2's original BPM
			targetBPM := originalBPM2
			requiredRatio := targetBPM / originalBPM1
			tempo1Adj = math.Round((requiredRatio - 1.0) * 100.0) - baseTempo1
			fmt.Printf("BPM Match: Setting track 1 to %.1f BPM to match track 2\n", targetBPM)
		} else if adj == "bpm2to1" || adj == "match2to1" {
			// Make track 2 BPM match track 1's original BPM
			targetBPM := originalBPM1
			requiredRatio := targetBPM / originalBPM2
			requiredTotalTempo := (requiredRatio - 1.0) * 100.0
			tempo2Adj = requiredTotalTempo - baseTempo2
			
			fmt.Printf("BPM Match: Setting track 2 to %.1f BPM to match track 1\n", targetBPM)
		} else if adj == "key1to2" {
			// Make both tracks match track 2's key - override base pitch for both
			keyDiff1 := calculateKeyDifference(*metadata1.Key, *metadata2.Key)
			pitch1Adj = keyDiff1 - basePitch1  // Track 1 to target key
			pitch2Adj = 0 - basePitch2         // Track 2 stays in original key (cancel base adjustment)
			fmt.Printf("Key Match: Setting both tracks to %s\n", *metadata2.Key)
		} else if adj == "key2to1" {
			// Make both tracks match track 1's key - override base pitch for both
			keyDiff2 := calculateKeyDifference(*metadata2.Key, *metadata1.Key)
			pitch1Adj = 0 - basePitch1         // Track 1 stays in original key (cancel base adjustment)
			pitch2Adj = keyDiff2 - basePitch2  // Track 2 to target key
			fmt.Printf("Key Match: Setting both tracks to %s\n", *metadata1.Key)
		}
	}
	
	// Apply other adjustments
	for _, adj := range adjustments {
		if adj == "bpm1to2" || adj == "bpm2to1" || adj == "key1to2" || adj == "key2to1" || 
		   adj == "match1to2" || adj == "match2to1" {
			continue
		}
		if len(adj) < 3 {
			continue
		}
		
		trackPrefix := adj[0:1]
		operation := adj[1:2]
		param := adj[2:]
		
		if trackPrefix == type1 {
			if param == "tempo" {
				if operation == "+" {
					tempo1Adj += 10.0
				} else if operation == "-" {
					tempo1Adj -= 10.0
				}
			} else if param == "pitch" {
				if operation == "+" {
					pitch1Adj += 2
				} else if operation == "-" {
					pitch1Adj -= 2
				}
			}
		} else if trackPrefix == type2 {
			if param == "tempo" {
				if operation == "+" {
					tempo2Adj += 10.0
				} else if operation == "-" {
					tempo2Adj -= 10.0
				}
			} else if param == "pitch" {
				if operation == "+" {
					pitch2Adj += 2
				} else if operation == "-" {
					pitch2Adj -= 2
				}
			}
		}
	}
	
	// Apply limits
	tempo1Adj = clampFloat(tempo1Adj, -50.0, 50.0)
	tempo2Adj = clampFloat(tempo2Adj, -50.0, 50.0)
	pitch1Adj = clamp(pitch1Adj, -12, 12)
	pitch2Adj = clamp(pitch2Adj, -12, 12)
	
	// Save back to file
	saveAccumulatedAdjustments(tmpFile, type1, type2, pitch1Adj, tempo1Adj, pitch2Adj, tempo2Adj)
	
	return basePitch1 + pitch1Adj, baseTempo1 + tempo1Adj, basePitch2 + pitch2Adj, baseTempo2 + tempo2Adj
}

func loadAccumulatedAdjustments(tmpFile string) (int, float64, int, float64) {
	if data, err := os.ReadFile(tmpFile); err == nil {
		parts := strings.Split(strings.TrimSpace(string(data)), ",")
		if len(parts) == 6 {
			pitch1, _ := strconv.Atoi(parts[2])
			tempo1, _ := strconv.ParseFloat(parts[3], 64)
			pitch2, _ := strconv.Atoi(parts[4])
			tempo2, _ := strconv.ParseFloat(parts[5], 64)
			return pitch1, tempo1, pitch2, tempo2
		}
	}
	return 0, 0, 0, 0
}

func saveAccumulatedAdjustments(tmpFile, type1, type2 string, pitch1 int, tempo1 float64, pitch2 int, tempo2 float64) {
	data := fmt.Sprintf("%s,%s,%d,%.3f,%d,%.3f", type1, type2, pitch1, tempo1, pitch2, tempo2)
	os.WriteFile(tmpFile, []byte(data), 0644)
}

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