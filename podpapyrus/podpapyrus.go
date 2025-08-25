package podpapyrus

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"starchive/media"
)

var (
	processingVideos = make(map[string]bool)
	processingMutex  sync.RWMutex
)

// const BasePath = "./data/podpapyrus"
const BasePath = "../andrewarrow.dev/podpapyrus"

type Config struct {
	BasePath string
}

type ProcessingResult struct {
	HasContent bool   `json:"hasContent"`
	Content    string `json:"content,omitempty"`
	Message    string `json:"message,omitempty"`
	VideoId    string `json:"videoId"`
}

func StripHTMLTags(html string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(html, " ")

	// Clean up extra whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

func ExtractShortSummary(htmlSummary string, wordLimit int) string {
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

func ProcessTranscriptText(rawText string) []template.HTML {
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

func ProcessVideo(videoId string, basePath string) (*ProcessingResult, error) {
	fmt.Printf("[Podpapyrus] Processing video ID: %s\n", videoId)

	// Check if HTML file already exists
	htmlFilePath := filepath.Join(basePath, "summaries", videoId+".html")
	if _, err := os.Stat(htmlFilePath); err == nil {
		fmt.Printf("[Podpapyrus] HTML file exists, serving content\n")
		content, err := os.ReadFile(htmlFilePath)
		if err != nil {
			return nil, fmt.Errorf("error reading HTML file: %v", err)
		}
		return &ProcessingResult{
			HasContent: true,
			Content:    string(content),
			VideoId:    videoId,
		}, nil
	}

	// Check if transcript (.txt) file exists for green highlight
	txtFilePath := fmt.Sprintf("./data/%s.txt", videoId)
	if _, err := os.Stat(txtFilePath); err == nil {
		fmt.Printf("[Podpapyrus] Transcript exists for %s, checking if processing needed\n", videoId)

		// Check if this video is already being processed
		processingMutex.RLock()
		isProcessing := processingVideos[videoId]
		processingMutex.RUnlock()

		if isProcessing {
			// Transcript exists but HTML is being processed - show green highlight with transcript
			fmt.Printf("[Podpapyrus] Video %s is being processed but transcript exists, showing green\n", videoId)
			content, err := os.ReadFile(txtFilePath)
			if err == nil {
				return &ProcessingResult{
					HasContent: true,
					Content:    string(content),
					VideoId:    videoId,
				}, nil
			}
		}
	}

	// Check if this video is already being processed (for when transcript doesn't exist)
	processingMutex.Lock()
	if processingVideos[videoId] {
		processingMutex.Unlock()
		fmt.Printf("[Podpapyrus] Video %s is already being processed, returning waiting message\n", videoId)
		return &ProcessingResult{
			HasContent: false,
			Message:    "Video is currently being processed",
			VideoId:    videoId,
		}, nil
	}
	// Mark this video as being processed
	processingVideos[videoId] = true
	processingMutex.Unlock()

	// Ensure we clean up the processing flag when done
	defer func() {
		processingMutex.Lock()
		delete(processingVideos, videoId)
		processingMutex.Unlock()
		fmt.Printf("[Podpapyrus] Finished processing video %s\n", videoId)
	}()

	fmt.Printf("[Podpapyrus] HTML file not found, running podpapyrus processing\n")

	// Get YouTube cookie file
	cookieFile := media.GetCookieFile("youtube")

	// Download thumbnail and subtitles
	fmt.Printf("[Podpapyrus] Downloading thumbnail and subtitles for %s...\n", videoId)

	if err := media.DownloadYouTubeThumbnail(videoId, cookieFile); err != nil {
		return nil, fmt.Errorf("error downloading thumbnail: %v", err)
	}

	if err := media.DownloadYouTubeSubtitles(videoId, cookieFile); err != nil {
		return nil, fmt.Errorf("error downloading subtitles: %v", err)
	}

	fmt.Printf("[Podpapyrus] Successfully downloaded thumbnail and subtitles for %s\n", videoId)

	// Check if txt file was created by VTT parsing
	txtPath := fmt.Sprintf("./data/%s.txt", videoId)
	if _, err := os.Stat(txtPath); err != nil {
		return nil, fmt.Errorf("text file was not created from VTT parsing: %v", err)
	}
	fmt.Printf("[Podpapyrus] Text file created: %s\n", txtPath)

	// Read the text file content
	textContent, err := os.ReadFile(txtPath)
	if err != nil {
		return nil, fmt.Errorf("error reading text file: %v", err)
	}

	// Download and read JSON metadata to get title
	jsonPath := fmt.Sprintf("./data/%s.json", videoId)
	if err := media.DownloadYouTubeJSON(videoId, cookieFile); err != nil {
		return nil, fmt.Errorf("error downloading JSON metadata: %v", err)
	}

	jsonContent, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, fmt.Errorf("error reading JSON metadata: %v", err)
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(jsonContent, &metadata); err != nil {
		return nil, fmt.Errorf("error parsing JSON metadata: %v", err)
	}

	title, ok := metadata["title"].(string)
	if !ok {
		return nil, fmt.Errorf("could not extract title from metadata")
	}

	// Generate bullets using claude CLI
	fmt.Printf("[Podpapyrus] Generating bullets using claude CLI...\n")
	bulletsCmd := exec.Command("claude", "-p", "list the top 18 important things from all this text and return the response as clean HTML using <ul> and <li> tags. Do not include <html>, <head>, or <body> tags, just the content: "+string(textContent))
	bulletsCmd.Dir = "/Users/aa"
	bulletsOutput, err := bulletsCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error generating bullets: %v", err)
	}
	bullets := string(bulletsOutput)

	// Process the text content into paragraphs
	paragraphs := ProcessTranscriptText(string(textContent))

	// Extract short summary (40-50 words)
	shortSummary := ExtractShortSummary(bullets, 45)

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
		Id:         videoId,
		Text:       string(textContent),
		Summary:    "",
		Short:      template.HTML(StripHTMLTags(shortSummary)),
		Bullets:    template.HTML(bullets),
		Paragraphs: paragraphs,
	}

	// Parse template
	tmpl, err := template.ParseFiles("./templates/id.html")
	if err != nil {
		return nil, fmt.Errorf("error parsing template: %v", err)
	}

	// Ensure output directory exists
	outputDir := filepath.Join(basePath, "summaries")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("error creating output directory: %v", err)
	}

	// Create output file
	outputPath := filepath.Join(outputDir, videoId+".html")
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return nil, fmt.Errorf("error creating output file: %v", err)
	}
	defer outputFile.Close()

	// Execute template
	if err := tmpl.Execute(outputFile, templateData); err != nil {
		return nil, fmt.Errorf("error executing template: %v", err)
	}

	// Superimpose podpapyrus.png onto thumbnail image and save to images directory
	imgSourcePath := fmt.Sprintf("./data/%s.jpg", videoId)
	imgDir := filepath.Join(basePath, "images")
	if err := os.MkdirAll(imgDir, 0755); err != nil {
		return nil, fmt.Errorf("error creating images directory: %v", err)
	}

	imgDestPath := filepath.Join(imgDir, videoId+".jpg")
	podpapyrusPath := "./firefox/podpapyrus.png"

	// Use ImageMagick composite to superimpose podpapyrus.png onto the thumbnail
	// Make podpapyrus.png 4x larger and center it
	cmd := exec.Command("composite", "-resize", "400%", "-gravity", "center", podpapyrusPath, imgSourcePath, imgDestPath)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("error compositing images with ImageMagick: %v", err)
	}

	fmt.Printf("[Podpapyrus] Successfully created HTML file: %s\n", outputPath)
	fmt.Printf("[Podpapyrus] Successfully composited image with podpapyrus overlay to: %s\n", imgDestPath)

	// Generate item HTML using item.html template
	fmt.Printf("[Podpapyrus] Generating item HTML for summaries list...\n")
	itemTmpl, err := template.ParseFiles("./templates/item.html")
	if err != nil {
		return nil, fmt.Errorf("error parsing item template: %v", err)
	}

	// Use a string builder to capture the item HTML
	var itemHTML strings.Builder
	if err := itemTmpl.Execute(&itemHTML, templateData); err != nil {
		return nil, fmt.Errorf("error executing item template: %v", err)
	}

	// Read the current summaries/index.html
	indexPath := filepath.Join(basePath, "summaries", "index.html")
	indexContent, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, fmt.Errorf("error reading index file: %v", err)
	}

	// Find the "<!-- top of list -->" marker and insert the new item after it
	marker := "<!-- top of list -->"
	endMarker := "<!-- end of a tags -->"
	indexStr := string(indexContent)
	markerIndex := strings.Index(indexStr, marker)
	if markerIndex == -1 {
		return nil, fmt.Errorf("could not find '<!-- top of list -->' marker in index.html")
	}

	endMarkerIndex := strings.Index(indexStr, endMarker)
	if endMarkerIndex == -1 {
		return nil, fmt.Errorf("could not find '<!-- end of a tags -->' marker in index.html")
	}

	// Find the last </a> tag before the end marker to remove the bottom item
	beforeEndMarker := indexStr[:endMarkerIndex]
	lastATagIndex := strings.LastIndex(beforeEndMarker, "</a>")
	if lastATagIndex == -1 {
		return nil, fmt.Errorf("could not find last </a> tag before end marker")
	}

	// Find the start of this last <a> tag by working backwards
	// Look for the opening <a tag that corresponds to this closing tag
	aTagPattern := `<a href="[^"]*" class="block bg-gray-800`
	aTagRe := regexp.MustCompile(aTagPattern)
	allMatches := aTagRe.FindAllStringIndex(beforeEndMarker, -1)
	if len(allMatches) == 0 {
		return nil, fmt.Errorf("could not find any <a> tags before end marker")
	}

	// Get the last match (which should be the start of the last item)
	lastATagStartIndex := allMatches[len(allMatches)-1][0]

	// Remove the last item by excluding it from the content
	beforeLastItem := indexStr[:lastATagStartIndex]
	afterEndMarker := indexStr[endMarkerIndex:]

	// Insert the new item HTML after the top marker
	beforeMarker := beforeLastItem[:markerIndex+len(marker)]
	afterMarker := beforeLastItem[markerIndex+len(marker):]
	// Ensure we include the end marker in the reconstruction
	endMarkerWithNewline := endMarker + "\n"
	newIndexContent := beforeMarker + "\n          " + itemHTML.String() + afterMarker + endMarkerWithNewline + afterEndMarker

	// Write the updated content back to index.html
	if err := os.WriteFile(indexPath, []byte(newIndexContent), 0644); err != nil {
		return nil, fmt.Errorf("error writing updated index file: %v", err)
	}

	fmt.Printf("[Podpapyrus] Successfully updated summaries index with new item\n")

	// Generate homepage HTML using homepage.html template
	fmt.Printf("[Podpapyrus] Generating homepage HTML for main page...\n")
	homepageTmpl, err := template.ParseFiles("./templates/homepage.html")
	if err != nil {
		return nil, fmt.Errorf("error parsing homepage template: %v", err)
	}

	// Use a string builder to capture the homepage HTML
	var homepageHTML strings.Builder
	if err := homepageTmpl.Execute(&homepageHTML, templateData); err != nil {
		return nil, fmt.Errorf("error executing homepage template: %v", err)
	}

	fmt.Printf("[Podpapyrus] Successfully generated homepage HTML\n")

	// Read the current index.html file
	homepageIndexPath := filepath.Join(basePath, "index.html")
	homepageIndexContent, err := os.ReadFile(homepageIndexPath)
	if err != nil {
		return nil, fmt.Errorf("error reading homepage index file: %v", err)
	}

	// Find the "<!-- recent 3 -->" marker and the grid div after it
	homepageMarker := "<!-- recent 3 -->"
	gridStart := "<div class=\"grid md:grid-cols-3 gap-8 mb-12\">"
	homepageIndexStr := string(homepageIndexContent)
	homepageMarkerIndex := strings.Index(homepageIndexStr, homepageMarker)
	if homepageMarkerIndex == -1 {
		return nil, fmt.Errorf("could not find '<!-- recent 3 -->' marker in homepage index.html")
	}

	gridStartIndex := strings.Index(homepageIndexStr[homepageMarkerIndex:], gridStart)
	if gridStartIndex == -1 {
		return nil, fmt.Errorf("could not find grid div after marker in homepage index.html")
	}
	gridStartIndex += homepageMarkerIndex + len(gridStart)

	// Parse the existing grid items to maintain exactly 3 items
	beforeGrid := homepageIndexStr[:gridStartIndex]
	afterGridStart := homepageIndexStr[gridStartIndex:]

	// Find the end of the grid div to extract existing items
	gridEndTag := "</div>"
	gridEndIndex := strings.Index(afterGridStart, gridEndTag)
	if gridEndIndex == -1 {
		return nil, fmt.Errorf("could not find end of grid div in homepage index.html")
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
		return nil, fmt.Errorf("error writing updated homepage index file: %v", err)
	}

	fmt.Printf("[Podpapyrus] Successfully updated homepage index with new item\n")

	// Read the created HTML file and return it
	htmlContent, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("error reading created HTML file: %v", err)
	}

	return &ProcessingResult{
		HasContent: true,
		Content:    string(htmlContent),
		VideoId:    videoId,
	}, nil
}

func ProcessCommandLine(videoId string, basePath string) error {
	_, err := ProcessVideo(videoId, basePath)
	return err
}
