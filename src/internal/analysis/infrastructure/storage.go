package infrastructure

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// StorageRepository implements the domain.StorageRepository interface using AWS S3 or maybe we use coolify s3 bucket.
// This is part of the infrastructure layer - it knows about AWS SDK details,
// while the domain only knows about the storage abstraction.
type StorageRepository struct {
	s3Client *s3.Client
	bucket   string
}

func NewStorageRepository(s3Client *s3.Client, bucket string) *StorageRepository {
	return &StorageRepository{
		s3Client: s3Client,
		bucket:   bucket,
	}
}

func (r *StorageRepository) GenerateUploadURL(ctx context.Context, userID, matchID string) (string, error) {
	filename := fmt.Sprintf("users/%s/videos/%s", userID, matchID)
	//TODO!: Add proper video size validation by using an S3 Presigned POST to have a set of strictly enforced rules 
	s3PutObj := &s3.PutObjectInput{
		Bucket:      &r.bucket,
		Key:         &filename,
		ContentType: aws.String("video/mp4"),
	}
	presignClient := s3.NewPresignClient(r.s3Client)

	request, err := presignClient.PresignPutObject(ctx, s3PutObj)
	if err != nil {
		return "", err
	}

	return request.URL, nil
}

func (r *StorageRepository) GetDownloadURL(ctx context.Context, videoID string) (string, error) {
	return "", nil
}

func (r *StorageRepository) DeleteVideo(ctx context.Context, videoID string) error {
	// TODO!: Implement S3 deletion when account available
	return nil
}
