package delivery

import (
	"errors"

	"github.com/Aboody-Studios/ballr/src/internal/identity/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

func ExtractToken(echoCtx *echo.Context) (*domain.JWTCustomClaims, error) {
	token, err := echo.ContextGet[*jwt.Token](echoCtx, "user")
	if err != nil {
		return nil, echo.ErrUnauthorized.Wrap(err)
	}
	if !token.Valid {
		return nil, echo.ErrUnauthorized
	}
	claims, ok := token.Claims.(*domain.JWTCustomClaims)
	if !ok {
		return nil, errors.New("failed to cast claims as JWTCustomClaims")
	}

	return claims, nil
}

func ExtractEmailFromJWT(echoCtx *echo.Context) (string, error) {
	claims, err := ExtractToken(echoCtx)
	if err != nil {
		return "", err
	}
	return claims.Email, nil
}
