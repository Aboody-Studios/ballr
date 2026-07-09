package infrastructure

import "github.com/Aboody-Studios/ballr/src/internal/match/domain"

func FromMatchInfraToDomain(matchInfra Match) *domain.Match {
	return &domain.Match{
		ID:             matchInfra.ID,
		UserID:         matchInfra.UserID,
		ShirtNumber:    matchInfra.ShirtNumber,
		PositionPlayed: domain.Position(matchInfra.PositionPlayed),
		VideoURL:       matchInfra.VideoURL,
		Status:         domain.MatchStatus(matchInfra.Status),
		AnalysisFlag:   matchInfra.AnalysisFlag,
		CreatedAt:      matchInfra.CreatedAt,
		UpdatedAt:      matchInfra.UpdatedAt,
	}
}

func FromMatchDomainToInfra(matchDomain domain.Match) *Match {
	return &Match{
		ID:             matchDomain.ID,
		UserID:         matchDomain.UserID,
		ShirtNumber:    matchDomain.ShirtNumber,
		PositionPlayed: Position(matchDomain.PositionPlayed),
		VideoURL:       matchDomain.VideoURL,
		Status:         MatchStatus(matchDomain.Status),
		AnalysisFlag:   matchDomain.AnalysisFlag,
		AnalysisResult: matchDomain.AnalysisResult,
		CreatedAt:      matchDomain.CreatedAt,
		UpdatedAt:      matchDomain.UpdatedAt,
		Metadata:       MatchMetadata(matchDomain.Metadata),
	}
}
