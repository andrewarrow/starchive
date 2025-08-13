package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/chzyer/readline"
)

func newBlendShell(id1, id2 string) *BlendShell {
	db, err := initDatabase()
	if err != nil {
		fmt.Printf("Error initializing database: %v\n", err)
		os.Exit(1)
	}

	metadata1, found1 := getCachedMetadata(db, id1)
	metadata2, found2 := getCachedMetadata(db, id2)

	if !found1 {
		fmt.Printf("Warning: No metadata found for %s\n", id1)
	}
	if !found2 {
		fmt.Printf("Warning: No metadata found for %s\n", id2)
	}

	type1, type2 := detectTrackTypes(id1, id2)
	
	shell := &BlendShell{
		id1:       id1,
		id2:       id2,
		metadata1: metadata1,
		metadata2: metadata2,
		type1:     type1,
		type2:     type2,
		pitch1:    0,
		pitch2:    0,
		tempo1:    0.0,
		tempo2:    0.0,
		volume1:   100.0,
		volume2:   100.0,
		window1:   0.0,
		window2:   0.0,
		db:        db,
		segments1: []VocalSegment{},
		segments2: []VocalSegment{},
		segmentsDir1: fmt.Sprintf("./data/%s", id1),
		segmentsDir2: fmt.Sprintf("./data/%s", id2),
	}

	shell.inputPath1 = getAudioFilename(id1, type1)
	shell.inputPath2 = getAudioFilename(id2, type2)

	if _, err := os.Stat(shell.inputPath1); os.IsNotExist(err) {
		fmt.Printf("Error: Input file %s does not exist\n", shell.inputPath1)
		os.Exit(1)
	}
	if _, err := os.Stat(shell.inputPath2); os.IsNotExist(err) {
		fmt.Printf("Error: Input file %s does not exist\n", shell.inputPath2)
		os.Exit(1)
	}

	shell.duration1, _ = getAudioDuration(shell.inputPath1)
	shell.duration2, _ = getAudioDuration(shell.inputPath2)

	return shell
}

func (bs *BlendShell) run() {
	defer bs.db.Close()
	
	fmt.Printf("=== Blend Shell ===\n")
	fmt.Printf("Track 1: %s (%s)\n", bs.id1, bs.getTrackTypeDesc(bs.type1))
	fmt.Printf("Track 2: %s (%s)\n", bs.id2, bs.getTrackTypeDesc(bs.type2))
	
	if bs.metadata1 != nil && bs.metadata1.BPM != nil && bs.metadata1.Key != nil {
		fmt.Printf("  %.1f BPM, %s\n", *bs.metadata1.BPM, *bs.metadata1.Key)
	}
	if bs.metadata2 != nil && bs.metadata2.BPM != nil && bs.metadata2.Key != nil {
		fmt.Printf("  %.1f BPM, %s\n", *bs.metadata2.BPM, *bs.metadata2.Key)
	}
	
	fmt.Printf("\nCommands:\n")
	fmt.Printf("  play                 Play current blend\n")
	fmt.Printf("  pitch1 <n>           Adjust track 1 pitch (semitones)\n")
	fmt.Printf("  pitch2 <n>           Adjust track 2 pitch (semitones)\n")
	fmt.Printf("  tempo1 <n>           Adjust track 1 tempo (%%)\n")
	fmt.Printf("  tempo2 <n>           Adjust track 2 tempo (%%)\n")
	fmt.Printf("  volume1 <n>          Set track 1 volume (0-200)\n")
	fmt.Printf("  volume2 <n>          Set track 2 volume (0-200)\n")
	fmt.Printf("  window <n1> <n2>     Set track start offsets from middle (seconds)\n")
	fmt.Printf("  match bpm1to2        Match track 1 BPM to track 2\n")
	fmt.Printf("  match bpm2to1        Match track 2 BPM to track 1\n")
	fmt.Printf("  match key1to2        Match track 1 key to track 2\n")
	fmt.Printf("  match key2to1        Match track 2 key to track 1\n")
	fmt.Printf("  invert               Reset and intelligently match tracks\n")
	fmt.Printf("  type1 <vocal|instrumental> Set track 1 type\n")
	fmt.Printf("  type2 <vocal|instrumental> Set track 2 type\n")
	fmt.Printf("  split <1|2>          Split track into vocal segments\n")
	fmt.Printf("  segments [1|2]       List vocal segments\n")
	fmt.Printf("  place <track:seg> at <time> Place segment at specific time\n")
	fmt.Printf("  shift <track:seg> <+/-time> Adjust segment timing\n")
	fmt.Printf("  toggle <track:seg>   Enable/disable segment\n")
	fmt.Printf("  preview <track:seg>  Preview single segment\n")
	fmt.Printf("  random <track>       Randomly place all segments\n")
	fmt.Printf("  reset                Reset all adjustments\n")
	fmt.Printf("  status               Show current settings\n")
	fmt.Printf("  help                 Show this help\n")
	fmt.Printf("  exit                 Exit blend shell\n")
	fmt.Printf("\n")

	bs.showStatus()
	
	// Set up history file
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Warning: Could not get home directory: %v\n", err)
		homeDir = "."
	}
	historyFile := filepath.Join(homeDir, ".blend_history")
	
	config := &readline.Config{
		Prompt:      "blend> ",
		HistoryFile: historyFile,
		AutoComplete: bs.completer(),
	}
	
	rl, err := readline.NewEx(config)
	if err != nil {
		fmt.Printf("Error initializing readline: %v\n", err)
		return
	}
	defer rl.Close()
	
	for {
		input, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				fmt.Println("\nExiting blend shell...")
				break
			}
			fmt.Printf("Error reading input: %v\n", err)
			break
		}
		
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		
		if !bs.handleCommand(input) {
			break
		}
	}
}

func (bs *BlendShell) completer() readline.AutoCompleter {
	return readline.NewPrefixCompleter(
		readline.PcItem("play"),
		readline.PcItem("pitch1"),
		readline.PcItem("pitch2"), 
		readline.PcItem("tempo1"),
		readline.PcItem("tempo2"),
		readline.PcItem("volume1"),
		readline.PcItem("volume2"),
		readline.PcItem("window"),
		readline.PcItem("match",
			readline.PcItem("bpm1to2"),
			readline.PcItem("bpm2to1"),
			readline.PcItem("key1to2"),
			readline.PcItem("key2to1"),
		),
		readline.PcItem("type1",
			readline.PcItem("vocal"),
			readline.PcItem("instrumental"),
		),
		readline.PcItem("type2",
			readline.PcItem("vocal"),
			readline.PcItem("instrumental"),
		),
		readline.PcItem("split",
			readline.PcItem("1"),
			readline.PcItem("2"),
		),
		readline.PcItem("segments",
			readline.PcItem("1"),
			readline.PcItem("2"),
		),
		readline.PcItem("place"),
		readline.PcItem("shift"),
		readline.PcItem("toggle"),
		readline.PcItem("preview"),
		readline.PcItem("random",
			readline.PcItem("1"),
			readline.PcItem("2"),
		),
		readline.PcItem("invert"),
		readline.PcItem("reset"),
		readline.PcItem("status"),
		readline.PcItem("help"),
		readline.PcItem("exit"),
	)
}

func (bs *BlendShell) showStatus() {
	fmt.Printf("--- Current Settings ---\n")
	fmt.Printf("Track 1 (%s %s): pitch %+d, tempo %+.1f%%, volume %.0f%%, window %+.1fs\n", 
		bs.id1, bs.getTrackTypeDesc(bs.type1), bs.pitch1, bs.tempo1, bs.volume1, bs.window1)
	fmt.Printf("Track 2 (%s %s): pitch %+d, tempo %+.1f%%, volume %.0f%%, window %+.1fs\n", 
		bs.id2, bs.getTrackTypeDesc(bs.type2), bs.pitch2, bs.tempo2, bs.volume2, bs.window2)
		
	if bs.metadata1 != nil && bs.metadata1.BPM != nil && bs.metadata1.Key != nil {
		effectiveBPM1 := calculateEffectiveBPM(*bs.metadata1.BPM, bs.tempo1)
		effectiveKey1 := calculateEffectiveKey(*bs.metadata1.Key, bs.pitch1)
		fmt.Printf("  Effective: %.1f BPM, %s (was %.1f BPM, %s)\n", 
			effectiveBPM1, effectiveKey1, *bs.metadata1.BPM, *bs.metadata1.Key)
	}
	if bs.metadata2 != nil && bs.metadata2.BPM != nil && bs.metadata2.Key != nil {
		effectiveBPM2 := calculateEffectiveBPM(*bs.metadata2.BPM, bs.tempo2)
		effectiveKey2 := calculateEffectiveKey(*bs.metadata2.Key, bs.pitch2)
		fmt.Printf("  Effective: %.1f BPM, %s (was %.1f BPM, %s)\n", 
			effectiveBPM2, effectiveKey2, *bs.metadata2.BPM, *bs.metadata2.Key)
	}
	
	// Show segment information
	activeSegments1 := 0
	activeSegments2 := 0
	for _, seg := range bs.segments1 {
		if seg.Active { activeSegments1++ }
	}
	for _, seg := range bs.segments2 {
		if seg.Active { activeSegments2++ }
	}
	
	if len(bs.segments1) > 0 || len(bs.segments2) > 0 {
		fmt.Printf("Segments: Track 1: %d/%d active, Track 2: %d/%d active\n", 
			activeSegments1, len(bs.segments1), activeSegments2, len(bs.segments2))
	}
	fmt.Printf("\n")
}

func (bs *BlendShell) showHelp() {
	fmt.Printf("--- Blend Shell Commands ---\n")
	fmt.Printf("Playback:\n")
	fmt.Printf("  play                Play current blend for 10 seconds\n")
	fmt.Printf("Adjustments:\n")
	fmt.Printf("  pitch1 <n>          Adjust track 1 pitch (-12 to +12 semitones)\n")
	fmt.Printf("  pitch2 <n>          Adjust track 2 pitch (-12 to +12 semitones)\n")
	fmt.Printf("  tempo1 <n>          Adjust track 1 tempo (-50 to +100%%)\n")
	fmt.Printf("  tempo2 <n>          Adjust track 2 tempo (-50 to +100%%)\n")
	fmt.Printf("  volume1 <n>         Set track 1 volume (0 to 200)\n")
	fmt.Printf("  volume2 <n>         Set track 2 volume (0 to 200)\n")
	fmt.Printf("  window <n1> <n2>    Set start offsets from middle (seconds)\n")
	fmt.Printf("Matching:\n")
	fmt.Printf("  match bpm1to2       Match track 1 BPM to track 2\n")
	fmt.Printf("  match bpm2to1       Match track 2 BPM to track 1\n")
	fmt.Printf("  match key1to2       Match track 1 key to track 2\n")
	fmt.Printf("  match key2to1       Match track 2 key to track 1\n")
	fmt.Printf("  invert              Reset and intelligently match tracks\n")
	fmt.Printf("Track Types:\n")
	fmt.Printf("  type1 <type>        Set track 1 type (vocal/instrumental)\n")
	fmt.Printf("  type2 <type>        Set track 2 type (vocal/instrumental)\n")
	fmt.Printf("Vocal Segments:\n")
	fmt.Printf("  split <1|2>         Split vocal track into segments by silence\n")
	fmt.Printf("  segments [1|2]      List available segments\n")
	fmt.Printf("  place <track:seg> at <time> Place segment (e.g. '1:3 at 45.2')\n")
	fmt.Printf("  shift <track:seg> <+/-time> Adjust segment timing (e.g. '1:3 +2.5')\n")
	fmt.Printf("  toggle <track:seg>  Enable/disable segment (e.g. '1:3')\n")
	fmt.Printf("  preview <track:seg> Preview individual segment (e.g. '1:3')\n")
	fmt.Printf("  random <1|2>        Randomly place all segments from track\n")
	fmt.Printf("Utility:\n")
	fmt.Printf("  reset               Reset all adjustments to zero\n")
	fmt.Printf("  status              Show current settings\n")
	fmt.Printf("  exit                Exit blend shell\n")
	fmt.Printf("\n")
}

func (bs *BlendShell) resetAdjustments() {
	bs.pitch1 = 0
	bs.pitch2 = 0
	bs.tempo1 = 0.0
	bs.tempo2 = 0.0
	bs.volume1 = 100.0
	bs.volume2 = 100.0
	bs.window1 = 0.0
	bs.window2 = 0.0
	fmt.Printf("All adjustments reset to defaults\n")
}

func (bs *BlendShell) getTrackTypeDesc(trackType string) string {
	if trackType == "V" {
		return "vocal"
	}
	return "instrumental"
}

func (bs *BlendShell) playBlend() {
	startPosition1 := (bs.duration1 / 2) + bs.window1
	startPosition2 := (bs.duration2 / 2) + bs.window2
	
	if startPosition1 < 0 {
		startPosition1 = 0
	}
	if startPosition2 < 0 {
		startPosition2 = 0
	}
	
	if startPosition1 >= bs.duration1 {
		startPosition1 = bs.duration1 - 1
	}
	if startPosition2 >= bs.duration2 {
		startPosition2 = bs.duration2 - 1
	}

	// Calculate maximum available play duration for both tracks
	remainingDuration1 := bs.duration1 - startPosition1
	remainingDuration2 := bs.duration2 - startPosition2
	playDuration := 10.0 // Default 10 seconds
	
	// Use the smaller of the two remaining durations, but cap at 10 seconds
	maxAvailableDuration := remainingDuration1
	if remainingDuration2 < maxAvailableDuration {
		maxAvailableDuration = remainingDuration2
	}
	
	if maxAvailableDuration < playDuration {
		playDuration = maxAvailableDuration
	}

	fmt.Printf("Playing blend for %.1f seconds...\n", playDuration)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(playDuration*1000)*time.Millisecond)
	defer cancel()

	ffplayArgs1 := bs.buildFFplayArgs(bs.inputPath1, startPosition1, bs.pitch1, bs.tempo1, bs.volume1, playDuration)
	ffplayArgs2 := bs.buildFFplayArgs(bs.inputPath2, startPosition2, bs.pitch2, bs.tempo2, bs.volume2, playDuration)

	go func() {
		cmd1 := exec.CommandContext(ctx, "ffplay", ffplayArgs1...)
		cmd1.Run()
	}()

	go func() {
		cmd2 := exec.CommandContext(ctx, "ffplay", ffplayArgs2...)
		cmd2.Run()
	}()

	<-ctx.Done()
	fmt.Printf("Playback completed.\n\n")
}

func (bs *BlendShell) buildFFplayArgs(inputPath string, startPos float64, pitch int, tempo float64, volume float64, playDuration float64) []string {
	args := []string{
		"-ss", fmt.Sprintf("%.1f", startPos),
		"-t", fmt.Sprintf("%.1f", playDuration),
		"-autoexit",
		"-nodisp",
		"-loglevel", "quiet",
	}

	if pitch != 0 || tempo != 0 || volume != 100 {
		var filters []string
		
		if tempo != 0 {
			tempoMultiplier := 1.0 + (tempo / 100.0)
			if tempoMultiplier > 0.5 && tempoMultiplier <= 2.0 {
				filters = append(filters, fmt.Sprintf("atempo=%.6f", tempoMultiplier))
			}
		}
		
		if pitch != 0 {
			pitchSemitones := float64(pitch)
			filters = append(filters, fmt.Sprintf("asetrate=44100*%.6f,aresample=44100,atempo=%.6f", 
				math.Pow(2, pitchSemitones/12.0), 1.0/math.Pow(2, pitchSemitones/12.0)))
		}
		
		if volume != 100 {
			volumeMultiplier := volume / 100.0
			filters = append(filters, fmt.Sprintf("volume=%.6f", volumeMultiplier))
		}
		
		if len(filters) > 0 {
			filter := strings.Join(filters, ",")
			args = append(args, "-af", filter)
		}
	}

	args = append(args, inputPath)
	return args
}