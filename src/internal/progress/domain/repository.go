package domain

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// ProgressRepository defines the contract for user progress persistence.
type ProgressRepository interface {
	// Save creates or updates a user's progress record.
	Save(ctx context.Context, progress *Progress) error

	// FindByUserID retrieves the progress record for a specific user.
	FindByUserID(ctx context.Context, userID string) (*Progress, error)
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

type LeaderboardRepository interface {
	// UpdateScore updates a user's score on the leaderboard.
	UpdateScore(ctx context.Context, userID string, points int) error

	GetPlayers(ctx context.Context, offset, limit int64) ([]redis.Z, error)
}

type TransactionManager interface {
	Transact(ctx context.Context, fn func(ctx context.Context) error) error
}

var ErrAchievementNotFound = errAchievementNotFound{}

type errAchievementNotFound struct{}

func (errAchievementNotFound) Error() string { return "achievement not found" }
