package blend

// HandleAudioCommand processes audio parameter commands by delegating to specialized modules
func (bs *Shell) HandleAudioCommand(cmd string, args []string) bool {
	// Try audio parameter commands first
	if bs.HandleAudioParameterCommand(cmd, args) {
		return true
	}
	
	// Try beat detection commands
	if bs.HandleBeatDetectionCommand(cmd, args) {
		return true
	}
	
	// Try gap finder commands
	if bs.HandleGapFinderCommand(cmd, args) {
		return true
	}
	
	return false // Command not handled by any audio module
}

