package infrastructure

import (
	"context"
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/match/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

//TODO!: Use infrastructure layer structs

type PostgresMatchRepository struct {
	*gorm.DB
}

func (r *PostgresMatchRepository) Save(ctx context.Context, match *domain.Match) error {
	if match.ID == "" {
		match.ID = uuid.New().String()
	}

	matchInfra := FromMatchDomainToInfra(*match)
	tx := r.DB.WithContext(ctx).Model(Match{}).Create(&matchInfra)
	return tx.Error
}

func (r *PostgresMatchRepository) FindByID(ctx context.Context, id string) (*domain.Match, error) {
	var match Match

	tx := r.DB.WithContext(ctx).Model(Match{}).Where("id = ?", id).First(&match)
	if tx.Error != nil {
		return nil, tx.Error
	}

	matchDomain := FromMatchInfraToDomain(match)
	return matchDomain, nil
}

func (r *PostgresMatchRepository) FindByUserID(ctx context.Context, userID string) ([]*domain.Match, error) {
	var matchesInfra []*Match

	tx := r.DB.WithContext(ctx).Where("user_id = ?", userID).Find(&matchesInfra)
	matchesDom := make([]*domain.Match, len(matchesInfra))

	if tx.Error != nil {
		return nil, tx.Error
	}

	for i, infraMatch := range matchesInfra {
		matchesDom[i] = FromMatchInfraToDomain(*infraMatch)
	}

	return matchesDom, nil
}

func (r *PostgresMatchRepository) UpdateStatus(ctx context.Context, matchID string, status domain.MatchStatus) error {
	tx := r.DB.WithContext(ctx).Model(Match{}).Where("id = ?", matchID).Update("status", string(status))
	return tx.Error
}

func (r *PostgresMatchRepository) GetStuckMatches(ctx context.Context, cutOffTime time.Time) ([]*domain.Match, error) {
	var matches []*Match

	tx := r.DB.WithContext(ctx).Model(Match{}).Where("status = 'PROCESSING' AND analysis_flag = false AND updated_at < ?", cutOffTime).Find(&matches)
	if tx.Error != nil {
		return nil, tx.Error
	}

	matchesDomain := make([]*domain.Match, len(matches))
	for i, infraMatch := range matches {
		matchesDomain[i] = FromMatchInfraToDomain(*infraMatch)
	}

	return matchesDomain, nil
}

// ClaimStuckMatch atomically sets analysis_flag = true only if it was previously false.
// It returns true if this call successfully claimed the match (rows affected == 1).
func (r *PostgresMatchRepository) ClaimStuckMatch(ctx context.Context, matchID string) (bool, error) {
	tx := r.DB.WithContext(ctx).Model(Match{}).Where("id = ? AND analysis_flag = false", matchID).Update("analysis_flag", true)
	if tx.Error != nil {
		return false, tx.Error
	}
	return tx.RowsAffected > 0, nil
}

// UnclaimMatch sets analysis_flag = false for a match, used to revert a claim on failure.
func (r *PostgresMatchRepository) UnclaimMatch(ctx context.Context, matchID string) error {
	tx := r.DB.WithContext(ctx).Model(Match{}).Where("id = ?", matchID).Update("analysis_flag", false)
	return tx.Error
}
