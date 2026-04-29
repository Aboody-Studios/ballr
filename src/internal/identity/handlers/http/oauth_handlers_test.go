package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/Aboody-Studios/ballr/src/internal/identity/application"
	"github.com/Aboody-Studios/ballr/src/internal/identity/domain"
	"github.com/Aboody-Studios/ballr/src/internal/identity/infrastructure"
	"github.com/labstack/echo/v5"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

// inMemoryUserRepo is a tiny in-memory implementation of domain.UserRepository
// so tests can run without a database.
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
		return nil, gorm.ErrRecordNotFound
	}
	return u, nil
}

func (r *inMemoryUserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	return nil, gorm.ErrRecordNotFound
}

func (r *inMemoryUserRepo) Update(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[user.Email] = user
	return nil
}

// fakeOAuthProvider implements domain.OAuthProvider for tests.
type fakeOAuthProvider struct {
	resp *domain.GoogleUserInfo
	err  error
}

func (f *fakeOAuthProvider) FetchUserInfo(ctx context.Context, token *oauth2.Token) (*domain.GoogleUserInfo, error) {
	return f.resp, f.err
}

func TestSignInWithGoogleHandler(t *testing.T) {
	// preserve original config
	orig := infrastructure.GoogleOauthConfig
	defer func() { infrastructure.GoogleOauthConfig = orig }()

	// set a deterministic auth URL for the test
	infrastructure.GoogleOauthConfig = &oauth2.Config{
		ClientID:     "cid",
		ClientSecret: "csecret",
		RedirectURL:  "http://localhost/auth/google/callback",
		Scopes:       []string{"email", "profile"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/auth/google", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := NewIdentityHandler(nil)
	if err := h.SignInWithGoogleHandler(c); err != nil {
		t.Fatalf("SignInWithGoogleHandler error: %v", err)
	}

	res := rec.Result()
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected status %d got %d", http.StatusSeeOther, res.StatusCode)
	}

	if loc := res.Header.Get("Location"); loc == "" {
		t.Fatalf("expected redirect Location header")
	}

	// ensure oauthstate cookie was set
	found := false
	for _, ck := range res.Cookies() {
		if ck.Name == "oauthstate" && ck.Value != "" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("oauthstate cookie not present")
	}
}

func TestGoogleCallbackHandler_Success(t *testing.T) {
	// Mock token exchange endpoint
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"mockaccesstoken","token_type":"Bearer","expires_in":3600}`))
	}))
	defer ts.Close()

	// preserve original config
	orig := infrastructure.GoogleOauthConfig
	defer func() { infrastructure.GoogleOauthConfig = orig }()

	infrastructure.GoogleOauthConfig = &oauth2.Config{
		ClientID:     "cid",
		ClientSecret: "csecret",
		RedirectURL:  "http://localhost/auth/google/callback",
		Scopes:       []string{"email", "profile"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  ts.URL + "/auth",
			TokenURL: ts.URL + "/token",
		},
	}

	// JWT secret for token generation
	os.Setenv("JWT_SECRET", "test-secret")
	defer os.Unsetenv("JWT_SECRET")

	fakeProv := &fakeOAuthProvider{resp: &domain.GoogleUserInfo{
		Email:         "alice@example.com",
		VerifiedEmail: true,
		Name:          "Alice",
		Profile:       "http://example.com/avatar.png",
	}}

	repo := newInMemoryUserRepo()
	svc := application.NewService(repo, fakeProv)
	h := NewIdentityHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=abc&code=123", nil)
	req.AddCookie(&http.Cookie{Name: "oauthstate", Value: "abc"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.GoogleCallbackHandler(c); err != nil {
		t.Fatalf("GoogleCallbackHandler error: %v", err)
	}

	res := rec.Result()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d got %d", http.StatusOK, res.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body["token"] == "" {
		t.Fatalf("expected token in response body")
	}
}
