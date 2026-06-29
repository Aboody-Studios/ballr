package domain

import (
	"context"
)

type AICoachProvider interface {
	GenerateResponse(ctx context.Context, msg string, chat_messages []ChatMessage) (string, error)
}

type CoachRepo interface {
	GetChatHistory(ctx context.Context, userID string) ([]ChatMessage, error)
	SaveMessage(ctx context.Context, chatMsg ChatMessage) error
}
