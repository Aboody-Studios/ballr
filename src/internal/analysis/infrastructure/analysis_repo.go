package infrastructure

import (
	"context"

	"github.com/Aboody-Studios/ballr/src/internal/analysis/domain"
	"gorm.io/gorm"
)

type PostgresAnalysisRepository struct {
	*gorm.DB
}

func (r *PostgresAnalysisRepository) Save(ctx context.Context, analysis *domain.AnalysisResult) error {
	match, err := gorm.G[domain.Match](r.DB).Where("id = ?", analysis.MatchID).First(ctx)
	if err != nil {
		return err
	}
	match.AnalysisResult = analysis
	tx := r.DB.Save(&match)
	return tx.Error
}

func (r *PostgresAnalysisRepository) FindByMatchID(ctx context.Context, matchID string) (*domain.AnalysisResult, error) {
	match, err := gorm.G[domain.Match](r.DB).Where("id = ?", matchID).First(ctx)
	if err != nil {
		return nil, err
	}
	if match.AnalysisResult == nil {
		return nil, domain.ErrAnalysisNotFound
	}
	return match.AnalysisResult, nil
}

func (r *PostgresAnalysisRepository) FindByID(ctx context.Context, id string) (*domain.AnalysisResult, error) {
	return r.FindByMatchID(ctx, id)
}

func (r *PostgresAnalysisRepository) UpdateSummary(ctx context.Context, matchID string, summary domain.AnalysisSummary) error {
	match, err := gorm.G[domain.Match](r.DB).Where("id = ?", matchID).First(ctx)
	if err != nil {
		return err
	}
	if match.AnalysisResult == nil {
		match.AnalysisResult = &domain.AnalysisResult{MatchID: matchID}
	}
	match.AnalysisResult.Summary = summary
	tx := r.DB.Save(&match)
	return tx.Error
}

func (r *PostgresAnalysisRepository) AddEvent(ctx context.Context, matchID string, event domain.MatchEvent) error {
	match, err := gorm.G[domain.Match](r.DB).Where("id = ?", matchID).First(ctx)
	if err != nil {
		return err
	}
	if match.AnalysisResult == nil {
		match.AnalysisResult = &domain.AnalysisResult{MatchID: matchID}
	}
	match.AnalysisResult.Events = append(match.AnalysisResult.Events, event)
	tx := r.DB.Save(&match)
	return tx.Error
}
