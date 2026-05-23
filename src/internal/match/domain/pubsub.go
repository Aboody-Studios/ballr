package domain

import "context"

type VideoUpload struct {
	S3key    string
	VideoURL string
}

type EventPublisher interface {
	Publish(ctx context.Context, vu VideoUpload) error
}
