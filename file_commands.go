package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func handleRmCommand() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: starchive rm <id>")
		fmt.Println("Example: starchive rm abc123")
		fmt.Println("This will remove all files matching data/<id>*")
		os.Exit(1)
	}

	id := os.Args[2]
	dataDir := "./data"
	
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

func handleRetryCommand() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: starchive retry <subcommand> <id>")
		fmt.Println("Subcommands:")
		fmt.Println("  vtt        Retry downloading subtitles/vtt file for the given ID")
		fmt.Println("  json       Retry downloading JSON metadata for the given ID")
		fmt.Println("  thumbnail  Retry downloading thumbnail for the given ID")
		fmt.Println("  video      Retry downloading video for the given ID")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  starchive retry vtt YVVhTQyL6po")
		fmt.Println("  starchive retry json 4qEpXAu4fAU")
		fmt.Println("  starchive retry thumbnail NdYWuo9OFAw")
		fmt.Println("  starchive retry video 8IXXOZhkaPE")
		os.Exit(1)
	}

	subcommand := os.Args[2]
	id := os.Args[3]

	if !IsYouTubeID(id) {
		fmt.Printf("Error: '%s' doesn't look like a valid YouTube video ID (should be 11 characters)\n", id)
		os.Exit(1)
	}

	switch subcommand {
	case "vtt", "subtitles", "subs":
		handleRetryVtt(id)
	case "json", "metadata":
		handleRetryJson(id)
	case "thumbnail", "thumb", "jpg":
		handleRetryThumbnail(id)
	case "video", "mp4":
		handleRetryVideo(id)
	default:
		fmt.Printf("Unknown retry subcommand: %s\n", subcommand)
		fmt.Println("Valid subcommands: vtt, json, thumbnail, video")
		os.Exit(1)
	}
}

func handleRetryVtt(id string) {
	fmt.Printf("Retrying VTT download for ID: %s\n", id)
	
	vttPath := fmt.Sprintf("./data/%s.en.vtt", id)
	txtPath := fmt.Sprintf("./data/%s.txt", id)
	
	if _, err := os.Stat(vttPath); err == nil {
		fmt.Printf("Removing existing VTT file: %s\n", vttPath)
		if err := os.Remove(vttPath); err != nil {
			fmt.Printf("Warning: Could not remove existing VTT file: %v\n", err)
		}
	}
	
	if _, err := os.Stat(txtPath); err == nil {
		fmt.Printf("Removing existing TXT file: %s\n", txtPath)
		if err := os.Remove(txtPath); err != nil {
			fmt.Printf("Warning: Could not remove existing TXT file: %v\n", err)
		}
	}

	if err := DownloadSubtitles(id); err != nil {
		fmt.Printf("Error downloading VTT: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Successfully retried VTT download for %s\n", id)
}

func handleRetryJson(id string) {
	fmt.Printf("Retrying JSON download for ID: %s\n", id)
	
	jsonPath := fmt.Sprintf("./data/%s.json", id)
	if _, err := os.Stat(jsonPath); err == nil {
		fmt.Printf("Removing existing JSON file: %s\n", jsonPath)
		if err := os.Remove(jsonPath); err != nil {
			fmt.Printf("Warning: Could not remove existing JSON file: %v\n", err)
		}
	}

	if err := DownloadJSON(id); err != nil {
		fmt.Printf("Error downloading JSON: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Successfully retried JSON download for %s\n", id)
}

func handleRetryThumbnail(id string) {
	fmt.Printf("Retrying thumbnail download for ID: %s\n", id)
	
	jpgPath := fmt.Sprintf("./data/%s.jpg", id)
	if _, err := os.Stat(jpgPath); err == nil {
		fmt.Printf("Removing existing thumbnail file: %s\n", jpgPath)
		if err := os.Remove(jpgPath); err != nil {
			fmt.Printf("Warning: Could not remove existing thumbnail file: %v\n", err)
		}
	}

	if err := DownloadThumbnail(id); err != nil {
		fmt.Printf("Error downloading thumbnail: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Successfully retried thumbnail download for %s\n", id)
}

func handleRetryVideo(id string) {
	fmt.Printf("Retrying video download for ID: %s\n", id)
	
	mp4Path := fmt.Sprintf("./data/%s.mp4", id)
	wavPath := fmt.Sprintf("./data/%s.wav", id)
	
	if _, err := os.Stat(mp4Path); err == nil {
		fmt.Printf("Removing existing MP4 file: %s\n", mp4Path)
		if err := os.Remove(mp4Path); err != nil {
			fmt.Printf("Warning: Could not remove existing MP4 file: %v\n", err)
		}
	}
	
	if _, err := os.Stat(wavPath); err == nil {
		fmt.Printf("Removing existing WAV file: %s\n", wavPath)
		if err := os.Remove(wavPath); err != nil {
			fmt.Printf("Warning: Could not remove existing WAV file: %v\n", err)
		}
	}

	originalDownloadVideos := downloadVideos
	downloadVideos = true
	defer func() { downloadVideos = originalDownloadVideos }()

	fmt.Printf("Downloading video using yt-dlp...\n")
	youtubeURL := "https://www.youtube.com/watch?v=" + id
	
	cmd := exec.Command("yt-dlp",
		"--cookies", "./cookies.txt",
		"-o", "./data/%(id)s.%(ext)s",
		"-f", "bv*[vcodec^=avc1][ext=mp4]+ba[acodec^=mp4a][ext=m4a]/best[ext=mp4][vcodec^=avc1]",
		"--merge-output-format", "mp4",
		youtubeURL)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error downloading video: %v\n", err)
		os.Exit(1)
	}

	if err := EnsureWav(id); err != nil {
		fmt.Printf("Error creating WAV: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Successfully retried video download for %s\n", id)
}