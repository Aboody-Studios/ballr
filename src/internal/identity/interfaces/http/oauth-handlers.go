package http

import (
	"net/http"
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/identity/infrastructure"
	"github.com/labstack/echo/v5"
	"golang.org/x/oauth2"
)

// Activated when user clicks "sign in with google"
func (h *IdentityHandler) SignInWithGoogleHandler(echoCtx *echo.Context) error {
	state := oauth2.GenerateVerifier()
	url := infrastructure.GoogleOauthConfig.AuthCodeURL(state)

	echoCtx.SetCookie(&http.Cookie{
		Name:     "oauthstate",
		Value:    state,
		Expires:  time.Now().Add(15 * time.Minute),
		HttpOnly: true,
		Secure:   false, //must set to true in production
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
	token, exchangeErr := infrastructure.GoogleOauthConfig.Exchange(ctx, codeQueryParam)
	if exchangeErr != nil {
		return echoCtx.JSON(http.StatusBadRequest, map[string]string{"error": "Google access token exchange error"})
	}

	JWTToken, googleFetchErr := h.authService.LoginWithGoogle(ctx, token)
	if googleFetchErr != nil {
		return googleFetchErr
	}
	return echoCtx.JSON(http.StatusOK, map[string]string{"token": JWTToken})
}
