package http

import (
	"net/http"
	"strconv"

	"github.com/Aboody-Studios/ballr/src/internal/progress/application"
	"github.com/Aboody-Studios/ballr/src/internal/shared/delivery"
	"github.com/labstack/echo/v5"
)

type ProgressHandler struct {
	progressService *application.GamificationService
}

func NewProgressHandler(service *application.GamificationService) *ProgressHandler {
	return &ProgressHandler{progressService: service}
}

func (h *ProgressHandler) GetProgressSummaryHandler(c *echo.Context) error {
	claims, err := delivery.ExtractToken(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	summary, err := h.progressService.GetProgressSummary(c.Request().Context(), claims.ID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Progress not found"})
	}

	return c.JSON(http.StatusOK, summary)
}

func (h *ProgressHandler) ListAchievementsHandler(c *echo.Context) error {
	claims, err := delivery.ExtractToken(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	achievements, err := h.progressService.GetAchievements(c.Request().Context(), claims.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to load achievements"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"achievements": achievements,
	})
}

func (h *ProgressHandler) GetLeaderboardHandler(c *echo.Context) error {
	offsetStr := c.QueryParam("offset")
	limitStr := c.QueryParam("limit")

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 25
	}

	leaderboard, err := h.progressService.GetLeaderboard(c.Request().Context(), offset, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to load leaderboard"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"leaderboard": leaderboard,
		"offset":      offset,
		"limit":       limit,
	})
}
