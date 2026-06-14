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
	tx := r.DB.Save(progress)
	return tx.Error
}

func (r *PostgresProgressRepository) FindByUserID(ctx context.Context, userID string) (*domain.Progress, error) {
	progress, err := gorm.G[domain.Progress](r.DB).Where("user_id = ?", userID).First(ctx)
	if err != nil {
		return nil, err
	}
	return &progress, nil
}
