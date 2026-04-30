package http

import (
	"errors"
	"net/http"

	"github.com/Aboody-Studios/ballr/src/internal/identity/application"
	"github.com/Aboody-Studios/ballr/src/internal/shared/delivery"
	"github.com/labstack/echo/v5"
)

// IdentityHandler handles HTTP requests for the Identity bounded context.
type IdentityHandler struct {
	authService *application.Service
}

func NewIdentityHandler(authService *application.Service) *IdentityHandler {
	return &IdentityHandler{authService: authService}
}

func (h *IdentityHandler) SignUpHandler(context *echo.Context) error {
	httpReq := context.Request()
	ctx := httpReq.Context()
	var req application.UserDTO

	if err := context.Bind(&req); err != nil {
		return context.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid JSON format"})
	}

	if err := context.Validate(&req); err != nil {
		return context.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid data"})
	}

	user, registerErr := h.authService.RegisterUser(&req, ctx)
	if registerErr != nil {
		if errors.Is(registerErr, application.ErrEmailAlreadyExists) {
			return context.JSON(http.StatusConflict, map[string]string{"error": "Email already exists"})
		}
		return context.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	token, err := h.authService.GenerateToken(user.Email, user.ID)
	if err != nil {
		return context.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	return context.JSON(http.StatusCreated, map[string]string{"token": token})
}

func (h *IdentityHandler) UpdateDataHandler(echoCtx *echo.Context) error {
	httpReq := echoCtx.Request()
	ctx := httpReq.Context()
	var updateReq application.UpdateUserRequest

	if err := echoCtx.Bind(&updateReq); err != nil {
		return echoCtx.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid JSON format"})
	}

	claims, err := delivery.ExtractToken(echoCtx)
	if err != nil {
		return err
	}

	if err := h.authService.FetchMutateSave(&updateReq, ctx, claims.ID); err != nil {
		return echoCtx.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	return nil
}

// ExtractToken extracts and returns the full JWT claims from the context.
// Unused for now.
/*func ExtractTokenForFrontend(echoCtx *echo.Context) error {
	claims, err := ExtractToken(echoCtx)
	if err != nil {
		return err
	}
	return echoCtx.JSON(http.StatusOK, claims)
}*/

// ExtractEmailFromJWT extracts the email claim from the JWT token in the context.
// Used in rate limiter.
func ExtractEmailFromJWT(echoCtx *echo.Context) (string, error) {
	claims, err := delivery.ExtractToken(echoCtx)
	if err != nil {
		return "", err
	}
	return claims.Email, nil
}
