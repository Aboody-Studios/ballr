package domain

import (
	"context"
	"time"
)

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

type RefreshTokenData struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	FamilyID  string `json:"family_id"`
	CreatedAt int64  `json:"created_at"`
}

type RefreshTokenStore interface {
	Store(ctx context.Context, rawToken string, data *RefreshTokenData, ttl time.Duration) error
	Get(ctx context.Context, rawToken string) (*RefreshTokenData, error)
	Delete(ctx context.Context, rawToken string) error
}
