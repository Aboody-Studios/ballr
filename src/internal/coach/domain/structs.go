package domain

import (
	"time"
)

type UserMessage struct {
	Text string `json:"message"`
}

type CoachMessage struct {
	Text string
}

type ChatMessage struct {
	ID        string
	UserID    string
	Role      string
	Content   string
	CreatedAt time.Time
}
