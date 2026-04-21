package infrastructure

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/identity/domain"
	"github.com/redis/go-redis/v9"
)

type RedisRefreshTokenStore struct {
	Client *redis.Client
	TTL    time.Duration
}

func NewRedisRefreshTokenStore(client *redis.Client, ttl time.Duration) *RedisRefreshTokenStore {
	return &RedisRefreshTokenStore{Client: client, TTL: ttl}
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func (s *RedisRefreshTokenStore) Store(ctx context.Context, rawToken string, data *domain.RefreshTokenData, ttl time.Duration) error {
	h := hashToken(rawToken)
	key := fmt.Sprintf("refresh_token:%s", h)
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	userKey := fmt.Sprintf("user_tokens:%s", data.UserID)
	pipe := s.Client.Pipeline()
	pipe.Set(ctx, key, payload, ttl)
	pipe.SAdd(ctx, userKey, h)
	pipe.Expire(ctx, userKey, ttl)
	_, err = pipe.Exec(ctx)
	return err
}

func (s *RedisRefreshTokenStore) Get(ctx context.Context, rawToken string) (*domain.RefreshTokenData, error) {
	h := hashToken(rawToken)
	key := fmt.Sprintf("refresh_token:%s", h)
	payload, err := s.Client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	var data domain.RefreshTokenData
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (s *RedisRefreshTokenStore) Delete(ctx context.Context, rawToken string) error {
	h := hashToken(rawToken)
	key := fmt.Sprintf("refresh_token:%s", h)
	return s.Client.Del(ctx, key).Err()
}

func (s *RedisRefreshTokenStore) RevokeAllForUser(ctx context.Context, userID string) error {
	userKey := fmt.Sprintf("user_tokens:%s", userID)
	hashes, err := s.Client.SMembers(ctx, userKey).Result()
	if err != nil {
		return err
	}
	if len(hashes) == 0 {
		return nil
	}
	keys := make([]string, len(hashes))
	for i, h := range hashes {
		keys[i] = fmt.Sprintf("refresh_token:%s", h)
	}
	keys = append(keys, userKey)
	return s.Client.Del(ctx, keys...).Err()
}
