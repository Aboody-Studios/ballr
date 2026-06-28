package application

import (
	"context"
	"slices"

	"github.com/Aboody-Studios/ballr/src/internal/coach/domain"
)

type CoachService struct {
	CoachRepo       domain.CoachRepo
	AICoachProvider domain.AICoachProvider
}

//TODO!: Save the user's new message and the AI's reply to the database

func (cs *CoachService) GenResponse(ctx context.Context, userID string, msg string) (string, error) {
	chatMessages, err := cs.CoachRepo.GetChatHistory(ctx, userID)
	if err != nil {
		return "", err
	}

	slices.Reverse(chatMessages)
	response, resErr := cs.AICoachProvider.GenerateResponse(ctx, msg, chatMessages)
	if resErr != nil {
		return "", resErr
	}

	return response, nil

}
