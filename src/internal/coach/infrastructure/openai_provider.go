package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	coachdomain "github.com/Aboody-Studios/ballr/src/internal/coach/domain"
	"github.com/google/uuid"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type OpenAIProvider struct {
	client *openai.Client
	model  string
}

func NewOpenAIProvider(apiKey, model, baseURL string) *OpenAIProvider {
	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	client := openai.NewClient(opts...)
	return &OpenAIProvider{client: &client, model: model}
}

func (p *OpenAIProvider) GenerateResponse(ctx context.Context, messages []coachdomain.Message, coachCtx coachdomain.CoachContext) (string, error) {
	contextInfo, historyText := buildChatHistory(messages, coachCtx)

	systemMsg := fmt.Sprintf("%s\n\nCurrent context:\n%s", systemPrompt, contextInfo)

	apiMessages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemMsg),
	}
	if historyText != "" {
		apiMessages = append(apiMessages, openai.UserMessage(historyText))
	}

	resp, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: apiMessages,
		Model:    p.model,
	})
	if err != nil {
		return "", fmt.Errorf("openai chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}

	return resp.Choices[0].Message.Content, nil
}

func (p *OpenAIProvider) GenerateTrainingPlan(ctx context.Context, userProfile coachdomain.UserProfile, focusAreas []string) (*coachdomain.TrainingPlan, error) {
	userInfo := fmt.Sprintf(
		"Generate a training plan for a %d-year-old %s %s-footed player. Goals: %s. Focus areas: %v",
		userProfile.Age, userProfile.Position, userProfile.Footedness, userProfile.Goals, focusAreas,
	)

	resp, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(trainingPlanPrompt),
			openai.UserMessage(userInfo),
		},
		Model: p.model,
	})
	if err != nil {
		return nil, fmt.Errorf("openai training plan: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("openai returned no choices")
	}

	return parseTrainingPlanJSON(resp.Choices[0].Message.Content, focusAreas)
}

func (p *OpenAIProvider) GenerateDietPlan(ctx context.Context, userProfile coachdomain.UserProfile, trainingLoad coachdomain.TrainingLoad) (*coachdomain.DietPlan, error) {
	userInfo := fmt.Sprintf(
		"Generate a diet plan for a %d-year-old %s-footed %s. Training load: %d matches in %s, avg intensity %.1f/10.",
		userProfile.Age, userProfile.Footedness, userProfile.Position,
		trainingLoad.RecentMatches, trainingLoad.TimePeriod, trainingLoad.AverageIntensity,
	)

	resp, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(dietPlanPrompt),
			openai.UserMessage(userInfo),
		},
		Model: p.model,
	})
	if err != nil {
		return nil, fmt.Errorf("openai diet plan: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("openai returned no choices")
	}

	return parseDietPlanJSON(resp.Choices[0].Message.Content)
}

type trainingPlanJSON struct {
	Objective   string       `json:"objective"`
	FocusAreas  []string     `json:"focus_areas"`
	Drills      []drillJSON  `json:"drills"`
	Schedule    scheduleJSON `json:"schedule"`
	AIReasoning string       `json:"ai_reasoning"`
}

type drillJSON struct {
	Name          string `json:"name"`
	DurationMin   int    `json:"duration_min"`
	Sets          int    `json:"sets"`
	RepsPerSet    int    `json:"reps_per_set"`
	Intensity     string `json:"intensity"`
	CoachingPoint string `json:"coaching_point"`
}

type scheduleJSON struct {
	Monday    []activityJSON `json:"monday"`
	Tuesday   []activityJSON `json:"tuesday"`
	Wednesday []activityJSON `json:"wednesday"`
	Thursday  []activityJSON `json:"thursday"`
	Friday    []activityJSON `json:"friday"`
	Saturday  []activityJSON `json:"saturday"`
	Sunday    []activityJSON `json:"sunday"`
}

type activityJSON struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	DurationMin int    `json:"duration_min"`
}

type dietPlanJSON struct {
	Calories    int        `json:"calories"`
	Macros      macrosJSON `json:"macros"`
	Meals       []mealJSON `json:"meals"`
	Hydration   string     `json:"hydration"`
	Supplements []string   `json:"supplements"`
	AIReasoning string     `json:"ai_reasoning"`
}

type macrosJSON struct {
	ProteinG int `json:"protein_g"`
	CarbsG   int `json:"carbs_g"`
	FatsG    int `json:"fats_g"`
}

type mealJSON struct {
	Name      string   `json:"name"`
	Time      string   `json:"time"`
	Foods     []string `json:"foods"`
	Calories  int      `json:"calories"`
	PrepNotes string   `json:"prep_notes"`
}

func parseTrainingPlanJSON(content string, focusAreas []string) (*coachdomain.TrainingPlan, error) {
	var raw trainingPlanJSON
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("parse training plan: %w", err)
	}

	now := time.Now()
	plan := &coachdomain.TrainingPlan{
		ID:          uuid.New().String(),
		UserID:      "",
		CreatedAt:   now,
		ExpiresAt:   now.Add(7 * 24 * time.Hour),
		Objective:   raw.Objective,
		FocusAreas:  raw.FocusAreas,
		AIReasoning: raw.AIReasoning,
	}

	for _, d := range raw.Drills {
		plan.Drills = append(plan.Drills, coachdomain.PlanDrill{
			DrillID:       uuid.New().String(),
			Name:          d.Name,
			DurationMin:   d.DurationMin,
			Sets:          d.Sets,
			RepsPerSet:    d.RepsPerSet,
			Intensity:     d.Intensity,
			CoachingPoint: d.CoachingPoint,
		})
	}

	plan.Schedule = coachdomain.WeeklySchedule{
		Monday:    parseActivities(raw.Schedule.Monday),
		Tuesday:   parseActivities(raw.Schedule.Tuesday),
		Wednesday: parseActivities(raw.Schedule.Wednesday),
		Thursday:  parseActivities(raw.Schedule.Thursday),
		Friday:    parseActivities(raw.Schedule.Friday),
		Saturday:  parseActivities(raw.Schedule.Saturday),
		Sunday:    parseActivities(raw.Schedule.Sunday),
	}

	return plan, nil
}

func parseActivities(activities []activityJSON) []coachdomain.Activity {
	var result []coachdomain.Activity
	for _, a := range activities {
		result = append(result, coachdomain.Activity{
			Type:        a.Type,
			Description: a.Description,
			DurationMin: a.DurationMin,
		})
	}
	return result
}

func parseDietPlanJSON(content string) (*coachdomain.DietPlan, error) {
	var raw dietPlanJSON
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("parse diet plan: %w", err)
	}

	now := time.Now()
	plan := &coachdomain.DietPlan{
		ID:          uuid.New().String(),
		UserID:      "",
		CreatedAt:   now,
		ExpiresAt:   now.Add(7 * 24 * time.Hour),
		Calories:    raw.Calories,
		Macros:      coachdomain.Macros{ProteinG: raw.Macros.ProteinG, CarbsG: raw.Macros.CarbsG, FatsG: raw.Macros.FatsG},
		Hydration:   raw.Hydration,
		Supplements: raw.Supplements,
		AIReasoning: raw.AIReasoning,
	}

	for _, m := range raw.Meals {
		plan.Meals = append(plan.Meals, coachdomain.Meal{
			Name:      m.Name,
			Time:      m.Time,
			Foods:     m.Foods,
			Calories:  m.Calories,
			PrepNotes: m.PrepNotes,
		})
	}

	return plan, nil
}
