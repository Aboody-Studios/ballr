package domain

import (
	"context"
	"golang.org/x/oauth2"
)

type GoogleUserInfo struct {
	Email         string
	VerifiedEmail bool
	Name          string
	Profile       string
}

type OAuthProvider interface {
	FetchUserInfo(ctx context.Context, token *oauth2.Token) (*GoogleUserInfo, error)
}
