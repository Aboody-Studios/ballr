package domain

import (
	"encoding/json"
	"fmt"
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

// Since I can only attach methods to types that I define in my own local package,
// creating TrainingDay is essential because it is used in UnmarshalJSON
type TrainingDay time.Weekday

var WeekDayTranslation = map[string]time.Weekday{
	"SUNDAY":    time.Sunday,
	"MONDAY":    time.Monday,
	"TUESDAY":   time.Tuesday,
	"WEDNESDAY": time.Wednesday,
	"THURSDAY":  time.Thursday,
	"FRIDAY":    time.Friday,
	"SATURDAY":  time.Saturday,
}

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
	TrainingDays  []TrainingDay
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

func (td *TrainingDay) UnmarshalJSON(data []byte) error {
	var cleanString string
	if err := json.Unmarshal(data, &cleanString); err != nil {
		return err
	}

	value, ok := WeekDayTranslation[cleanString]
	if !ok {
		return fmt.Errorf("Invalid training day")
	}

	*td = TrainingDay(value)

	return nil
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
