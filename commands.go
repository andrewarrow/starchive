package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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
				vocalStatus := "NO"
				if cachedMetadata.VocalDone {
					vocalStatus = "YES"
				}
				bmpInfo := ""
				if cachedMetadata.BPM != nil && cachedMetadata.Key != nil {
					bmpInfo = fmt.Sprintf(" [BPM: %.1f/%s]", 
						*cachedMetadata.BPM, *cachedMetadata.Key)
				}
				fmt.Printf("%s\t%s\t[Vocal: %s]%s\n", cachedMetadata.ID, cachedMetadata.Title, vocalStatus, bmpInfo)
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
			
			vocalStatus := "NO"
			if metadata.VocalDone {
				vocalStatus = "YES"
			}
			bmpInfo := ""
			if metadata.BPM != nil && metadata.Key != nil {
				bmpInfo = fmt.Sprintf(" [BPM: %.1f/%s]", 
					*metadata.BPM, *metadata.Key)
			}
			fmt.Printf("%s\t%s\t[Vocal: %s]%s\n", metadata.ID, metadata.Title, vocalStatus, bmpInfo)
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