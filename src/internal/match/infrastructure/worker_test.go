package infrastructure

import (
	"context"
	"errors"
	"os"
	"os/exec"
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

type mockStorageRepo struct {
	mu sync.Mutex
}

func (r *mockStorageRepo) DownloadVideo(_ context.Context, _, _ string) (string, error) {
	f, err := os.CreateTemp("", "ballr-test-video-*.mp4")
	if err != nil {
		return "", err
	}
	f.Close()
	return f.Name(), nil
}

func (r *mockStorageRepo) UploadFile(_ context.Context, key, _, _ string) (string, error) {
	return "https://mock-bucket.s3.amazonaws.com/" + key, nil
}

func (r *mockStorageRepo) GenerateUploadURL(_ context.Context, _, _ string) (string, error) {
	return "https://mock-upload-url", nil
}

func (r *mockStorageRepo) GetDownloadURL(_ context.Context, _ string) (string, error) {
	return "https://mock-download-url", nil
}

func (r *mockStorageRepo) DeleteVideo(_ context.Context, _ string) error {
	return nil
}

func TestWorkerProcessJob(t *testing.T) {
	matchRepo := &mockMatchRepo{matches: make(map[string]*domain.Match)}
	if err := matchRepo.Save(context.Background(), &domain.Match{
		ID: "m-1", UserID: "u-1", ShirtNumber: 10, Status: domain.MatchStatusUploading,
	}); err != nil {
		t.Fatal(err)
	}
	analysisRepo := &mockAnalysisRepo{analyses: make(map[string]*domain.AnalysisResult)}
	jobQueue := newMockJobQueueWithJobs([]*domain.AnalysisJob{
		{MatchID: "m-1", UserID: "u-1", VideoURL: "https://video.url", ShirtNumber: 10, Position: "CM"},
	})
	storageRepo := &mockStorageRepo{}

	oldExec := execCommandContext
	execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "echo", `{"summary":{"total_distance":10.5,"top_speed":30.0,"pass_accuracy":0.75,"touches":50,"sprints":10},"heatmaps":{"overall_url":"https://s3.com/hm.png","defensive_url":"","attacking_url":""},"events":[{"timestamp":"15:20","type":"PASS","result":"SUCCESS","coordinates":{"x":45.0,"y":30.0},"insight":"Good pass"}],"tracking_data_url":""}`)
	}
	t.Cleanup(func() { execCommandContext = oldExec })

	worker := NewWorker(matchRepo, analysisRepo, jobQueue, storageRepo)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	worker.Start(ctx)

	time.Sleep(1 * time.Second)

	match, err := matchRepo.FindByID(context.Background(), "m-1")
	if err != nil {
		t.Fatal(err)
	}
	if match.Status != domain.MatchStatusCompleted {
		t.Errorf("expected COMPLETED, got %s", match.Status)
	}

	analysisRepo.mu.Lock()
	analysis, ok := analysisRepo.analyses["m-1"]
	analysisRepo.mu.Unlock()
	if !ok {
		t.Fatal("expected analysis to be saved")
	}
	if analysis.Summary.TotalDistanceKM != 10.5 {
		t.Errorf("expected total_distance 10.5, got %f", analysis.Summary.TotalDistanceKM)
	}
	if analysis.Summary.TopSpeedKMH != 30.0 {
		t.Errorf("expected top_speed 30.0, got %f", analysis.Summary.TopSpeedKMH)
	}
	if analysis.Summary.PassAccuracy != 0.75 {
		t.Errorf("expected pass_accuracy 0.75, got %f", analysis.Summary.PassAccuracy)
	}
}

func TestWorkerHandlesPythonCrash(t *testing.T) {
	matchRepo := &mockMatchRepo{matches: make(map[string]*domain.Match)}
	if err := matchRepo.Save(context.Background(), &domain.Match{
		ID: "m-1", UserID: "u-1", ShirtNumber: 10, Status: domain.MatchStatusUploading,
	}); err != nil {
		t.Fatal(err)
	}
	analysisRepo := &mockAnalysisRepo{analyses: make(map[string]*domain.AnalysisResult)}
	jobQueue := newMockJobQueueWithJobs([]*domain.AnalysisJob{
		{MatchID: "m-1", UserID: "u-1", VideoURL: "https://video.url", ShirtNumber: 10, Position: "CM"},
	})
	storageRepo := &mockStorageRepo{}

	oldExec := execCommandContext
	execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sh", "-c", `echo "cv error" >&2; exit 1`)
	}
	t.Cleanup(func() { execCommandContext = oldExec })

	worker := NewWorker(matchRepo, analysisRepo, jobQueue, storageRepo)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	worker.Start(ctx)

	time.Sleep(18 * time.Second)

	match, err := matchRepo.FindByID(context.Background(), "m-1")
	if err != nil {
		t.Fatal(err)
	}
	if match.Status != domain.MatchStatusFailed {
		t.Errorf("expected FAILED after retries exhausted, got %s", match.Status)
	}
}

func TestWorkerHandlesTimeout(t *testing.T) {
	matchRepo := &mockMatchRepo{matches: make(map[string]*domain.Match)}
	if err := matchRepo.Save(context.Background(), &domain.Match{
		ID: "m-1", UserID: "u-1", ShirtNumber: 10, Status: domain.MatchStatusUploading,
	}); err != nil {
		t.Fatal(err)
	}
	analysisRepo := &mockAnalysisRepo{analyses: make(map[string]*domain.AnalysisResult)}
	jobQueue := newMockJobQueueWithJobs([]*domain.AnalysisJob{
		{MatchID: "m-1", UserID: "u-1", VideoURL: "https://video.url", ShirtNumber: 10, Position: "CM"},
	})
	storageRepo := &mockStorageRepo{}

	oldExec := execCommandContext
	execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sleep", "60")
	}
	t.Cleanup(func() { execCommandContext = oldExec })

	worker := NewWorker(matchRepo, analysisRepo, jobQueue, storageRepo)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	worker.Start(ctx)

	<-ctx.Done()
	time.Sleep(200 * time.Millisecond)

	match, err := matchRepo.FindByID(context.Background(), "m-1")
	if err != nil {
		t.Fatal(err)
	}
	if match.Status != domain.MatchStatusFailed {
		t.Errorf("expected FAILED after timeout, got %s", match.Status)
	}
}

func TestWorkerContextCancellation(t *testing.T) {
	matchRepo := &mockMatchRepo{matches: make(map[string]*domain.Match)}
	analysisRepo := &mockAnalysisRepo{analyses: make(map[string]*domain.AnalysisResult)}
	jobQueue := newMockJobQueueWithJobs(nil)
	storageRepo := &mockStorageRepo{}

	worker := NewWorker(matchRepo, analysisRepo, jobQueue, storageRepo)
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
	storageRepo := &mockStorageRepo{}

	worker := NewWorker(matchRepo, analysisRepo, jobQueue, storageRepo)
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
