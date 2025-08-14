package util

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"starchive/media"
)

// HandleRmCommand removes all files with specified ID from data directory
func HandleRmCommand(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: starchive rm <id>")
		fmt.Println("Example: starchive rm abc123")
		fmt.Println("This will remove all files matching data/<id>*")
		os.Exit(1)
	}

	id := args[0]
	dataDir := "./data"
	
	// Check if data directory exists
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		fmt.Printf("Data directory %s does not exist\n", dataDir)
		os.Exit(1)
	}

	// Read directory entries
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		fmt.Printf("Error reading data directory: %v\n", err)
		os.Exit(1)
	}

	var filesToRemove []string
	for _, entry := range entries {
		filename := entry.Name()
		if filename == id || 
		   filename == id+".mp4" ||
		   filename == id+".wav" ||
		   filename == id+".json" ||
		   filename == id+".jpg" ||
		   filename == id+".txt" ||
		   filename == id+".vtt" ||
		   filename == id+".en.vtt" ||
		   filename == id+"_(Vocals)_UVR_MDXNET_Main.wav" ||
		   filename == id+"_(Instrumental)_UVR_MDXNET_Main.wav" {
			filesToRemove = append(filesToRemove, filename)
		}
	}

	if len(filesToRemove) == 0 {
		fmt.Printf("No files found matching ID: %s\n", id)
		return
	}

	fmt.Printf("Found %d files to remove:\n", len(filesToRemove))
	for _, filename := range filesToRemove {
		fmt.Printf("  %s\n", filename)
	}

	fmt.Printf("Remove these files? (y/N): ")
	var response string
	fmt.Scanln(&response)
	
	if response != "y" && response != "Y" && response != "yes" {
		fmt.Println("Cancelled.")
		return
	}

	removedCount := 0
	for _, filename := range filesToRemove {
		fullPath := filepath.Join(dataDir, filename)
		
		// Check if it's a directory
		if stat, err := os.Stat(fullPath); err == nil && stat.IsDir() {
			err = os.RemoveAll(fullPath)
		} else {
			err = os.Remove(fullPath)
		}
		
		if err != nil {
			fmt.Printf("Error removing %s: %v\n", filename, err)
		} else {
			fmt.Printf("Removed: %s\n", filename)
			removedCount++
		}
	}

	fmt.Printf("Removed %d files\n", removedCount)
}

// HandleRetryCommand retries downloading specific components
func HandleRetryCommand(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: starchive retry <id> <component> [component...]")
		fmt.Println("Components: vtt, json, thumbnail, video")
		fmt.Println("Example: starchive retry abc123 vtt json")
		os.Exit(1)
	}

	id := args[0]
	components := args[1:]

	for _, component := range components {
		switch component {
		case "vtt":
			retryVTT(id)
		case "json":
			retryJSON(id)
		case "thumbnail":
			retryThumbnail(id)
		case "video":
			retryVideo(id)
		default:
			fmt.Printf("Unknown component: %s\n", component)
		}
	}
}

func retryVTT(id string) {
	fmt.Printf("Retrying VTT download for %s...\n", id)
	cmd := exec.Command("yt-dlp", "--write-auto-sub", "--sub-lang", "en", 
		"--skip-download", "--output", fmt.Sprintf("./data/%s.%%(ext)s", id),
		fmt.Sprintf("https://www.youtube.com/watch?v=%s", id))
	
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error downloading VTT: %v\n", err)
		return
	}
	
	fmt.Printf("VTT download completed for %s\n", id)
	
	// Parse VTT file to create .txt file
	vttPath := fmt.Sprintf("./data/%s.en.vtt", id)
	if err := media.ParseVttFile(vttPath, id); err != nil {
		fmt.Printf("Warning: failed to parse VTT file: %v\n", err)
	} else {
		fmt.Printf("Created .txt file from VTT for %s\n", id)
	}
}

func retryJSON(id string) {
	fmt.Printf("Retrying JSON metadata download for %s...\n", id)
	cmd := exec.Command("yt-dlp", "--write-info-json", "--skip-download",
		"--output", fmt.Sprintf("./data/%s.%%(ext)s", id),
		fmt.Sprintf("https://www.youtube.com/watch?v=%s", id))
	
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error downloading JSON: %v\n", err)
	} else {
		fmt.Printf("JSON download completed for %s\n", id)
	}
}

func retryThumbnail(id string) {
	fmt.Printf("Retrying thumbnail download for %s...\n", id)
	cmd := exec.Command("yt-dlp", "--write-thumbnail", "--skip-download",
		"--output", fmt.Sprintf("./data/%s.%%(ext)s", id),
		fmt.Sprintf("https://www.youtube.com/watch?v=%s", id))
	
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error downloading thumbnail: %v\n", err)
	} else {
		fmt.Printf("Thumbnail download completed for %s\n", id)
	}
}

func retryVideo(id string) {
	fmt.Printf("Retrying video download for %s...\n", id)
	cmd := exec.Command("yt-dlp", "--format", "best[height<=720]",
		"--output", fmt.Sprintf("./data/%s.%%(ext)s", id),
		fmt.Sprintf("https://www.youtube.com/watch?v=%s", id))
	
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error downloading video: %v\n", err)
	} else {
		fmt.Printf("Video download completed for %s\n", id)
	}
}