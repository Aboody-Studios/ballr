package application

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/identity/domain"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

type Service struct {
	UserRepo      domain.UserRepository
	OauthProvider domain.OAuthProvider
}

func NewService(repo domain.UserRepository, oauthProv domain.OAuthProvider) *Service {
	return &Service{UserRepo: repo, OauthProvider: oauthProv}
}

// This function is here because it will be used in both concrete signup and login
func (s *Service) GenerateToken(email string) (string, error) {
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

// This function is here because it will be used in both concrete signup and login
func hashPass(password string) (string, error) {
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPass), nil
}

func (s *Service) LoginWithGoogle(ctx context.Context, token *oauth2.Token) (string, error) {
	socialUser, err := s.OauthProvider.Fetch(ctx, token)
	if err != nil {
		return "", err
	}

	if !socialUser.VerifiedEmail {
		return "", fmt.Errorf("Unverified email: %d", http.StatusUnauthorized)
	}

	_, findErr := s.UserRepo.FindByEmail(ctx, socialUser.Email)

	if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound){
		return "", findErr
	}

	if findErr != nil && errors.Is(findErr, gorm.ErrRecordNotFound) {
		domainUser := &domain.User{
			Email:    socialUser.Email,
			FullName: socialUser.Name,
		}
		saveErr := s.UserRepo.Save(ctx, domainUser)
		if saveErr != nil {
			return "", saveErr
		}
	}

	JWTToken, genErr := s.GenerateToken(socialUser.Email)

	if genErr != nil {
		return "", genErr
	}

	return JWTToken, nil
}
