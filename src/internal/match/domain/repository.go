package domain

import (
	"context"
	"time"
)

//TODO!: Consolidate match & analysis repos

// MatchRepository defines the contract for match persistence operations.
// Implemented in infrastructure layer to maintain dependency inversion.
type MatchRepository interface {
	Save(ctx context.Context, match *Match) error

	FindByID(ctx context.Context, id string) (*Match, error)

	FindByUserID(ctx context.Context, userID string) ([]*Match, error)

	// UpdateStatus modifies the processing status of a match.
	UpdateStatus(ctx context.Context, matchID string, status MatchStatus) error

	// GetStuckMatches Gets any matches whose status is PROCESSING and AnalysisFlag is false
	GetStuckMatches(ctx context.Context, cutOffTime time.Time) ([]*Match, error)

	// ClaimStuckMatch atomically claims a stuck match by setting AnalysisFlag=true.
	// Returns true if this call successfully claimed the match, false if it was already claimed.
	ClaimStuckMatch(ctx context.Context, matchID string) (bool, error)

	// UnclaimMatch releases a previously claimed match by setting AnalysisFlag=false.
	// Used to revert claims if downstream publish/enqueue fails.
	UnclaimMatch(ctx context.Context, matchID string) error
}

// AnalysisRepository defines the contract for analysis result persistence.
// Analysis results are document-style data (JSONB in PostgreSQL or MongoDB).
type AnalysisRepository interface {
	// Save persists analysis results for a match.
	Save(ctx context.Context, analysis *AnalysisResult) error

	FindByMatchID(ctx context.Context, matchID string) (*AnalysisResult, error)

	FindByID(ctx context.Context, id string) (*AnalysisResult, error)

	// UpdateSummary updates the summary statistics (distance, speed, accuracy).
	UpdateSummary(ctx context.Context, matchID string, summary AnalysisSummary) error

	// UpdateAnalysisID links a match to its analysis results.
	UpdateAnalysisID(ctx context.Context, matchID string, analysisID string) error

	// AddEvent appends a new event to the analysis events list.
	AddEvent(ctx context.Context, matchID string, event MatchEvent) error
}

// ErrMatchNotFound is returned when a match lookup fails.
var ErrMatchNotFound = errMatchNotFound{}

type errMatchNotFound struct{}

func (errMatchNotFound) Error() string { return "match not found" }

// ErrAnalysisNotFound is returned when analysis results are not found.
var ErrAnalysisNotFound = errAnalysisNotFound{}

type errAnalysisNotFound struct{}

func (errAnalysisNotFound) Error() string { return "analysis not found" }
