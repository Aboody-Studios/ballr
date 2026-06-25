package infrastructure

import (
	"context"

	coachdomain "github.com/Aboody-Studios/ballr/src/internal/coach/domain"
	identitydomain "github.com/Aboody-Studios/ballr/src/internal/identity/domain"
)

type CoachUserBridge struct {
	userRepo identitydomain.UserRepository
}

func NewCoachUserBridge(repo identitydomain.UserRepository) *CoachUserBridge {
	return &CoachUserBridge{userRepo: repo}
}

func (b *CoachUserBridge) GetUserProfile(ctx context.Context, userID string) (*coachdomain.UserProfile, error) {
	user, err := b.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &coachdomain.UserProfile{
		Age:        user.CalculateAge(),
		Footedness: string(user.Footedness),
		Goals:      user.Goals,
	}, nil
}
