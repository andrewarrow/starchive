package blend

import (
	"fmt"
)

// HandleBasicCommand processes basic shell commands (help, status, reset, exit)
func (bs *Shell) HandleBasicCommand(cmd string, args []string) bool {
	switch cmd {
	case "exit", "quit", "q":
		fmt.Println("Exiting blend shell...")
		return false
		
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