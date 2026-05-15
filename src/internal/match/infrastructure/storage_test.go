package infrastructure

import (
	"testing"
)

func TestNewStorageRepository(t *testing.T) {
	r := NewStorageRepository(nil, "test-bucket")
	if r == nil {
		t.Fatal("expected non-nil repo")
	}
	if r.bucket != "test-bucket" {
		t.Errorf("expected bucket test-bucket, got %s", r.bucket)
	}
}
