package infrastructure

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/identity/domain"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestStore(t *testing.T) (*RedisRefreshTokenStore, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := NewRedisRefreshTokenStore(client, time.Hour)
	return store, mr
}

func randomToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func TestStoreAndGet(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	token := randomToken()

	data := &domain.RefreshTokenData{
		UserID:   "user-1",
		Email:    "test@example.com",
		FamilyID: "family-1",
	}

	err := store.Store(ctx, token, data, time.Hour)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	got, err := store.Get(ctx, token)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got.UserID != "user-1" {
		t.Errorf("expected UserID user-1, got %s", got.UserID)
	}
	if got.Email != "test@example.com" {
		t.Errorf("expected Email test@example.com, got %s", got.Email)
	}
	if got.FamilyID != "family-1" {
		t.Errorf("expected FamilyID family-1, got %s", got.FamilyID)
	}
}

func TestGet_NonExistent(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	_, err := store.Get(ctx, "nonexistent-token")
	if err == nil {
		t.Fatal("expected error for non-existent token")
	}
}

func TestDelete(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	token := randomToken()

	data := &domain.RefreshTokenData{UserID: "user-1", Email: "test@example.com"}
	store.Store(ctx, token, data, time.Hour)

	err := store.Delete(ctx, token)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = store.Get(ctx, token)
	if err == nil {
		t.Error("expected token to be deleted")
	}
}

func TestRevokeAllForUser(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	tokens := make([]string, 3)
	for i := 0; i < 3; i++ {
		tok := randomToken()
		tokens[i] = tok
		store.Store(ctx, tok, &domain.RefreshTokenData{UserID: "user-1", Email: "user@example.com"}, time.Hour)
	}

	err := store.RevokeAllForUser(ctx, "user-1")
	if err != nil {
		t.Fatalf("RevokeAllForUser failed: %v", err)
	}

	for _, tok := range tokens {
		_, err := store.Get(ctx, tok)
		if err == nil {
			t.Errorf("expected token %s to be revoked", tok[:10])
		}
	}
}

func TestRevokeAllForUser_NoTokens(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	err := store.RevokeAllForUser(ctx, "user-with-no-tokens")
	if err != nil {
		t.Fatalf("RevokeAllForUser on empty user failed: %v", err)
	}
}

func TestTTLExpiry(t *testing.T) {
	store, mr := newTestStore(t)
	ctx := context.Background()
	token := randomToken()

	store.Store(ctx, token, &domain.RefreshTokenData{UserID: "user-1", Email: "test@example.com"}, 50*time.Millisecond)

	_, err := store.Get(ctx, token)
	if err != nil {
		t.Fatalf("expected token to exist before TTL expiry: %v", err)
	}

	mr.FastForward(100 * time.Millisecond)

	_, err = store.Get(ctx, token)
	if err == nil {
		t.Error("expected token to expire after TTL")
	}
}

func TestGet_MissingReturnsError(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	_, err := store.Get(ctx, "does-not-exist")
	if err == nil {
		t.Error("expected error for missing token")
	}
}

func TestHashConsistency(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	token := "same-token-value"
	data := &domain.RefreshTokenData{UserID: "user-1", Email: "test@example.com"}
	store.Store(ctx, token, data, time.Hour)

	_, err := store.Get(ctx, token)
	if err != nil {
		t.Errorf("token lookup with same raw value should work: %v", err)
	}
}
