package events

import (
	"context"
)

type Publisher interface {
	PublishEvent(ctx context.Context, event Event) error
}

type noopPublisher struct{}

func (n *noopPublisher) PublishEvent(_ context.Context, _ Event) error {
	return nil
}

func NoopPublisher() Publisher {
	return &noopPublisher{}
}
