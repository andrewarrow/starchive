# Starchive

A browser extension and local backend system that automatically archives YouTube videos when visited. The system consists of a Firefox extension that detects YouTube video pages and a Go backend that downloads the videos using yt-dlp.

## Components

### Backend (Go)
- **HTTP Server** (`main.go`): Runs on port 3009 with two endpoints:
  - `/`: Basic health check endpoint
  - `/youtube`: POST endpoint that accepts video IDs and triggers downloads
- **Video Downloader** (`youtube.go`): Uses yt-dlp and ffmpeg to download YouTube videos as mp4 (playable in quicktime) 
- **Subtitle Support**: Downloads English subtitles in VTT format (currently disabled in retry loop)

### Browser Extension (Firefox)
- **Manifest** (`manifest.json`): Defines extension permissions and structure
- **Content Script** (`content.js`): Automatically detects YouTube video pages and extracts video IDs from URLs
- **Background Script** (`background.js`): Handles communication between content script and backend API
- **Popup Interface** (`popup.html/js`): Provides a simple UI for manual data fetching

## How It Works

1. When you visit a YouTube video page, the content script automatically detects the video ID from the URL
2. The video ID is sent to the background script, which makes a POST request to `http://localhost:3009/youtube`
3. The Go backend receives the video ID and uses yt-dlp to download the video
4. Videos are saved to the `./data/` directory and an audio-only `id.wav` file is extracted via ffmpeg
5. The system also supports subtitle downloading (though currently limited to 1 attempt)

## Setup

1. Start the Go backend: `go run .`
2. Load the Firefox extension from the `firefox/` directory
3. Visit any YouTube video page to automatically trigger archiving

### Flags

- `--download-videos` (default `true`): When set to `false`, the backend skips downloading full videos and only fetches subtitles and thumbnails.

## Dependencies

- **yt-dlp**: For downloading YouTube videos
- **ffmpeg**: For video conversion and audio extraction (creates `id.wav`)
- **Go**: Backend server runtime

## Other Starchive

Not associated with [https://www.starchive.io/](https://www.starchive.io/)
