package media

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func parseVttFile(filename, id string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Create output file
	outputPath := fmt.Sprintf("./data/%s.txt", id)
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outputFile.Close()

	scanner := bufio.NewScanner(file)
	var lastLine string

	// Skip WebVTT header
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "WEBVTT" || line == "" {
			continue
		}
		break
	}

	// Process subtitle entries
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip timing lines (contain -->)
		if strings.Contains(line, "-->") {
			continue
		}

		// Skip empty lines and sequence numbers
		if line == "" || (len(line) < 10 && !strings.Contains(line, " ")) {
			continue
		}

		// Remove HTML-like tags and positioning data
		line = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(line, "")
		line = regexp.MustCompile(`\{[^}]*\}`).ReplaceAllString(line, "")

		cleanLine := strings.TrimSpace(line)
		if cleanLine != "" && cleanLine != lastLine {
			_, err := fmt.Fprintln(outputFile, cleanLine)
			if err != nil {
				return fmt.Errorf("failed to write to output file: %v", err)
			}
			lastLine = cleanLine
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %v", err)
	}

	return nil
}