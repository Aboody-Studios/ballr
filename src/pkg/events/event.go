package events

import "time"

type Event struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	UserID    string         `json:"user_id"`
	Timestamp time.Time      `json:"timestamp"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// TODO!: Remove from here or from progress.go
const (
	EventMatchUploaded     = "MATCH_UPLOADED"
	EventAnalysisCompleted = "ANALYSIS_COMPLETED"
	EventAnalysisStart     = "ANALYSIS_START"
	EventCoachInteraction  = "COACH_INTERACTION"
	EventDrillCompleted    = "DRILL_COMPLETED"
	EventStreakMaintained  = "STREAK_MAINTAINED"
)

const (
	DefaultStream    = "ballr:events"
	DefaultGroup     = "ballr:event-handlers"
	DeadLetterStream = "ballr:events:dead"
)
