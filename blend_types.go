package main

import (
	"database/sql"
)

type VocalSegment struct {
	Index     int     `json:"index"`
	StartTime float64 `json:"start_time"`
	Duration  float64 `json:"duration"`
	Placement float64 `json:"placement"` // Where to place in target track
	Active    bool    `json:"active"`    // Whether this segment is enabled
}

type BlendShell struct {
	id1, id2           string
	metadata1, metadata2 *VideoMetadata
	type1, type2       string
	pitch1, pitch2     int
	tempo1, tempo2     float64
	volume1, volume2   float64
	duration1, duration2 float64
	window1, window2   float64
	inputPath1, inputPath2 string
	db                 *sql.DB
	previousBPMMatch, previousKeyMatch string
	segments1, segments2 []VocalSegment // Vocal segments for each track
	segmentsDir1, segmentsDir2 string   // Directories containing split files
}

type InvertState struct {
	BPMMatch string
	KeyMatch string
}