package audio

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

// GetAudioDuration returns the duration of an audio file in seconds
func GetAudioDuration(filePath string) (float64, error) {
	cmd := exec.Command("ffprobe", 
		"-v", "quiet",
		"-show_entries", "format=duration",
		"-of", "csv=p=0",
		filePath)
	
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	
	durationStr := strings.TrimSpace(string(output))
	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, err
	}
	
	return duration, nil
}

// WaitForKeyPress waits for user input or interrupt signal
func WaitForKeyPress() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	keyChan := make(chan bool, 1)

	go func() {
		reader := bufio.NewReader(os.Stdin)
		reader.ReadByte()
		keyChan <- true
	}()

	select {
	case <-keyChan:
		return
	case <-sigChan:
		fmt.Println("\nInterrupted.")
		return
	}
}

// GetVocalFilePath returns the path to the vocal version of a track
func GetVocalFilePath(id string) string {
	return fmt.Sprintf("./data/%s_(Vocals)_UVR_MDXNET_Main.wav", id)
}

// GetInstrumentalFilePath returns the path to the instrumental version of a track
func GetInstrumentalFilePath(id string) string {
	return fmt.Sprintf("./data/%s_(Instrumental)_UVR_MDXNET_Main.wav", id)
}

// GetVocalFilename returns just the filename for the vocal version
func GetVocalFilename(id string) string {
	return fmt.Sprintf("%s_(Vocals)_UVR_MDXNET_Main.wav", id)
}

// GetInstrumentalFilename returns just the filename for the instrumental version
func GetInstrumentalFilename(id string) string {
	return fmt.Sprintf("%s_(Instrumental)_UVR_MDXNET_Main.wav", id)
}

// GetAudioFilename returns the appropriate audio file path based on type
func GetAudioFilename(id, audioType string) string {
	switch audioType {
	case "V", "vocal", "vocals":
		return GetVocalFilePath(id)
	case "I", "instrumental", "instrumentals":
		return GetInstrumentalFilePath(id)
	default:
		return fmt.Sprintf("./data/%s.wav", id)
	}
}

// HasVocalFile checks if a vocal file exists for the given ID
func HasVocalFile(id string) bool {
	_, err := os.Stat(GetVocalFilePath(id))
	return !os.IsNotExist(err)
}

// HasInstrumentalFile checks if an instrumental file exists for the given ID
func HasInstrumentalFile(id string) bool {
	_, err := os.Stat(GetInstrumentalFilePath(id))
	return !os.IsNotExist(err)
}

// CalculateEffectiveBPM calculates the effective BPM after tempo adjustment
func CalculateEffectiveBPM(originalBPM float64, tempoAdjustment float64) float64 {
	multiplier := 1.0 + (tempoAdjustment / 100.0)
	return originalBPM * multiplier
}

// CalculateEffectiveKey calculates the effective key after pitch adjustment
func CalculateEffectiveKey(originalKey string, pitchAdjustment int) string {
	if pitchAdjustment == 0 {
		return originalKey
	}
	
	// Key mapping for pitch calculations
	keyMap := map[string]int{
		"C major": 0, "C minor": 0,
		"C# major": 1, "C# minor": 1, "Db major": 1, "Db minor": 1,
		"D major": 2, "D minor": 2,
		"D# major": 3, "D# minor": 3, "Eb major": 3, "Eb minor": 3,
		"E major": 4, "E minor": 4,
		"F major": 5, "F minor": 5,
		"F# major": 6, "F# minor": 6, "Gb major": 6, "Gb minor": 6,
		"G major": 7, "G minor": 7,
		"G# major": 8, "G# minor": 8, "Ab major": 8, "Ab minor": 8,
		"A major": 9, "A minor": 9,
		"A# major": 10, "A# minor": 10, "Bb major": 10, "Bb minor": 10,
		"B major": 11, "B minor": 11,
	}
	
	reverseKeyMap := make(map[int]string)
	isMinor := strings.Contains(originalKey, "minor")
	
	for key, value := range keyMap {
		if (isMinor && strings.Contains(key, "minor")) || (!isMinor && strings.Contains(key, "major")) {
			reverseKeyMap[value] = key
		}
	}
	
	originalValue, exists := keyMap[originalKey]
	if !exists {
		return originalKey // Return original if not found
	}
	
	newValue := (originalValue + pitchAdjustment) % 12
	if newValue < 0 {
		newValue += 12
	}
	
	if newKey, exists := reverseKeyMap[newValue]; exists {
		return newKey
	}
	
	return originalKey
}