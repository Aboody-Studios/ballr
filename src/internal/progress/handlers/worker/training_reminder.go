package worker

import (
	"context"

	"github.com/hibiken/asynq"
)

// TODO!: Query database for users that have training days and the time is 8 pm
func HandleBatchTrainingReminders(ctx context.Context, task *asynq.Task) error {

}
