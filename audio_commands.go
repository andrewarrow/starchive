package main

import (
	"context"
	"flag"
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

func handleSplitCommand() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: starchive split <filename>")
		fmt.Println("Example: starchive split beINamVRGy4_(Vocals)_UVR_MDXNET_Main_sync_to_qgaRVvAKoqQ.wav")
		os.Exit(1)
	}

	filename := os.Args[2]
	inputPath := fmt.Sprintf("./data/%s", filename)

	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		fmt.Printf("Error: Input file %s does not exist\n", inputPath)
		os.Exit(1)
	}

	id := strings.Split(filename, "_")[0]
	if id == "" {
		fmt.Printf("Error: Could not extract ID from filename %s\n", filename)
		os.Exit(1)
	}

	outputDir := fmt.Sprintf("./data/%s", id)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("Error creating directory %s: %v\n", outputDir, err)
		os.Exit(1)
	}

	fmt.Printf("Extracting ID: %s\n", id)
	fmt.Printf("Created directory: %s\n", outputDir)
	fmt.Printf("Splitting %s by silence detection...\n", filename)

	silenceCmd := exec.Command("ffmpeg", "-hide_banner", "-i", inputPath,
		"-af", "silencedetect=noise=-35dB:d=0.5", "-f", "null", "-")

	silenceOutput, err := silenceCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error detecting silence: %v\n", err)
		fmt.Printf("Output: %s\n", string(silenceOutput))
		os.Exit(1)
	}

	sedCmd := exec.Command("sed", "-n", "s/.*silence_end: \\([0-9.]*\\).*/\\1/p")
	sedCmd.Stdin = strings.NewReader(string(silenceOutput))
	sedOutput, err := sedCmd.Output()
	if err != nil {
		fmt.Printf("Error extracting timestamps: %v\n", err)
		os.Exit(1)
	}

	timestamps := strings.TrimSpace(string(sedOutput))
	timestamps = strings.ReplaceAll(timestamps, "\n", ",")

	if timestamps == "" {
		fmt.Printf("Warning: No silence detected. Creating single output file.\n")
		outputPath := fmt.Sprintf("%s/part_001.wav", outputDir)
		copyCmd := exec.Command("cp", inputPath, outputPath)
		if err := copyCmd.Run(); err != nil {
			fmt.Printf("Error copying file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Created: %s\n", outputPath)
		return
	}

	fmt.Printf("Detected silence timestamps: %s\n", timestamps)

	outputPattern := fmt.Sprintf("%s/part_%%03d.wav", outputDir)
	splitCmd := exec.Command("ffmpeg", "-hide_banner", "-i", inputPath,
		"-c", "copy", "-f", "segment", "-segment_times", timestamps, outputPattern)

	fmt.Printf("Running: %s\n", splitCmd.String())

	splitCmd.Stdout = os.Stdout
	splitCmd.Stderr = os.Stderr

	err = splitCmd.Run()
	if err != nil {
		fmt.Printf("Error splitting file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully split %s into parts in directory %s\n", filename, outputDir)
}

func handleDemoCommand() {
	demoCmd := flag.NewFlagSet("demo", flag.ExitOnError)
	pitchShift := demoCmd.Int("pitch", 3, "Pitch shift in semitones (can be negative)")
	tempoChange := demoCmd.Float64("tempo", 0.0, "Tempo change as percentage (e.g., 30 for 30% faster, -20 for 20% slower)")
	
	demoCmd.Usage = func() {
		fmt.Println("Usage: starchive demo [options] <id> <I|V>")
		fmt.Println("Creates a 30-second demo from the middle of the track with pitch/tempo adjustments")
		fmt.Println("\nOptions:")
		demoCmd.PrintDefaults()
		fmt.Println("\nExamples:")
		fmt.Println("  starchive demo NdYWuo9OFAw I                    # +3 semitones, no tempo change")
		fmt.Println("  starchive demo -pitch -2 NdYWuo9OFAw V          # -2 semitones, no tempo change")
		fmt.Println("  starchive demo -tempo 25 NdYWuo9OFAw I          # +3 semitones, 25% faster")
		fmt.Println("  starchive demo -pitch 0 -tempo -15 NdYWuo9OFAw V # no pitch change, 15% slower")
	}

	if len(os.Args) < 4 {
		demoCmd.Usage()
		os.Exit(1)
	}

	if err := demoCmd.Parse(os.Args[2:]); err != nil {
		fmt.Println("Error parsing flags:", err)
		os.Exit(1)
	}

	args := demoCmd.Args()
	if len(args) < 2 {
		demoCmd.Usage()
		os.Exit(1)
	}

	id := args[0]
	audioType := args[1]

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

	startPosition := (duration - 30) / 2
	if startPosition < 0 {
		startPosition = 0
	}

	tempClipPath := fmt.Sprintf("/tmp/%s_%s_30sec.wav", id, audioType)
	demoPath := fmt.Sprintf("/tmp/%s_%s_demo.wav", id, audioType)

	trackDesc := "main track"
	switch audioType {
	case "I", "instrumental", "instrumentals":
		trackDesc = "instrumental track"
	case "V", "vocal", "vocals":
		trackDesc = "vocal track"
	}

	fmt.Printf("Creating demo for %s (%s)\n", id, trackDesc)
	if *pitchShift != 0 {
		if *pitchShift > 0 {
			fmt.Printf("Pitch shift: +%d semitones\n", *pitchShift)
		} else {
			fmt.Printf("Pitch shift: %d semitones\n", *pitchShift)
		}
	}
	if *tempoChange != 0 {
		if *tempoChange > 0 {
			fmt.Printf("Tempo change: %.1f%% faster\n", *tempoChange)
		} else {
			fmt.Printf("Tempo change: %.1f%% slower\n", -*tempoChange)
		}
	}
	fmt.Printf("Extracting 30 seconds from position %.1fs...\n", startPosition)

	extractCmd := exec.Command("ffmpeg", "-y",
		"-ss", fmt.Sprintf("%.1f", startPosition),
		"-t", "30",
		"-i", inputPath,
		"-c", "copy",
		tempClipPath)

	extractCmd.Stdout = os.Stdout
	extractCmd.Stderr = os.Stderr

	err = extractCmd.Run()
	if err != nil {
		fmt.Printf("Error extracting 30-second clip: %v\n", err)
		os.Exit(1)
	}

	defer os.Remove(tempClipPath)
	defer os.Remove(demoPath)

	rubberbandArgs := []string{"--fine", "--formant"}
	
	if *pitchShift != 0 {
		rubberbandArgs = append(rubberbandArgs, "--pitch", fmt.Sprintf("%d", *pitchShift))
	}
	
	if *tempoChange != 0 {
		speedMultiplier := 1.0 + (*tempoChange / 100.0)
		timeRatio := 1.0 / speedMultiplier
		rubberbandArgs = append(rubberbandArgs, "--time", fmt.Sprintf("%.6f", timeRatio))
	}
	
	rubberbandArgs = append(rubberbandArgs, tempClipPath, demoPath)

	fmt.Printf("Applying audio processing with rubberband...\n")

	pitchCmd := exec.Command("rubberband", rubberbandArgs...)

	pitchCmd.Stdout = os.Stdout
	pitchCmd.Stderr = os.Stderr

	err = pitchCmd.Run()
	if err != nil {
		fmt.Printf("Error applying audio processing: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Playing processed demo (press any key to stop)...\n")

	playTempFile(demoPath)

	fmt.Printf("Demo preview completed. All temp files cleaned up.\n")
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