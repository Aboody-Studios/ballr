package infrastructure

import (
	"os"
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/identity/domain"
	"github.com/golang-jwt/jwt/v5"
)

func GenerateToken(email string) (string, error) {
	secretKey := os.Getenv("JWT_SECRET")
	var customClaims domain.JWTCustomClaims
	customClaims.Email = email
	customClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Hour * 24))
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, customClaims)
	signedToken, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	return signedToken, nil
}
