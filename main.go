package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Request received:", r.Method, r.URL.Path)
		fmt.Fprintf(w, "hello world")
	})
	
	fmt.Println("Server starting on port 3009...")
	http.ListenAndServe(":3009", nil)
}