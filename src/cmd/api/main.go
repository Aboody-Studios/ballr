package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	analysisapplication "github.com/Aboody-Studios/ballr/src/internal/match/application"
	matchhandlers "github.com/Aboody-Studios/ballr/src/internal/match/handlers/http"
	analysisinfrastructure "github.com/Aboody-Studios/ballr/src/internal/match/infrastructure"
	coachapplication "github.com/Aboody-Studios/ballr/src/internal/coach/application"
	coachhttp "github.com/Aboody-Studios/ballr/src/internal/coach/handlers/http"
	coachinfrastructure "github.com/Aboody-Studios/ballr/src/internal/coach/infrastructure"
	identityapplication "github.com/Aboody-Studios/ballr/src/internal/identity/application"
	identityhttp "github.com/Aboody-Studios/ballr/src/internal/identity/handlers/http"
	identityinfrastructure "github.com/Aboody-Studios/ballr/src/internal/identity/infrastructure"
	progressapplication "github.com/Aboody-Studios/ballr/src/internal/progress/application"
	progressdomain "github.com/Aboody-Studios/ballr/src/internal/progress/domain"
	progresshttp "github.com/Aboody-Studios/ballr/src/internal/progress/handlers/http"
	progressinfrastructure "github.com/Aboody-Studios/ballr/src/internal/progress/infrastructure"
	shareddelivery "github.com/Aboody-Studios/ballr/src/internal/shared/delivery"
	"github.com/Aboody-Studios/ballr/src/internal/shared/infrastructure"
	"github.com/Aboody-Studios/ballr/src/pkg/validator"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/joho/godotenv"
	echojwt "github.com/labstack/echo-jwt/v5"
	"github.com/labstack/echo/v5"
	echomw "github.com/labstack/echo/v5/middleware"
)

func main() {
	godotenv.Load()

	secretKey := os.Getenv("JWT_SECRET")
	rdb := infrastructure.InitiateRedis()
	db, dbErr := infrastructure.InitiatePostgres()
	if dbErr != nil {
		log.Fatal("database connection error")
	}

	if err := infrastructure.RunMigrations(db); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	// --- Identity ---
	postgresRepo := identityinfrastructure.PostgresUserRepo{DB: db}
	oauthProvider := identityinfrastructure.GoogleOAuthAPI{}
	refreshStore := identityinfrastructure.NewRedisRefreshTokenStore(rdb, 15*24*time.Hour)

	identityService := identityapplication.NewService(&postgresRepo, &oauthProvider, refreshStore)
	identityHandler := identityhttp.NewIdentityHandler(identityService)

	// --- Analysis ---
	cfg, cfgErr := config.LoadDefaultConfig(context.TODO())
	if cfgErr != nil {
		log.Fatalf("aws default config loading failed:%s", cfgErr)
	}
	s3Client := s3.NewFromConfig(cfg)
	storageRepo := analysisinfrastructure.NewStorageRepository(s3Client, os.Getenv("S3_BUCKET"))
	matchRepo := &analysisinfrastructure.PostgresMatchRepository{DB: db}
	uploadService := analysisapplication.NewUploadService(storageRepo, matchRepo)
	uploadHandler := matchhandlers.NewUploadHandler(uploadService)
	analysisRepo := &analysisinfrastructure.PostgresAnalysisRepository{DB: db}
	jobQueue := analysisinfrastructure.NewRedisJobQueue(rdb)
	analysisService := analysisapplication.NewAnalysisService(analysisRepo, matchRepo, jobQueue)
	analysisHandler := matchhandlers.NewAnalysisHandler(analysisService)
	analysisWorker := analysisinfrastructure.NewWorker(matchRepo, analysisRepo, jobQueue)
	workerCtx, stopWorker := context.WithCancel(context.Background())
	analysisWorker.Start(workerCtx)

	// --- Coach ---
	llmProvider, err := coachinfrastructure.NewLLMProvider()
	if err != nil {
		log.Fatalf("coach provider: %v", err)
	}
	coachAnalysisBridge := coachinfrastructure.NewCoachAnalysisBridge(matchRepo, analysisRepo)
	coachUserBridge := coachinfrastructure.NewCoachUserBridge(&postgresRepo)
	convRepo := coachinfrastructure.NewPostgresConversationRepository(db)
	coachService := coachapplication.NewService(llmProvider, coachAnalysisBridge, coachUserBridge, convRepo)
	coachHandler := coachhttp.NewCoachHandler(coachService)

	// --- Progress ---
	progressRepo := &progressinfrastructure.PostgresProgressRepository{DB: db}
	achievementRepo := &progressinfrastructure.PostgresAchievementRepository{DB: db}
	eventLogRepo := &progressinfrastructure.PostgresEventLogRepository{DB: db}
	leaderboardRepo := &progressinfrastructure.PostgresLeaderboardRepository{DB: db}
	gamificationService := progressapplication.NewGamificationService(progressRepo, achievementRepo, eventLogRepo, leaderboardRepo)
	progressHandler := progresshttp.NewProgressHandler(gamificationService)

	// --- Event Wiring ---
	eventPublisher := &gamificationAdapter{svc: gamificationService}
	uploadService.SetEventPublisher(eventPublisher)
	analysisService.SetEventPublisher(eventPublisher)
	analysisWorker.SetEventPublisher(eventPublisher)
	coachService.SetEventPublisher(eventPublisher)

	// --- Server ---
	echoServer := echo.New()
	echoServer.Validator = validator.New()
	echoServer.Use(echomw.RequestLogger())
	echoServer.Use(echomw.Recover())
	echoServer.Use(echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Authorization", "Content-Type"},
	}))

	secureGroup := echoServer.Group("/secure")
	echoJWTConfig := echojwt.Config{SigningKey: []byte(secretKey)}
	secureGroup.Use(echojwt.WithConfig(echoJWTConfig))
	secureGroup.Use(shareddelivery.RateLimiter(rdb))

	secureGroup.GET("/auth/me", identityHandler.GetProfileHandler)
	secureGroup.PUT("/auth/profile", identityHandler.CompleteProfileHandler)

	secureGroup.POST("/analysis/upload-url", uploadHandler.UploadURLHandler)
	secureGroup.GET("/analysis/status/:id", analysisHandler.GetAnalysisStatusHandler)
	secureGroup.GET("/analysis/report/:id", analysisHandler.GetAnalysisReportHandler)
	secureGroup.POST("/analysis/start", analysisHandler.StartAnalysisHandler)

	secureGroup.POST("/coach/chat", coachHandler.ChatHandler)
	secureGroup.POST("/coach/plan/generate", coachHandler.GeneratePlanHandler)
	secureGroup.POST("/coach/diet/generate", coachHandler.GenerateDietHandler)
	secureGroup.GET("/coach/history", coachHandler.GetHistoryHandler)

	secureGroup.GET("/progress/summary", progressHandler.GetProgressSummaryHandler)
	secureGroup.GET("/achievements/list", progressHandler.ListAchievementsHandler)
	secureGroup.GET("/leaderboard", progressHandler.GetLeaderboardHandler)

	// Public routes
	echoServer.GET("/auth/google", identityHandler.SignInWithGoogleHandler)
	echoServer.GET("/auth/google/callback", identityHandler.GoogleCallbackHandler)
	echoServer.POST("/auth/refresh", identityHandler.RefreshTokenHandler)

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = ":3000"
	}

	s := &http.Server{
		Addr:    port,
		Handler: echoServer,
	}

	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")

	stopWorker()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}

	if rdb != nil {
		rdb.Close()
	}

	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.Close()
	}

	log.Println("server stopped")
}

type gamificationAdapter struct {
	svc *progressapplication.GamificationService
}

func (a *gamificationAdapter) PublishEvent(ctx context.Context, userID string, eventType string, metadata map[string]interface{}) error {
	return a.svc.ProcessEvent(ctx, userID, progressdomain.EventType(eventType), progressdomain.EventMetadata(metadata))
}
