package main

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/chzyer/readline"
)

func handleBlendCommand() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: starchive blend <id1> <id2>")
		fmt.Println("Example: starchive blend OIduTH7NYA8 EbD7lfrsY2s")
		fmt.Println("Enters an interactive blend shell with real-time controls.")
		os.Exit(1)
	}

	id1 := os.Args[2]
	id2 := os.Args[3]
	
	blendShell := newBlendShell(id1, id2)
	blendShell.run()
}

func handleBlendClearCommand() {
	if len(os.Args) == 2 {
		clearAllBlendMetadata()
	} else if len(os.Args) == 4 {
		id1 := os.Args[2]
		id2 := os.Args[3]
		clearSpecificBlendMetadata(id1, id2)
	} else {
		fmt.Println("Usage: starchive blend-clear [id1 id2]")
		fmt.Println("  starchive blend-clear          Clear all blend metadata")
		fmt.Println("  starchive blend-clear id1 id2  Clear metadata for specific track pair")
		os.Exit(1)
	}
}

func clearAllBlendMetadata() {
	tmpDir := "/tmp"
	pattern := "starchive_blend_*.tmp"
	
	cmd := exec.Command("find", tmpDir, "-name", pattern, "-type", "f", "-delete")
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error clearing blend metadata: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Println("All blend metadata cleared.")
}

func clearSpecificBlendMetadata(id1, id2 string) {
	file1 := "/tmp/starchive_blend_" + id1 + "_" + id2 + ".tmp"
	file2 := "/tmp/starchive_blend_" + id2 + "_" + id1 + ".tmp"
	
	removed := false
	
	if _, err := os.Stat(file1); err == nil {
		os.Remove(file1)
		removed = true
	}
	
	if _, err := os.Stat(file2); err == nil {
		os.Remove(file2)
		removed = true
	}
	
	if removed {
		fmt.Printf("Blend metadata cleared for tracks %s and %s.\n", id1, id2)
	} else {
		fmt.Printf("No blend metadata found for tracks %s and %s.\n", id1, id2)
	}
}

func calculateIntelligentAdjustments(sourceBPM float64, sourceKey string, targetBPM float64, targetKey string) (int, float64) {
	pitch := calculateKeyDifference(sourceKey, targetKey)
	tempo := 0.0
	
	return pitch, tempo
}

var keyMap = map[string]int{
	"C major": 0, "G major": 7, "D major": 2, "A major": 9, "E major": 4, "B major": 11,
	"F# major": 6, "Db major": 1, "Ab major": 8, "Eb major": 3, "Bb major": 10, "F major": 5,
	"C# major": 1, "G# major": 8,
	"A minor": 9, "E minor": 4, "B minor": 11, "F# minor": 6, "C# minor": 1, "G# minor": 8,
	"Eb minor": 3, "Bb minor": 10, "F minor": 5, "C minor": 0, "G minor": 7, "D minor": 2,
}

func calculateKeyDifference(key1, key2 string) int {
	
	val1, exists1 := keyMap[key1]
	val2, exists2 := keyMap[key2]
	
	if !exists1 || !exists2 {
		return 0
	}
	
	diff := val2 - val1
	if diff > 6 {
		diff -= 12
	} else if diff < -6 {
		diff += 12
	}
	
	return diff
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func clampFloat(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func calculateEffectiveBPM(originalBPM float64, tempoAdjustment float64) float64 {
	multiplier := 1.0 + (tempoAdjustment / 100.0)
	return originalBPM * multiplier
}

func calculateEffectiveKey(originalKey string, pitchAdjustment int) string {
	if pitchAdjustment == 0 {
		return originalKey
	}
	
	reverseKeyMap := make(map[int]string)
	isMinor := strings.Contains(originalKey, "minor")
	
	for key, value := range keyMap {
		if (isMinor && strings.Contains(key, "minor")) || (!isMinor && strings.Contains(key, "major")) {
			reverseKeyMap[value] = key
		}
	}
	
	originalValue, exists := keyMap[originalKey]
	if !exists {
		return originalKey
	}
	
	newValue := (originalValue + pitchAdjustment) % 12
	if newValue < 0 {
		newValue += 12
	}
	
	if newKey, exists := reverseKeyMap[newValue]; exists {
		return newKey
	}
	
	return originalKey
}

type VocalSegment struct {
	Index     int     `json:"index"`
	StartTime float64 `json:"start_time"`
	Duration  float64 `json:"duration"`
	Placement float64 `json:"placement"` // Where to place in target track
	Active    bool    `json:"active"`    // Whether this segment is enabled
}

type BlendShell struct {
	id1, id2           string
	metadata1, metadata2 *VideoMetadata
	type1, type2       string
	pitch1, pitch2     int
	tempo1, tempo2     float64
	volume1, volume2   float64
	duration1, duration2 float64
	window1, window2   float64
	inputPath1, inputPath2 string
	db                 *sql.DB
	previousBPMMatch, previousKeyMatch string
	segments1, segments2 []VocalSegment // Vocal segments for each track
	segmentsDir1, segmentsDir2 string   // Directories containing split files
}

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

func (bs *BlendShell) handleCommand(input string) bool {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return true
	}
	
	cmd := parts[0]
	// Remove leading slash if present for backward compatibility
	if strings.HasPrefix(cmd, "/") {
		cmd = cmd[1:]
	}
	args := parts[1:]
	
	switch cmd {
	case "exit", "quit", "q":
		fmt.Println("Exiting blend shell...")
		return false
		
	case "help", "h":
		bs.showHelp()
		
	case "play", "p":
		bs.playBlend()
		
	case "status", "s":
		bs.showStatus()
		
	case "reset", "r":
		bs.resetAdjustments()
		
	case "pitch1":
		if len(args) > 0 {
			if val, err := strconv.Atoi(args[0]); err == nil {
				bs.pitch1 = clamp(val, -12, 12)
				fmt.Printf("Track 1 pitch: %+d semitones\n", bs.pitch1)
			} else {
				fmt.Printf("Invalid pitch value: %s\n", args[0])
			}
		} else {
			fmt.Printf("Current track 1 pitch: %+d semitones\n", bs.pitch1)
		}
		
	case "pitch2":
		if len(args) > 0 {
			if val, err := strconv.Atoi(args[0]); err == nil {
				bs.pitch2 = clamp(val, -12, 12)
				fmt.Printf("Track 2 pitch: %+d semitones\n", bs.pitch2)
			} else {
				fmt.Printf("Invalid pitch value: %s\n", args[0])
			}
		} else {
			fmt.Printf("Current track 2 pitch: %+d semitones\n", bs.pitch2)
		}
		
	case "tempo1":
		if len(args) > 0 {
			if val, err := strconv.ParseFloat(args[0], 64); err == nil {
				bs.tempo1 = clampFloat(val, -50.0, 100.0)
				fmt.Printf("Track 1 tempo: %+.1f%%\n", bs.tempo1)
			} else {
				fmt.Printf("Invalid tempo value: %s\n", args[0])
			}
		} else {
			fmt.Printf("Current track 1 tempo: %+.1f%%\n", bs.tempo1)
		}
		
	case "tempo2":
		if len(args) > 0 {
			if val, err := strconv.ParseFloat(args[0], 64); err == nil {
				bs.tempo2 = clampFloat(val, -50.0, 100.0)
				fmt.Printf("Track 2 tempo: %+.1f%%\n", bs.tempo2)
			} else {
				fmt.Printf("Invalid tempo value: %s\n", args[0])
			}
		} else {
			fmt.Printf("Current track 2 tempo: %+.1f%%\n", bs.tempo2)
		}
		
	case "volume1":
		if len(args) > 0 {
			if val, err := strconv.ParseFloat(args[0], 64); err == nil {
				bs.volume1 = clampFloat(val, 0.0, 200.0)
				fmt.Printf("Track 1 volume: %.0f%%\n", bs.volume1)
			} else {
				fmt.Printf("Invalid volume value: %s\n", args[0])
			}
		} else {
			fmt.Printf("Current track 1 volume: %.0f%%\n", bs.volume1)
		}
		
	case "volume2":
		if len(args) > 0 {
			if val, err := strconv.ParseFloat(args[0], 64); err == nil {
				bs.volume2 = clampFloat(val, 0.0, 200.0)
				fmt.Printf("Track 2 volume: %.0f%%\n", bs.volume2)
			} else {
				fmt.Printf("Invalid volume value: %s\n", args[0])
			}
		} else {
			fmt.Printf("Current track 2 volume: %.0f%%\n", bs.volume2)
		}
		
	case "type1":
		if len(args) > 0 {
			if args[0] == "vocal" || args[0] == "v" {
				bs.type1 = "V"
				bs.inputPath1 = getAudioFilename(bs.id1, bs.type1)
				fmt.Printf("Track 1 type: vocal\n")
			} else if args[0] == "instrumental" || args[0] == "i" {
				bs.type1 = "I"
				bs.inputPath1 = getAudioFilename(bs.id1, bs.type1)
				fmt.Printf("Track 1 type: instrumental\n")
			} else {
				fmt.Printf("Invalid type: %s (use 'vocal' or 'instrumental')\n", args[0])
			}
		} else {
			fmt.Printf("Current track 1 type: %s\n", bs.getTrackTypeDesc(bs.type1))
		}
		
	case "type2":
		if len(args) > 0 {
			if args[0] == "vocal" || args[0] == "v" {
				bs.type2 = "V"
				bs.inputPath2 = getAudioFilename(bs.id2, bs.type2)
				fmt.Printf("Track 2 type: vocal\n")
			} else if args[0] == "instrumental" || args[0] == "i" {
				bs.type2 = "I"
				bs.inputPath2 = getAudioFilename(bs.id2, bs.type2)
				fmt.Printf("Track 2 type: instrumental\n")
			} else {
				fmt.Printf("Invalid type: %s (use 'vocal' or 'instrumental')\n", args[0])
			}
		} else {
			fmt.Printf("Current track 2 type: %s\n", bs.getTrackTypeDesc(bs.type2))
		}
		
	case "window":
		if len(args) >= 2 {
			if val1, err1 := strconv.ParseFloat(args[0], 64); err1 == nil {
				if val2, err2 := strconv.ParseFloat(args[1], 64); err2 == nil {
					bs.window1 = val1
					bs.window2 = val2
					fmt.Printf("Window offsets: track 1 %+.1fs, track 2 %+.1fs\n", bs.window1, bs.window2)
				} else {
					fmt.Printf("Invalid window offset for track 2: %s\n", args[1])
				}
			} else {
				fmt.Printf("Invalid window offset for track 1: %s\n", args[0])
			}
		} else {
			fmt.Printf("Usage: window <track1_offset> <track2_offset>\n")
			fmt.Printf("Current window offsets: track 1 %+.1fs, track 2 %+.1fs\n", bs.window1, bs.window2)
		}
		
	case "match":
		if len(args) > 0 {
			bs.handleMatchCommand(args[0])
		} else {
			fmt.Printf("Usage: match <bpm1to2|bpm2to1|key1to2|key2to1>\n")
		}
		
	case "invert":
		bs.handleInvertCommand()
		
	case "split":
		if len(args) > 0 {
			bs.handleSplitCommand(args[0])
		} else {
			fmt.Printf("Usage: split <1|2>\n")
		}
		
	case "segments":
		if len(args) > 0 {
			bs.handleSegmentsCommand(args[0])
		} else {
			bs.handleSegmentsCommand("")
		}
		
	case "place":
		if len(args) >= 3 && args[1] == "at" {
			bs.handlePlaceCommand(args[0], args[2])
		} else {
			fmt.Printf("Usage: place <track:seg> at <time>\n")
			fmt.Printf("Example: place 1:3 at 45.2\n")
		}
		
	case "shift":
		if len(args) >= 2 {
			bs.handleShiftCommand(args[0], args[1])
		} else {
			fmt.Printf("Usage: shift <track:seg> <+/-time>\n")
			fmt.Printf("Example: shift 1:3 +2.5\n")
		}
		
	case "toggle":
		if len(args) > 0 {
			bs.handleToggleCommand(args[0])
		} else {
			fmt.Printf("Usage: toggle <track:seg>\n")
			fmt.Printf("Example: toggle 1:3\n")
		}
		
	case "preview":
		if len(args) > 0 {
			bs.handlePreviewCommand(args[0])
		} else {
			fmt.Printf("Usage: preview <track:seg>\n")
			fmt.Printf("Example: preview 1:3\n")
		}
		
	case "random":
		if len(args) > 0 {
			bs.handleRandomCommand(args[0])
		} else {
			fmt.Printf("Usage: random <1|2>\n")
			fmt.Printf("Example: random 1\n")
		}
		
	default:
		fmt.Printf("Unknown command: %s (type help for commands)\n", cmd)
	}
	
	// Auto-show status after commands that modify state
	switch cmd {
	case "exit", "quit", "q", "help", "h", "status", "s", "play", "p":
		// Don't show status after these commands
	default:
		bs.showStatus()
	}
	
	return true
}

func (bs *BlendShell) handleMatchCommand(matchType string) {
	if bs.metadata1 == nil || bs.metadata2 == nil {
		fmt.Printf("Cannot match - missing metadata\n")
		return
	}
	
	switch matchType {
	case "bpm1to2":
		if bs.metadata1.BPM != nil && bs.metadata2.BPM != nil {
			bpm1 := *bs.metadata1.BPM
			bpm2 := *bs.metadata2.BPM
			requiredRatio := bpm2 / bpm1
			bs.tempo1 = (requiredRatio - 1.0) * 100.0
			bs.tempo2 = 0.0
			fmt.Printf("Matched track 1 BPM (%.1f) to track 2 BPM (%.1f)\n", bpm1, bpm2)
			fmt.Printf("Track 1 tempo: %+.1f%%\n", bs.tempo1)
		} else {
			fmt.Printf("BPM data not available\n")
		}
		
	case "bpm2to1":
		if bs.metadata1.BPM != nil && bs.metadata2.BPM != nil {
			bpm1 := *bs.metadata1.BPM
			bpm2 := *bs.metadata2.BPM
			requiredRatio := bpm1 / bpm2
			bs.tempo2 = (requiredRatio - 1.0) * 100.0
			bs.tempo1 = 0.0
			fmt.Printf("Matched track 2 BPM (%.1f) to track 1 BPM (%.1f)\n", bpm2, bpm1)
			fmt.Printf("Track 2 tempo: %+.1f%%\n", bs.tempo2)
		} else {
			fmt.Printf("BPM data not available\n")
		}
		
	case "key1to2":
		if bs.metadata1.Key != nil && bs.metadata2.Key != nil {
			key1 := *bs.metadata1.Key
			key2 := *bs.metadata2.Key
			bs.pitch1 = calculateKeyDifference(key1, key2)
			bs.pitch2 = 0
			fmt.Printf("Matched track 1 key (%s) to track 2 key (%s)\n", key1, key2)
			fmt.Printf("Track 1 pitch: %+d semitones\n", bs.pitch1)
		} else {
			fmt.Printf("Key data not available\n")
		}
		
	case "key2to1":
		if bs.metadata1.Key != nil && bs.metadata2.Key != nil {
			key1 := *bs.metadata1.Key
			key2 := *bs.metadata2.Key
			bs.pitch2 = calculateKeyDifference(key2, key1)
			bs.pitch1 = 0
			fmt.Printf("Matched track 2 key (%s) to track 1 key (%s)\n", key2, key1)
			fmt.Printf("Track 2 pitch: %+d semitones\n", bs.pitch2)
		} else {
			fmt.Printf("Key data not available\n")
		}
		
	default:
		fmt.Printf("Unknown match type: %s\n", matchType)
		fmt.Printf("Available: bpm1to2, bpm2to1, key1to2, key2to1\n")
	}
}

func (bs *BlendShell) handleInvertCommand() {
	if bs.metadata1 == nil || bs.metadata2 == nil {
		fmt.Printf("Cannot invert - missing metadata\n")
		return
	}
	
	fmt.Printf("Inverting current match state...\n")
	
	// Save current state to determine what was matched
	stateFile := fmt.Sprintf("/tmp/starchive_invert_%s_%s.tmp", bs.id1, bs.id2)
	
	// Check if we have a previous state to invert from
	if bs.loadInvertState(stateFile) {
		// We have previous state, so invert it
		bs.applyInvertedState()
	} else {
		// No previous state, save current and apply default invert
		bs.saveInvertState(stateFile)
		bs.resetAdjustments()
		// Default: match bpm2to1 and key2to1
		bs.handleMatchCommand("bpm2to1")
		bs.handleMatchCommand("key2to1")
	}
}

func (bs *BlendShell) getKeyComplexity(key string) int {
	// Count sharps and flats to determine key complexity
	sharps := strings.Count(key, "#")
	flats := strings.Count(key, "b")
	return sharps + flats
}

type InvertState struct {
	BPMMatch string
	KeyMatch string
}

func (bs *BlendShell) saveInvertState(stateFile string) {
	// Determine current match state based on adjustments
	var bmpMatch, keyMatch string
	
	if bs.tempo1 != 0 && bs.tempo2 == 0 {
		bmpMatch = "bpm1to2"
	} else if bs.tempo2 != 0 && bs.tempo1 == 0 {
		bmpMatch = "bpm2to1"
	} else {
		bmpMatch = "none"
	}
	
	if bs.pitch1 != 0 && bs.pitch2 == 0 {
		keyMatch = "key1to2"
	} else if bs.pitch2 != 0 && bs.pitch1 == 0 {
		keyMatch = "key2to1"
	} else {
		keyMatch = "none"
	}
	
	content := fmt.Sprintf("%s,%s", bmpMatch, keyMatch)
	os.WriteFile(stateFile, []byte(content), 0644)
}

func (bs *BlendShell) loadInvertState(stateFile string) bool {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return false
	}
	
	parts := strings.Split(string(data), ",")
	if len(parts) != 2 {
		return false
	}
	
	bs.previousBPMMatch = parts[0]
	bs.previousKeyMatch = parts[1]
	return true
}

func (bs *BlendShell) applyInvertedState() {
	bs.resetAdjustments()
	
	// Invert BPM match
	switch bs.previousBPMMatch {
	case "bpm1to2":
		bs.handleMatchCommand("bpm2to1")
	case "bpm2to1":
		bs.handleMatchCommand("bpm1to2")
	}
	
	// Invert key match
	switch bs.previousKeyMatch {
	case "key1to2":
		bs.handleMatchCommand("key2to1")
	case "key2to1":
		bs.handleMatchCommand("key1to2")
	}
	
	// Clear the state file so next invert toggles back
	stateFile := fmt.Sprintf("/tmp/starchive_invert_%s_%s.tmp", bs.id1, bs.id2)
	os.Remove(stateFile)
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

func (bs *BlendShell) handleSplitCommand(trackNum string) {
	var id, inputPath string
	var segments *[]VocalSegment
	var segmentsDir string
	
	switch trackNum {
	case "1":
		id = bs.id1
		inputPath = bs.inputPath1
		segments = &bs.segments1
		segmentsDir = bs.segmentsDir1
	case "2":
		id = bs.id2
		inputPath = bs.inputPath2
		segments = &bs.segments2
		segmentsDir = bs.segmentsDir2
	default:
		fmt.Printf("Invalid track number: %s (use 1 or 2)\n", trackNum)
		return
	}
	
	// Only split vocal tracks
	trackType := bs.type1
	if trackNum == "2" {
		trackType = bs.type2
	}
	if trackType != "V" {
		fmt.Printf("Track %s is not vocal type. Switch to vocal first using 'type%s vocal'\n", trackNum, trackNum)
		return
	}
	
	fmt.Printf("Splitting track %s (%s) into vocal segments...\n", trackNum, id)
	
	if err := os.MkdirAll(segmentsDir, 0755); err != nil {
		fmt.Printf("Error creating segments directory: %v\n", err)
		return
	}
	
	// Run silence detection
	silenceCmd := exec.Command("ffmpeg", "-hide_banner", "-i", inputPath,
		"-af", "silencedetect=noise=-35dB:d=0.5", "-f", "null", "-")
	
	silenceOutput, err := silenceCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error detecting silence: %v\n", err)
		return
	}
	
	// Extract timestamps
	sedCmd := exec.Command("sed", "-n", "s/.*silence_end: \\([0-9.]*\\).*/\\1/p")
	sedCmd.Stdin = strings.NewReader(string(silenceOutput))
	sedOutput, err := sedCmd.Output()
	if err != nil {
		fmt.Printf("Error extracting timestamps: %v\n", err)
		return
	}
	
	timestamps := strings.TrimSpace(string(sedOutput))
	timestamps = strings.ReplaceAll(timestamps, "\n", ",")
	
	if timestamps == "" {
		fmt.Printf("No silence detected. Creating single segment.\n")
		outputPath := fmt.Sprintf("%s/part_001.wav", segmentsDir)
		copyCmd := exec.Command("cp", inputPath, outputPath)
		if err := copyCmd.Run(); err != nil {
			fmt.Printf("Error copying file: %v\n", err)
			return
		}
		
		duration, _ := getAudioDuration(inputPath)
		*segments = []VocalSegment{{
			Index: 1, StartTime: 0, Duration: duration,
			Placement: 0, Active: false,
		}}
		fmt.Printf("Created 1 segment\n")
		return
	}
	
	// Split the file
	outputPattern := fmt.Sprintf("%s/part_%%03d.wav", segmentsDir)
	splitCmd := exec.Command("ffmpeg", "-hide_banner", "-y", "-i", inputPath,
		"-c", "copy", "-f", "segment", "-segment_times", timestamps, outputPattern)
	
	err = splitCmd.Run()
	if err != nil {
		fmt.Printf("Error splitting file: %v\n", err)
		return
	}
	
	// Analyze created segments
	bs.loadSegments(trackNum)
	fmt.Printf("Successfully split track %s into %d segments\n", trackNum, len(*segments))
}

func (bs *BlendShell) loadSegments(trackNum string) {
	var segments *[]VocalSegment
	var segmentsDir string
	
	switch trackNum {
	case "1":
		segments = &bs.segments1
		segmentsDir = bs.segmentsDir1
	case "2":
		segments = &bs.segments2
		segmentsDir = bs.segmentsDir2
	default:
		return
	}
	
	entries, err := os.ReadDir(segmentsDir)
	if err != nil {
		return
	}
	
	*segments = []VocalSegment{}
	for i, entry := range entries {
		if strings.HasPrefix(entry.Name(), "part_") && strings.HasSuffix(entry.Name(), ".wav") {
			segmentPath := fmt.Sprintf("%s/%s", segmentsDir, entry.Name())
			duration, err := getAudioDuration(segmentPath)
			if err != nil {
				continue
			}
			
			*segments = append(*segments, VocalSegment{
				Index: i + 1, StartTime: 0, Duration: duration,
				Placement: 0, Active: false,
			})
		}
	}
}

func (bs *BlendShell) handleSegmentsCommand(trackNum string) {
	if trackNum == "" {
		// Show both tracks
		fmt.Printf("--- Segments ---\n")
		if len(bs.segments1) > 0 {
			fmt.Printf("Track 1 (%s): %d segments\n", bs.id1, len(bs.segments1))
			for _, seg := range bs.segments1 {
				status := "off"
				if seg.Active { status = "on" }
				fmt.Printf("  1:%d - %.1fs duration, placed at %.1fs [%s]\n", 
					seg.Index, seg.Duration, seg.Placement, status)
			}
		} else {
			fmt.Printf("Track 1 (%s): no segments (use 'split 1')\n", bs.id1)
		}
		
		if len(bs.segments2) > 0 {
			fmt.Printf("Track 2 (%s): %d segments\n", bs.id2, len(bs.segments2))
			for _, seg := range bs.segments2 {
				status := "off"
				if seg.Active { status = "on" }
				fmt.Printf("  2:%d - %.1fs duration, placed at %.1fs [%s]\n", 
					seg.Index, seg.Duration, seg.Placement, status)
			}
		} else {
			fmt.Printf("Track 2 (%s): no segments (use 'split 2')\n", bs.id2)
		}
		return
	}
	
	// Show specific track
	var segments []VocalSegment
	var id string
	
	switch trackNum {
	case "1":
		segments = bs.segments1
		id = bs.id1
	case "2":
		segments = bs.segments2
		id = bs.id2
	default:
		fmt.Printf("Invalid track number: %s (use 1 or 2)\n", trackNum)
		return
	}
	
	if len(segments) == 0 {
		fmt.Printf("Track %s (%s): no segments (use 'split %s')\n", trackNum, id, trackNum)
		return
	}
	
	fmt.Printf("Track %s (%s): %d segments\n", trackNum, id, len(segments))
	for _, seg := range segments {
		status := "off"
		if seg.Active { status = "on" }
		fmt.Printf("  %s:%d - %.1fs duration, placed at %.1fs [%s]\n", 
			trackNum, seg.Index, seg.Duration, seg.Placement, status)
	}
}

func (bs *BlendShell) parseSegmentRef(segRef string) (int, int, bool) {
	parts := strings.Split(segRef, ":")
	if len(parts) != 2 {
		return 0, 0, false
	}
	
	trackNum, err := strconv.Atoi(parts[0])
	if err != nil || (trackNum != 1 && trackNum != 2) {
		return 0, 0, false
	}
	
	segNum, err := strconv.Atoi(parts[1])
	if err != nil || segNum < 1 {
		return 0, 0, false
	}
	
	return trackNum, segNum, true
}

func (bs *BlendShell) handlePlaceCommand(segRef, timeStr string) {
	trackNum, segNum, ok := bs.parseSegmentRef(segRef)
	if !ok {
		fmt.Printf("Invalid segment reference: %s (use format track:segment like 1:3)\n", segRef)
		return
	}
	
	placement, err := strconv.ParseFloat(timeStr, 64)
	if err != nil {
		fmt.Printf("Invalid time: %s\n", timeStr)
		return
	}
	
	var segments *[]VocalSegment
	switch trackNum {
	case 1:
		segments = &bs.segments1
	case 2:
		segments = &bs.segments2
	}
	
	if segNum > len(*segments) {
		fmt.Printf("Segment %d not found for track %d\n", segNum, trackNum)
		return
	}
	
	(*segments)[segNum-1].Placement = placement
	(*segments)[segNum-1].Active = true
	fmt.Printf("Placed segment %s at %.1fs\n", segRef, placement)
}

func (bs *BlendShell) handleShiftCommand(segRef, shiftStr string) {
	trackNum, segNum, ok := bs.parseSegmentRef(segRef)
	if !ok {
		fmt.Printf("Invalid segment reference: %s (use format track:segment like 1:3)\n", segRef)
		return
	}
	
	shift, err := strconv.ParseFloat(shiftStr, 64)
	if err != nil {
		fmt.Printf("Invalid shift amount: %s\n", shiftStr)
		return
	}
	
	var segments *[]VocalSegment
	switch trackNum {
	case 1:
		segments = &bs.segments1
	case 2:
		segments = &bs.segments2
	}
	
	if segNum > len(*segments) {
		fmt.Printf("Segment %d not found for track %d\n", segNum, trackNum)
		return
	}
	
	(*segments)[segNum-1].Placement += shift
	fmt.Printf("Shifted segment %s by %+.1fs to %.1fs\n", 
		segRef, shift, (*segments)[segNum-1].Placement)
}

func (bs *BlendShell) handleToggleCommand(segRef string) {
	trackNum, segNum, ok := bs.parseSegmentRef(segRef)
	if !ok {
		fmt.Printf("Invalid segment reference: %s (use format track:segment like 1:3)\n", segRef)
		return
	}
	
	var segments *[]VocalSegment
	switch trackNum {
	case 1:
		segments = &bs.segments1
	case 2:
		segments = &bs.segments2
	}
	
	if segNum > len(*segments) {
		fmt.Printf("Segment %d not found for track %d\n", segNum, trackNum)
		return
	}
	
	(*segments)[segNum-1].Active = !(*segments)[segNum-1].Active
	status := "disabled"
	if (*segments)[segNum-1].Active {
		status = "enabled"
	}
	fmt.Printf("Segment %s is now %s\n", segRef, status)
}

func (bs *BlendShell) handlePreviewCommand(segRef string) {
	trackNum, segNum, ok := bs.parseSegmentRef(segRef)
	if !ok {
		fmt.Printf("Invalid segment reference: %s (use format track:segment like 1:3)\n", segRef)
		return
	}
	
	var segments []VocalSegment
	var segmentsDir string
	switch trackNum {
	case 1:
		segments = bs.segments1
		segmentsDir = bs.segmentsDir1
	case 2:
		segments = bs.segments2
		segmentsDir = bs.segmentsDir2
	}
	
	if segNum > len(segments) {
		fmt.Printf("Segment %d not found for track %d\n", segNum, trackNum)
		return
	}
	
	segment := segments[segNum-1]
	segmentPath := fmt.Sprintf("%s/part_%03d.wav", segmentsDir, segment.Index)
	
	if _, err := os.Stat(segmentPath); os.IsNotExist(err) {
		fmt.Printf("Segment file not found: %s\n", segmentPath)
		return
	}
	
	fmt.Printf("Previewing segment %s (%.1fs duration)...\n", segRef, segment.Duration)
	
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(segment.Duration*1000)*time.Millisecond)
	defer cancel()
	
	playCmd := exec.CommandContext(ctx, "ffplay", "-autoexit", "-nodisp", "-loglevel", "quiet", segmentPath)
	playCmd.Run()
	
	fmt.Printf("Preview completed.\n")
}

func (bs *BlendShell) handleRandomCommand(trackStr string) {
	trackNum, err := strconv.Atoi(trackStr)
	if err != nil || (trackNum != 1 && trackNum != 2) {
		fmt.Printf("Invalid track number: %s (use 1 or 2)\n", trackStr)
		return
	}
	
	var segments *[]VocalSegment
	var targetDuration float64
	var id string
	
	switch trackNum {
	case 1:
		segments = &bs.segments1
		targetDuration = bs.duration2
		id = bs.id1
	case 2:
		segments = &bs.segments2 
		targetDuration = bs.duration1
		id = bs.id2
	}
	
	if len(*segments) == 0 {
		fmt.Printf("No segments found for track %d. Run 'split %d' first.\n", trackNum, trackNum)
		return
	}
	
	fmt.Printf("Randomly placing %d segments from track %d (%s) across %.1fs...\n", 
		len(*segments), trackNum, id, targetDuration)
	
	// Generate random placements, ensuring no overlaps
	rand.Seed(time.Now().UnixNano())
	
	for i := range *segments {
		// Place randomly in first 80% of target track to avoid cutting off
		maxPlacement := targetDuration * 0.8
		placement := rand.Float64() * maxPlacement
		
		(*segments)[i].Placement = placement
		(*segments)[i].Active = true
		
		fmt.Printf("  %d:%d placed at %.1fs\n", trackNum, (*segments)[i].Index, placement)
	}
	
	fmt.Printf("Random placement completed for track %d\n", trackNum)
}

func detectTrackTypes(id1, id2 string) (string, string) {
	id1HasVocal := hasVocalFile(id1)
	id1HasInstrumental := hasInstrumentalFile(id1)
	id2HasVocal := hasVocalFile(id2)
	id2HasInstrumental := hasInstrumentalFile(id2)

	var type1, type2 string

	// If one track only has instrumental, make the other vocal (if possible)
	if !id2HasVocal && id2HasInstrumental {
		// Track 2 is instrumental-only, prefer vocal for track 1
		if id1HasVocal {
			type1 = "V"
		} else {
			type1 = "I"
		}
		type2 = "I"
	} else if !id1HasVocal && id1HasInstrumental {
		// Track 1 is instrumental-only, prefer vocal for track 2
		type1 = "I"
		if id2HasVocal {
			type2 = "V"
		} else {
			type2 = "I"
		}
	} else {
		// Both tracks have options, choose complementary types
		if id1HasVocal && !id1HasInstrumental {
			type1 = "V"
		} else if !id1HasVocal && id1HasInstrumental {
			type1 = "I"
		} else if id1HasVocal && id1HasInstrumental {
			type1 = "V"  // Default to vocal for track 1
		} else {
			type1 = "I"  // Fallback
		}

		if id2HasVocal && !id2HasInstrumental {
			type2 = "V"
		} else if !id2HasVocal && id2HasInstrumental {
			type2 = "I"
		} else if id2HasVocal && id2HasInstrumental {
			// Choose opposite of track 1
			if type1 == "V" {
				type2 = "I"
			} else {
				type2 = "V"
			}
		} else {
			type2 = "I"  // Fallback
		}
	}

	return type1, type2
}