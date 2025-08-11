package main

import (
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
				fmt.Printf("%s\t%s\t[Vocal: %s]\n", cachedMetadata.ID, cachedMetadata.Title, vocalStatus)
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
			fmt.Printf("%s\t%s\t[Vocal: %s]\n", metadata.ID, metadata.Title, vocalStatus)
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
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error running audio-separator: %v\n", err)
		fmt.Printf("Output: %s\n", string(output))
		os.Exit(1)
	}

	fmt.Printf("Successfully separated vocals for %s\n", id)
	fmt.Printf("Output: %s\n", string(output))

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