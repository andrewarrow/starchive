package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	
	"starchive/audio"
)

func handlePlayCommand() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: starchive play <id> [I|V]")
		fmt.Println("Example: starchive play NdYWuo9OFAw")
		fmt.Println("         starchive play NdYWuo9OFAw I  (instrumental)")
		fmt.Println("         starchive play NdYWuo9OFAw V  (vocals)")
		fmt.Println("Plays the wav file starting from the middle. Press any key to stop.")
		os.Exit(1)
	}

	id := os.Args[2]
	
	audioType := ""
	if len(os.Args) > 3 {
		audioType = os.Args[3]
	}
	
	inputPath := audio.GetAudioFilename(id, audioType)

	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		fmt.Printf("Error: Input file %s does not exist\n", inputPath)
		os.Exit(1)
	}

	duration, err := audio.GetAudioDuration(inputPath)
	if err != nil {
		fmt.Printf("Error getting audio duration: %v\n", err)
		os.Exit(1)
	}

	startPosition := duration / 2
	
	trackDesc := "main track"
	switch audioType {
	case "I", "instrumental", "instrumentals":
		trackDesc = "instrumental track"
	case "V", "vocal", "vocals":
		trackDesc = "vocal track"
	}
	
	fmt.Printf("Playing %s (%s) from position %.1fs (middle of %.1fs total)\n", id, trackDesc, startPosition, duration)
	fmt.Println("Press any key to stop playback...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		cmd := exec.CommandContext(ctx, "ffplay", 
			"-ss", fmt.Sprintf("%.1f", startPosition),
			"-autoexit",
			"-nodisp",
			"-loglevel", "quiet",
			inputPath)
		
		cmd.Run()
	}()

	audio.WaitForKeyPress()
	cancel()
	fmt.Println("\nPlayback stopped.")
}

func playTempFile(filePath string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		cmd := exec.CommandContext(ctx, "ffplay", 
			"-autoexit",
			"-nodisp",
			"-loglevel", "quiet",
			filePath)
		
		cmd.Run()
	}()

	audio.WaitForKeyPress()
	cancel()
	fmt.Println("Preview stopped.")
}