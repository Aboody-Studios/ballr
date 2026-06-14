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

func (h *ProgressHandler) GetProgressSummaryHandler(echoCtx *echo.Context) error {
	claims, err := delivery.ExtractToken(echoCtx)
	if err != nil {
		return echoCtx.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	summary, err := h.progressService.GetProgressSummary(echoCtx.Request().Context(), claims.ID)
	if err != nil {
		return echoCtx.JSON(http.StatusNotFound, map[string]string{"error": "Progress not found"})
	}
	// TODO!: Maybe return map text like the handlers below
	return echoCtx.JSON(http.StatusOK, summary)
}

func (h *ProgressHandler) ListAchievementsHandler(echoCtx *echo.Context) error {
	claims, err := delivery.ExtractToken(echoCtx)
	if err != nil {
		return echoCtx.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	achievements, err := h.progressService.GetAchievements(echoCtx.Request().Context(), claims.ID)
	if err != nil {
		return echoCtx.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to load achievements"})
	}

	return echoCtx.JSON(http.StatusOK, map[string]any{
		"achievements": achievements,
	})
}

func (h *ProgressHandler) GetLeaderboardHandler(echoCtx *echo.Context) error {
	offsetStr := echoCtx.QueryParam("offset")
	limitStr := echoCtx.QueryParam("limit")

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 25
	}

	leaderboard, err := h.progressService.GetLeaderboard(echoCtx.Request().Context(), int64(offset), int64(limit))
	if err != nil {
		return echoCtx.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to load leaderboard"})
	}

	return echoCtx.JSON(http.StatusOK, map[string]any{
		"leaderboard": leaderboard,
		"offset":      offset,
		"limit":       limit,
	})
}
