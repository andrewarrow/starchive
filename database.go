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
		last_modified INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_last_modified ON video_metadata(last_modified);
	`
	
	if _, err := db.Exec(createTableSQL); err != nil {
		return nil, fmt.Errorf("failed to create table: %v", err)
	}
	
	return db, nil
}

func getCachedMetadata(db *sql.DB, id string, fileModTime time.Time) (*VideoMetadata, bool) {
	var metadata VideoMetadata
	var lastModified int64
	
	err := db.QueryRow("SELECT id, title, last_modified FROM video_metadata WHERE id = ?", id).
		Scan(&metadata.ID, &metadata.Title, &lastModified)
	
	if err != nil {
		return nil, false
	}
	
	metadata.LastModified = time.Unix(lastModified, 0)
	
	// Check if cached data is still valid (file hasn't been modified)
	if metadata.LastModified.Equal(fileModTime) || metadata.LastModified.After(fileModTime) {
		return &metadata, true
	}
	
	return nil, false
}

func cacheMetadata(db *sql.DB, metadata VideoMetadata) error {
	_, err := db.Exec(`
		INSERT OR REPLACE INTO video_metadata (id, title, last_modified) 
		VALUES (?, ?, ?)`,
		metadata.ID, metadata.Title, metadata.LastModified.Unix())
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
	}, nil
}