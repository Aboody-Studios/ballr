package infrastructure

import (
	"context"

	"github.com/Aboody-Studios/ballr/src/internal/progress/domain"
	"gorm.io/gorm"
)

//TODO!: Use infrastructure layer structs

type PostgresEventLogRepository struct {
	*gorm.DB
}

func (r *PostgresEventLogRepository) Save(ctx context.Context, event *domain.EventLog) error {
	eventInfra := FromEventLogDomainToInfra(event)
	tx := r.DB.WithContext(ctx).Model(EventLog{}).Create(eventInfra)

	return tx.Error
}

func (r *PostgresEventLogRepository) FindRecentByUserID(ctx context.Context, userID string, limit int) ([]*domain.EventLog, error) {
	var events []EventLog

	tx := r.DB.WithContext(ctx).Model(EventLog{}).Where("user_id = ?", userID).Order("timestamp DESC").Limit(limit).Find(&events)
	if tx.Error != nil {
		return nil, tx.Error
	}
	
	eventsInfra := make([]*domain.EventLog, len(events))
	for i, event := range events {
		eventsInfra[i] = FromEventlogInfraToDomain(event)
	}

	return eventsInfra, nil
}
