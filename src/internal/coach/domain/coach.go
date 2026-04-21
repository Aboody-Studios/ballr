package domain

import (
	"context"
	"time"
)

// Conversation represents an AI coaching session with message history.
// This is the aggregate root for the Coach bounded context.
type Conversation struct {
	ID        string
	UserID    string
	SessionID string
	Messages  []Message
	CreatedAt time.Time
	UpdatedAt time.Time
	Context   Context
}

type Message struct {
	ID        string
	Role      Role
	Content   string
	Timestamp time.Time
}

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system" // For system prompts, rarely stored
)

// Context contains data used for RAG (Retrieval-Augmented Generation).
type Context struct {
	UserProfile     UserProfileContext
	RecentAnalysis  *RecentAnalysisContext
	TrainingHistory TrainingHistoryContext
}

type UserProfileContext struct {
	Age        int
	Position   string
	Footedness string
	Goals      string
}

type RecentAnalysisContext struct {
	MatchID       string
	MatchDate     time.Time
	TotalDistance float64
	TopSpeed      float64
	PassAccuracy  float64
	KeyInsight    string
	Events        []string
}

type TrainingHistoryContext struct {
	CurrentStreak int
	TotalPoints   int
	LastActive    time.Time
	RecentDrills  []string
}

type TrainingPlan struct {
	ID          string
	UserID      string
	CreatedAt   time.Time
	ExpiresAt   time.Time
	Objective   string
	FocusAreas  []string
	Drills      []PlanDrill
	Schedule    WeeklySchedule
	AIReasoning string
}

type PlanDrill struct {
	DrillID       string
	Name          string
	DurationMin   int
	Sets          int
	RepsPerSet    int
	Intensity     string
	CoachingPoint string
}

type WeeklySchedule struct {
	Monday    []Activity
	Tuesday   []Activity
	Wednesday []Activity
	Thursday  []Activity
	Friday    []Activity
	Saturday  []Activity
	Sunday    []Activity
}

type Activity struct {
	Type        string
	DrillID     string
	Description string
	DurationMin int
}

type DietPlan struct {
	ID          string
	UserID      string
	CreatedAt   time.Time
	ExpiresAt   time.Time
	Calories    int
	Macros      Macros
	Meals       []Meal
	Hydration   string
	Supplements []string
	AIReasoning string
}

type Macros struct {
	ProteinG int
	CarbsG   int
	FatsG    int
}

type Meal struct {
	Name      string
	Time      string
	Foods     []string
	Calories  int
	PrepNotes string
}

type DrillsBankEntry struct {
	ID             string
	Name           string
	Category       Category
	Difficulty     Difficulty
	Content        string
	Steps          []string
	Setup          string
	CoachingPoints []string
	DurationMin    int
	VideoURL       string
}

type Category string

const (
	CategoryPassing     Category = "PASSING"
	CategoryFinishing   Category = "FINISHING"
	CategoryDribbling   Category = "DRIBBLING"
	CategoryDefending   Category = "DEFENDING"
	CategoryGoalkeeping Category = "GOALKEEPING"
	CategoryPhysical    Category = "PHYSICAL"
	CategoryTactical    Category = "TACTICAL"
)

type Difficulty string

const (
	DifficultyBeginner     Difficulty = "BEGINNER"
	DifficultyIntermediate Difficulty = "INTERMEDIATE"
	DifficultyAdvanced     Difficulty = "ADVANCED"
)

func NewConversation(id, userID, sessionID string, ctx Context) *Conversation {
	return &Conversation{
		ID:        id,
		UserID:    userID,
		SessionID: sessionID,
		Messages:  make([]Message, 0),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Context:   ctx,
	}
}

func (c *Conversation) AddMessage(role Role, content string) Message {
	msg := Message{
		ID:        generateID(),
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}
	c.Messages = append(c.Messages, msg)
	c.UpdatedAt = time.Now()
	return msg
}

func (c *Conversation) GetLastMessage() *Message {
	if len(c.Messages) == 0 {
		return nil
	}
	return &c.Messages[len(c.Messages)-1]
}

// IsExpired returns true if the conversation has been inactive for too long.
// Conversations expire after 24 hours of inactivity to manage storage.
func (c *Conversation) IsExpired() bool {
	return time.Since(c.UpdatedAt) > 24*time.Hour
}

// ToPromptContext serializes the conversation and context for LLM consumption.
// This formats the RAG context and recent messages into a prompt-ready string.
func (c *Conversation) ToPromptContext() string {
	// Implementation would format context for the specific LLM
	// Includes: user profile summary, recent analysis, training history,
	// and the last N messages from this conversation.
	return ""
}

// TODO: Use github.com/google/uuid or similar for production.
func generateID() string {
	return "stub-id"
}

type TrainingLoad struct {
	RecentMatches    int
	TimePeriod       string
	TotalDistance    float64
	TotalDuration    int
	IntensityScore   int
	AverageIntensity float64
}

type MatchInsight struct {
	MatchID         string
	MatchDate       time.Time
	DistanceKM      float64
	TopSpeedKMH     float64
	DurationMin     int
	PassAccuracy    float64
	IntensityRating int
}

// ConversationRepository defines the contract for persisting conversations.
// Implemented in infrastructure layer.
type ConversationRepository interface {
	GetConversation(ctx context.Context, userID, sessionID string) (*Conversation, error)
	SaveConversation(ctx context.Context, conversation *Conversation) error
}

type CoachContext = Context
type UserProfile = UserProfileContext

var (
	ErrConversationNotFound = &CoachError{"conversation not found"}
	ErrConversationExpired  = &CoachError{"conversation has expired"}
	ErrEmptyMessage         = &CoachError{"message content cannot be empty"}
	ErrInvalidDrillCategory = &CoachError{"invalid drill category"}
)

type CoachError struct {
	Message string
}

func (e *CoachError) Error() string {
	return e.Message
}
