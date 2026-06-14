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

// TODO!: Remove position from here as the player can have different positions in different matches.
type ProfileResponse struct {
	ID            string            `json:"id"`
	Email         string            `json:"email"`
	FullName      string            `json:"full_name"`
	AvatarURL     string            `json:"avatar_url"`
	Position      domain.Position   `json:"position,omitempty"`
	Footedness    domain.Footedness `json:"footedness,omitempty"`
	Goals         string            `json:"goals,omitempty"`
	BirthDate     time.Time         `json:"birthdate,omitempty"`
	OAuthProvider string            `json:"oauth_provider"`
	CreatedAt     time.Time         `json:"created_at"`
}

type OnboardingRequest struct {
	Email        string               `json:"email" validate:"required,email"`
	FullName     string               `json:"fullName" validate:"required"`
	BirthDate    time.Time            `json:"birthdate" validate:"required,ltefield=Now"`
	Position     domain.Position      `json:"position" validate:"required,oneof=GK CB LB RB CM LW RW ST"`
	Footedness   domain.Footedness    `json:"footedness" validate:"required,oneof=Left Right Both"`
	Goals        string               `json:"goals"`
	TrainingDays []domain.TrainingDay `json:"trainingdays"`
}
