package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/repository"
	"github.com/RomaticDOG/fund/internal/service"
	"github.com/shopspring/decimal"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type stats struct {
	SectorSnapshots int64
	ThemeSnapshots  int64
	OtherSector     int64
	OtherTheme      int64
}

func main() {
	limit := flag.Int("limit", 0, "optional max number of funds to rebuild")
	flag.Parse()

	cfg := database.DefaultConfig()
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}

	ctx := context.Background()
	before, err := snapshotStats(db)
	if err != nil {
		log.Fatalf("load stats(before): %v", err)
	}

	fundIDs, err := collectFundIDs(db)
	if err != nil {
		log.Fatalf("collect fund ids: %v", err)
	}
	if *limit > 0 && *limit < len(fundIDs) {
		fundIDs = fundIDs[:*limit]
	}

	fundRepo := repository.NewPostgresFundRepository(db)
	sectorStore := service.NewFundSectorStore(db)
	resolver := service.NewFundResolver(db, fundRepo)

	processed := 0
	skipped := 0
	for _, fundID := range fundIDs {
		ok, err := rebuildFundClassification(ctx, fundRepo, resolver, sectorStore, fundID)
		if err != nil {
			log.Printf("⚠️ rebuild classification failed for %s: %v", fundID, err)
			continue
		}
		if ok {
			processed++
		} else {
			skipped++
		}
	}

	after, err := snapshotStats(db)
	if err != nil {
		log.Fatalf("load stats(after): %v", err)
	}

	fmt.Printf("classification rebuild complete\n")
	fmt.Printf("funds_targeted=%d processed=%d skipped=%d\n", len(fundIDs), processed, skipped)
	fmt.Printf("before sector_snapshots=%d theme_snapshots=%d other_equity=%d other_theme=%d\n", before.SectorSnapshots, before.ThemeSnapshots, before.OtherSector, before.OtherTheme)
	fmt.Printf("after  sector_snapshots=%d theme_snapshots=%d other_equity=%d other_theme=%d\n", after.SectorSnapshots, after.ThemeSnapshots, after.OtherSector, after.OtherTheme)
}

func rebuildFundClassification(
	ctx context.Context,
	fundRepo domain.FundRepository,
	resolver *service.FundResolver,
	sectorStore *service.FundSectorStore,
	fundID string,
) (bool, error) {
	fund, err := fundRepo.GetFundByID(ctx, fundID)
	if err != nil || fund == nil {
		return false, err
	}

	holdings, source, err := resolveClassificationHoldings(ctx, fundRepo, resolver, fundID, fund)
	if err != nil {
		return false, err
	}
	if !hasEffectiveHoldingsLocal(holdings) {
		return false, nil
	}

	_, err = sectorStore.UpsertFromHoldings(ctx, fundID, holdings, source)
	if err != nil {
		return false, err
	}
	return true, nil
}

func resolveClassificationHoldings(
	ctx context.Context,
	fundRepo domain.FundRepository,
	resolver *service.FundResolver,
	fundID string,
	fund *domain.Fund,
) ([]domain.StockHolding, string, error) {
	holdings, err := fundRepo.GetFundHoldings(ctx, fundID)
	if err != nil {
		return nil, "", err
	}

	source := service.SectorSourceDirectHoldings

	if resolver != nil {
		if display, displayErr := resolver.ResolveDisplayHoldings(ctx, fundID, fund.Name); displayErr == nil {
			if targetItem, ok := domain.PrimaryTrackedETF(display); ok {
				targetCode := strings.TrimSpace(targetItem.Code)
				if targetCode != "" {
					targetHoldings, targetErr := fundRepo.GetFundHoldings(ctx, targetCode)
					if targetErr == nil && hasEffectiveHoldingsLocal(targetHoldings) {
						return targetHoldings, service.SectorSourceTargetETFFallback, nil
					}
				}
			}
		}
	}

	if !hasEffectiveHoldingsLocal(holdings) && resolver != nil {
		resolvedHoldings, holdingsSource, resolveErr := resolver.GetHoldingsWithFallback(ctx, fundID, fund.Name)
		if resolveErr != nil {
			return nil, "", resolveErr
		}
		if hasEffectiveHoldingsLocal(resolvedHoldings) {
			if holdingsSource != "" && holdingsSource != fundID {
				source = service.SectorSourceTargetETFFallback
			}
			holdings = resolvedHoldings
		}
	}

	if isQDIIFundLocal(fund) && source == service.SectorSourceDirectHoldings {
		source = service.SectorSourceQDIIHoldings
	}
	return holdings, source, nil
}

func collectFundIDs(db *gorm.DB) ([]string, error) {
	var rows []struct{ FundID string }
	err := db.Raw(`
WITH candidate_ids AS (
  SELECT fund_id FROM fund_sector_snapshots
  UNION
  SELECT fund_id FROM fund_theme_snapshots
  UNION
  SELECT feeder_code AS fund_id FROM fund_mappings WHERE is_resolved = true AND target_code <> ''
)
SELECT DISTINCT fund_id FROM candidate_ids ORDER BY fund_id ASC
`).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]string, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.FundID) == "" {
			continue
		}
		result = append(result, row.FundID)
	}
	sort.Strings(result)
	return result, nil
}

func snapshotStats(db *gorm.DB) (stats, error) {
	var result stats
	if err := db.Raw(`SELECT COUNT(*) FROM fund_sector_snapshots`).Scan(&result.SectorSnapshots).Error; err != nil {
		return result, err
	}
	if err := db.Raw(`SELECT COUNT(*) FROM fund_theme_snapshots`).Scan(&result.ThemeSnapshots).Error; err != nil {
		return result, err
	}
	if err := db.Raw(`SELECT COUNT(*) FROM fund_sector_snapshots WHERE primary_sector_code = 'other_equity'`).Scan(&result.OtherSector).Error; err != nil {
		return result, err
	}
	if err := db.Raw(`SELECT COUNT(*) FROM fund_theme_snapshots WHERE primary_theme_code = 'other_theme'`).Scan(&result.OtherTheme).Error; err != nil {
		return result, err
	}
	return result, nil
}

func hasEffectiveHoldingsLocal(holdings []domain.StockHolding) bool {
	if len(holdings) == 0 {
		return false
	}
	for _, holding := range holdings {
		if holding.HoldingRatio.GreaterThan(decimal.Zero) {
			return true
		}
	}
	return false
}

func isQDIIFundLocal(fund *domain.Fund) bool {
	if fund == nil {
		return false
	}
	fundType := strings.ToLower(strings.TrimSpace(fund.Type))
	name := strings.ToLower(strings.TrimSpace(fund.Name))
	return strings.Contains(fundType, "qdii") || strings.Contains(name, "qdii")
}
