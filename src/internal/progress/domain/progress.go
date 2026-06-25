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
	TotalPoints   int    `gorm:"default:0"`
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
	ProgressID  string
	User        domain.User     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	UserID      string          `gorm:"uniqueIndex:idx_user_achievement;not null"`
	Type        AchievementType `gorm:"uniqueIndex:idx_user_achievement;not null"`
	UnlockedAt  time.Time       `gorm:"autoCreateTime"`
	PointsValue int
	Badge       bool
}

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
	User          domain.User      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	UserID        string           `gorm:"index:idx_user_event;not null"`
	Type          events.EventType `gorm:"index:idx_user_event;not null"`
	PointsAwarded int
	ID            string         `gorm:"primaryKey"`
	Timestamp     time.Time      `gorm:"index"`
	Metadata      map[string]any `gorm:"type:jsonb;serializer:json"`
}

type EventSummary struct {
	Type      string
	Points    int
	Timestamp time.Time
}

// ProgressSummary is a read-optimized view of a user's progress.
// Used for leaderboard display and dashboard summary.
type ProgressSummary struct {
	UserID           string
	DisplayName      string
	TotalPoints      int
	CurrentStreak    int
	LastActive       time.Time
	NextStreakExpiry time.Time
	AchievementCount int
	RecentEvents     []EventSummary
	Rank             int
}

type NotificationTarget struct {
	ID                string
	NotificationToken string
}

type LeaderboardEntry struct {
	Rank        int
	UserID      string
	DisplayName string
	TotalPoints int64
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

func (p *Progress) AddPoints(points int) {
	if points <= 0 {
		return
	}
	p.TotalPoints += points
	p.UpdatedAt = time.Now()
}

// RecordEvent records a gamification event and awards points accordingly.
func (p *Progress) RecordEvent(eventType events.EventType) int {
	points, ok := events.PointValue[eventType]
	if !ok {
		return 0
	}

	p.AddPoints(points)
	return points
}

// UpdateStreak updates the user's streak based on activity date.
// If the date is consecutive to LastActive, increments streak.
// If the date is the same as LastActive, does nothing.
// If the date is not consecutive, resets streak to 1.
func (p *Progress) UpdateStreak(activityDate time.Time) bool {
	lastActiveYear, lastActiveMonth, LastActiveDay := p.LastActive.Date()
	currentActivityYear, currentActivityMonth, currentActivityDay := activityDate.Date()

	if lastActiveYear == currentActivityYear && lastActiveMonth == currentActivityMonth && LastActiveDay == currentActivityDay {
		return false
	}

	nextDay := time.Date(lastActiveYear, lastActiveMonth, LastActiveDay, 0, 0, 0, 0, p.LastActive.Location()).Add(24 * time.Hour)
	yNext, mNext, dNext := nextDay.Date()

	if currentActivityYear == yNext && currentActivityMonth == mNext && currentActivityDay == dNext {
		p.CurrentStreak++
	} else {
		p.CurrentStreak = 1
	}

	p.LastActive = activityDate
	p.UpdatedAt = time.Now()
	return true
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
func CalculatePoints(event events.Event) (int, bool) {
	if event.Type == events.EventAchievementCompleted {
		points, ok := event.Metadata["points"].(float64)
		if !ok {
			return 0, false
		}
		return int(points), true
	}

	points, ok := events.PointValue[event.Type]
	return points, ok
}

// EventMetadata contains additional data about an event.
// Used for potential bonus point calculations.
type EventMetadata map[string]any

func NewEventLog(event events.Event, points int) *EventLog {
	return &EventLog{
		UserID:        event.UserID,
		ID:            uuid.NewString(),
		Type:          event.Type,
		PointsAwarded: points,
		Timestamp:     event.Timestamp,
		Metadata:      event.Metadata,
	}
}

// NewAchievement creates a new achievement for a user.
func NewAchievement(userID string, achievementType AchievementType, points int, badgeBool bool) *Achievement {
	return &Achievement{
		ID:          generateID(),
		UserID:      userID,
		Type:        achievementType,
		UnlockedAt:  time.Now(),
		PointsValue: points,
		Badge:       badgeBool,
	}
}

// AwardAchievement creates and returns a new Achievement for the progress owner.
func (p *Progress) AwardAchievement(typ AchievementType) *Achievement {
	var points int
	var badge bool
	switch typ {
	case AchievementTypeFirstUpload:
		points = 100
		badge = true
	case AchievementTypeStreakWeek:
		points = 500
		badge = false
	default:
		points = 100
		badge = false
	}

	ach := NewAchievement(p.UserID, typ, points, badge)
	p.AddPoints(points)
	return ach
}

// PointValue returns the point value for an achievement.
func (a *Achievement) PointValue() int {
	return a.PointsValue
}

func generateID() string {
	return uuid.New().String()
}
