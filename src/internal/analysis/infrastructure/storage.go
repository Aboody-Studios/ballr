package infrastructure

import (
	"context"

	"github.com/Aboody-Studios/ballr/src/internal/analysis/domain"
)

// StorageRepository implements the domain.StorageRepository interface using AWS S3 or maybe we use coolify s3 bucket.
// This is part of the infrastructure layer - it knows about AWS SDK details,
// while the domain only knows about the storage abstraction.
type StorageRepository struct {
	// TODO!: Add AWS S3 client when account is available
	// s3Client *s3.Client
	// bucket   string
}

func NewStorageRepository() *StorageRepository {
	return &StorageRepository{}
}

func (r *StorageRepository) GenerateUploadURL(ctx context.Context, video *domain.Video) (string, error) {
	// TODO!: Implement S3 pre-signed URL generation when account available
	return "lol", nil
}

func (r *StorageRepository) GetDownloadURL(ctx context.Context, videoID string) (string, error) {
	return "", nil
}

func (r *StorageRepository) DeleteVideo(ctx context.Context, videoID string) error {
	// TODO!: Implement S3 deletion when account available
	return nil
}
