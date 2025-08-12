package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
)

var downloadQueue *DownloadQueue
var downloadVideos bool

func main() {
	// Simple subcommand dispatch: first arg is the command
	if len(os.Args) < 2 {
		fmt.Println("Usage: starchive <command> [args]\n\nCommands:\n  run     Start the server (default features)\n  ls      List files in ./data\n  vocal   Extract vocals from audio file using audio-separator\n  bpm     Analyze BPM and key of vocal and instrumental files\n  sync    Synchronize two audio files for mashups using rubberband\n  split   Split audio file by silence detection\n  rm      Remove all files with specified id from ./data\n  play    Play a wav file starting from the middle (press any key to stop)")
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "run":
		runCmd := flag.NewFlagSet("run", flag.ExitOnError)
		runCmd.BoolVar(&downloadVideos, "download-videos", true, "Download full videos; if false, only subtitles and thumbnails")
		// Parse flags after the subcommand
		if err := runCmd.Parse(os.Args[2:]); err != nil {
			fmt.Println("Error parsing flags:", err)
			os.Exit(2)
		}

		downloadQueue = NewDownloadQueue()
		setupRoutes(downloadQueue)

		fmt.Println("Server starting on port 3009...")
		if err := http.ListenAndServe(":3009", nil); err != nil {
			fmt.Printf("Server error: %v\n", err)
			os.Exit(1)
		}
	case "ls":
		handleLsCommand()
	case "vocal", "vocals":
		handleVocalCommand()
	case "bpm":
		handleBpmCommand()
	case "sync":
		handleSyncCommand()
	case "split":
		handleSplitCommand()
	case "rm":
		handleRmCommand()
	case "play":
		handlePlayCommand()
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		fmt.Println("Usage: starchive <command> [args]\n\nCommands:\n  run     Start the server (default features)\n  ls      List files in ./data\n  vocal   Extract vocals from audio file using audio-separator\n  bpm     Analyze BPM and key of vocal and instrumental files\n  sync    Synchronize two audio files for mashups using rubberband\n  split   Split audio file by silence detection\n  rm      Remove all files with specified id from ./data\n  play    Play a wav file starting from the middle (press any key to stop)")
		os.Exit(1)
	}
}