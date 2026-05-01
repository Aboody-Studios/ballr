package infrastructure

import (
	"context"
	"fmt"
	"strings"

	coachdomain "github.com/Aboody-Studios/ballr/src/internal/coach/domain"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type GeminiProvider struct {
	client *genai.Client
	model  string
}

func NewGeminiProvider(apiKey, model string) (*GeminiProvider, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("gemini client: %w", err)
	}
	return &GeminiProvider{client: client, model: model}, nil
}

func (p *GeminiProvider) GenerateResponse(ctx context.Context, messages []coachdomain.Message, coachCtx coachdomain.CoachContext) (string, error) {
	contextInfo, historyText := buildChatHistory(messages, coachCtx)

	model := p.client.GenerativeModel(p.model)
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(fmt.Sprintf("%s\n\nCurrent context:\n%s", systemPrompt, contextInfo))},
	}

	chat := model.StartChat()

	if historyText != "" {
		chat.History = append(chat.History, &genai.Content{
			Parts: []genai.Part{genai.Text(historyText)},
			Role:  "user",
		})
	}

	// Find the last user message to send
	var lastMsg string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == coachdomain.RoleUser {
			lastMsg = messages[i].Content
			break
		}
	}
	if lastMsg == "" {
		lastMsg = "Provide coaching advice based on the context above."
	}

	resp, err := chat.SendMessage(ctx, genai.Text(lastMsg))
	if err != nil {
		return "", fmt.Errorf("gemini send message: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini returned no content")
	}

	var result strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			result.WriteString(string(text))
		}
	}

	return result.String(), nil
}

func (p *GeminiProvider) GenerateTrainingPlan(ctx context.Context, userProfile coachdomain.UserProfile, focusAreas []string) (*coachdomain.TrainingPlan, error) {
	userInfo := fmt.Sprintf(
		"Generate a training plan for a %d-year-old %s %s-footed player. Goals: %s. Focus areas: %v",
		userProfile.Age, userProfile.Position, userProfile.Footedness, userProfile.Goals, focusAreas,
	)

	model := p.client.GenerativeModel(p.model)
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(trainingPlanPrompt)},
	}

	resp, err := model.GenerateContent(ctx, genai.Text(userInfo))
	if err != nil {
		return nil, fmt.Errorf("gemini training plan: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini returned no content")
	}

	var content string
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			content += string(text)
		}
	}

	return parseTrainingPlanJSON(cleanJSON(content), focusAreas)
}

func (p *GeminiProvider) GenerateDietPlan(ctx context.Context, userProfile coachdomain.UserProfile, trainingLoad coachdomain.TrainingLoad) (*coachdomain.DietPlan, error) {
	userInfo := fmt.Sprintf(
		"Generate a diet plan for a %d-year-old %s-footed %s. Training load: %d matches in %s, avg intensity %.1f/10.",
		userProfile.Age, userProfile.Footedness, userProfile.Position,
		trainingLoad.RecentMatches, trainingLoad.TimePeriod, trainingLoad.AverageIntensity,
	)

	model := p.client.GenerativeModel(p.model)
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(dietPlanPrompt)},
	}

	resp, err := model.GenerateContent(ctx, genai.Text(userInfo))
	if err != nil {
		return nil, fmt.Errorf("gemini diet plan: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini returned no content")
	}

	var content string
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			content += string(text)
		}
	}

	return parseDietPlanJSON(cleanJSON(content))
}

func cleanJSON(content string) string {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
		if idx := strings.LastIndex(content, "```"); idx >= 0 {
			content = content[:idx]
		}
	}
	if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
		if idx := strings.LastIndex(content, "```"); idx >= 0 {
			content = content[:idx]
		}
	}
	return strings.TrimSpace(content)
}
