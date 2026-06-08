package domain

import (
	"context"

	"github.com/Aboody-Studios/ballr/src/internal/identity/domain"
)

type AnalysisJob struct {
	MatchID     string          `json:"match_id"`
	UserID      string          `json:"user_id"`
	VideoURL    string          `json:"video_url"`
	ShirtNumber uint            `json:"shirt_number"`
	Position    domain.Position `json:"position"`
}

type JobQueue interface {
	Push(ctx context.Context, job *AnalysisJob) error
	Pop(ctx context.Context) (*AnalysisJob, error)
}
