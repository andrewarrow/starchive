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
		
		var jsonData interface{}
		if err := json.Unmarshal(body, &jsonData); err != nil {
			fmt.Printf("Invalid JSON received: %s\n", string(body))
		} else {
			fmt.Printf("JSON received: %+v\n", jsonData)
		}
		
		fmt.Fprintf(w, "Received")
	})
	
	fmt.Println("Server starting on port 3009...")
	http.ListenAndServe(":3009", nil)
}