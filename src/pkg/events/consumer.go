package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type Consumer struct {
	rdb      *redis.Client
	stream   string
	group    string
	consumer string

	mu       sync.RWMutex
	handlers map[string][]Handler
}

func NewConsumer(rdb *redis.Client, stream, group, consumer string) *Consumer {
	return &Consumer{
		rdb:      rdb,
		stream:   stream,
		group:    group,
		consumer: consumer,
		handlers: make(map[string][]Handler),
	}
}

func (c *Consumer) handle(eventType string, handler Handler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[eventType] = append(c.handlers[eventType], handler)
}

func (c *Consumer) HandleFunc(eventType string, fn func(ctx context.Context, event Event) error) {
	c.handle(eventType, HandlerFunc(fn))
}

func (c *Consumer) Start(ctx context.Context) error {
	if err := c.rdb.XGroupCreateMkStream(context.Background(), c.stream, c.group, "0").Err(); err != nil {
		if !strings.Contains(err.Error(), "BUSYGROUP") {
			return fmt.Errorf("create consumer group: %w", err)
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		result, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    c.group,
			Consumer: c.consumer,
			Streams:  []string{c.stream, ">"},
			Count:    10,
			Block:    2 * time.Second,
		}).Result()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			log.Printf("event consumer: xreadgroup error: %v", err)
			continue
		}

		for _, stream := range result {
			for _, msg := range stream.Messages {
				c.processMessage(ctx, msg)
			}
		}
	}
}

func (c *Consumer) processMessage(ctx context.Context, msg redis.XMessage) {
	raw, ok := msg.Values["data"].(string)
	if !ok {
		c.ack(ctx, msg.ID)
		return
	}

	var event Event
	if err := json.Unmarshal([]byte(raw), &event); err != nil {
		log.Printf("event consumer: unmarshal error for msg %s: %v", msg.ID, err)
		c.ack(ctx, msg.ID)
		return
	}

	c.mu.RLock()
	handlers := c.handlers[event.Type]
	c.mu.RUnlock()

	if len(handlers) == 0 {
		c.ack(ctx, msg.ID)
		return
	}

	for _, handler := range handlers {
		if err := handler.HandleEvent(ctx, event); err != nil {
			log.Printf("event consumer: handler error for event %s (msg %s): %v", event.Type, msg.ID, err)
			c.sendToDeadLetter(ctx, msg.ID, raw, err)
		}
	}

	c.ack(ctx, msg.ID)
}

func (c *Consumer) ack(ctx context.Context, msgID string) {
	if err := c.rdb.XAck(ctx, c.stream, c.group, msgID).Err(); err != nil {
		log.Printf("event consumer: ack error for msg %s: %v", msgID, err)
	}
}

func (c *Consumer) sendToDeadLetter(ctx context.Context, originalID, data string, handlerErr error) {
	if err := c.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: DeadLetterStream,
		ID:     "*",
		Values: map[string]interface{}{
			"original_id": originalID,
			"data":        data,
			"error":       handlerErr.Error(),
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
		},
	}).Err(); err != nil {
		log.Printf("event consumer: dead letter write error for msg %s: %v", originalID, err)
	}
}
