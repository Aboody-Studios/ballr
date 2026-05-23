package http

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/identity/domain"
	"github.com/Aboody-Studios/ballr/src/internal/progress/application"
	progressdomain "github.com/Aboody-Studios/ballr/src/internal/progress/domain"
	"github.com/Aboody-Studios/ballr/src/pkg/validator"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

type mockProgRepo struct {
	mu       sync.Mutex
	progress map[string]*progressdomain.Progress
}

func newMockProgRepo() *mockProgRepo {
	return &mockProgRepo{progress: make(map[string]*progressdomain.Progress)}
}

func (r *mockProgRepo) Save(_ context.Context, p *progressdomain.Progress) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.progress[p.UserID] = p
	return nil
}

func (r *mockProgRepo) FindByUserID(_ context.Context, userID string) (*progressdomain.Progress, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.progress[userID]; ok {
		return p, nil
	}
	return nil, progressdomain.ErrProgressNotFound
}

func (r *mockProgRepo) AddPoints(_ context.Context, _ string, _ int64) error     { return nil }
func (r *mockProgRepo) UpdateStreak(_ context.Context, _ string, _ string) error { return nil }
func (r *mockProgRepo) GetLeaderboard(_ context.Context, _ int) ([]*progressdomain.LeaderboardEntry, error) {
	return nil, nil
}

type mockAchRepo struct {
	mu           sync.Mutex
	achievements map[string][]*progressdomain.Achievement
}

func newMockAchRepo() *mockAchRepo {
	return &mockAchRepo{achievements: make(map[string][]*progressdomain.Achievement)}
}

func (r *mockAchRepo) Save(_ context.Context, a *progressdomain.Achievement) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.achievements[a.UserID] = append(r.achievements[a.UserID], a)
	return nil
}

func (r *mockAchRepo) FindByUserID(_ context.Context, userID string) ([]*progressdomain.Achievement, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.achievements[userID], nil
}

func (r *mockAchRepo) FindByType(_ context.Context, _ string) ([]*progressdomain.Achievement, error) {
	return nil, nil
}

func (r *mockAchRepo) HasAchievement(_ context.Context, _, _ string) (bool, error) { return false, nil }

type mockEvtRepo struct{}

func (r *mockEvtRepo) Save(_ context.Context, _ *progressdomain.EventLog) error { return nil }
func (r *mockEvtRepo) FindRecentByUserID(_ context.Context, _ string, _ int) ([]*progressdomain.EventLog, error) {
	return nil, nil
}

type mockLeadRepo struct{}

func (r *mockLeadRepo) UpdateScore(_ context.Context, _ string, _ int64) error { return nil }
func (r *mockLeadRepo) GetTopPlayers(_ context.Context, _, _ int) ([]progressdomain.LeaderboardEntry, error) {
	return []progressdomain.LeaderboardEntry{
		{Rank: 1, UserID: "user-1", TotalPoints: 500},
	}, nil
}

func progressAuthCtx(method, path string) (*echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	e.Validator = validator.New()
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &domain.JWTCustomClaims{
		Email: "test@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        "user-1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	signed, _ := token.SignedString([]byte("test-secret"))
	parsed, _ := jwt.ParseWithClaims(signed, &domain.JWTCustomClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte("test-secret"), nil
	})
	c.Set("user", parsed)
	return c, rec
}

func TestProgressHandlers(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret")
	defer os.Unsetenv("JWT_SECRET")

	progRepo := newMockProgRepo()
	progRepo.Save(context.Background(), &progressdomain.Progress{
		ID: "p-1", UserID: "user-1", TotalPoints: 500, CurrentStreak: 7, LastActive: time.Now(),
	})
	achRepo := newMockAchRepo()
	achRepo.Save(context.Background(), &progressdomain.Achievement{
		UserID: "user-1", Type: "FIRST_UPLOAD", UnlockedAt: time.Now(),
	})

	svc := application.NewGamificationService(progRepo, achRepo, &mockEvtRepo{}, &mockLeadRepo{})
	handler := NewProgressHandler(svc)

	t.Run("GetProgressSummary", func(t *testing.T) {
		c, rec := progressAuthCtx("GET", "/secure/progress/summary")
		if err := handler.GetProgressSummaryHandler(c); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var resp map[string]interface{}
		json.NewDecoder(rec.Body).Decode(&resp)
		if resp["TotalPoints"] != float64(500) {
			t.Errorf("expected 500, got %v", resp["TotalPoints"])
		}
		if resp["CurrentStreak"] != float64(7) {
			t.Errorf("expected streak 7, got %v", resp["CurrentStreak"])
		}
	})

	t.Run("ListAchievements", func(t *testing.T) {
		c, rec := progressAuthCtx("GET", "/secure/achievements/list")
		if err := handler.ListAchievementsHandler(c); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var resp map[string]interface{}
		json.NewDecoder(rec.Body).Decode(&resp)
		ach, ok := resp["achievements"].([]interface{})
		if !ok {
			t.Fatal("expected achievements array")
		}
		if len(ach) != 1 {
			t.Errorf("expected 1 achievement, got %d", len(ach))
		}
	})

	t.Run("GetLeaderboard", func(t *testing.T) {
		c, rec := progressAuthCtx("GET", "/secure/leaderboard")
		if err := handler.GetLeaderboardHandler(c); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var resp map[string]interface{}
		json.NewDecoder(rec.Body).Decode(&resp)
		lb, ok := resp["leaderboard"].([]interface{})
		if !ok {
			t.Fatal("expected leaderboard array")
		}
		if len(lb) != 1 {
			t.Errorf("expected 1 entry, got %d", len(lb))
		}
	})
}
