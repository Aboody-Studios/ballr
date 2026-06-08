package domain

import (
	"context"
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/identity/domain"
	"github.com/google/uuid"
)

// Conversation represents an AI coaching session with message history.
// This is the aggregate root for the Coach bounded context.
// TODO!: Remove gorm
type Conversation struct {
	ID        string      `gorm:"primaryKey"`
	User      domain.User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	UserID    string      `gorm:"index;not null"`
	SessionID string      `gorm:"uniqueIndex:idx_user_session;not null"`
	Messages  []Message   `gorm:"type:jsonb;serializer:json"`
	CreatedAt time.Time   `gorm:"autoCreateTime"`
	UpdatedAt time.Time   `gorm:"autoUpdateTime"`
	Context   Context     `gorm:"type:jsonb;serializer:json"`
}

type Message struct {
	ID        string    `json:"id"`
	Role      Role      `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
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

// TODO!: Remove gorm
type TrainingPlan struct {
	ID          string    `gorm:"primaryKey"`
	UserID      string    `gorm:"index;not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	ExpiresAt   time.Time
	Objective   string         `gorm:"type:text"`
	FocusAreas  []string       `gorm:"type:jsonb;serializer:json"`
	Drills      []PlanDrill    `gorm:"type:jsonb;serializer:json"`
	Schedule    WeeklySchedule `gorm:"type:jsonb;serializer:json"`
	AIReasoning string         `gorm:"type:text"`
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

// TODO!: Remove gorm
type DietPlan struct {
	ID          string    `gorm:"primaryKey"`
	UserID      string    `gorm:"index;not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	ExpiresAt   time.Time
	Calories    int
	Macros      Macros   `gorm:"type:jsonb;serializer:json"`
	Meals       []Meal   `gorm:"type:jsonb;serializer:json"`
	Hydration   string   `gorm:"type:text"`
	Supplements []string `gorm:"type:jsonb;serializer:json"`
	AIReasoning string   `gorm:"type:text"`
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

func generateID() string {
	return uuid.New().String()
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
	FindByUserID(ctx context.Context, userID string, limit, offset int) ([]*Conversation, error)
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
