package infrastructure

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/match/domain"
	"gorm.io/gorm"
)

type mockMatchRepo struct {
	mu      sync.Mutex
	matches map[string]*domain.Match
}

func (r *mockMatchRepo) Save(_ context.Context, match *domain.Match) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if match.ID == "" {
		match.ID = "generated-id"
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

func (r *mockMatchRepo) UpdateAnalysisID(_ context.Context, _, _ string) error {
	return nil
}

type mockAnalysisRepo struct {
	mu       sync.Mutex
	analyses map[string]*domain.AnalysisResult
}

func (r *mockAnalysisRepo) Save(_ context.Context, analysis *domain.AnalysisResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.analyses[analysis.MatchID] = analysis
	return nil
}

func (r *mockAnalysisRepo) FindByMatchID(_ context.Context, _ string) (*domain.AnalysisResult, error) {
	return nil, nil
}

func (r *mockAnalysisRepo) FindByID(_ context.Context, _ string) (*domain.AnalysisResult, error) {
	return nil, nil
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

func newMockJobQueueWithJobs(jobs []*domain.AnalysisJob) *mockJobQueue {
	return &mockJobQueue{jobs: jobs}
}

func (q *mockJobQueue) Push(_ context.Context, _ *domain.AnalysisJob) error {
	return nil
}

func (q *mockJobQueue) Pop(ctx context.Context) (*domain.AnalysisJob, error) {
	q.mu.Lock()
	if len(q.jobs) == 0 {
		q.mu.Unlock()
		<-ctx.Done()
		return nil, ctx.Err()
	}
	job := q.jobs[0]
	q.jobs = q.jobs[1:]
	q.mu.Unlock()
	return job, nil
}

func TestMockAnalysis(t *testing.T) {
	result := mockAnalysis("m-1", "u-1")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.MatchID != "m-1" {
		t.Errorf("expected match m-1, got %s", result.MatchID)
	}
	if result.Summary.TotalDistanceKM < 8.5 || result.Summary.TotalDistanceKM > 12.5 {
		t.Errorf("unexpected distance: %f", result.Summary.TotalDistanceKM)
	}
	if result.Summary.TopSpeedKMH < 28 || result.Summary.TopSpeedKMH > 36 {
		t.Errorf("unexpected top speed: %f", result.Summary.TopSpeedKMH)
	}
	if len(result.Events) < 3 || len(result.Events) > 10 {
		t.Errorf("expected 3-10 events, got %d", len(result.Events))
	}
}

func TestGenerateMockEvents(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	events := generateMockEvents(rng)
	if len(events) < 3 {
		t.Errorf("expected at least 3 events, got %d", len(events))
	}
	for i, e := range events {
		if e.Timestamp == "" {
			t.Errorf("event %d has empty timestamp", i)
		}
		if e.Type == "" {
			t.Errorf("event %d has empty type", i)
		}
		if e.Result == "" {
			t.Errorf("event %d has empty result", i)
		}
	}
}

func TestWorkerProcessJob(t *testing.T) {
	matchRepo := &mockMatchRepo{matches: make(map[string]*domain.Match)}
	matchRepo.Save(context.Background(), &domain.Match{
		ID: "m-1", UserID: "u-1", ShirtNumber: 10, Status: domain.MatchStatusUploading,
	})
	analysisRepo := &mockAnalysisRepo{analyses: make(map[string]*domain.AnalysisResult)}
	jobQueue := newMockJobQueueWithJobs([]*domain.AnalysisJob{
		{MatchID: "m-1", UserID: "u-1", VideoURL: "https://video.url", ShirtNumber: 10, Position: "CM"},
	})

	worker := NewWorker(matchRepo, analysisRepo, jobQueue)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	worker.Start(ctx)

	time.Sleep(6 * time.Second)

	match, _ := matchRepo.FindByID(context.Background(), "m-1")
	if match != nil && match.Status != domain.MatchStatusCompleted {
		t.Errorf("expected COMPLETED status, got %s", match.Status)
	}
}

func TestWorkerContextCancellation(t *testing.T) {
	matchRepo := &mockMatchRepo{matches: make(map[string]*domain.Match)}
	analysisRepo := &mockAnalysisRepo{analyses: make(map[string]*domain.AnalysisResult)}
	jobQueue := newMockJobQueueWithJobs(nil)

	worker := NewWorker(matchRepo, analysisRepo, jobQueue)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	worker.Start(ctx)
	go func() {
		time.Sleep(100 * time.Millisecond)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop after context cancellation")
	}
}

func TestWorkerNoJobs(t *testing.T) {
	matchRepo := &mockMatchRepo{matches: make(map[string]*domain.Match)}
	analysisRepo := &mockAnalysisRepo{analyses: make(map[string]*domain.AnalysisResult)}
	jobQueue := newMockJobQueueWithJobs(nil)

	worker := NewWorker(matchRepo, analysisRepo, jobQueue)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	worker.Start(ctx)
	<-ctx.Done()
}

func TestErrMatchNotFound(t *testing.T) {
	if domain.ErrMatchNotFound.Error() != "match not found" {
		t.Errorf("unexpected error message")
	}
}

func TestIsGormError(t *testing.T) {
	err := gorm.ErrRecordNotFound
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Error("expected gorm.ErrRecordNotFound")
	}
}
