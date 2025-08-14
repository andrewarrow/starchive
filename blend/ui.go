package blend

import (
	"fmt"

	"github.com/chzyer/readline"
	"starchive/audio"
)

// Completer returns the auto-completion functionality for the shell
func (bs *Shell) Completer() readline.AutoCompleter {
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
		readline.PcItem("analyze-segments",
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

// ShowStatus displays the current blend settings
func (bs *Shell) ShowStatus() {
	fmt.Printf("--- Current Settings ---\n")
	fmt.Printf("Track 1 (%s %s): pitch %+d, tempo %+.1f%%, volume %.0f%%, window %+.1fs\n", 
		bs.ID1, bs.getTrackTypeDesc(bs.Type1), bs.Pitch1, bs.Tempo1, bs.Volume1, bs.Window1)
	fmt.Printf("Track 2 (%s %s): pitch %+d, tempo %+.1f%%, volume %.0f%%, window %+.1fs\n", 
		bs.ID2, bs.getTrackTypeDesc(bs.Type2), bs.Pitch2, bs.Tempo2, bs.Volume2, bs.Window2)
		
	if bs.Metadata1 != nil && bs.Metadata1.BPM != nil && bs.Metadata1.Key != nil {
		effectiveBPM1 := audio.CalculateEffectiveBPM(*bs.Metadata1.BPM, bs.Tempo1)
		effectiveKey1 := audio.CalculateEffectiveKey(*bs.Metadata1.Key, bs.Pitch1)
		fmt.Printf("  Effective: %.1f BPM, %s (was %.1f BPM, %s)\n", 
			effectiveBPM1, effectiveKey1, *bs.Metadata1.BPM, *bs.Metadata1.Key)
	}
	if bs.Metadata2 != nil && bs.Metadata2.BPM != nil && bs.Metadata2.Key != nil {
		effectiveBPM2 := audio.CalculateEffectiveBPM(*bs.Metadata2.BPM, bs.Tempo2)
		effectiveKey2 := audio.CalculateEffectiveKey(*bs.Metadata2.Key, bs.Pitch2)
		fmt.Printf("  Effective: %.1f BPM, %s (was %.1f BPM, %s)\n", 
			effectiveBPM2, effectiveKey2, *bs.Metadata2.BPM, *bs.Metadata2.Key)
	}
	
	// Show segment information
	activeSegments1 := 0
	activeSegments2 := 0
	for _, seg := range bs.Segments1 {
		if seg.Active { activeSegments1++ }
	}
	for _, seg := range bs.Segments2 {
		if seg.Active { activeSegments2++ }
	}
	
	if len(bs.Segments1) > 0 || len(bs.Segments2) > 0 {
		fmt.Printf("Segments: Track 1: %d/%d active, Track 2: %d/%d active\n", 
			activeSegments1, len(bs.Segments1), activeSegments2, len(bs.Segments2))
	}
	fmt.Printf("\n")
}

// ShowHelp displays detailed help information
func (bs *Shell) ShowHelp() {
	fmt.Printf("--- Blend Shell Commands ---\n")
	fmt.Printf("Playback:\n")
	fmt.Printf("  play [start_pos]    Play current blend (press any key to stop)\n")
	fmt.Printf("                      start_pos: seconds (default: middle, 0 = beginning)\n")
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
	fmt.Printf("  analyze-segments <1|2> Analyze energy levels of segments\n")
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