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
	var issueRepo domain.IssueRepository
	var announcementRepo domain.AnnouncementRepository
	var vipRepo domain.VIPRepository
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
		issueRepo = repository.NewPostgresIssueRepository(db)
		announcementRepo = repository.NewPostgresAnnouncementRepository(db)
		vipRepo = repository.NewPostgresVIPRepository(db)
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
		issueRepo = repository.NewMemoryIssueRepository()
		announcementRepo = repository.NewMemoryAnnouncementRepository()
		vipRepo = repository.NewMemoryVIPRepository()
		log.Println("✅ Using in-memory storage (set STORAGE_MODE=postgres to use PostgreSQL)")
	}

	// Initialize cache repository
	cacheRepo := repository.NewMemoryCacheRepository(60*time.Second, 5*time.Minute)
	defaultQuoteSource := loadDefaultQuoteSource(fileCfg)

	// Initialize quote provider (Sina Finance)
	quoteProvider := adapter.NewSinaFinanceProvider()
	fundDataLoader := service.NewFundDataLoader(fundRepo)

	// Initialize services
	valuationService := service.NewValuationService(fundRepo, quoteProvider, cacheRepo)
	valuationService.SetQuoteProvider(domain.QuoteSourceSina, quoteProvider)
	valuationService.SetQuoteProvider(domain.QuoteSourceTencent, adapter.NewTencentQuoteProvider())
	valuationService.SetOverseasQuoteProvider(adapter.NewTencentQuoteProvider())
	valuationService.SetDefaultQuoteSource(defaultQuoteSource)
	valuationService.SetFundDataLoader(fundDataLoader)
	authConfig := loadAuthConfig(fileCfg)
	authConfig.DefaultQuoteSource = defaultQuoteSource
	authService := service.NewAuthService(userRepo, sessionRepo, authConfig)
	userPreferenceService := service.NewUserPreferenceService(fundRepo, favoriteRepo, watchlistRepo, fundHoldingRepo, overrideRepo)
	issueService := service.NewIssueService(issueRepo)
	announcementService := service.NewAnnouncementService(announcementRepo)
	vipService := service.NewVIPService(vipRepo)
	wechatPayConfig := loadWeChatPayConfig(fileCfg)
	if wechatPayConfig.Enabled {
		wechatPayClient, err := service.NewWeChatPayClient(wechatPayConfig)
		if err != nil {
			log.Fatalf("❌ Failed to initialize WeChat Pay client: %v", err)
		}
		vipService.SetWeChatPayClient(wechatPayClient, wechatPayConfig)
		log.Println("💸 WeChat Pay payment channel enabled")
	}

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

		holdingsRefresh := service.NewFundHoldingsRefreshService(fundRepo)
		holdingsRefresh.Start(context.Background())
		log.Println("🗓️ Monthly holdings refresh scheduled for day 1 at 01:00 Asia/Shanghai")
	}

	// Start background data collector
	// This ensures time series data is collected from market open (09:30)
	// regardless of frontend activity. Empty list = start idle until funds are tracked by requests.
	valuationService.StartBackgroundCollector(context.Background(), nil, 1*time.Minute)

	// Initialize handlers
	fundHandler := handler.NewFundHandler(valuationService, fundRepo, fundResolver)
	fundHandler.SetTransientFundDataLoader(fundDataLoader)
	authHandler := handler.NewAuthHandler(authService, authConfig.CookieName, authConfig.CookieSecure)
	userHandler := handler.NewUserHandler(userPreferenceService, userRepo, defaultQuoteSource)
	issueHandler := handler.NewIssueHandler(issueService)
	announcementHandler := handler.NewAnnouncementHandler(announcementService)
	vipHandler := handler.NewVIPHandler(vipService)

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	allowedOrigins := loadCORSAllowedOrigins(fileCfg)

	// Apply middleware
	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())
	router.Use(middleware.CORS(allowedOrigins))
	router.Use(middleware.ResolveViewer(authService, authConfig.CookieName, defaultQuoteSource))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":       "ok",
			"timestamp":    time.Now().Unix(),
			"service":      "FundLive API",
			"version":      "2026.4.17",
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
			user.GET("/quote-source", userHandler.GetQuoteSource)
			user.PUT("/quote-source", userHandler.UpdateQuoteSource)
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
			admin.PUT("/issues/:id/status", issueHandler.UpdateStatus)
			admin.PUT("/issues/:id/reply", issueHandler.UpdateReply)
			admin.POST("/announcements", announcementHandler.Create)
			admin.POST("/announcements/import-changelog", announcementHandler.ImportChangelog)
		}

		vip := v1.Group("/vip")
		{
			vip.GET("/reports/:id", vipHandler.GetReport)
			vip.POST("/payments/wechat/notify", vipHandler.HandleWeChatPayNotify)

			vipProtected := vip.Group("")
			vipProtected.Use(middleware.RequireAuth(authService, authConfig.CookieName))
			vipProtected.GET("/membership", vipHandler.GetMembership)
			vipProtected.POST("/membership/preview-activate", vipHandler.PreviewActivateMembership)
			vipProtected.POST("/preview/reset", vipHandler.PreviewReset)
			vipProtected.GET("/quota", vipHandler.GetQuota)
			vipProtected.GET("/tasks", vipHandler.ListTasks)
			vipProtected.POST("/tasks", vipHandler.CreateTask)
			vipProtected.POST("/orders", vipHandler.CreateOrder)
			vipProtected.GET("/orders/:orderId", vipHandler.GetOrder)
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
		log.Printf("   GET /api/v1/issues - List public issues")
		log.Printf("   GET /api/v1/issues/:id - Get issue detail")
		log.Printf("   POST /api/v1/issues - Create issue (auth required)")
		log.Printf("   PUT /api/v1/admin/issues/:id/status - Update issue status (admin)")
		log.Printf("   PUT /api/v1/admin/issues/:id/reply - Update issue official reply (admin)")
		log.Printf("   GET /api/v1/announcements - List announcements")
		log.Printf("   GET /api/v1/announcements/:id - Get announcement detail")
		log.Printf("   GET /api/v1/announcements/unread - List unread announcements (auth)")
		log.Printf("   POST /api/v1/announcements/:id/read - Mark announcement as read (auth)")
		log.Printf("   POST /api/v1/admin/announcements - Create announcement (admin)")
		log.Printf("   POST /api/v1/admin/announcements/import-changelog - Import CHANGELOG announcements (admin)")
		log.Printf("   GET /api/v1/vip/reports/:id - Get VIP report detail")
		log.Printf("   GET /api/v1/vip/membership - Get VIP membership state")
		log.Printf("   POST /api/v1/vip/membership/preview-activate - Activate preview VIP membership")
		log.Printf("   POST /api/v1/vip/preview/reset - Reset VIP preview state")
		log.Printf("   GET /api/v1/vip/quota - Get VIP daily quota")
		log.Printf("   GET /api/v1/vip/tasks - List VIP tasks")
		log.Printf("   POST /api/v1/vip/tasks - Create VIP task")
		log.Printf("   POST /api/v1/vip/orders - Create VIP payment order")
		log.Printf("   GET /api/v1/vip/orders/:orderId - Get VIP payment order")
		log.Printf("   POST /api/v1/vip/payments/wechat/notify - Handle WeChat Pay callback")
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

func loadWeChatPayConfig(fileCfg *appconfig.Config) service.WeChatPayConfig {
	cfg := service.DefaultWeChatPayConfig()

	if fileCfg != nil {
		wechatCfg := fileCfg.Payment.WeChatPay
		cfg.Enabled = wechatCfg.Enabled
		if wechatCfg.AppID != "" {
			cfg.AppID = wechatCfg.AppID
		}
		if wechatCfg.MerchantID != "" {
			cfg.MerchantID = wechatCfg.MerchantID
		}
		if wechatCfg.MerchantCertificateSerialNo != "" {
			cfg.MerchantCertificateSerialNo = wechatCfg.MerchantCertificateSerialNo
		}
		if wechatCfg.MerchantPrivateKeyPath != "" {
			cfg.MerchantPrivateKeyPath = wechatCfg.MerchantPrivateKeyPath
		}
		if wechatCfg.APIV3Key != "" {
			cfg.APIV3Key = wechatCfg.APIV3Key
		}
		if wechatCfg.NotifyURL != "" {
			cfg.NotifyURL = wechatCfg.NotifyURL
		}
		if wechatCfg.PlatformCertificatePath != "" {
			cfg.PlatformCertificatePath = wechatCfg.PlatformCertificatePath
		}
		if wechatCfg.PlatformPublicKeyPath != "" {
			cfg.PlatformPublicKeyPath = wechatCfg.PlatformPublicKeyPath
		}
		if wechatCfg.PlatformSerialNo != "" {
			cfg.PlatformSerialNo = wechatCfg.PlatformSerialNo
		}
	}

	if env := os.Getenv("WECHAT_PAY_ENABLED"); env != "" {
		if enabled, err := strconv.ParseBool(env); err == nil {
			cfg.Enabled = enabled
		}
	}
	if env := os.Getenv("WECHAT_PAY_APP_ID"); env != "" {
		cfg.AppID = env
	}
	if env := os.Getenv("WECHAT_PAY_MERCHANT_ID"); env != "" {
		cfg.MerchantID = env
	}
	if env := os.Getenv("WECHAT_PAY_MERCHANT_CERTIFICATE_SERIAL_NO"); env != "" {
		cfg.MerchantCertificateSerialNo = env
	}
	if env := os.Getenv("WECHAT_PAY_MERCHANT_PRIVATE_KEY_PATH"); env != "" {
		cfg.MerchantPrivateKeyPath = env
	}
	if env := os.Getenv("WECHAT_PAY_API_V3_KEY"); env != "" {
		cfg.APIV3Key = env
	}
	if env := os.Getenv("WECHAT_PAY_NOTIFY_URL"); env != "" {
		cfg.NotifyURL = env
	}
	if env := os.Getenv("WECHAT_PAY_PLATFORM_CERTIFICATE_PATH"); env != "" {
		cfg.PlatformCertificatePath = env
	}
	if env := os.Getenv("WECHAT_PAY_PLATFORM_PUBLIC_KEY_PATH"); env != "" {
		cfg.PlatformPublicKeyPath = env
	}
	if env := os.Getenv("WECHAT_PAY_PLATFORM_SERIAL_NO"); env != "" {
		cfg.PlatformSerialNo = env
	}

	return cfg
}
