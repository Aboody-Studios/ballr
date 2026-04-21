package domain

import "context"

// MatchRepository defines the contract for match persistence operations.
// Implemented in infrastructure layer to maintain dependency inversion.
type MatchRepository interface {
	// Save persists a new match record.
	Save(ctx context.Context, match *Match) error

	// FindByID retrieves a match by its UUID.
	FindByID(ctx context.Context, id string) (*Match, error)

	// FindByUserID retrieves all matches for a specific user.
	FindByUserID(ctx context.Context, userID string) ([]*Match, error)

	// UpdateStatus modifies the processing status of a match.
	UpdateStatus(ctx context.Context, matchID string, status MatchStatus) error

	// UpdateAnalysisID links a match to its analysis results.
	UpdateAnalysisID(ctx context.Context, matchID string, analysisID string) error
}

// AnalysisRepository defines the contract for analysis result persistence.
// Analysis results are document-style data (JSONB in PostgreSQL or MongoDB).
type AnalysisRepository interface {
	// Save persists analysis results for a match.
	Save(ctx context.Context, analysis *AnalysisResult) error

	// FindByMatchID retrieves analysis results by the associated match ID.
	FindByMatchID(ctx context.Context, matchID string) (*AnalysisResult, error)

	// FindByID retrieves analysis results by its own UUID.
	FindByID(ctx context.Context, id string) (*AnalysisResult, error)

	// UpdateSummary updates the summary statistics (distance, speed, accuracy).
	UpdateSummary(ctx context.Context, matchID string, summary AnalysisSummary) error

	// AddEvent appends a new event to the analysis events list.
	AddEvent(ctx context.Context, matchID string, event MatchEvent) error
}

// UploadURLRepository manages pre-signed URL generation and validation.
// This abstracts the cloud storage provider (S3, GCS, etc.).
type UploadURLRepository interface {
	// GeneratePresignedURL creates a temporary upload URL for a video.
	// The URL is valid for a limited time and restricted to specific content types.
	GeneratePresignedURL(ctx context.Context, videoName string, contentType string, sizeLimit uint64) (string, error)

	// ValidateUpload confirms that an upload completed successfully at the given path.
	ValidateUpload(ctx context.Context, storagePath string) (bool, error)
}

// ErrMatchNotFound is returned when a match lookup fails.
var ErrMatchNotFound = errMatchNotFound{}

type errMatchNotFound struct{}

func (errMatchNotFound) Error() string { return "match not found" }

// ErrAnalysisNotFound is returned when analysis results are not found.
var ErrAnalysisNotFound = errAnalysisNotFound{}

type errAnalysisNotFound struct{}

func (errAnalysisNotFound) Error() string { return "analysis not found" }
