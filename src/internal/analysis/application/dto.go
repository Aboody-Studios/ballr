package application

import "time"

// S3 success request struct
type Key struct {
	Key string `json:"key"`
}

type ObjectObject struct {
	Object Key `json:"object"`
}

type S3Field struct {
	S3Field ObjectObject `json:"s3"`
}

type S3Success struct {
	Records S3Field `json:"Records"`
}

type MatchStatus string

const (
	MatchStatusUploading  MatchStatus = "UPLOADING"
	MatchStatusProcessing MatchStatus = "PROCESSING"
	MatchStatusCompleted  MatchStatus = "COMPLETED"
	MatchStatusFailed     MatchStatus = "FAILED"
)

type MatchRequest struct {
	ShirtNumber int           `json:"shirt_number"`
	Position    string        `json:"position"`
	Size        uint64        `json:"size"`
	Metadata    MatchMetadata `json:"metadata"`
}

type MatchMetadata struct {
	MatchDate time.Time `json:"match_date"`
	Duration  int       `json:"duration_seconds"`
	Score     string    `json:"score,omitempty"`
	Location  string    `json:"location,omitempty"`
}
