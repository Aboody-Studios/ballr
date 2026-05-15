package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RedisPublisher struct {
	rdb    *redis.Client
	stream string
}

func NewRedisPublisher(rdb *redis.Client) *RedisPublisher {
	return &RedisPublisher{
		rdb:    rdb,
		stream: DefaultStream,
	}
}

func (p *RedisPublisher) PublishEvent(ctx context.Context, userID string, eventType string, metadata map[string]interface{}) error {
	event := Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		UserID:    userID,
		Timestamp: time.Now().UTC(),
		Metadata:  metadata,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	return p.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: p.stream,
		ID:     "*",
		Values: map[string]interface{}{
			"data": string(data),
		},
	}).Err()
}
