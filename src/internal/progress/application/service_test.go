package application

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/progress/domain"
)

type mockProgressRepo struct {
	mu       sync.Mutex
	progress map[string]*domain.Progress
}

func newMockProgressRepo() *mockProgressRepo {
	return &mockProgressRepo{progress: make(map[string]*domain.Progress)}
}

func (r *mockProgressRepo) Save(_ context.Context, p *domain.Progress) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.progress[p.UserID] = p
	return nil
}

func (r *mockProgressRepo) FindByUserID(_ context.Context, userID string) (*domain.Progress, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.progress[userID]; ok {
		return p, nil
	}
	return nil, domain.ErrProgressNotFound
}

func (r *mockProgressRepo) AddPoints(_ context.Context, _ string, _ int64) error {
	return nil
}

func (r *mockProgressRepo) UpdateStreak(_ context.Context, _ string, _ string) error {
	return nil
}

func (r *mockProgressRepo) GetLeaderboard(_ context.Context, _ int) ([]*domain.LeaderboardEntry, error) {
	return nil, nil
}

type mockAchievementRepo struct {
	mu           sync.Mutex
	achievements map[string][]*domain.Achievement
}

func newMockAchievementRepo() *mockAchievementRepo {
	return &mockAchievementRepo{achievements: make(map[string][]*domain.Achievement)}
}

func (r *mockAchievementRepo) Save(_ context.Context, a *domain.Achievement) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.achievements[a.UserID] = append(r.achievements[a.UserID], a)
	return nil
}

func (r *mockAchievementRepo) FindByUserID(_ context.Context, userID string) ([]*domain.Achievement, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.achievements[userID], nil
}

func (r *mockAchievementRepo) FindByType(_ context.Context, _ string) ([]*domain.Achievement, error) {
	return nil, nil
}

func (r *mockAchievementRepo) HasAchievement(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

type mockEventLogRepo struct {
	mu   sync.Mutex
	logs []*domain.EventLog
}

func newMockEventLogRepo() *mockEventLogRepo {
	return &mockEventLogRepo{}
}

func (r *mockEventLogRepo) Save(_ context.Context, e *domain.EventLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logs = append(r.logs, e)
	return nil
}

func (r *mockEventLogRepo) FindRecentByUserID(_ context.Context, _ string, _ int) ([]*domain.EventLog, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.logs, nil
}

type mockLeaderboardRepo struct {
	mu     sync.Mutex
	scores map[string]int64
}

func newMockLeaderboardRepo() *mockLeaderboardRepo {
	return &mockLeaderboardRepo{scores: make(map[string]int64)}
}

func (r *mockLeaderboardRepo) UpdateScore(_ context.Context, userID string, points int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.scores[userID] = points
	return nil
}

func (r *mockLeaderboardRepo) GetTopPlayers(_ context.Context, offset, limit int) ([]domain.LeaderboardEntry, error) {
	return []domain.LeaderboardEntry{}, nil
}

func TestProcessEvent_NewUser(t *testing.T) {
	progressRepo := newMockProgressRepo()
	achievementRepo := newMockAchievementRepo()
	eventLogRepo := newMockEventLogRepo()
	leaderboardRepo := newMockLeaderboardRepo()
	svc := NewGamificationService(progressRepo, achievementRepo, eventLogRepo, leaderboardRepo)

	err := svc.ProcessEvent(context.Background(), "user-1", domain.EventMatchUploaded, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	p, err := progressRepo.FindByUserID(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("progress not found: %v", err)
	}
	if p.TotalPoints != 150 {
		t.Errorf("expected 150 points (50 event + 100 achievement), got %d", p.TotalPoints)
	}
	if p.CurrentStreak != 1 {
		t.Errorf("expected streak 1, got %d", p.CurrentStreak)
	}
}

func TestProcessEvent_ExistingUser(t *testing.T) {
	progressRepo := newMockProgressRepo()
	p := domain.NewProgress("prog-1", "user-1")
	p.TotalPoints = 100
	progressRepo.Save(context.Background(), p)

	achievementRepo := newMockAchievementRepo()
	eventLogRepo := newMockEventLogRepo()
	leaderboardRepo := newMockLeaderboardRepo()
	svc := NewGamificationService(progressRepo, achievementRepo, eventLogRepo, leaderboardRepo)

	err := svc.ProcessEvent(context.Background(), "user-1", domain.EventAnalysisCompleted, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	p2, _ := progressRepo.FindByUserID(context.Background(), "user-1")
	if p2.TotalPoints != 200 {
		t.Errorf("expected 200 points, got %d", p2.TotalPoints)
	}
}

func TestProcessEvent_CoachInteractionNoStreakUpdate(t *testing.T) {
	progressRepo := newMockProgressRepo()
	achievementRepo := newMockAchievementRepo()
	eventLogRepo := newMockEventLogRepo()
	leaderboardRepo := newMockLeaderboardRepo()
	svc := NewGamificationService(progressRepo, achievementRepo, eventLogRepo, leaderboardRepo)

	svc.ProcessEvent(context.Background(), "user-1", domain.EventCoachInteraction, nil)
	p, _ := progressRepo.FindByUserID(context.Background(), "user-1")
	if p.TotalPoints != 10 {
		t.Errorf("expected 10 points, got %d", p.TotalPoints)
	}
}

func TestProcessEvent_FirstUploadAchievement(t *testing.T) {
	progressRepo := newMockProgressRepo()
	achievementRepo := newMockAchievementRepo()
	eventLogRepo := newMockEventLogRepo()
	leaderboardRepo := newMockLeaderboardRepo()
	svc := NewGamificationService(progressRepo, achievementRepo, eventLogRepo, leaderboardRepo)

	err := svc.ProcessEvent(context.Background(), "user-1", domain.EventMatchUploaded, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	achievements, _ := achievementRepo.FindByUserID(context.Background(), "user-1")
	hasFirstUpload := false
	for _, a := range achievements {
		if a.Type == string(domain.AchievementTypeFirstUpload) {
			hasFirstUpload = true
			break
		}
	}
	if !hasFirstUpload {
		t.Error("expected FIRST_UPLOAD achievement to be unlocked")
	}

	p, _ := progressRepo.FindByUserID(context.Background(), "user-1")
	if p.TotalPoints < 150 {
		t.Errorf("expected at least 150 points (50 base + 100 achievement), got %d", p.TotalPoints)
	}
}

func TestGetProgressSummary(t *testing.T) {
	progressRepo := newMockProgressRepo()
	progressRepo.Save(context.Background(), &domain.Progress{
		ID: "p-1", UserID: "user-1", TotalPoints: 500, CurrentStreak: 7, LastActive: time.Now(),
	})

	achievementRepo := newMockAchievementRepo()
	achievementRepo.Save(context.Background(), &domain.Achievement{
		UserID: "user-1", Type: string(domain.AchievementTypeFirstUpload), UnlockedAt: time.Now(),
	})

	eventLogRepo := newMockEventLogRepo()
	leaderboardRepo := newMockLeaderboardRepo()
	svc := NewGamificationService(progressRepo, achievementRepo, eventLogRepo, leaderboardRepo)

	summary, err := svc.GetProgressSummary(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.TotalPoints != 500 {
		t.Errorf("expected 500, got %d", summary.TotalPoints)
	}
	if summary.CurrentStreak != 7 {
		t.Errorf("expected streak 7, got %d", summary.CurrentStreak)
	}
	if summary.AchievementCount != 1 {
		t.Errorf("expected 1 achievement, got %d", summary.AchievementCount)
	}
}

func TestGetLeaderboard(t *testing.T) {
	progressRepo := newMockProgressRepo()
	achievementRepo := newMockAchievementRepo()
	eventLogRepo := newMockEventLogRepo()
	leaderboardRepo := newMockLeaderboardRepo()
	svc := NewGamificationService(progressRepo, achievementRepo, eventLogRepo, leaderboardRepo)

	entries, err := svc.GetLeaderboard(context.Background(), 0, 25)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entries == nil {
		t.Error("expected non-nil entries")
	}
}

func TestGetAchievements(t *testing.T) {
	achievementRepo := newMockAchievementRepo()
	achievementRepo.Save(context.Background(), &domain.Achievement{
		UserID: "user-1", Type: string(domain.AchievementTypeFirstUpload),
	})
	achievementRepo.Save(context.Background(), &domain.Achievement{
		UserID: "user-1", Type: string(domain.AchievementTypeStreakWeek),
	})

	progressRepo := newMockProgressRepo()
	eventLogRepo := newMockEventLogRepo()
	leaderboardRepo := newMockLeaderboardRepo()
	svc := NewGamificationService(progressRepo, achievementRepo, eventLogRepo, leaderboardRepo)

	achievements, err := svc.GetAchievements(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(achievements) != 2 {
		t.Errorf("expected 2 achievements, got %d", len(achievements))
	}
}
