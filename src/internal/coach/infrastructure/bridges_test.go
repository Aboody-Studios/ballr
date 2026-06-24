package infrastructure

import (
	"context"
	"sync"
	"testing"
	"time"

	identitydomain "github.com/Aboody-Studios/ballr/src/internal/identity/domain"
	analysisdomain "github.com/Aboody-Studios/ballr/src/internal/match/domain"
)

type mockMatchRepo struct {
	mu      sync.Mutex
	matches map[string]*analysisdomain.Match
}

func newMockMatchRepoForBridges() *mockMatchRepo {
	return &mockMatchRepo{matches: make(map[string]*analysisdomain.Match)}
}

func (r *mockMatchRepo) Save(_ context.Context, match *analysisdomain.Match) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.matches[match.ID] = match
	return nil
}

func (r *mockMatchRepo) FindByID(_ context.Context, id string) (*analysisdomain.Match, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.matches[id]
	if !ok {
		return nil, analysisdomain.ErrMatchNotFound
	}
	return m, nil
}

func (r *mockMatchRepo) FindByUserID(_ context.Context, userID string) ([]*analysisdomain.Match, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*analysisdomain.Match
	for _, m := range r.matches {
		if m.UserID == userID {
			result = append(result, m)
		}
	}
	return result, nil
}

func (r *mockMatchRepo) UpdateStatus(_ context.Context, _ string, _ analysisdomain.MatchStatus) error {
	return nil
}

func (r *mockMatchRepo) UpdateAnalysisID(_ context.Context, _, _ string) error {
	return nil
}

func (r *mockMatchRepo) GetStuckMatches(_ context.Context, _ time.Time) ([]*analysisdomain.Match, error) {
	return nil, nil
}

func (r *mockMatchRepo) ClaimStuckMatch(_ context.Context, matchID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.matches[matchID]
	if !ok {
		return false, nil
	}
	if m.AnalysisFlag {
		return false, nil
	}
	m.AnalysisFlag = true
	return true, nil
}

func (r *mockMatchRepo) UnclaimMatch(_ context.Context, matchID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if m, ok := r.matches[matchID]; ok {
		m.AnalysisFlag = false
	}
	return nil
}

type mockAnalysisRepo struct {
	mu       sync.Mutex
	analyses map[string]*analysisdomain.AnalysisResult
}

func newMockAnalysisRepoForBridges() *mockAnalysisRepo {
	return &mockAnalysisRepo{analyses: make(map[string]*analysisdomain.AnalysisResult)}
}

func (r *mockAnalysisRepo) Save(_ context.Context, _ *analysisdomain.AnalysisResult) error {
	return nil
}

func (r *mockAnalysisRepo) FindByMatchID(_ context.Context, _ string) (*analysisdomain.AnalysisResult, error) {
	return nil, nil
}

func (r *mockAnalysisRepo) FindByID(_ context.Context, _ string) (*analysisdomain.AnalysisResult, error) {
	return nil, nil
}

func (r *mockAnalysisRepo) UpdateSummary(_ context.Context, _ string, _ analysisdomain.AnalysisSummary) error {
	return nil
}

func (r *mockAnalysisRepo) AddEvent(_ context.Context, _ string, _ analysisdomain.MatchEvent) error {
	return nil
}

type mockUserRepo struct {
	mu    sync.Mutex
	users map[string]*identitydomain.User
}

func newMockUserRepoForBridges() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*identitydomain.User)}
}

func (r *mockUserRepo) Create(_ context.Context, user *identitydomain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[user.Email] = user
	return nil
}

func (r *mockUserRepo) FindByEmail(_ context.Context, email string) (*identitydomain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[email]
	if !ok {
		return nil, identitydomain.ErrUserNotFound
	}
	return u, nil
}

func (r *mockUserRepo) FindByID(_ context.Context, id string) (*identitydomain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, identitydomain.ErrUserNotFound
}

func (r *mockUserRepo) Update(_ context.Context, _ *identitydomain.User) error {
	return nil
}

func TestCoachUserBridge_GetUserProfile(t *testing.T) {
	userRepo := newMockUserRepoForBridges()
	userRepo.Create(context.Background(), &identitydomain.User{
		ID: "u-1", Email: "test@example.com", FullName: "Test", BirthDate: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		Position: identitydomain.PositionCM, Footedness: identitydomain.FootednessRight, Goals: "improve",
	})

	bridge := NewCoachUserBridge(userRepo)
	profile, err := bridge.GetUserProfile(context.Background(), "u-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile.Age < 20 {
		t.Errorf("expected age >= 20, got %d", profile.Age)
	}
	if profile.Position != "CM" {
		t.Errorf("expected CM, got %s", profile.Position)
	}
	if profile.Footedness != "Right" {
		t.Errorf("expected Right, got %s", profile.Footedness)
	}
}

func TestCoachUserBridge_UserNotFound(t *testing.T) {
	userRepo := newMockUserRepoForBridges()
	bridge := NewCoachUserBridge(userRepo)

	_, err := bridge.GetUserProfile(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func TestCoachAnalysisBridge_GetLatestAnalysis_NoResults(t *testing.T) {
	matchRepo := newMockMatchRepoForBridges()
	analysisRepo := newMockAnalysisRepoForBridges()
	bridge := NewCoachAnalysisBridge(matchRepo, analysisRepo)

	insight, err := bridge.GetLatestAnalysis(context.Background(), "u-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if insight != nil {
		t.Error("expected nil insight when no results")
	}
}

func TestCoachAnalysisBridge_GetLatestAnalysis_WithResults(t *testing.T) {
	matchRepo := newMockMatchRepoForBridges()
	analysisRepo := newMockAnalysisRepoForBridges()

	savedMatch := &analysisdomain.Match{
		ID: "m-1", UserID: "u-1", ShirtNumber: 10, Status: analysisdomain.MatchStatusProcessing,
	}
	savedMatch.SetAnalysisResult(&analysisdomain.AnalysisResult{
		MatchID: "m-1",
		Summary: analysisdomain.AnalysisSummary{TotalDistanceKM: 10.5, TopSpeedKMH: 30.0, PassAccuracy: 0.82},
	})
	matchRepo.Save(context.Background(), savedMatch)

	bridge := NewCoachAnalysisBridge(matchRepo, analysisRepo)
	insight, err := bridge.GetLatestAnalysis(context.Background(), "u-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if insight == nil {
		t.Fatal("expected non-nil insight")
	}
	if insight.MatchID != "m-1" {
		t.Errorf("expected m-1, got %s", insight.MatchID)
	}
	if insight.DistanceKM != 10.5 {
		t.Errorf("expected 10.5, got %f", insight.DistanceKM)
	}
}

func TestCoachAnalysisBridge_GetMatchHistory(t *testing.T) {
	matchRepo := newMockMatchRepoForBridges()
	analysisRepo := newMockAnalysisRepoForBridges()

	for i := 0; i < 3; i++ {
		m := &analysisdomain.Match{
			ID: string(rune('a' + i)), UserID: "u-1", ShirtNumber: 10, Status: analysisdomain.MatchStatusProcessing,
		}
		m.SetAnalysisResult(&analysisdomain.AnalysisResult{
			MatchID: string(rune('a' + i)),
			Summary: analysisdomain.AnalysisSummary{TotalDistanceKM: 10.0 + float64(i)},
		})
		matchRepo.Save(context.Background(), m)
	}

	bridge := NewCoachAnalysisBridge(matchRepo, analysisRepo)
	history, err := bridge.GetMatchHistory(context.Background(), "u-1", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(history) > 2 {
		t.Errorf("expected at most 2 results, got %d", len(history))
	}
}
