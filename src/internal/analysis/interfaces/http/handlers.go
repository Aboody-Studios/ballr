package http

import (
	"net/http"
	"strings"

	"github.com/Aboody-Studios/ballr/src/internal/analysis/application"
	"github.com/Aboody-Studios/ballr/src/internal/analysis/domain"
	"github.com/labstack/echo/v5"
)

// AnalysisHandler handles HTTP requests for the Analysis bounded context.
// This includes video upload, match processing status, and analysis results.
type AnalysisHandler struct {
	analysisService *application.Service
}

// NewAnalysisHandler creates a new analysis handler with the required service.
func NewAnalysisHandler(service *application.Service) *AnalysisHandler {
	return &AnalysisHandler{analysisService: service}
}

// UploadURLHandler generates pre-signed URLs for direct-to-S3 video uploads.
// This follows the workflow as stated in architecture docs:
// 1. Client requests upload URL (this endpoint)
// 2. Backend validates video metadata
// 3. Backend generates pre-signed S3 URL
// 4. Client uploads directly to S3
// 5. Client notifies backend to start analysis
// WARN!
func (h *AnalysisHandler) UploadURLHandler(context *echo.Context) error {
	var video domain.Video
	if err := context.Bind(&video); err != nil {
		return err
	}

	if !strings.HasSuffix(video.Name, ".mp4") {
		return context.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid file format"})
	}

	if video.Size > 3375000000 {
		return context.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid file size"})
	}

	uploadURL, s3Err := h.analysisService.GenerateUploadURL(context.Request().Context(), &video)

	if s3Err != nil {
		return context.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	return context.JSON(http.StatusOK, map[string]string{"URL": uploadURL})
}

// GetAnalysisStatusHandler retrieves the current processing status of a match analysis.
// Statuses: UPLOADING, PROCESSING, COMPLETED, FAILED
func (h *AnalysisHandler) GetAnalysisStatusHandler(c *echo.Context) error {
	matchID := c.Param("id")
	if matchID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "match ID is required"})
	}

	status, err := h.analysisService.GetAnalysisStatus(c.Request().Context(), matchID)
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
func (h *AnalysisHandler) GetAnalysisReportHandler(c *echo.Context) error {
	matchID := c.Param("id")
	if matchID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "match ID is required"})
	}

	report, err := h.analysisService.GetAnalysisReport(c.Request().Context(), matchID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "analysis report not found"})
	}

	return c.JSON(http.StatusOK, report)
}

// StartAnalysisHandler initiates the analysis pipeline after video upload completes.
// Pushes job to Redis/SQS queue for the analysis worker to process.
func (h *AnalysisHandler) StartAnalysisHandler(c *echo.Context) error {
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

	err := h.analysisService.StartAnalysis(c.Request().Context(), req.MatchID, req.ShirtNumber, req.Position, req.VideoURL)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to start analysis"})
	}

	return c.JSON(http.StatusAccepted, map[string]string{
		"match_id": req.MatchID,
		"status":   "PROCESSING",
	})
}
