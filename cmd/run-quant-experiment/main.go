package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/appconfig"
	cache "github.com/RomaticDOG/fund/internal/cache"
	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/service"
)

func main() {
	preset := flag.String("preset", service.QuantExperimentPresetRiskV1, "experiment preset")
	startDate := flag.String("start", "", "backtest start date (defaults to earliest available signal)")
	endDate := flag.String("end", "", "backtest end date (defaults to latest shared signal/market date)")
	flag.Parse()
	if strings.TrimSpace(*preset) != service.QuantExperimentPresetRiskV1 {
		log.Fatalf("unsupported preset %q", *preset)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	fileConfig, err := appconfig.LoadConfig()
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}
	db, err := database.InitDB(database.DefaultConfig(), database.AllModels()...)
	if err != nil {
		log.Fatalf("initialize database: %v", err)
	}
	defer database.Close()

	store := service.NewQuantResearchStore(db)
	resolvedStart, resolvedEnd, err := store.ResolveBacktestDateRange(ctx, service.QuantSignalModeHistoryProxy)
	if err != nil {
		log.Fatalf("resolve research date range: %v", err)
	}
	if strings.TrimSpace(*startDate) != "" {
		resolvedStart = strings.TrimSpace(*startDate)
	}
	if strings.TrimSpace(*endDate) != "" {
		resolvedEnd = strings.TrimSpace(*endDate)
	}
	request := service.DefaultQuantBacktestRequest()
	request.StartDate = resolvedStart
	request.EndDate = resolvedEnd
	experiment, err := store.CreateRiskV1Experiment(ctx, request)
	if err != nil {
		log.Fatalf("create experiment: %v", err)
	}

	redisURL, keyPrefix := dragonflyConfig(fileConfig)
	queue, err := cache.NewDragonflyQuantQueue(redisURL, keyPrefix)
	if err != nil {
		log.Fatalf("initialize Dragonfly queue: %v", err)
	}
	defer queue.Close()
	if err := queue.Ping(ctx); err != nil {
		log.Fatalf("connect Dragonfly queue: %v", err)
	}

	for _, variant := range experiment.Variants {
		job := variant.Job
		switch job.Status {
		case "completed", "running":
			fmt.Printf("variant=%s job=%s status=%s reused\n", variant.Key, job.ID, job.Status)
			continue
		case "failed", "queue_failed":
			retried, retryErr := store.RetryBacktestJob(ctx, job.ID)
			if retryErr != nil || !retried {
				log.Printf("variant=%s job=%s retry failed: %v", variant.Key, job.ID, retryErr)
				continue
			}
		}
		if err := queue.EnqueueBacktest(ctx, job.ID); err != nil {
			_ = store.MarkBacktestQueueFailed(ctx, job.ID, err)
			log.Printf("variant=%s job=%s enqueue failed: %v", variant.Key, job.ID, err)
			continue
		}
		fmt.Printf("variant=%s job=%s status=queued\n", variant.Key, job.ID)
	}
	fmt.Printf("experiment=%s preset=%s range=%s..%s\n", experiment.ID, experiment.Preset, experiment.StartDate, experiment.EndDate)
}

func dragonflyConfig(config *appconfig.Config) (string, string) {
	url := "redis://127.0.0.1:16380/0"
	prefix := "fundlive"
	if config != nil {
		if strings.TrimSpace(config.Cache.RedisURL) != "" {
			url = config.Cache.RedisURL
		}
		if strings.TrimSpace(config.Cache.KeyPrefix) != "" {
			prefix = config.Cache.KeyPrefix
		}
	}
	if value := strings.TrimSpace(os.Getenv("REDIS_URL")); value != "" {
		url = value
	}
	if value := strings.TrimSpace(os.Getenv("REDIS_KEY_PREFIX")); value != "" {
		prefix = value
	}
	return url, prefix
}
