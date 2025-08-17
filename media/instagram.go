package media

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
)

func IsInstagramID(input string) bool {
	return regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(input) && len(input) > 5
}

func ParseInstagramInput(input string) string {
	// Instagram URL patterns
	instagramPatterns := []*regexp.Regexp{
		regexp.MustCompile(`instagram\.com/(?:p|reels)/([a-zA-Z0-9_-]+)`),
	}
	
	// Check Instagram patterns
	for _, pattern := range instagramPatterns {
		if match := pattern.FindStringSubmatch(input); match != nil {
			return match[1]
		}
	}
	
	// Check if it's a raw Instagram ID
	if IsInstagramID(input) {
		return input
	}
	
	return ""
}

func DownloadInstagram(videoID string) (string, error) {
	mFile := fmt.Sprintf("./data/%s.mp4", videoID)

	if _, err := os.Stat(mFile); err == nil {
		fmt.Printf("Video %s.mp4 already exists, skipping\n", videoID)
		return videoID, nil
	}

	// Get Instagram cookie file
	cookieFile := "./cookies_instagram.txt"
	
	// Download metadata first
	DownloadInstagramThumbnail(videoID, cookieFile)
	DownloadInstagramJSON(videoID, cookieFile)

	fmt.Printf("Downloading Instagram video %s...\n", videoID)

	// Try reels first, fallback to posts
	videoURL := "https://www.instagram.com/reels/" + videoID + "/"

	cmd := exec.Command("yt-dlp",
		"--cookies", cookieFile,
		"-o", "./data/%(id)s.%(ext)s",
		"-f", "bv*[vcodec^=avc1][ext=mp4]+ba[acodec^=mp4a][ext=m4a]/best[ext=mp4][vcodec^=avc1]",
		"--merge-output-format", "mp4",
		videoURL)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error downloading and converting Instagram video: %v", err)
	}

	if err := EnsureWav(videoID); err != nil {
		fmt.Printf("Warning: failed to create WAV: %v\n", err)
	}

	return videoID, nil
}

func DownloadInstagramThumbnail(videoID, cookieFile string) error {
	jpgPath := fmt.Sprintf("./data/%s.jpg", videoID)

	if _, err := os.Stat(jpgPath); err == nil {
		fmt.Printf("Thumbnail %s already exists, skipping download\n", jpgPath)
		return nil
	}

	fmt.Printf("Downloading Instagram thumbnail...\n")

	videoURL := "https://www.instagram.com/reels/" + videoID + "/"

	cmd := exec.Command(
		"yt-dlp",
		"--cookies", cookieFile,
		"-o", "./data/"+videoID,
		"--skip-download",
		"--write-thumbnail",
		"--convert-thumbnails", "jpg",
		videoURL,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error downloading Instagram thumbnail: %v", err)
	}

	return nil
}

func DownloadInstagramJSON(videoID, cookieFile string) error {
	jsonPath := fmt.Sprintf("./data/%s.json", videoID)

	if _, err := os.Stat(jsonPath); err == nil {
		fmt.Printf("JSON metadata %s already exists, skipping download\n", jsonPath)
		return nil
	}

	fmt.Printf("Downloading Instagram JSON metadata...\n")

	videoURL := "https://www.instagram.com/reels/" + videoID + "/"

	cmd := exec.Command(
		"yt-dlp",
		"--cookies", cookieFile,
		"-j",
		"--no-warnings",
		videoURL,
	)

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("error downloading Instagram JSON metadata: %v", err)
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