package main

import (
	"fmt"
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