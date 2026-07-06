package worker

import (
	"context"

	"github.com/Aboody-Studios/ballr/src/internal/progress/application"
	"github.com/hibiken/asynq"
	//"github.com/sideshow/apns2"
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
// TODO!: use APN here
func (gs *TrainingReminderHandler) BatchTrainingRemindersHandler(ctx context.Context, task *asynq.Task) error {
	_, err := gs.Get8pmAndTrainingDayUsersService(ctx)
	if err != nil {
		return err
	}

	//TODO!: Add abody's certs after signing up to the apple developer program
	// client := apns2.NewClient(cert).Production()

	/*for _, target := range targets {
		notification := &apns2.Notification{
			DeviceToken: target.DeviceToken,
			Payload:     []byte(`{"aps":{"alert":"Did you train today ?"}}`),
			Topic:       "change to app id whenever there is one",
		}
		// res, err := client.Push(notification)

	}*/

	return nil
}
