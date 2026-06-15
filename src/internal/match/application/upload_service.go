package application

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/Aboody-Studios/ballr/src/internal/match/domain"
	"github.com/Aboody-Studios/ballr/src/pkg/events"
	"github.com/google/uuid"
)

type UploadService struct {
	storageProvider domain.S3StorageProvider
	matchRepo       domain.MatchRepository
	eventPublisher  events.Publisher
}

func NewUploadService(provider domain.S3StorageProvider, repo domain.MatchRepository) *UploadService {
	return &UploadService{
		storageProvider: provider,
		matchRepo:       repo,
		eventPublisher:  events.NoopPublisher(),
	}
}

func (s *UploadService) SetEventPublisher(p events.Publisher) {
	s.eventPublisher = p
}

func (s *UploadService) StartUploadURLService(ctx context.Context, matchRequest *MatchRequest, userID string) (*domain.PresignedUpload, error) {
	return s.RequestUploadURL(ctx, matchRequest, userID)
}

// Business Rules Validated:
//   - File extension must be .mp4 (case-sensitive)
//   - File size must not exceed 3,375,000,000 bytes (~3.14 GB : PI GB)
//
// a nice Easter egg for  future reference yk
func (s *UploadService) RequestUploadURL(ctx context.Context, matchRequest *MatchRequest, userID string) (*domain.PresignedUpload, error) {
	if matchRequest.Size > 3375000000 {
		return nil, fmt.Errorf("%w: size %d exceeds maximum %d", ErrFileTooLarge, matchRequest.Size, 3375000000)
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
		return nil, err
	}

	//match.ID is accessible here because a pointer to the match struct is passed to the database.
	// Which means any changes to the struct in the repo layer will have its effects here also.
	PresignedUploadStruct, err := s.storageProvider.GeneratePresignedPostObj(ctx, userID, match.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate upload url: %w", err)
	}

	return PresignedUploadStruct, nil
}

// ConfirmUploadByS3Key updates match status to PROCESSING after S3 confirms upload.
// The s3Key format is: users/{userID}/videos/{matchID}
func (s *UploadService) StartMatchProcessingWorflow(ctx context.Context, s3Key string) error {
	matchID, err := parseMatchIDFromS3Key(s3Key)
	if err != nil {
		return fmt.Errorf("parse s3 key: %w", err)
	}
	match, err := s.matchRepo.FindByID(ctx, matchID)
	if err != nil {
		return fmt.Errorf("match not found: %w", err)
	}

	if err := match.UpdateMatchStatusToProcessing(s3Key); err != nil {
		return fmt.Errorf("status update error: %w", err)
	}

	if err := s.matchRepo.Save(ctx, match); err != nil {
		return fmt.Errorf("save match: %w", err)
	}

	matchUploaded := events.Event{
		ID:     uuid.NewString(),
		Type:   events.EventMatchUploaded,
		UserID: match.UserID,
	}

	if err := s.eventPublisher.PublishEvent(ctx, matchUploaded); err != nil {
		log.Printf("failed to publish event: %w", err)
	}
	metadata := map[string]any{
		"video_url": match.VideoURL,
		"match_id":  matchID,
	}

	startAnalysis := events.Event{
		ID:       uuid.NewString(),
		Type:     events.EventAnalysisStart,
		UserID:   match.UserID,
		Metadata: metadata,
	}

	if err := s.eventPublisher.PublishEvent(ctx, startAnalysis); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	match.MarkAnalysisInit()
	if err := s.matchRepo.Save(ctx, match); err != nil {
		return fmt.Errorf("save match: %w", err)
	}

	return nil
}

func parseMatchIDFromS3Key(key string) (string, error) {
	parts := strings.Split(key, "/")
	if len(parts) < 4 || parts[0] != "users" || parts[2] != "videos" {
		return "", fmt.Errorf("unexpected s3 key format: %s", key)
	}
	return parts[3], nil
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
