package blend

// HandleSegmentManipulationCommand processes segment manipulation commands
func (bs *Shell) HandleSegmentManipulationCommand(cmd string, args []string) bool {
	// Try basic commands first
	if bs.HandleSegmentBasicCommand(cmd, args) {
		return true
	}
	
	// Try advanced commands
	return bs.HandleSegmentAdvancedCommand(cmd, args)
}

