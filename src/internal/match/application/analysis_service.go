package application

import (
	"context"
	"fmt"

	"github.com/Aboody-Studios/ballr/src/internal/match/domain"
	"github.com/Aboody-Studios/ballr/src/pkg/events"
)

type AnalysisService struct {
	matchRepo      domain.MatchRepository
	analysisRepo   domain.AnalysisRepository
	jobQueue       domain.JobQueue
	eventPublisher events.Publisher
}

func NewAnalysisService(analysisRepo domain.AnalysisRepository, matchRepo domain.MatchRepository, queue domain.JobQueue) *AnalysisService {
	return &AnalysisService{
		analysisRepo:   analysisRepo,
		matchRepo:      matchRepo,
		jobQueue:       queue,
		eventPublisher: events.NoopPublisher(),
	}
}

func (s *AnalysisService) SetEventPublisher(p events.Publisher) {
	s.eventPublisher = p
}

// GetAnalysisStatus retrieves the current processing status of a match analysis.
func (s *AnalysisService) GetAnalysisStatus(ctx context.Context, matchID string) (string, error) {
	match, err := s.matchRepo.FindByID(ctx, matchID)
	if err != nil {
		return "", err
	}
	return string(match.Status), nil
}

// GetAnalysisReport retrieves the complete analysis results for a match.
func (s *AnalysisService) GetAnalysisReport(ctx context.Context, matchID string) (*domain.AnalysisResult, error) {
	analysisResult, err := s.analysisRepo.FindByID(ctx, matchID)
	if err != nil {
		return nil, err
	}

	if !analysisResult.CanViewResults(matchID) {
		return nil, domain.ErrAnalysisNotFound
	}

	analysis, err := s.analysisRepo.FindByMatchID(ctx, matchID)
	if err != nil {
		return nil, err
	}

	return analysis, nil
}

// StartAnalysis initiates the CV analysis pipeline after video upload.
func (s *AnalysisService) StartAnalysis(ctx context.Context, matchID, videoURL string) error {
	match, err := s.matchRepo.FindByID(ctx, matchID)
	if err != nil {
		return fmt.Errorf("match not found: %w", err)
	}

	job := &domain.AnalysisJob{
		MatchID:     matchID,
		UserID:      match.UserID,
		VideoURL:    videoURL,
		ShirtNumber: match.ShirtNumber,
		Position:    match.PositionPlayed,
	}
	if err := s.jobQueue.Push(ctx, job); err != nil {
		return err
	}

	return nil
}
