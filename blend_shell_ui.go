package main

import (
	"fmt"

	"github.com/chzyer/readline"
)

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