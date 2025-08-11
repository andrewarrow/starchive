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
				if cachedMetadata.VocalBPM != nil && cachedMetadata.InstrumentalBPM != nil {
					bmpInfo = fmt.Sprintf(" [BPM: V:%.1f/%s I:%.1f/%s]", 
						*cachedMetadata.VocalBPM, *cachedMetadata.VocalKey,
						*cachedMetadata.InstrumentalBPM, *cachedMetadata.InstrumentalKey)
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
			if metadata.VocalBPM != nil && metadata.InstrumentalBPM != nil {
				bmpInfo = fmt.Sprintf(" [BPM: V:%.1f/%s I:%.1f/%s]", 
					*metadata.VocalBPM, *metadata.VocalKey,
					*metadata.InstrumentalBPM, *metadata.InstrumentalKey)
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
	vocalPath := fmt.Sprintf("./data/%s_(Vocals)_UVR_MDXNET_Main.wav", id)
	instrumentalPath := fmt.Sprintf("./data/%s_(Instrumental)_UVR_MDXNET_Main.wav", id)

	// Check if both files exist
	if _, err := os.Stat(vocalPath); os.IsNotExist(err) {
		fmt.Printf("Error: Vocal file %s does not exist\n", vocalPath)
		os.Exit(1)
	}
	if _, err := os.Stat(instrumentalPath); os.IsNotExist(err) {
		fmt.Printf("Error: Instrumental file %s does not exist\n", instrumentalPath)
		os.Exit(1)
	}

	fmt.Printf("Analyzing BPM for %s...\n", id)
	
	// Run BPM analysis on vocal track
	fmt.Println("\n=== Vocal Track Analysis ===")
	vocalCmd := exec.Command("python3", "bpm/beats_per_min.py", vocalPath)
	vocalOutput, err := vocalCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error analyzing vocal track: %v\n", err)
		fmt.Printf("Output: %s\n", string(vocalOutput))
		os.Exit(1)
	}
	
	fmt.Printf("%s\n", string(vocalOutput))
	
	// Parse vocal track JSON
	var vocalData map[string]interface{}
	if err := json.Unmarshal(vocalOutput, &vocalData); err != nil {
		fmt.Printf("Error parsing vocal track JSON: %v\n", err)
		os.Exit(1)
	}

	// Run BPM analysis on instrumental track
	fmt.Println("\n=== Instrumental Track Analysis ===")
	instrumentalCmd := exec.Command("python3", "bpm/beats_per_min.py", instrumentalPath)
	instrumentalOutput, err := instrumentalCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error analyzing instrumental track: %v\n", err)
		fmt.Printf("Output: %s\n", string(instrumentalOutput))
		os.Exit(1)
	}
	
	fmt.Printf("%s\n", string(instrumentalOutput))
	
	// Parse instrumental track JSON
	var instrumentalData map[string]interface{}
	if err := json.Unmarshal(instrumentalOutput, &instrumentalData); err != nil {
		fmt.Printf("Error parsing instrumental track JSON: %v\n", err)
		os.Exit(1)
	}

	// Store results in database
	db, err := initDatabase()
	if err != nil {
		fmt.Printf("Warning: Could not initialize database to store BPM data: %v\n", err)
		return
	}
	defer db.Close()

	vocalBPM := vocalData["bpm"].(float64)
	vocalKey := vocalData["key"].(string)
	instrumentalBPM := instrumentalData["bpm"].(float64)
	instrumentalKey := instrumentalData["key"].(string)

	if err := storeBPMData(db, id, vocalBPM, vocalKey, instrumentalBPM, instrumentalKey); err != nil {
		fmt.Printf("Warning: Could not store BPM data in database: %v\n", err)
	} else {
		fmt.Printf("\nBPM data stored in database for %s\n", id)
	}
}