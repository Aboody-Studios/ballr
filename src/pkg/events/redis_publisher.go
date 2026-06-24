package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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

func (rp *RedisPublisher) PublishEvent(ctx context.Context, ev Event) error {
	if ev.ID == "" {
		ev.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now()
	}

	data, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	return rp.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: rp.stream,
		ID:     "*",
		Values: map[string]any{
			"data": string(data),
		},
	}).Err()
}
