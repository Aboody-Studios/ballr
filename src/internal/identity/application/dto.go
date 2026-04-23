package application

import (
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/identity/domain"
)

type SignupRequest struct {
	Email      string            `json:"email"`
	Password   string            `json:"password"`
	FullName   string            `json:"FullName"`
	BirthDate  time.Time         `json:"birthdate"`
	Position   domain.Position   `json:"position"`
	Footedness domain.Footedness `json:"footedness"`
	Goals      string            `json:"goals"`
}

//TODO!: Create LoginRequest dto
