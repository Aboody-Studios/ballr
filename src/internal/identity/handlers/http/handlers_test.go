package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/identity/application"
	"github.com/Aboody-Studios/ballr/src/internal/identity/domain"
	"github.com/Aboody-Studios/ballr/src/internal/shared/delivery"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
	"golang.org/x/oauth2"
)

type testUserRepo struct {
	mu    sync.Mutex
	users map[string]*domain.User
}

func newTestUserRepo() *testUserRepo {
	return &testUserRepo{users: make(map[string]*domain.User)}
}

func (r *testUserRepo) Create(_ context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[user.Email] = user
	return nil
}

func (r *testUserRepo) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[email]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (r *testUserRepo) FindByID(_ context.Context, id string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (r *testUserRepo) Update(_ context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[user.Email] = user
	return nil
}

type testRefreshStore struct {
	mu    sync.Mutex
	store map[string]*domain.RefreshTokenData
}

func newTestRefreshStore() *testRefreshStore {
	return &testRefreshStore{store: make(map[string]*domain.RefreshTokenData)}
}

func testHashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func (s *testRefreshStore) Store(_ context.Context, rawToken string, data *domain.RefreshTokenData, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[testHashToken(rawToken)] = data
	return nil
}

func (s *testRefreshStore) Get(_ context.Context, rawToken string) (*domain.RefreshTokenData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.store[testHashToken(rawToken)]
	if !ok {
		return nil, errors.New("not found")
	}
	return d, nil
}

func (s *testRefreshStore) Delete(_ context.Context, rawToken string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.store, testHashToken(rawToken))
	return nil
}

type testOAuthProv struct{}

func (f *testOAuthProv) FetchUserInfo(_ context.Context, _ *oauth2.Token) (*domain.GoogleUserInfo, error) {
	return &domain.GoogleUserInfo{
		Email: "test@example.com", VerifiedEmail: true, Name: "Test",
	}, nil
}

func setupTest(t *testing.T) (*IdentityHandler, *echo.Echo, *testUserRepo) {
	t.Helper()
	os.Setenv("JWT_SECRET", "test-secret")
	t.Cleanup(func() { os.Unsetenv("JWT_SECRET") })

	repo := newTestUserRepo()
	refresh := newTestRefreshStore()
	svc := application.NewService(repo, &testOAuthProv{}, refresh)
	svc.AccessTokenTTL = time.Hour
	svc.RefreshTokenTTL = time.Hour

	handler := NewIdentityHandler(svc)
	e := echo.New()

	return handler, e, repo
}

func authContext(t *testing.T, e *echo.Echo, method, path, userID, email string, body string) (*echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	claims := &domain.JWTCustomClaims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte("test-secret"))
	parsed, _ := jwt.ParseWithClaims(signed, &domain.JWTCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte("test-secret"), nil
	})
	c.Set("user", parsed)

	return c, rec
}

func TestGetProfileHandler_Success(t *testing.T) {
	handler, e, repo := setupTest(t)
	userID := "user-1"
	user := domain.NewUser(userID, "test@example.com", "google", "", "Test User", time.Time{}, domain.PositionCM, domain.FootednessRight, "improve")
	repo.Create(context.Background(), user)

	c, rec := authContext(t, e, http.MethodGet, "/secure/auth/me", userID, "test@example.com", "")
	if err := handler.GetProfileHandler(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	res := rec.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", res.StatusCode)
	}
}

func TestGetProfileHandler_NotFound(t *testing.T) {
	handler, e, _ := setupTest(t)
	c, rec := authContext(t, e, http.MethodGet, "/secure/auth/me", "nonexistent", "test@example.com", "")
	if err := handler.GetProfileHandler(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	res := rec.Result()
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", res.StatusCode)
	}
}

func TestDeliveryExtractToken_Success(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	claims := &domain.JWTCustomClaims{Email: "test@example.com"}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte("test-secret"))
	parsed, _ := jwt.ParseWithClaims(signed, &domain.JWTCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte("test-secret"), nil
	})
	c.Set("user", parsed)

	extracted, err := delivery.ExtractToken(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if extracted.Email != "test@example.com" {
		t.Errorf("expected test@example.com, got %s", extracted.Email)
	}
}

func TestDeliveryExtractToken_NoToken(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_, err := delivery.ExtractToken(c)
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}
