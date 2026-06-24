package handlers

import (
	"net/http"

	"github.com/Aboody-Studios/ballr/src/internal/match/application"
	"github.com/Aboody-Studios/ballr/src/internal/match/domain"
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
func (uploadHandler *UploadHandler) PresignedPostObjHandler(echoCtx *echo.Context) error {
	var matchRequest application.MatchRequest
	if err := echoCtx.Bind(&matchRequest); err != nil {
		return err
	}

	jwt, err := delivery.ExtractToken(echoCtx)

	if err != nil {
		return echoCtx.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	presignedUpload, s3Err := uploadHandler.uploadService.StartUploadURLService(echoCtx.Request().Context(), &matchRequest, jwt.ID)

	if s3Err != nil {
		return echoCtx.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	return echoCtx.JSON(http.StatusOK, map[string]*domain.PresignedUpload{"presigned_upload_records": presignedUpload})
}

// UploadURLHandler is a compatibility wrapper that returns a simple URL for tests and older clients.
func (uploadHandler *UploadHandler) UploadURLHandler(echoCtx *echo.Context) error {
	var matchRequest application.MatchRequest
	if err := echoCtx.Bind(&matchRequest); err != nil {
		return err
	}

	jwt, err := delivery.ExtractToken(echoCtx)
	if err != nil {
		return echoCtx.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	presigned, s3Err := uploadHandler.uploadService.StartUploadURLService(echoCtx.Request().Context(), &matchRequest, jwt.ID)
	if s3Err != nil {
		return echoCtx.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	return echoCtx.JSON(http.StatusOK, map[string]string{"URL": presigned.URL})
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

func (uploadHandler *UploadHandler) SuccessfulVideoUploadHandler(echoCtx *echo.Context) error {
	if uploadHandler.uploadService == nil {
		return echoCtx.JSON(http.StatusInternalServerError, map[string]string{"error": "upload service not configured"})
	}

	var s3Success application.S3Success
	if err := echoCtx.Bind(&s3Success); err != nil {
		return echoCtx.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	var s3Key string
	if len(s3Success.Records) > 0 {
		s3Key = s3Success.Records[0].S3.Object.Key
	}
	if s3Key == "" {
		return echoCtx.JSON(http.StatusBadRequest, map[string]string{"error": "Missing S3 object key"})
	}

	ctx := echoCtx.Request().Context()

	if err := uploadHandler.uploadService.StartMatchProcessingWorflow(ctx, s3Key); err != nil {
		return echoCtx.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update match status"})

	}

	return echoCtx.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
