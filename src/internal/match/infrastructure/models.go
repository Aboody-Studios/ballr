package infrastructure

import (
	"time"

	matchdomain "github.com/Aboody-Studios/ballr/src/internal/match/domain"
)

type MatchStatus string

const (
	MatchStatusUploading  MatchStatus = "UPLOADING"
	MatchStatusProcessing MatchStatus = "PROCESSING"
	MatchStatusCompleted  MatchStatus = "COMPLETED"
	MatchStatusFailed     MatchStatus = "FAILED"
)

type Position string

const (
	PositionGK Position = "GK"
	PositionCB Position = "CB"
	PositionLB Position = "LB"
	PositionRB Position = "RB"
	PositionCM Position = "CM"
	PositionLW Position = "LW"
	PositionRW Position = "RW"
	PositionST Position = "ST"
)

type Match struct {
	ID             string      `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID         string      `gorm:"index;not null"`
	ShirtNumber    uint        `gorm:"not null" fake:"{number:1,99}"`
	PositionPlayed Position    `gorm:"type:varchar(20)" fake:"{randomstring:[GK,CB,LB,RB,CM,CAM,RW,LW,ST]}"`
	VideoURL       string      `gorm:"type:varchar(100)"`
	Status         MatchStatus `gorm:"type:varchar(20);index;default:UPLOADING"`
	AnalysisFlag   bool        `gorm:"default:false"`
	AnalysisResult *matchdomain.AnalysisResult
	CreatedAt      time.Time     `gorm:"autoCreateTime"`
	UpdatedAt      time.Time     `gorm:"autoUpdateTime"`
	Metadata       MatchMetadata `gorm:"type:jsonb;serializer:json"`
}

type MatchMetadata struct {
	MatchDate time.Time
	Duration  int
	Score     string
	Location  string
}
