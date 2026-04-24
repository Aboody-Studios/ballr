package application

import (
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/identity/domain"
)

type SignupRequest struct {
	Email      string            `json:"email" validate:"required,email"`
	Password   string            `json:"password" validate:"required,min=8"`
	FullName   string            `json:"fullName" validate:"required"`
	BirthDate  time.Time         `json:"birthdate" validate:"required,ltefield=Now"`
	Position   domain.Position   `json:"position" validate:"required,oneof=GK CB LB RB CM LW RW ST"`
	Footedness domain.Footedness `json:"footedness" validate:"required,oneof=Left Right Both"`
	Goals      string            `json:"goals"`
}

//TODO!: Create LoginRequest dto
