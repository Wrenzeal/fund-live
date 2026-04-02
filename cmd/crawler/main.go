// Package main provides a CLI tool for crawling fund data from Eastmoney.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/crawler"
	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/repository"
)

func main() {
	// Command line flags
	codes := flag.String("codes", "", "Comma-separated list of fund codes to crawl")
	listMode := flag.String("list", "", "Fetch fund list: 'all', 'stock' (股票+混合), 'popular' (热门20只)")
	concurrency := flag.Int("concurrency", 3, "Maximum number of concurrent requests")
	output := flag.String("output", "", "Output JSON file path (if empty, prints to stdout)")
	timeout := flag.Duration("timeout", 120*time.Second, "Request timeout duration")
	saveDB := flag.Bool("save-db", false, "Save crawled data to PostgreSQL database")
	limit := flag.Int("limit", 0, "Limit number of funds to crawl (0 = no limit)")
	fixNames := flag.Bool("fix-names", false, "Fix garbled stock names in database using Sina Finance API")
	fixAllNames := flag.Bool("fix-all-names", false, "Refresh ALL stock names from Sina Finance API")

	flag.Parse()

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
	} else if *codes != "" {
		// Use provided codes
		fundCodes = strings.Split(*codes, ",")
		for i := range fundCodes {
			fundCodes[i] = strings.TrimSpace(fundCodes[i])
		}
	} else {
		// Default codes
		fundCodes = []string{"005827", "003095", "320007"}
	}

	log.Printf("🚀 Starting Eastmoney Fund Crawler")
	log.Printf("📊 Fund codes: %d funds", len(fundCodes))
	log.Printf("🔄 Concurrency: %d", *concurrency)
	log.Printf("⏱️  Timeout: %s", *timeout)
	log.Printf("💾 Save to DB: %v", *saveDB)
	log.Println()

	// Initialize database if saving to DB
	var fundRepo *repository.PostgresFundRepository
	if *saveDB {
		log.Println("🔧 Connecting to PostgreSQL database...")
		cfg := database.DefaultConfig()
		db, err := database.InitDB(cfg, database.AllModels()...)
		if err != nil {
			log.Fatalf("❌ Failed to connect to database: %v", err)
		}
		defer database.Close()

		fundRepo = repository.NewPostgresFundRepository(db)
		log.Println("✅ Database connected")
	}

	// Create crawler service
	crawlService := crawler.NewCrawlService(*concurrency)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

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

			// Convert and save holdings
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
