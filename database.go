package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type VideoMetadata struct {
	ID           string
	Title        string
	LastModified time.Time
	VocalDone    bool
}

func initDatabase() (*sql.DB, error) {
	dbPath := "./data/starchive.db"
	
	// Ensure data directory exists
	if err := os.MkdirAll("./data", 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}
	
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}
	
	// Create table if it doesn't exist
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS video_metadata (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		last_modified INTEGER NOT NULL,
		vocal_done BOOLEAN DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_last_modified ON video_metadata(last_modified);
	`
	
	if _, err := db.Exec(createTableSQL); err != nil {
		return nil, fmt.Errorf("failed to create table: %v", err)
	}
	
	return db, nil
}

func getCachedMetadata(db *sql.DB, id string) (*VideoMetadata, bool) {
	var metadata VideoMetadata
	var lastModified int64
	var vocalDoneInt int
	
	err := db.QueryRow("SELECT id, title, last_modified, vocal_done FROM video_metadata WHERE id = ?", id).
		Scan(&metadata.ID, &metadata.Title, &lastModified, &vocalDoneInt)
	
	if err != nil {
		return nil, false
	}
	
	metadata.LastModified = time.Unix(lastModified, 0)
	metadata.VocalDone = vocalDoneInt == 1
	
	return &metadata, true
}

func cacheMetadata(db *sql.DB, metadata VideoMetadata) error {
	// Check if record exists and preserve vocal_done value
	var existingVocalDoneInt int
	err := db.QueryRow("SELECT vocal_done FROM video_metadata WHERE id = ?", metadata.ID).Scan(&existingVocalDoneInt)
	
	// If record exists, preserve the existing vocal_done value
	vocalDone := metadata.VocalDone
	if err == nil {
		vocalDone = existingVocalDoneInt == 1
	}
	
	vocalDoneInt := 0
	if vocalDone {
		vocalDoneInt = 1
	}
	
	_, err = db.Exec(`
		INSERT OR REPLACE INTO video_metadata (id, title, last_modified, vocal_done) 
		VALUES (?, ?, ?, ?)`,
		metadata.ID, metadata.Title, metadata.LastModified.Unix(), vocalDoneInt)
	return err
}

func parseJSONMetadata(filePath string) (*VideoMetadata, error) {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	
	var jsonData map[string]interface{}
	if err := json.Unmarshal(fileContent, &jsonData); err != nil {
		return nil, err
	}
	
	id := strings.TrimSuffix(filepath.Base(filePath), ".json")
	title, ok := jsonData["title"].(string)
	if !ok {
		title = "<no title>"
	}
	
	// Get file modification time
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	
	return &VideoMetadata{
		ID:           id,
		Title:        title,
		LastModified: fileInfo.ModTime(),
		VocalDone:    false,
	}, nil
}

func markVocalDone(db *sql.DB, id string) error {
	_, err := db.Exec("UPDATE video_metadata SET vocal_done = 1 WHERE id = ?", id)
	return err
}