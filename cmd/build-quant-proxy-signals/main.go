package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/service"
)

func main() {
	startFlag := flag.String("start", "", "first signal date in YYYY-MM-DD; defaults to five years ago")
	endFlag := flag.String("end", "", "last signal date in YYYY-MM-DD; defaults to today")
	flag.Parse()

	end := parseDateOrDefault(*endFlag, time.Now())
	start := parseDateOrDefault(*startFlag, end.AddDate(-5, 0, 0))
	if end.Before(start) {
		log.Fatal("end must not be before start")
	}

	db, err := database.InitDB(database.DefaultConfig(), database.AllModels()...)
	if err != nil {
		log.Fatalf("initialize database: %v", err)
	}
	defer database.Close()
	store := service.NewQuantResearchStore(db)
	inserted, err := store.BuildHistoricalProxySignals(context.Background(), start, end)
	if err != nil {
		log.Fatalf("build historical proxy signals: %v", err)
	}
	log.Printf("historical proxy build completed: inserted=%d start=%s end=%s", inserted, start.Format("2006-01-02"), end.Format("2006-01-02"))
}

func parseDateOrDefault(value string, fallback time.Time) time.Time {
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, time.FixedZone("CST", 8*60*60))
	if err != nil {
		log.Fatalf("invalid date %q: %v", value, err)
	}
	return parsed
}
