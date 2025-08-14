package util

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

// VideoMetadata represents metadata for a video/audio track
type VideoMetadata struct {
	ID          string   `json:"id"`
	Title       *string  `json:"title"`
	Author      *string  `json:"author"`
	BPM         *float64 `json:"bpm"`
	Key         *string  `json:"key"`
	VocalBPM    *float64 `json:"vocal_bpm"`
	VocalKey    *string  `json:"vocal_key"`
	InstrumentalBPM *float64 `json:"instrumental_bpm"`
	InstrumentalKey *string  `json:"instrumental_key"`
	Duration    *float64 `json:"duration"`
	TrackType   *string  `json:"track_type"`
	LastModified time.Time
	VocalDone   bool
}

// Database wraps the SQL database with higher-level methods
type Database struct {
	db *sql.DB
}

// InitDatabase creates and initializes the database
func InitDatabase() (*Database, error) {
	dbPath := "./data/starchive.db"
	
	// Ensure data directory exists
	if err := os.MkdirAll("./data", 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}
	
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}
	
	// Create table if it doesn't exist (using original schema for compatibility)
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
		db.Close()
		return nil, fmt.Errorf("failed to create table: %v", err)
	}
	
	return &Database{db: db}, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// GetCachedMetadata retrieves cached metadata for a video ID
func (d *Database) GetCachedMetadata(id string) (*VideoMetadata, bool) {
	var metadata VideoMetadata
	var lastModified int64
	
	// Use the original database schema to maintain compatibility
	query := `SELECT id, title, last_modified, vocal_done, bpm, key
	          FROM video_metadata WHERE id = ?`
	
	row := d.db.QueryRow(query, id)
	
	err := row.Scan(&metadata.ID, &metadata.Title, &lastModified, &metadata.VocalDone,
		&metadata.BPM, &metadata.Key)
	
	if err == sql.ErrNoRows {
		return tryLoadFromJSON(id)
	}
	if err != nil {
		fmt.Printf("Database error: %v\n", err)
		return tryLoadFromJSON(id)
	}
	
	metadata.LastModified = time.Unix(lastModified, 0)
	return &metadata, true
}

func tryLoadFromJSON(id string) (*VideoMetadata, bool) {
	jsonPath := filepath.Join("./data", id+".json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, false
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return nil, false
	}

	metadata := &VideoMetadata{ID: id}
	
	if title, ok := jsonData["title"].(string); ok {
		metadata.Title = &title
	}
	if author, ok := jsonData["uploader"].(string); ok {
		metadata.Author = &author
	}
	if duration, ok := jsonData["duration"].(float64); ok {
		metadata.Duration = &duration
	}

	return metadata, true
}

// SaveMetadata saves metadata to the database
func (d *Database) SaveMetadata(metadata *VideoMetadata) error {
	query := `INSERT OR REPLACE INTO video_metadata 
		(id, title, last_modified, vocal_done, bpm, key) 
		VALUES (?, ?, ?, ?, ?, ?)`
	
	title := ""
	if metadata.Title != nil {
		title = *metadata.Title
	}
	
	_, err := d.db.Exec(query, metadata.ID, title, metadata.LastModified.Unix(),
		metadata.VocalDone, metadata.BPM, metadata.Key)
	
	return err
}

// GetAllMetadata returns all metadata entries
func (d *Database) GetAllMetadata() ([]VideoMetadata, error) {
	query := `SELECT id, title, last_modified, vocal_done, bpm, key
	          FROM video_metadata ORDER BY last_modified DESC`
	
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var results []VideoMetadata
	for rows.Next() {
		var metadata VideoMetadata
		var lastModified int64
		
		err := rows.Scan(&metadata.ID, &metadata.Title, &lastModified, 
			&metadata.VocalDone, &metadata.BPM, &metadata.Key)
		if err != nil {
			continue
		}
		
		metadata.LastModified = time.Unix(lastModified, 0)
		results = append(results, metadata)
	}
	
	return results, nil
}

// UpdateBPMAndKey updates BPM and key information for a track
func (d *Database) UpdateBPMAndKey(id string, bpm *float64, key *string, trackType string) error {
	var query string
	var args []interface{}
	
	switch trackType {
	case "vocal":
		query = "UPDATE video_metadata SET vocal_bpm = ?, vocal_key = ? WHERE id = ?"
		args = []interface{}{bpm, key, id}
	case "instrumental":
		query = "UPDATE video_metadata SET instrumental_bpm = ?, instrumental_key = ? WHERE id = ?"
		args = []interface{}{bpm, key, id}
	default:
		query = "UPDATE video_metadata SET bpm = ?, key = ? WHERE id = ?"
		args = []interface{}{bpm, key, id}
	}
	
	_, err := d.db.Exec(query, args...)
	return err
}

// DeleteMetadata removes metadata for a given ID
func (d *Database) DeleteMetadata(id string) error {
	query := "DELETE FROM video_metadata WHERE id = ?"
	_, err := d.db.Exec(query, id)
	return err
}

// FindMetadataByPattern finds metadata entries matching a pattern
func (d *Database) FindMetadataByPattern(pattern string) ([]VideoMetadata, error) {
	pattern = strings.ToLower(pattern)
	query := `SELECT id, title, last_modified, vocal_done, bpm, key
	          FROM video_metadata 
	          WHERE LOWER(id) LIKE ? OR LOWER(title) LIKE ?
	          ORDER BY last_modified DESC`
	
	searchPattern := "%" + pattern + "%"
	rows, err := d.db.Query(query, searchPattern, searchPattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var results []VideoMetadata
	for rows.Next() {
		var metadata VideoMetadata
		var lastModified int64
		
		err := rows.Scan(&metadata.ID, &metadata.Title, &lastModified,
			&metadata.VocalDone, &metadata.BPM, &metadata.Key)
		if err != nil {
			continue
		}
		
		metadata.LastModified = time.Unix(lastModified, 0)
		results = append(results, metadata)
	}
	
	return results, nil
}