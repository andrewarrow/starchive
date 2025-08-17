package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// WriteCookiesFile creates a Netscape format cookies file
func WriteCookiesFile(cookiesStr string) error {
	file, err := os.Create("./cookies.txt")
	if err != nil {
		return fmt.Errorf("failed to create cookies file: %v", err)
	}
	defer file.Close()

	// Write Netscape HTTP Cookie File header
	file.WriteString("# Netscape HTTP Cookie File\n")
	file.WriteString("# This is a generated file! Do not edit.\n\n")

	// Parse and write cookies
	cookies := strings.Split(cookiesStr, "; ")
	for _, cookie := range cookies {
		parts := strings.SplitN(cookie, "=", 2)
		if len(parts) == 2 {
			// Format: domain, domain_specified, path, secure, expiration, name, value
			// Using simplified format for youtube.com
			line := fmt.Sprintf(".youtube.com\tTRUE\t/\tFALSE\t0\t%s\t%s\n", parts[0], parts[1])
			file.WriteString(line)
		}
	}

	return nil
}

// writeCookiesFile is the original working implementation
func writeCookiesFile(cookiesStr string) error {
	file, err := os.Create("./cookies.txt")
	if err != nil {
		return fmt.Errorf("failed to create cookies file: %v", err)
	}
	defer file.Close()

	// Write Netscape HTTP Cookie File header
	file.WriteString("# Netscape HTTP Cookie File\n")
	
	// Parse cookies string and convert to Netscape format
	// Expected format: "name1=value1; name2=value2; ..."
	cookies := strings.Split(cookiesStr, "; ")
	for _, cookie := range cookies {
		parts := strings.SplitN(cookie, "=", 2)
		if len(parts) == 2 {
			name := parts[0]
			value := parts[1]
			// Write in Netscape format: domain, flag, path, secure, expiration, name, value
			fmt.Fprintf(file, ".youtube.com\tTRUE\t/\tFALSE\t0\t%s\t%s\n", name, value)
		}
	}
	
	return nil
}

// SetupRoutes configures HTTP routes for the web server
func SetupRoutes(downloadQueue interface{}) {
	http.HandleFunc("/api/download", func(w http.ResponseWriter, r *http.Request) {
		handleDownload(w, r, downloadQueue)
	})
	http.HandleFunc("/api/cookies", handleSetCookies)
	http.HandleFunc("/youtube", func(w http.ResponseWriter, r *http.Request) {
		handleYouTube(w, r, downloadQueue)
	})
	http.HandleFunc("/get-txt", func(w http.ResponseWriter, r *http.Request) {
		handleGetTxt(w, r, downloadQueue)
	})
	http.HandleFunc("/data", handleData)
	http.HandleFunc("/", handleStatic)
}

func handleDownload(w http.ResponseWriter, r *http.Request, downloadQueue interface{}) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		URL string `json:"url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	// Add to download queue (implementation depends on queue structure)
	fmt.Printf("Download requested: %s\n", req.URL)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "queued",
		"url":    req.URL,
	})
}

func handleSetCookies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	var req struct {
		Cookies string `json:"cookies"`
	}

	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Cookies == "" {
		http.Error(w, "Cookies are required", http.StatusBadRequest)
		return
	}

	if err := WriteCookiesFile(req.Cookies); err != nil {
		http.Error(w, fmt.Sprintf("Error writing cookies file: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Println("Cookies file updated successfully")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"message": "Cookies updated successfully",
	})
}

func handleYouTube(w http.ResponseWriter, r *http.Request, downloadQueue interface{}) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var jsonData map[string]interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		fmt.Printf("Invalid JSON received: %s\n", string(body))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	fmt.Printf("JSON received: %+v\n", jsonData)

	id, ok := jsonData["videoId"].(string)
	if !ok {
		http.Error(w, "Missing or invalid 'videoId' field", http.StatusBadRequest)
		return
	}

	// Handle cookies if provided - support both string and array formats
	if cookies, ok := jsonData["cookies"].(string); ok && cookies != "" {
		if err := writeCookiesFile(cookies); err != nil {
			fmt.Printf("Warning: failed to write cookies: %v\n", err)
		}
	} else if cookiesArray, ok := jsonData["cookies"].([]interface{}); ok && len(cookiesArray) > 0 {
		// Convert array format to string format
		cookiesStr := ""
		for _, cookie := range cookiesArray {
			if c, ok := cookie.(map[string]interface{}); ok {
				name, nameOk := c["name"].(string)
				value, valueOk := c["value"].(string)
				if nameOk && valueOk && name != "" && value != "" {
					if cookiesStr != "" {
						cookiesStr += "; "
					}
					cookiesStr += name + "=" + value
				}
			}
		}
		if cookiesStr != "" {
			if err := writeCookiesFile(cookiesStr); err != nil {
				fmt.Printf("Warning: failed to write cookies: %v\n", err)
			}
		}
	}

	// Add to download queue
	if queue, ok := downloadQueue.(*DownloadQueue); ok {
		added := queue.AddToQueue(id)
		if !added {
			fmt.Fprintf(w, "Video %s is already in download queue", id)
			return
		}

		queueLength, isRunning := queue.GetQueueStatus()
		fmt.Fprintf(w, "Video %s added to download queue. Queue length: %d, Processing: %t", id, queueLength, isRunning)
	} else {
		http.Error(w, "Download queue not available", http.StatusInternalServerError)
	}
}

func handleData(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[Starchive] GET /data request received\n")
	
	if r.Method != http.MethodGet {
		fmt.Printf("[Starchive] Wrong method for /data: %s\n", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status": "ok",
		"message": "Starchive server is running",
	}
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Printf("[Starchive] Error encoding /data response: %v\n", err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	} else {
		fmt.Printf("[Starchive] /data response sent successfully\n")
	}
}

func handleStatic(w http.ResponseWriter, r *http.Request) {
	// Simple static file serving - in production, use proper static file server
	if r.URL.Path == "/" {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>Starchive</title>
</head>
<body>
    <h1>Starchive Download Server</h1>
    <p>Use the API endpoints to download videos and set cookies.</p>
    <h2>API Endpoints:</h2>
    <ul>
        <li><strong>POST /api/download</strong> - Queue a download</li>
        <li><strong>POST /api/cookies</strong> - Set cookies</li>
    </ul>
</body>
</html>
		`))
		return
	}
	
	http.NotFound(w, r)
}

func handleGetTxt(w http.ResponseWriter, r *http.Request, downloadQueue interface{}) {
	fmt.Printf("[Starchive] GET /get-txt request received\n")
	
	if r.Method != http.MethodGet {
		fmt.Printf("[Starchive] Wrong method: %s\n", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	videoId := r.URL.Query().Get("id")
	fmt.Printf("[Starchive] Requested video ID: %s\n", videoId)
	
	if videoId == "" {
		fmt.Printf("[Starchive] No video ID provided\n")
		http.Error(w, "Video ID is required", http.StatusBadRequest)
		return
	}

	txtFilePath := fmt.Sprintf("./data/%s.txt", videoId)
	fmt.Printf("[Starchive] Checking for txt file at: %s\n", txtFilePath)
	
	if _, err := os.Stat(txtFilePath); err == nil {
		fmt.Printf("[Starchive] Txt file exists, serving content\n")
		content, err := os.ReadFile(txtFilePath)
		if err != nil {
			fmt.Printf("[Starchive] Error reading txt file: %v\n", err)
			http.Error(w, "Error reading txt file", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"hasContent": true,
			"content":    string(content),
			"videoId":    videoId,
		}
		json.NewEncoder(w).Encode(response)
		fmt.Printf("[Starchive] Served %d bytes for video %s\n", len(content), videoId)
		return
	}

	fmt.Printf("[Starchive] Txt file not found, attempting to queue download\n")
	
	if queue, ok := downloadQueue.(*DownloadQueue); ok {
		added := queue.AddToQueue(videoId)
		w.Header().Set("Content-Type", "application/json")
		if added {
			fmt.Printf("[Starchive] Added video %s to download queue\n", videoId)
			response := map[string]interface{}{
				"hasContent": false,
				"message":    "Download started",
				"videoId":    videoId,
			}
			json.NewEncoder(w).Encode(response)
		} else {
			fmt.Printf("[Starchive] Video %s already in queue\n", videoId)
			response := map[string]interface{}{
				"hasContent": false,
				"message":    "Already in download queue",
				"videoId":    videoId,
			}
			json.NewEncoder(w).Encode(response)
		}
	} else {
		fmt.Printf("[Starchive] Download queue not available\n")
		http.Error(w, "Download queue not available", http.StatusInternalServerError)
	}
}