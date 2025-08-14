package blend

import (
	"starchive/util"
)

// VideoMetadata is an alias to the util package type
type VideoMetadata = util.VideoMetadata

// VocalSegment represents a segment of vocal audio
type VocalSegment struct {
	Index     int     `json:"index"`
	StartTime float64 `json:"start_time"`
	Duration  float64 `json:"duration"`
	Placement float64 `json:"placement"` // Where to place in target track
	Active    bool    `json:"active"`    // Whether this segment is enabled
	RMSEnergy float64 `json:"rms_energy"` // Root Mean Square energy level (0.0-1.0)
	PeakLevel float64 `json:"peak_level"` // Peak amplitude level (0.0-1.0)
	EnergyCategory string `json:"energy_category"` // "low", "medium", "high"
}

// Shell represents the blend shell for mixing two tracks
type Shell struct {
	ID1, ID2           string
	Metadata1, Metadata2 *VideoMetadata
	Type1, Type2       string
	Pitch1, Pitch2     int
	Tempo1, Tempo2     float64
	Volume1, Volume2   float64
	Duration1, Duration2 float64
	Window1, Window2   float64
	InputPath1, InputPath2 string
	DB                 *util.Database
	PreviousBPMMatch, PreviousKeyMatch string
	Segments1, Segments2 []VocalSegment // Vocal segments for each track
	SegmentsDir1, SegmentsDir2 string   // Directories containing split files
}

// InvertState stores the state for intelligent track matching
type InvertState struct {
	BPMMatch string
	KeyMatch string
}