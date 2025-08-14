package media

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// IsYouTubeID checks if input string is a valid YouTube video ID
func IsYouTubeID(input string) bool {
	return len(input) == 11 && !strings.Contains(input, ".")
}

// DownloadVideo downloads a YouTube video by ID
func DownloadVideo(youtubeID string) (string, error) {
	mFile := fmt.Sprintf("./data/%s.mp4", youtubeID)

	if _, err := os.Stat(mFile); err == nil {
		fmt.Printf("Video %s.mp4 already exists, skipping\n", youtubeID)
		return mFile, nil
	}

	fmt.Printf("Downloading video %s...\n", youtubeID)

	cmd := exec.Command("yt-dlp", 
		"--cookies", "./cookies.txt",
		"--format", "best[height<=720]",
		"--output", fmt.Sprintf("./data/%s.%%(ext)s", youtubeID),
		fmt.Sprintf("https://www.youtube.com/watch?v=%s", youtubeID))

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to download video: %v", err)
	}

	fmt.Printf("Video downloaded: %s\n", mFile)
	return mFile, nil
}

// DownloadVideoComponents downloads various components of a YouTube video
func DownloadVideoComponents(youtubeID string, downloadVideo bool) error {
	fmt.Printf("Processing YouTube ID: %s\n", youtubeID)

	// Create data directory if it doesn't exist
	if err := os.MkdirAll("./data", 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %v", err)
	}

	// Download metadata (JSON)
	if err := downloadJSON(youtubeID); err != nil {
		fmt.Printf("Warning: Failed to download JSON metadata: %v\n", err)
	}

	// Download thumbnail
	if err := downloadThumbnail(youtubeID); err != nil {
		fmt.Printf("Warning: Failed to download thumbnail: %v\n", err)
	}

	// Download subtitles (VTT)
	if err := downloadVTT(youtubeID); err != nil {
		fmt.Printf("Warning: Failed to download VTT subtitles: %v\n", err)
	}

	// Download audio
	if err := downloadAudio(youtubeID); err != nil {
		return fmt.Errorf("failed to download audio: %v", err)
	}

	// Optionally download video
	if downloadVideo {
		if _, err := DownloadVideo(youtubeID); err != nil {
			fmt.Printf("Warning: Failed to download video: %v\n", err)
		}
	}

	return nil
}

func downloadJSON(youtubeID string) error {
	jsonFile := fmt.Sprintf("./data/%s.json", youtubeID)
	if _, err := os.Stat(jsonFile); err == nil {
		return nil // Already exists
	}

	cmd := exec.Command("yt-dlp", 
		"--cookies", "./cookies.txt",
		"--write-info-json", 
		"--skip-download",
		"--output", fmt.Sprintf("./data/%s.%%(ext)s", youtubeID),
		fmt.Sprintf("https://www.youtube.com/watch?v=%s", youtubeID))

	return cmd.Run()
}

func downloadThumbnail(youtubeID string) error {
	thumbnailFile := fmt.Sprintf("./data/%s.jpg", youtubeID)
	if _, err := os.Stat(thumbnailFile); err == nil {
		return nil // Already exists
	}

	cmd := exec.Command("yt-dlp", 
		"--cookies", "./cookies.txt",
		"--write-thumbnail", 
		"--skip-download",
		"--output", fmt.Sprintf("./data/%s.%%(ext)s", youtubeID),
		fmt.Sprintf("https://www.youtube.com/watch?v=%s", youtubeID))

	return cmd.Run()
}

func downloadVTT(youtubeID string) error {
	vttFile := fmt.Sprintf("./data/%s.en.vtt", youtubeID)
	if _, err := os.Stat(vttFile); err == nil {
		return nil // Already exists
	}

	cmd := exec.Command("yt-dlp", 
		"--cookies", "./cookies.txt",
		"--write-auto-sub", 
		"--sub-lang", "en",
		"--skip-download",
		"--output", fmt.Sprintf("./data/%s.%%(ext)s", youtubeID),
		fmt.Sprintf("https://www.youtube.com/watch?v=%s", youtubeID))

	return cmd.Run()
}

func downloadAudio(youtubeID string) error {
	wavFile := fmt.Sprintf("./data/%s.wav", youtubeID)
	if _, err := os.Stat(wavFile); err == nil {
		return nil // Already exists
	}

	fmt.Printf("Downloading and converting audio for %s...\n", youtubeID)

	// Use yt-dlp to download best audio and convert to WAV
	cmd := exec.Command("yt-dlp",
		"--cookies", "./cookies.txt",
		"--extract-audio",
		"--audio-format", "wav",
		"--audio-quality", "0", // best quality
		"--output", fmt.Sprintf("./data/%s.%%(ext)s", youtubeID),
		fmt.Sprintf("https://www.youtube.com/watch?v=%s", youtubeID))

	// Set timeout for long downloads
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("download failed: %v", err)
		}
		fmt.Printf("Audio downloaded and converted: %s\n", wavFile)
		return nil
	case <-time.After(10 * time.Minute):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return fmt.Errorf("download timeout after 10 minutes")
	}
}