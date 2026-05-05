package handlers

import (
	"net/http"

	"github.com/Aboody-Studios/ballr/src/internal/match/application"
	"github.com/Aboody-Studios/ballr/src/internal/shared/delivery"
	"github.com/labstack/echo/v5"
)

// AnalysisHandler handles HTTP requests for the Analysis bounded context.
// This includes video upload, match processing status, and analysis results.
type AnalysisHandler struct {
	analysisService *application.AnalysisService
}

type UploadHandler struct {
	uploadService *application.UploadService
}

func NewAnalysisHandler(service *application.AnalysisService) *AnalysisHandler {
	return &AnalysisHandler{analysisService: service}
}

func NewUploadHandler(service *application.UploadService) *UploadHandler {
	return &UploadHandler{uploadService: service}
}

// UploadURLHandler generates pre-signed URLs for direct-to-S3 video uploads.
// This follows the workflow as stated in architecture docs:
// 1. Client requests upload URL (this endpoint)
// 2. Backend validates video metadata
// 3. Backend generates pre-signed S3 URL
// 4. Client uploads directly to S3
// 5. Client notifies backend to start analysis
// WARN!
func (uploadHandler *UploadHandler) UploadURLHandler(echoCtx *echo.Context) error {
	var matchRequest application.MatchRequest
	if err := echoCtx.Bind(&matchRequest); err != nil {
		return err
	}

	jwt, err := delivery.ExtractToken(echoCtx)

	if err != nil {
		return echoCtx.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	uploadURL, s3Err := uploadHandler.uploadService.StartUploadURLService(echoCtx.Request().Context(), &matchRequest, jwt.ID)

	if s3Err != nil {
		return echoCtx.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	return echoCtx.JSON(http.StatusOK, map[string]string{"URL": uploadURL})
}

// GetAnalysisStatusHandler retrieves the current processing status of a match analysis.
// Statuses: UPLOADING, PROCESSING, COMPLETED, FAILED
func (analysisHandler *AnalysisHandler) GetAnalysisStatusHandler(c *echo.Context) error {
	matchID := c.Param("id")
	if matchID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "match ID is required"})
	}

	status, err := analysisHandler.analysisService.GetAnalysisStatus(c.Request().Context(), matchID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "analysis not found"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"match_id": matchID,
		"status":   status,
	})
}

// GetAnalysisReportHandler retrieves the complete analysis results for a match.
// Returns the structured JSON with tracking data, heatmaps, events, and insights.
func (analysisHandler *AnalysisHandler) GetAnalysisReportHandler(c *echo.Context) error {
	matchID := c.Param("id")
	if matchID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "match ID is required"})
	}

	report, err := analysisHandler.analysisService.GetAnalysisReport(c.Request().Context(), matchID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "analysis report not found"})
	}

	return c.JSON(http.StatusOK, report)
}

// StartAnalysisHandler initiates the analysis pipeline after video upload completes.
// Pushes job to Redis/SQS queue for the analysis worker to process.
func (analysisHandler *AnalysisHandler) StartAnalysisHandler(c *echo.Context) error {
	var req struct {
		MatchID     string `json:"match_id"`
		ShirtNumber int    `json:"shirt_number"`
		Position    string `json:"position"`
		VideoURL    string `json:"video_url"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid JSON format"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Validation failed"})
	}

	claims, err := delivery.ExtractToken(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	err = analysisHandler.analysisService.StartAnalysis(c.Request().Context(), req.MatchID, claims.ID, req.ShirtNumber, req.Position, req.VideoURL)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to start analysis"})
	}

	return c.JSON(http.StatusAccepted, map[string]string{
		"match_id": req.MatchID,
		"status":   "PROCESSING",
	})
}

func (ah *AnalysisHandler) SuccessfulVideoUploadHandler(echoCtx *echo.Context) error {
	var s3Success application.S3Success
	if err := echoCtx.Bind(&s3Success); err != nil {
		return err
	}

	//ctx := echoCtx.Request().Context()
	//TODO!: Persist success to db uploading column

	return nil
}
