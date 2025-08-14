package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	
	"starchive/audio"
	"starchive/util"
)

func handleLsCommand() {
	db, err := util.InitDatabase()
	if err != nil {
		fmt.Printf("Error initializing database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	
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
			
			if cachedMetadata, found := db.GetCachedMetadata(id); found {
				vocalStatus := "N"
				if cachedMetadata.VocalDone {
					vocalStatus = "Y"
				}
				bmpInfo := ""
				if cachedMetadata.BPM != nil && cachedMetadata.Key != nil {
					bmpInfo = fmt.Sprintf("%.1f/%s", 
						*cachedMetadata.BPM, *cachedMetadata.Key)
				}
				title := ""
				if cachedMetadata.Title != nil {
					title = *cachedMetadata.Title
				}
				fmt.Printf("%-15s %-60s %-1s %-10s\n", cachedMetadata.ID, util.TruncateString(title, 60), vocalStatus, bmpInfo)
				continue
			}
			
			// Parse JSON file if not in cache or cache is stale
			filePath := filepath.Join("./data", id+".json")
			metadata, err := util.ParseJSONMetadata(filePath)
			if err != nil {
				fmt.Printf("%s\t<error parsing file: %v>\n", id, err)
				continue
			}
			
			// Cache the metadata
			if err := db.CacheMetadata(*metadata); err != nil {
				fmt.Printf("Warning: failed to cache metadata for %s: %v\n", id, err)
			}
			
			// Get the updated metadata
			if updatedMetadata, found := db.GetCachedMetadata(id); found {
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
			title := ""
			if metadata.Title != nil {
				title = *metadata.Title
			}
			fmt.Printf("%-15s %-60s %-1s %-10s\n", metadata.ID, util.TruncateString(title, 60), vocalStatus, bmpInfo)
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

	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		fmt.Printf("Error: Input file %s does not exist\n", inputPath)
		os.Exit(1)
	}

	cmd := exec.Command("audio-separator", inputPath,
		"--output_dir", "./data/",
		"--model_filename", "UVR_MDXNET_Main.onnx",
		"--output_format", "wav")

	fmt.Printf("Running: %s\n", cmd.String())
	
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error running audio-separator: %v\n", err)
		os.Exit(1)
	}

	if strings.HasPrefix(id, "_") {
		expectedVocalPath := audio.GetVocalFilePath(id)
		expectedInstrumentalPath := audio.GetInstrumentalFilePath(id)
		
		strippedID := strings.TrimPrefix(id, "_")
		actualVocalPath := audio.GetVocalFilePath(strippedID)
		actualInstrumentalPath := audio.GetInstrumentalFilePath(strippedID)
		
		if _, err := os.Stat(actualVocalPath); err == nil {
			if err := os.Rename(actualVocalPath, expectedVocalPath); err != nil {
				fmt.Printf("Warning: Could not rename vocal file: %v\n", err)
			} else {
				fmt.Printf("Renamed: %s -> %s\n", filepath.Base(actualVocalPath), filepath.Base(expectedVocalPath))
			}
		}
		
		if _, err := os.Stat(actualInstrumentalPath); err == nil {
			if err := os.Rename(actualInstrumentalPath, expectedInstrumentalPath); err != nil {
				fmt.Printf("Warning: Could not rename instrumental file: %v\n", err)
			} else {
				fmt.Printf("Renamed: %s -> %s\n", filepath.Base(actualInstrumentalPath), filepath.Base(expectedInstrumentalPath))
			}
		}
	}

	fmt.Printf("Successfully separated vocals for %s\n", id)

	db, err := util.InitDatabase()
	if err != nil {
		fmt.Printf("Warning: Could not initialize database to mark vocal as done: %v\n", err)
		return
	}
	defer db.Close()

	if err := db.MarkVocalDone(id); err != nil {
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

	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		fmt.Printf("Error: Input file %s does not exist\n", inputPath)
		os.Exit(1)
	}

	fmt.Printf("Analyzing BPM for %s...\n", id)
	
	bpmCmd := exec.Command("python3", "bpm/beats_per_min.py", inputPath)
	output, err := bpmCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error analyzing BPM: %v\n", err)
		fmt.Printf("Output: %s\n", string(output))
		os.Exit(1)
	}
	
	fmt.Printf("%s\n", string(output))
	
	var bpmData map[string]interface{}
	if err := json.Unmarshal(output, &bpmData); err != nil {
		fmt.Printf("Error parsing BPM JSON: %v\n", err)
		os.Exit(1)
	}

	db, err := util.InitDatabase()
	if err != nil {
		fmt.Printf("Warning: Could not initialize database to store BPM data: %v\n", err)
		return
	}
	defer db.Close()

	bpm := bpmData["bpm"].(float64)
	key := bpmData["key"].(string)

	if err := db.StoreBPMData(id, bpm, key); err != nil {
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

	db, err := util.InitDatabase()
	if err != nil {
		fmt.Printf("Error initializing database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	metadata1, found1 := db.GetCachedMetadata(id1)
	metadata2, found2 := db.GetCachedMetadata(id2)

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

	ratio1to2 := bpm2 / bpm1
	ratio2to1 := bpm1 / bpm2

	fmt.Printf("Tempo ratios:\n")
	fmt.Printf("  %s->%s: %.3f (%.1f%%)\n", id1, id2, ratio1to2, (ratio1to2-1)*100)
	fmt.Printf("  %s->%s: %.3f (%.1f%%)\n", id2, id1, ratio2to1, (ratio2to1-1)*100)

	if !metadata1.VocalDone {
		fmt.Printf("Error: Vocal separation not done for %s. Run 'starchive vocal %s' first\n", id1, id1)
		os.Exit(1)
	}
	if !metadata2.VocalDone {
		fmt.Printf("Error: Vocal separation not done for %s. Run 'starchive vocal %s' first\n", id2, id2)
		os.Exit(1)
	}

	invRatio1to2 := 1.0 / ratio1to2
	invRatio2to1 := 1.0 / ratio2to1

	fmt.Printf("Inverse ratios:\n")
	fmt.Printf("  %s inverse: %.3f (%.1f%%)\n", id1, invRatio1to2, (invRatio1to2-1)*100)
	fmt.Printf("  %s inverse: %.3f (%.1f%%)\n", id2, invRatio2to1, (invRatio2to1-1)*100)

	files := []string{
		audio.GetVocalFilename(id1),
		audio.GetInstrumentalFilename(id1),
		audio.GetVocalFilename(id2),
		audio.GetInstrumentalFilename(id2),
		audio.GetVocalFilename(id1),
		audio.GetInstrumentalFilename(id1),
		audio.GetVocalFilename(id2),
		audio.GetInstrumentalFilename(id2),
	}

	ratios := []float64{ratio1to2, ratio1to2, ratio2to1, ratio2to1, invRatio1to2, invRatio1to2, invRatio2to1, invRatio2to1}
	suffixes := []string{"sync_to_" + id2, "sync_to_" + id2, "sync_to_" + id1, "sync_to_" + id1, 
		"inv_sync_to_" + id2, "inv_sync_to_" + id2, "inv_sync_to_" + id1, "inv_sync_to_" + id1}

	for i, file := range files {
		inputPath := fmt.Sprintf("./data/%s", file)
		
		if _, err := os.Stat(inputPath); os.IsNotExist(err) {
			fmt.Printf("Error: Input file %s does not exist\n", inputPath)
			os.Exit(1)
		}

		baseName := strings.TrimSuffix(file, ".wav")
		outputPath := fmt.Sprintf("./data/%s_%s.wav", baseName, suffixes[i])

		fmt.Printf("\nProcessing: %s -> %s (ratio: %.3f)\n", file, filepath.Base(outputPath), ratios[i])

		cmd := exec.Command("rubberband", 
			"--time", fmt.Sprintf("%.6f", ratios[i]),
			"--fine",
			"--formant",
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