package domain

import (
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/identity/domain"
	"github.com/Aboody-Studios/ballr/src/pkg/events"
	"github.com/google/uuid"
)

// Progress is the aggregate root for the Progress bounded context.
// It represents a user's gamification state including points, streaks, and activity tracking.
// TODO!: Remove gorm
type Progress struct {
	ID            string `gorm:"primaryKey"`
	UserID        string `gorm:"uniqueIndex;not null"`
	TotalPoints   int64  `gorm:"default:0"`
	CurrentStreak int    `gorm:"default:0"`
	LastActive    time.Time
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
}

// Achievement represents a badge or milestone unlocked by a user.
// Achievements are value objects that are part of the Progress aggregate.
// TODO!: Remove gorm
type Achievement struct {
	ID          string   `gorm:"primaryKey"`
	Progress    Progress `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	progressID  string
	User        domain.User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	UserID      string      `gorm:"uniqueIndex:idx_user_achievement;not null"`
	Type        string      `gorm:"uniqueIndex:idx_user_achievement;not null"`
	UnlockedAt  time.Time   `gorm:"autoCreateTime"`
	PointsValue int
}

// AchievementType defines the types of achievements available in the system.
// These are part of the ubiquitous language of the gamification domain.
type AchievementType string

const (
	AchievementTypeFirstUpload    AchievementType = "FIRST_UPLOAD"
	AchievementTypeFirstAnalysis  AchievementType = "FIRST_ANALYSIS"
	AchievementTypeStreakWeek     AchievementType = "STREAK_WEEK"
	AchievementTypeStreakMonth    AchievementType = "STREAK_MONTH"
	AchievementTypeStreak7        AchievementType = "STREAK_7"
	AchievementTypeStreak30       AchievementType = "STREAK_30"
	AchievementTypeAnalysisMaster AchievementType = "ANALYSIS_MASTER"
	AchievementTypeDrillCompleter AchievementType = "DRILL_COMPLETER"
	AchievementTypeTopPerformer   AchievementType = "TOP_PERFORMER"
	AchievementTypeCoachConsult   AchievementType = "COACH_CONSULT"
)

type EventLog struct {
	User           domain.User      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	UserID         string           `gorm:"index:idx_user_event;not null"`
	Type           events.EventType `gorm:"index:idx_user_event;not null"`
	PointsAwarded  int64
	IdempotencyKey string         //TODO!: Add unique constraint here
	Timestamp      time.Time      `gorm:"index"`
	Metadata       map[string]any `gorm:"type:jsonb;serializer:json"`
}

// EventSummary represents a single event for the activity feed.
type EventSummary struct {
	Type      string
	Points    int64
	Timestamp time.Time
}

// ProgressSummary is a read-optimized view of a user's progress.
// Used for leaderboard display and dashboard summary.
type ProgressSummary struct {
	UserID           string
	DisplayName      string
	TotalPoints      int64
	CurrentStreak    int
	LastActive       time.Time
	NextStreakExpiry time.Time
	AchievementCount int64
	RecentEvents     []EventSummary
	Rank             int
}

// LeaderboardEntry represents a single entry in the leaderboard.
type LeaderboardEntry struct {
	Rank        int
	UserID      string
	DisplayName string
	TotalPoints int64
	Streak      int
}

// NewProgress creates a new progress aggregate for a user.
// Called when a user first registers to initialize their gamification state.
func NewProgress(id, userID string) *Progress {
	now := time.Now()
	return &Progress{
		ID:            id,
		UserID:        userID,
		TotalPoints:   0,
		CurrentStreak: 0,
		LastActive:    time.Time{},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// AddPoints increments the user's total points by the given amount.
// This is a lower-level operation than RecordEvent - it just adds points without event tracking.
func (p *Progress) AddPoints(points int64) {
	if points > 0 {
		p.TotalPoints += points
		p.UpdatedAt = time.Now()
	}
}

// UpdateStreak updates the user's streak based on activity date.
// If the date is consecutive to LastActive, increments streak.
// If the date is the same as LastActive, does nothing.
// If the date is not consecutive, resets streak to 1.
func (p *Progress) UpdateStreak(activityDate time.Time) {
	y1, m1, d1 := p.LastActive.Date()
	y2, m2, d2 := activityDate.Date()

	if y1 == y2 && m1 == m2 && d1 == d2 {
		return
	}

	nextDay := time.Date(y1, m1, d1, 0, 0, 0, 0, p.LastActive.Location()).Add(24 * time.Hour)
	yNext, mNext, dNext := nextDay.Date()

	if y2 == yNext && m2 == mNext && d2 == dNext {
		p.CurrentStreak++
	} else {
		p.CurrentStreak = 1
	}

	p.LastActive = activityDate
	p.UpdatedAt = time.Now()
}

// NextStreakExpiry returns the time when the current streak will expire if no activity occurs.
// A streak expires at midnight after the last active day.
func (p *Progress) NextStreakExpiry() time.Time {
	y, m, d := p.LastActive.Date()
	nextDay := time.Date(y, m, d, 0, 0, 0, 0, p.LastActive.Location()).Add(48 * time.Hour)
	return nextDay
}

// Domain errors for Progress aggregate validation failures.
var (
	ErrNegativePoints = &ProgressError{"points cannot be negative"}
	ErrInvalidEvent   = &ProgressError{"invalid event type"}
)

// ProgressError represents domain-specific errors for the Progress aggregate.
type ProgressError struct {
	Message string
}

func (e *ProgressError) Error() string { return e.Message }

// CalculatePoints returns the points value for a given event type.
// This is a helper function used by the application service.
func CalculatePoints(eventType events.EventType) int64 {
	points := int64(events.PointValue[eventType])
	return points
}

// EventMetadata contains additional data about an event.
// Used for potential bonus point calculations.
type EventMetadata map[string]any

// NewEventLog creates a new event log entry.
func NewEventLog(userID string, eventType events.EventType, points int64, metadata EventMetadata) *EventLog {
	return &EventLog{
		UserID:        userID,
		Type:          eventType,
		PointsAwarded: points,
		Timestamp:     time.Now(),
		Metadata:      metadata,
	}
}

// NewAchievement creates a new achievement for a user.
func NewAchievement(userID string, achievementType AchievementType, points int64) *Achievement {
	return &Achievement{
		ID:          generateID(),
		UserID:      userID,
		Type:        string(achievementType),
		UnlockedAt:  time.Now(),
		PointsValue: int(points),
	}
}

// PointValue returns the point value for an achievement.
func (a *Achievement) PointValue() int64 {
	return int64(a.PointsValue)
}

func generateID() string {
	return uuid.New().String()
}
