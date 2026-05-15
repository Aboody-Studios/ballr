package infrastructure

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisStore_Allow_FirstRequest(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := &RedisStore{Client: client}

	allowed, err := store.Allow("test-user")
	if err != nil {
		t.Fatalf("Allow failed: %v", err)
	}
	if !allowed {
		t.Error("expected first request to be allowed")
	}
}

func TestRedisStore_Allow_UnderLimit(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := &RedisStore{Client: client}

	for i := 0; i < 5; i++ {
		allowed, err := store.Allow("test-user")
		if err != nil {
			t.Fatalf("Allow failed on iteration %d: %v", i, err)
		}
		if !allowed {
			t.Errorf("request %d should be allowed", i)
		}
	}
}

func TestRedisStore_Allow_ExceedsLimit(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := &RedisStore{Client: client}

	for i := 0; i < 21; i++ {
		store.Allow("test-user")
	}

	allowed, err := store.Allow("test-user")
	if err != nil {
		t.Fatalf("Allow failed: %v", err)
	}
	if allowed {
		t.Error("expected 22nd request to be denied")
	}
}

func TestRedisStore_Allow_DifferentIdentifiers(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := &RedisStore{Client: client}

	for i := 0; i < 25; i++ {
		store.Allow("user-1")
	}

	allowed, err := store.Allow("user-2")
	if err != nil {
		t.Fatalf("Allow failed: %v", err)
	}
	if !allowed {
		t.Error("expected different identifier to be allowed")
	}
}

func TestRedisStore_Allow_TTLReset(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := &RedisStore{Client: client}

	for i := 0; i < 25; i++ {
		store.Allow("test-user")
	}

	mr.FastForward(21 * time.Second)

	allowed, err := store.Allow("test-user")
	if err != nil {
		t.Fatalf("Allow failed: %v", err)
	}
	if !allowed {
		t.Error("expected request to be allowed after TTL expiry")
	}
}
