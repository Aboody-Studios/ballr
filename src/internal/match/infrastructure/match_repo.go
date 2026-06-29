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
	tx := r.DB.WithContext(ctx).Save(match)
	return tx.Error
}

func (r *PostgresMatchRepository) FindByID(ctx context.Context, id string) (*domain.Match, error) {
	var match domain.Match

	tx := r.DB.WithContext(ctx).Where("id = ?", id).Find(&match)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return &match, nil
}

func (r *PostgresMatchRepository) FindByUserID(ctx context.Context, userID string) ([]*domain.Match, error) {
	var matches []domain.Match

	tx := r.DB.WithContext(ctx).Where("user_id = ?", userID).Find(&matches)
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
	tx := r.DB.WithContext(ctx).Model(&domain.Match{}).Where("id = ?", matchID).Update("status", string(status))
	return tx.Error
}

func (r *PostgresMatchRepository) GetStuckMatches(ctx context.Context, cutOffTime time.Time) ([]*domain.Match, error) {
	var matches []*domain.Match
	
	tx := r.DB.WithContext(ctx).Model(&domain.Match{}).Where("status = ? AND analysis_flag = false AND updated_at < ?").Find(matches)
	if tx.Error != nil {
		return nil, tx.Error
	}
	
	return matches, nil
}

// ClaimStuckMatch atomically sets analysis_flag = true only if it was previously false.
// It returns true if this call successfully claimed the match (rows affected == 1).
func (r *PostgresMatchRepository) ClaimStuckMatch(ctx context.Context, matchID string) (bool, error) {
	tx := r.DB.WithContext(ctx).Model(&domain.Match{}).Where("id = ? AND analysis_flag = false", matchID).Update("analysis_flag", true)
	if tx.Error != nil {
		return false, tx.Error
	}
	return tx.RowsAffected > 0, nil
}

// UnclaimMatch sets analysis_flag = false for a match, used to revert a claim on failure.
func (r *PostgresMatchRepository) UnclaimMatch(ctx context.Context, matchID string) error {
	tx := r.DB.WithContext(ctx).Model(&domain.Match{}).Where("id = ?", matchID).Update("analysis_flag", false)
	return tx.Error
}
