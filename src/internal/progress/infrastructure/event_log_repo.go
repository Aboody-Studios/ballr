package infrastructure

import (
	"context"

	"github.com/Aboody-Studios/ballr/src/internal/progress/domain"
	"gorm.io/gorm"
)

type PostgresEventLogRepository struct {
	*gorm.DB
}

func (r *PostgresEventLogRepository) Save(ctx context.Context, event *domain.EventLog) error {
	tx := r.DB.Create(event)
	return tx.Error
}

func (r *PostgresEventLogRepository) FindRecentByUserID(ctx context.Context, userID string, limit int) ([]*domain.EventLog, error) {
	var events []domain.EventLog
	tx := r.DB.Where("user_id = ?", userID).Order("timestamp DESC").Limit(limit).Find(&events)
	if tx.Error != nil {
		return nil, tx.Error
	}
	result := make([]*domain.EventLog, len(events))
	for i := range events {
		result[i] = &events[i]
	}
	return result, nil
}
