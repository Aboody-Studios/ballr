package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/Aboody-Studios/ballr/src/internal/analysis/domain"
)

// UploadService handles video upload validation and pre-signed URL generation.
type UploadService struct {
	storageProvider StorageProvider
}

// StorageProvider defines the interface for cloud storage operations.
// Implemented in infrastructure layer (AWS S3 but we may need a bu bucket in coolify hostinger to test).
type StorageProvider interface {
	// GenerateUploadURL creates a pre-signed URL for direct client upload.
	// Returns the URL and any error from the storage service.
	GenerateUploadURL(ctx context.Context, video *domain.Video) (string, error)
}

// NewUploadService creates a new upload service with the given storage provider.
func NewUploadService(provider StorageProvider) *UploadService {
	return &UploadService{
		storageProvider: provider,
	}
}

// RequestUploadURL handles the complete upload URL generation use case.
//
// Business Rules Validated:
//   - File extension must be .mp4 (case-sensitive)
//   - File size must not exceed 3,375,000,000 bytes (~3.14 GB : PI GB)
//
// a nice Easter egg for  future reference yk
func (s *UploadService) RequestUploadURL(ctx context.Context, name string, size uint64) (string, error) {
	if !strings.HasSuffix(name, ".mp4") {
		return "", fmt.Errorf("%w: file must be .mp4 format", ErrInvalidFileFormat)
	}

	const maxSize uint64 = 3375000000
	if size > maxSize {
		return "", fmt.Errorf("%w: size %d exceeds maximum %d", ErrFileTooLarge, size, maxSize)
	}

	video := domain.NewVideo(name, size)

	uploadURL, err := s.storageProvider.GenerateUploadURL(ctx, video)
	if err != nil {
		return "", fmt.Errorf("failed to generate upload url: %w", err)
	}

	return uploadURL, nil
}

// Service is the main application service for the Analysis bounded context.
type Service struct {
	uploadService   *UploadService
	matchRepo       domain.MatchRepository
	analysisRepo    domain.AnalysisRepository
	storageProvider StorageProvider
}

// NewService creates a new analysis application service with all dependencies.
func NewService(
	uploadService *UploadService,
	matchRepo domain.MatchRepository,
	analysisRepo domain.AnalysisRepository,
	storageProvider StorageProvider,
) *Service {
	return &Service{
		uploadService:   uploadService,
		matchRepo:       matchRepo,
		analysisRepo:    analysisRepo,
		storageProvider: storageProvider,
	}
}

// GenerateUploadURL handles the upload URL generation use case.
func (s *Service) GenerateUploadURL(ctx context.Context, video *domain.Video) (string, error) {
	return s.uploadService.RequestUploadURL(ctx, video.Name, video.Size)
}

// GetAnalysisStatus retrieves the current processing status of a match analysis.
func (s *Service) GetAnalysisStatus(ctx context.Context, matchID string) (string, error) {
	match, err := s.matchRepo.FindByID(ctx, matchID)
	if err != nil {
		return "", err
	}
	return string(match.Status), nil
}

// GetAnalysisReport retrieves the complete analysis results for a match.
func (s *Service) GetAnalysisReport(ctx context.Context, matchID string) (*domain.AnalysisResult, error) {
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
func (s *Service) StartAnalysis(ctx context.Context, matchID string, shirtNumber int, position, videoURL string) error {
	metadata := domain.MatchMetadata{}

	match, err := domain.NewMatch(matchID, "", shirtNumber, position, metadata)
	if err != nil {
		return err
	}

	if err := match.MarkUploadComplete(videoURL); err != nil {
		return err
	}

	if err := s.matchRepo.Save(ctx, match); err != nil {
		return err
	}

	// TODO!: Push job to Redis/SQS queue for analysis worker

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
