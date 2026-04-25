package domain

import (
	"context"
	"golang.org/x/oauth2"
)

type SocialUserInfo struct {
	Email         string
	VerifiedEmail bool
	Name          string
	Profile       string
}

type OAuthProvider interface {
	Fetch(ctx context.Context, token *oauth2.Token) (*SocialUserInfo, error)
}
