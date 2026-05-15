package infrastructure

import (
	"os"
	"testing"

	coachdomain "github.com/Aboody-Studios/ballr/src/internal/coach/domain"
)

func TestNewLLMProvider_NoConfig(t *testing.T) {
	os.Unsetenv("OPENAI_API_KEY")
	os.Unsetenv("GOOGLE_AI_API_KEY")
	_, err := NewLLMProvider()
	if err == nil {
		t.Fatal("expected error when no API keys configured")
	}
}

func TestBuildChatHistory(t *testing.T) {
	ctx := coachdomain.CoachContext{
		UserProfile: coachdomain.UserProfileContext{
			Age: 20, Position: "CM", Footedness: "Right", Goals: "improve passing",
		},
		RecentAnalysis: &coachdomain.RecentAnalysisContext{
			TotalDistance: 10.5, TopSpeed: 30.0, PassAccuracy: 0.82, KeyInsight: "Great movement",
		},
		TrainingHistory: coachdomain.TrainingHistoryContext{
			CurrentStreak: 5,
		},
	}
	messages := []coachdomain.Message{
		{Role: coachdomain.RoleUser, Content: "How was my match?"},
		{Role: coachdomain.RoleAssistant, Content: "You played well."},
	}

	contextInfo, historyText := buildChatHistory(messages, ctx)
	if contextInfo == "" {
		t.Fatal("expected non-empty context info")
	}
	if historyText == "" {
		t.Fatal("expected non-empty history text")
	}
	if historyText != "user: How was my match?\nassistant: You played well.\n" {
		t.Errorf("unexpected history: %s", historyText)
	}
}

func TestBuildChatHistory_Empty(t *testing.T) {
	ctx := coachdomain.CoachContext{
		UserProfile: coachdomain.UserProfileContext{Age: 18, Position: "ST", Footedness: "Left", Goals: "score"},
	}
	contextInfo, historyText := buildChatHistory(nil, ctx)
	if contextInfo == "" {
		t.Fatal("expected non-empty context info")
	}
	if historyText != "" {
		t.Errorf("expected empty history, got %s", historyText)
	}
}

func TestCleanJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`{"key": "value"}`, `{"key": "value"}`},
		{"```json\n{\"key\": \"value\"}\n```", `{"key": "value"}`},
		{"```\n{\"key\": \"value\"}\n```", `{"key": "value"}`},
		{`   {"key": "value"}   `, `{"key": "value"}`},
	}
	for _, tc := range tests {
		got := cleanJSON(tc.input)
		if got != tc.expected {
			t.Errorf("cleanJSON(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestParseTrainingPlanJSON(t *testing.T) {
	json := `{
		"objective": "Improve passing accuracy",
		"focus_areas": ["short passing", "long passing"],
		"ai_reasoning": "Based on match analysis",
		"drills": [
			{
				"name": "Passing drill",
				"duration_min": 15,
				"sets": 3,
				"reps_per_set": 10,
				"intensity": "medium",
				"coaching_point": "Keep your head up"
			}
		],
		"schedule": {
			"monday": [{"type": "practice", "description": "Passing practice", "duration_min": 30}],
			"tuesday": [],
			"wednesday": [],
			"thursday": [],
			"friday": [],
			"saturday": [],
			"sunday": []
		}
	}`
	plan, err := parseTrainingPlanJSON(json, []string{"passing"})
	if err != nil {
		t.Fatalf("parseTrainingPlanJSON failed: %v", err)
	}
	if plan.Objective != "Improve passing accuracy" {
		t.Errorf("expected 'Improve passing accuracy', got '%s'", plan.Objective)
	}
	if len(plan.Drills) != 1 {
		t.Errorf("expected 1 drill, got %d", len(plan.Drills))
	}
}

func TestParseTrainingPlanJSON_Invalid(t *testing.T) {
	_, err := parseTrainingPlanJSON("invalid json", nil)
	if err == nil {
		t.Fatal("expected parse error for invalid JSON")
	}
}

func TestParseDietPlanJSON(t *testing.T) {
	json := `{
		"calories": 2500,
		"macros": {"protein_g": 150, "carbs_g": 300, "fats_g": 60},
		"hydration": "2-3 liters daily",
		"supplements": ["vitamin D", "omega-3"],
		"ai_reasoning": "Based on training load",
		"meals": [
			{
				"name": "Breakfast",
				"time": "07:00",
				"foods": ["oatmeal", "eggs"],
				"calories": 600,
				"prep_notes": "Quick prep"
			}
		]
	}`
	plan, err := parseDietPlanJSON(json)
	if err != nil {
		t.Fatalf("parseDietPlanJSON failed: %v", err)
	}
	if plan.Calories != 2500 {
		t.Errorf("expected 2500, got %d", plan.Calories)
	}
	if len(plan.Meals) != 1 {
		t.Errorf("expected 1 meal, got %d", len(plan.Meals))
	}
}

func TestParseDietPlanJSON_Invalid(t *testing.T) {
	_, err := parseDietPlanJSON("not json")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseActivities(t *testing.T) {
	activities := parseActivities([]activityJSON{
		{Type: "practice", Description: "Dribbling", DurationMin: 20},
	})
	if len(activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(activities))
	}
	if activities[0].Type != "practice" {
		t.Errorf("expected 'practice', got '%s'", activities[0].Type)
	}
	if activities[0].DurationMin != 20 {
		t.Errorf("expected 20, got %d", activities[0].DurationMin)
	}
}

func TestParseActivities_Empty(t *testing.T) {
	activities := parseActivities(nil)
	if len(activities) != 0 {
		t.Errorf("expected 0, got %d", len(activities))
	}
}
