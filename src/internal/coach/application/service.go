package application

import (
	"context"
	"fmt"
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/coach/domain"
	"github.com/Aboody-Studios/ballr/src/pkg/events"
	"github.com/google/uuid"
)

// Service is the main application service for the Coach bounded context.
// It coordinates AI-driven coaching interactions including chat, training plans,
// and diet recommendations using LLM integration with RAG.
type Service struct {
	llmProvider     LLMProvider
	analysisService AnalysisService
	userService     UserService
	historyRepo     domain.ConversationRepository
	eventPublisher  events.Publisher
}

// LLMProvider defines the interface for AI model interactions.
// Implemented in infrastructure layer (OpenAI, Gemini, Z.ai, MiMo, etc.).
type LLMProvider interface {
	GenerateResponse(ctx context.Context, messages []domain.Message, context domain.CoachContext) (string, error)
	GenerateTrainingPlan(ctx context.Context, userProfile domain.UserProfile, focusAreas []string) (*domain.TrainingPlan, error)
	GenerateDietPlan(ctx context.Context, userProfile domain.UserProfile, trainingLoad domain.TrainingLoad) (*domain.DietPlan, error)
}

// AnalysisService provides integration with the Analysis bounded context.
type AnalysisService interface {
	GetLatestAnalysis(ctx context.Context, userID string) (*domain.MatchInsight, error)
	GetMatchHistory(ctx context.Context, userID string, limit int) ([]*domain.MatchInsight, error)
}

// UserService provides integration with the Identity bounded context.
type UserService interface {
	GetUserProfile(ctx context.Context, userID string) (*domain.UserProfile, error)
}

func NewService(
	llmProvider LLMProvider,
	analysisService AnalysisService,
	userService UserService,
	historyRepo domain.ConversationRepository,
) *Service {
	return &Service{
		llmProvider:     llmProvider,
		analysisService: analysisService,
		userService:     userService,
		historyRepo:     historyRepo,
		eventPublisher:  events.NoopPublisher(),
	}
}

func (s *Service) SetEventPublisher(p events.Publisher) {
	s.eventPublisher = p
}

// Chat handles a single message in an AI coach conversation using RAG pattern.
// Context includes user profile, latest match analysis, and recent training history.
func (s *Service) Chat(ctx context.Context, userID, sessionID, message string) (*domain.Message, error) {
	coachCtx, err := s.buildCoachContext(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to build coach context: %w", err)
	}

	conversation, err := s.historyRepo.GetConversation(ctx, userID, sessionID)
	if err != nil {
		conversation = &domain.Conversation{
			ID:        generateID(),
			UserID:    userID,
			SessionID: sessionID,
			Messages:  make([]domain.Message, 0),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Context:   *coachCtx,
		}
	}

	_ = conversation.AddMessage(domain.RoleUser, message)

	response, err := s.llmProvider.GenerateResponse(ctx, conversation.Messages, *coachCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate coach response: %w", err)
	}

	assistantMsg := conversation.AddMessage(domain.RoleAssistant, response)

	if err := s.historyRepo.SaveConversation(ctx, conversation); err != nil {
		fmt.Printf("failed to save conversation history: %v\n", err)
	}

	//TODO!: Create EventCoachInteraction struct here

	if err := s.eventPublisher.PublishEvent(ctx, userID, events.EventCoachInteraction, nil); err != nil {
		fmt.Printf("failed to publish coach interaction event: %v\n", err)
	}

	return &assistantMsg, nil
}

func (s *Service) GenerateTrainingPlan(ctx context.Context, userID string, focusAreas []string) (*domain.TrainingPlan, error) {
	profile, err := s.userService.GetUserProfile(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	plan, err := s.llmProvider.GenerateTrainingPlan(ctx, *profile, focusAreas)
	if err != nil {
		return nil, fmt.Errorf("failed to generate training plan: %w", err)
	}

	plan.UserID = userID
	plan.CreatedAt = time.Now()

	return plan, nil
}

func (s *Service) GenerateDietPlan(ctx context.Context, userID string) (*domain.DietPlan, error) {
	profile, err := s.userService.GetUserProfile(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	trainingLoad, err := s.calculateTrainingLoad(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate training load: %w", err)
	}

	plan, err := s.llmProvider.GenerateDietPlan(ctx, *profile, *trainingLoad)
	if err != nil {
		return nil, fmt.Errorf("failed to generate diet plan: %w", err)
	}

	plan.UserID = userID
	plan.CreatedAt = time.Now()

	return plan, nil
}

func (s *Service) buildCoachContext(ctx context.Context, userID string) (*domain.CoachContext, error) {
	profile, err := s.userService.GetUserProfile(ctx, userID)
	if err != nil {
		return nil, err
	}

	latestAnalysis, err := s.analysisService.GetLatestAnalysis(ctx, userID)
	if err != nil {
		latestAnalysis = nil
	}

	history, err := s.analysisService.GetMatchHistory(ctx, userID, 5)
	if err != nil {
		history = nil
	}

	var recentAnalysis *domain.RecentAnalysisContext
	if latestAnalysis != nil {
		recentAnalysis = &domain.RecentAnalysisContext{
			MatchID:       latestAnalysis.MatchID,
			MatchDate:     latestAnalysis.MatchDate,
			TotalDistance: latestAnalysis.DistanceKM,
			TopSpeed:      latestAnalysis.TopSpeedKMH,
			PassAccuracy:  latestAnalysis.PassAccuracy,
		}
	}

	var trainingHistory domain.TrainingHistoryContext
	if len(history) > 0 {
		trainingHistory.CurrentStreak = 1
		trainingHistory.LastActive = history[0].MatchDate
	}

	return &domain.CoachContext{
		UserProfile:     *profile,
		RecentAnalysis:  recentAnalysis,
		TrainingHistory: trainingHistory,
	}, nil
}

func (s *Service) GetHistory(ctx context.Context, userID string, limit, offset int) ([]*domain.Conversation, error) {
	return s.historyRepo.FindByUserID(ctx, userID, limit, offset)
}

func (s *Service) calculateTrainingLoad(ctx context.Context, userID string) (*domain.TrainingLoad, error) {
	history, err := s.analysisService.GetMatchHistory(ctx, userID, 7)
	if err != nil {
		return nil, err
	}

	load := &domain.TrainingLoad{
		RecentMatches: len(history),
		TimePeriod:    "7 days",
	}

	for _, match := range history {
		load.TotalDistance += match.DistanceKM
		load.TotalDuration += match.DurationMin
		load.IntensityScore += match.IntensityRating
	}

	if len(history) > 0 {
		load.AverageIntensity = float64(load.IntensityScore) / float64(len(history))
	}

	return load, nil
}

func generateID() string {
	return uuid.New().String()
}
