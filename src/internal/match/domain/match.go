package domain

import (
	"time"
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
	ID             string          `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID         string          `gorm:"index;not null"`
	ShirtNumber    int             `gorm:"not null"`
	PositionPlayed string          `gorm:"type:varchar(20)"`
	VideoURL       string          `gorm:"type:varchar(100)"`
	Status         MatchStatus     `gorm:"type:varchar(20);index;default:UPLOADING"`
	UploadedAt     time.Time       `gorm:"autoCreateTime"`
	Metadata       MatchMetadata   `gorm:"type:jsonb;serializer:json"`
	AnalysisResult *AnalysisResult `gorm:"type:jsonb;serializer:json"`
}

// MatchMetadata contains flexible match information.
type MatchMetadata struct {
	MatchDate time.Time `json:"match_date"`
	Duration  int       `json:"duration_seconds"`
	Score     string    `json:"score,omitempty"`
	Location  string    `json:"location,omitempty"`
}

// AnalysisResult contains the computer vision analysis output.
type AnalysisResult struct {
	MatchID         string          `json:"match_id" gorm:"index"`
	GeneratedAt     time.Time       `json:"generated_at"`
	Summary         AnalysisSummary `json:"summary" gorm:"type:jsonb;serializer:json"`
	Heatmaps        Heatmaps        `json:"heatmaps" gorm:"type:jsonb;serializer:json"`
	Events          []MatchEvent    `json:"events" gorm:"type:jsonb;serializer:json"`
	TrackingDataURL string          `json:"tracking_data_url,omitempty"`
}

// AnalysisSummary contains key metrics from the match analysis.
type AnalysisSummary struct {
	TotalDistanceKM float64 `json:"total_distance"`
	TopSpeedKMH     float64 `json:"top_speed"`
	PassAccuracy    float64 `json:"pass_accuracy"`
	Touches         int     `json:"touches"`
	Sprints         int     `json:"sprints"`
}

// Heatmaps contains visualization data for player positioning.
type Heatmaps struct {
	OverallURL   string `json:"overall_url"`
	DefensiveURL string `json:"defensive_url"`
	AttackingURL string `json:"attacking_url"`
}

// MatchEvent represents a single event detected during match analysis.
type MatchEvent struct {
	Timestamp   string      `json:"timestamp"`
	Type        EventType   `json:"type"`
	Result      EventResult `json:"result"`
	Coordinates Position2D  `json:"coordinates"`
	Insight     string      `json:"insight"`
}

// EventType categorizes match events detected by computer vision.
type EventType string

const (
	EventTypePass     EventType = "PASS"
	EventTypeShot     EventType = "SHOT"
	EventTypeDribble  EventType = "DRIBBLE"
	EventTypeTackle   EventType = "TACKLE"
	EventTypeSave     EventType = "SAVE"
	EventTypeSprint   EventType = "SPRINT"
	EventTypeRecovery EventType = "RECOVERY"
)

// EventResult indicates the outcome of an event.
type EventResult string

const (
	EventResultSuccess EventResult = "SUCCESS"
	EventResultFailure EventResult = "FAILURE"
	EventResultNeutral EventResult = "NEUTRAL"
)

// Position2D represents coordinates on the pitch.
// X: 0 = left touchline, 100 = right touchline
// Y: 0 = own goal line, 100 = opponent goal line
type Position2D struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// NewMatch creates a new match aggregate in UPLOADING state.
func NewMatch(id, userID string, shirtNumber int, positionPlayed string, metadata MatchMetadata) (*Match, error) {
	match := &Match{
		ID:             id,
		UserID:         userID,
		ShirtNumber:    shirtNumber,
		PositionPlayed: positionPlayed,
		Status:         MatchStatusUploading,
		UploadedAt:     time.Now(),
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
func (m *Match) MarkUploadComplete(videoURL string) error {
	if m.Status != MatchStatusUploading {
		return ErrInvalidStatusTransition
	}

	m.Status = MatchStatusProcessing
	m.UploadedAt = time.Now()

	return nil
}

// SetAnalysisResult transitions to COMPLETED and stores the results.
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
func (m *Match) CanViewResults() bool {
	return m.Status == MatchStatusCompleted && m.AnalysisResult != nil
}

// GetTopInsight returns the most significant event insight for quick display.
// Returns empty string if no events or no completed analysis.
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
