package infrastructure

import (
	"context"
	"sort"

	coachdomain "github.com/Aboody-Studios/ballr/src/internal/coach/domain"
	analysisdomain "github.com/Aboody-Studios/ballr/src/internal/match/domain"
)

type CoachAnalysisBridge struct {
	matchRepo    analysisdomain.MatchRepository
	analysisRepo analysisdomain.AnalysisRepository
}

func NewCoachAnalysisBridge(matchRepo analysisdomain.MatchRepository, analysisRepo analysisdomain.AnalysisRepository) *CoachAnalysisBridge {
	return &CoachAnalysisBridge{
		matchRepo:    matchRepo,
		analysisRepo: analysisRepo,
	}
}

func (b *CoachAnalysisBridge) GetLatestAnalysis(ctx context.Context, userID string) (*coachdomain.MatchInsight, error) {
	matches, err := b.matchRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].CreatedAt.After(matches[j].CreatedAt)
	})

	for _, m := range matches {
		if m.CanViewResults() {
			return matchToInsight(m), nil
		}
	}

	return nil, nil
}

func (b *CoachAnalysisBridge) GetMatchHistory(ctx context.Context, userID string, limit int) ([]*coachdomain.MatchInsight, error) {
	if limit <= 0 {
		return []*coachdomain.MatchInsight{}, nil
	}

	matches, err := b.matchRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].CreatedAt.After(matches[j].CreatedAt)
	})

	result := make([]*coachdomain.MatchInsight, 0, min(limit, len(matches)))
	for _, m := range matches {
		if m.CanViewResults() {
			result = append(result, matchToInsight(m))
			if len(result) >= limit {
				break
			}
		}
	}

	return result, nil
}

func matchToInsight(m *analysisdomain.AnalysisResult) *coachdomain.MatchInsight {
	insight := &coachdomain.MatchInsight{
		MatchID:     m.ID,
		MatchDate:   m.Match.Metadata.MatchDate,
		DurationMin: m.Match.Metadata.Duration / 60,
	}
	if m != nil {
		insight.DistanceKM = m.Summary.TotalDistanceKM
		insight.TopSpeedKMH = m.Summary.TopSpeedKMH
		insight.PassAccuracy = m.Summary.PassAccuracy
	}
	return insight
}
