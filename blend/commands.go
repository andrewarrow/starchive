package blend

import (
	"fmt"
	"strings"
)

// HandleCommand processes user commands in the blend shell
// Returns false if the command indicates exit
func (bs *Shell) HandleCommand(input string) bool {
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
	
	// Try each command module in order
	if !bs.HandleBasicCommand(cmd, args) {
		return false // Exit command was processed
	}
	
	if bs.HandleAudioCommand(cmd, args) {
		return true
	}
	
	if bs.HandleMatchingCommand(cmd, args) {
		return true
	}
	
	if bs.HandlePlaybackCommand(cmd, args) {
		return true
	}
	
	if bs.HandleSegmentCreationCommand(cmd, args) {
		return true
	}
	
	if bs.HandleSegmentManipulationCommand(cmd, args) {
		return true
	}
	
	// If no module handled the command, show error
	fmt.Printf("Unknown command: %s. Type 'help' for available commands.\n", cmd)
	return true
}

