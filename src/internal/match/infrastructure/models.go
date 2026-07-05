package infrastructure

import (
	"time"

	matchdomain "github.com/Aboody-Studios/ballr/src/internal/match/domain"
)

type Match struct {
	ID             string                  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID         string                  `gorm:"index;not null"`
	ShirtNumber    uint                    `gorm:"not null"`
	PositionPlayed matchdomain.Position    `gorm:"type:varchar(20)"`
	VideoURL       string                  `gorm:"type:varchar(100)"`
	Status         matchdomain.MatchStatus `gorm:"type:varchar(20);index;default:UPLOADING"`
	AnalysisFlag   bool                    `gorm:"default:false"`
	AnalysisResult *matchdomain.AnalysisResult
	CreatedAt      time.Time                 `gorm:"autoCreateTime"`
	UpdatedAt      time.Time                 `gorm:"autoUpdateTime"`
	Metadata       matchdomain.MatchMetadata `gorm:"type:jsonb;serializer:json"`
}

type MatchMetadata struct {
	MatchDate time.Time
	Duration  int
	Score     string
	Location  string
}
