package domain

import (
	"context"

)

type PresignedUpload struct {
	URL    string            `json:"url"`
	Fields map[string]string `json:"fields"`
}

// StorageProvider defines the interface for cloud storage operations.
// Implemented in infrastructure layer (AWS S3 but we may need a bu bucket in coolify hostinger to test).
type S3StorageProvider interface {
	// GenerateUploadURL creates a pre-signed URL for direct client upload.
	// Returns the URL and any error from the storage service.
	GeneratePresignedPostObj(ctx context.Context, userID, matchID string) (*PresignedUpload, error)

	GetDownloadURL(ctx context.Context, videoID string) (string, error)

	DeleteVideo(ctx context.Context, videoID string) error

	DownloadVideo(ctx context.Context, userID, matchID string) (string, error)

	UploadFile(ctx context.Context, key, filePath, contentType string) (string, error)
}
