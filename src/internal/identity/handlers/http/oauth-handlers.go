package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"os"
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/identity/application"
	"github.com/Aboody-Studios/ballr/src/internal/identity/infrastructure"
	"github.com/labstack/echo/v5"
)

func (h *IdentityHandler) SignInWithGoogleHandler(echoCtx *echo.Context) error {
	state, stateErr := generateState()
	if stateErr != nil {
		return echoCtx.JSON(http.StatusInternalServerError, map[string]string{"error": "state generation error"})
	}

	config := infrastructure.GetGoogleOAuthConfig()
	googleOauthuUrl := config.AuthCodeURL(state)

	secure := os.Getenv("COOKIE_SECURE") == "true"
	echoCtx.SetCookie(&http.Cookie{
		Name:     "oauthstate",
		Value:    state,
		Expires:  time.Now().Add(15 * time.Minute),
		HttpOnly: true,
		Secure:   secure,
	})

	return echoCtx.Redirect(http.StatusSeeOther, googleOauthuUrl)
}

func (h *IdentityHandler) GoogleCallbackHandler(echoCtx *echo.Context) error {
	cookie, err := echoCtx.Cookie("oauthstate")
	if err != nil {
		return echoCtx.JSON(http.StatusBadRequest, map[string]string{"error": "OAuth state cookie missing"})
	}

	stateQueryParam := echoCtx.QueryParam("state")
	if stateQueryParam != cookie.Value {
		return echoCtx.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid OAuth state"})
	}

	codeQueryParam := echoCtx.QueryParam("code")
	ctx := echoCtx.Request().Context()

	config := infrastructure.GetGoogleOAuthConfig()
	googleToken, exchangeErr := config.Exchange(ctx, codeQueryParam)
	if exchangeErr != nil {
		return echoCtx.JSON(http.StatusBadRequest, map[string]string{"error": "Google access token exchange error"})
	}

	tokenPair, googleFetchErr := h.authService.LoginWithGoogle(ctx, googleToken)
	if googleFetchErr != nil {
		return echoCtx.JSON(http.StatusInternalServerError, map[string]string{"error": "Google login failed"})
	}

	return echoCtx.JSON(http.StatusOK, application.TokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	})
}

func generateState() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
