package blend

import (
	"fmt"
)

// HandleBasicCommand processes basic shell commands (help, status, reset, exit)
// Returns: true if command was handled, false if not handled
func (bs *Shell) HandleBasicCommand(cmd string, args []string) bool {
	switch cmd {
	case "exit", "quit", "q":
		fmt.Println("Exiting blend shell...")
		// We need to signal exit differently - we'll handle this in the main command handler
		return true
		
	case "help", "h":
		bs.ShowHelp()
		
	case "status", "s":
		bs.ShowStatus()
		
	case "reset", "r":
		bs.ResetAdjustments()
		
	case "conflict-detect":
		bs.handleConflictDetectCommand()
		
	default:
		return false // Command not handled by this module
	}
	
	return true
}

// IsExitCommand checks if the command should exit the shell
func IsExitCommand(cmd string) bool {
	return cmd == "exit" || cmd == "quit" || cmd == "q"
}

// handleConflictDetectCommand analyzes potential vocal segment overlaps
func (bs *Shell) handleConflictDetectCommand() {
	fmt.Printf("Analyzing potential vocal conflicts...\n")
	
	// Check if we have active segments to analyze
	activeSegments1 := bs.getActiveSegments(1)
	activeSegments2 := bs.getActiveSegments(2)
	
	if len(activeSegments1) == 0 && len(activeSegments2) == 0 {
		fmt.Printf("No active segments to analyze. Use 'add' commands to place segments first.\n")
		return
	}
	
	conflicts := 0
	warnings := 0
	
	fmt.Printf("Checking %d active segments...\n", len(activeSegments1)+len(activeSegments2))
	
	// Analyze overlaps between all active segments
	for _, seg1 := range activeSegments1 {
		for _, seg2 := range activeSegments1 {
			if seg1.Index != seg2.Index {
				overlap := bs.calculateOverlap(seg1, seg2)
				if overlap > 0 {
					conflicts++
					fmt.Printf("  ⚠️  CONFLICT: Segments %d and %d overlap by %.1fs\n", 
						seg1.Index, seg2.Index, overlap)
				}
			}
		}
		
		for _, seg2 := range activeSegments2 {
			overlap := bs.calculateOverlap(seg1, seg2)
			if overlap > 0 {
				if bs.Type1 == "V" && bs.Type2 == "V" {
					conflicts++
					fmt.Printf("  ⚠️  VOCAL CONFLICT: Track 1 seg %d and Track 2 seg %d overlap by %.1fs\n", 
						seg1.Index, seg2.Index, overlap)
				} else {
					warnings++
					fmt.Printf("  ℹ️  OVERLAP: Track 1 seg %d and Track 2 seg %d overlap by %.1fs\n", 
						seg1.Index, seg2.Index, overlap)
				}
			}
		}
	}
	
	for _, seg1 := range activeSegments2 {
		for _, seg2 := range activeSegments2 {
			if seg1.Index != seg2.Index {
				overlap := bs.calculateOverlap(seg1, seg2)
				if overlap > 0 {
					conflicts++
					fmt.Printf("  ⚠️  CONFLICT: Track 2 segments %d and %d overlap by %.1fs\n", 
						seg1.Index, seg2.Index, overlap)
				}
			}
		}
	}
	
	// Summary
	fmt.Printf("\n--- Conflict Analysis Summary ---\n")
	if conflicts > 0 {
		fmt.Printf("🚨 %d CONFLICTS found (segments overlap problematically)\n", conflicts)
	}
	if warnings > 0 {
		fmt.Printf("⚠️  %d overlaps detected (may be acceptable depending on arrangement)\n", warnings)
	}
	if conflicts == 0 && warnings == 0 {
		fmt.Printf("✅ No conflicts detected - segments are well spaced\n")
	}
	
	// Suggestions
	if conflicts > 0 || warnings > 0 {
		fmt.Printf("\nSuggestions:\n")
		fmt.Printf("  - Use 'move' command to reposition conflicting segments\n")
		fmt.Printf("  - Use 'gap-finder' to find better placement spots\n")
		fmt.Printf("  - Consider shorter segment durations\n")
	}
}

// getActiveSegments returns all active segments for a track
func (bs *Shell) getActiveSegments(track int) []VocalSegment {
	var segments []VocalSegment
	
	if track == 1 {
		for _, seg := range bs.Segments1 {
			if seg.Active {
				segments = append(segments, seg)
			}
		}
	} else {
		for _, seg := range bs.Segments2 {
			if seg.Active {
				segments = append(segments, seg)
			}
		}
	}
	
	return segments
}

// calculateOverlap calculates the overlap time between two segments in seconds
func (bs *Shell) calculateOverlap(seg1, seg2 VocalSegment) float64 {
	start1 := seg1.Placement
	end1 := seg1.Placement + seg1.Duration
	start2 := seg2.Placement
	end2 := seg2.Placement + seg2.Duration
	
	// No overlap if one ends before the other starts
	if end1 <= start2 || end2 <= start1 {
		return 0
	}
	
	// Calculate overlap duration
	overlapStart := max(start1, start2)
	overlapEnd := min(end1, end2)
	
	return overlapEnd - overlapStart
}

// Helper functions
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}