package infrastructure

import (
	"context"

	"github.com/Aboody-Studios/ballr/src/internal/analysis/domain"
	"gorm.io/gorm"
)

type PostgresMatchRepository struct {
	*gorm.DB
}

func (r *PostgresMatchRepository) Save(ctx context.Context, match *domain.Match) error {
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

func (r *PostgresMatchRepository) UpdateAnalysisID(ctx context.Context, matchID string, analysisID string) error {
	match, err := gorm.G[domain.Match](r.DB).Where("id = ?", matchID).First(ctx)
	if err != nil {
		return err
	}
	match.AnalysisResult = &domain.AnalysisResult{MatchID: analysisID}
	tx := r.DB.Save(&match)
	return tx.Error
}
