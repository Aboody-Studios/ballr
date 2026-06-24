package application

import (
	"context"
	"sync"
	"testing"

	"github.com/Aboody-Studios/ballr/src/internal/coach/domain"
	"github.com/Aboody-Studios/ballr/src/pkg/events"
)

type mockLLMProvider struct {
	response     string
	trainingPlan *domain.TrainingPlan
	dietPlan     *domain.DietPlan
	err          error
}

func (m *mockLLMProvider) GenerateResponse(_ context.Context, _ []domain.Message, _ domain.CoachContext) (string, error) {
	return m.response, m.err
}

func (m *mockLLMProvider) GenerateTrainingPlan(_ context.Context, _ domain.UserProfile, _ []string) (*domain.TrainingPlan, error) {
	return m.trainingPlan, m.err
}

func (m *mockLLMProvider) GenerateDietPlan(_ context.Context, _ domain.UserProfile, _ domain.TrainingLoad) (*domain.DietPlan, error) {
	return m.dietPlan, m.err
}

type mockAnalysisService struct {
	insight *domain.MatchInsight
	history []*domain.MatchInsight
	err     error
}

func (m *mockAnalysisService) GetLatestAnalysis(_ context.Context, _ string) (*domain.MatchInsight, error) {
	return m.insight, m.err
}

func (m *mockAnalysisService) GetMatchHistory(_ context.Context, _ string, _ int) ([]*domain.MatchInsight, error) {
	return m.history, m.err
}

type mockUserService struct {
	profile *domain.UserProfile
	err     error
}

func (m *mockUserService) GetUserProfile(_ context.Context, _ string) (*domain.UserProfile, error) {
	return m.profile, m.err
}

type mockConvRepo struct {
	mu            sync.Mutex
	conversations map[string]*domain.Conversation
}

func newMockConvRepo() *mockConvRepo {
	return &mockConvRepo{conversations: make(map[string]*domain.Conversation)}
}

func (r *mockConvRepo) GetConversation(_ context.Context, _ string, sessionID string) (*domain.Conversation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.conversations[sessionID]
	if !ok {
		return nil, domain.ErrConversationNotFound
	}
	return c, nil
}

func (r *mockConvRepo) SaveConversation(_ context.Context, c *domain.Conversation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.conversations[c.SessionID] = c
	return nil
}

func (r *mockConvRepo) FindByUserID(_ context.Context, userID string, limit, offset int) ([]*domain.Conversation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*domain.Conversation
	for _, c := range r.conversations {
		if c.UserID == userID {
			result = append(result, c)
		}
	}
	return result, nil
}

type mockEventPublisher struct {
	mu     sync.Mutex
	events []string
}

func (m *mockEventPublisher) PublishEvent(_ context.Context, ev events.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, string(ev.Type))
	return nil
}

func setupCoachService() (*Service, *mockLLMProvider, *mockConvRepo, *mockEventPublisher) {
	llm := &mockLLMProvider{
		response: "Great pass! Try to keep your head up.",
		trainingPlan: &domain.TrainingPlan{
			Objective:  "Improve passing accuracy",
			FocusAreas: []string{"short passing", "long passing"},
		},
		dietPlan: &domain.DietPlan{
			Calories: 2500,
			Macros:   domain.Macros{ProteinG: 150, CarbsG: 300, FatsG: 60},
		},
	}
	analysisSvc := &mockAnalysisService{
		insight: &domain.MatchInsight{MatchID: "m-1", DistanceKM: 10.5, TopSpeedKMH: 30.0, PassAccuracy: 0.82},
		history: []*domain.MatchInsight{
			{MatchID: "m-1", DistanceKM: 10.5},
			{MatchID: "m-2", DistanceKM: 8.2},
		},
	}
	userSvc := &mockUserService{
		profile: &domain.UserProfile{
			Age: 18, Position: "CM", Footedness: "Right", Goals: "improve progressive passing",
		},
	}
	convRepo := newMockConvRepo()
	pub := &mockEventPublisher{}
	svc := NewService(llm, analysisSvc, userSvc, convRepo)
	svc.SetEventPublisher(pub)
	return svc, llm, convRepo, pub
}

func TestChat_NewConversation(t *testing.T) {
	svc, llm, _, pub := setupCoachService()

	resp, err := svc.Chat(context.Background(), "user-1", "session-1", "How was my positioning?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "Great pass! Try to keep your head up." {
		t.Errorf("expected response text, got %s", resp.Content)
	}
	if resp.Role != domain.RoleAssistant {
		t.Errorf("expected RoleAssistant, got %s", resp.Role)
	}

	if len(pub.events) != 1 || pub.events[0] != "COACH_INTERACTION" {
		t.Errorf("expected COACH_INTERACTION event, got %v", pub.events)
	}

	_ = llm
}

func TestChat_ExistingConversation(t *testing.T) {
	svc, _, convRepo, _ := setupCoachService()

	svc.Chat(context.Background(), "user-1", "session-2", "First message")
	svc.Chat(context.Background(), "user-1", "session-2", "Second message")

	conv, err := convRepo.GetConversation(context.Background(), "user-1", "session-2")
	if err != nil {
		t.Fatalf("conversation not found: %v", err)
	}
	if len(conv.Messages) != 4 {
		t.Errorf("expected 4 messages (2 user + 2 assistant), got %d", len(conv.Messages))
	}
}

func TestChat_LLMError(t *testing.T) {
	svc, llm, _, _ := setupCoachService()
	llm.err = domain.ErrEmptyMessage

	_, err := svc.Chat(context.Background(), "user-1", "session-3", "Hello")
	if err == nil {
		t.Fatal("expected error from LLM, got nil")
	}
}

func TestGenerateTrainingPlan(t *testing.T) {
	svc, _, _, _ := setupCoachService()

	plan, err := svc.GenerateTrainingPlan(context.Background(), "user-1", []string{"passing", "positioning"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Objective != "Improve passing accuracy" {
		t.Errorf("expected objective, got %s", plan.Objective)
	}
	if plan.UserID != "user-1" {
		t.Errorf("expected user user-1, got %s", plan.UserID)
	}
}

func TestGenerateDietPlan(t *testing.T) {
	svc, _, _, _ := setupCoachService()

	plan, err := svc.GenerateDietPlan(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Calories != 2500 {
		t.Errorf("expected 2500 calories, got %d", plan.Calories)
	}
	if plan.UserID != "user-1" {
		t.Errorf("expected user user-1, got %s", plan.UserID)
	}
}

func TestGetHistory(t *testing.T) {
	svc, _, convRepo, _ := setupCoachService()

	svc.Chat(context.Background(), "user-1", "s-1", "Hello")
	svc.Chat(context.Background(), "user-1", "s-2", "Hi")

	conversations, err := svc.GetHistory(context.Background(), "user-1", 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conversations) != 2 {
		t.Errorf("expected 2 conversations, got %d", len(conversations))
	}

	_ = convRepo
}
