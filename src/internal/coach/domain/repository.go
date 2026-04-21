package domain

import "context"

type CoachRepository interface {
	SaveConversation(ctx context.Context, conversation *Conversation) error
	GetConversation(ctx context.Context, sessionID string) (*Conversation, error)
	ListConversations(ctx context.Context, userID string, limit int) ([]*Conversation, error)
	SaveTrainingPlan(ctx context.Context, plan *TrainingPlan) error
	GetTrainingPlan(ctx context.Context, planID string) (*TrainingPlan, error)
	ListTrainingPlans(ctx context.Context, userID string) ([]*TrainingPlan, error)
	SaveDietPlan(ctx context.Context, plan *DietPlan) error
	GetDietPlan(ctx context.Context, planID string) (*DietPlan, error)
}

var ErrPlanNotFound = errPlanNotFound{}

type errPlanNotFound struct{}

func (errPlanNotFound) Error() string { return "plan not found" }
