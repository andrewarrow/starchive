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
	ID              string
	Title           string
	LastModified    time.Time
	VocalDone       bool
	BPM             *float64
	Key             *string
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
		vocal_done BOOLEAN DEFAULT 0,
		bpm REAL,
		key TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_last_modified ON video_metadata(last_modified);
	`
	
	if _, err := db.Exec(createTableSQL); err != nil {
		return nil, fmt.Errorf("failed to create table: %v", err)
	}
	
	// Execute each ALTER statement separately, ignoring errors for columns that already exist
	alterStatements := []string{
		"ALTER TABLE video_metadata ADD COLUMN bpm REAL",
		"ALTER TABLE video_metadata ADD COLUMN key TEXT",
	}
	
	for _, stmt := range alterStatements {
		db.Exec(stmt) // Ignore errors - column may already exist
	}
	
	return db, nil
}

func getCachedMetadata(db *sql.DB, id string) (*VideoMetadata, bool) {
	var metadata VideoMetadata
	var lastModified int64
	var vocalDoneInt int
	var bpm sql.NullFloat64
	var key sql.NullString
	
	err := db.QueryRow("SELECT id, title, last_modified, vocal_done, bpm, key FROM video_metadata WHERE id = ?", id).
		Scan(&metadata.ID, &metadata.Title, &lastModified, &vocalDoneInt, &bpm, &key)
	
	if err != nil {
		return nil, false
	}
	
	metadata.LastModified = time.Unix(lastModified, 0)
	metadata.VocalDone = vocalDoneInt == 1
	
	if bpm.Valid {
		metadata.BPM = &bpm.Float64
	}
	if key.Valid {
		metadata.Key = &key.String
	}
	
	return &metadata, true
}

func cacheMetadata(db *sql.DB, metadata VideoMetadata) error {
	// Check if record exists and preserve existing values
	var existingVocalDoneInt int
	var existingBPM sql.NullFloat64
	var existingKey sql.NullString
	
	err := db.QueryRow("SELECT vocal_done, bpm, key FROM video_metadata WHERE id = ?", metadata.ID).
		Scan(&existingVocalDoneInt, &existingBPM, &existingKey)
	
	// If record exists, preserve existing values unless metadata has new values
	vocalDone := metadata.VocalDone
	bpm := metadata.BPM
	key := metadata.Key
	
	if err == nil {
		// Preserve existing values if new metadata doesn't have them
		vocalDone = existingVocalDoneInt == 1
		if bpm == nil && existingBPM.Valid {
			bpm = &existingBPM.Float64
		}
		if key == nil && existingKey.Valid {
			key = &existingKey.String
		}
	}
	
	vocalDoneInt := 0
	if vocalDone {
		vocalDoneInt = 1
	}
	
	_, err = db.Exec(`
		INSERT OR REPLACE INTO video_metadata (id, title, last_modified, vocal_done, bpm, key) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		metadata.ID, metadata.Title, metadata.LastModified.Unix(), vocalDoneInt, bpm, key)
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

func storeBPMData(db *sql.DB, id string, bpm float64, key string) error {
	// First ensure record exists
	_, err := db.Exec(`INSERT OR IGNORE INTO video_metadata (id, title, last_modified, vocal_done) VALUES (?, '', 0, 0)`, id)
	if err != nil {
		return err
	}
	
	// Then update with BPM data
	_, err = db.Exec(`UPDATE video_metadata SET bpm = ?, key = ? WHERE id = ?`,
		bpm, key, id)
	return err
}