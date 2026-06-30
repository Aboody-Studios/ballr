package handlers

import (
	"net/http"

	"github.com/Aboody-Studios/ballr/src/internal/coach/application"
	"github.com/Aboody-Studios/ballr/src/internal/coach/domain"
	"github.com/Aboody-Studios/ballr/src/internal/shared/delivery"
	"github.com/labstack/echo/v5"
)

type CoachHandler struct {
	CoachService application.CoachService
}

func NewCoachHandler(coachService application.CoachService) CoachHandler {
	return CoachHandler{
		CoachService: coachService,
	}
}

func (ch CoachHandler) NewUserMessageHandler(echoCtx *echo.Context) error {
	claims, err := delivery.ExtractToken(echoCtx)
	if err != nil {
		return echoCtx.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	var req domain.UserMessage

	if err := echoCtx.Bind(&req); err != nil {
		return echoCtx.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	response, resErr := ch.CoachService.ResponseGenerationOrchestrator(echoCtx.Request().Context(), claims.ID, req.Text)
	if resErr != nil {
		return echoCtx.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate AI response"})
	}

	return echoCtx.JSON(http.StatusOK, map[string]string{"response": response})
}
