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

	// Process all files: vocals and instrumentals for both directions
	files := []string{
		fmt.Sprintf("%s_(Vocals)_UVR_MDXNET_Main.wav", id1),
		fmt.Sprintf("%s_(Instrumental)_UVR_MDXNET_Main.wav", id1),
		fmt.Sprintf("%s_(Vocals)_UVR_MDXNET_Main.wav", id2),
		fmt.Sprintf("%s_(Instrumental)_UVR_MDXNET_Main.wav", id2),
	}

	ratios := []float64{ratio1to2, ratio1to2, ratio2to1, ratio2to1}
	targetIds := []string{id2, id2, id1, id1}

	for i, file := range files {
		inputPath := fmt.Sprintf("./data/%s", file)
		
		// Check if input file exists
		if _, err := os.Stat(inputPath); os.IsNotExist(err) {
			fmt.Printf("Error: Input file %s does not exist\n", inputPath)
			os.Exit(1)
		}

		// Create output filename
		baseName := strings.TrimSuffix(file, ".wav")
		outputPath := fmt.Sprintf("./data/%s_sync_to_%s.wav", baseName, targetIds[i])

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

	fmt.Printf("\nSync complete! Created synchronized versions of all tracks.\n")
	fmt.Printf("Files created:\n")
	for i, file := range files {
		baseName := strings.TrimSuffix(file, ".wav")
		outputName := fmt.Sprintf("%s_sync_to_%s.wav", baseName, targetIds[i])
		fmt.Printf("  %s\n", outputName)
	}
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