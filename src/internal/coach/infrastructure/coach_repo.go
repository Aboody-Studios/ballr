package infrastructure

import (
	"context"

	"github.com/Aboody-Studios/ballr/src/internal/coach/domain"
	"gorm.io/gorm"
)

type PostgresCoachRepo struct {
	*gorm.DB
}

func (pcr *PostgresCoachRepo) GetChatHistory(ctx context.Context, userID string) ([]domain.ChatMessage, error) {
	var chat_messages []domain.ChatMessage

	// This fetches the messages reversed.
	tx := pcr.WithContext(ctx).Table("chat_message").Where("user_id = ?", userID).Order("created_at DESC").Limit(10).Find(&chat_messages)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return chat_messages, nil
}

func (pcr *PostgresCoachRepo) SaveMessage(ctx context.Context, msg domain.ChatMessage) error {
	tx := pcr.WithContext(ctx).Create(&msg)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}
