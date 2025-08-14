package audio

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// HandleSplitCommand splits audio files by silence detection
func HandleSplitCommand(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: starchive split <filename>")
		fmt.Println("Example: starchive split beINamVRGy4_(Vocals)_UVR_MDXNET_Main_sync_to_qgaRVvAKoqQ.wav")
		os.Exit(1)
	}

	filename := args[0]
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
		fmt.Printf("No silence detected, not splitting.\n")
		os.Exit(0)
	}

	fmt.Printf("Detected silence at timestamps: %s\n", timestamps)

	outputPattern := fmt.Sprintf("%s/part_%%03d.wav", outputDir)
	splitCmd := exec.Command("ffmpeg", "-hide_banner", "-y", "-i", inputPath,
		"-c", "copy", "-f", "segment", "-segment_times", timestamps, outputPattern)

	err = splitCmd.Run()
	if err != nil {
		fmt.Printf("Error splitting file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("File split successfully into %s/\n", outputDir)
}

// HandleDemoCommand creates demo files with pitch adjustment
func HandleDemoCommand(args []string) {
	demoCmd := flag.NewFlagSet("demo", flag.ExitOnError)
	
	err := demoCmd.Parse(args)
	if err != nil {
		fmt.Println("Error parsing demo command:", err)
		os.Exit(2)
	}

	demoArgs := demoCmd.Args()
	if len(demoArgs) < 2 {
		fmt.Println("Usage: starchive demo <id> <audio_type>")
		fmt.Println("Example: starchive demo NdYWuo9OFAw I")
		fmt.Println("         starchive demo NdYWuo9OFAw V")
		fmt.Println("Creates a 30-second demo with +3 pitch shift from middle of track.")
		os.Exit(1)
	}

	id := demoArgs[0]
	audioType := demoArgs[1]

	inputPath := GetAudioFilename(id, audioType)

	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		fmt.Printf("Error: Input file %s does not exist\n", inputPath)
		os.Exit(1)
	}

	duration, err := GetAudioDuration(inputPath)
	if err != nil {
		fmt.Printf("Error getting audio duration: %v\n", err)
		os.Exit(1)
	}

	start := duration/2 - 15
	if start < 0 {
		start = 0
	}

	outputFile := fmt.Sprintf("./data/%s_%s_demo.wav", id, audioType)

	ffmpegCmd := exec.Command("ffmpeg", "-y", "-ss", fmt.Sprintf("%.1f", start),
		"-i", inputPath, "-t", "30",
		"-af", "asetrate=44100*1.189207,aresample=44100,atempo=0.840896",
		outputFile)

	err = ffmpegCmd.Run()
	if err != nil {
		fmt.Printf("Error creating demo: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Demo created: %s\n", outputFile)
}

// HandlePlayCommand plays audio files
func HandlePlayCommand(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: starchive play <id> [I|V]")
		fmt.Println("Example: starchive play NdYWuo9OFAw")
		fmt.Println("         starchive play NdYWuo9OFAw I  (instrumental)")
		fmt.Println("         starchive play NdYWuo9OFAw V  (vocals)")
		fmt.Println("Plays the wav file starting from the middle. Press any key to stop.")
		os.Exit(1)
	}

	id := args[0]
	
	audioType := ""
	if len(args) > 1 {
		audioType = args[1]
	}
	
	inputPath := GetAudioFilename(id, audioType)

	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		fmt.Printf("Error: Input file %s does not exist\n", inputPath)
		os.Exit(1)
	}

	duration, err := GetAudioDuration(inputPath)
	if err != nil {
		fmt.Printf("Error getting audio duration: %v\n", err)
		os.Exit(1)
	}

	start := duration / 2
	fmt.Printf("Playing %s from %.1fs (middle). Press any key to stop...\n", inputPath, start)

	ffplayCmd := exec.Command("ffplay", "-ss", fmt.Sprintf("%.1f", start),
		"-autoexit", "-nodisp", "-loglevel", "quiet", inputPath)

	go func() {
		ffplayCmd.Run()
	}()

	WaitForKeyPress()
	
	if ffplayCmd.Process != nil {
		ffplayCmd.Process.Kill()
	}
	
	fmt.Println("\nPlayback stopped.")

	WaitForKeyPress()
	
	if ffplayCmd.Process != nil {
		ffplayCmd.Process.Kill()
	}
	
	fmt.Println("\nPlayback stopped.")
}