package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}