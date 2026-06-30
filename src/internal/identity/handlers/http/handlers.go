package handlers

import (
	"errors"
	"net/http"

	"github.com/Aboody-Studios/ballr/src/internal/identity/application"
	"github.com/Aboody-Studios/ballr/src/internal/shared/delivery"
	"github.com/labstack/echo/v5"
)

type IdentityHandler struct {
	authService *application.IdentityService
}

func NewIdentityHandler(authService *application.IdentityService) *IdentityHandler {
	return &IdentityHandler{authService: authService}
}

func (h *IdentityHandler) GetProfileHandler(echoCtx *echo.Context) error {
	claims, err := delivery.ExtractToken(echoCtx)
	if err != nil {
		return echoCtx.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	profile, err := h.authService.GetProfile(echoCtx.Request().Context(), claims.ID)
	if err != nil {
		return echoCtx.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}

	return echoCtx.JSON(http.StatusOK, profile)
}

func (h *IdentityHandler) CompleteProfileHandler(echoCtx *echo.Context) error {
	claims, err := delivery.ExtractToken(echoCtx)
	if err != nil {
		return echoCtx.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	var req application.OnboardingRequest

	if err := echoCtx.Bind(&req); err != nil {
		return echoCtx.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid JSON format"})
	}

	if err := echoCtx.Validate(&req); err != nil {
		return echoCtx.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid data"})
	}

	if err := h.authService.CompleteProfile(echoCtx.Request().Context(), claims.ID, &req); err != nil {
		return echoCtx.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update profile"})
	}

	return echoCtx.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *IdentityHandler) RefreshTokenHandler(echoCtx *echo.Context) error {
	var req application.RefreshRequest
	if err := echoCtx.Bind(&req); err != nil {
		return echoCtx.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid JSON format"})
	}

	if err := echoCtx.Validate(&req); err != nil {
		return echoCtx.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid refresh token"})
	}

	tokenPair, err := h.authService.RefreshAccessToken(echoCtx.Request().Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, application.ErrRefreshExpired) {
			return echoCtx.JSON(http.StatusUnauthorized, map[string]string{"error": "Refresh token expired"})
		}
		return echoCtx.JSON(http.StatusInternalServerError, map[string]string{"error": "Token refresh failed"})
	}

	return echoCtx.JSON(http.StatusOK, application.TokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	})
}
