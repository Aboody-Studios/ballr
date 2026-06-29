package application

import (
	"context"
	"slices"
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/coach/domain"
	"github.com/google/uuid"
)

type CoachService struct {
	CoachRepo       domain.CoachRepo
	AICoachProvider domain.AICoachProvider
}

func NewCoachService(coachRepo domain.CoachRepo, coachProvider domain.AICoachProvider) CoachService {
	return CoachService{
		CoachRepo: coachRepo,
		AICoachProvider: coachProvider,
	}
}

func (cs *CoachService) ResponseGenerationOrchestrator(ctx context.Context, userID string, msg string) (string, error) {
	// Get chat history before saving new user message to not include it in the history
	chatMessages, err := cs.CoachRepo.GetChatHistory(ctx, userID)
	if err != nil {
		return "", err
	}

	// Persist user message to database before sending it to coach
	userMsg := createMsgStruct("user", msg, userID, time.Now())
	if err := cs.saveChatMsgService(ctx, userMsg); err != nil {
		return "", err
	}

	slices.Reverse(chatMessages)

	response, resErr := cs.AICoachProvider.GenerateResponse(ctx, msg, chatMessages)
	if resErr != nil {
		return "", resErr
	}

	// Persist coach message to database before sending it back to the user
	coachMsg := createMsgStruct("model", response, userID, time.Now())
	if err := cs.saveChatMsgService(ctx, coachMsg); err != nil {
		return "", err
	}

	return response, nil
}

func (cs *CoachService) saveChatMsgService(ctx context.Context, msg domain.ChatMessage) error {
	if err := cs.CoachRepo.SaveMessage(ctx, msg); err != nil {
		return err
	}

	return nil
}

func createMsgStruct(role, msgTxt, userID string, msgTime time.Time) domain.ChatMessage {
	return domain.ChatMessage{
		ID:        uuid.NewString(),
		UserID:    userID,
		Role:      role,
		Content:   msgTxt,
		CreatedAt: msgTime,
	}
}
