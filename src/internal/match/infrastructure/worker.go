package infrastructure

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/match/domain"
)

type Worker struct {
	matchRepo    domain.MatchRepository
	analysisRepo domain.AnalysisRepository
	jobQueue     domain.JobQueue
}

func NewWorker(matchRepo domain.MatchRepository, analysisRepo domain.AnalysisRepository, jobQueue domain.JobQueue) *Worker {
	return &Worker{
		matchRepo:    matchRepo,
		analysisRepo: analysisRepo,
		jobQueue:     jobQueue,
	}
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
			w.processJob(job)
		}
	}
}

func (w *Worker) processJob(job *domain.AnalysisJob) {
	sleepDuration := time.Duration(2+rand.Intn(3)) * time.Second
	time.Sleep(sleepDuration)

	result := mockAnalysis(job.MatchID, job.UserID)

	if err := w.analysisRepo.Save(context.Background(), result); err != nil {
		return
	}

	if err := w.matchRepo.UpdateStatus(context.Background(), job.MatchID, domain.MatchStatusCompleted); err != nil {
		return
	}
}

func mockAnalysis(matchID, userID string) *domain.AnalysisResult {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &domain.AnalysisResult{
		MatchID:     matchID,
		GeneratedAt: time.Now(),
		Summary: domain.AnalysisSummary{
			TotalDistanceKM: 8.5 + rng.Float64()*4,
			TopSpeedKMH:     28 + rng.Float64()*8,
			PassAccuracy:    0.65 + rng.Float64()*0.3,
			Touches:         40 + rng.Intn(60),
			Sprints:         8 + rng.Intn(15),
		},
		Heatmaps: domain.Heatmaps{
			OverallURL:   fmt.Sprintf("https://storage.example.com/heatmaps/%s/overall.png", matchID),
			DefensiveURL: fmt.Sprintf("https://storage.example.com/heatmaps/%s/defensive.png", matchID),
			AttackingURL: fmt.Sprintf("https://storage.example.com/heatmaps/%s/attacking.png", matchID),
		},
		Events:          generateMockEvents(rng),
		TrackingDataURL: fmt.Sprintf("https://storage.example.com/tracking/%s/data.json", matchID),
	}
}

func generateMockEvents(rng *rand.Rand) []domain.MatchEvent {
	eventTypes := []domain.EventType{
		domain.EventTypePass, domain.EventTypeShot, domain.EventTypeDribble,
		domain.EventTypeTackle, domain.EventTypeSprint, domain.EventTypeRecovery,
	}
	results := []domain.EventResult{
		domain.EventResultSuccess, domain.EventResultFailure, domain.EventResultNeutral,
	}
	insights := []string{
		"Accurate through ball to striker",
		"Powerful shot on target",
		"Close control dribble past defender",
		"Clean tackle to regain possession",
		"Explosive sprint to chase down attacker",
		"Quick recovery to defensive position",
	}

	n := 3 + rng.Intn(8)
	events := make([]domain.MatchEvent, n)
	for i := range events {
		events[i] = domain.MatchEvent{
			Timestamp:   fmt.Sprintf("%02d:%02d:%02d", rng.Intn(90), rng.Intn(60), rng.Intn(60)),
			Type:        eventTypes[rng.Intn(len(eventTypes))],
			Result:      results[rng.Intn(len(results))],
			Coordinates: domain.Position2D{X: rng.Float64() * 100, Y: rng.Float64() * 100},
			Insight:     insights[rng.Intn(len(insights))],
		}
	}
	return events
}
