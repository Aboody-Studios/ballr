package application

import (
	"context"

	"github.com/Aboody-Studios/ballr/src/internal/match/infrastructure"
)

// StorageProvider defines the interface for cloud storage operations.
// Implemented in infrastructure layer (AWS S3 but we may need a bu bucket in coolify hostinger to test).
type StorageProvider interface {
	// GenerateUploadURL creates a pre-signed URL for direct client upload.
	// Returns the URL and any error from the storage service.
	GenerateUploadURL(ctx context.Context, userID, matchID string) (*infrastructure.PresignedUpload, error)
}
