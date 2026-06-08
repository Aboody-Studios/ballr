package application

import (
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/identity/domain"
)

type S3Object struct {
	Key string `json:"key"`
}

type S3Entity struct {
	Object S3Object `json:"object"`
}

type S3Record struct {
	S3 S3Entity `json:"s3"`
}

type S3Success struct {
	Records []S3Record `json:"Records"`
}

type MatchStatus string

const (
	MatchStatusUploading  MatchStatus = "UPLOADING"
	MatchStatusProcessing MatchStatus = "PROCESSING"
	MatchStatusCompleted  MatchStatus = "COMPLETED"
	MatchStatusFailed     MatchStatus = "FAILED"
)

type MatchRequest struct {
	ShirtNumber uint            `json:"shirt_number"`
	Position    domain.Position `json:"position"`
	Size        uint64          `json:"size"`
	Metadata    MatchMetadata   `json:"metadata"`
}

type MatchMetadata struct {
	MatchDate time.Time `json:"match_date"`
	Duration  int       `json:"duration_seconds"`
	Score     string    `json:"score,omitempty"`
	Location  string    `json:"location,omitempty"`
}
