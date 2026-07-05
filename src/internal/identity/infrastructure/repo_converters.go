package infrastructure

import "github.com/Aboody-Studios/ballr/src/internal/identity/domain"

func FromUserInfraToDomain(userInfra User) *domain.User {
	return &domain.User{
		ID:            userInfra.ID,
		Email:         userInfra.Email,
		Username:      userInfra.Username,
		AvatarURL:     userInfra.AvatarURL,
		FullName:      userInfra.FullName,
		Footedness:    userInfra.Footedness,
		BirthDate:     userInfra.BirthDate,
		Timezone:      userInfra.Timezone,
		CreatedAt:     userInfra.CreatedAt,
		TrainingDays:  userInfra.TrainingDays,
		OAuthProvider: userInfra.OAuthProvider,
	}
}
