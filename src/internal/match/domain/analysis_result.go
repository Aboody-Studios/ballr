package domain

import "time"

// AnalysisResult contains the computer vision analysis output.
type AnalysisResult struct {
	ID              string          `json:"analysis_id" gorm:"index"`
	MatchID         string          `json:"match_id"`
	Match           Match           `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
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

func (m *Match) MarkAnalysisInit() {
	m.AnalysisFlag = true
}

// CanViewResults returns true if the analysis's associated match is completed.
func (ar *AnalysisResult) CanViewResults(matchID string) bool {
	return ar.Match.Status == MatchStatusCompleted && ar.Match.Status != ""
}

// SetAnalysisResult transitions to COMPLETED and stores the results.
// TODO!: Set analysis result to analysis table and not match table
// This function seems unnecessary
/*func (m *Match) SetAnalysisID(result *AnalysisResult) error {
	if m.Status != MatchStatusProcessing {
		return ErrInvalidStatusTransition
	}

	if result == nil {
		return ErrNilAnalysisResult
	}

	result.GeneratedAt = time.Now()
	m.AnalysisResult = *result
	m.Status = MatchStatusCompleted

	return nil
}*/
