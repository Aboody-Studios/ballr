package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/Aboody-Studios/ballr/src/internal/identity/domain"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var GoogleOauthConfig = &oauth2.Config{
	ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
	ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
	RedirectURL:  "http://localhost:8080/auth/google/callback",
	Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
	Endpoint:     google.Endpoint, // Tells the package to use Google's specific login URLs
}

type GoogleOAuthUserInfoResponse struct {
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Profile       string `json:"picture"`
}

type GoogleOAuthAPI struct{}

func (g *GoogleOAuthAPI) FetchUserInfo(ctx context.Context, token *oauth2.Token) (*domain.GoogleUserInfo, error) {
	client := GoogleOauthConfig.Client(ctx, token)
	res, getErr := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if getErr != nil {
		return nil, getErr
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Token rejected by google:%d", res.StatusCode)
	}
	defer res.Body.Close()

	bodyBytes, byteReadErr := io.ReadAll(res.Body)
	if byteReadErr != nil {
		return nil, byteReadErr
	}

	var gResp GoogleOAuthUserInfoResponse
	if err := json.Unmarshal(bodyBytes, &gResp); err != nil {
		return nil, err
	}

	return &domain.GoogleUserInfo{
		Email:         gResp.Email,
		VerifiedEmail: gResp.VerifiedEmail,
		Name:          gResp.Name,
		Profile:       gResp.Profile,
	}, nil
}
