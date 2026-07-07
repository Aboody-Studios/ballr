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
	progressInfra := FromProgressDomainToInfra(progress)
	tx := r.DB.WithContext(ctx).Model(Progress{}).Save(progressInfra)

	return tx.Error
}

func (r *PostgresProgressRepository) FindByUserID(ctx context.Context, userID string) (*domain.Progress, error) {
	var progress Progress
	tx := r.DB.WithContext(ctx).Model(Progress{}).Where("user_id = ?", userID).Find(&progress)
	progressDomain := FromProgressInfraToDomain(progress)

	return progressDomain, tx.Error
}
