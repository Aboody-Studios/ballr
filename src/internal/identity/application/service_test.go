package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/identity/domain"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

type inMemoryUserRepo struct {
	mu    sync.Mutex
	users map[string]*domain.User
}

func newInMemoryUserRepo() *inMemoryUserRepo {
	return &inMemoryUserRepo{users: make(map[string]*domain.User)}
}

func (r *inMemoryUserRepo) Create(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[user.Email] = user
	return nil
}

func (r *inMemoryUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[email]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (r *inMemoryUserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (r *inMemoryUserRepo) Update(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[user.Email] = user
	return nil
}

type inMemoryRefreshStore struct {
	mu   sync.Mutex
	data map[string]*domain.RefreshTokenData
}

func newInMemoryRefreshStore() *inMemoryRefreshStore {
	return &inMemoryRefreshStore{data: make(map[string]*domain.RefreshTokenData)}
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func (s *inMemoryRefreshStore) Store(ctx context.Context, rawToken string, data *domain.RefreshTokenData, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[hashToken(rawToken)] = data
	return nil
}

func (s *inMemoryRefreshStore) Get(ctx context.Context, rawToken string) (*domain.RefreshTokenData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.data[hashToken(rawToken)]
	if !ok {
		return nil, errors.New("not found")
	}
	return d, nil
}

func (s *inMemoryRefreshStore) Delete(ctx context.Context, rawToken string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, hashToken(rawToken))
	return nil
}

type mockOAuthProvider struct {
	userInfo *domain.GoogleUserInfo
	err      error
}

func (m *mockOAuthProvider) FetchUserInfo(ctx context.Context, token *oauth2.Token) (*domain.GoogleUserInfo, error) {
	return m.userInfo, m.err
}

func setupService(t *testing.T) (*Service, *inMemoryUserRepo, *inMemoryRefreshStore) {
	t.Helper()
	os.Setenv("JWT_SECRET", "test-secret-key-for-testing-only")
	t.Cleanup(func() { os.Unsetenv("JWT_SECRET") })

	repo := newInMemoryUserRepo()
	refresh := newInMemoryRefreshStore()
	svc := NewService(repo, &mockOAuthProvider{}, refresh)
	svc.AccessTokenTTL = time.Hour
	svc.RefreshTokenTTL = time.Hour
	return svc, repo, refresh
}

func TestLoginWithGoogle_CreatesNewUser(t *testing.T) {
	googleUser := &domain.GoogleUserInfo{
		Email:         "new@example.com",
		VerifiedEmail: true,
		Name:          "New User",
		Profile:       "https://example.com/avatar.png",
	}
	svc, repo, _ := setupService(t)
	svc.OauthProvider = &mockOAuthProvider{userInfo: googleUser}

	tokenPair, err := svc.LoginWithGoogle(context.Background(), &oauth2.Token{})
	if err != nil {
		t.Fatalf("LoginWithGoogle failed: %v", err)
	}

	if tokenPair.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if tokenPair.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
	if tokenPair.ExpiresIn == 0 {
		t.Error("expected non-zero expires_in")
	}

	user, _ := repo.FindByEmail(context.Background(), "new@example.com")
	if user == nil {
		t.Fatal("expected user to be created")
	}
	if user.OAuthProvider != "google" {
		t.Errorf("expected oauth_provider google, got %s", user.OAuthProvider)
	}
	if user.AvatarURL != "https://example.com/avatar.png" {
		t.Errorf("expected avatar URL, got %s", user.AvatarURL)
	}
}

func TestLoginWithGoogle_ExistingUser(t *testing.T) {
	svc, repo, _ := setupService(t)

	existing := domain.NewUser(uuid.NewString(), "existing@example.com", "google", "", "Existing", time.Time{}, "", "", "")
	repo.Create(context.Background(), existing)

	svc.OauthProvider = &mockOAuthProvider{userInfo: &domain.GoogleUserInfo{
		Email:         "existing@example.com",
		VerifiedEmail: true,
		Name:          "Existing",
		Profile:       "",
	}}

	tokenPair, err := svc.LoginWithGoogle(context.Background(), &oauth2.Token{})
	if err != nil {
		t.Fatalf("LoginWithGoogle failed: %v", err)
	}

	if tokenPair.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
}

func TestLoginWithGoogle_UnverifiedEmail(t *testing.T) {
	svc, _, _ := setupService(t)
	svc.OauthProvider = &mockOAuthProvider{userInfo: &domain.GoogleUserInfo{
		Email:         "unverified@example.com",
		VerifiedEmail: false,
	}}

	_, err := svc.LoginWithGoogle(context.Background(), &oauth2.Token{})
	if err == nil {
		t.Fatal("expected error for unverified email")
	}
}

func TestRefreshAccessToken_Valid(t *testing.T) {
	svc, _, refresh := setupService(t)
	svc.OauthProvider = &mockOAuthProvider{userInfo: &domain.GoogleUserInfo{
		Email:         "test@example.com",
		VerifiedEmail: true,
		Name:          "Test",
	}}

	firstPair, err := svc.LoginWithGoogle(context.Background(), &oauth2.Token{})
	if err != nil {
		t.Fatalf("LoginWithGoogle failed: %v", err)
	}

	secondPair, err := svc.RefreshAccessToken(context.Background(), firstPair.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshAccessToken failed: %v", err)
	}

	if secondPair.AccessToken == "" {
		t.Error("expected non-empty access token after refresh")
	}
	if secondPair.RefreshToken == "" {
		t.Error("expected non-empty refresh token after refresh")
	}
	if secondPair.RefreshToken == firstPair.RefreshToken {
		t.Error("expected refresh token to be rotated")
	}

	_, err = refresh.Get(context.Background(), firstPair.RefreshToken)
	if err == nil {
		t.Error("expected old refresh token to be deleted after rotation")
	}
}

func TestRefreshAccessToken_Expired(t *testing.T) {
	svc, _, _ := setupService(t)

	_, err := svc.RefreshAccessToken(context.Background(), "nonexistent-token")
	if !errors.Is(err, ErrRefreshExpired) {
		t.Errorf("expected ErrRefreshExpired, got %v", err)
	}
}

func TestGetProfile_ExistingUser(t *testing.T) {
	svc, repo, _ := setupService(t)
	birthDate := time.Date(2000, 5, 10, 0, 0, 0, 0, time.UTC)
	user := domain.NewUser(uuid.NewString(), "profile@example.com", "google", "https://avatar.url", "Profile User", birthDate, domain.PositionCM, domain.FootednessRight, "get better")
	repo.Create(context.Background(), user)

	profile, err := svc.GetProfile(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("GetProfile failed: %v", err)
	}

	if profile.Email != "profile@example.com" {
		t.Errorf("expected email profile@example.com, got %s", profile.Email)
	}
	if profile.FullName != "Profile User" {
		t.Errorf("expected name Profile User, got %s", profile.FullName)
	}
	if profile.Position != domain.PositionCM {
		t.Errorf("expected position CM, got %s", profile.Position)
	}
	if profile.Footedness != domain.FootednessRight {
		t.Errorf("expected footedness Right, got %s", profile.Footedness)
	}
	if profile.AvatarURL != "https://avatar.url" {
		t.Errorf("expected avatar URL, got %s", profile.AvatarURL)
	}
}

func TestGetProfile_NonExistentUser(t *testing.T) {
	svc, _, _ := setupService(t)

	_, err := svc.GetProfile(context.Background(), "nonexistent-id")
	if err == nil {
		t.Fatal("expected error for non-existent user")
	}
}

func TestCompleteProfile(t *testing.T) {
	svc, repo, _ := setupService(t)
	user := domain.NewUser(uuid.NewString(), "complete@example.com", "google", "", "Initial", time.Time{}, "", "", "")
	repo.Create(context.Background(), user)

	birthDate := time.Date(1999, 3, 20, 0, 0, 0, 0, time.UTC)
	req := &OnboardingRequest{
		FullName:   "Updated Name",
		BirthDate:  birthDate,
		Position:   domain.PositionST,
		Footedness: domain.FootednessLeft,
		Goals:      "score more",
	}

	err := svc.CompleteProfile(context.Background(), user.ID, req)
	if err != nil {
		t.Fatalf("CompleteProfile failed: %v", err)
	}

	updated, _ := repo.FindByID(context.Background(), user.ID)
	if updated.FullName != "Updated Name" {
		t.Errorf("expected name Updated Name, got %s", updated.FullName)
	}
	if updated.Position != domain.PositionST {
		t.Errorf("expected position ST, got %s", updated.Position)
	}
	if updated.Goals != "score more" {
		t.Errorf("expected goals 'score more', got %s", updated.Goals)
	}
}

func TestGenerateTokenPair_ProducesValidJWT(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret")
	defer os.Unsetenv("JWT_SECRET")

	svc, repo, refresh := setupService(t)
	user := domain.NewUser(uuid.NewString(), "jwt@example.com", "google", "", "JWT User", time.Time{}, "", "", "")
	repo.Create(context.Background(), user)
	_ = repo

	pair, err := svc.generateTokenPair(context.Background(), user.ID, user.Email)
	if err != nil {
		t.Fatalf("generateTokenPair failed: %v", err)
	}

	if pair.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if pair.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}

	_, err = refresh.Get(context.Background(), pair.RefreshToken)
	if err != nil {
		t.Error("expected refresh token to be stored in redis")
	}
}
