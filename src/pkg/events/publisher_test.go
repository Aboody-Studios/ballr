package events

import (
	"context"
	"testing"
	"time"
)

func TestNoopPublisher(t *testing.T) {
	p := NoopPublisher()
	err := p.PublishEvent(context.Background(), Event{Type: EventType("TEST_EVENT"), UserID: "user-1", Metadata: nil, Timestamp: time.Now()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNoopPublisherWithMetadata(t *testing.T) {
	p := NoopPublisher()
	metadata := map[string]interface{}{"key": "value"}
	err := p.PublishEvent(context.Background(), Event{Type: EventType("TEST_EVENT"), UserID: "user-1", Metadata: metadata, Timestamp: time.Now()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
