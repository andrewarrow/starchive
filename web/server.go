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

// SetupRoutes configures HTTP routes for the web server
func SetupRoutes(downloadQueue interface{}) {
	http.HandleFunc("/api/download", func(w http.ResponseWriter, r *http.Request) {
		handleDownload(w, r, downloadQueue)
	})
	http.HandleFunc("/api/cookies", handleSetCookies)
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