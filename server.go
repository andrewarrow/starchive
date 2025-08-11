package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

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

func setupRoutes(downloadQueue *DownloadQueue) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Request received:", r.Method, r.URL.Path)
		fmt.Fprintf(w, "hello world")
	})

	http.HandleFunc("/youtube", func(w http.ResponseWriter, r *http.Request) {
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
			http.Error(w, "Missing or invalid 'id' field", http.StatusBadRequest)
			return
		}

		// Handle cookies if provided
		if cookies, ok := jsonData["cookies"].(string); ok && cookies != "" {
			if err := writeCookiesFile(cookies); err != nil {
				fmt.Printf("Warning: failed to write cookies: %v\n", err)
			}
		}

		added := downloadQueue.AddToQueue(id)
		if !added {
			fmt.Fprintf(w, "Video %s is already in download queue", id)
			return
		}

		queueLength, isRunning := downloadQueue.GetQueueStatus()
		fmt.Fprintf(w, "Video %s added to download queue. Queue length: %d, Processing: %t", id, queueLength, isRunning)
	})
}