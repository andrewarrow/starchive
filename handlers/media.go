package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"starchive/media"
	"starchive/util"
)

// const podpapyrusBasePath = "../andrewarrow.dev/podpapyrus"
const podpapyrusBasePath = "./data/podpapyrus"

func stripHTMLTags(html string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(html, " ")

	// Clean up extra whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

func extractShortSummary(htmlSummary string, wordLimit int) string {
	// Remove HTML tags temporarily to count words
	re := regexp.MustCompile(`<[^>]*>`)
	textOnly := re.ReplaceAllString(htmlSummary, " ")

	// Split into words
	words := regexp.MustCompile(`\s+`).Split(strings.TrimSpace(textOnly), -1)

	if len(words) <= wordLimit {
		return htmlSummary
	}

	// Find the position in the original HTML where we should cut
	wordCount := 0
	var result strings.Builder
	inTag := false

	for i, char := range htmlSummary {
		result.WriteRune(char)

		if char == '<' {
			inTag = true
		} else if char == '>' {
			inTag = false
		} else if !inTag && (char == ' ' || char == '\t' || char == '\n') {
			// We're at a word boundary, check if we've reached our limit
			if wordCount >= wordLimit {
				// Look ahead to see if we're in the middle of a tag
				remaining := htmlSummary[i+1:]
				if strings.Contains(remaining, ">") && strings.Index(remaining, ">") < strings.Index(remaining, "<") {
					// We're in the middle of a tag, continue until we close it
					for j := i + 1; j < len(htmlSummary); j++ {
						result.WriteRune(rune(htmlSummary[j]))
						if htmlSummary[j] == '>' {
							break
						}
					}
				}
				break
			}

			// Count the word we just passed
			if i > 0 && htmlSummary[i-1] != ' ' && htmlSummary[i-1] != '\t' && htmlSummary[i-1] != '\n' {
				wordCount++
			}
		}
	}

	return result.String()
}

func processTranscriptText(rawText string) []template.HTML {
	// Remove "Language: en" prefix if present
	text := regexp.MustCompile(`^Language: \w+\s*`).ReplaceAllString(rawText, "")

	// Replace [&nbsp;__&nbsp;] with [censored]
	text = regexp.MustCompile(`\[&nbsp;__&nbsp;\]`).ReplaceAllString(text, "[censored]")

	// Clean up extra whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	// Split into sentences and group into paragraphs
	sentences := regexp.MustCompile(`[.!?]+\s+`).Split(text, -1)

	var paragraphs []template.HTML
	var currentParagraph []string
	sentenceCount := 0

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}

		currentParagraph = append(currentParagraph, sentence)
		sentenceCount++

		// Create a new paragraph every 4-6 sentences
		if sentenceCount >= 4 && len(currentParagraph) > 0 {
			paragraphs = append(paragraphs, template.HTML(strings.Join(currentParagraph, ". ")+"."))
			currentParagraph = []string{}
			sentenceCount = 0
		}
	}

	// Add any remaining sentences as a final paragraph
	if len(currentParagraph) > 0 {
		paragraphs = append(paragraphs, template.HTML(strings.Join(currentParagraph, ". ")+"."))
	}

	return paragraphs
}

func HandleDl() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: starchive dl <id_or_url>")
		fmt.Println("Examples:")
		fmt.Println("  starchive dl abc123")
		fmt.Println("  starchive dl https://www.youtube.com/watch?v=abc123")
		fmt.Println("  starchive dl https://www.instagram.com/p/DMxMgnvhwmK/")
		fmt.Println("  starchive dl https://www.instagram.com/reels/DMxMgnvhwmK/")
		os.Exit(1)
	}

	input := os.Args[2]

	// Extract ID and platform from input
	id, platform := media.ParseVideoInput(input)
	if id == "" {
		fmt.Printf("Error: Could not extract ID from input: %s\n", input)
		os.Exit(1)
	}

	fmt.Printf("Detected platform: %s, ID: %s\n", platform, id)

	_, err := media.DownloadVideo(id, platform)
	if err != nil {
		fmt.Printf("Error downloading video: %v\n", err)
		os.Exit(1)
	}
}

func HandleExternal() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: starchive external <file_path>")
		fmt.Println("Example: starchive external ~/Documents/cd_audio_from_gnr_lies.wav")
		fmt.Println("This will copy the external file into the data directory and create a metadata JSON file")
		os.Exit(1)
	}

	sourceFilePath := os.Args[2]

	// Expand tilde to home directory
	if strings.HasPrefix(sourceFilePath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("Error getting home directory: %v\n", err)
			os.Exit(1)
		}
		sourceFilePath = filepath.Join(homeDir, sourceFilePath[2:])
	}

	// Check if source file exists
	if _, err := os.Stat(sourceFilePath); os.IsNotExist(err) {
		fmt.Printf("Error: Source file %s does not exist\n", sourceFilePath)
		os.Exit(1)
	}

	// Get filename without extension for title and ID
	filename := filepath.Base(sourceFilePath)
	title := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Generate ID from first 11 characters of filename (without extension)
	id := title
	if len(id) > 11 {
		id = id[:11]
	}

	fmt.Printf("Generated ID: %s\n", id)

	// Determine file extension
	ext := strings.ToLower(filepath.Ext(sourceFilePath))

	// Copy file to data directory
	dataDir := "./data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		fmt.Printf("Error creating data directory: %v\n", err)
		os.Exit(1)
	}

	destPath := filepath.Join(dataDir, id+ext)

	// Check if destination already exists
	if _, err := os.Stat(destPath); err == nil {
		fmt.Printf("File with ID %s already exists in data directory\n", id)
		os.Exit(1)
	}

	// Copy file
	sourceFile, err := os.Open(sourceFilePath)
	if err != nil {
		fmt.Printf("Error opening source file: %v\n", err)
		os.Exit(1)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		fmt.Printf("Error creating destination file: %v\n", err)
		os.Exit(1)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		fmt.Printf("Error copying file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Copied file to: %s\n", destPath)

	// Create JSON metadata file
	metadata := map[string]interface{}{
		"id":            id,
		"title":         title,
		"source":        "external",
		"original_path": sourceFilePath,
		"imported_at":   time.Now().Format(time.RFC3339),
		"filename":      filename,
	}

	jsonPath := filepath.Join(dataDir, id+".json")
	jsonData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		fmt.Printf("Error creating JSON metadata: %v\n", err)
		os.Exit(1)
	}

	err = os.WriteFile(jsonPath, jsonData, 0644)
	if err != nil {
		fmt.Printf("Error writing JSON file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created metadata file: %s\n", jsonPath)

	// Add to database
	db, err := util.InitDatabase()
	if err != nil {
		fmt.Printf("Warning: Could not initialize database: %v\n", err)
	} else {
		defer db.Close()

		videoMetadata := util.VideoMetadata{
			ID:           id,
			Title:        &title,
			LastModified: time.Now(),
			VocalDone:    false,
		}

		if err := db.SaveMetadata(&videoMetadata); err != nil {
			fmt.Printf("Warning: Could not save to database: %v\n", err)
		} else {
			fmt.Printf("Added to database with ID: %s\n", id)
		}
	}

	fmt.Printf("\nExternal file successfully imported!\n")
	fmt.Printf("ID: %s\n", id)
	fmt.Printf("Title: %s\n", title)
	fmt.Printf("File: %s\n", destPath)
}

func HandleUl() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: starchive ul <id>")
		fmt.Println("Example: starchive ul abc123")
		fmt.Println("This will upload the mp4 file with the given ID to YouTube")
		os.Exit(1)
	}

	id := os.Args[2]

	mp4Path := fmt.Sprintf("./data/%s.mp4", id)
	if _, err := os.Stat(mp4Path); os.IsNotExist(err) {
		fmt.Printf("Error: MP4 file %s does not exist\n", mp4Path)
		os.Exit(1)
	}

	uploadScript := "./media/upload_to_youtube.py"
	if _, err := os.Stat(uploadScript); os.IsNotExist(err) {
		fmt.Printf("Error: Upload script %s does not exist\n", uploadScript)
		os.Exit(1)
	}

	fmt.Printf("Uploading %s to YouTube...\n", mp4Path)

	absMP4Path, err := filepath.Abs(mp4Path)
	if err != nil {
		fmt.Printf("Error getting absolute path for mp4: %v\n", err)
		os.Exit(1)
	}

	cmd := exec.Command("python3", "upload_to_youtube.py", absMP4Path, "--title", id)
	cmd.Dir = "./media"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		fmt.Printf("Error uploading to YouTube: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully uploaded %s to YouTube\n", id)
}

func HandleSmall() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: starchive small <id>")
		fmt.Println("Example: starchive small abc123")
		fmt.Println("This will create a small optimized video from data/id.mp4")
		os.Exit(1)
	}

	id := os.Args[2]
	inputPath := fmt.Sprintf("./data/%s.mp4", id)

	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		fmt.Printf("Error: Input file %s does not exist\n", inputPath)
		os.Exit(1)
	}

	fmt.Printf("Creating small optimized video for %s...\n", id)

	// Step 1: Prepare video with specific settings
	fmt.Println("Step 1: Preparing video...")
	prepCmd := exec.Command("ffmpeg", "-i", inputPath,
		"-vf", "fps=24,scale=242:428:flags=lanczos:force_original_aspect_ratio=decrease,pad=242:428:(ow-iw)/2:(oh-ih)/2,hqdn3d=1.5:1.5:6:6",
		"-c:v", "h264_videotoolbox", "-b:v", "2000k", "-maxrate", "2000k", "-bufsize", "4000k",
		"-c:a", "aac", "-b:a", "96k", "-ar", "48000", "-ac", "1",
		"-movflags", "+faststart",
		"tmp_prep.mp4")

	fmt.Printf("Running: %s\n", prepCmd.String())
	prepCmd.Stdout = os.Stdout
	prepCmd.Stderr = os.Stderr

	err := prepCmd.Run()
	if err != nil {
		fmt.Printf("Error in preparation step: %v\n", err)
		os.Exit(1)
	}

	// Step 2: First pass (no audio)
	fmt.Println("\nStep 2: First pass encoding (no audio)...")
	pass1Cmd := exec.Command("ffmpeg", "-y", "-i", "tmp_prep.mp4",
		"-c:v", "libx264", "-b:v", "150k", "-maxrate", "150k", "-bufsize", "300k",
		"-pix_fmt", "yuv420p", "-profile:v", "high", "-level", "3.1",
		"-x264-params", "aq-mode=2:aq-strength=1.0:rc-lookahead=40:ref=4:keyint=120:scenecut=40:deblock=1,1:me=umh:subme=8",
		"-an", "-f", "mp4", "/dev/null")

	fmt.Printf("Running: %s\n", pass1Cmd.String())
	pass1Cmd.Stdout = os.Stdout
	pass1Cmd.Stderr = os.Stderr

	err = pass1Cmd.Run()
	if err != nil {
		fmt.Printf("Error in first pass: %v\n", err)
		os.Exit(1)
	}

	// Step 3: Second pass (add audio)
	fmt.Println("\nStep 3: Second pass encoding (with audio)...")
	outputPath := fmt.Sprintf("./data/%s-small.mp4", id)
	pass2Cmd := exec.Command("ffmpeg", "-i", "tmp_prep.mp4",
		"-c:v", "libx264", "-b:v", "150k", "-maxrate", "150k", "-bufsize", "300k",
		"-pix_fmt", "yuv420p", "-profile:v", "high", "-level", "3.1",
		"-x264-params", "aq-mode=2:aq-strength=1.0:rc-lookahead=40:ref=4:keyint=120:scenecut=40:deblock=1,1:me=umh:subme=8",
		"-c:a", "aac", "-b:a", "32k", "-ar", "48000", "-ac", "1",
		"-movflags", "+faststart",
		outputPath)

	fmt.Printf("Running: %s\n", pass2Cmd.String())
	pass2Cmd.Stdout = os.Stdout
	pass2Cmd.Stderr = os.Stderr

	err = pass2Cmd.Run()
	if err != nil {
		fmt.Printf("Error in second pass: %v\n", err)
		os.Exit(1)
	}

	// Clean up temporary file
	if err := os.Remove("tmp_prep.mp4"); err != nil {
		fmt.Printf("Warning: Could not remove temporary file tmp_prep.mp4: %v\n", err)
	}

	fmt.Printf("\nSuccessfully created small optimized video: %s\n", outputPath)
}

func HandlePodpapyrus() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: starchive podpapyrus <id_or_url>")
		fmt.Println("Examples:")
		fmt.Println("  starchive podpapyrus abc123")
		fmt.Println("  starchive podpapyrus https://www.youtube.com/watch?v=abc123")
		fmt.Println("This downloads thumbnail and VTT subtitle files, then creates a text file")
		os.Exit(1)
	}

	input := os.Args[2]

	// Extract ID and platform from input
	id, platform := media.ParseVideoInput(input)
	if id == "" {
		fmt.Printf("Error: Could not extract ID from input: %s\n", input)
		os.Exit(1)
	}

	fmt.Printf("Detected platform: %s, ID: %s\n", platform, id)

	// Currently only support YouTube
	if platform != "youtube" {
		fmt.Printf("Error: podpapyrus currently only supports YouTube videos\n")
		os.Exit(1)
	}

	// Get YouTube cookie file
	cookieFile := media.GetCookieFile(platform)

	// Download thumbnail and subtitles only
	fmt.Printf("Downloading thumbnail and subtitles for %s...\n", id)

	if err := media.DownloadYouTubeThumbnail(id, cookieFile); err != nil {
		fmt.Printf("Error downloading thumbnail: %v\n", err)
		os.Exit(1)
	}

	if err := media.DownloadYouTubeSubtitles(id, cookieFile); err != nil {
		fmt.Printf("Error downloading subtitles: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully downloaded thumbnail and subtitles for %s\n", id)

	// Check if txt file was created by VTT parsing
	txtPath := fmt.Sprintf("./data/%s.txt", id)
	if _, err := os.Stat(txtPath); err != nil {
		fmt.Printf("Error: Text file was not created from VTT parsing: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Text file created: %s\n", txtPath)

	// Read the text file content
	textContent, err := os.ReadFile(txtPath)
	if err != nil {
		fmt.Printf("Error reading text file: %v\n", err)
		os.Exit(1)
	}

	// Download and read JSON metadata to get title
	jsonPath := fmt.Sprintf("./data/%s.json", id)
	if err := media.DownloadYouTubeJSON(id, cookieFile); err != nil {
		fmt.Printf("Error downloading JSON metadata: %v\n", err)
		os.Exit(1)
	}

	jsonContent, err := os.ReadFile(jsonPath)
	if err != nil {
		fmt.Printf("Error reading JSON metadata: %v\n", err)
		os.Exit(1)
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(jsonContent, &metadata); err != nil {
		fmt.Printf("Error parsing JSON metadata: %v\n", err)
		os.Exit(1)
	}

	title, ok := metadata["title"].(string)
	if !ok {
		fmt.Printf("Error: Could not extract title from metadata\n")
		os.Exit(1)
	}

	// Generate summary using claude CLI
	fmt.Printf("Generating summary using claude CLI...\n")
	summaryCmd := exec.Command("claude", "-p", "summarize this text and return the response as clean HTML with appropriate tags like <p>, <strong>, <em>, etc. Do not include <html>, <head>, or <body> tags, just the content: "+string(textContent))
	summaryOutput, err := summaryCmd.Output()
	if err != nil {
		fmt.Printf("Error generating summary: %v\n", err)
		os.Exit(1)
	}
	summary := string(summaryOutput)

	// Generate bullets using claude CLI
	fmt.Printf("Generating bullets using claude CLI...\n")
	bulletsCmd := exec.Command("claude", "-p", "list the top 18 important things from all this text and return the response as clean HTML using <ul> and <li> tags. Do not include <html>, <head>, or <body> tags, just the content: "+string(textContent))
	bulletsOutput, err := bulletsCmd.Output()
	if err != nil {
		fmt.Printf("Error generating bullets: %v\n", err)
		os.Exit(1)
	}
	bullets := string(bulletsOutput)

	// Process the text content into paragraphs
	paragraphs := processTranscriptText(string(textContent))

	// Extract short summary (40-50 words)
	shortSummary := extractShortSummary(summary, 45)

	// Prepare template data
	templateData := struct {
		Title      string
		Id         string
		Text       string
		Summary    template.HTML
		Short      template.HTML
		Bullets    template.HTML
		Paragraphs []template.HTML
	}{
		Title:      title,
		Id:         id,
		Text:       string(textContent),
		Summary:    template.HTML(summary),
		Short:      template.HTML(stripHTMLTags(shortSummary)),
		Bullets:    template.HTML(bullets),
		Paragraphs: paragraphs,
	}

	// Parse template
	tmpl, err := template.ParseFiles("./templates/id.html")
	if err != nil {
		fmt.Printf("Error parsing template: %v\n", err)
		os.Exit(1)
	}

	// Ensure output directory exists
	outputDir := filepath.Join(podpapyrusBasePath, "summaries")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Create output file
	outputPath := filepath.Join(outputDir, id+".html")
	outputFile, err := os.Create(outputPath)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer outputFile.Close()

	// Execute template
	if err := tmpl.Execute(outputFile, templateData); err != nil {
		fmt.Printf("Error executing template: %v\n", err)
		os.Exit(1)
	}

	// Copy thumbnail image to the images directory
	imgSourcePath := fmt.Sprintf("./data/%s.jpg", id)
	imgDir := filepath.Join(podpapyrusBasePath, "images")
	if err := os.MkdirAll(imgDir, 0755); err != nil {
		fmt.Printf("Error creating images directory: %v\n", err)
		os.Exit(1)
	}

	imgDestPath := filepath.Join(imgDir, id+".jpg")
	sourceFile, err := os.Open(imgSourcePath)
	if err != nil {
		fmt.Printf("Error opening source image: %v\n", err)
		os.Exit(1)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(imgDestPath)
	if err != nil {
		fmt.Printf("Error creating destination image: %v\n", err)
		os.Exit(1)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		fmt.Printf("Error copying image: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully created HTML file: %s\n", outputPath)
	fmt.Printf("Successfully copied image to: %s\n", imgDestPath)

	// Generate item HTML using item.html template
	fmt.Printf("Generating item HTML for summaries list...\n")
	itemTmpl, err := template.ParseFiles("./templates/item.html")
	if err != nil {
		fmt.Printf("Error parsing item template: %v\n", err)
		os.Exit(1)
	}

	// Use a string builder to capture the item HTML
	var itemHTML strings.Builder
	if err := itemTmpl.Execute(&itemHTML, templateData); err != nil {
		fmt.Printf("Error executing item template: %v\n", err)
		os.Exit(1)
	}

	// Read the current summaries/index.html
	indexPath := filepath.Join(podpapyrusBasePath, "summaries", "index.html")
	indexContent, err := os.ReadFile(indexPath)
	if err != nil {
		fmt.Printf("Error reading index file: %v\n", err)
		os.Exit(1)
	}

	// Find the "<!-- top of list -->" marker and insert the new item after it
	marker := "<!-- top of list -->"
	indexStr := string(indexContent)
	markerIndex := strings.Index(indexStr, marker)
	if markerIndex == -1 {
		fmt.Printf("Error: Could not find '<!-- top of list -->' marker in index.html\n")
		os.Exit(1)
	}

	// Insert the new item HTML after the marker
	beforeMarker := indexStr[:markerIndex+len(marker)]
	afterMarker := indexStr[markerIndex+len(marker):]
	newIndexContent := beforeMarker + "\n          " + itemHTML.String() + afterMarker

	// Write the updated content back to index.html
	if err := os.WriteFile(indexPath, []byte(newIndexContent), 0644); err != nil {
		fmt.Printf("Error writing updated index file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully updated summaries index with new item\n")

	// Generate homepage HTML using homepage.html template
	fmt.Printf("Generating homepage HTML for main page...\n")
	homepageTmpl, err := template.ParseFiles("./templates/homepage.html")
	if err != nil {
		fmt.Printf("Error parsing homepage template: %v\n", err)
		os.Exit(1)
	}

	// Use a string builder to capture the homepage HTML
	var homepageHTML strings.Builder
	if err := homepageTmpl.Execute(&homepageHTML, templateData); err != nil {
		fmt.Printf("Error executing homepage template: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully generated homepage HTML\n")

	// Read the current index.html file
	homepageIndexPath := filepath.Join(podpapyrusBasePath, "index.html")
	homepageIndexContent, err := os.ReadFile(homepageIndexPath)
	if err != nil {
		fmt.Printf("Error reading homepage index file: %v\n", err)
		os.Exit(1)
	}

	// Find the "<!-- recent 3 -->" marker and the grid div after it
	homepageMarker := "<!-- recent 3 -->"
	gridStart := "<div class=\"grid md:grid-cols-3 gap-8 mb-12\">"
	homepageIndexStr := string(homepageIndexContent)
	homepageMarkerIndex := strings.Index(homepageIndexStr, homepageMarker)
	if homepageMarkerIndex == -1 {
		fmt.Printf("Error: Could not find '<!-- recent 3 -->' marker in homepage index.html\n")
		os.Exit(1)
	}

	gridStartIndex := strings.Index(homepageIndexStr[homepageMarkerIndex:], gridStart)
	if gridStartIndex == -1 {
		fmt.Printf("Error: Could not find grid div after marker in homepage index.html\n")
		os.Exit(1)
	}
	gridStartIndex += homepageMarkerIndex + len(gridStart)

	// Parse the existing grid items to maintain exactly 3 items
	beforeGrid := homepageIndexStr[:gridStartIndex]
	afterGridStart := homepageIndexStr[gridStartIndex:]

	// Find the end of the grid div to extract existing items
	gridEndTag := "</div>"
	gridEndIndex := strings.Index(afterGridStart, gridEndTag)
	if gridEndIndex == -1 {
		fmt.Printf("Error: Could not find end of grid div in homepage index.html\n")
		os.Exit(1)
	}

	existingGridContent := afterGridStart[:gridEndIndex]
	afterGridEnd := afterGridStart[gridEndIndex:]

	// Parse existing items by looking for <a href="./summaries/ patterns
	itemPattern := `<a href="./summaries/[^"]+\.html"`
	re := regexp.MustCompile(itemPattern)
	matches := re.FindAllStringIndex(existingGridContent, -1)

	var existingItems []string
	if len(matches) > 0 {
		// Extract first 2 items to keep (making room for the new item at the beginning)
		for i, match := range matches {
			if i >= 2 {
				break // Only keep first 2 items
			}

			// Find the end of this item (next <a tag or end of content)
			itemStart := match[0]
			var itemEnd int
			if i+1 < len(matches) {
				itemEnd = matches[i+1][0]
			} else {
				itemEnd = len(existingGridContent)
			}

			// Find the closing </a> tag for this item
			itemContent := existingGridContent[itemStart:itemEnd]
			closingTagIndex := strings.LastIndex(itemContent, "</a>")
			if closingTagIndex != -1 {
				existingItems = append(existingItems, strings.TrimSpace(existingGridContent[itemStart:itemStart+closingTagIndex+4]))
			}
		}
	}

	// Build the new grid content with exactly 3 items: new item + first 2 existing items
	var newGridContent strings.Builder
	newGridContent.WriteString("\n          ")
	newGridContent.WriteString(homepageHTML.String())
	newGridContent.WriteString("\n          ")

	for _, item := range existingItems {
		newGridContent.WriteString("\n          ")
		newGridContent.WriteString(item)
		newGridContent.WriteString("\n          ")
	}

	newHomepageContent := beforeGrid + newGridContent.String() + afterGridEnd

	// Write the updated content back to index.html
	if err := os.WriteFile(homepageIndexPath, []byte(newHomepageContent), 0644); err != nil {
		fmt.Printf("Error writing updated homepage index file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully updated homepage index with new item\n")
}
