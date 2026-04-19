package main

import (
	"github.com/Aboody-Studios/ballr/backend/internal/handlers"
	"github.com/joho/godotenv"
	"github.com/labstack/echo-jwt/v5"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"os"
)

func main() {
	godotenv.Load()
	secretKey := os.Getenv("JWT_SECRET")
	echoServer := echo.New()
	echoJWTStruct := echojwt.Config{SigningKey: []byte(secretKey)}
	echoServer.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20.0)))
	echoServer.Use(middleware.RequestLogger())
	echoServer.POST("/signup", handlers.SignUpHandler)
	echoServer.POST("/login", handlers.LoginHandler)
	secureGroup := echoServer.Group("/secure")
	secureGroup.Use(echojwt.WithConfig(echoJWTStruct))
	secureGroup.POST("/analysis/upload-url", handlers.UploadURLHandler)

	if err := echoServer.Start(":3000"); err != nil {
		echoServer.Logger.Error("failed to start server", "error", err)
	}
}
