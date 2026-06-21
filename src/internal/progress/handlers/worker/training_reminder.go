package worker

import (
	"context"

	"github.com/Aboody-Studios/ballr/src/internal/progress/application"
	"github.com/hibiken/asynq"
)

type TrainingReminderHandler struct {
	*application.GamificationService
}

func NewTrainingReminderHandler(gService *application.GamificationService) *TrainingReminderHandler {
	return &TrainingReminderHandler{
		GamificationService: gService,
	}
}

// Gets executed at the start of every hour to fetch users whose local time is 8 pm and today is a training day for them
// // TODO!: Decide on the notification sending infrastructure (firebase, onesignal, etc.) to store the targets returned successfully.
func (gs *TrainingReminderHandler) HandleBatchTrainingReminders(ctx context.Context, task *asynq.Task) error {
	_, err := gs.Get8pmAndTrainingDayUsersService(ctx)
	if err != nil {
		return err
	}

	return nil
}
