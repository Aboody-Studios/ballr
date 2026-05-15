package application

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Aboody-Studios/ballr/src/internal/match/domain"
)

type mockMatchRepo struct {
	mu       sync.Mutex
	matches  map[string]*domain.Match
	saveErr  error
	findErr  error
}

func newMockMatchRepo() *mockMatchRepo {
	return &mockMatchRepo{matches: make(map[string]*domain.Match)}
}

func (r *mockMatchRepo) Save(_ context.Context, match *domain.Match) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.saveErr != nil {
		return r.saveErr
	}
	r.matches[match.ID] = match
	return nil
}

func (r *mockMatchRepo) FindByID(_ context.Context, id string) (*domain.Match, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.findErr != nil {
		return nil, r.findErr
	}
	m, ok := r.matches[id]
	if !ok {
		return nil, domain.ErrMatchNotFound
	}
	return m, nil
}

func (r *mockMatchRepo) FindByUserID(_ context.Context, userID string) ([]*domain.Match, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*domain.Match
	for _, m := range r.matches {
		if m.UserID == userID {
			result = append(result, m)
		}
	}
	return result, nil
}

func (r *mockMatchRepo) UpdateStatus(_ context.Context, matchID string, status domain.MatchStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.matches[matchID]
	if !ok {
		return domain.ErrMatchNotFound
	}
	m.Status = status
	return nil
}

func (r *mockMatchRepo) UpdateAnalysisID(_ context.Context, _ string, _ string) error {
	return nil
}

type mockAnalysisRepo struct {
	mu         sync.Mutex
	analyses   map[string]*domain.AnalysisResult
}

func newMockAnalysisRepo() *mockAnalysisRepo {
	return &mockAnalysisRepo{analyses: make(map[string]*domain.AnalysisResult)}
}

func (r *mockAnalysisRepo) Save(_ context.Context, analysis *domain.AnalysisResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.analyses[analysis.MatchID] = analysis
	return nil
}

func (r *mockAnalysisRepo) FindByMatchID(_ context.Context, matchID string) (*domain.AnalysisResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.analyses[matchID]
	if !ok {
		return nil, domain.ErrAnalysisNotFound
	}
	return a, nil
}

func (r *mockAnalysisRepo) FindByID(ctx context.Context, id string) (*domain.AnalysisResult, error) {
	return r.FindByMatchID(ctx, id)
}

func (r *mockAnalysisRepo) UpdateSummary(_ context.Context, _ string, _ domain.AnalysisSummary) error {
	return nil
}

func (r *mockAnalysisRepo) AddEvent(_ context.Context, _ string, _ domain.MatchEvent) error {
	return nil
}

type mockJobQueue struct {
	mu   sync.Mutex
	jobs []*domain.AnalysisJob
}

func newMockJobQueue() *mockJobQueue {
	return &mockJobQueue{}
}

func (q *mockJobQueue) Push(_ context.Context, job *domain.AnalysisJob) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.jobs = append(q.jobs, job)
	return nil
}

func (q *mockJobQueue) Pop(_ context.Context) (*domain.AnalysisJob, error) {
	return nil, nil
}

type mockEventPublisher struct {
	mu     sync.Mutex
	events []string
}

func (p *mockEventPublisher) PublishEvent(_ context.Context, _ string, eventType string, _ map[string]interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, eventType)
	return nil
}

type mockStorageProvider struct {
	url string
	err error
}

func (s *mockStorageProvider) GenerateUploadURL(_ context.Context, _, _ string) (string, error) {
	return s.url, s.err
}

func TestRequestUploadURL(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		storage := &mockStorageProvider{url: "https://s3.example.com/upload"}
		matchRepo := newMockMatchRepo()
		svc := NewUploadService(storage, matchRepo)
		pub := &mockEventPublisher{}
		svc.SetEventPublisher(pub)

		req := &MatchRequest{ShirtNumber: 10, Position: "CM", Size: 1000}
		url, err := svc.RequestUploadURL(context.Background(), req, "user-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if url != "https://s3.example.com/upload" {
			t.Errorf("expected upload URL, got %s", url)
		}

		if len(pub.events) != 1 || pub.events[0] != "MATCH_UPLOADED" {
			t.Errorf("expected MATCH_UPLOADED event, got %v", pub.events)
		}
	})

	t.Run("file too large", func(t *testing.T) {
		storage := &mockStorageProvider{url: "https://s3.example.com/upload"}
		matchRepo := newMockMatchRepo()
		svc := NewUploadService(storage, matchRepo)

		req := &MatchRequest{ShirtNumber: 10, Position: "CM", Size: 4000000000}
		_, err := svc.RequestUploadURL(context.Background(), req, "user-1")
		if !errors.Is(err, ErrFileTooLarge) {
			t.Errorf("expected ErrFileTooLarge, got %v", err)
		}
	})

	t.Run("storage error", func(t *testing.T) {
		storage := &mockStorageProvider{err: errors.New("s3 error")}
		matchRepo := newMockMatchRepo()
		svc := NewUploadService(storage, matchRepo)

		req := &MatchRequest{ShirtNumber: 10, Position: "CM", Size: 1000}
		_, err := svc.RequestUploadURL(context.Background(), req, "user-1")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestStartUploadURLService(t *testing.T) {
	storage := &mockStorageProvider{url: "https://s3.example.com/upload"}
	matchRepo := newMockMatchRepo()
	svc := NewUploadService(storage, matchRepo)

	req := &MatchRequest{ShirtNumber: 7, Position: "ST", Size: 500}
	url, err := svc.StartUploadURLService(context.Background(), req, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url == "" {
		t.Error("expected non-empty URL")
	}
}

func TestGetAnalysisStatus(t *testing.T) {
	t.Run("existing match", func(t *testing.T) {
		matchRepo := newMockMatchRepo()
		matchRepo.Save(context.Background(), &domain.Match{
			ID: "m-1", UserID: "u-1", ShirtNumber: 10, Status: domain.MatchStatusProcessing,
		})
		analysisRepo := newMockAnalysisRepo()
		svc := NewAnalysisService(analysisRepo, matchRepo, newMockJobQueue())

		status, err := svc.GetAnalysisStatus(context.Background(), "m-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if status != string(domain.MatchStatusProcessing) {
			t.Errorf("expected PROCESSING, got %s", status)
		}
	})

	t.Run("nonexistent match", func(t *testing.T) {
		matchRepo := newMockMatchRepo()
		analysisRepo := newMockAnalysisRepo()
		svc := NewAnalysisService(analysisRepo, matchRepo, newMockJobQueue())

		_, err := svc.GetAnalysisStatus(context.Background(), "nonexistent")
		if err == nil {
			t.Fatal("expected error for nonexistent match")
		}
	})
}

func TestGetAnalysisReport(t *testing.T) {
	t.Run("completed match returns analysis", func(t *testing.T) {
		matchRepo := newMockMatchRepo()
		m := &domain.Match{ID: "m-1", UserID: "u-1", ShirtNumber: 10, Status: domain.MatchStatusProcessing}
		if err := m.SetAnalysisResult(&domain.AnalysisResult{Summary: domain.AnalysisSummary{TotalDistanceKM: 10.5}}); err != nil {
			t.Fatalf("SetAnalysisResult failed: %v", err)
		}
		matchRepo.Save(context.Background(), m)

		analysisRepo := newMockAnalysisRepo()
		analysisRepo.Save(context.Background(), &domain.AnalysisResult{
			MatchID: "m-1", Summary: domain.AnalysisSummary{TotalDistanceKM: 10.5},
		})
		svc := NewAnalysisService(analysisRepo, matchRepo, newMockJobQueue())

		report, err := svc.GetAnalysisReport(context.Background(), "m-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if report.Summary.TotalDistanceKM != 10.5 {
			t.Errorf("expected 10.5 distance, got %f", report.Summary.TotalDistanceKM)
		}
	})

	t.Run("processing match returns error", func(t *testing.T) {
		matchRepo := newMockMatchRepo()
		matchRepo.Save(context.Background(), &domain.Match{
			ID: "m-1", UserID: "u-1", ShirtNumber: 10, Status: domain.MatchStatusProcessing,
		})
		analysisRepo := newMockAnalysisRepo()
		svc := NewAnalysisService(analysisRepo, matchRepo, newMockJobQueue())

		_, err := svc.GetAnalysisReport(context.Background(), "m-1")
		if !errors.Is(err, domain.ErrAnalysisNotFound) {
			t.Errorf("expected ErrAnalysisNotFound, got %v", err)
		}
	})
}

func TestStartAnalysis(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		matchRepo := newMockMatchRepo()
		matchRepo.Save(context.Background(), &domain.Match{
			ID: "m-1", UserID: "u-1", ShirtNumber: 10, PositionPlayed: "CM",
			Status: domain.MatchStatusUploading,
		})
		analysisRepo := newMockAnalysisRepo()
		jobQueue := newMockJobQueue()
		svc := NewAnalysisService(analysisRepo, matchRepo, jobQueue)

		err := svc.StartAnalysis(context.Background(), "m-1", "u-1", 10, "CM", "https://video.url")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		m, _ := matchRepo.FindByID(context.Background(), "m-1")
		if m.Status != domain.MatchStatusProcessing {
			t.Errorf("expected PROCESSING, got %s", m.Status)
		}

		if len(jobQueue.jobs) != 1 {
			t.Errorf("expected 1 job, got %d", len(jobQueue.jobs))
		}
	})

	t.Run("match not found", func(t *testing.T) {
		matchRepo := newMockMatchRepo()
		svc := NewAnalysisService(newMockAnalysisRepo(), matchRepo, newMockJobQueue())

		err := svc.StartAnalysis(context.Background(), "nonexistent", "u-1", 10, "CM", "https://video.url")
		if err == nil {
			t.Fatal("expected error for nonexistent match")
		}
	})

	t.Run("wrong user", func(t *testing.T) {
		matchRepo := newMockMatchRepo()
		matchRepo.Save(context.Background(), &domain.Match{
			ID: "m-1", UserID: "u-1", ShirtNumber: 10, Status: domain.MatchStatusUploading,
		})
		svc := NewAnalysisService(newMockAnalysisRepo(), matchRepo, newMockJobQueue())

		err := svc.StartAnalysis(context.Background(), "m-1", "u-2", 10, "CM", "https://video.url")
		if err == nil {
			t.Fatal("expected error for wrong user")
		}
	})

	t.Run("invalid transition from COMPLETED", func(t *testing.T) {
		matchRepo := newMockMatchRepo()
		matchRepo.Save(context.Background(), &domain.Match{
			ID: "m-1", UserID: "u-1", ShirtNumber: 10, Status: domain.MatchStatusCompleted,
		})
		svc := NewAnalysisService(newMockAnalysisRepo(), matchRepo, newMockJobQueue())

		err := svc.StartAnalysis(context.Background(), "m-1", "u-1", 10, "CM", "https://video.url")
		if err == nil {
			t.Fatal("expected error for invalid transition")
		}
	})
}
