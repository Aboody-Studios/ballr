package infrastructure

import (
	"time"
)

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

type Footedness string

const (
	FootednessLeft  Footedness = "Left"
	FootednessRight Footedness = "Right"
	FootednessBoth  Footedness = "Both"
)

type User struct {
	ID            string
	Email         string `gorm:"uniqueIndex" fake:"{username}@gmail.com"`
	Username      string
	AvatarURL     string
	OAuthProvider string
	FullName      string
	BirthDate     time.Time
	Footedness    Footedness
	CreatedAt     time.Time
	TrainingDays  []TrainingDay `gorm:"type:jsonb;serializer:json"`
	Timezone      string
}
