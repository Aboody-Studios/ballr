package events

import "context"

type Publisher interface {
	PublishEvent(ctx context.Context, userID string, eventType string, metadata map[string]any) error
}

type noopPublisher struct{}

func (n *noopPublisher) PublishEvent(_ context.Context, _ string, _ string, _ map[string]any) error {
	return nil
}

func NoopPublisher() Publisher {
	return &noopPublisher{}
}
