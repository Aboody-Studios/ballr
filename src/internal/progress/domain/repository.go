package domain

import (
	"context"
	"time"
)

// ProgressRepository defines the contract for user progress persistence.
type ProgressRepository interface {
	// Save creates or updates a user's progress record.
	Save(ctx context.Context, progress *Progress) error

	// FindByUserID retrieves the progress record for a specific user.
	FindByUserID(ctx context.Context, userID string) (*Progress, error)

	// AddPoints increments the user's total points by the given amount.
	AddPoints(ctx context.Context, userID string, points int64) error

	// UpdateStreak updates the current streak and last active date.
	// Streak logic: if last_active was yesterday, increment streak; if older, reset to 1.
	UpdateStreak(ctx context.Context, userID string, activeDate string) error

	// GetLeaderboard retrieves top users by points for the leaderboard.
	GetLeaderboard(ctx context.Context, limit int) ([]*LeaderboardEntry, error)
}

// AchievementRepository defines the contract for achievement persistence.
type AchievementRepository interface {
	// Save records a new unlocked achievement for a user.
	Save(ctx context.Context, achievement *Achievement) error

	// FindByUserID retrieves all achievements for a specific user.
	FindByUserID(ctx context.Context, userID string) ([]*Achievement, error)

	// FindByType retrieves all users who have unlocked a specific achievement type.
	FindByType(ctx context.Context, achievementType string) ([]*Achievement, error)

	// HasAchievement checks if a user has already unlocked a specific achievement.
	HasAchievement(ctx context.Context, userID string, achievementType string) (bool, error)
}

// ErrProgressNotFound is returned when a progress record is not found.
var ErrProgressNotFound = errProgressNotFound{}

type errProgressNotFound struct{}

func (errProgressNotFound) Error() string { return "progress record not found" }

// EventLogRepository defines the contract for logging gamification events.
type EventLogRepository interface {
	// Save logs a gamification event.
	Save(ctx context.Context, event *EventLog) error

	// FindRecentByUserID retrieves recent events for a user's activity feed.
	FindRecentByUserID(ctx context.Context, userID string, limit int) ([]*EventLog, error)
}

// EventLog represents a single gamification event for analytics.
type EventLog struct {
	UserID        string
	Type          string
	PointsAwarded int64
	Timestamp     time.Time
	Metadata      map[string]interface{}
}

// LeaderboardRepository defines the contract for leaderboard operations.
type LeaderboardRepository interface {
	// UpdateScore updates a user's score on the leaderboard.
	UpdateScore(ctx context.Context, userID string, points int64) error

	// GetTopPlayers retrieves the top players for the leaderboard.
	GetTopPlayers(ctx context.Context, offset, limit int) ([]LeaderboardEntry, error)
}

// ErrAchievementNotFound is returned when an achievement is not found.
var ErrAchievementNotFound = errAchievementNotFound{}

type errAchievementNotFound struct{}

func (errAchievementNotFound) Error() string { return "achievement not found" }
