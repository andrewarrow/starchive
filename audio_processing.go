package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	
	"starchive/audio"
)

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