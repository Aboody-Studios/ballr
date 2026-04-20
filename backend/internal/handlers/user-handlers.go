package handlers

import (
	"errors"
	"github.com/Aboody-Studios/ballr/backend/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
	"net/http"
)

func ExtractToken(context *echo.Context) error {
	token, err := echo.ContextGet[*jwt.Token](context, "user")
	if err != nil {
		return echo.ErrUnauthorized.Wrap(err)
	}
	claims, ok := token.Claims.(*models.JWTCustomClaims)
	if !ok {
		return errors.New("failed to cast claims as JWTCustomClaims")
	}
	return context.JSON(http.StatusOK, claims)

}
