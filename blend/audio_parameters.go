package blend

import (
	"fmt"
	"strconv"
)

// HandleAudioParameterCommand processes audio parameter commands (pitch, tempo, volume, window)
func (bs *Shell) HandleAudioParameterCommand(cmd string, args []string) bool {
	switch cmd {
	case "pitch1":
		if len(args) > 0 {
			if val, err := strconv.Atoi(args[0]); err == nil {
				bs.Pitch1 = clamp(val, -12, 12)
				fmt.Printf("Track 1 pitch set to %+d semitones\n", bs.Pitch1)
			} else {
				fmt.Printf("Invalid pitch value: %s\n", args[0])
			}
		} else {
			fmt.Printf("Usage: pitch1 <semitones> (-12 to +12)\n")
		}
		
	case "pitch2":
		if len(args) > 0 {
			if val, err := strconv.Atoi(args[0]); err == nil {
				bs.Pitch2 = clamp(val, -12, 12)
				fmt.Printf("Track 2 pitch set to %+d semitones\n", bs.Pitch2)
			} else {
				fmt.Printf("Invalid pitch value: %s\n", args[0])
			}
		} else {
			fmt.Printf("Usage: pitch2 <semitones> (-12 to +12)\n")
		}
		
	case "tempo1":
		if len(args) > 0 {
			if val, err := strconv.ParseFloat(args[0], 64); err == nil {
				bs.Tempo1 = clampFloat(val, -50.0, 100.0)
				fmt.Printf("Track 1 tempo adjustment set to %+.1f%%\n", bs.Tempo1)
			} else {
				fmt.Printf("Invalid tempo value: %s\n", args[0])
			}
		} else {
			fmt.Printf("Usage: tempo1 <percentage> (-50 to +100)\n")
		}
		
	case "tempo2":
		if len(args) > 0 {
			if val, err := strconv.ParseFloat(args[0], 64); err == nil {
				bs.Tempo2 = clampFloat(val, -50.0, 100.0)
				fmt.Printf("Track 2 tempo adjustment set to %+.1f%%\n", bs.Tempo2)
			} else {
				fmt.Printf("Invalid tempo value: %s\n", args[0])
			}
		} else {
			fmt.Printf("Usage: tempo2 <percentage> (-50 to +100)\n")
		}
		
	case "volume1":
		if len(args) > 0 {
			if val, err := strconv.ParseFloat(args[0], 64); err == nil {
				bs.Volume1 = clampFloat(val, 0.0, 200.0)
				fmt.Printf("Track 1 volume set to %.0f%%\n", bs.Volume1)
			} else {
				fmt.Printf("Invalid volume value: %s\n", args[0])
			}
		} else {
			fmt.Printf("Usage: volume1 <percentage> (0 to 200)\n")
		}
		
	case "volume2":
		if len(args) > 0 {
			if val, err := strconv.ParseFloat(args[0], 64); err == nil {
				bs.Volume2 = clampFloat(val, 0.0, 200.0)
				fmt.Printf("Track 2 volume set to %.0f%%\n", bs.Volume2)
			} else {
				fmt.Printf("Invalid volume value: %s\n", args[0])
			}
		} else {
			fmt.Printf("Usage: volume2 <percentage> (0 to 200)\n")
		}
		
	case "window":
		if len(args) >= 2 {
			if val1, err1 := strconv.ParseFloat(args[0], 64); err1 == nil {
				if val2, err2 := strconv.ParseFloat(args[1], 64); err2 == nil {
					bs.Window1 = val1
					bs.Window2 = val2
					fmt.Printf("Track windows set to %+.1fs, %+.1fs\n", bs.Window1, bs.Window2)
				} else {
					fmt.Printf("Invalid second window value: %s\n", args[1])
				}
			} else {
				fmt.Printf("Invalid first window value: %s\n", args[0])
			}
		} else {
			fmt.Printf("Usage: window <seconds1> <seconds2>\n")
		}
		
	default:
		return false // Command not handled by this module
	}
	
	return true
}