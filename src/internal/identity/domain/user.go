package domain

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Position string

// TODO!: Add position to analysis domain and remove it from here
const (
	PositionGK Position = "GK"
	PositionCB Position = "CB"
	PositionLB Position = "LB"
	PositionRB Position = "RB"
	PositionCM Position = "CM"
	PositionLW Position = "LW"
	PositionRW Position = "RW"
	PositionST Position = "ST"
)

type Footedness string

const (
	FootednessLeft  Footedness = "Left"
	FootednessRight Footedness = "Right"
	FootednessBoth  Footedness = "Both"
)

// TODO!: Remove gorm
type User struct {
	ID            string
	Email         string `gorm:"uniqueIndex"`
	AvatarURL     string
	OAuthProvider string
	FullName      string
	BirthDate     time.Time
	Position      Position
	Footedness    Footedness
	Goals         string
	CreatedAt     time.Time
	TrainingDays  []time.Weekday
}

type JWTCustomClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func NewUser(id, email, oauthProvider, avatarURL, fullName string, birthDate time.Time, position Position, footedness Footedness, goals string) *User {
	user := &User{
		ID:            id,
		Email:         email,
		OAuthProvider: oauthProvider,
		AvatarURL:     avatarURL,
		FullName:      fullName,
		BirthDate:     birthDate,
		Position:      position,
		Footedness:    footedness,
		Goals:         goals,
		CreatedAt:     time.Now(),
	}
	return user
}

func (u *User) UpdateProfile(fullName string, position Position, footedness Footedness, goals string) {
	u.FullName = fullName
	u.Position = position
	u.Footedness = footedness
	u.Goals = goals
}

func (u *User) CalculateAge() int {
	now := time.Now()
	age := now.Year() - u.BirthDate.Year()
	if now.Month() < u.BirthDate.Month() ||
		(now.Month() == u.BirthDate.Month() && now.Day() < u.BirthDate.Day()) {
		age--
	}
	return age
}

var (
	ErrInvalidEmail      = &UserError{"invalid email address"}
	ErrInvalidBirthDate  = &UserError{"birth date must be in the past"}
	ErrInvalidPosition   = &UserError{"invalid playing position"}
	ErrInvalidFootedness = &UserError{"invalid footedness value"}
)

type UserError struct {
	Message string
}

func (e *UserError) Error() string {
	return e.Message
}
