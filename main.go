package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	
	"starchive/audio"
	"starchive/handlers"
	"starchive/util"
	"starchive/web"
)

var downloadQueue *web.DownloadQueue
var downloadVideos bool

func updateYtDlp() {
	fmt.Println("Updating yt-dlp...")
	updateCmd := exec.Command("python3", "-m", "pip", "install", "-U", "yt-dlp")
	output, err := updateCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Warning: Failed to update yt-dlp: %v\n", err)
		fmt.Printf("Output: %s\n", output)
	} else {
		fmt.Println("yt-dlp updated successfully")
		fmt.Printf("Output: %s\n", output)
	}
}

func main() {
	// Simple subcommand dispatch: first arg is the command
	if len(os.Args) < 2 {
		fmt.Println("Usage: starchive <command> [args]\n\nCommands:\n  run         Start the server (default features)\n  ls          List files in ./data\n  dl          Download video with given ID\n  external    Import external audio file to data directory\n  vocal       Extract vocals from audio file using audio-separator\n  bpm         Analyze BPM and key of vocal and instrumental files\n  sync        Synchronize two audio files for mashups using rubberband\n  split       Split audio file by silence detection\n  rm          Remove all files with specified id from ./data\n  play        Play a wav file starting from the middle (press any key to stop)\n  demo        Create 30-second demo with +3 pitch shift from middle of track\n  blend       Interactive blend shell for mixing two tracks\n  blend-clear Clear blend metadata for track combinations\n  retry       Retry downloading specific components (vtt, json, thumbnail, video) for a given ID\n  ul          Upload mp4 to YouTube using the given ID\n  small       Create small optimized video from data/id.mp4")
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

		updateYtDlp()

		downloadQueue = web.NewDownloadQueue()
		web.SetupRoutes(downloadQueue)

		fmt.Println("Server starting on port 3009...")
		if err := http.ListenAndServe(":3009", nil); err != nil {
			fmt.Printf("Server error: %v\n", err)
			os.Exit(1)
		}
	case "ls":
		handlers.HandleLs()
	case "dl":
		updateYtDlp()
		handlers.HandleDl()
	case "external":
		handlers.HandleExternal()
	case "vocal", "vocals":
		handlers.HandleVocal()
	case "bpm":
		handlers.HandleBpm()
	case "sync":
		handlers.HandleSync()
	case "split":
		audio.HandleSplitCommand(os.Args[2:])
	case "rm":
		util.HandleRmCommand(os.Args[2:])
	case "play":
		audio.HandlePlayCommand(os.Args[2:])
	case "demo":
		audio.HandleDemoCommand(os.Args[2:])
	case "blend":
		handlers.HandleBlend()
	case "blend-clear":
		handlers.HandleBlendClear()
	case "retry":
		util.HandleRetryCommand(os.Args[2:])
	case "ul":
		handlers.HandleUl()
	case "small":
		handlers.HandleSmall()
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		fmt.Println("Usage: starchive <command> [args]\n\nCommands:\n  run         Start the server (default features)\n  ls          List files in ./data\n  dl          Download video with given ID\n  external    Import external audio file to data directory\n  vocal       Extract vocals from audio file using audio-separator\n  bpm         Analyze BPM and key of vocal and instrumental files\n  sync        Synchronize two audio files for mashups using rubberband\n  split       Split audio file by silence detection\n  rm          Remove all files with specified id from ./data\n  play        Play a wav file starting from the middle (press any key to stop)\n  demo        Create 30-second demo with +3 pitch shift from middle of track\n  blend       Interactive blend shell for mixing two tracks\n  blend-clear Clear blend metadata for track combinations\n  retry       Retry downloading specific components (vtt, json, thumbnail, video) for a given ID\n  ul          Upload mp4 to YouTube using the given ID\n  small       Create small optimized video from data/id.mp4")
		os.Exit(1)
	}
}