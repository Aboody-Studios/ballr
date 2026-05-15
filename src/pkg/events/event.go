package events

import "time"

type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	UserID    string                 `json:"user_id"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

const (
	EventMatchUploaded     = "MATCH_UPLOADED"
	EventAnalysisCompleted = "ANALYSIS_COMPLETED"
	EventCoachInteraction  = "COACH_INTERACTION"
	EventDrillCompleted    = "DRILL_COMPLETED"
	EventStreakMaintained  = "STREAK_MAINTAINED"
)

const (
	DefaultStream    = "ballr:events"
	DefaultGroup     = "ballr:event-handlers"
	DeadLetterStream = "ballr:events:dead"
)
