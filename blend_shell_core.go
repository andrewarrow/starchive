package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	fmt.Printf("  play [start_pos]     Play current blend (press any key to stop)\n")
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