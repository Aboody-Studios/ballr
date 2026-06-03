package infrastructure

import (
	"context"
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/match/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostgresMatchRepository struct {
	*gorm.DB
}

func (r *PostgresMatchRepository) Save(ctx context.Context, match *domain.Match) error {
	if match.ID == "" {
		match.ID = uuid.New().String()
	}
	tx := r.DB.Save(match)
	return tx.Error
}

func (r *PostgresMatchRepository) FindByID(ctx context.Context, id string) (*domain.Match, error) {
	match, err := gorm.G[domain.Match](r.DB).Where("id = ?", id).First(ctx)
	if err != nil {
		return nil, err
	}
	return &match, nil
}

func (r *PostgresMatchRepository) FindByUserID(ctx context.Context, userID string) ([]*domain.Match, error) {
	var matches []domain.Match
	tx := r.DB.Where("user_id = ?", userID).Find(&matches)
	if tx.Error != nil {
		return nil, tx.Error
	}
	result := make([]*domain.Match, len(matches))
	for i := range matches {
		result[i] = &matches[i]
	}
	return result, nil
}

func (r *PostgresMatchRepository) UpdateStatus(ctx context.Context, matchID string, status domain.MatchStatus) error {
	tx := r.DB.Model(&domain.Match{}).Where("id = ?", matchID).Update("status", string(status))
	return tx.Error
}

func (r *PostgresMatchRepository) GetStuckMatches(ctx context.Context, cutOffTime time.Time) ([]*domain.Match, error) {
	matches, err := gorm.G[*domain.Match](r.DB).Where("status = ? AND analysis_flag = false AND updated_at < ?", domain.MatchStatusProcessing, cutOffTime).Find(ctx)
	if err != nil {
		return nil, err
	}

	return matches, nil
}

// ClaimStuckMatch atomically sets analysis_flag = true only if it was previously false.
// It returns true if this call successfully claimed the match (rows affected == 1).
func (r *PostgresMatchRepository) ClaimStuckMatch(ctx context.Context, matchID string) (bool, error) {
	tx := r.DB.Model(&domain.Match{}).Where("id = ? AND analysis_flag = false", matchID).Update("analysis_flag", true)
	if tx.Error != nil {
		return false, tx.Error
	}
	return tx.RowsAffected > 0, nil
}

// UnclaimMatch sets analysis_flag = false for a match, used to revert a claim on failure.
func (r *PostgresMatchRepository) UnclaimMatch(ctx context.Context, matchID string) error {
	tx := r.DB.Model(&domain.Match{}).Where("id = ?", matchID).Update("analysis_flag", false)
	return tx.Error
}
