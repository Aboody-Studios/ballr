package main

import (
	"os"

	"github.com/Aboody-Studios/ballr/backend/internal/handlers"
	"github.com/Aboody-Studios/ballr/backend/internal/infrastructure"
	"github.com/Aboody-Studios/ballr/backend/internal/middleware"
	"github.com/joho/godotenv"
	"github.com/labstack/echo-jwt/v5"
	"github.com/labstack/echo/v5"
	echomw "github.com/labstack/echo/v5/middleware"
	"github.com/go-playground/validator/v10"
)

// validation needs to be moved to a separate file
type playVal struct {
	validator *validator.Validate
}

func (v *playVal) Validate(i any) error {
	if err := v.validator.Struct(i); err != nil{
		return err
	}
	return nil
}

func main() {
	rdb := infrastructure.InitiateRedis()
	godotenv.Load()
	secretKey := os.Getenv("JWT_SECRET")
	echoServer := echo.New()
	echoServer.Validator = &playVal{validator: validator.New()}
	secureGroup := echoServer.Group("/secure")
	echoJWTStruct := echojwt.Config{SigningKey: []byte(secretKey)}
	echoServer.Use(echomw.RequestLogger())
	secureGroup.Use(echojwt.WithConfig(echoJWTStruct))
	secureGroup.Use(middleware.RateLimiter(rdb))
	echoServer.POST("/signup", handlers.SignUpHandler)
	//echoServer.POST("/login", handlers.LoginHandler)
	secureGroup.POST("/analysis/upload-url", handlers.UploadURLHandler)

	if err := echoServer.Start(":3000"); err != nil {
		echoServer.Logger.Error("failed to start server", "error", err)
	}
}
