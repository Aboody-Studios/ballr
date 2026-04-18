package handlers

import (
	"github.com/Aboody-Studios/ballr/backend/internal/aws"
	"github.com/Aboody-Studios/ballr/backend/internal/models"
	"github.com/labstack/echo/v5"
	"net/http"
	"strings"
)

func UploadURLHandler(context *echo.Context) error {
	var video models.Video
	if err := context.Bind(&video); err != nil {
		return err
	}

	//send video to another function that is responsible for checking that name ends in .mp4 (use HasSuffix() in strings)
	// and size isn't more than 500 mb (for example).
	// Then, there should be another function for generating a pre-signed url after the checking is successful.

	if !strings.HasSuffix(video.Name, ".mp4") {
		return context.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid file format"})
	}

	// a one minute video with 30 fps, 1080p resolution, and 5 Mbps bitrate has a size of approximately 37.5 MB

	if video.Size > 3375000000 {
		return context.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid file size"})
	}

	uploadURL, s3Err := aws.GenerateUploadURL(video)

	if s3Err != nil {
		return context.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	return context.JSON(http.StatusOK, map[string]string{"URL": uploadURL})
}
