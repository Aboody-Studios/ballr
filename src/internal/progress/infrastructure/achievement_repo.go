package infrastructure

import (
	"context"

	"github.com/Aboody-Studios/ballr/src/internal/progress/domain"
	"gorm.io/gorm"
)

type PostgresAchievementRepository struct {
	*gorm.DB
}

func (r *PostgresAchievementRepository) Save(ctx context.Context, achievement *domain.Achievement) error {
	tx := r.DB.Create(achievement)
	return tx.Error
}

func (r *PostgresAchievementRepository) FindByUserID(ctx context.Context, userID string) ([]*domain.Achievement, error) {
	var achievements []domain.Achievement
	tx := r.DB.Where("user_id = ?", userID).Order("unlocked_at DESC").Find(&achievements)
	if tx.Error != nil {
		return nil, tx.Error
	}
	result := make([]*domain.Achievement, len(achievements))
	for i := range achievements {
		result[i] = &achievements[i]
	}
	return result, nil
}

func (r *PostgresAchievementRepository) FindByType(ctx context.Context, achievementType string) ([]*domain.Achievement, error) {
	var achievements []domain.Achievement
	tx := r.DB.Where("type = ?", achievementType).Find(&achievements)
	if tx.Error != nil {
		return nil, tx.Error
	}
	result := make([]*domain.Achievement, len(achievements))
	for i := range achievements {
		result[i] = &achievements[i]
	}
	return result, nil
}

func (r *PostgresAchievementRepository) HasAchievement(ctx context.Context, userID string, achievementType string) (bool, error) {
	var count int64
	tx := r.DB.Model(&domain.Achievement{}).
		Where("user_id = ? AND type = ?", userID, achievementType).
		Count(&count)
	return count > 0, tx.Error
}
