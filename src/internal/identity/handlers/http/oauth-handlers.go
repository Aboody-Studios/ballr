package http

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/identity/infrastructure"
	"github.com/labstack/echo/v5"
)

// Activated when user clicks "sign in with google"
func (h *IdentityHandler) SignInWithGoogleHandler(echoCtx *echo.Context) error {
	state, stateErr := generateState()
	if stateErr != nil {
		return echoCtx.JSON(http.StatusInternalServerError, map[string]string{"error": "state generation error"})
	}
	url := infrastructure.GoogleOauthConfig.AuthCodeURL(state)

	echoCtx.SetCookie(&http.Cookie{
		Name:     "oauthstate",
		Value:    state,
		Expires:  time.Now().Add(15 * time.Minute),
		HttpOnly: true,
		Secure:   false, //TODO!: Set to true in production
	})

	if err := echoCtx.Redirect(http.StatusSeeOther, url); err != nil {
		return err
	}
	return nil
}

// Activated when user clicks allow and is used to catch "state" and "code" url params
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
	googleToken, exchangeErr := infrastructure.GoogleOauthConfig.Exchange(ctx, codeQueryParam)
	if exchangeErr != nil {
		return echoCtx.JSON(http.StatusBadRequest, map[string]string{"error": "Google access token exchange error"})
	}

	JWTToken, googleFetchErr := h.authService.LoginWithGoogle(ctx, googleToken)
	if googleFetchErr != nil {
		return googleFetchErr
	}
	return echoCtx.JSON(http.StatusOK, map[string]string{"token": JWTToken})
}

func generateState() (string, error) {
	stateByteSlice := make([]byte, 16)
	_, err := rand.Read(stateByteSlice)
	if err != nil {
		return "", err
	}
	state := base64.URLEncoding.EncodeToString(stateByteSlice)

	return state, nil
}
