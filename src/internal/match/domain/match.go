package domain

import (
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/identity/domain"
)

// I want to admit that go really looks so good and makes me feel like i was fighting something in rust and all those structs are insane
// thanks google

// MatchStatus represents the processing state of a match video.
type MatchStatus string

const (
	MatchStatusUploading  MatchStatus = "UPLOADING"
	MatchStatusProcessing MatchStatus = "PROCESSING"
	MatchStatusCompleted  MatchStatus = "COMPLETED"
	MatchStatusFailed     MatchStatus = "FAILED"
)

// Match is the aggregate root for the Analysis bounded context.
// TODO!: Remove gorm
type Match struct {
	ID             string        `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	User           domain.User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	UserID         string        `gorm:"index;not null"`
	ShirtNumber    uint          `gorm:"not null"`
	PositionPlayed string        `gorm:"type:varchar(20)"`
	VideoURL       string        `gorm:"type:varchar(100)"`
	Status         MatchStatus   `gorm:"type:varchar(20);index;default:UPLOADING"`
	AnalysisFlag   bool          `gorm:"default:false"`
	CreatedAt      time.Time     `gorm:"autoCreateTime"`
	UpdatedAt      time.Time     `gorm:"autoUpdateTime"`
	Metadata       MatchMetadata `gorm:"type:jsonb;serializer:json"`
}

// MatchMetadata contains flexible match information.
type MatchMetadata struct {
	MatchDate time.Time `json:"match_date"`
	Duration  int       `json:"duration_seconds"`
	Score     string    `json:"score,omitempty"`
	Location  string    `json:"location,omitempty"`
}

// NewMatch creates a new match aggregate in UPLOADING state.
func NewMatch(id, userID string, shirtNumber uint, positionPlayed string, metadata MatchMetadata) (*Match, error) {
	match := &Match{
		ID:             id,
		UserID:         userID,
		ShirtNumber:    shirtNumber,
		PositionPlayed: positionPlayed,
		Status:         MatchStatusUploading,
		UpdatedAt:      time.Now(),
		Metadata:       metadata,
	}

	if err := match.Validate(); err != nil {
		return nil, err
	}

	return match, nil
}

// Validate checks domain invariants for the Match aggregate.
func (m *Match) Validate() error {
	if m.ID == "" {
		return ErrInvalidMatchID
	}
	if m.UserID == "" {
		return ErrMissingUserID
	}
	if m.ShirtNumber < 1 || m.ShirtNumber > 99 {
		return ErrInvalidShirtNumber
	}

	switch m.Status {
	case MatchStatusUploading, MatchStatusProcessing, MatchStatusCompleted, MatchStatusFailed:
	default:
		return ErrInvalidMatchStatus
	}

	return nil
}

// MarkUploadComplete transitions from UPLOADING to PROCESSING.
func (m *Match) UpdateMatchStatusToProcessing(videoURL string) error {
	if m.Status != MatchStatusUploading {
		return ErrInvalidStatusTransition
	}
	m.VideoURL = videoURL
	m.Status = MatchStatusProcessing
	m.UpdatedAt = time.Now()

	return nil
}

// MarkFailed transitions to FAILED state.
func (m *Match) MarkFailed() error {
	if m.Status != MatchStatusProcessing {
		return ErrInvalidStatusTransition
	}

	m.Status = MatchStatusFailed
	return nil
}

// CanViewResults returns true if the match has completed analysis.
// TODO!: Call analysis result table here instead of match
func (ar *AnalysisResult) CanViewResults(matchID string) bool {
	//TODO!: Fetch match from database using matchID to check its status
	return m.Status == MatchStatusCompleted && m.AnalysisResult != nil
}

// GetTopInsight returns the most significant event insight for quick display.
// Returns empty string if no events or no completed analysis.
//TODO!: GetTopInsight should probably have matchID as parameter and take *AnalysisResult instead of *Match
/*func (m *Match) GetTopInsight() string {
	if !m.CanViewResults() || len(m.AnalysisResult.Events) == 0 {
		return ""
	}

	for _, event := range m.AnalysisResult.Events {
		if event.Result == EventResultSuccess && event.Insight != "" {
			return event.Insight
		}
	}

	return ""
}*/

var (
	ErrInvalidMatchID          = &MatchError{"match ID is required"}
	ErrMissingUserID           = &MatchError{"user ID is required"}
	ErrInvalidShirtNumber      = &MatchError{"shirt number must be between 1 and 99"}
	ErrInvalidMatchStatus      = &MatchError{"invalid match status"}
	ErrInvalidStatusTransition = &MatchError{"invalid status transition"}
	ErrNilAnalysisResult       = &MatchError{"analysis result cannot be nil"}
)

// MatchError represents domain-specific errors for the Match aggregate.
type MatchError struct {
	Message string
}

func (e *MatchError) Error() string {
	return e.Message
}
