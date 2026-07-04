package infrastructure

import (
	"time"

	userdomain "github.com/Aboody-Studios/ballr/src/internal/identity/domain"
	matchdomain "github.com/Aboody-Studios/ballr/src/internal/match/domain"
)

// TODO!: See if connecting Match with User requires passing the whole User struct or just the UserID as a foreign key ?
type Match struct {
	ID             string                      `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	User           userdomain.User             `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	UserID         string                      `gorm:"index;not null"`
	ShirtNumber    uint                        `gorm:"not null"`
	PositionPlayed matchdomain.Position        `gorm:"type:varchar(20)"`
	VideoURL       string                      `gorm:"type:varchar(100)"`
	Status         matchdomain.MatchStatus     `gorm:"type:varchar(20);index;default:UPLOADING"`
	AnalysisFlag   bool                        `gorm:"default:false"`
	AnalysisResult *matchdomain.AnalysisResult `gorm:"type:jsonb;serializer:json"`
	CreatedAt      time.Time                   `gorm:"autoCreateTime"`
	UpdatedAt      time.Time                   `gorm:"autoUpdateTime"`
	Metadata       matchdomain.MatchMetadata   `gorm:"type:jsonb;serializer:json"`
}

type MatchMetadata struct {
	MatchDate time.Time
	Duration  int
	Score     string
	Location  string
}
