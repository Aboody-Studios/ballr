package application

import (
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/identity/domain"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type ProfileResponse struct {
	ID            string            `json:"id"`
	Email         string            `json:"email"`
	FullName      string            `json:"full_name"`
	AvatarURL     string            `json:"avatar_url"`
	Footedness    domain.Footedness `json:"footedness,omitempty"`
	Goals         string            `json:"goals,omitempty"`
	BirthDate     time.Time         `json:"birthdate,omitempty"`
	OAuthProvider string            `json:"oauth_provider"`
	CreatedAt     time.Time         `json:"created_at"`
}

type OnboardingRequest struct {
	Username     string               `json:"username" validate:"required"`
	FullName     string               `json:"fullName" validate:"required"`
	BirthDate    time.Time            `json:"birthdate" validate:"required,ltefield=Now"`
	Footedness   domain.Footedness    `json:"footedness" validate:"required,oneof=Left Right Both"`
	Goals        string               `json:"goals"`
	TrainingDays []domain.TrainingDay `json:"trainingdays"`
	Timezone     string               `json:"timezone"`
}
