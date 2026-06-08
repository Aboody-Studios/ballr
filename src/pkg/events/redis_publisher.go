package events

import (
	"context"
	"encoding/json"
	"fmt"

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

func (p *RedisPublisher) PublishEvent(ctx context.Context, event Event) error {

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	return p.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: p.stream,
		ID:     "*",
		Values: map[string]any{
			"data": string(data),
		},
	}).Err()
}
