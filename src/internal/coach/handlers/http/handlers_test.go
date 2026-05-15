package http

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/coach/application"
	coachdomain "github.com/Aboody-Studios/ballr/src/internal/coach/domain"
	iddomain "github.com/Aboody-Studios/ballr/src/internal/identity/domain"
	"github.com/Aboody-Studios/ballr/src/pkg/validator"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

type mockLLM struct{}
func (m *mockLLM) GenerateResponse(_ context.Context, _ []coachdomain.Message, _ coachdomain.CoachContext) (string, error) {
	return "Great work!", nil
}
func (m *mockLLM) GenerateTrainingPlan(_ context.Context, _ coachdomain.UserProfile, _ []string) (*coachdomain.TrainingPlan, error) {
	return &coachdomain.TrainingPlan{Objective: "Improve passing"}, nil
}
func (m *mockLLM) GenerateDietPlan(_ context.Context, _ coachdomain.UserProfile, _ coachdomain.TrainingLoad) (*coachdomain.DietPlan, error) {
	return &coachdomain.DietPlan{Calories: 2500}, nil
}

type mockAnalysis struct{}
func (m *mockAnalysis) GetLatestAnalysis(_ context.Context, _ string) (*coachdomain.MatchInsight, error) {
	return &coachdomain.MatchInsight{MatchID: "m-1"}, nil
}
func (m *mockAnalysis) GetMatchHistory(_ context.Context, _ string, _ int) ([]*coachdomain.MatchInsight, error) {
	return nil, nil
}

type mockCoachUser struct{}
func (m *mockCoachUser) GetUserProfile(_ context.Context, _ string) (*coachdomain.UserProfile, error) {
	return &coachdomain.UserProfile{Age: 18, Position: "CM", Footedness: "Right", Goals: "improve"}, nil
}

type mockHistory struct{}
func (m *mockHistory) GetConversation(_ context.Context, _, _ string) (*coachdomain.Conversation, error) {
	return nil, coachdomain.ErrConversationNotFound
}
func (m *mockHistory) SaveConversation(_ context.Context, _ *coachdomain.Conversation) error {
	return nil
}
func (m *mockHistory) FindByUserID(_ context.Context, _ string, _, _ int) ([]*coachdomain.Conversation, error) {
	return []*coachdomain.Conversation{{ID: "c-1", UserID: "user-1", SessionID: "s-1"}}, nil
}

type mockCoachEvent struct{}
func (m *mockCoachEvent) PublishEvent(_ context.Context, _ string, _ string, _ map[string]interface{}) error {
	return nil
}

func setupCoachHandler() *CoachHandler {
	svc := application.NewService(&mockLLM{}, &mockAnalysis{}, &mockCoachUser{}, &mockHistory{})
	svc.SetEventPublisher(&mockCoachEvent{})
	return NewCoachHandler(svc)
}

func coachAuthCtx(method, path, body string) (*echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	e.Validator = validator.New()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &iddomain.JWTCustomClaims{
		Email: "test@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        "user-1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	signed, _ := token.SignedString([]byte("test-secret"))
	parsed, _ := jwt.ParseWithClaims(signed, &iddomain.JWTCustomClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte("test-secret"), nil
	})
	c.Set("user", parsed)
	return c, rec
}

func TestCoachChatHandler(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret")
	defer os.Unsetenv("JWT_SECRET")

	handler := setupCoachHandler()
	c, rec := coachAuthCtx("POST", "/secure/coach/chat", `{"message":"How was my match?","session_id":"s-1"}`)

	if err := handler.ChatHandler(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["response"] != "Great work!" {
		t.Errorf("expected 'Great work!', got '%v'", resp["response"])
	}
}

func TestCoachGeneratePlanHandler(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret")
	defer os.Unsetenv("JWT_SECRET")

	handler := setupCoachHandler()
	c, rec := coachAuthCtx("POST", "/secure/coach/plan/generate", `{"focus_areas":["passing"]}`)

	if err := handler.GeneratePlanHandler(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var plan coachdomain.TrainingPlan
	json.NewDecoder(rec.Body).Decode(&plan)
	if plan.Objective != "Improve passing" {
		t.Errorf("expected 'Improve passing', got '%s'", plan.Objective)
	}
}

func TestCoachGenerateDietHandler(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret")
	defer os.Unsetenv("JWT_SECRET")

	handler := setupCoachHandler()
	c, rec := coachAuthCtx("POST", "/secure/coach/diet/generate", `{}`)

	if err := handler.GenerateDietHandler(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var plan coachdomain.DietPlan
	json.NewDecoder(rec.Body).Decode(&plan)
	if plan.Calories != 2500 {
		t.Errorf("expected 2500, got %d", plan.Calories)
	}
}

func TestCoachGetHistoryHandler(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret")
	defer os.Unsetenv("JWT_SECRET")

	handler := setupCoachHandler()
	c, rec := coachAuthCtx("GET", "/secure/coach/history", "")

	if err := handler.GetHistoryHandler(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	convs, ok := resp["conversations"].([]interface{})
	if !ok {
		t.Fatal("expected conversations array")
	}
	if len(convs) != 1 {
		t.Errorf("expected 1 conversation, got %d", len(convs))
	}
}
