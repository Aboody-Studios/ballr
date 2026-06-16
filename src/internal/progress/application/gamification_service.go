package application

import (
	"context"
	"fmt"
	"time"

	userDomain "github.com/Aboody-Studios/ballr/src/internal/identity/domain"
	progressDomain "github.com/Aboody-Studios/ballr/src/internal/progress/domain"
	"github.com/Aboody-Studios/ballr/src/pkg/events"
	"github.com/google/uuid"
)

// GamificationService handles progress tracking, points calculation,
// achievements, streaks, and leaderboards for the Progress bounded context.
// Something as infra for the orb app aboody showed us
type GamificationService struct {
	progressRepo       progressDomain.ProgressRepository
	achievementRepo    progressDomain.AchievementRepository
	eventLogRepo       progressDomain.EventLogRepository
	leaderboardRepo    progressDomain.LeaderboardRepository
	userRepo           userDomain.UserRepository
	TransactionManager progressDomain.TransactionManager
	EventPublisher     events.Publisher
}

// NewGamificationService creates a new gamification service with required dependencies.
func NewGamificationService(
	progressRepo progressDomain.ProgressRepository,
	achievementRepo progressDomain.AchievementRepository,
	eventLogRepo progressDomain.EventLogRepository,
	leaderboardRepo progressDomain.LeaderboardRepository,
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
	var progress *progressDomain.Progress

	if err := gs.TransactionManager.Transact(ctx, func(ctx context.Context) error {
		innerScopeProgress, err := gs.progressRepo.FindByUserID(ctx, event.UserID)
		if err != nil {
			id := fmt.Sprintf("prog_%d", time.Now().UnixNano())
			innerScopeProgress = progressDomain.NewProgress(id, event.UserID)
		}
		progress = innerScopeProgress

		points, ok := progressDomain.CalculatePoints(event.Type)

		if ok {
			timeNow := time.Now()
			progress.UpdateStreak(timeNow)
			progress.AddPoints(points, timeNow)
		}

		if err := gs.progressRepo.Save(ctx, progress); err != nil {
			return fmt.Errorf("failed to save progress: %w", err)
		}

		eventLog := progressDomain.NewEventLog(event, points)
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
func (s *GamificationService) checkAchievements(ctx context.Context, userID string, progress *progressDomain.Progress, recentEvent events.EventType) ([]*progressDomain.Achievement, error) {
	var newAchievements []*progressDomain.Achievement

	existingAchievements, err := s.achievementRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	existingTypes := make([]progressDomain.AchievementType, len(existingAchievements))
	for i, a := range existingAchievements {
		existingTypes[i] = progressDomain.AchievementType(a.Type)
	}

	criteria := []struct {
		typ     progressDomain.AchievementType
		checker func(*progressDomain.Progress, []progressDomain.AchievementType) bool
		points  int
	}{
		{
			progressDomain.AchievementTypeFirstUpload,
			func(p *progressDomain.Progress, existing []progressDomain.AchievementType) bool {
				return recentEvent == events.EventMatchUploaded &&
					!contains(existing, progressDomain.AchievementTypeFirstUpload)
			},
			100,
		},
		{
			progressDomain.AchievementTypeStreakWeek,
			func(p *progressDomain.Progress, existing []progressDomain.AchievementType) bool {
				return p.CurrentStreak >= 7 &&
					!contains(existing, progressDomain.AchievementTypeStreakWeek)
			},
			500,
		},
		{
			progressDomain.AchievementTypeStreakMonth,
			func(p *progressDomain.Progress, existing []progressDomain.AchievementType) bool {
				return p.CurrentStreak >= 30 &&
					!contains(existing, progressDomain.AchievementTypeStreakMonth)
			},
			2000,
		},
		{
			progressDomain.AchievementTypeAnalysisMaster,
			func(p *progressDomain.Progress, existing []progressDomain.AchievementType) bool {
				return p.TotalPoints >= 5000 &&
					!contains(existing, progressDomain.AchievementTypeAnalysisMaster)
			},
			1000,
		},
		{
			progressDomain.AchievementTypeDrillCompleter,
			func(p *progressDomain.Progress, existing []progressDomain.AchievementType) bool {
				return recentEvent == events.EventDrillCompleted &&
					!contains(existing, progressDomain.AchievementTypeDrillCompleter)
			},
			250,
		},
	}

	for _, c := range criteria {
		if c.checker(progress, existingTypes) {
			achievement := progressDomain.NewAchievement(userID, c.typ, c.points)
			newAchievements = append(newAchievements, achievement)
		}
	}

	return newAchievements, nil
}

// GetProgressSummary returns the user's complete progress summary.
// Includes total points, current streak, and recent activity.
func (s *GamificationService) GetProgressSummary(ctx context.Context, userID string) (*progressDomain.ProgressSummary, error) {
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

	summary := &progressDomain.ProgressSummary{
		UserID:           userID,
		TotalPoints:      progress.TotalPoints,
		CurrentStreak:    progress.CurrentStreak,
		LastActive:       progress.LastActive,
		NextStreakExpiry: progress.NextStreakExpiry(),
		AchievementCount: len(achievements),
		RecentEvents:     make([]progressDomain.EventSummary, 0, len(recentEvents)),
	}

	for _, event := range recentEvents {
		summary.RecentEvents = append(summary.RecentEvents, progressDomain.EventSummary{
			Type:      string(event.Type),
			Points:    event.PointsAwarded,
			Timestamp: event.Timestamp,
		})
	}

	return summary, nil
}

// GetLeaderboard returns the global or friend-group leaderboard.
// Supports pagination with offset and limit. Offset is necesssary because each time the user wants to fetch new users on the leaderboard,
// a frontend request is fired with a new offset. While limit variable is important for future features.
func (s *GamificationService) GetLeaderboard(ctx context.Context, offset, limit int64) ([]progressDomain.LeaderboardEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	redisZ, err := s.leaderboardRepo.GetPlayers(ctx, offset, limit)
	if err != nil {
		return nil, err
	}

	if len(redisZ) == 0 {
		return []progressDomain.LeaderboardEntry{}, nil
	}

	userIDs := make([]string, len(redisZ))
	for i, z := range redisZ {
		userIDs[i] = z.Member.(string)
	}

	nameMap, err := s.userRepo.GetUsernames(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	leaderboard := make([]progressDomain.LeaderboardEntry, len(redisZ))

	for i, value := range redisZ {

		leaderboard[i] = progressDomain.LeaderboardEntry{
			Rank:        int(offset) + i + 1,
			UserID:      userIDs[i],
			DisplayName: nameMap[userIDs[i]],
			TotalPoints: int64(value.Score),
		}
	}

	return leaderboard, nil
}

func (s *GamificationService) GetUserOffset(ctx context.Context, userID string) (int64, error) {
	offset, err := s.leaderboardRepo.GetPlayerOffset(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to get user offset")
	}

	return offset, nil
}

// GetAchievements returns all achievements for a user.
func (s *GamificationService) GetAchievements(ctx context.Context, userID string) ([]*progressDomain.Achievement, error) {
	return s.achievementRepo.FindByUserID(ctx, userID)
}

func contains(slice []progressDomain.AchievementType, item progressDomain.AchievementType) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
