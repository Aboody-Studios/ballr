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
	"github.com/google/uuid"
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

	dbUser, findErr := s.UserRepo.FindByEmail(ctx, googleUser.Email)
	if findErr != nil {
		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return "", findErr
		}

		domainUser := &domain.User{
			ID:       uuid.NewString(),
			Email:    googleUser.Email,
			FullName: googleUser.Name,
		}
		if err := s.UserRepo.Create(ctx, domainUser); err != nil {
			return "", err
		}

		dbUser = domainUser
	}

	JWTToken, genErr := s.GenerateToken(googleUser.Email, dbUser.ID)
	if genErr != nil {
		return "", genErr
	}

	return JWTToken, nil
}

// This function is here because it will be used in both concrete signup and login
func (s *Service) GenerateToken(id, email string) (string, error) {
	//TODO!: Rotate secret key
	secretKey := os.Getenv("JWT_SECRET")
	var customClaims domain.JWTCustomClaims
	customClaims.Email = email
	customClaims.ID = id
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
