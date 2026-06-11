package application

import (
	"context"
	"fmt"
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/progress/domain"
	"github.com/Aboody-Studios/ballr/src/pkg/events"
	"github.com/google/uuid"
)

// GamificationService handles progress tracking, points calculation,
// achievements, streaks, and leaderboards for the Progress bounded context.
// Something as infra for the orb app aboody showed us
type GamificationService struct {
	progressRepo       domain.ProgressRepository
	achievementRepo    domain.AchievementRepository
	eventLogRepo       domain.EventLogRepository
	leaderboardRepo    domain.LeaderboardRepository
	TransactionManager domain.TransactionManager
	EventPublisher     events.Publisher
}

// NewGamificationService creates a new gamification service with required dependencies.
func NewGamificationService(
	progressRepo domain.ProgressRepository,
	achievementRepo domain.AchievementRepository,
	eventLogRepo domain.EventLogRepository,
	leaderboardRepo domain.LeaderboardRepository,
) *GamificationService {
	return &GamificationService{
		progressRepo:    progressRepo,
		achievementRepo: achievementRepo,
		eventLogRepo:    eventLogRepo,
		leaderboardRepo: leaderboardRepo,
	}
}

// ProcessEvent handles a gamification event and calculates points/achievements.
// This is the core entry point for the event-based gamification system.
// TODO!: Make this function only for processing events with points.
func (gs *GamificationService) ProcessEvent(ctx context.Context, event events.Event) error {
	var progress *domain.Progress

	if err := gs.TransactionManager.Transact(ctx, func(ctx context.Context) error {
		innerScopeProgress, err := gs.progressRepo.FindByUserID(ctx, event.UserID)
		if err != nil {
			id := fmt.Sprintf("prog_%d", time.Now().UnixNano())
			innerScopeProgress = domain.NewProgress(id, event.UserID)
		}
		progress = innerScopeProgress

		points, ok := domain.CalculatePoints(event.Type)

		if ok {
			timeNow := time.Now()
			progress.UpdateStreak(timeNow)
			progress.AddPoints(points, timeNow)
		}

		if err := gs.progressRepo.Save(ctx, progress); err != nil {
			return fmt.Errorf("failed to save progress: %w", err)
		}

		eventLog := domain.NewEventLog(event, points)
		if err := gs.eventLogRepo.Save(ctx, eventLog); err != nil {
			return fmt.Errorf("failed to save event log: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("Transaction failed: %w", err)
	}

	// Publish an event for checking if rewarding an achievement to the user is applicable or not.
	checkAchievementsEvent := events.Event{
		ID:        uuid.NewString(),
		Type:      events.EventAchievementCheckRequested,
		UserID:    event.UserID,
		Timestamp: time.Now(),
	}
	gs.EventPublisher.PublishEvent(ctx, checkAchievementsEvent)

	//TODO!: Implement leaderboard using Redis and not PostgreSQL
	if err := gs.leaderboardRepo.UpdateScore(ctx, event.UserID, progress.TotalPoints); err != nil {
		fmt.Errorf("failed to update score: %w", err)
	}

	return nil
}

func (gs *GamificationService) CheckAndAwardAchievements(ctx context.Context, event events.Event) error {
	if err := gs.TransactionManager.Transact(ctx, func(ctx context.Context) error {

		progress, err := gs.progressRepo.FindByUserID(ctx, event.UserID)
		if err != nil {
			return fmt.Errorf("Failed to get progress: %w", err)
		}

		newAchievements, err := gs.checkAchievements(ctx, event.UserID, progress, event.Type)
		if err != nil {
			return fmt.Errorf("failed to check achievements: %w", err)
		}

		for _, achievement := range newAchievements {
			if err := gs.achievementRepo.Save(ctx, achievement); err != nil {
				return fmt.Errorf("failed to save achievement: %w", err)
			}

			bonusPoints := achievement.PointValue()
			progress.AddPoints(bonusPoints, time.Now())
		}

		if err := gs.progressRepo.Save(ctx, progress); err != nil {
			return fmt.Errorf("failed to save progress: %w", err)
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// checkAchievements evaluates if user qualifies for any new achievements.
// Checks against all achievement criteria and returns newly unlocked achievements.
func (s *GamificationService) checkAchievements(ctx context.Context, userID string, progress *domain.Progress, recentEvent events.EventType) ([]*domain.Achievement, error) {
	var newAchievements []*domain.Achievement

	existingAchievements, err := s.achievementRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	existingTypes := make([]domain.AchievementType, len(existingAchievements))
	for i, a := range existingAchievements {
		existingTypes[i] = domain.AchievementType(a.Type)
	}

	criteria := []struct {
		typ     domain.AchievementType
		checker func(*domain.Progress, []domain.AchievementType) bool
		points  int
	}{
		{
			domain.AchievementTypeFirstUpload,
			func(p *domain.Progress, existing []domain.AchievementType) bool {
				return recentEvent == events.EventMatchUploaded &&
					!contains(existing, domain.AchievementTypeFirstUpload)
			},
			100,
		},
		{
			domain.AchievementTypeStreakWeek,
			func(p *domain.Progress, existing []domain.AchievementType) bool {
				return p.CurrentStreak >= 7 &&
					!contains(existing, domain.AchievementTypeStreakWeek)
			},
			500,
		},
		{
			domain.AchievementTypeStreakMonth,
			func(p *domain.Progress, existing []domain.AchievementType) bool {
				return p.CurrentStreak >= 30 &&
					!contains(existing, domain.AchievementTypeStreakMonth)
			},
			2000,
		},
		{
			domain.AchievementTypeAnalysisMaster,
			func(p *domain.Progress, existing []domain.AchievementType) bool {
				return p.TotalPoints >= 5000 &&
					!contains(existing, domain.AchievementTypeAnalysisMaster)
			},
			1000,
		},
		{
			domain.AchievementTypeDrillCompleter,
			func(p *domain.Progress, existing []domain.AchievementType) bool {
				return recentEvent == events.EventDrillCompleted &&
					!contains(existing, domain.AchievementTypeDrillCompleter)
			},
			250,
		},
	}

	for _, c := range criteria {
		if c.checker(progress, existingTypes) {
			achievement := domain.NewAchievement(userID, c.typ, c.points)
			newAchievements = append(newAchievements, achievement)
		}
	}

	return newAchievements, nil
}

// GetProgressSummary returns the user's complete progress summary.
// Includes total points, current streak, and recent activity.
func (s *GamificationService) GetProgressSummary(ctx context.Context, userID string) (*domain.ProgressSummary, error) {
	progress, err := s.progressRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to load progress: %w", err)
	}

	achievements, err := s.achievementRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to load achievements: %w", err)
	}

	recentEvents, err := s.eventLogRepo.FindRecentByUserID(ctx, userID, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to find recent user activities: %w", err)
	}

	summary := &domain.ProgressSummary{
		UserID:           userID,
		TotalPoints:      progress.TotalPoints,
		CurrentStreak:    progress.CurrentStreak,
		LastActive:       progress.LastActive,
		NextStreakExpiry: progress.NextStreakExpiry(),
		AchievementCount: len(achievements),
		RecentEvents:     make([]domain.EventSummary, 0, len(recentEvents)),
	}

	for _, event := range recentEvents {
		summary.RecentEvents = append(summary.RecentEvents, domain.EventSummary{
			Type:      string(event.Type),
			Points:    event.PointsAwarded,
			Timestamp: event.Timestamp,
		})
	}

	return summary, nil
}

// GetLeaderboard returns the global or friend-group leaderboard.
// Supports pagination with offset and limit.
func (s *GamificationService) GetLeaderboard(ctx context.Context, offset, limit int) ([]domain.LeaderboardEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	return s.leaderboardRepo.GetTopPlayers(ctx, offset, limit)
}

// GetAchievements returns all achievements for a user.
func (s *GamificationService) GetAchievements(ctx context.Context, userID string) ([]*domain.Achievement, error) {
	return s.achievementRepo.FindByUserID(ctx, userID)
}

func contains(slice []domain.AchievementType, item domain.AchievementType) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
