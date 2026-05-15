package delivery

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Aboody-Studios/ballr/src/internal/identity/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

func TestExtractToken_Success(t *testing.T) {
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

	extracted, err := ExtractToken(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if extracted.Email != "test@example.com" {
		t.Errorf("expected test@example.com, got %s", extracted.Email)
	}
}

func TestExtractToken_NoToken(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_, err := ExtractToken(c)
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestExtractEmailFromJWT(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	claims := &domain.JWTCustomClaims{Email: "user@example.com"}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte("test-secret"))
	parsed, _ := jwt.ParseWithClaims(signed, &domain.JWTCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte("test-secret"), nil
	})
	c.Set("user", parsed)

	email, err := ExtractEmailFromJWT(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if email != "user@example.com" {
		t.Errorf("expected user@example.com, got %s", email)
	}
}
