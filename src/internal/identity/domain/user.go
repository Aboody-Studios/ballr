package domain

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Position represents the player's position on the field.
// This is a value object - immutable and identified by its value.
type Position string

// shoutout pes mobile
const (
	PositionGK Position = "GK" // Goalkeeper
	PositionCB Position = "CB" // Center Back
	PositionLB Position = "LB" // Left Back
	PositionRB Position = "RB" // Right Back
	PositionCM Position = "CM" // Center Midfielder
	PositionLW Position = "LW" // Left Winger
	PositionRW Position = "RW" // Right Winger
	PositionST Position = "ST" // Striker
)

// Footedness represents which foot the player prefers.
type Footedness string

const (
	FootednessLeft  Footedness = "Left"
	FootednessRight Footedness = "Right"
	FootednessBoth  Footedness = "Both"
)

// User is the aggregate root for the Identity bounded context.
// As an aggregate root, it maintains its own invariants and consistency boundaries.
// TODO!: Remove `gorm:"uniqueIndex"` from here as it violates clean architecture. (Used temporarily)
type User struct {
	ID           string
	Email        string `gorm:"uniqueIndex"`
	PasswordHash string
	FullName     string
	BirthDate    time.Time
	Position     Position
	Footedness   Footedness
	Goals        string
	CreatedAt    time.Time
}

// JWTCustomClaims represents the claims we embed in JWT tokens.
type JWTCustomClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// NewUser creates a new user with validation.
// Factory method ensures invariants are maintained during creation.
// This doesn't need validation as it will be called after using SignUpRequest (for example), which itself has validation.
func NewUser(id, email, passwordHash, fullName string, birthDate time.Time, position Position, footedness Footedness, goals string) *User {
	user := &User{
		ID:           id,
		Email:        email,
		PasswordHash: passwordHash,
		FullName:     fullName,
		BirthDate:    birthDate,
		Position:     position,
		Footedness:   footedness,
		Goals:        goals,
		CreatedAt:    time.Now(),
	}

	/*if err := user.Validate(); err != nil {
		return nil, err
	}*/

	return user
}

// commented this out as it is already being done by playgorund validator (I think)
/*func (u *User) Validate() error {
	if u.Email == "" {
		return ErrInvalidEmail
	}

	if u.PasswordHash == "" {
		return ErrMissingPassword
	}

	if u.BirthDate.After(time.Now()) {
		return ErrInvalidBirthDate
	}

	switch u.Position {
	case PositionGK, PositionCB, PositionLB, PositionRB,
		PositionCM, PositionLW, PositionRW, PositionST:
	default:
		return ErrInvalidPosition
	}

	switch u.Footedness {
	case FootednessLeft, FootednessRight, FootednessBoth:
	default:
		return ErrInvalidFootedness
	}

	return nil
}*/

// UpdateProfile updates the user's profile information.
// TODO!: Use playgorund validator on this. Probably by creating a struct.
func (u *User) UpdateProfile(fullName string, position Position, footedness Footedness, goals string) {
	u.FullName = fullName
	u.Position = position
	u.Footedness = footedness
	u.Goals = goals
}

// CalculateAge returns the user's current age based on birth date.
func (u *User) CalculateAge() int {
	now := time.Now()
	age := now.Year() - u.BirthDate.Year()

	if now.Month() < u.BirthDate.Month() ||
		(now.Month() == u.BirthDate.Month() && now.Day() < u.BirthDate.Day()) {
		age--
	}

	return age
}

// Domain errors for User entity validation failures.
var (
	ErrInvalidEmail      = &UserError{"invalid email address"}
	ErrMissingPassword   = &UserError{"password is required"}
	ErrInvalidBirthDate  = &UserError{"birth date must be in the past"}
	ErrInvalidPosition   = &UserError{"invalid playing position"}
	ErrInvalidFootedness = &UserError{"invalid footedness value"}
)

// UserError represents domain-specific errors for the User entity.
type UserError struct {
	Message string
}

func (e *UserError) Error() string {
	return e.Message
}
