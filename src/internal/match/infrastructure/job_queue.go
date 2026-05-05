package infrastructure

import (
	"context"
	"encoding/json"

	"github.com/Aboody-Studios/ballr/src/internal/match/domain"
	"github.com/redis/go-redis/v9"
)

const analysisQueueKey = "analysis:queue"

type RedisJobQueue struct {
	client *redis.Client
}

func NewRedisJobQueue(client *redis.Client) *RedisJobQueue {
	return &RedisJobQueue{client: client}
}

func (q *RedisJobQueue) Push(ctx context.Context, job *domain.AnalysisJob) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return q.client.LPush(ctx, analysisQueueKey, data).Err()
}

func (q *RedisJobQueue) Pop(ctx context.Context) (*domain.AnalysisJob, error) {
	result, err := q.client.BRPop(ctx, 0, analysisQueueKey).Result()
	if err != nil {
		return nil, err
	}
	var job domain.AnalysisJob
	if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
		return nil, err
	}
	return &job, nil
}
