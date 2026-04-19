package services

import (
	"github.com/Aboody-Studios/ballr/backend/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"os"
	"time"
)

func GenerateToken(email string) (string, error) {
	secretKey := os.Getenv("JWT_SECRET")
	var customClaims models.JWTCustomClaims
	customClaims.Email = email
	customClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Hour * 24))
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, customClaims)
	// The any parameter type is due to different signing keys requiring different types, which is why a generic type like any is required.
	// However, when passing the key, it should be in the specified type required by the algorithm.
	// RSA requires *rsa.PrivateKey, HS256 requires []byte, and so on.
	signedToken, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	return signedToken, nil
}
