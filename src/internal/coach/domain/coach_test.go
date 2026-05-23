package domain

import (
	"testing"
	"time"
)

func TestNewConversation(t *testing.T) {
	ctx := Context{
		UserProfile: UserProfileContext{
			Age: 18, Position: "CM", Footedness: "Right", Goals: "improve passing",
		},
	}
	c := NewConversation("conv-1", "user-1", "session-1", ctx)
	if c.ID != "conv-1" {
		t.Errorf("expected conv-1, got %s", c.ID)
	}
	if c.UserID != "user-1" {
		t.Errorf("expected user-1, got %s", c.UserID)
	}
	if c.SessionID != "session-1" {
		t.Errorf("expected session-1, got %s", c.SessionID)
	}
	if len(c.Messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(c.Messages))
	}
}

func TestAddMessage(t *testing.T) {
	c := NewConversation("conv-1", "user-1", "session-1", Context{})
	msg := c.AddMessage(RoleUser, "Hello coach")
	if msg.Role != RoleUser {
		t.Errorf("expected RoleUser, got %s", msg.Role)
	}
	if msg.Content != "Hello coach" {
		t.Errorf("expected 'Hello coach', got '%s'", msg.Content)
	}
	if msg.ID == "" {
		t.Error("expected non-empty message ID")
	}
	if len(c.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(c.Messages))
	}

	assistantMsg := c.AddMessage(RoleAssistant, "Let me analyze your match")
	if len(c.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(c.Messages))
	}
	if assistantMsg.Role != RoleAssistant {
		t.Errorf("expected RoleAssistant, got %s", assistantMsg.Role)
	}
}

func TestGetLastMessage(t *testing.T) {
	t.Run("has messages", func(t *testing.T) {
		c := NewConversation("conv-1", "user-1", "session-1", Context{})
		c.AddMessage(RoleUser, "First")
		c.AddMessage(RoleAssistant, "Second")
		last := c.GetLastMessage()
		if last == nil {
			t.Fatal("expected non-nil")
		}
		if last.Content != "Second" {
			t.Errorf("expected 'Second', got '%s'", last.Content)
		}
	})

	t.Run("no messages", func(t *testing.T) {
		c := NewConversation("conv-1", "user-1", "session-1", Context{})
		if last := c.GetLastMessage(); last != nil {
			t.Error("expected nil")
		}
	})
}

func TestIsExpired(t *testing.T) {
	t.Run("recent conversation not expired", func(t *testing.T) {
		c := NewConversation("conv-1", "user-1", "session-1", Context{})
		if c.IsExpired() {
			t.Error("expected not expired")
		}
	})

	t.Run("old conversation expired", func(t *testing.T) {
		c := NewConversation("conv-1", "user-1", "session-1", Context{})
		c.UpdatedAt = time.Now().Add(-48 * time.Hour)
		if !c.IsExpired() {
			t.Error("expected expired")
		}
	})
}

func TestToPromptContext(t *testing.T) {
	t.Run("with user profile only", func(t *testing.T) {
		ctx := Context{
			UserProfile:     UserProfileContext{Age: 20, Position: "ST", Footedness: "Left", Goals: "score more"},
			TrainingHistory: TrainingHistoryContext{CurrentStreak: 3},
		}
		c := NewConversation("conv-1", "user-1", "session-1", ctx)
		result := c.ToPromptContext()
		if result == "" {
			t.Fatal("expected non-empty context")
		}
	})

	t.Run("with recent analysis", func(t *testing.T) {
		ctx := Context{
			UserProfile: UserProfileContext{Age: 22, Position: "GK", Footedness: "Right", Goals: "better reflexes"},
			RecentAnalysis: &RecentAnalysisContext{
				TotalDistance: 5.2, TopSpeed: 28.0, PassAccuracy: 0.75,
			},
			TrainingHistory: TrainingHistoryContext{CurrentStreak: 5},
		}
		c := NewConversation("conv-1", "user-1", "session-1", ctx)
		c.AddMessage(RoleUser, "How was my positioning?")
		c.AddMessage(RoleAssistant, "Your positioning was solid")
		result := c.ToPromptContext()
		if result == "" {
			t.Fatal("expected non-empty context")
		}
	})
}

func TestConversationTypes(t *testing.T) {
	if RoleUser != "user" {
		t.Errorf("expected 'user', got '%s'", RoleUser)
	}
	if RoleAssistant != "assistant" {
		t.Errorf("expected 'assistant', got '%s'", RoleAssistant)
	}
}

func TestCategory(t *testing.T) {
	if CategoryPassing != "PASSING" {
		t.Errorf("expected PASSING, got '%s'", CategoryPassing)
	}
}

func TestDifficulty(t *testing.T) {
	if DifficultyBeginner != "BEGINNER" {
		t.Errorf("expected BEGINNER, got '%s'", DifficultyBeginner)
	}
}

func TestCoachError(t *testing.T) {
	err := &CoachError{"test error"}
	if err.Error() != "test error" {
		t.Errorf("expected 'test error', got '%s'", err.Error())
	}
}

func TestErrPlanNotFound(t *testing.T) {
	if ErrConversationNotFound.Error() != "conversation not found" {
		t.Errorf("expected 'conversation not found', got '%s'", ErrConversationNotFound.Error())
	}
	if ErrConversationExpired.Error() != "conversation has expired" {
		t.Errorf("expected 'conversation has expired', got '%s'", ErrConversationExpired.Error())
	}
	if ErrEmptyMessage.Error() != "message content cannot be empty" {
		t.Errorf("expected 'message content cannot be empty', got '%s'", ErrEmptyMessage.Error())
	}
}
