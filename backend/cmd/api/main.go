package main

import (
	"github.com/Aboody-Studios/ballr/backend/internal/handlers"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func main() {
	echoServer := echo.New()
	echoServer.Use(middleware.RequestLogger())

	echoServer.POST("/analysis/upload-url", handlers.UploadURLHandler)

	if err := echoServer.Start(":3000"); err != nil {
		echoServer.Logger.Error("failed to start server", "error", err)
	}
}
