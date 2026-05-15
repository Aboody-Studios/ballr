package events

import (
	"context"
	"testing"
)

func TestNoopPublisher(t *testing.T) {
	p := NoopPublisher()
	err := p.PublishEvent(context.Background(), "user-1", "TEST_EVENT", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNoopPublisherWithMetadata(t *testing.T) {
	p := NoopPublisher()
	metadata := map[string]interface{}{"key": "value"}
	err := p.PublishEvent(context.Background(), "user-1", "TEST_EVENT", metadata)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
