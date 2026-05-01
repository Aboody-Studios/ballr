package infrastructure

import (
	"context"

	coachdomain "github.com/Aboody-Studios/ballr/src/internal/coach/domain"
	"gorm.io/gorm"
)

type PostgresConversationRepository struct {
	*gorm.DB
}

func NewPostgresConversationRepository(db *gorm.DB) *PostgresConversationRepository {
	return &PostgresConversationRepository{DB: db}
}

func (r *PostgresConversationRepository) GetConversation(ctx context.Context, userID, sessionID string) (*coachdomain.Conversation, error) {
	conv, err := gorm.G[coachdomain.Conversation](r.DB).
		Where("user_id = ? AND session_id = ?", userID, sessionID).
		First(ctx)
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *PostgresConversationRepository) SaveConversation(ctx context.Context, conversation *coachdomain.Conversation) error {
	tx := r.DB.Save(conversation)
	return tx.Error
}
