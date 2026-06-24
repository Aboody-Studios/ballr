package application

import (
	"context"
	"fmt"
	"time"

	progressDomain "github.com/Aboody-Studios/ballr/src/internal/progress/domain"
	"github.com/Aboody-Studios/ballr/src/pkg/events"
)

// GamificationService handles progress tracking, points calculation,
// achievements, streaks, and leaderboards for the Progress bounded context.
// Something as infra for the orb app aboody showed us
type GamificationService struct {
	progressRepo           progressDomain.ProgressRepository
	achievementRepo        progressDomain.AchievementRepository
	eventLogRepo           progressDomain.EventLogRepository
	leaderboardRepo        progressDomain.LeaderboardRepository
	progressUserBridgeRepo progressDomain.ProgressUserBridgeRepository
	transactionManager     progressDomain.TransactionManager
	EventPublisher         events.Publisher
}

// NewGamificationService creates a new gamification service with required dependencies.
func NewGamificationService(
	progressRepo progressDomain.ProgressRepository,
	achievementRepo progressDomain.AchievementRepository,
	eventLogRepo progressDomain.EventLogRepository,
	leaderboardRepo progressDomain.LeaderboardRepository,
	progressUserBridgeRepo progressDomain.ProgressUserBridgeRepository,
	transactionManager progressDomain.TransactionManager,
	eventPublisher events.Publisher,
) *GamificationService {
	return &GamificationService{
		progressRepo:           progressRepo,
		achievementRepo:        achievementRepo,
		eventLogRepo:           eventLogRepo,
		leaderboardRepo:        leaderboardRepo,
		progressUserBridgeRepo: progressUserBridgeRepo,
		transactionManager:     transactionManager,
		EventPublisher:         eventPublisher,
	}
}

// ProcessEvent handles a gamification event and calculates points/achievements.
// This is the core entry point for the event-based gamification system.
func (gs *GamificationService) GrantPoints(ctx context.Context, event events.Event) error {
	var progress *progressDomain.Progress
	var streakUpdated bool

	if err := gs.transactionManager.Transact(ctx, func(ctx context.Context) error {
		innerScopeProgress, err := gs.progressRepo.FindByUserID(ctx, event.UserID)
		if err != nil {
			id := fmt.Sprintf("prog_%d", time.Now().UnixNano())
			innerScopeProgress = progressDomain.NewProgress(id, event.UserID)
		}
		progress = innerScopeProgress

		points, ok := progressDomain.CalculatePoints(event)

		if ok {
			timeNow := time.Now()
			streakUpdated = progress.UpdateStreak(timeNow)
			progress.AddPoints(points)
		} else {
			return fmt.Errorf("achievement points aren't of type int")
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

	if err := gs.initiateAchievementsCheckAndAward(ctx, event, streakUpdated); err != nil {
		return fmt.Errorf("achievements failure: %w", err)
	}

	if err := gs.leaderboardRepo.UpdateScore(ctx, event.UserID, progress.TotalPoints); err != nil {
		return fmt.Errorf("failed to update score: %w", err)
	}

	return nil
}

func (gs *GamificationService) initiateAchievementsCheckAndAward(ctx context.Context, event events.Event, streakUpdated bool) error {
	var newAchievements []*progressDomain.Achievement
	if err := gs.transactionManager.Transact(ctx, func(ctx context.Context) error {
		progress, err := gs.progressRepo.FindByUserID(ctx, event.UserID)
		if err != nil {
			return fmt.Errorf("Failed to get progress: %w", err)
		}

		newAchievements, err = gs.checkAchievements(ctx, event.UserID, progress, event.Type, streakUpdated)
		if err != nil {
			return fmt.Errorf("failed to check achievements: %w", err)
		}

		for _, achievement := range newAchievements {
			if achievement.Badge {
				if err := gs.achievementRepo.Save(ctx, achievement); err != nil {
					return fmt.Errorf("failed to save achievement: %w", err)
				}
			}
		}

		if err := gs.progressRepo.Save(ctx, progress); err != nil {
			return fmt.Errorf("failed to save progress: %w", err)
		}
		return nil

	}); err != nil {
		return err
	}

	gs.awardNewAchievements(ctx, newAchievements)
	return nil
}

// checkAchievements evaluates if user qualifies for any new achievements.
// Checks against all achievement criteria and returns newly unlocked achievements.
func (s *GamificationService) checkAchievements(ctx context.Context, userID string,
	progress *progressDomain.Progress, recentEvent events.EventType, streakUpdated bool) ([]*progressDomain.Achievement, error) {
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
		typ       progressDomain.AchievementType
		checker   func(*progressDomain.Progress, []progressDomain.AchievementType) bool
		points    int
		badgebool bool
	}{
		{
			progressDomain.AchievementTypeFirstUpload,
			func(p *progressDomain.Progress, existing []progressDomain.AchievementType) bool {
				return recentEvent == events.EventMatchUploaded &&
					!contains(existing, progressDomain.AchievementTypeFirstUpload)
			},
			100,
			true,
		},
		{
			// if p.CurrentStreak % 7 == 0 and streakUpdated is false this means that the user has already been awarded points for
			// AchievementTypeStreakWeek today, i.e. if today is the 7th consecutive day for the user.
			// if streakUpdated is true, this means that it has just been updated currently, and that the still has NOT been awarded
			// the points yet.
			progressDomain.AchievementTypeStreakWeek,
			func(p *progressDomain.Progress, existing []progressDomain.AchievementType) bool {
				return p.CurrentStreak%7 == 0 && streakUpdated == true
			},
			500,
			false,
		},
		{
			progressDomain.AchievementTypeStreakMonth,
			func(p *progressDomain.Progress, existing []progressDomain.AchievementType) bool {
				return p.CurrentStreak >= 30 &&
					!contains(existing, progressDomain.AchievementTypeStreakMonth)
			},
			2000,
			true,
		},
		{
			progressDomain.AchievementTypeAnalysisMaster,
			func(p *progressDomain.Progress, existing []progressDomain.AchievementType) bool {
				return p.TotalPoints >= 5000 &&
					!contains(existing, progressDomain.AchievementTypeAnalysisMaster)
			},
			1000,
			true,
		},
		{
			progressDomain.AchievementTypeDrillCompleter,
			func(p *progressDomain.Progress, existing []progressDomain.AchievementType) bool {
				return recentEvent == events.EventDrillCompleted &&
					!contains(existing, progressDomain.AchievementTypeDrillCompleter)
			},
			250,
			true,
		},
	}

	for _, c := range criteria {
		if c.checker(progress, existingTypes) {
			achievement := progressDomain.NewAchievement(userID, c.typ, c.points, c.badgebool)
			newAchievements = append(newAchievements, achievement)
		}
	}

	return newAchievements, nil
}

func (gs *GamificationService) awardNewAchievements(ctx context.Context, newAchievements []*progressDomain.Achievement) {
	for _, achievement := range newAchievements {
		metadata := map[string]any{
			"points": achievement.PointsValue,
		}

		if err := gs.EventPublisher.PublishEvent(ctx, events.Event{Type: events.EventAchievementCompleted, UserID: achievement.UserID, Metadata: metadata}); err != nil {
			// Log but don't fail the award flow on publish errors.
			// Publisher failure should not block achievement granting.
		}
	}
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

	nameMap, err := s.progressUserBridgeRepo.GetUsernames(ctx, userIDs)
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
		return 0, err
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

func (s *GamificationService) Get8pmAndTrainingDayUsersService(ctx context.Context) ([]progressDomain.NotificationTarget, error) {
	targets, err := s.progressUserBridgeRepo.GetEightPmAndTrainingDayUsers(ctx)
	if err != nil {
		return nil, err
	}

	return targets, nil
}
