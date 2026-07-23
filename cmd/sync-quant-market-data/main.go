package main

import (
	"context"
	"flag"
	"log"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/service"
)

func main() {
	years := flag.Int("years", 5, "number of calendar years to synchronize")
	symbolsFlag := flag.String("symbols", "", "comma-separated symbols; empty uses pilot-v1 plus 000300")
	flag.Parse()
	if *years <= 0 || *years > 10 {
		log.Fatal("years must be between 1 and 10")
	}

	db, err := database.InitDB(database.DefaultConfig(), database.AllModels()...)
	if err != nil {
		log.Fatalf("initialize database: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	store := service.NewQuantResearchStore(db)
	if err := store.SeedPilotUniverse(ctx, time.Now()); err != nil {
		log.Fatalf("seed pilot universe: %v", err)
	}
	symbols := requestedSymbols(*symbolsFlag)
	provider := service.NewEastmoneyQuantMarketDataProvider()
	end := time.Now().In(time.FixedZone("CST", 8*60*60))
	start := end.AddDate(-*years, 0, 0)

	failed := 0
	for index, symbol := range symbols {
		bars, fetchErr := provider.FetchDailyBars(ctx, symbol, start, end)
		if fetchErr != nil {
			failed++
			log.Printf("[%d/%d] %s fetch failed: %v", index+1, len(symbols), symbol, fetchErr)
			continue
		}
		if saveErr := store.UpsertMarketBars(ctx, bars); saveErr != nil {
			failed++
			log.Printf("[%d/%d] %s save failed: %v", index+1, len(symbols), symbol, saveErr)
			continue
		}
		if actionErr := store.InferAdjustmentActions(ctx, symbol); actionErr != nil {
			log.Printf("[%d/%d] %s adjustment inference warning: %v", index+1, len(symbols), symbol, actionErr)
		}
		log.Printf("[%d/%d] %s synchronized %d daily bars", index+1, len(symbols), symbol, len(bars))
	}
	if calendarErr := store.RebuildTradingCalendar(ctx, start, end); calendarErr != nil {
		log.Printf("trading calendar rebuild warning: %v", calendarErr)
	}
	if failed > 0 {
		log.Fatalf("market data synchronization completed with %d failed symbols", failed)
	}
	log.Printf("quant market data synchronization completed for %d symbols", len(symbols))
}

func requestedSymbols(value string) []string {
	if strings.TrimSpace(value) != "" {
		seen := make(map[string]struct{})
		result := make([]string, 0)
		for _, symbol := range strings.Split(value, ",") {
			symbol = strings.TrimSpace(symbol)
			if symbol == "" {
				continue
			}
			if _, ok := seen[symbol]; ok {
				continue
			}
			seen[symbol] = struct{}{}
			result = append(result, symbol)
		}
		return result
	}
	result := []string{"000300"}
	for _, item := range service.PilotV1Instruments() {
		result = append(result, item.Symbol)
	}
	return result
}
