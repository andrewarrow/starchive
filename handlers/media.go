package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"starchive/media"
	"starchive/podpapyrus"
	"starchive/util"
)



func HandleDl() {
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

func HandleExternal() {
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
		"id":            id,
		"title":         title,
		"source":        "external",
		"original_path": sourceFilePath,
		"imported_at":   time.Now().Format(time.RFC3339),
		"filename":      filename,
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

func HandleUl() {
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

func HandleSmall() {
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

func HandlePodpapyrus() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: starchive podpapyrus <id_or_url>")
		fmt.Println("Examples:")
		fmt.Println("  starchive podpapyrus abc123")
		fmt.Println("  starchive podpapyrus https://www.youtube.com/watch?v=abc123")
		fmt.Println("This downloads thumbnail and VTT subtitle files, then creates a text file")
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

	// Currently only support YouTube
	if platform != "youtube" {
		fmt.Printf("Error: podpapyrus currently only supports YouTube videos\n")
		os.Exit(1)
	}

	if err := podpapyrus.ProcessCommandLine(id, podpapyrus.HandlerBasePath); err != nil {
		fmt.Printf("Error processing video: %v\n", err)
		os.Exit(1)
	}
}
