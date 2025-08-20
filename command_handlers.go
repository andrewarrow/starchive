package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	
	"starchive/audio"
	"starchive/media"
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

func handleDlCommand() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: starchive dl <id_or_url>")
		fmt.Println("Examples:")
		fmt.Println("  starchive dl abc123")
		fmt.Println("  starchive dl https://www.youtube.com/watch?v=abc123")
		fmt.Println("  starchive dl https://www.instagram.com/p/DMxMgnvhwmK/")
		fmt.Println("  starchive dl https://www.instagram.com/reels/DMxMgnvhwmK/")
		os.Exit(1)
	}

	input := os.Args[2]
	
	// Extract ID and platform from input
	id, platform := media.ParseVideoInput(input)
	if id == "" {
		fmt.Printf("Error: Could not extract ID from input: %s\n", input)
		os.Exit(1)
	}
	
	fmt.Printf("Detected platform: %s, ID: %s\n", platform, id)
	
	_, err := media.DownloadVideo(id, platform)
	if err != nil {
		fmt.Printf("Error downloading video: %v\n", err)
		os.Exit(1)
	}
}

func handleExternalCommand() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: starchive external <file_path>")
		fmt.Println("Example: starchive external ~/Documents/cd_audio_from_gnr_lies.wav")
		fmt.Println("This will copy the external file into the data directory and create a metadata JSON file")
		os.Exit(1)
	}

	sourceFilePath := os.Args[2]
	
	// Expand tilde to home directory
	if strings.HasPrefix(sourceFilePath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("Error getting home directory: %v\n", err)
			os.Exit(1)
		}
		sourceFilePath = filepath.Join(homeDir, sourceFilePath[2:])
	}

	// Check if source file exists
	if _, err := os.Stat(sourceFilePath); os.IsNotExist(err) {
		fmt.Printf("Error: Source file %s does not exist\n", sourceFilePath)
		os.Exit(1)
	}

	// Get filename without extension for title and ID
	filename := filepath.Base(sourceFilePath)
	title := strings.TrimSuffix(filename, filepath.Ext(filename))
	
	// Generate ID from first 11 characters of filename (without extension)
	id := title
	if len(id) > 11 {
		id = id[:11]
	}

	fmt.Printf("Generated ID: %s\n", id)

	// Determine file extension
	ext := strings.ToLower(filepath.Ext(sourceFilePath))
	
	// Copy file to data directory
	dataDir := "./data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		fmt.Printf("Error creating data directory: %v\n", err)
		os.Exit(1)
	}

	destPath := filepath.Join(dataDir, id+ext)
	
	// Check if destination already exists
	if _, err := os.Stat(destPath); err == nil {
		fmt.Printf("File with ID %s already exists in data directory\n", id)
		os.Exit(1)
	}

	// Copy file
	sourceFile, err := os.Open(sourceFilePath)
	if err != nil {
		fmt.Printf("Error opening source file: %v\n", err)
		os.Exit(1)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		fmt.Printf("Error creating destination file: %v\n", err)
		os.Exit(1)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		fmt.Printf("Error copying file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Copied file to: %s\n", destPath)

	// Create JSON metadata file
	metadata := map[string]interface{}{
		"id":       id,
		"title":    title,
		"source":   "external",
		"original_path": sourceFilePath,
		"imported_at": time.Now().Format(time.RFC3339),
		"filename": filename,
	}

	jsonPath := filepath.Join(dataDir, id+".json")
	jsonData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		fmt.Printf("Error creating JSON metadata: %v\n", err)
		os.Exit(1)
	}

	err = os.WriteFile(jsonPath, jsonData, 0644)
	if err != nil {
		fmt.Printf("Error writing JSON file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created metadata file: %s\n", jsonPath)
	
	// Add to database
	db, err := util.InitDatabase()
	if err != nil {
		fmt.Printf("Warning: Could not initialize database: %v\n", err)
	} else {
		defer db.Close()
		
		videoMetadata := util.VideoMetadata{
			ID:           id,
			Title:        &title,
			LastModified: time.Now(),
			VocalDone:    false,
		}
		
		if err := db.SaveMetadata(&videoMetadata); err != nil {
			fmt.Printf("Warning: Could not save to database: %v\n", err)
		} else {
			fmt.Printf("Added to database with ID: %s\n", id)
		}
	}

	fmt.Printf("\nExternal file successfully imported!\n")
	fmt.Printf("ID: %s\n", id)
	fmt.Printf("Title: %s\n", title)
	fmt.Printf("File: %s\n", destPath)
}

func handleUlCommand() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: starchive ul <id>")
		fmt.Println("Example: starchive ul abc123")
		fmt.Println("This will upload the mp4 file with the given ID to YouTube")
		os.Exit(1)
	}

	id := os.Args[2]
	
	mp4Path := fmt.Sprintf("./data/%s.mp4", id)
	if _, err := os.Stat(mp4Path); os.IsNotExist(err) {
		fmt.Printf("Error: MP4 file %s does not exist\n", mp4Path)
		os.Exit(1)
	}

	uploadScript := "./media/upload_to_youtube.py"
	if _, err := os.Stat(uploadScript); os.IsNotExist(err) {
		fmt.Printf("Error: Upload script %s does not exist\n", uploadScript)
		os.Exit(1)
	}

	fmt.Printf("Uploading %s to YouTube...\n", mp4Path)
	
	absMP4Path, err := filepath.Abs(mp4Path)
	if err != nil {
		fmt.Printf("Error getting absolute path for mp4: %v\n", err)
		os.Exit(1)
	}
	
	cmd := exec.Command("python3", "upload_to_youtube.py", absMP4Path, "--title", id)
	cmd.Dir = "./media"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	err = cmd.Run()
	if err != nil {
		fmt.Printf("Error uploading to YouTube: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully uploaded %s to YouTube\n", id)
}

func handleSmallCommand() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: starchive small <id>")
		fmt.Println("Example: starchive small abc123")
		fmt.Println("This will create a small optimized video from data/id.mp4")
		os.Exit(1)
	}

	id := os.Args[2]
	inputPath := fmt.Sprintf("./data/%s.mp4", id)

	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		fmt.Printf("Error: Input file %s does not exist\n", inputPath)
		os.Exit(1)
	}

	fmt.Printf("Creating small optimized video for %s...\n", id)

	// Step 1: Prepare video with specific settings
	fmt.Println("Step 1: Preparing video...")
	prepCmd := exec.Command("ffmpeg", "-i", inputPath,
		"-vf", "fps=24,scale=242:428:flags=lanczos:force_original_aspect_ratio=decrease,pad=242:428:(ow-iw)/2:(oh-ih)/2,hqdn3d=1.5:1.5:6:6",
		"-c:v", "h264_videotoolbox", "-b:v", "2000k", "-maxrate", "2000k", "-bufsize", "4000k",
		"-c:a", "aac", "-b:a", "96k", "-ar", "48000", "-ac", "1",
		"-movflags", "+faststart",
		"tmp_prep.mp4")

	fmt.Printf("Running: %s\n", prepCmd.String())
	prepCmd.Stdout = os.Stdout
	prepCmd.Stderr = os.Stderr

	err := prepCmd.Run()
	if err != nil {
		fmt.Printf("Error in preparation step: %v\n", err)
		os.Exit(1)
	}

	// Step 2: First pass (no audio)
	fmt.Println("\nStep 2: First pass encoding (no audio)...")
	pass1Cmd := exec.Command("ffmpeg", "-y", "-i", "tmp_prep.mp4",
		"-c:v", "libx264", "-b:v", "150k", "-maxrate", "150k", "-bufsize", "300k",
		"-pix_fmt", "yuv420p", "-profile:v", "high", "-level", "3.1",
		"-x264-params", "aq-mode=2:aq-strength=1.0:rc-lookahead=40:ref=4:keyint=120:scenecut=40:deblock=1,1:me=umh:subme=8",
		"-an", "-f", "mp4", "/dev/null")

	fmt.Printf("Running: %s\n", pass1Cmd.String())
	pass1Cmd.Stdout = os.Stdout
	pass1Cmd.Stderr = os.Stderr

	err = pass1Cmd.Run()
	if err != nil {
		fmt.Printf("Error in first pass: %v\n", err)
		os.Exit(1)
	}

	// Step 3: Second pass (add audio)
	fmt.Println("\nStep 3: Second pass encoding (with audio)...")
	outputPath := fmt.Sprintf("./data/%s-small.mp4", id)
	pass2Cmd := exec.Command("ffmpeg", "-i", "tmp_prep.mp4",
		"-c:v", "libx264", "-b:v", "150k", "-maxrate", "150k", "-bufsize", "300k",
		"-pix_fmt", "yuv420p", "-profile:v", "high", "-level", "3.1",
		"-x264-params", "aq-mode=2:aq-strength=1.0:rc-lookahead=40:ref=4:keyint=120:scenecut=40:deblock=1,1:me=umh:subme=8",
		"-c:a", "aac", "-b:a", "32k", "-ar", "48000", "-ac", "1",
		"-movflags", "+faststart",
		outputPath)

	fmt.Printf("Running: %s\n", pass2Cmd.String())
	pass2Cmd.Stdout = os.Stdout
	pass2Cmd.Stderr = os.Stderr

	err = pass2Cmd.Run()
	if err != nil {
		fmt.Printf("Error in second pass: %v\n", err)
		os.Exit(1)
	}

	// Clean up temporary file
	if err := os.Remove("tmp_prep.mp4"); err != nil {
		fmt.Printf("Warning: Could not remove temporary file tmp_prep.mp4: %v\n", err)
	}

	fmt.Printf("\nSuccessfully created small optimized video: %s\n", outputPath)
}