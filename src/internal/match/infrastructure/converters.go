package infrastructure

import "github.com/Aboody-Studios/ballr/src/internal/match/domain"

func FromMatchInfraToDomain(matchInfra Match) *domain.Match {
	return &domain.Match{
		ID: matchInfra.ID,
		UserID: matchInfra.UserID,
		ShirtNumber: matchInfra.ShirtNumber,
		PositionPlayed: matchInfra.PositionPlayed,
		VideoURL: matchInfra.VideoURL,
		Status: matchInfra.Status,
		AnalysisFlag: matchInfra.AnalysisFlag,
		CreatedAt: matchInfra.CreatedAt,
		UpdatedAt: matchInfra.UpdatedAt,
	}
}