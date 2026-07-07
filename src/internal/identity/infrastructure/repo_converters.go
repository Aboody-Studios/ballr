package infrastructure

import (
	"github.com/Aboody-Studios/ballr/src/internal/identity/domain"
)

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

func FromUserDomainToInfra(userDomain *domain.User) *User {
	userInfraTDays := make([]TrainingDay, len(userDomain.TrainingDays))
	for i, day := range userDomain.TrainingDays {
		userInfraTDays[i] = TrainingDay(day)
	}

	return &User{
		ID:            userDomain.ID,
		Email:         userDomain.Email,
		Username:      userDomain.Username,
		AvatarURL:     userDomain.AvatarURL,
		FullName:      userDomain.FullName,
		Footedness:    Footedness(userDomain.Footedness),
		BirthDate:     userDomain.BirthDate,
		Timezone:      userDomain.Timezone,
		CreatedAt:     userDomain.CreatedAt,
		TrainingDays:  userInfraTDays,
		OAuthProvider: userDomain.OAuthProvider,
	}
}
