package domain

import "context"

type AnalysisJob struct {
	MatchID     string `json:"match_id"`
	UserID      string `json:"user_id"`
	VideoURL    string `json:"video_url"`
	ShirtNumber int    `json:"shirt_number"`
	Position    string `json:"position"`
}

type JobQueue interface {
	Push(ctx context.Context, job *AnalysisJob) error
	Pop(ctx context.Context) (*AnalysisJob, error)
}
