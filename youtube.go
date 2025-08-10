package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func IsYouTubeID(input string) bool {
	return len(input) == 11 && !strings.Contains(input, ".")
}

func DownloadVideo(youtubeID string) (string, error) {

	mFile := fmt.Sprintf("./data/%s.mp4", youtubeID)

	if _, err := os.Stat(mFile); err == nil {
		fmt.Printf("Video %s.mov already exists, skipping\n", youtubeID)
		return youtubeID, nil
	}

	DownloadSubtitles(youtubeID)
	DownloadThumbnail(youtubeID)
	DownloadJSON(youtubeID)
	if !downloadVideos {
		return youtubeID, nil
	}

	cmd := exec.Command("yt-dlp",
		"-o", "./data/%(id)s.%(ext)s",
		"-f", "bv*[vcodec^=avc1][ext=mp4]+ba[acodec^=mp4a][ext=m4a]/best[ext=mp4][vcodec^=avc1]",
		"--merge-output-format", "mp4",
		youtubeID)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error downloading and converting YouTube video: %v", err)
	}

	return youtubeID, nil
}

func DownloadSubtitles(youtubeID string) error {
	vttFile := youtubeID + ".en.vtt"

	// Check if .en.vtt file already exists
	if _, err := os.Stat(vttFile); err == nil {
		fmt.Printf("Subtitles file %s already exists, skipping download\n", vttFile)
		return nil
	}

	fmt.Printf("Downloading subtitles...\n")
	youtubeURL := "https://www.youtube.com/watch?v=" + youtubeID

	// Retry with exponential backoff up to 50 times
	var lastErr error
	for attempt := 1; attempt <= 1; attempt++ {
		subCmd := exec.Command("yt-dlp", "-o", "./data/"+youtubeID, "--skip-download", "--write-auto-sub", "--sub-lang", "en", youtubeURL)
		subCmd.Stdout = os.Stdout
		subCmd.Stderr = os.Stderr

		if err := subCmd.Run(); err != nil {
			lastErr = err
			if attempt < 50 {
				// Exponential backoff: wait 2^(attempt-1) seconds, capped at 60 seconds
				delay := time.Duration(1<<uint(attempt-1)) * time.Second
				if delay > 60*time.Second {
					delay = 60 * time.Second
				}
				fmt.Printf("Subtitle download failed (attempt %d/50), retrying in %v...\n", attempt, delay)
				time.Sleep(delay)
				continue
			}
		} else {
			// Success - parse the VTT file
			vttPath := fmt.Sprintf("./data/%s.en.vtt", youtubeID)
			if err := parseVttFile(vttPath, youtubeID); err != nil {
				fmt.Printf("Warning: failed to parse VTT file: %v\n", err)
			}
			return nil
		}
	}

	return fmt.Errorf("could not download subtitles after 50 attempts: %v", lastErr)
}

func DownloadThumbnail(youtubeID string) error {
	jpgPath := fmt.Sprintf("./data/%s.jpg", youtubeID)

	if _, err := os.Stat(jpgPath); err == nil {
		fmt.Printf("Thumbnail %s already exists, skipping download\n", jpgPath)
		return nil
	}

	fmt.Printf("Downloading thumbnail...\n")

	cmd := exec.Command(
		"yt-dlp",
		"-o", "./data/"+youtubeID,
		"--skip-download",
		"--write-thumbnail",
		"--convert-thumbnails", "jpg",
		youtubeID,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error downloading thumbnail: %v", err)
	}

	return nil
}

func DownloadJSON(youtubeID string) error {
	jsonPath := fmt.Sprintf("./data/%s.json", youtubeID)

	if _, err := os.Stat(jsonPath); err == nil {
		fmt.Printf("JSON metadata %s already exists, skipping download\n", jsonPath)
		return nil
	}

	fmt.Printf("Downloading JSON metadata...\n")

	cmd := exec.Command(
		"yt-dlp",
		"-j",
		"--no-warnings",
		youtubeID,
	)

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("error downloading JSON metadata: %v", err)
	}

	file, err := os.Create(jsonPath)
	if err != nil {
		return fmt.Errorf("error creating JSON file: %v", err)
	}
	defer file.Close()

	_, err = file.Write(output)
	if err != nil {
		return fmt.Errorf("error writing JSON file: %v", err)
	}

	return nil
}
