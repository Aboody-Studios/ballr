package infrastructure

import (
	"context"

	"github.com/Aboody-Studios/ballr/src/internal/progress/domain"
	"gorm.io/gorm"
)

type PostgresProgressRepository struct {
	*gorm.DB
}

func (r *PostgresProgressRepository) Save(ctx context.Context, progress *domain.Progress) error {
	tx := r.DB.WithContext(ctx).Save(progress)
	return tx.Error
}

func (r *PostgresProgressRepository) FindByUserID(ctx context.Context, userID string) (*domain.Progress, error) {
	var progress domain.Progress
	tx := r.DB.WithContext(ctx).Model(domain.Progress{}).Where("user_id = ?", userID).Find(&progress)
	
	return &progress, tx.Error
}
