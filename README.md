# Starchive

An advanced YouTube video archiving and audio processing system with intelligent music blending capabilities. Starchive combines automated video downloading with sophisticated audio manipulation tools for creating mashups and analyzing music.

## Features

### Core Archiving
- **Automatic YouTube Archiving**: Browser extension automatically downloads videos when visited
- **Multi-format Support**: Downloads video (MP4), audio (WAV), subtitles (VTT), thumbnails, and metadata (JSON)
- **Vocal Separation**: Extracts instrumental and vocal tracks using UVR (Ultimate Vocal Remover)
- **Audio Analysis**: BPM detection, key analysis, and beat detection

### Advanced Audio Processing
- **Interactive Blend Shell**: Real-time audio mixing interface for creating mashups
- **Intelligent Matching**: Automatic BPM and key matching with pitch/tempo adjustments
- **Sync Technology**: Precise audio synchronization using rubberband for seamless blending
- **Segment-based Editing**: Split tracks into segments with gap detection and smart placement
- **Conflict Detection**: Prevents vocal overlaps and ensures clean audio blending

### Command-line Interface
```
starchive <command> [args]

Commands:
  run         Start the web server for browser extension
  ls          List downloaded files with metadata
  dl          Download video by YouTube ID
  vocal       Extract vocal/instrumental tracks
  bpm         Analyze BPM and musical key
  sync        Synchronize audio files for mashups
  split       Split audio by silence detection
  blend       Interactive audio blending shell
  play        Play audio files with keyboard controls
  demo        Create 30-second preview clips
  rm          Remove files by video ID
  retry       Retry failed downloads
```

## System Architecture

### Backend (Go)
- **Web Server** (`web/`): HTTP API on port 3009 for browser extension
- **Media Processing** (`media/`): YouTube download and subtitle processing
- **Audio Engine** (`audio/`, `blend/`): Advanced audio processing and blending
- **Database** (`util/database.go`): SQLite storage for metadata and blend history
- **Command Handlers** (`command_handlers*.go`): CLI command implementations

### Browser Extension (Firefox)
- **Auto-detection**: Monitors YouTube visits and triggers downloads
- **Background Processing**: Queues downloads without interrupting browsing
- **Manual Interface**: Popup for direct video ID input

### Audio Processing Pipeline
1. **Download**: yt-dlp fetches video/audio/metadata
2. **Conversion**: ffmpeg extracts WAV audio
3. **Separation**: UVR splits vocals/instrumentals  
4. **Analysis**: Python scripts analyze BPM and musical key
5. **Blending**: Interactive shell for creating mashups

## Installation & Setup

### Prerequisites
- **Go** (1.19+)
- **Python 3.13** with virtual environment (included in `uvr/`)
- **yt-dlp**: Video downloading
- **ffmpeg**: Audio/video processing  
- **rubberband**: Audio time-stretching
- **Ultimate Vocal Remover**: AI vocal separation

### Quick Start
1. **Install dependencies**: Ensure yt-dlp, ffmpeg, and rubberband are in PATH
2. **Build**: `go build`
3. **Run server**: `./starchive run`
4. **Load extension**: Add `firefox/` directory to Firefox as temporary extension
5. **Start blending**: `./starchive blend` for interactive audio mixing

### Configuration
- **Data Storage**: Files saved to `./data/` and `./data2/` directories
- **Download Options**: Use `--download-videos=false` to skip video files
- **Cookies**: Place `cookies.txt` in root for private video access

## Advanced Usage

### Blend Shell Commands
The interactive blend shell supports sophisticated audio manipulation:
- **Track Loading**: Load two tracks for mixing
- **Parameter Control**: Adjust volume, pitch, tempo, and positioning
- **Smart Matching**: Automatic BPM/key alignment
- **Real-time Preview**: Live audio playback with modifications
- **Export Options**: Save blended results with detailed metadata

### Intelligent Features
- **Gap Analysis**: Finds optimal placement points in instrumental tracks
- **Beat Quantization**: Snaps audio segments to musical beats
- **Energy Matching**: Balances vocal and instrumental energy levels
- **Conflict Avoidance**: Prevents vocal overlaps in mashups

## File Organization

```
starchive/
├── main.go                 # CLI entry point and command dispatch
├── blend/                  # Audio blending engine
├── audio/                  # Core audio processing
├── media/                  # YouTube download and conversion
├── web/                    # HTTP server and browser extension API
├── util/                   # Database, file utilities, and helpers
├── firefox/                # Browser extension components
├── data/                   # Primary download storage
└── uvr/                    # Python virtual env for vocal separation
```

## Dependencies

- **yt-dlp**: YouTube video downloading
- **ffmpeg**: Audio/video conversion and processing
- **rubberband**: High-quality audio time-stretching
- **Ultimate Vocal Remover**: AI-powered vocal/instrumental separation
- **Go modules**: See `go.mod` for complete dependency list
- **Python packages**: Audio analysis and processing (isolated in `uvr/`)

## Disclaimer

This project is not associated with [https://www.starchive.io/](https://www.starchive.io/)
