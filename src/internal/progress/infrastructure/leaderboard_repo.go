package infrastructure

import (
	"context"

	"github.com/redis/go-redis/v9"
)

//TODO!: Use infrastructure layer structs

type RedisLeaderboardRepo struct {
	Client *redis.Client
}

func (rlr RedisLeaderboardRepo) UpdateScore(ctx context.Context, userID string, points int) error {
	err := rlr.Client.ZAdd(ctx, "global_leaderboard",
		redis.Z{
			Score:  float64(points),
			Member: userID,
		}).Err()

	return err
}

func (rlr RedisLeaderboardRepo) GetPlayers(ctx context.Context, offset, limit int64) ([]redis.Z, error) {
	redisZEntries, err := rlr.Client.ZRangeArgsWithScores(ctx, redis.ZRangeArgs{
		Key:   "global_leaderboard",
		Rev:   true,
		Start: offset,
		Stop:  offset + limit - 1,
	}).Result()

	if err != nil {
		return nil, err
	}

	return redisZEntries, nil
}

func (rlr RedisLeaderboardRepo) GetPlayerOffset(ctx context.Context, userID string) (int64, error) {
	rank, err := rlr.Client.ZRevRank(ctx, "global_leaderboard", userID).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, err
	}

	return rank, nil

}
