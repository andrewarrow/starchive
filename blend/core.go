package blend

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chzyer/readline"
	"starchive/audio"
	"starchive/util"
)

// NewShell creates a new blend shell for mixing two tracks
func NewShell(id1, id2 string, db *util.Database) *Shell {
	metadata1, found1 := db.GetCachedMetadata(id1)
	metadata2, found2 := db.GetCachedMetadata(id2)

	if !found1 {
		fmt.Printf("Warning: No metadata found for %s\n", id1)
	}
	if !found2 {
		fmt.Printf("Warning: No metadata found for %s\n", id2)
	}

	type1, type2 := audio.DetectTrackTypes(id1, id2)
	
	shell := &Shell{
		ID1:       id1,
		ID2:       id2,
		Metadata1: metadata1,
		Metadata2: metadata2,
		Type1:     type1,
		Type2:     type2,
		Pitch1:    0,
		Pitch2:    0,
		Tempo1:    0.0,
		Tempo2:    0.0,
		Volume1:   100.0,
		Volume2:   100.0,
		Window1:   0.0,
		Window2:   0.0,
		DB:        db,
		Segments1: []VocalSegment{},
		Segments2: []VocalSegment{},
		SegmentsDir1: fmt.Sprintf("./data/%s", id1),
		SegmentsDir2: fmt.Sprintf("./data/%s", id2),
	}

	shell.InputPath1 = audio.GetAudioFilename(id1, type1)
	shell.InputPath2 = audio.GetAudioFilename(id2, type2)

	if _, err := os.Stat(shell.InputPath1); os.IsNotExist(err) {
		fmt.Printf("Error: Input file %s does not exist\n", shell.InputPath1)
		os.Exit(1)
	}
	if _, err := os.Stat(shell.InputPath2); os.IsNotExist(err) {
		fmt.Printf("Error: Input file %s does not exist\n", shell.InputPath2)
		os.Exit(1)
	}

	shell.Duration1, _ = audio.GetAudioDuration(shell.InputPath1)
	shell.Duration2, _ = audio.GetAudioDuration(shell.InputPath2)

	return shell
}

// Run starts the interactive blend shell
func (bs *Shell) Run() {
	fmt.Printf("=== Blend Shell ===\n")
	fmt.Printf("Track 1: %s (%s)\n", bs.ID1, bs.getTrackTypeDesc(bs.Type1))
	fmt.Printf("Track 2: %s (%s)\n", bs.ID2, bs.getTrackTypeDesc(bs.Type2))
	
	if bs.Metadata1 != nil && bs.Metadata1.BPM != nil && bs.Metadata1.Key != nil {
		fmt.Printf("  %.1f BPM, %s\n", *bs.Metadata1.BPM, *bs.Metadata1.Key)
	}
	if bs.Metadata2 != nil && bs.Metadata2.BPM != nil && bs.Metadata2.Key != nil {
		fmt.Printf("  %.1f BPM, %s\n", *bs.Metadata2.BPM, *bs.Metadata2.Key)
	}
	
	bs.printCommands()
	bs.ShowStatus()
	
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
		AutoComplete: bs.Completer(),
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
		
		if !bs.HandleCommand(input) {
			break
		}
	}
}

func (bs *Shell) printCommands() {
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
}

// ResetAdjustments resets all blend adjustments to default values
func (bs *Shell) ResetAdjustments() {
	bs.Pitch1 = 0
	bs.Pitch2 = 0
	bs.Tempo1 = 0.0
	bs.Tempo2 = 0.0
	bs.Volume1 = 100.0
	bs.Volume2 = 100.0
	bs.Window1 = 0.0
	bs.Window2 = 0.0
	fmt.Printf("All adjustments reset to defaults\n")
}

func (bs *Shell) getTrackTypeDesc(trackType string) string {
	if trackType == "V" {
		return "vocal"
	}
	return "instrumental"
}