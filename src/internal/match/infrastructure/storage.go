package infrastructure

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// StorageRepository implements the domain.StorageRepository interface using AWS S3.
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
	key := fmt.Sprintf("users/videos/%s", videoID)
	presignClient := s3.NewPresignClient(r.s3Client)

	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: &r.bucket,
		Key:    &key,
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate download URL: %w", err)
	}

	return request.URL, nil
}

func (r *StorageRepository) DeleteVideo(ctx context.Context, videoID string) error {
	key := fmt.Sprintf("users/videos/%s", videoID)
	_, err := r.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &r.bucket,
		Key:    &key,
	})
	if err != nil {
		return fmt.Errorf("failed to delete video: %w", err)
	}

	return nil
}
