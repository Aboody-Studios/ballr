package infrastructure

import (
	"time"

	progressdomain "github.com/Aboody-Studios/ballr/src/internal/progress/domain"
	"github.com/Aboody-Studios/ballr/src/pkg/events"
)

type DeviceInfo struct {
	ID          string
	UserID      string
	DeviceToken string
}

type Progress struct {
	ID            string `gorm:"primaryKey"`
	UserID        string `gorm:"uniqueIndex;not null"`
	TotalPoints   int    `gorm:"default:0"`
	CurrentStreak int    `gorm:"default:0"`
	LastActive    time.Time
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
}

//TODO!: See if passing User struct is necessary or not
type Achievement struct {
	ID          string                         `gorm:"primaryKey"`
	ProgressID  string
	UserID      string                         `gorm:"uniqueIndex:idx_user_achievement;not null"`
	Type        progressdomain.AchievementType `gorm:"uniqueIndex:idx_user_achievement;not null"`
	UnlockedAt  time.Time                      `gorm:"autoCreateTime"`
	PointsValue int
	Badge       bool
}

type EventLog struct {
	UserID        string           `gorm:"index:idx_user_event;not null"`
	Type          events.EventType `gorm:"index:idx_user_event;not null"`
	PointsAwarded int
	ID            string         `gorm:"primaryKey"`
	Timestamp     time.Time      `gorm:"index"`
	Metadata      map[string]any `gorm:"type:jsonb;serializer:json"`
}
