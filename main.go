package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
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

func main() {
	// Parse CLI flags
	flag.BoolVar(&downloadVideos, "download-videos", true, "Download full videos; if false, only subtitles and thumbnails")
	flag.Parse()

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

		added := downloadQueue.AddToQueue(id)
		if !added {
			fmt.Fprintf(w, "Video %s is already in download queue", id)
			return
		}

		queueLength, isRunning := downloadQueue.GetQueueStatus()
		fmt.Fprintf(w, "Video %s added to download queue. Queue length: %d, Processing: %t", id, queueLength, isRunning)
	})

	fmt.Println("Server starting on port 3009...")
	http.ListenAndServe(":3009", nil)
}
