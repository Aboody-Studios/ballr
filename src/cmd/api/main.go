package main

import (
	"os"

	analysisapplication "github.com/Aboody-Studios/ballr/src/internal/analysis/application"
	analysisinfrastructure "github.com/Aboody-Studios/ballr/src/internal/analysis/infrastructure"
	analysishttp "github.com/Aboody-Studios/ballr/src/internal/analysis/interfaces/http"
	identityapplication "github.com/Aboody-Studios/ballr/src/internal/identity/application"
	identityhttp "github.com/Aboody-Studios/ballr/src/internal/identity/interfaces/http"
	"github.com/Aboody-Studios/ballr/src/internal/shared/infrastructure"
	sharedhttp "github.com/Aboody-Studios/ballr/src/internal/shared/interfaces/http"
	"github.com/Aboody-Studios/ballr/src/pkg/validator"

	"github.com/joho/godotenv"
	echojwt "github.com/labstack/echo-jwt/v5"
	"github.com/labstack/echo/v5"
	echomw "github.com/labstack/echo/v5/middleware"
)

func main() {
	// Initialize infrastructure dependencies
	rdb := infrastructure.InitiateRedis()
	godotenv.Load()
	secretKey := os.Getenv("JWT_SECRET")

	// initialize identity bounded context
	// TODO!: Wire up actual database repository when PostgreSQL is configured
	// for now, using this placeholder
	identityService := identityapplication.NewService(nil)
	identityHandler := identityhttp.NewIdentityHandler(identityService)

	// Initialize analysis bounded context
	storageRepo := analysisinfrastructure.NewStorageRepository()
	uploadService := analysisapplication.NewUploadService(storageRepo)
	// TODO!: Wire up matchRepo and analysisRepo when database is configured
	analysisService := analysisapplication.NewService(uploadService, nil, nil, storageRepo)
	analysisHandler := analysishttp.NewAnalysisHandler(analysisService)

	// Create Echo server
	echoServer := echo.New()
	echoServer.Validator = validator.New()

	// Configure secure group with JWT
	secureGroup := echoServer.Group("/secure")
	echoJWTStruct := echojwt.Config{SigningKey: []byte(secretKey)}
	echoServer.Use(echomw.RequestLogger())
	secureGroup.Use(echojwt.WithConfig(echoJWTStruct))
	secureGroup.Use(sharedhttp.RateLimiter(rdb))

	// Identity routes (public)
	echoServer.POST("/signup", identityHandler.SignUpHandler)
	// echoServer.POST("/login", identityHandler.LoginHandler) // TODO!: Implement login handler

	// Secure routes
	secureGroup.POST("/analysis/upload-url", analysisHandler.UploadURLHandler)
	secureGroup.GET("/analysis/status/:id", analysisHandler.GetAnalysisStatusHandler)
	secureGroup.GET("/analysis/report/:id", analysisHandler.GetAnalysisReportHandler)
	secureGroup.POST("/analysis/start", analysisHandler.StartAnalysisHandler)

	// Start server
	if err := echoServer.Start(":3000"); err != nil {
		echoServer.Logger.Error("failed to start server", "error", err)
	}
}
