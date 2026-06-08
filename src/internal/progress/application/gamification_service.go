package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/progress/domain"
	"github.com/Aboody-Studios/ballr/src/pkg/events"
)

// GamificationService handles progress tracking, points calculation,
// achievements, streaks, and leaderboards for the Progress bounded context.
// Something as infra for the orb app aboody showed us
type GamificationService struct {
	progressRepo    domain.ProgressRepository
	achievementRepo domain.AchievementRepository
	eventLogRepo    domain.EventLogRepository
	leaderboardRepo domain.LeaderboardRepository
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
// TODO!: Pass the whole Event struct instead of fields
func (s *GamificationService) ProcessEvent(ctx context.Context, userID string, eventType events.EventType, eventID string) error {
	progress, err := s.progressRepo.FindByUserID(ctx, userID)
	if err != nil {
		id := fmt.Sprintf("prog_%d", time.Now().UnixNano())
		progress = domain.NewProgress(id, userID)
	}

	points := domain.CalculatePoints(eventType)

	isActiveDay := eventType == events.EventMatchUploaded ||
		eventType == events.EventDrillCompleted
	if isActiveDay {
		progress.UpdateStreak(time.Now())
	}

	progress.AddPoints(points)

	if err := s.progressRepo.Save(ctx, progress); err != nil {
		return fmt.Errorf("failed to save progress: %w", err)
	}

	event := domain.NewEventLog(userID, eventType, points, metadata)
	if err := s.eventLogRepo.Save(ctx, event); err != nil {

	}

	newAchievements, err := s.checkAchievements(ctx, userID, progress, eventType)
	if err != nil {
		return fmt.Errorf("failed to check achievements: %w", err)
	}

	for _, achievement := range newAchievements {
		if err := s.achievementRepo.Save(ctx, achievement); err != nil {
			return fmt.Errorf("failed to save achievement: %w", err)
		}

		bonusPoints := achievement.PointValue()
		progress.AddPoints(bonusPoints)
	}

	if err := s.leaderboardRepo.UpdateScore(ctx, userID, progress.TotalPoints); err != nil {
		// Non-critical: leaderboard can be eventually consistent
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
		points  int64
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
		AchievementCount: int64(len(achievements)),
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
