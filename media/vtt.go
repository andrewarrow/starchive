package media

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// ParseVttFile parses a WebVTT subtitle file and extracts text content
func ParseVttFile(filename, id string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Create output file
	outputPath := fmt.Sprintf("data/%s.txt", id)
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outputFile.Close()

	scanner := bufio.NewScanner(file)
	inTextBlock := false
	
	// Regular expression to match timestamp lines
	timestampRegex := regexp.MustCompile(`^\d{2}:\d{2}:\d{2}\.\d{3} --> \d{2}:\d{2}:\d{2}\.\d{3}`)
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and WEBVTT header
		if line == "" || strings.HasPrefix(line, "WEBVTT") {
			inTextBlock = false
			continue
		}
		
		// Skip timestamp lines
		if timestampRegex.MatchString(line) {
			inTextBlock = true
			continue
		}
		
		// Skip cue settings (lines with positioning info)
		if strings.Contains(line, "align:") || strings.Contains(line, "position:") {
			continue
		}
		
		// Process text content
		if inTextBlock && line != "" {
			// Remove HTML tags and formatting
			cleanLine := removeHTMLTags(line)
			cleanLine = strings.TrimSpace(cleanLine)
			
			if cleanLine != "" {
				outputFile.WriteString(cleanLine + "\n")
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %v", err)
	}

	fmt.Printf("VTT file parsed successfully. Text saved to: %s\n", outputPath)
	return nil
}

// removeHTMLTags removes HTML tags and entities from text
func removeHTMLTags(text string) string {
	// Remove HTML tags
	tagRegex := regexp.MustCompile(`<[^>]*>`)
	text = tagRegex.ReplaceAllString(text, "")
	
	// Replace common HTML entities
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	
	// Remove extra whitespace
	spaceRegex := regexp.MustCompile(`\s+`)
	text = spaceRegex.ReplaceAllString(text, " ")
	
	return text
}

// HandleVttCommand processes VTT files for subtitle extraction
func HandleVttCommand(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: starchive vtt <vtt_file> <id>")
		fmt.Println("Example: starchive vtt video.en.vtt abc123")
		fmt.Println("This will parse the VTT file and extract text to data/<id>.txt")
		os.Exit(1)
	}

	vttFile := args[0]
	id := args[1]

	if _, err := os.Stat(vttFile); os.IsNotExist(err) {
		fmt.Printf("Error: VTT file %s does not exist\n", vttFile)
		os.Exit(1)
	}

	if err := ParseVttFile(vttFile, id); err != nil {
		fmt.Printf("Error parsing VTT file: %v\n", err)
		os.Exit(1)
	}
}