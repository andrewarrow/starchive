package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
)

type DownloadQueue struct {
	queue chan string
	log   *slog.Logger
}

func NewDownloadQueue(workers, depth int, logger *slog.Logger) *DownloadQueue {
	ret := DownloadQueue{
		queue: make(chan string, depth),
		log:   logger,
	}

	for idx := range workers {
		go ret.processQueue(fmt.Sprintf("worker-%d", idx))
	}

	return &ret
}

func (dq *DownloadQueue) AddToQueue(videoId string) bool {
	dq.log.With("operation", "AddToQueue", "videoId", videoId).Info("Added to queue")
	dq.queue <- videoId
	return true
}

func (dq *DownloadQueue) processQueue(workerId string) {
	log := dq.log.With("worker_id", workerId, "operation", "processQueue")
	for {
		select {
		case videoId := <-dq.queue:
			if _, err := DownloadVideoWithFormat(dq.log, videoId, videoFormat); err != nil {
				log.Error("Failed to download video", "videoId", videoId)
			} else {
				log.Info("Successfully downloaded video", "videoId", videoId)
			}
		}
	}
}

var downloadQueue *DownloadQueue
var videoFormat string

func main() {
	// Parse CLI flags
	flag.StringVar(&videoFormat, "format", "mov", "Video format (mov or mkv)")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	downloadQueue = NewDownloadQueue(1, 10, log.With("role", "downloader"))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.
			With("operation", "http",
				"method", r.Method,
				"url", r.URL.Path,
			).Info("Request received")
		if _, err := fmt.Fprintf(w, "hello world"); err != nil {
			log.
				With("operation", "http",
					"method", r.Method,
					"url", r.URL.Path,
				).
				Error("Failed to write response", "err", err)
		}
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

		var jsonData Api
		if err := json.Unmarshal(body, &jsonData); err != nil {
			log.
				With("operation", "http",
					"method", r.Method,
					"url", r.URL.Path,
				).
				Error("Failed to unmarshal json", "err", err)
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		log.With("operation", "http",
			"method", r.Method,
			"url", r.URL.Path,
		).Info("Request received")

		if jsonData.VideoId == "" {
			http.Error(w, "Missing or invalid 'id' field", http.StatusBadRequest)
			return
		}

		added := downloadQueue.AddToQueue(jsonData.VideoId)
		if !added {
			log.With("operation", "http",
				"method", r.Method,
				"url", r.URL.Path,
			).Error("Failed to add video", "videoId", jsonData.VideoId)
			return
		}

		log.With("operation", "http",
			"method", r.Method,
			"url", r.URL.Path,
		).Info("Request processed", "videoId", jsonData.VideoId, "queueDepth", len(downloadQueue.queue))
	})

	log.Info("Starting server on port 3009...")
	if err := http.ListenAndServe(":3009", nil); err != nil {
		log.With("operation", "http").Error("Failed to start server", "err", err)
	}
}
