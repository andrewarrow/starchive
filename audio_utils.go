package main

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

func getAudioDuration(filePath string) (float64, error) {
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

func waitForKeyPress() {
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

func getAudioFilename(id, audioType string) string {
	switch audioType {
	case "V", "vocal", "vocals":
		return fmt.Sprintf("./data/%s_(Vocals)_UVR_MDXNET_Main.wav", id)
	case "I", "instrumental", "instrumentals":
		return fmt.Sprintf("./data/%s_(Instrumental)_UVR_MDXNET_Main.wav", id)
	default:
		return fmt.Sprintf("./data/%s.wav", id)
	}
}

func hasVocalFile(id string) bool {
	vocalPath := fmt.Sprintf("./data/%s_(Vocals)_UVR_MDXNET_Main.wav", id)
	_, err := os.Stat(vocalPath)
	return !os.IsNotExist(err)
}

func hasInstrumentalFile(id string) bool {
	instrumentalPath := fmt.Sprintf("./data/%s_(Instrumental)_UVR_MDXNET_Main.wav", id)
	_, err := os.Stat(instrumentalPath)
	return !os.IsNotExist(err)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}