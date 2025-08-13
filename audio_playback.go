package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"
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
	
	inputPath := getAudioFilename(id, audioType)

	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		fmt.Printf("Error: Input file %s does not exist\n", inputPath)
		os.Exit(1)
	}

	duration, err := getAudioDuration(inputPath)
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

	waitForKeyPress()
	cancel()
	fmt.Println("\nPlayback stopped.")
}

func handleBlendCommand() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: starchive blend <id1> <id2>")
		fmt.Println("Example: starchive blend OIduTH7NYA8 EbD7lfrsY2s")
		fmt.Println("Plays two tracks simultaneously with random pitch/tempo adjustments for 10 seconds")
		os.Exit(1)
	}

	id1 := os.Args[2]
	id2 := os.Args[3]

	rand.Seed(time.Now().UnixNano())

	audioTypes := []string{"I", "V"}
	type1 := audioTypes[rand.Intn(2)]
	type2 := audioTypes[rand.Intn(2)]

	pitchRange := []int{-8, -6, -4, -2, 0, 2, 4, 6, 8}
	tempoRange := []int{-20, -15, -10, -5, 0, 5, 10, 15, 20}

	pitch1 := pitchRange[rand.Intn(len(pitchRange))]
	tempo1 := tempoRange[rand.Intn(len(tempoRange))]
	pitch2 := pitchRange[rand.Intn(len(pitchRange))]
	tempo2 := tempoRange[rand.Intn(len(tempoRange))]

	inputPath1 := getAudioFilename(id1, type1)
	inputPath2 := getAudioFilename(id2, type2)

	if _, err := os.Stat(inputPath1); os.IsNotExist(err) {
		fmt.Printf("Error: Input file %s does not exist\n", inputPath1)
		os.Exit(1)
	}
	if _, err := os.Stat(inputPath2); os.IsNotExist(err) {
		fmt.Printf("Error: Input file %s does not exist\n", inputPath2)
		os.Exit(1)
	}

	duration1, err := getAudioDuration(inputPath1)
	if err != nil {
		fmt.Printf("Error getting audio duration for %s: %v\n", inputPath1, err)
		os.Exit(1)
	}
	duration2, err := getAudioDuration(inputPath2)
	if err != nil {
		fmt.Printf("Error getting audio duration for %s: %v\n", inputPath2, err)
		os.Exit(1)
	}

	startPosition1 := duration1 / 2
	startPosition2 := duration2 / 2

	trackDesc1 := "instrumental"
	if type1 == "V" {
		trackDesc1 = "vocal"
	}
	trackDesc2 := "instrumental"
	if type2 == "V" {
		trackDesc2 = "vocal"
	}

	fmt.Printf("Blending tracks:\n")
	fmt.Printf("  %s (%s): pitch %+d semitones, tempo %+d%%\n", id1, trackDesc1, pitch1, tempo1)
	fmt.Printf("  %s (%s): pitch %+d semitones, tempo %+d%%\n", id2, trackDesc2, pitch2, tempo2)
	fmt.Printf("Playing for 10 seconds...\n")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ffplayArgs1 := []string{
		"-ss", fmt.Sprintf("%.1f", startPosition1),
		"-t", "10",
		"-autoexit",
		"-nodisp",
		"-loglevel", "quiet",
	}
	
	if pitch1 != 0 || tempo1 != 0 {
		var filters []string
		if pitch1 != 0 {
			pitchMultiplier := 1.0 + (float64(pitch1) / 12.0)
			filters = append(filters, fmt.Sprintf("asetrate=44100*%.6f,aresample=44100", pitchMultiplier))
		}
		if tempo1 != 0 {
			tempoMultiplier := 1.0 + (float64(tempo1) / 100.0)
			if tempoMultiplier > 0.5 && tempoMultiplier <= 2.0 {
				filters = append(filters, fmt.Sprintf("atempo=%.6f", tempoMultiplier))
			}
		}
		if len(filters) > 0 {
			filter := strings.Join(filters, ",")
			ffplayArgs1 = append(ffplayArgs1, "-af", filter)
		}
	}
	
	ffplayArgs1 = append(ffplayArgs1, inputPath1)

	ffplayArgs2 := []string{
		"-ss", fmt.Sprintf("%.1f", startPosition2),
		"-t", "10",
		"-autoexit",
		"-nodisp",
		"-loglevel", "quiet",
	}
	
	if pitch2 != 0 || tempo2 != 0 {
		var filters []string
		if pitch2 != 0 {
			pitchMultiplier := 1.0 + (float64(pitch2) / 12.0)
			filters = append(filters, fmt.Sprintf("asetrate=44100*%.6f,aresample=44100", pitchMultiplier))
		}
		if tempo2 != 0 {
			tempoMultiplier := 1.0 + (float64(tempo2) / 100.0)
			if tempoMultiplier > 0.5 && tempoMultiplier <= 2.0 {
				filters = append(filters, fmt.Sprintf("atempo=%.6f", tempoMultiplier))
			}
		}
		if len(filters) > 0 {
			filter := strings.Join(filters, ",")
			ffplayArgs2 = append(ffplayArgs2, "-af", filter)
		}
	}
	
	ffplayArgs2 = append(ffplayArgs2, inputPath2)

	go func() {
		cmd1 := exec.CommandContext(ctx, "ffplay", ffplayArgs1...)
		cmd1.Run()
	}()

	go func() {
		cmd2 := exec.CommandContext(ctx, "ffplay", ffplayArgs2...)
		cmd2.Run()
	}()

	<-ctx.Done()
	fmt.Println("Blend playback completed.")
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

	waitForKeyPress()
	cancel()
	fmt.Println("Preview stopped.")
}