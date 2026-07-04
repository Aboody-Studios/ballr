package infrastructure

import "time"

type ChatMessage struct {
	ID        string
	UserID    string
	Role      string
	Content   string
	CreatedAt time.Time
}
