package application

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/identity/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

var (
	ErrEmailAlreadyExists = errors.New("Email already exists")
	ErrRefreshExpired     = errors.New("Refresh token expired or invalid")
	ErrProfileIncomplete  = errors.New("Profile not fully set up")
)

type Service struct {
	UserRepo        domain.UserRepository
	OauthProvider   domain.OAuthProvider
	RefreshStore    domain.RefreshTokenStore
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

func NewService(repo domain.UserRepository, oauthProv domain.OAuthProvider, refreshStore domain.RefreshTokenStore) *Service {
	return &Service{
		UserRepo:        repo,
		OauthProvider:   oauthProv,
		RefreshStore:    refreshStore,
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 15 * 24 * time.Hour,
	}
}

func (s *Service) LoginWithGoogle(ctx context.Context, googleToken *oauth2.Token) (*domain.TokenPair, error) {
	googleUser, err := s.OauthProvider.FetchUserInfo(ctx, googleToken)
	if err != nil {
		return nil, err
	}

	if !googleUser.VerifiedEmail {
		return nil, fmt.Errorf("unverified email")
	}

	dbUser, findErr := s.UserRepo.FindByEmail(ctx, googleUser.Email)
	if findErr != nil {
		if !errors.Is(findErr, domain.ErrUserNotFound) {
			return nil, findErr
		}

		dbUser = domain.NewUser(
			uuid.NewString(),
			googleUser.Email,
			"google",
			googleUser.Profile,
			googleUser.Name,
			time.Time{},
			"",
			"",
		)
		if err := s.UserRepo.Create(ctx, dbUser); err != nil {
			return nil, err
		}
	}

	return s.generateTokenPair(ctx, dbUser.ID, dbUser.Email)
}

func (s *Service) RefreshAccessToken(ctx context.Context, rawRefreshToken string) (*domain.TokenPair, error) {
	stored, err := s.RefreshStore.Get(ctx, rawRefreshToken)
	if err != nil {
		return nil, ErrRefreshExpired
	}

	s.RefreshStore.Delete(ctx, rawRefreshToken)

	user, err := s.UserRepo.FindByID(ctx, stored.UserID)
	if err != nil {
		return nil, err
	}

	return s.generateTokenPair(ctx, user.ID, user.Email)
}

func (s *Service) GetProfile(ctx context.Context, userID string) (*ProfileResponse, error) {
	user, err := s.UserRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &ProfileResponse{
		ID:            user.ID,
		Email:         user.Email,
		FullName:      user.FullName,
		AvatarURL:     user.AvatarURL,
		Footedness:    user.Footedness,
		Goals:         user.Goals,
		BirthDate:     user.BirthDate,
		OAuthProvider: user.OAuthProvider,
		CreatedAt:     user.CreatedAt,
	}, nil
}

func (s *Service) CompleteProfile(ctx context.Context, userID string, req *OnboardingRequest) error {
	user, err := s.UserRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	user.FullName = req.FullName
	user.BirthDate = req.BirthDate
	user.Footedness = req.Footedness
	user.Goals = req.Goals
	user.TrainingDays = req.TrainingDays

	return s.UserRepo.Update(ctx, user)
}

func (s *Service) generateTokenPair(ctx context.Context, userID, email string) (*domain.TokenPair, error) {
	secretKey := os.Getenv("JWT_SECRET")

	now := time.Now()
	accessClaims := domain.JWTCustomClaims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.AccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	signedAccess, err := accessToken.SignedString([]byte(secretKey))
	if err != nil {
		return nil, err
	}

	rawRefresh := make([]byte, 32)
	if _, err := rand.Read(rawRefresh); err != nil {
		return nil, err
	}
	refreshToken := base64.URLEncoding.EncodeToString(rawRefresh)

	refreshData := &domain.RefreshTokenData{
		UserID:    userID,
		Email:     email,
		FamilyID:  uuid.NewString(),
		CreatedAt: now.Unix(),
	}

	if err := s.RefreshStore.Store(ctx, refreshToken, refreshData, s.RefreshTokenTTL); err != nil {
		return nil, err
	}

	return &domain.TokenPair{
		AccessToken:  signedAccess,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.AccessTokenTTL.Seconds()),
	}, nil
}
