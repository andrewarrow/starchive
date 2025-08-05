package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"
)

func IsYouTubeID(input string) bool {
	return len(input) == 11 && !strings.Contains(input, ".")
}

func DownloadVideoWithFormat(log *slog.Logger, youtubeID string, format string) (string, error) {
	log = log.With("videoId", youtubeID, "format", format, "operation", "DownloadVideoWithFormat")
	outputFile := youtubeID

	// Check if file already exists as .mov or .mkv
	movFile := fmt.Sprintf("./data/%s.mov", youtubeID)
	mkvFile := fmt.Sprintf("./data/%s.mkv", youtubeID)

	if _, err := os.Stat(movFile); err == nil {
		fmt.Printf("Video %s.mov already exists, skipping download\n", youtubeID)
		return outputFile, nil
	}

	if _, err := os.Stat(mkvFile); err == nil {
		fmt.Printf("Video %s.mkv already exists, skipping download\n", youtubeID)
		return outputFile, nil
	}

	var ffmpegCommand string

	if format == "mkv" {
		log.Info("downloading and converting to HEVC 265 MKV")
		// HEVC 265 encoding for MKV
		ffmpegCommand = fmt.Sprintf("ffmpeg -i {} -c:v libx265 -preset medium -crf 23 -c:a copy ./data/%s.mkv && rm {}", youtubeID)
	} else {
		log.Info("downloading and converting to MOV...")
		// Default H.264 encoding for MOV
		ffmpegCommand = fmt.Sprintf("ffmpeg -i {} -c:v h264_videotoolbox -b:v 10000k ./data/%s.mov && rm {}", youtubeID)
	}

	DownloadSubtitles(youtubeID)

	// Use yt-dlp with ffmpeg post-processing
	cmd := exec.Command("yt-dlp",
		"-o", "./data/"+outputFile,
		"--restrict-filenames",
		"--exec", ffmpegCommand,
		youtubeID)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.With("outputFile", outputFile).With("err", err).Error("Download video failed")
		return "", fmt.Errorf("error downloading and converting YouTube video: %v", err)
	}

	log.With("outputFile", outputFile).Info("Downloaded video successfully")
	return outputFile, DownloadSubtitles(log, youtubeID)
}

func DownloadSubtitles(log *slog.Logger, youtubeID string) error {
	vttFile := "./data/" + youtubeID + ".en.vtt"
	log = log.With("youtubeID", youtubeID, "file", vttFile, "operation", "DownloadSubtitles")

	// Check if .en.vtt file already exists
	if _, err := os.Stat(vttFile); err == nil {
		log.Warn("Subtitles file already exists, skipping download")
		return nil
	}

	log.Debug("Downloading subtitles...")
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
				delay := min(time.Duration(1<<uint(attempt-1))*time.Second, 60*time.Second)
				log.Error(fmt.Sprintf("Subtitle download failed (attempt %d/50), retrying in %v", attempt, delay))
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
