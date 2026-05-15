package application

import (
	"context"
	"fmt"

	"github.com/Aboody-Studios/ballr/src/internal/match/domain"
	"github.com/Aboody-Studios/ballr/src/pkg/events"
)

type UploadService struct {
	storageProvider StorageProvider
	matchRepo       domain.MatchRepository
	eventPublisher  events.Publisher
}

type AnalysisService struct {
	matchRepo       domain.MatchRepository
	analysisRepo    domain.AnalysisRepository
	jobQueue        domain.JobQueue
	eventPublisher  events.Publisher
}

func NewUploadService(provider StorageProvider, repo domain.MatchRepository) *UploadService {
	return &UploadService{
		storageProvider: provider,
		matchRepo:       repo,
		eventPublisher:  events.NoopPublisher(),
	}
}

func NewAnalysisService(analysisRepo domain.AnalysisRepository, matchRepo domain.MatchRepository, queue domain.JobQueue) *AnalysisService {
	return &AnalysisService{
		analysisRepo:   analysisRepo,
		matchRepo:      matchRepo,
		jobQueue:       queue,
		eventPublisher: events.NoopPublisher(),
	}
}

func (s *UploadService) SetEventPublisher(p events.Publisher) {
	s.eventPublisher = p
}

func (s *AnalysisService) SetEventPublisher(p events.Publisher) {
	s.eventPublisher = p
}

// Business Rules Validated:
//   - File extension must be .mp4 (case-sensitive)
//   - File size must not exceed 3,375,000,000 bytes (~3.14 GB : PI GB)
//
// a nice Easter egg for  future reference yk
func (s *UploadService) RequestUploadURL(ctx context.Context, matchRequest *MatchRequest, userID string) (string, error) {
	if matchRequest.Size > 3375000000 {
		return "", fmt.Errorf("%w: size %d exceeds maximum %d", ErrFileTooLarge, matchRequest.Size, 3375000000)
	}

	match := &domain.Match{
		UserID:         userID,
		ShirtNumber:    matchRequest.ShirtNumber,
		PositionPlayed: matchRequest.Position,
		Status:         domain.MatchStatusUploading,
		Metadata:       domain.MatchMetadata(matchRequest.Metadata),
	}
	// Saving to database first before generating upload url is essential because
	// if it was the other way around and the persistence fails,
	// we would have a match in s3 without a record in the database.
	if err := s.matchRepo.Save(ctx, match); err != nil {
		return "", err
	}

	//match.ID is accessible here because a pointer to the match struct is passed to the database.
	// Which means any changes to the struct in the repo layer will have its effects here also.
	uploadURL, err := s.storageProvider.GenerateUploadURL(ctx, userID, match.ID)
	if err != nil {
		return "", fmt.Errorf("failed to generate upload url: %w", err)
	}

	if err := s.eventPublisher.PublishEvent(ctx, userID, "MATCH_UPLOADED", nil); err != nil {
		return "", fmt.Errorf("failed to publish event: %w", err)
	}

	return uploadURL, nil
}

// GenerateUploadURL handles the upload URL generation use case.
func (s *UploadService) StartUploadURLService(ctx context.Context, matchRequest *MatchRequest, userID string) (string, error) {
	return s.RequestUploadURL(ctx, matchRequest, userID)
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
	match, err := s.matchRepo.FindByID(ctx, matchID)
	if err != nil {
		return nil, err
	}

	if !match.CanViewResults() {
		return nil, domain.ErrAnalysisNotFound
	}

	analysis, err := s.analysisRepo.FindByMatchID(ctx, matchID)
	if err != nil {
		return nil, err
	}

	return analysis, nil
}

// StartAnalysis initiates the CV analysis pipeline after video upload.
func (s *AnalysisService) StartAnalysis(ctx context.Context, matchID string, userID string, shirtNumber int, position, videoURL string) error {
	match, err := s.matchRepo.FindByID(ctx, matchID)
	if err != nil {
		return fmt.Errorf("match not found: %w", err)
	}

	if match.UserID != userID {
		return fmt.Errorf("match does not belong to user")
	}

	if err := match.MarkUploadComplete(videoURL); err != nil {
		return err
	}

	if err := s.matchRepo.Save(ctx, match); err != nil {
		return err
	}

	job := &domain.AnalysisJob{
		MatchID:     matchID,
		UserID:      userID,
		VideoURL:    videoURL,
		ShirtNumber: shirtNumber,
		Position:    position,
	}
	if err := s.jobQueue.Push(ctx, job); err != nil {
		return err
	}

	return nil
}

// Application layer errors for upload use cases.
var (
	// ErrInvalidFileFormat indicates the video file extension is not supported.
	ErrInvalidFileFormat = &UploadError{"invalid file format"}

	// ErrFileTooLarge indicates the video file exceeds size limits.
	ErrFileTooLarge = &UploadError{"file size exceeds maximum allowed"}
)

// UploadError represents domain-specific errors for upload operations.
type UploadError struct {
	Message string
}

func (e *UploadError) Error() string {
	return e.Message
}
