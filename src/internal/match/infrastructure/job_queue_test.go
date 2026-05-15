package infrastructure

import (
	"context"
	"testing"

	"github.com/Aboody-Studios/ballr/src/internal/match/domain"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func setupTestQueue(t *testing.T) (*RedisJobQueue, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	queue := NewRedisJobQueue(client)
	return queue, mr
}

func TestRedisJobQueue_PushPop(t *testing.T) {
	queue, mr := setupTestQueue(t)
	ctx := context.Background()

	job := &domain.AnalysisJob{
		MatchID: "m-1", UserID: "u-1", VideoURL: "https://video.url",
		ShirtNumber: 10, Position: "CM",
	}

	err := queue.Push(ctx, job)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	if !mr.Exists("analysis:queue") {
		t.Error("expected queue key to exist")
	}
}

func TestRedisJobQueue_PopBlocking(t *testing.T) {
	queue, mr := setupTestQueue(t)
	ctx := context.Background()

	job := &domain.AnalysisJob{
		MatchID: "m-1", UserID: "u-1", VideoURL: "https://video.url",
		ShirtNumber: 10, Position: "CM",
	}

	queue.Push(ctx, job)

	popped, err := queue.Pop(ctx)
	if err != nil {
		t.Fatalf("Pop failed: %v", err)
	}
	if popped.MatchID != "m-1" {
		t.Errorf("expected m-1, got %s", popped.MatchID)
	}
	if popped.ShirtNumber != 10 {
		t.Errorf("expected 10, got %d", popped.ShirtNumber)
	}
	if popped.Position != "CM" {
		t.Errorf("expected CM, got %s", popped.Position)
	}

	if mr.Exists("analysis:queue") {
		l, err := mr.List("analysis:queue")
		if err == nil && len(l) != 0 {
			t.Errorf("expected empty queue, got %d items", len(l))
		}
	}
}

func TestRedisJobQueue_FIFO(t *testing.T) {
	queue, _ := setupTestQueue(t)
	ctx := context.Background()

	queue.Push(ctx, &domain.AnalysisJob{MatchID: "first", UserID: "u-1"})
	queue.Push(ctx, &domain.AnalysisJob{MatchID: "second", UserID: "u-1"})

	first, _ := queue.Pop(ctx)
	if first.MatchID != "first" {
		t.Errorf("expected 'first', got '%s'", first.MatchID)
	}

	second, _ := queue.Pop(ctx)
	if second.MatchID != "second" {
		t.Errorf("expected 'second', got '%s'", second.MatchID)
	}
}

func TestRedisJobQueueRoundTrip(t *testing.T) {
	queue, _ := setupTestQueue(t)
	ctx := context.Background()

	original := &domain.AnalysisJob{
		MatchID: "m-1", UserID: "u-1", VideoURL: "https://example.com/video.mp4",
		ShirtNumber: 7, Position: "ST",
	}
	err := queue.Push(ctx, original)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	got, err := queue.Pop(ctx)
	if err != nil {
		t.Fatalf("Pop failed: %v", err)
	}
	if got.MatchID != original.MatchID {
		t.Errorf("MatchID: got %s, want %s", got.MatchID, original.MatchID)
	}
	if got.UserID != original.UserID {
		t.Errorf("UserID: got %s, want %s", got.UserID, original.UserID)
	}
	if got.VideoURL != original.VideoURL {
		t.Errorf("VideoURL: got %s, want %s", got.VideoURL, original.VideoURL)
	}
	if got.ShirtNumber != original.ShirtNumber {
		t.Errorf("ShirtNumber: got %d, want %d", got.ShirtNumber, original.ShirtNumber)
	}
	if got.Position != original.Position {
		t.Errorf("Position: got %s, want %s", got.Position, original.Position)
	}
}
