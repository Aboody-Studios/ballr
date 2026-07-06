package infrastructure

import "github.com/Aboody-Studios/ballr/src/internal/identity/domain"

func FromUserInfraToDomain(userInfra User) *domain.User {
	userDomainTDays := make([]domain.TrainingDay, len(userInfra.TrainingDays))
	for i, day := range userInfra.TrainingDays {
		userDomainTDays[i] = domain.TrainingDay(day)
	}

	return &domain.User{
		ID:            userInfra.ID,
		Email:         userInfra.Email,
		Username:      userInfra.Username,
		AvatarURL:     userInfra.AvatarURL,
		FullName:      userInfra.FullName,
		Footedness:    domain.Footedness(userInfra.Footedness),
		BirthDate:     userInfra.BirthDate,
		Timezone:      userInfra.Timezone,
		CreatedAt:     userInfra.CreatedAt,
		TrainingDays:  userDomainTDays,
		OAuthProvider: userInfra.OAuthProvider,
	}
}
