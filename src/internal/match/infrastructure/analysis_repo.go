package infrastructure

import (
	"context"

	"github.com/Aboody-Studios/ballr/src/internal/match/domain"
	"gorm.io/gorm"
)

//TODO!: Remove Generic usage and use traditional API instead
//TODO!: Use infrastructure layer structs

type PostgresAnalysisRepository struct {
	*gorm.DB
}

func (r *PostgresAnalysisRepository) Save(ctx context.Context, analysis *domain.AnalysisResult) error {
	tx := r.DB.WithContext(ctx).Save(&analysis)
	return tx.Error
}

func (r *PostgresAnalysisRepository) FindByMatchID(ctx context.Context, matchID string) (*domain.AnalysisResult, error) {
	analysisResult, err := gorm.G[domain.AnalysisResult](r.DB).Where("id = ?", matchID).First(ctx)
	if err != nil {
		return nil, err
	}

	return &analysisResult, nil
}

func (r *PostgresAnalysisRepository) FindByID(ctx context.Context, id string) (*domain.AnalysisResult, error) {
	return r.FindByMatchID(ctx, id)
}

func (r *PostgresAnalysisRepository) UpdateSummary(ctx context.Context, matchID string, summary domain.AnalysisSummary) error {
	analysisResult, err := gorm.G[domain.AnalysisResult](r.DB).Where("id = ?", matchID).First(ctx)
	if err != nil {
		return err
	}
	analysisResult.Summary = summary

	tx := r.DB.Save(&analysisResult)
	return tx.Error
}

func (r *PostgresAnalysisRepository) AddEvent(ctx context.Context, matchID string, event domain.MatchEvent) error {
	analysisResult, err := gorm.G[domain.AnalysisResult](r.DB).Where("id = ?", matchID).First(ctx)
	if err != nil {
		return err
	}

	analysisResult.Events = append(analysisResult.Events, event)
	tx := r.DB.Save(&analysisResult)
	return tx.Error
}

func (r *PostgresAnalysisRepository) UpdateAnalysisID(ctx context.Context, matchID string, analysisID string) error {
	tx := r.DB.Model(&domain.AnalysisResult{}).Where("match_id = ?", matchID).Update("id", analysisID)
	return tx.Error
}
