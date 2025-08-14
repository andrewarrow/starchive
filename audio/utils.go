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