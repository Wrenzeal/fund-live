// Package main provides a CLI tool for crawling fund data from Eastmoney.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/RomaticDOG/fund/internal/crawler"
	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/repository"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func main() {
	// Command line flags
	codes := flag.String("codes", "", "Comma-separated list of fund codes to crawl")
	listMode := flag.String("list", "", "Fetch fund list: 'all', 'stock' (股票+混合), 'popular' (热门20只)")
	catalogOnly := flag.Bool("catalog-only", false, "When used with --list and --save-db, only upsert fund catalog metadata without crawling details/holdings")
	concurrency := flag.Int("concurrency", 3, "Maximum number of concurrent requests")
	output := flag.String("output", "", "Output JSON file path (if empty, prints to stdout)")
	timeout := flag.Duration("timeout", 120*time.Second, "Request timeout duration")
	saveDB := flag.Bool("save-db", false, "Save crawled data to PostgreSQL database")
	limit := flag.Int("limit", 0, "Limit number of funds to crawl (0 = no limit)")
	fixNames := flag.Bool("fix-names", false, "Fix garbled stock names in database using Sina Finance API")
	fixAllNames := flag.Bool("fix-all-names", false, "Refresh ALL stock names from Sina Finance API")
	historyOnly := flag.Bool("history-only", false, "Fetch and save recent official daily NAV history only; requires --save-db")
	historyDays := flag.Int("history-days", 30, "Number of recent official NAV days to save in --history-only mode")
	trackedOnly := flag.Bool("tracked-only", false, "When used with --history-only, load fund codes from holdings, favorites, and watchlist groups")

	flag.Parse()

	if *catalogOnly && !*saveDB {
		log.Fatalf("❌ --catalog-only requires --save-db")
	}
	if *historyOnly && !*saveDB {
		log.Fatalf("❌ --history-only requires --save-db")
	}
	if *historyOnly && *catalogOnly {
		log.Fatalf("❌ --history-only cannot be combined with --catalog-only")
	}

	// Handle --fix-names mode: fix garbled stock names in database
	if *fixNames || *fixAllNames {
		log.Println("🔧 Connecting to PostgreSQL database...")
		cfg := database.DefaultConfig()
		db, err := database.InitDB(cfg, database.AllModels()...)
		if err != nil {
			log.Fatalf("❌ Failed to connect to database: %v", err)
		}
		defer database.Close()

		fixer := crawler.NewStockNameFixer(db)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		if *fixAllNames {
			log.Println("🔄 Refreshing ALL stock names from Sina Finance API...")
			count, err := fixer.FixAllStockNames(ctx)
			if err != nil {
				log.Fatalf("❌ Failed to fix stock names: %v", err)
			}
			log.Printf("✅ Updated %d stock names", count)
		} else {
			log.Println("🔍 Detecting and fixing garbled stock names...")
			count, err := fixer.FixGarbledStockNames(ctx)
			if err != nil {
				log.Fatalf("❌ Failed to fix stock names: %v", err)
			}
			log.Printf("✅ Fixed %d garbled stock names", count)
		}
		return
	}

	// Determine fund codes to crawl
	var fundCodes []string

	if *listMode != "" {
		// Fetch fund list from Eastmoney
		log.Printf("📋 Fetching fund list (mode: %s)...", *listMode)
		listCrawler := crawler.NewFundListCrawler()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)

		var funds []crawler.FundListItem
		var err error

		switch *listMode {
		case "all":
			funds, err = listCrawler.FetchAllFunds(ctx)
		case "stock":
			funds, err = listCrawler.FetchStockFunds(ctx)
		case "popular":
			funds, err = listCrawler.FetchPopularFunds(ctx)
		default:
			cancel()
			log.Fatalf("❌ Invalid list mode: %s (use 'all', 'stock', or 'popular')", *listMode)
		}
		cancel()

		if err != nil {
			log.Fatalf("❌ Failed to fetch fund list: %v", err)
		}

		log.Printf("📊 Found %d funds in list", len(funds))

		// Apply limit
		if *limit > 0 && len(funds) > *limit {
			funds = funds[:*limit]
			log.Printf("📉 Limited to %d funds", *limit)
		}

		// Extract codes
		fundCodes = make([]string, len(funds))
		for i, f := range funds {
			fundCodes[i] = f.Code
		}

		if *catalogOnly {
			log.Println("🔧 Connecting to PostgreSQL database...")
			cfg := database.DefaultConfig()
			db, err := database.InitDB(cfg, database.AllModels()...)
			if err != nil {
				log.Fatalf("❌ Failed to connect to database: %v", err)
			}
			defer database.Close()

			log.Println("💾 Saving fund catalog metadata to database...")
			dbCtx, dbCancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer dbCancel()

			markMissing := *listMode == "all" && *limit == 0
			stats, err := saveFundCatalogEntries(dbCtx, db, funds, markMissing)
			if err != nil {
				log.Fatalf("❌ Failed to save fund catalog: %v", err)
			}
			log.Printf(
				"💾 Fund catalog save complete: %d funds upserted (%d active, %d unavailable, %d marked catalog_missing)",
				stats.Upserted,
				stats.Active,
				stats.Unavailable,
				stats.MarkedMissing,
			)
			return
		}
	} else if *codes != "" {
		// Use provided codes
		fundCodes = strings.Split(*codes, ",")
		for i := range fundCodes {
			fundCodes[i] = strings.TrimSpace(fundCodes[i])
		}
	} else if *historyOnly && *trackedOnly {
		fundCodes = []string{}
	} else {
		// Default codes
		fundCodes = []string{"005827", "003095", "320007"}
	}
	fundCodes = uniqueFundCodes(fundCodes)

	log.Printf("🚀 Starting Eastmoney Fund Crawler")
	log.Printf("📊 Fund codes: %d funds", len(fundCodes))
	log.Printf("🔄 Concurrency: %d", *concurrency)
	log.Printf("⏱️  Timeout: %s", *timeout)
	log.Printf("💾 Save to DB: %v", *saveDB)
	log.Println()

	// Initialize database if saving to DB
	var fundRepo *repository.PostgresFundRepository
	var db *gorm.DB
	if *saveDB {
		log.Println("🔧 Connecting to PostgreSQL database...")
		cfg := database.DefaultConfig()
		var err error
		db, err = database.InitDB(cfg, database.AllModels()...)
		if err != nil {
			log.Fatalf("❌ Failed to connect to database: %v", err)
		}
		defer database.Close()

		fundRepo = repository.NewPostgresFundRepository(db)
		log.Println("✅ Database connected")
	}

	if *historyOnly && *trackedOnly {
		dbCtx, dbCancel := context.WithTimeout(context.Background(), *timeout)
		trackedCodes, err := listTrackedHistoryFundCodes(dbCtx, db)
		dbCancel()
		if err != nil {
			log.Fatalf("❌ Failed to load tracked fund codes: %v", err)
		}
		fundCodes = trackedCodes
		log.Printf("📌 Loaded %d tracked fund codes from holdings/favorites/watchlists", len(fundCodes))
	}

	// Create crawler service
	crawlService := crawler.NewCrawlService(*concurrency)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	if *historyOnly {
		if len(fundCodes) == 0 {
			log.Println("ℹ️ No fund codes to sync history for")
			return
		}

		days := normalizeHistoryDays(*historyDays)
		log.Printf("🕚 Fetching recent official NAV history: %d funds, %d days", len(fundCodes), days)
		startTime := time.Now()
		stats, err := saveRecentFundHistories(ctx, crawlService, fundRepo, fundCodes, days, *concurrency)
		if err != nil {
			log.Fatalf("❌ Failed to save official NAV histories: %v", err)
		}
		log.Printf(
			"✅ Official NAV history sync complete in %s: %d funds success, %d failed, %d history rows saved",
			time.Since(startTime),
			stats.SuccessFunds,
			stats.FailedFunds,
			stats.HistoryRows,
		)
		return
	}

	// Start crawling
	startTime := time.Now()
	results := crawlService.BatchFetchFundData(ctx, fundCodes)
	elapsed := time.Since(startTime)

	log.Printf("⏱️  Crawling completed in %s", elapsed)
	log.Println()

	// Print results
	crawlService.PrintResults(results)

	// Count success/failure
	success, failed := 0, 0
	for _, r := range results {
		if r.Error == nil {
			success++
		} else {
			failed++
		}
	}

	log.Printf("📈 Summary: %d success, %d failed", success, failed)

	// Save to database if enabled
	if *saveDB && fundRepo != nil {
		log.Println()
		log.Println("💾 Saving data to database...")
		savedCount := 0

		// Create a fresh context for database operations (not affected by crawl timeout)
		dbCtx, dbCancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer dbCancel()

		for code, result := range results {
			if result.Error != nil {
				// Don't log every skip, just count them
				continue
			}

			if result.Fund == nil {
				continue
			}

			// Convert crawler.Fund to domain.Fund
			domainFund := &domain.Fund{
				ID:          result.Fund.ID,
				Name:        result.Fund.Name,
				Type:        result.Fund.Type,
				Manager:     result.Fund.Manager,
				Company:     result.Fund.Company,
				NetAssetVal: result.Fund.NetAssetVal,
				TotalScale:  result.Fund.TotalScale,
				UpdatedAt:   time.Now(),
			}

			// Save fund
			if err := fundRepo.SaveFund(dbCtx, domainFund); err != nil {
				log.Printf("   ❌ Failed to save fund %s: %v", code, err)
				continue
			}

			// Convert and save holdings.
			// nil means upstream holdings fetch failed, so keep existing persisted holdings untouched.
			if result.Holdings == nil {
				log.Printf("   ⚠️ Skipped replacing holdings for %s because upstream holdings fetch failed; existing holdings kept", code)
			} else {
				domainHoldings := make([]domain.StockHolding, len(result.Holdings))
				for i, h := range result.Holdings {
					domainHoldings[i] = domain.StockHolding{
						StockCode:       h.StockCode,
						StockName:       h.StockName,
						Exchange:        h.Exchange,
						HoldingRatio:    h.HoldingRatio,
						HoldingShares:   h.HoldingShares,
						MarketValue:     h.MarketValue,
						ReportingPeriod: h.ReportingPeriod,
					}
				}

				if err := fundRepo.SaveHoldings(dbCtx, code, domainHoldings); err != nil {
					log.Printf("   ❌ Failed to save holdings for %s: %v", code, err)
					continue
				}
			}

			savedCount++
			// Only log every 100 funds to avoid too much output
			if savedCount%100 == 0 {
				log.Printf("   💾 Progress: %d funds saved...", savedCount)
			}
		}

		log.Printf("💾 Database save complete: %d funds saved (skipped %d with errors)", savedCount, failed)
	}

	// Output to JSON file if specified
	if *output != "" {
		if err := writeResultsToJSON(*output, results); err != nil {
			log.Printf("❌ Failed to write JSON: %v", err)
			os.Exit(1)
		}
		log.Printf("📄 Results saved to %s", *output)
	}
}

type catalogSaveStats struct {
	Upserted      int
	Active        int
	Unavailable   int
	MarkedMissing int64
}

func saveFundCatalogEntries(ctx context.Context, db *gorm.DB, funds []crawler.FundListItem, markMissing bool) (catalogSaveStats, error) {
	if len(funds) == 0 {
		return catalogSaveStats{}, nil
	}

	const batchSize = 1000
	upsertClause := clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "type", "catalog_status", "catalog_synced_at", "updated_at"}),
	}

	stats := catalogSaveStats{}
	now := time.Now()
	remoteCodes := make([]string, 0, len(funds))

	for start := 0; start < len(funds); start += batchSize {
		end := start + batchSize
		if end > len(funds) {
			end = len(funds)
		}

		rows := make([]database.Fund, 0, end-start)
		for _, fund := range funds[start:end] {
			status := resolveFundCatalogStatus(fund)
			if status == domain.FundCatalogStatusUnavailable {
				stats.Unavailable++
			} else {
				stats.Active++
			}
			remoteCodes = append(remoteCodes, fund.Code)
			rows = append(rows, database.Fund{
				ID:              fund.Code,
				Name:            fund.Name,
				Type:            fund.Type,
				CatalogStatus:   status,
				CatalogSyncedAt: &now,
				UpdatedAt:       now,
			})
		}

		if err := db.WithContext(ctx).Clauses(upsertClause).CreateInBatches(&rows, len(rows)).Error; err != nil {
			return stats, fmt.Errorf("failed to upsert fund catalog batch %d-%d: %w", start+1, end, err)
		}

		stats.Upserted += len(rows)
		if stats.Upserted%5000 == 0 || stats.Upserted == len(funds) {
			log.Printf("   💾 Progress: %d/%d catalog records saved...", stats.Upserted, len(funds))
		}
	}

	if markMissing && len(remoteCodes) > 0 {
		result := db.WithContext(ctx).
			Model(&database.Fund{}).
			Where("id NOT IN ?", remoteCodes).
			Where("id NOT IN (SELECT fund_id FROM fund_valuation_profiles)").
			Updates(map[string]interface{}{
				"catalog_status":    domain.FundCatalogStatusCatalogMissing,
				"catalog_synced_at": now,
				"updated_at":        now,
			})
		if result.Error != nil {
			return stats, fmt.Errorf("failed to mark missing catalog funds: %w", result.Error)
		}
		stats.MarkedMissing = result.RowsAffected
	}

	return stats, nil
}

func resolveFundCatalogStatus(fund crawler.FundListItem) string {
	name := strings.TrimSpace(fund.Name)
	if strings.Contains(name, "后端") {
		return domain.FundCatalogStatusUnavailable
	}
	return domain.FundCatalogStatusActive
}

type historySaveStats struct {
	SuccessFunds int64
	FailedFunds  int64
	HistoryRows  int64
}

func saveRecentFundHistories(
	ctx context.Context,
	crawlService *crawler.CrawlService,
	fundRepo domain.FundRepository,
	fundCodes []string,
	days int,
	concurrency int,
) (historySaveStats, error) {
	if crawlService == nil || fundRepo == nil {
		return historySaveStats{}, fmt.Errorf("crawler and fund repository are required")
	}
	if concurrency <= 0 {
		concurrency = 1
	}

	var stats historySaveStats
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)

	for _, fundCode := range uniqueFundCodes(fundCodes) {
		fundCode := fundCode
		g.Go(func() error {
			histories, err := crawlService.FetchRecentFundHistory(ctx, fundCode, days)
			if err != nil {
				atomic.AddInt64(&stats.FailedFunds, 1)
				log.Printf("   ⚠️ Failed to fetch history for %s: %v", fundCode, err)
				return nil
			}
			if len(histories) == 0 {
				atomic.AddInt64(&stats.FailedFunds, 1)
				log.Printf("   ⚠️ No history returned for %s", fundCode)
				return nil
			}

			for index := range histories {
				history := histories[index]
				if err := fundRepo.SaveFundHistory(ctx, &history); err != nil {
					atomic.AddInt64(&stats.FailedFunds, 1)
					log.Printf("   ⚠️ Failed to save history for %s/%s: %v", fundCode, history.Date, err)
					return nil
				}
				atomic.AddInt64(&stats.HistoryRows, 1)
			}

			latestHistory := histories[len(histories)-1]
			if fund, err := fundRepo.GetFundByID(ctx, fundCode); err == nil && fund != nil {
				fund.NetAssetVal = latestHistory.NetAssetVal
				fund.UpdatedAt = time.Now()
				if saveErr := fundRepo.SaveFund(ctx, fund); saveErr != nil {
					log.Printf("   ⚠️ Failed to update latest NAV for %s: %v", fundCode, saveErr)
				}
			}

			if success := atomic.AddInt64(&stats.SuccessFunds, 1); success%20 == 0 {
				log.Printf("   💾 History progress: %d funds saved...", success)
			}
			return nil
		})
	}

	return stats, g.Wait()
}

func listTrackedHistoryFundCodes(ctx context.Context, db *gorm.DB) ([]string, error) {
	if db == nil {
		return nil, fmt.Errorf("database is required")
	}

	seen := make(map[string]struct{})
	addCodes := func(codes []string) {
		for _, code := range codes {
			code = strings.TrimSpace(code)
			if code == "" {
				continue
			}
			seen[code] = struct{}{}
		}
	}

	var holdingCodes []string
	if err := db.WithContext(ctx).
		Model(&database.UserFundHolding{}).
		Distinct("fund_id").
		Pluck("fund_id", &holdingCodes).Error; err != nil {
		return nil, fmt.Errorf("list holding fund codes: %w", err)
	}
	addCodes(holdingCodes)

	var favoriteCodes []string
	if err := db.WithContext(ctx).
		Model(&database.UserFavoriteFund{}).
		Distinct("fund_id").
		Pluck("fund_id", &favoriteCodes).Error; err != nil {
		return nil, fmt.Errorf("list favorite fund codes: %w", err)
	}
	addCodes(favoriteCodes)

	var watchlistCodes []string
	if err := db.WithContext(ctx).
		Model(&database.UserWatchlistFund{}).
		Distinct("fund_id").
		Pluck("fund_id", &watchlistCodes).Error; err != nil {
		return nil, fmt.Errorf("list watchlist fund codes: %w", err)
	}
	addCodes(watchlistCodes)

	result := make([]string, 0, len(seen))
	for code := range seen {
		result = append(result, code)
	}
	sort.Strings(result)
	return result, nil
}

func normalizeHistoryDays(days int) int {
	if days <= 0 {
		return 30
	}
	if days > 180 {
		return 180
	}
	return days
}

func uniqueFundCodes(codes []string) []string {
	seen := make(map[string]struct{}, len(codes))
	result := make([]string, 0, len(codes))
	for _, code := range codes {
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		result = append(result, code)
	}
	sort.Strings(result)
	return result
}

// OutputData represents the JSON output structure.
type OutputData struct {
	CrawledAt string       `json:"crawled_at"`
	Funds     []FundOutput `json:"funds"`
}

// FundOutput represents a single fund in the output.
type FundOutput struct {
	Code     string          `json:"code"`
	Name     string          `json:"name"`
	Manager  string          `json:"manager"`
	Company  string          `json:"company"`
	NAV      string          `json:"nav"`
	Scale    string          `json:"scale"`
	Holdings []HoldingOutput `json:"holdings"`
	Error    string          `json:"error,omitempty"`
}

// HoldingOutput represents a single holding in the output.
type HoldingOutput struct {
	StockCode    string `json:"stock_code"`
	StockName    string `json:"stock_name"`
	HoldingRatio string `json:"holding_ratio"`
	Exchange     string `json:"exchange"`
}

func writeResultsToJSON(filepath string, results map[string]*crawler.CrawlResult) error {
	output := OutputData{
		CrawledAt: time.Now().Format(time.RFC3339),
		Funds:     make([]FundOutput, 0, len(results)),
	}

	for code, result := range results {
		fundOut := FundOutput{
			Code: code,
		}

		if result.Error != nil {
			fundOut.Error = result.Error.Error()
		} else {
			fundOut.Name = result.Fund.Name
			fundOut.Manager = result.Fund.Manager
			fundOut.Company = result.Fund.Company
			fundOut.NAV = result.Fund.NetAssetVal.String()
			fundOut.Scale = result.Fund.TotalScale.String()

			for _, h := range result.Holdings {
				fundOut.Holdings = append(fundOut.Holdings, HoldingOutput{
					StockCode:    h.StockCode,
					StockName:    h.StockName,
					HoldingRatio: h.HoldingRatio.String(),
					Exchange:     string(h.Exchange),
				})
			}
		}

		output.Funds = append(output.Funds, fundOut)
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON marshal failed: %w", err)
	}

	return os.WriteFile(filepath, data, 0644)
}
