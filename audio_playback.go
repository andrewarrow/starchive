package main

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
		fmt.Println("Usage: starchive blend <id1> <id2> [adjustments...]")
		fmt.Println("Example: starchive blend OIduTH7NYA8 EbD7lfrsY2s")
		fmt.Println("         starchive blend OIduTH7NYA8 EbD7lfrsY2s I+tempo V-pitch")
		fmt.Println("Adjustments: I+tempo, I-tempo, V+tempo, V-tempo, I+pitch, I-pitch, V+pitch, V-pitch")
		fmt.Println("Plays two tracks simultaneously with intelligent BPM/key-based adjustments for 10 seconds")
		os.Exit(1)
	}

	id1 := os.Args[2]
	id2 := os.Args[3]
	adjustments := os.Args[4:]

	db, err := initDatabase()
	if err != nil {
		fmt.Printf("Error initializing database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	metadata1, found1 := getCachedMetadata(db, id1)
	metadata2, found2 := getCachedMetadata(db, id2)

	rand.Seed(time.Now().UnixNano())

	var pitch1, tempo1, pitch2, tempo2 int
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
		
		pitch1, tempo1, pitch2, tempo2 = applyAndSaveAdjustments(id1, id2, pitch1, tempo1, pitch2, tempo2, type1, type2, adjustments)
		
		effectiveBPM1 := calculateEffectiveBPM(bpm1, tempo1)
		effectiveBPM2 := calculateEffectiveBPM(bpm2, tempo2)
		effectiveKey1 := calculateEffectiveKey(key1, pitch1)
		effectiveKey2 := calculateEffectiveKey(key2, pitch2)
		
		fmt.Printf("Effective values:\n")
		fmt.Printf("  Track 1 (%s): %.1f BPM, %s (was %.1f BPM, %s)\n", id1, effectiveBPM1, effectiveKey1, bpm1, key1)
		fmt.Printf("  Track 2 (%s): %.1f BPM, %s (was %.1f BPM, %s)\n", id2, effectiveBPM2, effectiveKey2, bpm2, key2)
	} else {
		fmt.Printf("BPM/key data not available, using random adjustments\n")
		
		audioTypes := []string{"I", "V"}
		type1 = audioTypes[rand.Intn(2)]
		type2 = audioTypes[rand.Intn(2)]

		pitchRange := []int{-8, -6, -4, -2, 0, 2, 4, 6, 8}
		tempoRange := []int{-20, -15, -10, -5, 0, 5, 10, 15, 20}

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
	fmt.Printf("  %s (%s): pitch %+d semitones, tempo %+d%%\n", id1, trackDesc1, pitch1, tempo1)
	fmt.Printf("  %s (%s): pitch %+d semitones, tempo %+d%%\n", id2, trackDesc2, pitch2, tempo2)
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
	
	if pitch1 != 0 || tempo1 != 0 {
		var filters []string
		if pitch1 != 0 {
			pitchMultiplier := 1.0 + (float64(pitch1) / 12.0)
			filters = append(filters, fmt.Sprintf("asetrate=44100*%.6f,aresample=44100", pitchMultiplier))
		}
		if tempo1 != 0 {
			tempoMultiplier := 1.0 + (float64(tempo1) / 100.0)
			if tempoMultiplier > 0.5 && tempoMultiplier <= 2.0 {
				filters = append(filters, fmt.Sprintf("atempo=%.6f", tempoMultiplier))
			}
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
	
	if pitch2 != 0 || tempo2 != 0 {
		var filters []string
		if pitch2 != 0 {
			pitchMultiplier := 1.0 + (float64(pitch2) / 12.0)
			filters = append(filters, fmt.Sprintf("asetrate=44100*%.6f,aresample=44100", pitchMultiplier))
		}
		if tempo2 != 0 {
			tempoMultiplier := 1.0 + (float64(tempo2) / 100.0)
			if tempoMultiplier > 0.5 && tempoMultiplier <= 2.0 {
				filters = append(filters, fmt.Sprintf("atempo=%.6f", tempoMultiplier))
			}
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

func calculateIntelligentAdjustments(sourceBPM float64, sourceKey string, targetBPM float64, targetKey string) (int, int) {
	pitch := calculateKeyDifference(sourceKey, targetKey)
	
	bpmRatio := targetBPM / sourceBPM
	var tempo int
	if bpmRatio > 1.25 {
		tempo = -15
	} else if bpmRatio > 1.10 {
		tempo = -8
	} else if bpmRatio < 0.8 {
		tempo = 20
	} else if bpmRatio < 0.9 {
		tempo = 10
	} else {
		tempo = 0
	}
	
	return pitch, tempo
}

func calculateKeyDifference(key1, key2 string) int {
	keyMap := map[string]int{
		"C major": 0, "G major": 7, "D major": 2, "A major": 9, "E major": 4, "B major": 11,
		"F# major": 6, "Db major": 1, "Ab major": 8, "Eb major": 3, "Bb major": 10, "F major": 5,
		"A minor": 9, "E minor": 4, "B minor": 11, "F# minor": 6, "C# minor": 1, "G# minor": 8,
		"Eb minor": 3, "Bb minor": 10, "F minor": 5, "C minor": 0, "G minor": 7, "D minor": 2,
	}
	
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
	
	audioTypes := []string{"V", "I"}
	rand.Shuffle(len(audioTypes), func(i, j int) {
		audioTypes[i], audioTypes[j] = audioTypes[j], audioTypes[i]
	})
	type1, type2 := audioTypes[0], audioTypes[1]
	
	// Format: type1,type2,pitch1,tempo1,pitch2,tempo2
	os.WriteFile(tmpFile, []byte(fmt.Sprintf("%s,%s,0,0,0,0", type1, type2)), 0644)
	return type1, type2
}

func applyAndSaveAdjustments(id1, id2 string, basePitch1, baseTempo1, basePitch2, baseTempo2 int, type1, type2 string, adjustments []string) (int, int, int, int) {
	tmpFile := "/tmp/starchive_blend_" + id1 + "_" + id2 + ".tmp"
	
	// Load existing adjustments
	pitch1Adj, tempo1Adj, pitch2Adj, tempo2Adj := loadAccumulatedAdjustments(tmpFile)
	
	// Apply new adjustments
	for _, adj := range adjustments {
		if len(adj) < 3 {
			continue
		}
		
		trackPrefix := adj[0:1]
		operation := adj[1:2]
		param := adj[2:]
		
		delta := 0
		switch param {
		case "tempo":
			if operation == "+" {
				delta = 10
			} else if operation == "-" {
				delta = -10
			}
		case "pitch":
			if operation == "+" {
				delta = 2
			} else if operation == "-" {
				delta = -2
			}
		}
		
		if trackPrefix == type1 {
			if param == "tempo" {
				tempo1Adj += delta
			} else if param == "pitch" {
				pitch1Adj += delta
			}
		} else if trackPrefix == type2 {
			if param == "tempo" {
				tempo2Adj += delta
			} else if param == "pitch" {
				pitch2Adj += delta
			}
		}
	}
	
	// Apply limits
	tempo1Adj = clamp(tempo1Adj, -50, 50)
	tempo2Adj = clamp(tempo2Adj, -50, 50)
	pitch1Adj = clamp(pitch1Adj, -12, 12)
	pitch2Adj = clamp(pitch2Adj, -12, 12)
	
	// Save back to file
	saveAccumulatedAdjustments(tmpFile, type1, type2, pitch1Adj, tempo1Adj, pitch2Adj, tempo2Adj)
	
	return basePitch1 + pitch1Adj, baseTempo1 + tempo1Adj, basePitch2 + pitch2Adj, baseTempo2 + tempo2Adj
}

func loadAccumulatedAdjustments(tmpFile string) (int, int, int, int) {
	if data, err := os.ReadFile(tmpFile); err == nil {
		parts := strings.Split(strings.TrimSpace(string(data)), ",")
		if len(parts) == 6 {
			pitch1, _ := strconv.Atoi(parts[2])
			tempo1, _ := strconv.Atoi(parts[3])
			pitch2, _ := strconv.Atoi(parts[4])
			tempo2, _ := strconv.Atoi(parts[5])
			return pitch1, tempo1, pitch2, tempo2
		}
	}
	return 0, 0, 0, 0
}

func saveAccumulatedAdjustments(tmpFile, type1, type2 string, pitch1, tempo1, pitch2, tempo2 int) {
	data := fmt.Sprintf("%s,%s,%d,%d,%d,%d", type1, type2, pitch1, tempo1, pitch2, tempo2)
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

func calculateEffectiveBPM(originalBPM float64, tempoAdjustment int) float64 {
	multiplier := 1.0 + (float64(tempoAdjustment) / 100.0)
	return originalBPM * multiplier
}

func calculateEffectiveKey(originalKey string, pitchAdjustment int) string {
	if pitchAdjustment == 0 {
		return originalKey
	}
	
	keyMap := map[string]int{
		"C major": 0, "G major": 7, "D major": 2, "A major": 9, "E major": 4, "B major": 11,
		"F# major": 6, "Db major": 1, "Ab major": 8, "Eb major": 3, "Bb major": 10, "F major": 5,
		"A minor": 9, "E minor": 4, "B minor": 11, "F# minor": 6, "C# minor": 1, "G# minor": 8,
		"Eb minor": 3, "Bb minor": 10, "F minor": 5, "C minor": 0, "G minor": 7, "D minor": 2,
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