package http

import (
	"net/http"

	"github.com/Aboody-Studios/ballr/src/internal/coach/application"
	"github.com/labstack/echo/v5"
)

// CoachHandler handles HTTP requests for the Coach bounded context.
// This includes AI chat interactions, training plan generation, and diet recommendations.
// The Coach uses RAG (Retrieval-Augmented Generation) with user context including
// age, position, match history, and training goals.
type CoachHandler struct {
	coachService *application.Service
}

func NewCoachHandler(service *application.Service) *CoachHandler {
	return &CoachHandler{coachService: service}
}

// ChatHandler handles conversational interactions with the AI Coach.
// Endpoint: POST /coach/chat
func (h *CoachHandler) ChatHandler(c *echo.Context) error {
	var req struct {
		Message   string `json:"message" validate:"required"`
		SessionID string `json:"session_id,omitempty"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid JSON format"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Validation failed"})
	}

	// TODO!: Extract from JWT claims once auth middleware is fully implemented
	userID := "stub-user-id"

	response, err := h.coachService.Chat(c.Request().Context(), userID, req.Message, req.SessionID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "AI service unavailable"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"response":   response.Content,
		"session_id": req.SessionID,
		"role":       response.Role,
	})
}

// GeneratePlanHandler creates personalized training plans based on user goals and recent performance.
// Endpoint: POST /coach/plan/generate
func (h *CoachHandler) GeneratePlanHandler(c *echo.Context) error {
	var req struct {
		FocusAreas []string `json:"focus_areas,omitempty"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid JSON format"})
	}

	// TODO!: Extract from JWT claims once auth middleware is fully implemented
	userID := "stub-user-id"

	plan, err := h.coachService.GenerateTrainingPlan(c.Request().Context(), userID, req.FocusAreas)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate plan"})
	}

	return c.JSON(http.StatusCreated, plan)
}

// GenerateDietHandler creates nutrition recommendations based on user profile and training load.
// Endpoint: POST /coach/diet/generate
func (h *CoachHandler) GenerateDietHandler(c *echo.Context) error {
	// TODO!: Extract from JWT claims once auth middleware is fully implemented
	userID := "stub-user-id"

	dietPlan, err := h.coachService.GenerateDietPlan(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate diet plan"})
	}

	return c.JSON(http.StatusCreated, dietPlan)
}

// GetHistoryHandler retrieves the conversation history for the authenticated user.
// Endpoint: GET /coach/history
func (h *CoachHandler) GetHistoryHandler(c *echo.Context) error {
	// TODO!: Implement conversation history retrieval
	// This requires a CoachHistoryRepository in the infrastructure layer

	return c.JSON(http.StatusOK, map[string]interface{}{
		"conversations": []interface{}{},
		"total":         0,
	})
}
