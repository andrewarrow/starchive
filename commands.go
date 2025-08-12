package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

func handleLsCommand() {
	// Initialize database
	db, err := initDatabase()
	if err != nil {
		fmt.Printf("Error initializing database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	
	// Print header
	fmt.Printf("%-15s %-60s %-1s %-10s\n", "ID", "Title", "V", "BPM/Key")
	fmt.Printf("%-15s %-60s %-1s %-10s\n", "---", "-----", "-", "-------")
	
	entries, err := os.ReadDir("./data")
	if err != nil {
		fmt.Println("Error reading ./data:", err)
		os.Exit(1)
	}
	
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			id := strings.TrimSuffix(e.Name(), ".json")
			filePath := "./data/" + e.Name()
			
			// Try to get from cache first
			if cachedMetadata, found := getCachedMetadata(db, id); found {
				vocalStatus := "N"
				if cachedMetadata.VocalDone {
					vocalStatus = "Y"
				}
				bmpInfo := ""
				if cachedMetadata.BPM != nil && cachedMetadata.Key != nil {
					bmpInfo = fmt.Sprintf("%.1f/%s", 
						*cachedMetadata.BPM, *cachedMetadata.Key)
				}
				fmt.Printf("%-15s %-60s %-1s %-10s\n", cachedMetadata.ID, truncateString(cachedMetadata.Title, 60), vocalStatus, bmpInfo)
				continue
			}
			
			// Parse JSON file if not in cache or cache is stale
			metadata, err := parseJSONMetadata(filePath)
			if err != nil {
				fmt.Printf("%s\t<error parsing file: %v>\n", id, err)
				continue
			}
			
			// Cache the metadata
			if err := cacheMetadata(db, *metadata); err != nil {
				fmt.Printf("Warning: failed to cache metadata for %s: %v\n", id, err)
			}
			
			// Get the updated metadata with correct vocal_done value
			if updatedMetadata, found := getCachedMetadata(db, id); found {
				metadata = updatedMetadata
			}
			
			vocalStatus := "N"
			if metadata.VocalDone {
				vocalStatus = "Y"
			}
			bmpInfo := ""
			if metadata.BPM != nil && metadata.Key != nil {
				bmpInfo = fmt.Sprintf("%.1f/%s", 
					*metadata.BPM, *metadata.Key)
			}
			fmt.Printf("%-15s %-60s %-1s %-10s\n", metadata.ID, truncateString(metadata.Title, 60), vocalStatus, bmpInfo)
		}
	}
}

func handleVocalCommand() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: starchive vocal <id>")
		fmt.Println("Example: starchive vocal abc123")
		os.Exit(1)
	}

	id := os.Args[2]
	inputPath := fmt.Sprintf("./data/%s.wav", id)

	// Check if input file exists
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		fmt.Printf("Error: Input file %s does not exist\n", inputPath)
		os.Exit(1)
	}

	// Run audio-separator command
	cmd := exec.Command("audio-separator", inputPath,
		"--output_dir", "./data/",
		"--model_filename", "UVR_MDXNET_Main.onnx",
		"--output_format", "wav")

	fmt.Printf("Running: %s\n", cmd.String())
	
	// Set command to output directly to stdout/stderr for verbose output
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error running audio-separator: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully separated vocals for %s\n", id)

	// Mark as vocal done in database
	db, err := initDatabase()
	if err != nil {
		fmt.Printf("Warning: Could not initialize database to mark vocal as done: %v\n", err)
		return
	}
	defer db.Close()

	if err := markVocalDone(db, id); err != nil {
		fmt.Printf("Warning: Could not mark vocal as done in database: %v\n", err)
	} else {
		fmt.Printf("Marked %s as vocal done in database\n", id)
	}
}

func handleBpmCommand() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: starchive bpm <id>")
		fmt.Println("Example: starchive bpm Oa_RSwwpPaA")
		os.Exit(1)
	}

	id := os.Args[2]
	inputPath := fmt.Sprintf("./data/%s.wav", id)

	// Check if input file exists
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		fmt.Printf("Error: Input file %s does not exist\n", inputPath)
		os.Exit(1)
	}

	fmt.Printf("Analyzing BPM for %s...\n", id)
	
	// Run BPM analysis on the main audio file
	bpmCmd := exec.Command("python3", "bpm/beats_per_min.py", inputPath)
	output, err := bpmCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error analyzing BPM: %v\n", err)
		fmt.Printf("Output: %s\n", string(output))
		os.Exit(1)
	}
	
	fmt.Printf("%s\n", string(output))
	
	// Parse JSON output
	var bpmData map[string]interface{}
	if err := json.Unmarshal(output, &bpmData); err != nil {
		fmt.Printf("Error parsing BPM JSON: %v\n", err)
		os.Exit(1)
	}

	// Store results in database
	db, err := initDatabase()
	if err != nil {
		fmt.Printf("Warning: Could not initialize database to store BPM data: %v\n", err)
		return
	}
	defer db.Close()

	bpm := bpmData["bpm"].(float64)
	key := bpmData["key"].(string)

	if err := storeBPMData(db, id, bpm, key); err != nil {
		fmt.Printf("Warning: Could not store BPM data in database: %v\n", err)
	} else {
		fmt.Printf("\nBPM data stored in database for %s\n", id)
	}
}

func handleSyncCommand() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: starchive sync <id1> <id2>")
		fmt.Println("Example: starchive sync NdYWuo9OFAw nlcIKh6sBtc")
		fmt.Println("This will create synchronized versions of both tracks for mashups")
		os.Exit(1)
	}

	id1 := os.Args[2]
	id2 := os.Args[3]

	// Initialize database to get BPM data
	db, err := initDatabase()
	if err != nil {
		fmt.Printf("Error initializing database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Get BPM data for both tracks
	metadata1, found1 := getCachedMetadata(db, id1)
	metadata2, found2 := getCachedMetadata(db, id2)

	if !found1 || metadata1.BPM == nil {
		fmt.Printf("Error: BPM data not found for %s. Run 'starchive bpm %s' first\n", id1, id1)
		os.Exit(1)
	}
	if !found2 || metadata2.BPM == nil {
		fmt.Printf("Error: BPM data not found for %s. Run 'starchive bpm %s' first\n", id2, id2)
		os.Exit(1)
	}

	bpm1 := *metadata1.BPM
	bpm2 := *metadata2.BPM

	fmt.Printf("Synchronizing tracks:\n")
	fmt.Printf("  %s: %.1f BPM\n", id1, bpm1)
	fmt.Printf("  %s: %.1f BPM\n", id2, bpm2)

	// Calculate tempo ratios
	ratio1to2 := bpm2 / bpm1  // Speed up track 1 to match track 2
	ratio2to1 := bpm1 / bpm2  // Speed up track 2 to match track 1

	fmt.Printf("Tempo ratios:\n")
	fmt.Printf("  %s->%s: %.3f (%.1f%%)\n", id1, id2, ratio1to2, (ratio1to2-1)*100)
	fmt.Printf("  %s->%s: %.3f (%.1f%%)\n", id2, id1, ratio2to1, (ratio2to1-1)*100)

	// Check if vocal separation has been done
	if !metadata1.VocalDone {
		fmt.Printf("Error: Vocal separation not done for %s. Run 'starchive vocal %s' first\n", id1, id1)
		os.Exit(1)
	}
	if !metadata2.VocalDone {
		fmt.Printf("Error: Vocal separation not done for %s. Run 'starchive vocal %s' first\n", id2, id2)
		os.Exit(1)
	}

	// Calculate inverse ratios for additional variations
	invRatio1to2 := 1.0 / ratio1to2  // Inverse of slowing down track1 to match track2
	invRatio2to1 := 1.0 / ratio2to1  // Inverse of speeding up track2 to match track1

	fmt.Printf("Inverse ratios:\n")
	fmt.Printf("  %s inverse: %.3f (%.1f%%)\n", id1, invRatio1to2, (invRatio1to2-1)*100)
	fmt.Printf("  %s inverse: %.3f (%.1f%%)\n", id2, invRatio2to1, (invRatio2to1-1)*100)

	// Process all files: vocals and instrumentals for both directions + inverses (8 files total)
	files := []string{
		fmt.Sprintf("%s_(Vocals)_UVR_MDXNET_Main.wav", id1),
		fmt.Sprintf("%s_(Instrumental)_UVR_MDXNET_Main.wav", id1),
		fmt.Sprintf("%s_(Vocals)_UVR_MDXNET_Main.wav", id2),
		fmt.Sprintf("%s_(Instrumental)_UVR_MDXNET_Main.wav", id2),
		fmt.Sprintf("%s_(Vocals)_UVR_MDXNET_Main.wav", id1),
		fmt.Sprintf("%s_(Instrumental)_UVR_MDXNET_Main.wav", id1),
		fmt.Sprintf("%s_(Vocals)_UVR_MDXNET_Main.wav", id2),
		fmt.Sprintf("%s_(Instrumental)_UVR_MDXNET_Main.wav", id2),
	}

	ratios := []float64{ratio1to2, ratio1to2, ratio2to1, ratio2to1, invRatio1to2, invRatio1to2, invRatio2to1, invRatio2to1}
	suffixes := []string{"sync_to_" + id2, "sync_to_" + id2, "sync_to_" + id1, "sync_to_" + id1, 
		"inv_sync_to_" + id2, "inv_sync_to_" + id2, "inv_sync_to_" + id1, "inv_sync_to_" + id1}

	for i, file := range files {
		inputPath := fmt.Sprintf("./data/%s", file)
		
		// Check if input file exists
		if _, err := os.Stat(inputPath); os.IsNotExist(err) {
			fmt.Printf("Error: Input file %s does not exist\n", inputPath)
			os.Exit(1)
		}

		// Create output filename
		baseName := strings.TrimSuffix(file, ".wav")
		outputPath := fmt.Sprintf("./data/%s_%s.wav", baseName, suffixes[i])

		fmt.Printf("\nProcessing: %s -> %s (ratio: %.3f)\n", file, filepath.Base(outputPath), ratios[i])

		// Run rubberband
		cmd := exec.Command("rubberband", 
			"--time", fmt.Sprintf("%.6f", ratios[i]),
			"--fine",  // Use R3 engine for better quality
			"--formant", // Preserve formants for vocals
			inputPath,
			outputPath)

		fmt.Printf("Running: %s\n", cmd.String())
		
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		err := cmd.Run()
		if err != nil {
			fmt.Printf("Error running rubberband: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Successfully created: %s\n", outputPath)
	}

	fmt.Printf("\nSync complete! Created 8 synchronized files with normal and inverse ratios.\n")
	fmt.Printf("Files created:\n")
	for i, file := range files {
		baseName := strings.TrimSuffix(file, ".wav")
		outputName := fmt.Sprintf("%s_%s.wav", baseName, suffixes[i])
		fmt.Printf("  %s\n", outputName)
	}
}

func handleSplitCommand() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: starchive split <filename>")
		fmt.Println("Example: starchive split beINamVRGy4_(Vocals)_UVR_MDXNET_Main_sync_to_qgaRVvAKoqQ.wav")
		os.Exit(1)
	}

	filename := os.Args[2]
	inputPath := fmt.Sprintf("./data/%s", filename)

	// Check if input file exists
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		fmt.Printf("Error: Input file %s does not exist\n", inputPath)
		os.Exit(1)
	}

	// Extract ID from filename (part before first underscore)
	id := strings.Split(filename, "_")[0]
	if id == "" {
		fmt.Printf("Error: Could not extract ID from filename %s\n", filename)
		os.Exit(1)
	}

	// Create output directory
	outputDir := fmt.Sprintf("./data/%s", id)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("Error creating directory %s: %v\n", outputDir, err)
		os.Exit(1)
	}

	fmt.Printf("Extracting ID: %s\n", id)
	fmt.Printf("Created directory: %s\n", outputDir)
	fmt.Printf("Splitting %s by silence detection...\n", filename)

	// First command: detect silence timestamps
	silenceCmd := exec.Command("ffmpeg", "-hide_banner", "-i", inputPath,
		"-af", "silencedetect=noise=-35dB:d=0.5", "-f", "null", "-")

	silenceOutput, err := silenceCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error detecting silence: %v\n", err)
		fmt.Printf("Output: %s\n", string(silenceOutput))
		os.Exit(1)
	}

	// Extract silence end times using sed equivalent
	sedCmd := exec.Command("sed", "-n", "s/.*silence_end: \\([0-9.]*\\).*/\\1/p")
	sedCmd.Stdin = strings.NewReader(string(silenceOutput))
	sedOutput, err := sedCmd.Output()
	if err != nil {
		fmt.Printf("Error extracting timestamps: %v\n", err)
		os.Exit(1)
	}

	// Convert to comma-separated list
	timestamps := strings.TrimSpace(string(sedOutput))
	timestamps = strings.ReplaceAll(timestamps, "\n", ",")

	if timestamps == "" {
		fmt.Printf("Warning: No silence detected. Creating single output file.\n")
		// Copy the original file to the output directory
		outputPath := fmt.Sprintf("%s/part_001.wav", outputDir)
		copyCmd := exec.Command("cp", inputPath, outputPath)
		if err := copyCmd.Run(); err != nil {
			fmt.Printf("Error copying file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Created: %s\n", outputPath)
		return
	}

	fmt.Printf("Detected silence timestamps: %s\n", timestamps)

	// Second command: split using detected timestamps
	outputPattern := fmt.Sprintf("%s/part_%%03d.wav", outputDir)
	splitCmd := exec.Command("ffmpeg", "-hide_banner", "-i", inputPath,
		"-c", "copy", "-f", "segment", "-segment_times", timestamps, outputPattern)

	fmt.Printf("Running: %s\n", splitCmd.String())

	splitCmd.Stdout = os.Stdout
	splitCmd.Stderr = os.Stderr

	err = splitCmd.Run()
	if err != nil {
		fmt.Printf("Error splitting file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully split %s into parts in directory %s\n", filename, outputDir)
}

func handleRmCommand() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: starchive rm <id>")
		fmt.Println("Example: starchive rm abc123")
		fmt.Println("This will remove all files matching data/<id>*")
		os.Exit(1)
	}

	id := os.Args[2]
	dataDir := "./data"
	
	// Use filepath.Glob to find all files matching the pattern
	pattern := fmt.Sprintf("%s/%s*", dataDir, id)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		fmt.Printf("Error finding files: %v\n", err)
		os.Exit(1)
	}
	
	if len(matches) == 0 {
		fmt.Printf("No files found matching pattern: %s\n", pattern)
		os.Exit(1)
	}
	
	fmt.Printf("Found %d file(s) matching pattern %s:\n", len(matches), pattern)
	for _, match := range matches {
		fmt.Printf("  %s\n", match)
	}
	
	// Remove each file
	for _, match := range matches {
		fmt.Printf("Removing: %s\n", match)
		err := os.Remove(match)
		if err != nil {
			fmt.Printf("Error removing %s: %v\n", match, err)
			os.Exit(1)
		}
	}
	
	fmt.Printf("Successfully removed %d file(s) with id: %s\n", len(matches), id)
}

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
	
	// Check for optional audio type parameter
	audioType := ""
	if len(os.Args) > 3 {
		audioType = os.Args[3]
	}
	
	inputPath := getAudioFilename(id, audioType)

	// Check if input file exists
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		fmt.Printf("Error: Input file %s does not exist\n", inputPath)
		os.Exit(1)
	}

	// Get duration of the audio file using ffprobe
	duration, err := getAudioDuration(inputPath)
	if err != nil {
		fmt.Printf("Error getting audio duration: %v\n", err)
		os.Exit(1)
	}

	// Calculate middle position (start from halfway through)
	startPosition := duration / 2
	
	// Determine track description for display
	trackDesc := "main track"
	switch audioType {
	case "I", "instrumental", "instrumentals":
		trackDesc = "instrumental track"
	case "V", "vocal", "vocals":
		trackDesc = "vocal track"
	}
	
	fmt.Printf("Playing %s (%s) from position %.1fs (middle of %.1fs total)\n", id, trackDesc, startPosition, duration)
	fmt.Println("Press any key to stop playback...")

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start playback in background
	go func() {
		// Use ffplay to play from the middle position
		cmd := exec.CommandContext(ctx, "ffplay", 
			"-ss", fmt.Sprintf("%.1f", startPosition), // Start position
			"-autoexit", // Exit when playback ends
			"-nodisp", // No video display (audio only)
			"-loglevel", "quiet", // Suppress output
			inputPath)
		
		cmd.Run() // This will block until playback ends or context is canceled
	}()

	// Wait for any key press
	waitForKeyPress()
	cancel() // Stop playback
	fmt.Println("\nPlayback stopped.")
}

func getAudioDuration(filePath string) (float64, error) {
	// Use ffprobe to get duration
	cmd := exec.Command("ffprobe", 
		"-v", "quiet",
		"-show_entries", "format=duration",
		"-of", "csv=p=0",
		filePath)
	
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	
	durationStr := strings.TrimSpace(string(output))
	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, err
	}
	
	return duration, nil
}

func waitForKeyPress() {
	// Set up signal handler for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Create a channel for key press detection
	keyChan := make(chan bool, 1)

	// Start goroutine to detect key press
	go func() {
		// Read from stdin
		reader := bufio.NewReader(os.Stdin)
		reader.ReadByte() // Wait for any byte input
		keyChan <- true
	}()

	// Wait for either key press or signal
	select {
	case <-keyChan:
		// Key was pressed
		return
	case <-sigChan:
		// Interrupt signal received
		fmt.Println("\nInterrupted.")
		return
	}
}

func getAudioFilename(id, audioType string) string {
	switch audioType {
	case "V", "vocal", "vocals":
		return fmt.Sprintf("./data/%s_(Vocals)_UVR_MDXNET_Main.wav", id)
	case "I", "instrumental", "instrumentals":
		return fmt.Sprintf("./data/%s_(Instrumental)_UVR_MDXNET_Main.wav", id)
	default:
		return fmt.Sprintf("./data/%s.wav", id)
	}
}

func handleDemoCommand() {
	demoCmd := flag.NewFlagSet("demo", flag.ExitOnError)
	pitchShift := demoCmd.Int("pitch", 3, "Pitch shift in semitones (can be negative)")
	tempoChange := demoCmd.Float64("tempo", 0.0, "Tempo change as percentage (e.g., 30 for 30% faster, -20 for 20% slower)")
	
	demoCmd.Usage = func() {
		fmt.Println("Usage: starchive demo [options] <id> <I|V>")
		fmt.Println("Creates a 30-second demo from the middle of the track with pitch/tempo adjustments")
		fmt.Println("\nOptions:")
		demoCmd.PrintDefaults()
		fmt.Println("\nExamples:")
		fmt.Println("  starchive demo NdYWuo9OFAw I                    # +3 semitones, no tempo change")
		fmt.Println("  starchive demo -pitch -2 NdYWuo9OFAw V          # -2 semitones, no tempo change")
		fmt.Println("  starchive demo -tempo 25 NdYWuo9OFAw I          # +3 semitones, 25% faster")
		fmt.Println("  starchive demo -pitch 0 -tempo -15 NdYWuo9OFAw V # no pitch change, 15% slower")
	}

	if len(os.Args) < 4 {
		demoCmd.Usage()
		os.Exit(1)
	}

	// Parse flags starting from args[2:]
	if err := demoCmd.Parse(os.Args[2:]); err != nil {
		fmt.Println("Error parsing flags:", err)
		os.Exit(1)
	}

	// Get remaining args after flag parsing
	args := demoCmd.Args()
	if len(args) < 2 {
		demoCmd.Usage()
		os.Exit(1)
	}

	id := args[0]
	audioType := args[1]

	inputPath := getAudioFilename(id, audioType)

	// Check if input file exists
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		fmt.Printf("Error: Input file %s does not exist\n", inputPath)
		os.Exit(1)
	}

	// Get duration of the audio file
	duration, err := getAudioDuration(inputPath)
	if err != nil {
		fmt.Printf("Error getting audio duration: %v\n", err)
		os.Exit(1)
	}

	// Calculate middle position for 30-second clip
	startPosition := (duration - 30) / 2
	if startPosition < 0 {
		startPosition = 0
	}

	// Create temporary file path for 30-second clip
	tempClipPath := fmt.Sprintf("/tmp/%s_%s_30sec.wav", id, audioType)
	
	// Create temporary file path for final demo (also in /tmp)
	demoPath := fmt.Sprintf("/tmp/%s_%s_demo.wav", id, audioType)

	// Determine track description for display
	trackDesc := "main track"
	switch audioType {
	case "I", "instrumental", "instrumentals":
		trackDesc = "instrumental track"
	case "V", "vocal", "vocals":
		trackDesc = "vocal track"
	}

	fmt.Printf("Creating demo for %s (%s)\n", id, trackDesc)
	if *pitchShift != 0 {
		if *pitchShift > 0 {
			fmt.Printf("Pitch shift: +%d semitones\n", *pitchShift)
		} else {
			fmt.Printf("Pitch shift: %d semitones\n", *pitchShift)
		}
	}
	if *tempoChange != 0 {
		if *tempoChange > 0 {
			fmt.Printf("Tempo change: %.1f%% faster\n", *tempoChange)
		} else {
			fmt.Printf("Tempo change: %.1f%% slower\n", -*tempoChange)
		}
	}
	fmt.Printf("Extracting 30 seconds from position %.1fs...\n", startPosition)

	// Extract 30 seconds from the middle using ffmpeg
	extractCmd := exec.Command("ffmpeg", "-y",
		"-ss", fmt.Sprintf("%.1f", startPosition),
		"-t", "30",
		"-i", inputPath,
		"-c", "copy",
		tempClipPath)

	extractCmd.Stdout = os.Stdout
	extractCmd.Stderr = os.Stderr

	err = extractCmd.Run()
	if err != nil {
		fmt.Printf("Error extracting 30-second clip: %v\n", err)
		os.Exit(1)
	}

	// Ensure temp files are always cleaned up
	defer os.Remove(tempClipPath)
	defer os.Remove(demoPath)

	// Build rubberband command with appropriate flags
	rubberbandArgs := []string{"--fine", "--formant"}
	
	// Add pitch shift if specified
	if *pitchShift != 0 {
		rubberbandArgs = append(rubberbandArgs, "--pitch", fmt.Sprintf("%d", *pitchShift))
	}
	
	// Add tempo change if specified
	if *tempoChange != 0 {
		// Convert percentage to time ratio for rubberband
		// Positive % = faster = smaller time ratio (e.g., +25% faster = 0.8)
		// Negative % = slower = larger time ratio (e.g., -20% slower = 1.25)
		speedMultiplier := 1.0 + (*tempoChange / 100.0)
		timeRatio := 1.0 / speedMultiplier
		rubberbandArgs = append(rubberbandArgs, "--time", fmt.Sprintf("%.6f", timeRatio))
	}
	
	// Add input and output paths (process directly to demo file)
	rubberbandArgs = append(rubberbandArgs, tempClipPath, demoPath)

	fmt.Printf("Applying audio processing with rubberband...\n")

	// Apply processing using rubberband
	pitchCmd := exec.Command("rubberband", rubberbandArgs...)

	pitchCmd.Stdout = os.Stdout
	pitchCmd.Stderr = os.Stderr

	err = pitchCmd.Run()
	if err != nil {
		fmt.Printf("Error applying audio processing: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Playing processed demo (press any key to stop)...\n")

	// Play processed demo file with ffplay and wait for keypress
	playTempFile(demoPath)

	fmt.Printf("Demo preview completed. All temp files cleaned up.\n")
}

func playTempFile(filePath string) {
	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start playback in background
	go func() {
		// Use ffplay to play the temp file
		cmd := exec.CommandContext(ctx, "ffplay", 
			"-autoexit", // Exit when playback ends
			"-nodisp", // No video display (audio only)
			"-loglevel", "quiet", // Suppress output
			filePath)
		
		cmd.Run() // This will block until playback ends or context is canceled
	}()

	// Wait for any key press
	waitForKeyPress()
	cancel() // Stop playback
	fmt.Println("Preview stopped.")
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}