package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type DownloadQueue struct {
	queue     []string
	isRunning bool
	mutex     sync.Mutex
}

func NewDownloadQueue() *DownloadQueue {
	return &DownloadQueue{
		queue:     make([]string, 0),
		isRunning: false,
	}
}

func (dq *DownloadQueue) AddToQueue(videoId string) bool {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()

	// Check if video is already in queue
	for _, id := range dq.queue {
		if id == videoId {
			fmt.Printf("Video %s is already in queue\n", videoId)
			return false
		}
	}

	dq.queue = append(dq.queue, videoId)
	fmt.Printf("Added video %s to queue. Queue length: %d\n", videoId, len(dq.queue))

	if !dq.isRunning {
		dq.isRunning = true
		go dq.processQueue()
	}

	return true
}

func (dq *DownloadQueue) processQueue() {
	for {
		dq.mutex.Lock()
		if len(dq.queue) == 0 {
			dq.isRunning = false
			dq.mutex.Unlock()
			fmt.Println("Queue is empty, stopping processor")
			return
		}

		videoId := dq.queue[0]
		dq.queue = dq.queue[1:]
		fmt.Printf("Processing video %s. Remaining in queue: %d\n", videoId, len(dq.queue))
		dq.mutex.Unlock()

		_, err := DownloadVideo(videoId)
		if err != nil {
			fmt.Printf("Error downloading video %s: %v\n", videoId, err)
		} else {
			fmt.Printf("Successfully downloaded video %s\n", videoId)
		}
	}
}

func (dq *DownloadQueue) GetQueueStatus() (int, bool) {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()
	return len(dq.queue), dq.isRunning
}

var downloadQueue *DownloadQueue
var downloadVideos bool

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

func main() {
	// Simple subcommand dispatch: first arg is the command
	if len(os.Args) < 2 {
		fmt.Println("Usage: starchive <command> [args]\n\nCommands:\n  run     Start the server (default features)\n  ls      List files in ./data\n  vocal   Extract vocals from audio file using audio-separator")
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "run":
		runCmd := flag.NewFlagSet("run", flag.ExitOnError)
		runCmd.BoolVar(&downloadVideos, "download-videos", true, "Download full videos; if false, only subtitles and thumbnails")
		// Parse flags after the subcommand
		if err := runCmd.Parse(os.Args[2:]); err != nil {
			fmt.Println("Error parsing flags:", err)
			os.Exit(2)
		}

		downloadQueue = NewDownloadQueue()
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

		fmt.Println("Server starting on port 3009...")
		if err := http.ListenAndServe(":3009", nil); err != nil {
			fmt.Printf("Server error: %v\n", err)
			os.Exit(1)
		}
	case "ls":
		entries, err := os.ReadDir("./data")
		if err != nil {
			fmt.Println("Error reading ./data:", err)
			os.Exit(1)
		}
		
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
				id := strings.TrimSuffix(e.Name(), ".json")
				
				filePath := "./data/" + e.Name()
				fileContent, err := os.ReadFile(filePath)
				if err != nil {
					fmt.Printf("%s\t<error reading file>\n", id)
					continue
				}
				
				var jsonData map[string]interface{}
				if err := json.Unmarshal(fileContent, &jsonData); err != nil {
					fmt.Printf("%s\t<error parsing JSON>\n", id)
					continue
				}
				
				title, ok := jsonData["title"].(string)
				if !ok {
					title = "<no title>"
				}
				
				fmt.Printf("%s\t%s\n", id, title)
			}
		}
	case "vocal", "vocals":
		if len(os.Args) < 3 {
			fmt.Println("Usage: starchive vocal <id>")
			fmt.Println("Example: starchive vocal abc123")
			os.Exit(1)
		}

		id := os.Args[2]
		inputPath := fmt.Sprintf("./data/%s.wav", id)

		// Check if input file exists
		if _, err := os.Stat(inputPath); os.IsNotExist(err) {
			fmt.Printf("Error: Input file %s does not exist\n", inputPath)
			os.Exit(1)
		}

		// Run audio-separator command
		cmd := exec.Command("audio-separator", inputPath,
			"--output_dir", "./data/",
			"--model_filename", "UVR_MDXNET_Main.onnx",
			"--output_format", "wav")

		fmt.Printf("Running: %s\n", cmd.String())
		
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Error running audio-separator: %v\n", err)
			fmt.Printf("Output: %s\n", string(output))
			os.Exit(1)
		}

		fmt.Printf("Successfully separated vocals for %s\n", id)
		fmt.Printf("Output: %s\n", string(output))
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		fmt.Println("Usage: starchive <command> [args]\n\nCommands:\n  run     Start the server (default features)\n  ls      List files in ./data\n  vocal   Extract vocals from audio file using audio-separator")
		os.Exit(1)
	}
}
