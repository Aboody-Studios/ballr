package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	coachapplication "github.com/Aboody-Studios/ballr/src/internal/coach/application"
	coachhttp "github.com/Aboody-Studios/ballr/src/internal/coach/handlers/http"
	coachinfrastructure "github.com/Aboody-Studios/ballr/src/internal/coach/infrastructure"
	identityapplication "github.com/Aboody-Studios/ballr/src/internal/identity/application"
	identityhttp "github.com/Aboody-Studios/ballr/src/internal/identity/handlers/http"
	identityinfrastructure "github.com/Aboody-Studios/ballr/src/internal/identity/infrastructure"
	analysisapplication "github.com/Aboody-Studios/ballr/src/internal/match/application"
	matchhandlers "github.com/Aboody-Studios/ballr/src/internal/match/handlers/http"
	analysisinfrastructure "github.com/Aboody-Studios/ballr/src/internal/match/infrastructure"
	progressapplication "github.com/Aboody-Studios/ballr/src/internal/progress/application"
	progressdomain "github.com/Aboody-Studios/ballr/src/internal/progress/domain"
	progresshttp "github.com/Aboody-Studios/ballr/src/internal/progress/handlers/http"
	progressinfrastructure "github.com/Aboody-Studios/ballr/src/internal/progress/infrastructure"
	shareddelivery "github.com/Aboody-Studios/ballr/src/internal/shared/delivery"
	"github.com/Aboody-Studios/ballr/src/internal/shared/infrastructure"
	"github.com/Aboody-Studios/ballr/src/pkg/events"
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
	if secretKey == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	rdb := infrastructure.InitiateRedis()
	db, dbErr := infrastructure.InitiatePostgres()
	if dbErr != nil {
		log.Fatalf("database connection error: %v", dbErr)
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
	redisJobQueue := analysisinfrastructure.NewRedisJobQueue(rdb)
	analysisService := analysisapplication.NewAnalysisService(analysisRepo, matchRepo, redisJobQueue)
	analysisHandler := matchhandlers.NewAnalysisHandler(analysisService)
	analysisWorker := analysisinfrastructure.NewWorker(matchRepo, analysisRepo, redisJobQueue, storageRepo)
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
	eventPublisher := events.NewRedisPublisher(rdb)
	uploadService.SetEventPublisher(eventPublisher)
	analysisService.SetEventPublisher(eventPublisher)
	analysisWorker.SetEventPublisher(eventPublisher)
	coachService.SetEventPublisher(eventPublisher)

	eventConsumer := events.NewConsumer(rdb, events.DefaultStream, events.DefaultGroup, "api-server")
	eventConsumer.HandleFunc(events.EventAnalysisStart, func(ctx context.Context, event events.Event) error {
		matchID, ok := event.Metadata["match_id"].(string)
		if !ok {
			return fmt.Errorf("missing match_id")
		}
		videoURL, ok := event.Metadata["video_url"].(string)
		if !ok {
			return fmt.Errorf("missing video_url")
		}
		return analysisService.StartAnalysis(ctx, matchID, videoURL)
	})
	eventConsumer.HandleFunc(events.EventMatchUploaded, func(ctx context.Context, event events.Event) error {
		return gamificationService.ProcessEvent(ctx, event.UserID, progressdomain.EventType(event.Type), progressdomain.EventMetadata(event.Metadata))
	})
	eventConsumer.HandleFunc(events.EventAnalysisCompleted, func(ctx context.Context, e events.Event) error {
		return gamificationService.ProcessEvent(ctx, e.UserID, progressdomain.EventType(e.Type), progressdomain.EventMetadata(e.Metadata))
	})
	//TODO!: PublishEvent for coach interaction
	eventConsumer.HandleFunc(events.EventCoachInteraction, func(ctx context.Context, e events.Event) error {
		return gamificationService.ProcessEvent(ctx, e.UserID, progressdomain.EventType(e.Type), progressdomain.EventMetadata(e.Metadata))
	})

	consumerCtx, stopConsumer := context.WithCancel(context.Background())
	go func() {
		if err := eventConsumer.Start(consumerCtx); err != nil && err != context.Canceled {
			log.Printf("event consumer error: %v", err)
		}
	}()

	sweeper := analysisinfrastructure.Sweeper{
		MatchRepo:      matchRepo,
		EventPublisher: eventPublisher,
	}
	go sweeper.SweepStuckMatches(context.Background())

	// --- Server ---
	echoServer := echo.New()
	echoServer.Validator = validator.New()
	echoServer.Use(echomw.RequestLogger())
	echoServer.Use(echomw.Recover())
	corsOrigins := os.Getenv("CORS_ORIGINS")
	var allowedOrigins []string

	if corsOrigins == "" || corsOrigins == "*" {
		allowedOrigins = []string{"*"}
		log.Println("WARNING: CORS allows all origins. Set CORS_ORIGINS env var in production.")
	} else {
		allowedOrigins = strings.Split(corsOrigins, ",")
	}

	echoServer.Use(
		echomw.CORSWithConfig(echomw.CORSConfig{
			AllowOrigins: allowedOrigins,
			AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders: []string{"Authorization", "Content-Type"},
		}))

	secureGroup := echoServer.Group("/secure")
	echoJWTConfig := echojwt.Config{SigningKey: []byte(secretKey)}
	secureGroup.Use(echojwt.WithConfig(echoJWTConfig))
	secureGroup.Use(shareddelivery.RateLimiter(rdb))

	secureGroup.GET("/auth/me", identityHandler.GetProfileHandler)
	secureGroup.PUT("/auth/profile", identityHandler.CompleteProfileHandler)

	secureGroup.POST("/match/upload-url", uploadHandler.UploadURLHandler)
	secureGroup.GET("/match/analysis-status/:id", analysisHandler.GetAnalysisStatusHandler)
	secureGroup.GET("/match/analysis-report/:id", analysisHandler.GetAnalysisReportHandler)
	secureGroup.POST("/match/upload-success", uploadHandler.SuccessfulVideoUploadHandler)

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

	certFile := os.Getenv("TLS_CERT_FILE")
	keyFile := os.Getenv("TLS_KEY_FILE")

	s := &http.Server{
		Addr:    port,
		Handler: echoServer,
	}

	go func() {
		var serveErr error
		if certFile != "" && keyFile != "" {
			log.Printf("starting HTTPS server on %s", port)
			serveErr = s.ListenAndServeTLS(certFile, keyFile)
		} else {
			log.Printf("starting HTTP server on %s (set TLS_CERT_FILE and TLS_KEY_FILE for HTTPS)", port)
			serveErr = s.ListenAndServe()
		}
		if serveErr != nil && serveErr != http.ErrServerClosed {
			log.Printf("server error: %v", serveErr)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")

	stopConsumer()
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
