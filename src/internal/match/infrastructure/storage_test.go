package infrastructure

import (
	"context"
	"testing"
)

func TestNewStorageRepository(t *testing.T) {
	r := NewStorageRepository(nil, "test-bucket")
	if r == nil {
		t.Fatal("expected non-nil repo")
	}
}

func TestStorageRepository_GetDownloadURL(t *testing.T) {
	r := NewStorageRepository(nil, "test-bucket")
	url, err := r.GetDownloadURL(context.Background(), "video-1")
	if url != "" {
		t.Errorf("expected empty URL, got %s", url)
	}
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestStorageRepository_DeleteVideo(t *testing.T) {
	r := NewStorageRepository(nil, "test-bucket")
	err := r.DeleteVideo(context.Background(), "video-1")
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}
