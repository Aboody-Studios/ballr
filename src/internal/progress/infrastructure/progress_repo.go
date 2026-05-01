package infrastructure

import (
	"context"
	"time"

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

func (r *PostgresProgressRepository) AddPoints(ctx context.Context, userID string, points int64) error {
	tx := r.DB.Model(&domain.Progress{}).Where("user_id = ?", userID).
		Update("total_points", gorm.Expr("total_points + ?", points))
	return tx.Error
}

func (r *PostgresProgressRepository) UpdateStreak(ctx context.Context, userID string, activeDate string) error {
	progress, err := r.FindByUserID(ctx, userID)
	if err != nil {
		return err
	}
	parsed, err := time.Parse(time.RFC3339, activeDate)
	if err != nil {
		parsed, err = time.Parse("2006-01-02", activeDate)
		if err != nil {
			return err
		}
	}
	progress.UpdateStreak(parsed)
	return r.Save(ctx, progress)
}

type progressLeaderboardRow struct {
	UserID        string
	TotalPoints   int64
	CurrentStreak int
}

func (r *PostgresProgressRepository) GetLeaderboard(ctx context.Context, limit int) ([]*domain.LeaderboardEntry, error) {
	var rows []progressLeaderboardRow
	tx := r.DB.Model(&domain.Progress{}).
		Select("user_id, total_points, current_streak").
		Order("total_points DESC").
		Limit(limit).
		Find(&rows)
	if tx.Error != nil {
		return nil, tx.Error
	}

	entries := make([]*domain.LeaderboardEntry, len(rows))
	for i, row := range rows {
		entries[i] = &domain.LeaderboardEntry{
			Rank:        i + 1,
			UserID:      row.UserID,
			DisplayName: "",
			TotalPoints: row.TotalPoints,
			Streak:      row.CurrentStreak,
		}
	}
	return entries, nil
}
