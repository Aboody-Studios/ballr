package handlers

import (
	"errors"
	"net/http"

	"github.com/Aboody-Studios/ballr/backend/internal/models"
	"github.com/Aboody-Studios/ballr/backend/internal/services"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

func SignUpHandler(context *echo.Context) error {
	var user models.User
	if err := context.Bind(&user); err != nil {
		return context.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid JSON format"})
	}

	if err := context.Validate(&user); err != nil {
		return context.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid data"})
	}

	/*persistErr := services.PersistToDatabase(&user)
	if persistErr != nil {
		return context.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}*/

	token, err := services.GenerateToken(user.Email)

	if err != nil {
		return context.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	return context.JSON(http.StatusCreated, map[string]string{"token": token})

}

// Not being used here yet. Used in rate limiter.
func ExtractEmailFromJWT(c *echo.Context) (string, error) {
	token, err := echo.ContextGet[*jwt.Token](c, "user")
	if err != nil {
		return "", echo.ErrUnauthorized.Wrap(err)
	}
	claims, ok := token.Claims.(*models.JWTCustomClaims)
	if !ok {
		return "", errors.New("failed to cast claims as JWTCustomClaims")
	}
	return claims.Email, nil
}
