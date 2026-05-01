package infrastructure

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	*redis.Client
}

func InitiateRedis() *redis.Client {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	password := os.Getenv("REDIS_PASSWORD")

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
		Protocol: 2,
	})

	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		log.Fatal("Database unreachable")
	}

	return rdb
}

func (r *RedisStore) Allow(identifier string) (bool, error) {
	redisCtx := context.Background()
	key := fmt.Sprintf("rate_limit:%s", identifier)
	// needs to be atomic with if count == 1 in case server fails after getting result.
	count, err := r.Incr(redisCtx, key).Result()

	if err != nil {
		return false, err
	}

	if count == 1 {
		_, err := r.Expire(redisCtx, key, time.Second*20).Result()
		if err != nil {
			return false, err
		}

		return true, nil
	}

	if count > 20 {
		return false, nil
	}

	return true, nil
}
