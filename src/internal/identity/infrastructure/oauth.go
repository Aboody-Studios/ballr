package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/Aboody-Studios/ballr/src/internal/identity/domain"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleOauthConfig is available for test override.
// In production it is lazy-initialised by getGoogleOAuthConfig.
var GoogleOauthConfig *oauth2.Config

var googleOauthOnce sync.Once

func GetGoogleOAuthConfig() *oauth2.Config {
	if GoogleOauthConfig != nil {
		return GoogleOauthConfig
	}
	googleOauthOnce.Do(func() {
		redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")
		if redirectURL == "" {
			redirectURL = "http://localhost:8080/auth/google/callback"
		}
		GoogleOauthConfig = &oauth2.Config{
			ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			RedirectURL:  redirectURL,
			Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
			Endpoint:     google.Endpoint,
		}
	})
	return GoogleOauthConfig
}

type GoogleOAuthUserInfoResponse struct {
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Profile       string `json:"picture"`
}

type GoogleOAuthAPI struct{}

func (g *GoogleOAuthAPI) FetchUserInfo(ctx context.Context, token *oauth2.Token) (*domain.GoogleUserInfo, error) {
	config := GetGoogleOAuthConfig()
	client := config.Client(ctx, token)
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
