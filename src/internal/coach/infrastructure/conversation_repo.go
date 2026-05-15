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

func (r *PostgresConversationRepository) FindByUserID(ctx context.Context, userID string, limit, offset int) ([]*coachdomain.Conversation, error) {
	var conversations []coachdomain.Conversation
	tx := r.DB.Where("user_id = ?", userID).
		Order("updated_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&conversations)
	if tx.Error != nil {
		return nil, tx.Error
	}
	result := make([]*coachdomain.Conversation, len(conversations))
	for i := range conversations {
		result[i] = &conversations[i]
	}
	return result, nil
}
