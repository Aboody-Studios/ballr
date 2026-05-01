package infrastructure

import (
	"context"
	"fmt"
	"os"

	coachdomain "github.com/Aboody-Studios/ballr/src/internal/coach/domain"
)

type LLMProvider interface {
	GenerateResponse(ctx context.Context, messages []coachdomain.Message, context coachdomain.CoachContext) (string, error)
	GenerateTrainingPlan(ctx context.Context, userProfile coachdomain.UserProfile, focusAreas []string) (*coachdomain.TrainingPlan, error)
	GenerateDietPlan(ctx context.Context, userProfile coachdomain.UserProfile, trainingLoad coachdomain.TrainingLoad) (*coachdomain.DietPlan, error)
}

func NewLLMProvider() (LLMProvider, error) {
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		model := os.Getenv("OPENAI_MODEL")
		if model == "" {
			model = "gpt-4o"
		}
		baseURL := os.Getenv("OPENAI_BASE_URL")
		return NewOpenAIProvider(key, model, baseURL), nil
	}

	if key := os.Getenv("GOOGLE_AI_API_KEY"); key != "" {
		model := os.Getenv("GEMINI_MODEL")
		if model == "" {
			model = "gemini-3.0-flash"
		}
		return NewGeminiProvider(key, model)
	}

	return nil, fmt.Errorf("no LLM provider configured: set OPENAI_API_KEY or GOOGLE_AI_API_KEY")
}

func buildChatHistory(messages []coachdomain.Message, ctx coachdomain.CoachContext) (string, string) {
	profile := ctx.UserProfile
	contextInfo := fmt.Sprintf(
		"User profile:\n- Age: %d\n- Position: %s\n- Footedness: %s\n- Goals: %s\n",
		profile.Age, profile.Position, profile.Footedness, profile.Goals,
	)

	if analysis := ctx.RecentAnalysis; analysis != nil {
		contextInfo += fmt.Sprintf(
			"\nLatest match analysis:\n- Total distance: %.1f km\n- Top speed: %.1f km/h\n- Pass accuracy: %.1f%%\n- Key insight: %s\n",
			analysis.TotalDistance, analysis.TopSpeed, analysis.PassAccuracy, analysis.KeyInsight,
		)
	}

	history := ctx.TrainingHistory
	if history.CurrentStreak > 0 {
		contextInfo += fmt.Sprintf("\nTraining streak: %d days\n", history.CurrentStreak)
	}

	var historyText string
	for _, msg := range messages {
		role := "user"
		if msg.Role == coachdomain.RoleAssistant {
			role = "assistant"
		}
		historyText += fmt.Sprintf("%s: %s\n", role, msg.Content)
	}

	return contextInfo, historyText
}

const systemPrompt = `You are Ballr Coach, an expert football (soccer) coaching AI assistant. You analyze match performance data and provide personalized training and nutrition advice.

Key capabilities:
- Analyze match statistics and provide actionable feedback
- Generate structured training plans based on user goals and weak areas
- Create nutrition and diet plans suited to training load and position

Rules:
- Be concise and specific in your responses
- Reference the user's match data when available
- Suggest drills and exercises that target specific improvements
- Always respond with valid JSON when generating plans`

const trainingPlanPrompt = `Generate a personalized 7-day training plan as JSON. The response must be valid JSON matching this schema:
{
  "objective": "string - main goal of this plan",
  "focus_areas": ["string - targeted skills"],
  "drills": [{
    "name": "string - drill name",
    "duration_min": int,
    "sets": int,
    "reps_per_set": int,
    "intensity": "string - LOW/MODERATE/HIGH",
    "coaching_point": "string - key technical focus"
  }],
  "schedule": {
    "monday": [{"type": "TRAINING/CONDITIONING/MATCH/REST", "description": "string", "duration_min": int}],
    "tuesday": [],
    "wednesday": [],
    "thursday": [],
    "friday": [],
    "saturday": [],
    "sunday": []
  },
  "ai_reasoning": "string - brief explanation"
}

Include 3-5 drills, spread across the week with appropriate rest days. Return ONLY the JSON object, no markdown or extra text.`

const dietPlanPrompt = `Generate a 7-day nutrition plan as JSON. The response must be valid JSON matching this schema:
{
  "calories": int,
  "macros": {"protein_g": int, "carbs_g": int, "fats_g": int},
  "meals": [{
    "name": "string",
    "time": "HH:MM",
    "foods": ["string - food items"],
    "calories": int,
    "prep_notes": "string"
  }],
  "hydration": "string - hydration guidelines",
  "supplements": ["string - supplement names"],
  "ai_reasoning": "string - brief explanation"
}

Include 5 meals (breakfast, post-training, lunch, snack, dinner). Return ONLY the JSON object, no markdown or extra text.`
