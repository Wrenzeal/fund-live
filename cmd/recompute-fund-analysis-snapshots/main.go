package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/adapter"
	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/repository"
	"github.com/RomaticDOG/fund/internal/service"
)

func main() {
	limit := flag.Int("limit", 80, "max candidate funds to recompute when --funds is empty")
	fundsArg := flag.String("funds", "", "comma separated fund ids to recompute")
	flag.Parse()

	cfg := database.DefaultConfig()
	db, err := database.InitDB(cfg, database.AllModels()...)
	if err != nil {
		log.Fatalf("init db: %v", err)
	}

	fundRepo := repository.NewPostgresFundRepository(db)
	defaultSource := domain.ResolveQuoteSource(domain.QuoteSource(""), domain.QuoteSourceSina)
	cacheRepo := repository.NewMemoryCacheRepository(60*time.Second, 5*time.Minute)
	quoteProvider := adapter.NewSinaFinanceProvider()
	valuationService := service.NewValuationService(fundRepo, quoteProvider, cacheRepo)
	valuationService.SetQuoteProvider(domain.QuoteSourceSina, quoteProvider)
	valuationService.SetQuoteProvider(domain.QuoteSourceTencent, adapter.NewTencentQuoteProvider())
	valuationService.SetOverseasQuoteProvider(adapter.NewTencentQuoteProvider())
	valuationService.SetDefaultQuoteSource(defaultSource)
	fundDataLoader := service.NewFundDataLoader(fundRepo)
	valuationService.SetFundDataLoader(fundDataLoader)
	fundResolver := service.NewFundResolver(db, fundRepo)
	fundResolver.SetFundDataLoader(fundDataLoader)
	valuationService.SetFundResolver(fundResolver)
	valuationService.SetValuationProfileStore(service.NewValuationProfileStore(db))
	sectorStore := service.NewFundSectorStore(db)
	snapshotStore := service.NewFundAnalysisSnapshotStore(db)
	capabilityService := service.NewEstimateCapabilityService(db)
	coordinator := service.NewFundAnalysisCoordinator(valuationService, fundRepo, fundResolver, sectorStore)

	ctx := context.Background()
	fundIDs := resolveFundIDs(strings.TrimSpace(*fundsArg), capabilityService, ctx, *limit)
	if len(fundIDs) == 0 {
		fmt.Println("no fund ids to recompute")
		return
	}

	now := time.Now()
	successCount := 0
	for _, fundID := range fundIDs {
		_, analysis, err := coordinator.BuildForFund(ctx, fundID, now)
		if err != nil {
			log.Printf("skip %s: %v", fundID, err)
			continue
		}
		if analysis == nil {
			log.Printf("skip %s: analysis nil", fundID)
			continue
		}
		if err := snapshotStore.Save(ctx, fundID, analysis, now); err != nil {
			log.Printf("save %s failed: %v", fundID, err)
			continue
		}
		successCount++
		fmt.Printf("saved %s | score=%s | risk=%s\n", fundID, analysis.TotalScore.StringFixed(1), analysis.RiskLevel)
	}
	fmt.Printf("recompute complete | total=%d saved=%d\n", len(fundIDs), successCount)
}

func resolveFundIDs(fundsArg string, capabilityService *service.EstimateCapabilityService, ctx context.Context, limit int) []string {
	if fundsArg != "" {
		parts := strings.Split(fundsArg, ",")
		result := make([]string, 0, len(parts))
		seen := make(map[string]struct{}, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if _, ok := seen[part]; ok {
				continue
			}
			seen[part] = struct{}{}
			result = append(result, part)
		}
		return result
	}
	if capabilityService == nil {
		return nil
	}
	fundIDs, err := capabilityService.ListRankingCandidateFundIDs(ctx, limit)
	if err != nil {
		fmt.Fprintln(os.Stderr, "list candidate fund ids:", err)
		return nil
	}
	return fundIDs
}
