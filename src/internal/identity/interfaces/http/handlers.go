package http

import (
	"errors"
	"net/http"

	"github.com/Aboody-Studios/ballr/src/internal/identity/application"
	"github.com/Aboody-Studios/ballr/src/internal/identity/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

// IdentityHandler handles HTTP requests for the Identity bounded context.
type IdentityHandler struct {
	authService *application.Service
}

// NewIdentityHandler creates a new identity handler with the required service.
func NewIdentityHandler(authService *application.Service) *IdentityHandler {
	return &IdentityHandler{authService: authService}
}

// SignUpHandler handles user registration.
func (h *IdentityHandler) SignUpHandler(context *echo.Context) error {
	var user domain.User
	if err := context.Bind(&user); err != nil {
		return context.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid JSON format"})
	}

	if err := context.Validate(&user); err != nil {
		return context.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid data"})
	}

	// TODO!: Persist user when database layer is ready

	token, err := h.authService.GenerateToken(user.Email)
	if err != nil {
		return context.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	return context.JSON(http.StatusCreated, map[string]string{"token": token})
}

// ExtractToken extracts and returns the full JWT claims from the context.
func ExtractToken(context *echo.Context) error {
	token, err := echo.ContextGet[*jwt.Token](context, "user")
	if err != nil {
		return echo.ErrUnauthorized.Wrap(err)
	}
	claims, ok := token.Claims.(*domain.JWTCustomClaims)
	if !ok {
		return errors.New("failed to cast claims as JWTCustomClaims")
	}
	return context.JSON(http.StatusOK, claims)
}

// ExtractEmailFromJWT extracts the email claim from the JWT token in the context.
// Used in rate limiter.
func ExtractEmailFromJWT(c *echo.Context) (string, error) {
	token, err := echo.ContextGet[*jwt.Token](c, "user")
	if err != nil {
		return "", echo.ErrUnauthorized.Wrap(err)
	}
	claims, ok := token.Claims.(*domain.JWTCustomClaims)
	if !ok {
		return "", errors.New("failed to cast claims as JWTCustomClaims")
	}
	return claims.Email, nil
}
