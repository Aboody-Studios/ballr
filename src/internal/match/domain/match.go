package domain

import (
	"time"
)

// I want to admit that go really looks so good and makes me feel like i was fighting something in rust and all those structs are insane
// thanks google

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

// Match is the aggregate root for the Analysis bounded context.
type Match struct {
	ID             string
	UserID         string
	ShirtNumber    uint
	PositionPlayed Position
	VideoURL       string
	Status         MatchStatus
	AnalysisFlag   bool
	AnalysisResult *AnalysisResult
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Metadata       MatchMetadata
}

type MatchMetadata struct {
	MatchDate time.Time `json:"match_date"`
	Duration  int       `json:"duration_seconds"`
	Score     string    `json:"score,omitempty"`
	Location  string    `json:"location,omitempty"`
}

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

func (m *Match) UpdateMatchStatusToProcessing(videoURL string) error {
	if m.Status != MatchStatusUploading {
		return ErrInvalidStatusTransition
	}
	m.VideoURL = videoURL
	m.Status = MatchStatusProcessing
	m.UpdatedAt = time.Now()

	return nil
}

// MarkUploadComplete transitions the match from UPLOADING -> PROCESSING
func (m *Match) MarkUploadComplete(videoURL string) error {
	return m.UpdateMatchStatusToProcessing(videoURL)
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
// CanViewResults returns true if the match has completed analysis and a result exists.
func (m *Match) CanViewResults() bool {
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

// SetAnalysisResult transitions to COMPLETED and stores the analysis results.
func (m *Match) SetAnalysisResult(result *AnalysisResult) error {
	if m.Status != MatchStatusProcessing {
		return ErrInvalidStatusTransition
	}
	if result == nil {
		return ErrNilAnalysisResult
	}

	result.GeneratedAt = time.Now()
	m.AnalysisResult = result
	m.Status = MatchStatusCompleted
	m.UpdatedAt = time.Now()

	return nil
}

// GetTopInsight returns the first successful insight from analysis events.
func (m *Match) GetTopInsight() string {
	if !m.CanViewResults() || len(m.AnalysisResult.Events) == 0 {
		return ""
	}

	for _, event := range m.AnalysisResult.Events {
		if event.Result == EventResultSuccess && event.Insight != "" {
			return event.Insight
		}
	}

	return ""
}

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

// --- tests ---

// Test only function
func NewMatch(id, userID string, shirtNumber uint, positionPlayed Position, metadata MatchMetadata) (*Match, error) {
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
