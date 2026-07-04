package infrastructure

import "github.com/Aboody-Studios/ballr/src/internal/progress/domain"

func FromProgressInfraToDomain(progressInfra Progress) *domain.Progress {
	return &domain.Progress{
		ID: progressInfra.ID,
		UserID: progressInfra.UserID,
		TotalPoints: progressInfra.TotalPoints,
		CurrentStreak: progressInfra.CurrentStreak,
		LastActive: progressInfra.LastActive,
		CreatedAt: progressInfra.CreatedAt,
		UpdatedAt: progressInfra.UpdatedAt,
	}
}

func FromAchievInfraToDomain(achievInfa Achievement) *domain.Achievement{
	return &domain.Achievement{
		ID: achievInfa.ID,
		ProgressID: achievInfa.ProgressID,
		UserID: achievInfa.UserID,
		Type: achievInfa.Type,
		UnlockedAt: achievInfa.UnlockedAt,
		PointsValue: achievInfa.PointsValue,
		Badge: achievInfa.Badge,
	}
}

func FromEventlogInfraToDomain(eventLog EventLog) *domain.EventLog {
	return &domain.EventLog{
		UserID: eventLog.UserID,
		Type: eventLog.Type,
		PointsAwarded: eventLog.PointsAwarded,
		ID: eventLog.ID,
		Timestamp: eventLog.Timestamp,
	}
}