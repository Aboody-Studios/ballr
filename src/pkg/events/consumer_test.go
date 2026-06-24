package events

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func setupTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis run: %v", err)
	}
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	t.Cleanup(func() { rdb.Close() })

	return mr, rdb
}

func TestRedisPublisher_PublishEvent(t *testing.T) {
	_, rdb := setupTestRedis(t)
	publisher := NewRedisPublisher(rdb)

	ctx := context.Background()
	err := publisher.PublishEvent(ctx, Event{Type: EventMatchUploaded, UserID: "user-1", Metadata: map[string]interface{}{"size": "10MB"}})
	if err != nil {
		t.Fatalf("PublishEvent failed: %v", err)
	}

	entries, err := rdb.XRange(ctx, DefaultStream, "-", "+").Result()
	if err != nil {
		t.Fatalf("XRange failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	raw, ok := entries[0].Values["data"].(string)
	if !ok {
		t.Fatal("expected data field as string")
	}

	var event Event
	if err := json.Unmarshal([]byte(raw), &event); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}

	if event.Type != EventMatchUploaded {
		t.Errorf("expected type %s, got %s", EventMatchUploaded, event.Type)
	}
	if event.UserID != "user-1" {
		t.Errorf("expected userID user-1, got %s", event.UserID)
	}
	if event.ID == "" {
		t.Error("expected non-empty event ID")
	}
	if event.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
	if event.Metadata["size"] != "10MB" {
		t.Errorf("expected metadata.size 10MB, got %v", event.Metadata["size"])
	}
}

func TestRedisPublisher_MultipleEvents(t *testing.T) {
	_, rdb := setupTestRedis(t)
	publisher := NewRedisPublisher(rdb)

	ctx := context.Background()
	events := []struct {
		userID    string
		eventType EventType
	}{
		{"user-1", EventMatchUploaded},
		{"user-2", EventAnalysisCompleted},
		{"user-1", EventCoachInteraction},
	}

	for _, e := range events {
		if err := publisher.PublishEvent(ctx, Event{Type: e.eventType, UserID: e.userID}); err != nil {
			t.Fatalf("PublishEvent failed: %v", err)
		}
	}

	entries, err := rdb.XRange(ctx, DefaultStream, "-", "+").Result()
	if err != nil {
		t.Fatalf("XRange failed: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
}

func TestConsumer_DispatchToHandler(t *testing.T) {
	_, rdb := setupTestRedis(t)
	publisher := NewRedisPublisher(rdb)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var received atomic.Value
	received.Store((*Event)(nil))

	consumer := NewConsumer(rdb, DefaultStream, "test-group", "test-consumer-1")
	consumer.HandleFunc(EventMatchUploaded, func(_ context.Context, event Event) error {
		received.Store(&event)
		return nil
	})

	go func() {
		consumer.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	err := publisher.PublishEvent(context.Background(), Event{Type: EventMatchUploaded, UserID: "user-1"})
	if err != nil {
		t.Fatalf("PublishEvent failed: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	ev := received.Load().(*Event)
	if ev == nil {
		t.Fatal("handler was not called")
	}
	if ev.Type != EventMatchUploaded {
		t.Errorf("expected type %s, got %s", EventMatchUploaded, ev.Type)
	}
	if ev.UserID != "user-1" {
		t.Errorf("expected userID user-1, got %s", ev.UserID)
	}
}

func TestConsumer_HandlerErrorGoesToDeadLetter(t *testing.T) {
	_, rdb := setupTestRedis(t)
	publisher := NewRedisPublisher(rdb)
	testErr := errors.New("handler failed")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consumer := NewConsumer(rdb, DefaultStream, "test-group-dl", "test-consumer-dl")
	consumer.HandleFunc(EventMatchUploaded, func(_ context.Context, event Event) error {
		return testErr
	})

	go func() {
		consumer.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	err := publisher.PublishEvent(context.Background(), Event{Type: EventMatchUploaded, UserID: "user-1"})
	if err != nil {
		t.Fatalf("PublishEvent failed: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	entries, err := rdb.XRange(ctx, DeadLetterStream, "-", "+").Result()
	if err != nil {
		t.Fatalf("XRange on dead letter failed: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected dead letter entry, got none")
	}
	if entries[0].Values["error"] != testErr.Error() {
		t.Errorf("expected error %q, got %q", testErr.Error(), entries[0].Values["error"])
	}
}

func TestConsumer_NoHandlerForTypeStillAcks(t *testing.T) {
	_, rdb := setupTestRedis(t)
	publisher := NewRedisPublisher(rdb)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consumer := NewConsumer(rdb, DefaultStream, "test-group-noop", "test-consumer-noop")
	consumer.HandleFunc(EventMatchUploaded, func(_ context.Context, event Event) error {
		return nil
	})

	go func() {
		consumer.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	err := publisher.PublishEvent(context.Background(), Event{Type: EventAnalysisCompleted, UserID: "user-1"})
	if err != nil {
		t.Fatalf("PublishEvent failed: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	pending, err := rdb.XPending(ctx, DefaultStream, "test-group-noop").Result()
	if err != nil {
		t.Fatalf("XPending failed: %v", err)
	}
	if pending.Count != 0 {
		t.Errorf("expected 0 pending, got %d -- message was not acked", pending.Count)
	}
}

func TestConsumer_GracefulShutdown(t *testing.T) {
	_, rdb := setupTestRedis(t)

	ctx, cancel := context.WithCancel(context.Background())

	consumer := NewConsumer(rdb, DefaultStream, "test-group-shutdown", "test-consumer-shutdown")

	go func() {
		cancel()
	}()

	err := consumer.Start(ctx)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestConsumer_EventTypeRouting(t *testing.T) {
	_, rdb := setupTestRedis(t)
	publisher := NewRedisPublisher(rdb)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var matchCount, analysisCount int32

	consumer := NewConsumer(rdb, DefaultStream, "test-group-route", "test-consumer-route")
	consumer.HandleFunc(EventMatchUploaded, func(_ context.Context, event Event) error {
		atomic.AddInt32(&matchCount, 1)
		return nil
	})
	consumer.HandleFunc(EventAnalysisCompleted, func(_ context.Context, event Event) error {
		atomic.AddInt32(&analysisCount, 1)
		return nil
	})

	go func() {
		consumer.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	publisher.PublishEvent(context.Background(), Event{Type: EventMatchUploaded, UserID: "user-1"})
	publisher.PublishEvent(context.Background(), Event{Type: EventAnalysisCompleted, UserID: "user-1"})
	publisher.PublishEvent(context.Background(), Event{Type: EventMatchUploaded, UserID: "user-2"})

	time.Sleep(200 * time.Millisecond)

	if n := atomic.LoadInt32(&matchCount); n != 2 {
		t.Errorf("expected 2 match events, got %d", n)
	}
	if n := atomic.LoadInt32(&analysisCount); n != 1 {
		t.Errorf("expected 1 analysis event, got %d", n)
	}
}
