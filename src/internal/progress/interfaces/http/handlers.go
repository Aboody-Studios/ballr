package http

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

// ProgressHandler handles HTTP requests for the Progress bounded context.
type ProgressHandler struct {
	// progressService will be wired when the application service is implemented
	// progressService *application.Service
}

// NewProgressHandler creates a new progress handler.
// TODO!: Wire up application service when implementing gamification logic.
func NewProgressHandler() *ProgressHandler {
	return &ProgressHandler{}
}

// GetProgressSummaryHandler returns the user's total points, current streak, and recent activity.
func (h *ProgressHandler) GetProgressSummaryHandler(c *echo.Context) error {
	// TODO!: Implement when progress application service is available

	return c.JSON(http.StatusOK, map[string]interface{}{
		"total_points":   0,
		"current_streak": 0,
		"last_active":    nil,
		"achievements":   []string{},
	})
}

// ListAchievementsHandler returns all achievements for the user with unlock status.
func (h *ProgressHandler) ListAchievementsHandler(c *echo.Context) error {
	// TODO!: Implement achievement listing

	return c.JSON(http.StatusOK, map[string]interface{}{
		"achievements": []string{},
	})
}

// GetLeaderboardHandler returns the global or friends-only leaderboard.
func (h *ProgressHandler) GetLeaderboardHandler(c *echo.Context) error {
	// TODO!: Implement leaderboard

	return c.JSON(http.StatusOK, map[string]interface{}{
		"leaderboard": []string{},
		"user_rank":   0,
	})
}
