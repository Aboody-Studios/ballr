package infrastructure

import (
	"context"

	"github.com/Aboody-Studios/ballr/src/internal/progress/domain"
	"gorm.io/gorm"
)

type PostgresLeaderboardRepository struct {
	*gorm.DB
}

func (r *PostgresLeaderboardRepository) UpdateScore(ctx context.Context, userID string, points int) error {
	tx := r.DB.Model(&domain.Progress{}).Where("user_id = ?", userID).
		Update("total_points", points)
	return tx.Error
}

type leaderboardTopPlayerRow struct {
	UserID        string
	TotalPoints   int
	CurrentStreak int
}

func (r *PostgresLeaderboardRepository) GetTopPlayers(ctx context.Context, offset, limit int) ([]domain.LeaderboardEntry, error) {
	var rows []leaderboardTopPlayerRow
	tx := r.DB.Model(&domain.Progress{}).
		Select("user_id, total_points, current_streak").
		Order("total_points DESC").
		Offset(offset).
		Limit(limit).
		Find(&rows)
	if tx.Error != nil {
		return nil, tx.Error
	}

	entries := make([]domain.LeaderboardEntry, len(rows))
	for i, row := range rows {
		entries[i] = domain.LeaderboardEntry{
			Rank:        offset + i + 1,
			UserID:      row.UserID,
			DisplayName: "",
			TotalPoints: row.TotalPoints,
			Streak:      row.CurrentStreak,
		}
	}
	return entries, nil
}
