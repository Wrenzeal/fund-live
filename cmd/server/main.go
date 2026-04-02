// Package main is the entry point for the FundLive backend server.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/RomaticDOG/fund/internal/adapter"
	"github.com/RomaticDOG/fund/internal/appconfig"
	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/handler"
	"github.com/RomaticDOG/fund/internal/middleware"
	"github.com/RomaticDOG/fund/internal/repository"
	"github.com/RomaticDOG/fund/internal/service"
	"github.com/gin-gonic/gin"
)

func main() {
	fileCfg, err := appconfig.LoadConfig()
	if err != nil {
		log.Fatalf("❌ Failed to load startup config: %v", err)
	}
	if fileCfg != nil {
		log.Printf("📄 Loaded startup config: %s", fileCfg.Path)
	}

	// Determine storage mode: "postgres" or "memory"
	storageMode := "memory"
	if fileCfg != nil && fileCfg.Storage.Mode != "" {
		storageMode = fileCfg.Storage.Mode
	}
	if envMode := os.Getenv("STORAGE_MODE"); envMode != "" {
		storageMode = envMode
	}

	var fundRepo domain.FundRepository
	var userRepo domain.UserRepository
	var sessionRepo domain.UserSessionRepository
	var favoriteRepo domain.UserFavoriteRepository
	var watchlistRepo domain.UserWatchlistRepository
	var fundHoldingRepo domain.UserFundHoldingRepository
	var overrideRepo domain.UserHoldingOverrideRepository
	var dbInstance = database.GetDB() // Will be nil if not initialized
	var fundResolver *service.FundResolver

	if storageMode == "postgres" {
		// Initialize PostgreSQL database
		log.Println("🔧 Initializing PostgreSQL database...")
		cfg := database.DefaultConfig()
		db, err := database.InitDB(cfg, database.AllModels()...)
		if err != nil {
			log.Fatalf("❌ Failed to initialize database: %v\n   Hint: verify the PostgreSQL connection configured in fundlive.yaml or set STORAGE_MODE=memory", err)
		}
		dbInstance = db

		// Use PostgreSQL repository
		fundRepo = repository.NewPostgresFundRepository(db)
		userStore := repository.NewPostgresUserRepository(db)
		userRepo = userStore
		sessionRepo = userStore
		favoriteRepo = userStore
		watchlistRepo = userStore
		fundHoldingRepo = userStore
		overrideRepo = userStore
		if err := service.SeedDefaultValuationProfiles(context.Background(), db); err != nil {
			log.Fatalf("❌ Failed to seed valuation profiles: %v", err)
		}
		log.Println("✅ Using PostgreSQL storage")
	} else {
		// Use in-memory repository (for development without Docker)
		fundRepo = repository.NewMemoryFundRepository()
		userStore := repository.NewMemoryUserRepository()
		userRepo = userStore
		sessionRepo = userStore
		favoriteRepo = userStore
		watchlistRepo = userStore
		fundHoldingRepo = userStore
		overrideRepo = userStore
		log.Println("✅ Using in-memory storage (set STORAGE_MODE=postgres to use PostgreSQL)")
	}

	// Initialize cache repository
	cacheRepo := repository.NewMemoryCacheRepository(60*time.Second, 5*time.Minute)

	// Initialize quote provider (Sina Finance)
	quoteProvider := adapter.NewSinaFinanceProvider()
	fundDataLoader := service.NewFundDataLoader(fundRepo)

	// Initialize services
	valuationService := service.NewValuationService(fundRepo, quoteProvider, cacheRepo)
	valuationService.SetFundDataLoader(fundDataLoader)
	authConfig := loadAuthConfig(fileCfg)
	authService := service.NewAuthService(userRepo, sessionRepo, authConfig)
	userPreferenceService := service.NewUserPreferenceService(fundRepo, favoriteRepo, watchlistRepo, fundHoldingRepo, overrideRepo)

	// Initialize fund resolver for feeder fund -> ETF resolution
	// This enables transparent access to ETF holdings for feeder funds (联接基金)
	if dbInstance != nil {
		fundResolver = service.NewFundResolver(dbInstance, fundRepo)
		fundResolver.SetFundDataLoader(fundDataLoader)
		valuationService.SetFundResolver(fundResolver)
		valuationService.SetValuationProfileStore(service.NewValuationProfileStore(dbInstance))
		log.Println("🔗 Fund resolver enabled for feeder fund resolution")
	}

	if dbInstance != nil {
		officialNavSync := service.NewOfficialNAVSyncService(fundRepo, fundHoldingRepo)
		officialNavSync.Start(context.Background())
		log.Println("🕚 Official NAV sync scheduled for 23:00 Asia/Shanghai")
	}

	// Start background data collector
	// This ensures time series data is collected from market open (09:30)
	// regardless of frontend activity. Empty list = start idle until funds are tracked by requests.
	valuationService.StartBackgroundCollector(context.Background(), nil, 1*time.Minute)

	// Initialize handlers
	fundHandler := handler.NewFundHandler(valuationService, fundRepo, fundResolver)
	fundHandler.SetTransientFundDataLoader(fundDataLoader)
	authHandler := handler.NewAuthHandler(authService, authConfig.CookieName, authConfig.CookieSecure)
	userHandler := handler.NewUserHandler(userPreferenceService)

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	allowedOrigins := loadCORSAllowedOrigins(fileCfg)

	// Apply middleware
	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())
	router.Use(middleware.CORS(allowedOrigins))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":       "ok",
			"timestamp":    time.Now().Unix(),
			"service":      "FundLive API",
			"version":      "2026.4.1",
			"storage_mode": storageMode,
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/google", authHandler.GoogleLogin)

			authProtected := auth.Group("")
			authProtected.Use(middleware.RequireAuth(authService, authConfig.CookieName))
			authProtected.GET("/me", authHandler.Me)
			authProtected.POST("/logout", authHandler.Logout)
		}

		user := v1.Group("/user")
		user.Use(middleware.RequireAuth(authService, authConfig.CookieName))
		{
			user.GET("/watchlist/groups", userHandler.ListWatchlistGroups)
			user.POST("/watchlist/groups", userHandler.CreateWatchlistGroup)
			user.DELETE("/watchlist/groups/:groupId", userHandler.DeleteWatchlistGroup)
			user.POST("/watchlist/groups/:groupId/funds", userHandler.AddWatchlistFund)
			user.DELETE("/watchlist/groups/:groupId/funds/:fundId", userHandler.RemoveWatchlistFund)
			user.GET("/holdings", userHandler.ListFundHoldings)
			user.POST("/holdings", userHandler.CreateFundHolding)
			user.DELETE("/holdings/:holdingId", userHandler.DeleteFundHolding)
			user.GET("/favorites", userHandler.ListFavoriteFunds)
			user.POST("/favorites", userHandler.AddFavoriteFund)
			user.DELETE("/favorites/:fundId", userHandler.RemoveFavoriteFund)
			user.GET("/funds/:fundId/holding-overrides", userHandler.GetHoldingOverrides)
			user.PUT("/funds/:fundId/holding-overrides", userHandler.ReplaceHoldingOverrides)
		}

		fund := v1.Group("/fund")
		{
			fund.GET("/search", fundHandler.Search)
			fund.GET("/:id", fundHandler.GetFund)
			fund.GET("/:id/estimate", fundHandler.GetEstimate)
			fund.GET("/:id/holdings", fundHandler.GetHoldings)
			fund.GET("/:id/timeseries", fundHandler.GetTimeSeries)
		}

		market := v1.Group("/market")
		{
			market.GET("/status", fundHandler.GetMarketStatus)
			market.GET("/pricing-date", fundHandler.GetPricingDatePreview)
		}
	}

	// Server configuration
	port := appconfig.NormalizePort("")
	if fileCfg != nil {
		port = appconfig.NormalizePort(fileCfg.Server.Port)
	}
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = appconfig.NormalizePort(envPort)
	}
	server := &http.Server{
		Addr:    port,
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		log.Printf("🚀 FundLive API server starting on port %s", port)
		log.Printf("📊 Available endpoints:")
		log.Printf("   GET /health - Health check")
		log.Printf("   POST /api/v1/auth/register - Register with email/password")
		log.Printf("   POST /api/v1/auth/login - Login with email/password")
		log.Printf("   POST /api/v1/auth/google - Login with Google ID token")
		log.Printf("   GET /api/v1/auth/me - Get current user")
		log.Printf("   POST /api/v1/auth/logout - Logout current session")
		log.Printf("   GET /api/v1/user/watchlist/groups - List grouped watchlists")
		log.Printf("   POST /api/v1/user/watchlist/groups - Create watchlist group")
		log.Printf("   DELETE /api/v1/user/watchlist/groups/:groupId - Delete watchlist group")
		log.Printf("   POST /api/v1/user/watchlist/groups/:groupId/funds - Add fund to watchlist group")
		log.Printf("   DELETE /api/v1/user/watchlist/groups/:groupId/funds/:fundId - Remove fund from watchlist group")
		log.Printf("   GET /api/v1/user/holdings - List fund holding records")
		log.Printf("   POST /api/v1/user/holdings - Create fund holding record")
		log.Printf("   DELETE /api/v1/user/holdings/:holdingId - Delete fund holding record")
		log.Printf("   GET /api/v1/user/favorites - List favorite funds")
		log.Printf("   POST /api/v1/user/favorites - Add favorite fund")
		log.Printf("   DELETE /api/v1/user/favorites/:fundId - Remove favorite fund")
		log.Printf("   GET /api/v1/user/funds/:fundId/holding-overrides - List holding overrides")
		log.Printf("   PUT /api/v1/user/funds/:fundId/holding-overrides - Replace holding overrides")
		log.Printf("   GET /api/v1/fund/search?q=<query> - Search funds")
		log.Printf("   GET /api/v1/fund/:id - Get fund info")
		log.Printf("   GET /api/v1/fund/:id/estimate - Get real-time estimate")
		log.Printf("   GET /api/v1/fund/:id/holdings - Get fund holdings")
		log.Printf("   GET /api/v1/fund/:id/timeseries - Get intraday time series")
		log.Printf("   GET /api/v1/market/status - Get A-Share market status")
		log.Printf("   GET /api/v1/market/pricing-date?trade_at=<RFC3339> - Preview holding pricing date")
		log.Printf("📈 Sample fund codes: 005827, 003095, 320007")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("🛑 Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// Close database connection
	if err := database.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}

	log.Println("👋 Server exited gracefully")
}

func loadAuthConfig(fileCfg *appconfig.Config) service.AuthConfig {
	cfg := service.DefaultAuthConfig()

	if fileCfg != nil {
		if fileCfg.Auth.CookieName != "" {
			cfg.CookieName = fileCfg.Auth.CookieName
		}
		if fileCfg.Auth.SessionTTLHours > 0 {
			cfg.SessionTTL = time.Duration(fileCfg.Auth.SessionTTLHours) * time.Hour
		}
		cfg.CookieSecure = fileCfg.Auth.CookieSecure
		if fileCfg.Auth.GoogleClientID != "" {
			cfg.GoogleClientID = fileCfg.Auth.GoogleClientID
		}
	}

	if env := os.Getenv("AUTH_COOKIE_NAME"); env != "" {
		cfg.CookieName = env
	}
	if env := os.Getenv("AUTH_SESSION_TTL_HOURS"); env != "" {
		if hours, err := strconv.Atoi(env); err == nil && hours > 0 {
			cfg.SessionTTL = time.Duration(hours) * time.Hour
		}
	}
	if env := os.Getenv("AUTH_COOKIE_SECURE"); env != "" {
		if secure, err := strconv.ParseBool(env); err == nil {
			cfg.CookieSecure = secure
		}
	}
	if env := os.Getenv("GOOGLE_CLIENT_ID"); env != "" {
		cfg.GoogleClientID = env
	}

	return cfg
}

func loadCORSAllowedOrigins(fileCfg *appconfig.Config) []string {
	var origins []string

	if fileCfg != nil {
		origins = append(origins, fileCfg.Server.AllowedOrigins...)
	}
	if env := os.Getenv("CORS_ALLOWED_ORIGINS"); env != "" {
		origins = strings.Split(env, ",")
	}

	seen := make(map[string]struct{}, len(origins))
	result := make([]string, 0, len(origins))
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		if _, ok := seen[origin]; ok {
			continue
		}
		seen[origin] = struct{}{}
		result = append(result, origin)
	}
	return result
}
