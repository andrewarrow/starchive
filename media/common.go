package media

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
)

func ParseVideoInput(input string) (string, string) {
	// YouTube URL patterns
	youtubePatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?:youtube\.com/watch\?v=|youtu\.be/)([a-zA-Z0-9_-]{11})`),
	}
	
	// Instagram URL patterns
	instagramPatterns := []*regexp.Regexp{
		regexp.MustCompile(`instagram\.com/(?:p|reels)/([a-zA-Z0-9_-]+)`),
	}
	
	// Check YouTube patterns
	for _, pattern := range youtubePatterns {
		if match := pattern.FindStringSubmatch(input); match != nil {
			return match[1], "youtube"
		}
	}
	
	// Check Instagram patterns
	for _, pattern := range instagramPatterns {
		if match := pattern.FindStringSubmatch(input); match != nil {
			return match[1], "instagram"
		}
	}
	
	// Check if it's a raw YouTube ID (11 characters, no dots)
	if IsYouTubeID(input) {
		return input, "youtube"
	}
	
	// Check if it's a raw Instagram ID (assume anything else that's alphanumeric)
	if IsInstagramID(input) {
		return input, "instagram"
	}
	
	return "", "unknown"
}

func DownloadVideo(videoID, platform string) (string, error) {
	switch platform {
	case "youtube":
		return DownloadYouTube(videoID)
	case "instagram":
		return DownloadInstagram(videoID)
	default:
		return "", fmt.Errorf("unsupported platform: %s", platform)
	}
}

func GetCookieFile(platform string) string {
	switch platform {
	case "youtube":
		// Try platform-specific first, fallback to generic
		if _, err := os.Stat("./cookies_youtube.txt"); err == nil {
			return "./cookies_youtube.txt"
		}
		return "./cookies.txt"
	case "instagram":
		return "./cookies_instagram.txt"
	default:
		return "./cookies.txt"
	}
}

func EnsureWav(videoID string) error {
	wavPath := fmt.Sprintf("./data/%s.wav", videoID)
	if _, err := os.Stat(wavPath); err == nil {
		fmt.Printf("WAV %s already exists, skipping creation\n", wavPath)
		return nil
	}

	cmd := exec.Command(
		"ffmpeg",
		"-y",
		"-i", fmt.Sprintf("./data/%s.mp4", videoID),
		"-vn",
		"-acodec", "pcm_s16le",
		"-ar", "44100",
		"-ac", "2",
		wavPath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg failed creating WAV: %v", err)
	}

	return nil
}