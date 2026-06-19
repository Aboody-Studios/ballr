package infrastructure

import (
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

func initiateAsynqServer(rdc redis.Client) (*asynq.Server, *asynq.RedisClientOpt) {
	asynqRedisClientOpt := asynq.RedisClientOpt{Addr: rdc.NodeAddress()}
	asynqServer := asynq.NewServer(
		asynq.RedisClientOpt{Addr: rdc.NodeAddress()},
		asynq.Config{
			Concurrency: 10,
		},
	)

	return asynqServer, &asynqRedisClientOpt
}

// Create a scheduler that adds batch_training_reminders to Redis as JSON at the top of every hour (UTC)
func AsynqScheduler(asynqRedisClientOpt *asynq.RedisClientOpt) error {
	scheduler := asynq.NewScheduler(asynqRedisClientOpt, &asynq.SchedulerOpts{})
	task := asynq.NewTask("task:batch_training_reminders", nil)

	_, err := scheduler.Register("0 * * * *", task)
	if err != nil {
		return err
	}

	scheduler.Start()
	return nil

}
