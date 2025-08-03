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
	ret.log.Info("starting video downloader", "queueDepth", depth, "workers", workers)

	for idx := range workers {
		go ret.processQueue(fmt.Sprintf("worker-%d", idx))
	}

	return &ret
}

func (dq *DownloadQueue) AddToQueue(videoID string) (bool, int) {
	dq.log.With("operation", "AddToQueue", "videoId", videoID).Info("Added to queue")
	dq.queue <- videoID
	return true, len(dq.queue)
}

func (dq *DownloadQueue) processQueue(workerID string) {
	log := dq.log.With("worker_id", workerID, "operation", "processQueue")
	log.Info("queue monitor started")
	for videoID := range dq.queue {
		if _, err := DownloadVideoWithFormat(dq.log, videoID, videoFormat); err != nil {
			log.Error("Failed to download video", "videoId", videoID)
		} else {
			log.Info("Successfully downloaded video", "videoId", videoID)
		}
	}
}

var (
	downloadQueue       *DownloadQueue
	videoFormat         string
	downloadConcurrency int
	queueDepth          int
)

func main() {
	// Parse CLI flags
	flag.StringVar(&videoFormat, "format", "mov", "Video format (mov or mkv)")
	flag.IntVar(&downloadConcurrency, "downloadConcurrency", 1, "how many concurrent downloads to perform")
	flag.IntVar(&queueDepth, "queueDepth", 10, "number of videos to queue before blocking the process")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	downloadQueue = NewDownloadQueue(downloadConcurrency, queueDepth, log.With("role", "downloader"))
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

		var jsonData API
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

		if jsonData.VideoID == "" {
			http.Error(w, "Missing or invalid 'id' field", http.StatusBadRequest)
			return
		}

		added, currLen := downloadQueue.AddToQueue(jsonData.VideoID)
		if !added {
			log.With("operation", "http",
				"method", r.Method,
				"url", r.URL.Path,
			).Error("Failed to add video", "videoId", jsonData.VideoID)
			return
		}

		log.With("operation", "http",
			"method", r.Method,
			"url", r.URL.Path,
		).Info("Request processed", "videoId", jsonData.VideoID, "queueDepth", len(downloadQueue.queue))
		fmt.Fprintf(w, "Video %s added to download queue. Queue length: %d", jsonData.VideoID, currLen)
	})

	log.Info("Starting server on port 3009...")
	if err := http.ListenAndServe(":3009", nil); err != nil {
		log.With("operation", "http").Error("Failed to start server", "err", err)
	}
}
