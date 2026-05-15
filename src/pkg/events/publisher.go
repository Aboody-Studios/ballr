package events

import "context"

type Publisher interface {
	PublishEvent(ctx context.Context, userID string, eventType string, metadata map[string]interface{}) error
}

type noopPublisher struct{}

func (n *noopPublisher) PublishEvent(_ context.Context, _ string, _ string, _ map[string]interface{}) error {
	return nil
}

func NoopPublisher() Publisher {
	return &noopPublisher{}
}
