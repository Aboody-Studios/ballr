package domain

import (
	"time"

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
	ID          string    `gorm:"primaryKey"`
	UserID      string    `gorm:"uniqueIndex:idx_user_achievement;not null"`
	Type        string    `gorm:"uniqueIndex:idx_user_achievement;not null"`
	UnlockedAt  time.Time `gorm:"autoCreateTime"`
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

// EventType categorizes activities that generate points.
type EventType string

const (
	EventMatchUploaded     EventType = "MATCH_UPLOADED"
	EventAnalysisCompleted EventType = "ANALYSIS_COMPLETED"
	EventDrillCompleted    EventType = "DRILL_COMPLETED"
	EventCoachInteraction  EventType = "COACH_INTERACTION"
	EventStreakMaintained  EventType = "STREAK_MAINTAINED"
)

// PointValue defines points awarded for each event type.
var PointValue = map[EventType]int{
	EventMatchUploaded:     50,
	EventAnalysisCompleted: 100,
	EventDrillCompleted:    25,
	EventCoachInteraction:  10,
	EventStreakMaintained:  5,
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

// RecordEvent processes an activity event and updates progress state.
// This is the primary domain method for gamification logic.
// Returns points earned from this event.
func (p *Progress) RecordEvent(eventType EventType) int {
	points := PointValue[eventType]
	p.TotalPoints += int64(points)

	now := time.Now()
	if !isSameDay(p.LastActive, now) {
		if isConsecutiveDay(p.LastActive, now) {
			p.CurrentStreak++
		} else {
			p.CurrentStreak = 1
		}
		p.LastActive = now
	}

	p.UpdatedAt = now
	return points
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

// CanUnlockStreakAchievement checks if the user qualifies for a streak badge.
func (p *Progress) CanUnlockStreakAchievement() (AchievementType, bool) {
	switch p.CurrentStreak {
	case 7:
		return AchievementTypeStreak7, true
	case 30:
		return AchievementTypeStreak30, true
	default:
		return "", false
	}
}

// AwardAchievement grants an achievement to the user.
// The achievement is a domain event that should be persisted and possibly published.
func (p *Progress) AwardAchievement(achievementType AchievementType) *Achievement {
	points := achievementPoints(achievementType)
	p.TotalPoints += int64(points)

	return &Achievement{
		UserID:      p.UserID,
		Type:        string(achievementType),
		UnlockedAt:  time.Now(),
		PointsValue: points,
	}
}

// GetLevel returns the current gamification level based on total points.
// Levels are calculated: Level = floor(sqrt(TotalPoints / 100))
func (p *Progress) GetLevel() int {
	if p.TotalPoints < 100 {
		return 1
	}
	level := 1
	pointsNeeded := int64(100)
	for p.TotalPoints >= pointsNeeded {
		level++
		pointsNeeded = int64(level * level * 100)
	}
	return level
}

// ProgressToNextLevel returns points needed to reach the next level.
func (p *Progress) ProgressToNextLevel() int64 {
	currentLevel := p.GetLevel()
	nextLevelPoints := int64(currentLevel * currentLevel * 100)
	return nextLevelPoints - p.TotalPoints
}

// isSameDay returns true if two times are on the same calendar day.
func isSameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// isConsecutiveDay returns true if t2 is the calendar day after t1.
func isConsecutiveDay(t1, t2 time.Time) bool {
	nextDay := t1.Add(24 * time.Hour)
	return isSameDay(nextDay, t2)
}

// achievementPoints returns the point value for a specific achievement type.
func achievementPoints(t AchievementType) int {
	switch t {
	case AchievementTypeFirstUpload:
		return 100
	case AchievementTypeFirstAnalysis:
		return 200
	case AchievementTypeStreakWeek:
		return 150
	case AchievementTypeStreakMonth:
		return 500
	case AchievementTypeTopPerformer:
		return 300
	case AchievementTypeCoachConsult:
		return 50
	default:
		return 0
	}
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
func CalculatePoints(eventType EventType, metadata EventMetadata) int64 {
	points := int64(PointValue[eventType])
	return points
}

// EventMetadata contains additional data about an event.
// Used for potential bonus point calculations.
type EventMetadata map[string]interface{}

// NewEventLog creates a new event log entry.
func NewEventLog(userID string, eventType EventType, points int64, metadata EventMetadata) *EventLog {
	return &EventLog{
		UserID:        userID,
		Type:          string(eventType),
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
