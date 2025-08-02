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
	mp4File := youtubeID

	fmt.Printf("Detected YouTube ID: %s, downloading and converting to MOV...\n", youtubeID)

	// Use yt-dlp with ffmpeg post-processing to convert directly to MOV
	cmd := exec.Command("yt-dlp",
		"-o", "./data/"+mp4File,
		"--restrict-filenames",
		//"--exec", fmt.Sprintf("ffmpeg -i {} -c:v h264_videotoolbox -b:v 10000k -c:a copy ./data/%s.mov && rm {}", youtubeID),
		"--exec", fmt.Sprintf("ffmpeg -i {} -c:v h264_videotoolbox -b:v 10000k ./data/%s.mov && rm {}", youtubeID),
		youtubeID)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error downloading and converting YouTube video: %v", err)
	}

	return mp4File, nil
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
			// Success
			return nil
		}
	}

	return fmt.Errorf("could not download subtitles after 50 attempts: %v", lastErr)
}
