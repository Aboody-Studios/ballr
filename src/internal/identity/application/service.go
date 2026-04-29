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
func hashPass(password string) (string, error) {
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPass), nil
}

func (s *Service) LoginWithGoogle(ctx context.Context, googleToken *oauth2.Token) (string, error) {
	googleUser, err := s.OauthProvider.FetchUserInfo(ctx, googleToken)
	if err != nil {
		return "", err
	}

	if !googleUser.VerifiedEmail {
		return "", fmt.Errorf("Unverified email: %d", http.StatusUnauthorized)
	}

	_, findErr := s.UserRepo.FindByEmail(ctx, googleUser.Email)

	if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
		return "", findErr
	}

	if findErr != nil && errors.Is(findErr, gorm.ErrRecordNotFound) {
		domainUser := &domain.User{
			Email:    googleUser.Email,
			FullName: googleUser.Name,
		}
		if err := s.UserRepo.Create(ctx, domainUser); err != nil {
			return "", err
		}
	}

	JWTToken, genErr := s.GenerateToken(googleUser.Email)

	if genErr != nil {
		return "", genErr
	}

	return JWTToken, nil
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

func (s *Service) FetchMutateSave(updateReq *UpdateUserRequest, ctx context.Context, id string) error {
	user, err := s.UserRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	user.UpdateProfile(updateReq.FullName, updateReq.Position, updateReq.Footedness, updateReq.Goals)
	if err := s.UserRepo.Update(ctx, user); err != nil {
		return err
	}

	return nil
}
