package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func main() {
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

		_, err = DownloadVideo(id)
		if err != nil {
			fmt.Printf("Error downloading video: %v\n", err)
			http.Error(w, "Error downloading video", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Video download started for ID: %s", id)
	})

	fmt.Println("Server starting on port 3009...")
	http.ListenAndServe(":3009", nil)
}
