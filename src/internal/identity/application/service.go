package application

import (
	"os"
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/identity/domain"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Service handles identity-related use cases.
type Service struct {
	userRepo domain.UserRepository
}

// NewService creates a new identity service.
func NewService(repo domain.UserRepository) *Service {
	return &Service{userRepo: repo}
}

// RegisterUser handles user registration with password hashing.
func (s *Service) RegisterUser(email, password string) (*domain.User, error) {
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		Email:        email,
		PasswordHash: string(hashedPass),
		CreatedAt:    time.Now(),
	}

	// TODO: Persist user via repository when database is configured

	return user, nil
}

// GenerateToken creates a JWT token for the user.
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

// ValidateCredentials checks if the provided password matches the stored hash.
func (s *Service) ValidateCredentials(user *domain.User, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
}
