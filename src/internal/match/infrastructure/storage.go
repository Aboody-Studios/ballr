package infrastructure

import (
	"context"
	"fmt"
	"io"
	"os"

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

func (r *StorageRepository) DownloadVideo(ctx context.Context, userID, matchID string) (string, error) {
	key := fmt.Sprintf("users/%s/videos/%s", userID, matchID)

	tmpFile, err := os.CreateTemp("", "ballr-video-*.mp4")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	output, err := r.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &r.bucket,
		Key:    &key,
	})
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to get object from S3: %w", err)
	}
	defer output.Body.Close()

	if _, err := io.Copy(tmpFile, output.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to copy S3 object to temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	return tmpFile.Name(), nil
}

func (r *StorageRepository) UploadFile(ctx context.Context, key, filePath, contentType string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	_, err = r.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &r.bucket,
		Key:         &key,
		Body:        file,
		ContentType: &contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %w", err)
	}

	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", r.bucket, key), nil
}
