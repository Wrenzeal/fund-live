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
	authcache "github.com/RomaticDOG/fund/internal/cache"
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
	var issueRepo domain.IssueRepository
	var announcementRepo domain.AnnouncementRepository
	var dbInstance = database.GetDB() // Will be nil if not initialized
	var fundResolver *service.FundResolver
	var fundSectorStore *service.FundSectorStore
	var estimateCapabilityService *service.EstimateCapabilityService
	var analysisSnapshotStore *service.FundAnalysisSnapshotStore

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
		issueRepo = repository.NewPostgresIssueRepository(db)
		announcementRepo = repository.NewPostgresAnnouncementRepository(db)
		if err := service.SeedDefaultValuationProfiles(context.Background(), db); err != nil {
			log.Fatalf("❌ Failed to seed valuation profiles: %v", err)
		}
		if err := service.SeedDefaultFundSectorData(context.Background(), db); err != nil {
			log.Fatalf("❌ Failed to seed fund sector data: %v", err)
		}
		fundSectorStore = service.NewFundSectorStore(db)
		estimateCapabilityService = service.NewEstimateCapabilityService(db)
		analysisSnapshotStore = service.NewFundAnalysisSnapshotStore(db)
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
		issueRepo = repository.NewMemoryIssueRepository()
		announcementRepo = repository.NewMemoryAnnouncementRepository()
		log.Println("✅ Using in-memory storage (set STORAGE_MODE=postgres to use PostgreSQL)")
	}

	// Initialize cache repository
	cacheRepo := repository.NewMemoryCacheRepository(60*time.Second, 5*time.Minute)
	defaultQuoteSource := loadDefaultQuoteSource(fileCfg)
	overseasQuoteSource := appconfig.ResolveOverseasQuoteSource(fileCfg)

	// Initialize quote provider (Sina Finance)
	quoteProvider := adapter.NewSinaFinanceProvider()
	fundDataLoader := service.NewFundDataLoader(fundRepo)
	if fundSectorStore != nil {
		fundDataLoader.SetFundSectorStore(fundSectorStore)
	}

	// Initialize services
	valuationService := service.NewValuationService(fundRepo, quoteProvider, cacheRepo)
	valuationService.SetQuoteProvider(domain.QuoteSourceSina, quoteProvider)
	valuationService.SetQuoteProvider(domain.QuoteSourceTencent, adapter.NewTencentQuoteProvider())
	valuationService.SetOverseasQuoteProvider(adapter.NewQuoteProviderForSource(overseasQuoteSource))
	valuationService.SetDefaultQuoteSource(defaultQuoteSource)
	valuationService.SetFundDataLoader(fundDataLoader)
	log.Printf("🌎 QDII overseas quote source: %s", overseasQuoteSource)
	authConfig := loadAuthConfig(fileCfg)
	authConfig.DefaultQuoteSource = defaultQuoteSource
	authService := service.NewAuthService(userRepo, sessionRepo, authConfig)
	var authCodeStore *authcache.DragonflyAuthCodeStore
	if authConfig.EmailCodeEnabled {
		redisURL, keyPrefix := loadDragonflyConfig(fileCfg)
		store, storeErr := authcache.NewDragonflyAuthCodeStore(redisURL, keyPrefix)
		sender, senderErr := service.NewEmailSender(
			loadAuthEmailDriver(fileCfg),
			loadSMTPEmailConfig(fileCfg),
			loadAppEnvironment(),
		)
		switch {
		case len(strings.TrimSpace(authConfig.EmailCodeSecret)) < 32:
			log.Printf("⚠️ Email code login disabled: AUTH_EMAIL_CODE_SECRET must contain at least 32 characters")
		case storeErr != nil:
			log.Printf("⚠️ Email code login disabled: %v", storeErr)
		case senderErr != nil:
			log.Printf("⚠️ Email code login disabled: %v", senderErr)
		default:
			authCodeStore = store
			authService.SetEmailCodeDependencies(store, sender)
			availabilityCtx, cancel := context.WithTimeout(context.Background(), time.Second)
			if err := store.Ping(availabilityCtx); err != nil {
				log.Printf("⚠️ DragonFly unavailable; email code login will degrade until it recovers: %v", err)
			} else {
				log.Printf("✅ Email code login enabled with DragonFly key prefix %q", keyPrefix)
			}
			cancel()
		}
		if authCodeStore == nil && store != nil {
			_ = store.Close()
		}
	}
	userPreferenceService := service.NewUserPreferenceService(fundRepo, favoriteRepo, watchlistRepo, fundHoldingRepo, overrideRepo)
	issueService := service.NewIssueService(issueRepo)
	announcementService := service.NewAnnouncementService(announcementRepo)
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
		officialNavSync := service.NewOfficialNAVSyncService(fundRepo, fundHoldingRepo, favoriteRepo, watchlistRepo)
		officialNavSync.Start(context.Background())
		log.Println("🕚 Official NAV sync scheduled for 23:00 Asia/Shanghai")

		holdingsRefresh := service.NewFundHoldingsRefreshService(fundRepo)
		holdingsRefresh.SetFundSectorStore(fundSectorStore)
		holdingsRefresh.Start(context.Background())
		log.Println("🗓️ Monthly holdings refresh scheduled for day 1 at 01:00 Asia/Shanghai")
	}

	// Start background data collector
	// This ensures time series data is collected from market open (09:30)
	// regardless of frontend activity. Empty list = start idle until funds are tracked by requests.
	valuationService.StartBackgroundCollector(context.Background(), nil, 1*time.Minute)
	if estimateCapabilityService != nil {
		estimateCoverageScheduler := service.NewEstimateCoverageScheduler(estimateCapabilityService, valuationService, defaultQuoteSource)
		estimateCoverageScheduler.Start(context.Background())
		log.Println("📡 Estimate coverage scheduler started (all supported funds will gradually enter collector pools)")
		if analysisSnapshotStore != nil {
			analysisRefresh := service.NewFundAnalysisSnapshotRefreshService(
				estimateCapabilityService,
				service.NewFundAnalysisCoordinator(valuationService, fundRepo, fundResolver, fundSectorStore),
				analysisSnapshotStore,
			)
			analysisRefresh.Start(context.Background())
			log.Println("🧠 Fund analysis snapshot refresh scheduler started (nightly snapshot recompute)")
		}
	}

	// Initialize handlers
	fundHandler := handler.NewFundHandler(valuationService, fundRepo, fundResolver)
	fundHandler.SetTransientFundDataLoader(fundDataLoader)
	fundHandler.SetFundSectorStore(fundSectorStore)
	if analysisSnapshotStore != nil {
		fundHandler.SetAnalysisSnapshotStore(analysisSnapshotStore)
	}
	fundHandler.SetAnalysisCoordinator(service.NewFundAnalysisCoordinator(valuationService, fundRepo, fundResolver, fundSectorStore))
	if estimateCapabilityService != nil {
		fundHandler.SetAnalysisRankingCandidateProvider(estimateCapabilityService)
	}
	authHandler := handler.NewAuthHandler(authService, authConfig.CookieName, authConfig.CookieSecure)
	authHandler.SetGoogleWebClientID(authConfig.GoogleClientID)
	userHandler := handler.NewUserHandler(userPreferenceService, userRepo, defaultQuoteSource)
	issueHandler := handler.NewIssueHandler(issueService)
	announcementHandler := handler.NewAnnouncementHandler(announcementService)

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	if err := router.SetTrustedProxies([]string{"127.0.0.1", "::1"}); err != nil {
		log.Fatalf("❌ Failed to configure trusted proxies: %v", err)
	}
	allowedOrigins := loadCORSAllowedOrigins(fileCfg)

	// Apply middleware
	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())
	router.Use(middleware.CORS(allowedOrigins))
	router.Use(middleware.ResolveViewer(authService, authConfig.CookieName, defaultQuoteSource))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":           "ok",
			"timestamp":        time.Now().Unix(),
			"service":          "FundLive API",
			"version":          "2026.7.15-email-branding",
			"storage_mode":     storageMode,
			"email_code_login": emailCodeHealthStatus(c.Request.Context(), authConfig.EmailCodeEnabled, authService),
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.GET("/config", authHandler.Config)
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/google", authHandler.GoogleLogin)
			auth.POST("/email/start", authHandler.StartEmailCode)
			auth.POST("/email/verify", authHandler.VerifyEmailCode)

			authProtected := auth.Group("")
			authProtected.Use(middleware.RequireAuth(authService, authConfig.CookieName))
			authProtected.GET("/me", authHandler.Me)
			authProtected.POST("/logout", authHandler.Logout)
		}

		user := v1.Group("/user")
		user.Use(middleware.RequireAuth(authService, authConfig.CookieName))
		{
			user.GET("/watchlist/groups", userHandler.ListWatchlistGroups)
			user.GET("/quote-source", userHandler.GetQuoteSource)
			user.PUT("/quote-source", userHandler.UpdateQuoteSource)
			user.POST("/watchlist/groups", userHandler.CreateWatchlistGroup)
			user.PUT("/watchlist/groups/reorder", userHandler.ReorderWatchlistGroups)
			user.PUT("/watchlist/groups/:groupId", userHandler.UpdateWatchlistGroup)
			user.DELETE("/watchlist/groups/:groupId", userHandler.DeleteWatchlistGroup)
			user.POST("/watchlist/groups/:groupId/funds", userHandler.AddWatchlistFund)
			user.DELETE("/watchlist/groups/:groupId/funds/:fundId", userHandler.RemoveWatchlistFund)
			user.GET("/holdings", userHandler.ListFundHoldings)
			user.POST("/holdings", userHandler.CreateFundHolding)
			user.POST("/holdings/batch", userHandler.CreateFundHoldingsBatch)
			user.GET("/holdings/transactions", userHandler.ListFundHoldingTransactions)
			user.GET("/holdings/transactions/:transactionId", userHandler.GetFundHoldingTransactionDetail)
			user.GET("/holdings/transactions/:transactionId/rollback-preview", userHandler.PreviewFundHoldingTransactionRollback)
			user.POST("/holdings/transactions/:transactionId/rollback-apply", userHandler.ApplyFundHoldingTransactionRollback)
			user.POST("/holdings/transactions/:transactionId/void", userHandler.VoidFundHoldingTransaction)
			user.PUT("/holdings/:holdingId", userHandler.UpdateFundHolding)
			user.POST("/holdings/:holdingId/sell", userHandler.SellFundHolding)
			user.POST("/holdings/:holdingId/dividend", userHandler.RecordFundHoldingDividend)
			user.POST("/holdings/:holdingId/adjustment", userHandler.AdjustFundHoldingShares)
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
			fund.GET("/history/batch", fundHandler.GetHistoryBatch)
			fund.GET("/:id", fundHandler.GetFund)
			fund.GET("/:id/dashboard", fundHandler.GetDashboard)
			fund.GET("/:id/analysis", fundHandler.GetAnalysis)
			fund.GET("/:id/estimate", fundHandler.GetEstimate)
			fund.GET("/:id/holdings", fundHandler.GetHoldings)
			fund.GET("/:id/history", fundHandler.GetHistory)
			fund.GET("/:id/timeseries", fundHandler.GetTimeSeries)
		}

		history := v1.Group("/history")
		{
			history.GET("/fund", fundHandler.GetHistoryBatch)
		}

		market := v1.Group("/market")
		{
			market.GET("/status", fundHandler.GetMarketStatus)
			market.GET("/pricing-date", fundHandler.GetPricingDatePreview)
		}

		analysis := v1.Group("/analysis")
		{
			analysis.GET("/batch", fundHandler.GetAnalysisBatch)
			analysis.GET("/rankings", fundHandler.GetAnalysisRankings)
		}

		issues := v1.Group("/issues")
		{
			issues.GET("", issueHandler.List)
			issues.GET("/:id", issueHandler.Get)
			issuesProtected := issues.Group("")
			issuesProtected.Use(middleware.RequireAuth(authService, authConfig.CookieName))
			issuesProtected.POST("", issueHandler.Create)
		}

		announcements := v1.Group("/announcements")
		{
			announcements.GET("", announcementHandler.List)

			announcementsProtected := announcements.Group("")
			announcementsProtected.Use(middleware.RequireAuth(authService, authConfig.CookieName))
			announcementsProtected.GET("/unread", announcementHandler.ListUnread)
			announcementsProtected.POST("/:id/read", announcementHandler.MarkRead)

			announcements.GET("/:id", announcementHandler.Get)
		}

		admin := v1.Group("/admin")
		admin.Use(middleware.RequireAuth(authService, authConfig.CookieName))
		admin.Use(middleware.RequireAdmin())
		{
			admin.GET("/funds/classification-options", fundHandler.GetClassificationOptions)
			admin.GET("/funds/:id/classification", fundHandler.GetClassificationOverride)
			admin.PUT("/funds/:id/classification", fundHandler.UpdateClassificationOverride)
			admin.PUT("/issues/:id/status", issueHandler.UpdateStatus)
			admin.PUT("/issues/:id/reply", issueHandler.UpdateReply)
			admin.POST("/announcements", announcementHandler.Create)
			admin.POST("/announcements/import-changelog", announcementHandler.ImportChangelog)
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
		Addr:              port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      90 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("🚀 FundLive API server starting on port %s", port)
		log.Printf("📊 Available endpoints:")
		log.Printf("   GET /health - Health check")
		log.Printf("   POST /api/v1/auth/register - Register with email/password")
		log.Printf("   POST /api/v1/auth/login - Login with email/password")
		log.Printf("   POST /api/v1/auth/google - Login with Google ID token")
		log.Printf("   POST /api/v1/auth/email/start - Send email login code")
		log.Printf("   POST /api/v1/auth/email/verify - Verify email login code")
		log.Printf("   GET /api/v1/auth/me - Get current user")
		log.Printf("   POST /api/v1/auth/logout - Logout current session")
		log.Printf("   GET /api/v1/user/watchlist/groups - List grouped watchlists")
		log.Printf("   POST /api/v1/user/watchlist/groups - Create watchlist group")
		log.Printf("   PUT /api/v1/user/watchlist/groups/reorder - Reorder watchlist groups")
		log.Printf("   PUT /api/v1/user/watchlist/groups/:groupId - Update watchlist group name/description")
		log.Printf("   DELETE /api/v1/user/watchlist/groups/:groupId - Delete watchlist group")
		log.Printf("   POST /api/v1/user/watchlist/groups/:groupId/funds - Add fund to watchlist group")
		log.Printf("   DELETE /api/v1/user/watchlist/groups/:groupId/funds/:fundId - Remove fund from watchlist group")
		log.Printf("   GET /api/v1/user/holdings - List fund holding records")
		log.Printf("   POST /api/v1/user/holdings - Create fund holding record")
		log.Printf("   POST /api/v1/user/holdings/batch - Batch create fund holding records")
		log.Printf("   GET /api/v1/user/holdings/transactions - List recent holding activity")
		log.Printf("   GET /api/v1/user/holdings/transactions/:transactionId - Get holding activity detail")
		log.Printf("   GET /api/v1/user/holdings/transactions/:transactionId/rollback-preview - Preview holding activity rollback")
		log.Printf("   POST /api/v1/user/holdings/transactions/:transactionId/rollback-apply - Apply safe holding activity rollback")
		log.Printf("   POST /api/v1/user/holdings/transactions/:transactionId/void - Void holding activity")
		log.Printf("   POST /api/v1/user/holdings/:holdingId/sell - Record holding redemption")
		log.Printf("   POST /api/v1/user/holdings/:holdingId/dividend - Record holding dividend")
		log.Printf("   POST /api/v1/user/holdings/:holdingId/adjustment - Record holding share adjustment")
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
		log.Printf("   GET /api/v1/fund/:id/history?days=30 - Get official daily NAV history")
		log.Printf("   GET /api/v1/history/fund?fund_ids=<ids>&days=15 - Get daily NAV histories")
		log.Printf("   GET /api/v1/fund/history/batch?fund_ids=<ids>&days=15 - Get daily NAV histories")
		log.Printf("   GET /api/v1/fund/:id/timeseries - Get intraday time series")
		log.Printf("   GET /api/v1/market/status - Get A-Share market status")
		log.Printf("   GET /api/v1/market/pricing-date?trade_at=<RFC3339> - Preview holding pricing date")
		log.Printf("   GET /api/v1/issues - List public issues")
		log.Printf("   GET /api/v1/issues/:id - Get issue detail")
		log.Printf("   POST /api/v1/issues - Create issue (auth required)")
		log.Printf("   GET /api/v1/admin/funds/classification-options - List fund classification dictionaries (admin)")
		log.Printf("   GET /api/v1/admin/funds/:id/classification - Get fund classification override (admin)")
		log.Printf("   PUT /api/v1/admin/funds/:id/classification - Update fund classification override (admin)")
		log.Printf("   PUT /api/v1/admin/issues/:id/status - Update issue status (admin)")
		log.Printf("   PUT /api/v1/admin/issues/:id/reply - Update issue official reply (admin)")
		log.Printf("   GET /api/v1/announcements - List announcements")
		log.Printf("   GET /api/v1/announcements/:id - Get announcement detail")
		log.Printf("   GET /api/v1/announcements/unread - List unread announcements (auth)")
		log.Printf("   POST /api/v1/announcements/:id/read - Mark announcement as read (auth)")
		log.Printf("   POST /api/v1/admin/announcements - Create announcement (admin)")
		log.Printf("   POST /api/v1/admin/announcements/import-changelog - Import CHANGELOG announcements (admin)")
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
	if authCodeStore != nil {
		if err := authCodeStore.Close(); err != nil {
			log.Printf("Error closing DragonFly client: %v", err)
		}
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
		if fileCfg.Auth.AuthAttemptWindowMinutes > 0 {
			cfg.AuthAttemptWindow = time.Duration(fileCfg.Auth.AuthAttemptWindowMinutes) * time.Minute
		}
		if fileCfg.Auth.MaxPasswordFailures > 0 {
			cfg.MaxPasswordFailures = fileCfg.Auth.MaxPasswordFailures
		}
		if fileCfg.Auth.MaxRegisterFailures > 0 {
			cfg.MaxRegisterFailures = fileCfg.Auth.MaxRegisterFailures
		}
		if fileCfg.Auth.MaxGoogleLoginFailures > 0 {
			cfg.MaxGoogleLoginFailures = fileCfg.Auth.MaxGoogleLoginFailures
		}
		cfg.EmailCodeEnabled = fileCfg.Auth.EmailCodeEnabled
		if fileCfg.Auth.EmailCodeSecret != "" {
			cfg.EmailCodeSecret = fileCfg.Auth.EmailCodeSecret
		}
		if fileCfg.Auth.EmailCodeTTLMinutes > 0 {
			cfg.EmailCodeTTL = time.Duration(fileCfg.Auth.EmailCodeTTLMinutes) * time.Minute
		}
		if fileCfg.Auth.EmailResendCooldownSecs > 0 {
			cfg.EmailResendCooldown = time.Duration(fileCfg.Auth.EmailResendCooldownSecs) * time.Second
		}
		if fileCfg.Auth.MaxEmailSendsPerHour > 0 {
			cfg.MaxEmailSendsPerHour = fileCfg.Auth.MaxEmailSendsPerHour
		}
		if fileCfg.Auth.MaxIPEmailSendsPerHour > 0 {
			cfg.MaxIPEmailSendsPerHour = fileCfg.Auth.MaxIPEmailSendsPerHour
		}
		if fileCfg.Auth.MaxEmailCodeFailures > 0 {
			cfg.MaxEmailCodeFailures = fileCfg.Auth.MaxEmailCodeFailures
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
	if env := os.Getenv("AUTH_ATTEMPT_WINDOW_MINUTES"); env != "" {
		if minutes, err := strconv.Atoi(env); err == nil && minutes > 0 {
			cfg.AuthAttemptWindow = time.Duration(minutes) * time.Minute
		}
	}
	if env := os.Getenv("AUTH_MAX_PASSWORD_FAILURES"); env != "" {
		if attempts, err := strconv.Atoi(env); err == nil && attempts > 0 {
			cfg.MaxPasswordFailures = attempts
		}
	}
	if env := os.Getenv("AUTH_MAX_REGISTER_FAILURES"); env != "" {
		if attempts, err := strconv.Atoi(env); err == nil && attempts > 0 {
			cfg.MaxRegisterFailures = attempts
		}
	}
	if env := os.Getenv("AUTH_MAX_GOOGLE_LOGIN_FAILURES"); env != "" {
		if attempts, err := strconv.Atoi(env); err == nil && attempts > 0 {
			cfg.MaxGoogleLoginFailures = attempts
		}
	}
	if env := os.Getenv("AUTH_EMAIL_CODE_ENABLED"); env != "" {
		if enabled, err := strconv.ParseBool(env); err == nil {
			cfg.EmailCodeEnabled = enabled
		}
	}
	if env := os.Getenv("AUTH_EMAIL_CODE_SECRET"); env != "" {
		cfg.EmailCodeSecret = env
	}
	if env := os.Getenv("EMAIL_CODE_TTL"); env != "" {
		if ttl, err := time.ParseDuration(env); err == nil && ttl > 0 {
			cfg.EmailCodeTTL = ttl
		}
	}
	if env := os.Getenv("AUTH_RESEND_COOLDOWN"); env != "" {
		if cooldown, err := time.ParseDuration(env); err == nil && cooldown > 0 {
			cfg.EmailResendCooldown = cooldown
		}
	}
	if env := os.Getenv("AUTH_EMAIL_LIMIT_PER_HOUR"); env != "" {
		if limit, err := strconv.Atoi(env); err == nil && limit > 0 {
			cfg.MaxEmailSendsPerHour = limit
		}
	}
	if env := os.Getenv("AUTH_IP_LIMIT_PER_HOUR"); env != "" {
		if limit, err := strconv.Atoi(env); err == nil && limit > 0 {
			cfg.MaxIPEmailSendsPerHour = limit
		}
	}
	if env := os.Getenv("AUTH_CODE_MAX_FAILURES"); env != "" {
		if limit, err := strconv.Atoi(env); err == nil && limit > 0 {
			cfg.MaxEmailCodeFailures = limit
		}
	}
	cfg.ExposeEmailDevCode = cfg.EmailCodeEnabled && loadAppEnvironment() != "production" && loadAuthEmailDriver(fileCfg) == "dev"

	return cfg
}

func loadAppEnvironment() string {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	if value == "" {
		return "development"
	}
	return value
}

func loadDragonflyConfig(fileCfg *appconfig.Config) (string, string) {
	redisURL := "redis://127.0.0.1:16380/0"
	keyPrefix := "fundlive"
	if fileCfg != nil {
		if fileCfg.Cache.RedisURL != "" {
			redisURL = fileCfg.Cache.RedisURL
		}
		if fileCfg.Cache.KeyPrefix != "" {
			keyPrefix = fileCfg.Cache.KeyPrefix
		}
	}
	if env := strings.TrimSpace(os.Getenv("REDIS_URL")); env != "" {
		redisURL = env
	}
	if env := strings.TrimSpace(os.Getenv("FUNDLIVE_REDIS_KEY_PREFIX")); env != "" {
		keyPrefix = env
	}
	return redisURL, keyPrefix
}

func loadAuthEmailDriver(fileCfg *appconfig.Config) string {
	driver := "dev"
	if loadAppEnvironment() == "production" {
		driver = "smtp"
	}
	if fileCfg != nil && strings.TrimSpace(fileCfg.Auth.EmailDriver) != "" {
		driver = strings.ToLower(strings.TrimSpace(fileCfg.Auth.EmailDriver))
	}
	if env := strings.TrimSpace(os.Getenv("AUTH_EMAIL_DRIVER")); env != "" {
		driver = strings.ToLower(env)
	}
	return driver
}

func loadSMTPEmailConfig(fileCfg *appconfig.Config) service.SMTPEmailConfig {
	config := service.SMTPEmailConfig{
		Port:     587,
		From:     "fundlive@mail.wrenzeal.top",
		FromName: "FundLive",
		Security: "starttls",
		Timeout:  15 * time.Second,
	}
	if fileCfg != nil {
		config.Host = fileCfg.Auth.SMTPHost
		config.Port = firstPositive(fileCfg.Auth.SMTPPort, config.Port)
		config.Username = fileCfg.Auth.SMTPUsername
		config.Password = fileCfg.Auth.SMTPPassword
		config.From = firstNonEmpty(fileCfg.Auth.SMTPFrom, config.From)
		config.FromName = firstNonEmpty(fileCfg.Auth.SMTPFromName, config.FromName)
		config.Security = firstNonEmpty(fileCfg.Auth.SMTPSecurity, config.Security)
		if fileCfg.Auth.SMTPTimeoutSeconds > 0 {
			config.Timeout = time.Duration(fileCfg.Auth.SMTPTimeoutSeconds) * time.Second
		}
	}
	config.Host = envOrDefault("SMTP_HOST", config.Host)
	if env := os.Getenv("SMTP_PORT"); env != "" {
		if port, err := strconv.Atoi(env); err == nil && port > 0 {
			config.Port = port
		}
	}
	config.Username = envOrDefault("SMTP_USERNAME", config.Username)
	config.Password = envOrDefault("SMTP_PASSWORD", config.Password)
	config.From = envOrDefault("SMTP_FROM", config.From)
	config.FromName = envOrDefault("SMTP_FROM_NAME", config.FromName)
	config.Security = envOrDefault("SMTP_SECURITY", config.Security)
	if env := os.Getenv("SMTP_TIMEOUT"); env != "" {
		if timeout, err := time.ParseDuration(env); err == nil && timeout > 0 {
			config.Timeout = timeout
		}
	}
	return config
}

func emailCodeHealthStatus(ctx context.Context, configured bool, authService *service.AuthService) string {
	if !configured {
		return "disabled"
	}
	if authService.EmailCodeLoginAvailable(ctx) {
		return "ok"
	}
	return "degraded"
}

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func loadDefaultQuoteSource(fileCfg *appconfig.Config) domain.QuoteSource {
	source := domain.QuoteSourceSina
	if fileCfg != nil {
		source = domain.ResolveQuoteSource(domain.NormalizeQuoteSource(fileCfg.Quote.DefaultSource), source)
	}
	if env := os.Getenv("QUOTE_DEFAULT_SOURCE"); env != "" {
		source = domain.ResolveQuoteSource(domain.NormalizeQuoteSource(env), source)
	}
	return source
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
