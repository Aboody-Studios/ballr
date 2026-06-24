package handlers

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	iddomain "github.com/Aboody-Studios/ballr/src/internal/identity/domain"
	"github.com/Aboody-Studios/ballr/src/internal/match/application"
	"github.com/Aboody-Studios/ballr/src/internal/match/domain"
	"github.com/Aboody-Studios/ballr/src/pkg/events"
	"github.com/Aboody-Studios/ballr/src/pkg/validator"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

type mockStorage struct {
	url string
	err error
}

func (s *mockStorage) GeneratePresignedPostObj(_ context.Context, _, _ string) (*domain.PresignedUpload, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &domain.PresignedUpload{URL: s.url, Fields: map[string]string{}}, nil
}

func (s *mockStorage) GetDownloadURL(_ context.Context, _ string) (string, error) {
	return "https://mock-download-url", nil
}
func (s *mockStorage) DeleteVideo(_ context.Context, _ string) error                { return nil }
func (s *mockStorage) DownloadVideo(_ context.Context, _, _ string) (string, error) { return "", nil }
func (s *mockStorage) UploadFile(_ context.Context, key, _, _ string) (string, error) {
	return "https://mock-bucket/" + key, nil
}

type mockMatchRepo struct {
	mu      sync.Mutex
	matches map[string]*domain.Match
}

func newMockMatchRepo() *mockMatchRepo {
	return &mockMatchRepo{matches: make(map[string]*domain.Match)}
}

func (r *mockMatchRepo) Save(_ context.Context, match *domain.Match) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if match.ID == "" {
		match.ID = "m-1"
	}
	r.matches[match.ID] = match
	return nil
}

func (r *mockMatchRepo) FindByID(_ context.Context, id string) (*domain.Match, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.matches[id]
	if !ok {
		return nil, domain.ErrMatchNotFound
	}
	return m, nil
}

func (r *mockMatchRepo) FindByUserID(_ context.Context, _ string) ([]*domain.Match, error) {
	return nil, nil
}

func (r *mockMatchRepo) UpdateStatus(_ context.Context, id string, status domain.MatchStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if m, ok := r.matches[id]; ok {
		m.Status = status
	}
	return nil
}

func (r *mockMatchRepo) UpdateAnalysisID(_ context.Context, _, _ string) error {
	return nil
}

func (r *mockMatchRepo) ClaimStuckMatch(_ context.Context, matchID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if m, ok := r.matches[matchID]; ok {
		if m.AnalysisFlag {
			return false, nil
		}
		m.AnalysisFlag = true
		return true, nil
	}
	return false, nil
}

func (r *mockMatchRepo) UnclaimMatch(_ context.Context, matchID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if m, ok := r.matches[matchID]; ok {
		m.AnalysisFlag = false
	}
	return nil
}

func (r *mockMatchRepo) GetStuckMatches(_ context.Context, _ time.Time) ([]*domain.Match, error) {
	return nil, nil
}

type mockAnaRepo struct{}

func (r *mockAnaRepo) Save(_ context.Context, _ *domain.AnalysisResult) error { return nil }
func (r *mockAnaRepo) FindByMatchID(_ context.Context, _ string) (*domain.AnalysisResult, error) {
	return nil, nil
}
func (r *mockAnaRepo) FindByID(_ context.Context, _ string) (*domain.AnalysisResult, error) {
	return nil, nil
}
func (r *mockAnaRepo) UpdateSummary(_ context.Context, _ string, _ domain.AnalysisSummary) error {
	return nil
}
func (r *mockAnaRepo) AddEvent(_ context.Context, _ string, _ domain.MatchEvent) error { return nil }

type mockMatchQueue struct{}

func (q *mockMatchQueue) Push(_ context.Context, _ *domain.AnalysisJob) error { return nil }
func (q *mockMatchQueue) Pop(_ context.Context) (*domain.AnalysisJob, error)  { return nil, nil }

type mockEventPub struct{}

func (p *mockEventPub) PublishEvent(_ context.Context, _ events.Event) error {
	return nil
}

func matchAuthCtx(method, path, body string) (*echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	e.Validator = validator.New()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &iddomain.JWTCustomClaims{
		Email: "test@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        "user-1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	signed, _ := token.SignedString([]byte("test-secret"))
	parsed, _ := jwt.ParseWithClaims(signed, &iddomain.JWTCustomClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte("test-secret"), nil
	})
	c.Set("user", parsed)
	return c, rec
}

func TestUploadURLHandler(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret")
	defer os.Unsetenv("JWT_SECRET")

	storage := &mockStorage{url: "https://s3.example.com/upload"}
	matchRepo := newMockMatchRepo()
	uploadSvc := application.NewUploadService(storage, matchRepo)
	uploadSvc.SetEventPublisher(&mockEventPub{})

	handler := NewUploadHandler(uploadSvc)
	c, rec := matchAuthCtx("POST", "/secure/analysis/upload-url",
		`{"shirt_number":10,"position":"CM","size":1000,"metadata":{}}`)

	if err := handler.UploadURLHandler(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["URL"] != "https://s3.example.com/upload" {
		t.Errorf("expected upload URL, got %s", resp["URL"])
	}
}

func TestStartAnalysisHandler(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret")
	defer os.Unsetenv("JWT_SECRET")

	matchRepo := newMockMatchRepo()
	matchRepo.Save(context.Background(), &domain.Match{
		ID: "m-1", UserID: "user-1", ShirtNumber: 10, Status: domain.MatchStatusUploading,
	})
	analysisSvc := application.NewAnalysisService(&mockAnaRepo{}, matchRepo, &mockMatchQueue{})
	analysisSvc.SetEventPublisher(&mockEventPub{})

	handler := NewAnalysisHandler(analysisSvc)
	c, rec := matchAuthCtx("POST", "/secure/analysis/start",
		`{"match_id":"m-1","shirt_number":10,"position":"CM","video_url":"https://video.url"}`)

	if err := handler.StartAnalysisHandler(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["status"] != "PROCESSING" {
		t.Errorf("expected PROCESSING, got %s", resp["status"])
	}
}
