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
		
	default:
		return false // Command not handled by this module
	}
	
	return true
}

// IsExitCommand checks if the command should exit the shell
func IsExitCommand(cmd string) bool {
	return cmd == "exit" || cmd == "quit" || cmd == "q"
}