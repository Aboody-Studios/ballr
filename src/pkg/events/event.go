package events

import "time"

type Event struct {
	ID        string    `json:"id"`
	Type      EventType `json:"type"`
	UserID    string    `json:"user_id"`
	Metadata  map[string]any
	Timestamp time.Time `json:"timestamp"`
}

type EventType string

// TODO!: Add EventStreakMultipleOf7
const (
	EventMatchUploaded             EventType = "MATCH_UPLOADED"
	EventAnalysisCompleted         EventType = "ANALYSIS_COMPLETED"
	EventDrillCompleted            EventType = "DRILL_COMPLETED"
	EventCoachInteraction          EventType = "COACH_INTERACTION"
	EventAnalysisStart             EventType = "ANALYSIS_START"
	EventAchievementCheckRequested EventType = "CHECK_ACHIEVEMENT"
)

// Either use PointValue or create a new array array for events with points to check whether event is considered an activity or not
var PointValue = map[EventType]int{
	EventMatchUploaded:     50,
	EventAnalysisCompleted: 100,
	EventDrillCompleted:    25,
	EventCoachInteraction:  10,
}

const (
	DefaultStream    = "ballr:events"
	DefaultGroup     = "ballr:event-handlers"
	DeadLetterStream = "ballr:events:dead"
)
