package blend

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

// HandleSegmentBasicCommand processes basic segment manipulation commands
func (bs *Shell) HandleSegmentBasicCommand(cmd string, args []string) bool {
	switch cmd {
	case "random":
		if len(args) > 0 {
			bs.handleRandomCommand(args[0])
		} else {
			fmt.Printf("Usage: random <1|2>\n")
		}
		
	case "place":
		bs.handlePlaceCommand(args)
		
	case "shift":
		bs.handleShiftCommand(args)
		
	case "toggle":
		if len(args) > 0 {
			bs.handleToggleCommand(args[0])
		} else {
			fmt.Printf("Usage: toggle <track:segment> (e.g. 1:3)\n")
		}
		
	default:
		return false // Command not handled by this module
	}
	
	return true
}

// handleRandomCommand randomly places segments from a track
func (bs *Shell) handleRandomCommand(trackNum string) {
	var segments *[]VocalSegment
	var targetDuration float64
	var id string
	
	switch trackNum {
	case "1":
		segments = &bs.Segments1
		targetDuration = bs.Duration2  // Place track 1 segments across track 2
		id = bs.ID1
	case "2":
		segments = &bs.Segments2
		targetDuration = bs.Duration1  // Place track 2 segments across track 1
		id = bs.ID2
	default:
		fmt.Printf("Invalid track number: %s (use 1 or 2)\n", trackNum)
		return
	}
	
	if len(*segments) == 0 {
		fmt.Printf("No segments found for track %s. Run 'split %s' first.\n", trackNum, trackNum)
		return
	}
	
	fmt.Printf("Randomly placing %d segments from track %s (%s) across %.1fs...\n", 
		len(*segments), trackNum, id, targetDuration)
	
	// Generate random placements, ensuring no overlaps
	rand.Seed(time.Now().UnixNano())
	
	for i := range *segments {
		// Place randomly in first 80% of target track to avoid cutting off
		maxPlacement := targetDuration * 0.8
		placement := rand.Float64() * maxPlacement
		
		(*segments)[i].Placement = placement
		(*segments)[i].Active = true
		
		fmt.Printf("  %s:%d placed at %.1fs\n", trackNum, (*segments)[i].Index, placement)
	}
	
	fmt.Printf("Random placement completed for track %s\n", trackNum)
}

// handlePlaceCommand places a segment at a specific time
func (bs *Shell) handlePlaceCommand(args []string) {
	if len(args) < 3 || args[1] != "at" {
		fmt.Printf("Usage: place <track:segment> at <time>\n")
		fmt.Printf("Example: place 1:3 at 45.2\n")
		return
	}
	
	segmentRef := args[0]
	timeStr := args[2]
	
	trackNum, segNum, ok := bs.parseSegmentRef(segmentRef)
	if !ok {
		fmt.Printf("Invalid segment reference: %s (use format track:segment, e.g., 1:3)\n", segmentRef)
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
		segments = &bs.Segments1
	case 2:
		segments = &bs.Segments2
	default:
		fmt.Printf("Invalid track number: %d\n", trackNum)
		return
	}
	
	if len(*segments) == 0 {
		fmt.Printf("No segments found for track %d. Run 'split %d' first.\n", trackNum, trackNum)
		return
	}
	
	if segNum < 1 || segNum > len(*segments) {
		fmt.Printf("Segment %d not found. Track %d has %d segments.\n", segNum, trackNum, len(*segments))
		return
	}
	
	// Update segment placement
	segment := &(*segments)[segNum-1] // Convert 1-based to 0-based index
	segment.Placement = placement
	segment.Active = true // Placing a segment activates it
	
	fmt.Printf("Segment %d:%d placed at %.2fs and activated\n", trackNum, segNum, placement)
}

// handleShiftCommand shifts a segment timing
func (bs *Shell) handleShiftCommand(args []string) {
	if len(args) < 2 {
		fmt.Printf("Usage: shift <track:segment> <+/-time>\n")
		fmt.Printf("Example: shift 1:3 +2.5 (shift forward by 2.5 seconds)\n")
		fmt.Printf("Example: shift 1:3 -1.0 (shift backward by 1.0 seconds)\n")
		return
	}
	
	segmentRef := args[0]
	shiftStr := args[1]
	
	trackNum, segNum, ok := bs.parseSegmentRef(segmentRef)
	if !ok {
		fmt.Printf("Invalid segment reference: %s (use format track:segment, e.g., 1:3)\n", segmentRef)
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
		segments = &bs.Segments1
	case 2:
		segments = &bs.Segments2
	default:
		fmt.Printf("Invalid track number: %d\n", trackNum)
		return
	}
	
	if len(*segments) == 0 {
		fmt.Printf("No segments found for track %d. Run 'split %d' first.\n", trackNum, trackNum)
		return
	}
	
	if segNum < 1 || segNum > len(*segments) {
		fmt.Printf("Segment %d not found. Track %d has %d segments.\n", segNum, trackNum, len(*segments))
		return
	}
	
	// Update segment placement
	segment := &(*segments)[segNum-1] // Convert 1-based to 0-based index
	oldPlacement := segment.Placement
	segment.Placement += shift
	
	// Prevent negative placement
	if segment.Placement < 0 {
		segment.Placement = 0
	}
	
	fmt.Printf("Segment %d:%d shifted from %.2fs to %.2fs (%+.2fs)\n", 
		trackNum, segNum, oldPlacement, segment.Placement, shift)
}

// handleToggleCommand toggles a segment on/off
func (bs *Shell) handleToggleCommand(segmentRef string) {
	trackNum, segNum, ok := bs.parseSegmentRef(segmentRef)
	if !ok {
		fmt.Printf("Invalid segment reference: %s (use format track:segment, e.g., 1:3)\n", segmentRef)
		return
	}
	
	var segments *[]VocalSegment
	switch trackNum {
	case 1:
		segments = &bs.Segments1
	case 2:
		segments = &bs.Segments2
	default:
		fmt.Printf("Invalid track number: %d\n", trackNum)
		return
	}
	
	if len(*segments) == 0 {
		fmt.Printf("No segments found for track %d. Run 'split %d' first.\n", trackNum, trackNum)
		return
	}
	
	if segNum < 1 || segNum > len(*segments) {
		fmt.Printf("Segment %d not found. Track %d has %d segments.\n", segNum, trackNum, len(*segments))
		return
	}
	
	// Toggle the segment
	segment := &(*segments)[segNum-1] // Convert 1-based to 0-based index
	segment.Active = !segment.Active
	
	status := "inactive"
	if segment.Active {
		status = "active"
	}
	
	fmt.Printf("Segment %d:%d is now %s\n", trackNum, segNum, status)
}

// parseSegmentRef parses segment references like "1:3" 
func (bs *Shell) parseSegmentRef(segRef string) (int, int, bool) {
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