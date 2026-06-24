package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/match/domain"
	"github.com/Aboody-Studios/ballr/src/pkg/events"
	"github.com/google/uuid"
)

// execCommandContext is overridable for testing.
var execCommandContext = exec.CommandContext

const (
	cvPythonPath   = "python3"
	cvScriptPath   = "src/cv/main.py"
	jobTimeout     = 30 * time.Minute
	maxRetries     = 3
	retryBaseDelay = 5 * time.Second
)

type Worker struct {
	matchRepo      domain.MatchRepository
	analysisRepo   domain.AnalysisRepository
	jobQueue       domain.JobQueue
	eventPublisher events.Publisher
	storageRepo    domain.S3StorageProvider
}

func NewWorker(matchRepo domain.MatchRepository, analysisRepo domain.AnalysisRepository, jobQueue domain.JobQueue, storageRepo domain.S3StorageProvider) *Worker {
	return &Worker{
		matchRepo:      matchRepo,
		analysisRepo:   analysisRepo,
		jobQueue:       jobQueue,
		eventPublisher: events.NoopPublisher(),
		storageRepo:    storageRepo,
	}
}

func (w *Worker) SetEventPublisher(p events.Publisher) {
	w.eventPublisher = p
}

func (w *Worker) Start(ctx context.Context) {
	go w.run(ctx)
}

func (w *Worker) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			job, err := w.jobQueue.Pop(ctx)
			if err != nil {
				if err == context.Canceled || err == context.DeadlineExceeded {
					return
				}
				continue
			}
			w.processJob(ctx, job)
		}
	}
}

func (w *Worker) processJob(ctx context.Context, job *domain.AnalysisJob) {
	log := slog.With("match_id", job.MatchID, "user_id", job.UserID)

	var lastErr error
	for attempt := range maxRetries {
		if attempt > 0 {
			delay := retryBaseDelay * time.Duration(1<<uint(attempt-1))
			log.Info("retrying analysis", "attempt", attempt+1, "delay", delay)
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}
		}

		err := w.runAnalysis(ctx, job, log)
		if err == nil {
			return
		}

		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			log.Error("analysis cancelled/timeout", "error", err)
			w.markFailed(ctx, job.MatchID, log)
			return
		}

		lastErr = err
		log.Error("analysis attempt failed", "attempt", attempt+1, "error", err)
	}

	log.Error("all analysis attempts failed", "last_error", lastErr)
	w.markFailed(ctx, job.MatchID, log)
}

func (w *Worker) runAnalysis(ctx context.Context, job *domain.AnalysisJob, log *slog.Logger) error {
	log.Info("downloading video from S3")
	videoPath, err := w.storageRepo.DownloadVideo(ctx, job.UserID, job.MatchID)
	if err != nil {
		return fmt.Errorf("download video: %w", err)
	}
	defer os.Remove(videoPath)

	jobCtx, cancel := context.WithTimeout(ctx, jobTimeout)
	defer cancel()

	scriptPath := os.Getenv("CV_SCRIPT_PATH")
	if scriptPath == "" {
		scriptPath = "src/cv/main.py"
	}

	log.Info("starting CV analysis")
	//TODO!: Change fmt to something from strconv as it is computationally heavy
	cmd := execCommandContext(jobCtx, "python3", scriptPath,
		"--video", videoPath,
		"--match-id", job.MatchID,
		"--user-id", job.UserID,
		"--shirt-number", fmt.Sprintf("%s", job.ShirtNumber),
		"--position", string(job.Position),
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if jobCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("analysis timed out after %v: %w", jobTimeout, context.DeadlineExceeded)
		}
		if jobCtx.Err() == context.Canceled {
			return fmt.Errorf("analysis cancelled: %w", context.Canceled)
		}
		return fmt.Errorf("cv pipeline failed: %w\nstderr: %s", err, stderr.String())
	}

	if stderr.Len() > 0 {
		log.Warn("cv pipeline stderr", "stderr", stderr.String())
	}

	var result cvResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return fmt.Errorf("parse cv output: %w\noutput: %s", err, stdout.String())
	}

	analysis := result.toDomain(job.MatchID)

	log.Info("saving analysis result")
	if err := w.analysisRepo.Save(ctx, analysis); err != nil {
		return fmt.Errorf("save analysis: %w", err)
	}

	log.Info("updating match status to completed")
	if err := w.matchRepo.UpdateStatus(ctx, job.MatchID, domain.MatchStatusCompleted); err != nil {
		return fmt.Errorf("update match status: %w", err)
	}

	if err := w.eventPublisher.PublishEvent(ctx, events.Event{Type: events.EventAnalysisCompleted, UserID: job.UserID}); err != nil {
		log.Warn("failed to publish completion event", "error", err)
	}

	log.Info("analysis completed successfully")
	return nil
}

func (w *Worker) markFailed(ctx context.Context, matchID string, log *slog.Logger) {
	if err := w.matchRepo.UpdateStatus(ctx, matchID, domain.MatchStatusFailed); err != nil {
		log.Error("failed to update match status to FAILED", "error", err)
	}
}

// cvResult maps the JSON output from the Python CV pipeline.
type cvResult struct {
	Summary     cvSummary  `json:"summary"`
	Heatmaps    cvHeatmaps `json:"heatmaps"`
	Events      []cvEvent  `json:"events"`
	TrackingURL string     `json:"tracking_data_url"`
}

type cvSummary struct {
	TotalDistance float64 `json:"total_distance"`
	TopSpeed      float64 `json:"top_speed"`
	PassAccuracy  float64 `json:"pass_accuracy"`
	Touches       int     `json:"touches"`
	Sprints       int     `json:"sprints"`
}

type cvHeatmaps struct {
	OverallURL   string `json:"overall_url"`
	DefensiveURL string `json:"defensive_url"`
	AttackingURL string `json:"attacking_url"`
}

type cvEvent struct {
	Timestamp   string     `json:"timestamp"`
	Type        string     `json:"type"`
	Result      string     `json:"result"`
	Coordinates cvPosition `json:"coordinates"`
	Insight     string     `json:"insight"`
}

type cvPosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func (r *cvResult) toDomain(matchID string) *domain.AnalysisResult {
	events := make([]domain.MatchEvent, len(r.Events))
	for i, e := range r.Events {
		events[i] = domain.MatchEvent{
			Timestamp:   e.Timestamp,
			Type:        domain.EventType(e.Type),
			Result:      domain.EventResult(e.Result),
			Coordinates: domain.Position2D{X: e.Coordinates.X, Y: e.Coordinates.Y},
			Insight:     e.Insight,
		}
	}

	return &domain.AnalysisResult{
		ID:          uuid.New().String(),
		MatchID:     matchID,
		GeneratedAt: time.Now(),
		Summary: domain.AnalysisSummary{
			TotalDistanceKM: r.Summary.TotalDistance,
			TopSpeedKMH:     r.Summary.TopSpeed,
			PassAccuracy:    r.Summary.PassAccuracy,
			Touches:         r.Summary.Touches,
			Sprints:         r.Summary.Sprints,
		},
		Heatmaps: domain.Heatmaps{
			OverallURL:   r.Heatmaps.OverallURL,
			DefensiveURL: r.Heatmaps.DefensiveURL,
			AttackingURL: r.Heatmaps.AttackingURL,
		},
		Events:          events,
		TrackingDataURL: r.TrackingURL,
	}
}
