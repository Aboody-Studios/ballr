package main

import (
	"log"
	"os"

	analysisapplication "github.com/Aboody-Studios/ballr/src/internal/analysis/application"
	analysishttp "github.com/Aboody-Studios/ballr/src/internal/analysis/handlers/http"
	analysisinfrastructure "github.com/Aboody-Studios/ballr/src/internal/analysis/infrastructure"
	identityapplication "github.com/Aboody-Studios/ballr/src/internal/identity/application"
	identityhttp "github.com/Aboody-Studios/ballr/src/internal/identity/handlers/http"
	identityinfrastructure "github.com/Aboody-Studios/ballr/src/internal/identity/infrastructure"
	"github.com/Aboody-Studios/ballr/src/internal/shared/infrastructure"
	sharedhttp "github.com/Aboody-Studios/ballr/src/internal/shared/interfaces/http"
	"github.com/Aboody-Studios/ballr/src/pkg/validator"
	"github.com/joho/godotenv"
	echojwt "github.com/labstack/echo-jwt/v5"
	"github.com/labstack/echo/v5"
	echomw "github.com/labstack/echo/v5/middleware"
)

func main() {

	rdb := infrastructure.InitiateRedis()
	godotenv.Load()
	secretKey := os.Getenv("JWT_SECRET")
	db, dbErr := infrastructure.InitiatePostgres()
	if dbErr != nil {
		log.Fatal("database connection error")
	}

	postgresRepo := identityinfrastructure.PostgresUserRepo{DB: db}
	oauthProvider := identityinfrastructure.GoogleOAuthAPI{}

	identityService := identityapplication.NewService(&postgresRepo, &oauthProvider)
	identityHandler := identityhttp.NewIdentityHandler(identityService)

	storageRepo := analysisinfrastructure.NewStorageRepository()
	uploadService := analysisapplication.NewUploadService(storageRepo)
	// TODO!: Wire up matchRepo and analysisRepo when database is configured
	analysisService := analysisapplication.NewService(uploadService, nil, nil, storageRepo)
	analysisHandler := analysishttp.NewAnalysisHandler(analysisService)

	echoServer := echo.New()
	echoServer.Validator = validator.New()
	echoServer.Use(echomw.RequestLogger())

	secureGroup := echoServer.Group("/secure")
	echoJWTStruct := echojwt.Config{SigningKey: []byte(secretKey)}
	secureGroup.Use(echojwt.WithConfig(echoJWTStruct))
	secureGroup.Use(sharedhttp.RateLimiter(rdb))

	// Identity routes (public)
	echoServer.POST("/signup", identityHandler.SignUpHandler)
	// echoServer.POST("/login", identityHandler.LoginHandler) // TODO!: Implement login handler

	secureGroup.POST("/analysis/upload-url", analysisHandler.UploadURLHandler)
	secureGroup.GET("/analysis/status/:id", analysisHandler.GetAnalysisStatusHandler)
	secureGroup.GET("/analysis/report/:id", analysisHandler.GetAnalysisReportHandler)
	secureGroup.POST("/analysis/start", analysisHandler.StartAnalysisHandler)

	if err := echoServer.Start(":3000"); err != nil {
		echoServer.Logger.Error("failed to start server", "error", err)
	}
}
