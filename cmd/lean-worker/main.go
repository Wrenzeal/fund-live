package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/RomaticDOG/fund/internal/appconfig"
	cache "github.com/RomaticDOG/fund/internal/cache"
	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/service"
)

const defaultLeanCommit = "0136529cd8d9194f401aa5322bf90e547d1f0b56"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	fileConfig, err := appconfig.LoadConfig()
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}
	db, err := database.InitDB(database.DefaultConfig(), database.AllModels()...)
	if err != nil {
		log.Fatalf("initialize database: %v", err)
	}
	defer database.Close()

	redisURL, prefix := quantDragonflyConfig(fileConfig)
	queue, err := cache.NewDragonflyQuantQueue(redisURL, prefix)
	if err != nil {
		log.Fatalf("initialize Dragonfly queue: %v", err)
	}
	defer queue.Close()
	if err := queue.Ping(ctx); err != nil {
		log.Fatalf("connect Dragonfly queue: %v", err)
	}
	group := envOrDefault("LEAN_WORKER_GROUP", "lean-workers")
	consumer := envOrDefault("LEAN_WORKER_CONSUMER", hostname())
	if err := queue.EnsureConsumerGroup(ctx, group); err != nil {
		log.Fatalf("ensure consumer group: %v", err)
	}

	store := service.NewQuantResearchStore(db)
	log.Printf("Lean worker ready | group=%s consumer=%s engine=%s", group, consumer, leanEngineVersion())
	for ctx.Err() == nil {
		staleAfter := envDurationMinutes("LEAN_STALE_CLAIM_MINUTES", 35)
		messages, readErr := queue.ClaimStaleBacktests(ctx, group, consumer, staleAfter, 1)
		if readErr == nil && len(messages) == 0 {
			messages, readErr = queue.ReadBacktests(ctx, group, consumer, 1, 5*time.Second)
		}
		if readErr != nil {
			if errors.Is(readErr, context.Canceled) {
				break
			}
			log.Printf("read queue: %v", readErr)
			time.Sleep(time.Second)
			continue
		}
		for _, message := range messages {
			processBacktest(ctx, store, queue, group, message)
		}
	}
}

func processBacktest(parent context.Context, store *service.QuantResearchStore, queue *cache.DragonflyQuantQueue, group string, message cache.QuantQueueMessage) {
	if message.Recovered {
		staleBefore := time.Now().Add(-envDurationMinutes("LEAN_STALE_CLAIM_MINUTES", 35))
		if err := store.RequeueStaleBacktestJob(parent, message.JobID, staleBefore); err != nil {
			log.Printf("recover stale job %s: %v", message.JobID, err)
			return
		}
	}
	claimed, err := store.ClaimBacktestJob(parent, message.JobID, leanEngineVersion())
	if err != nil {
		log.Printf("claim job %s: %v", message.JobID, err)
		return
	}
	if !claimed {
		_ = queue.AckBacktest(parent, group, message.ID)
		return
	}
	timeout := envDurationMinutes("LEAN_JOB_TIMEOUT_MINUTES", 30)
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	root := envOrDefault("LEAN_JOB_ROOT", "/var/lib/fundlive/lean-jobs")
	workDir, _, err := store.ExportLeanJob(ctx, message.JobID, root, leanEngineVersion())
	if err != nil {
		_ = store.FailBacktestJob(parent, message.JobID, err, "")
		_ = queue.AckBacktest(parent, group, message.ID)
		return
	}
	output, runErr := runLean(ctx, workDir)
	logSummary := truncate(string(output), 20000)
	if runErr != nil {
		_ = store.FailBacktestJob(parent, message.JobID, fmt.Errorf("run Lean: %w", runErr), logSummary)
		_ = queue.AckBacktest(parent, group, message.ID)
		return
	}
	result, resultErr := readLeanResult(filepath.Join(workDir, "output"), logSummary)
	if resultErr != nil {
		_ = store.FailBacktestJob(parent, message.JobID, resultErr, logSummary)
		_ = queue.AckBacktest(parent, group, message.ID)
		return
	}
	result.EngineVersion = leanEngineVersion()
	if err := store.CompleteBacktestJob(parent, message.JobID, result); err != nil {
		log.Printf("complete job %s: %v", message.JobID, err)
		return
	}
	_ = queue.AckBacktest(parent, group, message.ID)
	log.Printf("Lean backtest completed | job=%s", message.JobID)
}

func runLean(ctx context.Context, workDir string) ([]byte, error) {
	launcher := envOrDefault("LEAN_LAUNCHER_PATH", "/Lean/Launcher/bin/Release/QuantConnect.Lean.Launcher.dll")
	leanDataRoot := envOrDefault("LEAN_DATA_ROOT", "/Lean/Data")
	outputDir := filepath.Join(workDir, "output")
	command := exec.CommandContext(ctx, "dotnet", launcher,
		"--algorithm-type-name", "FundLiveTop5WeeklyAlgorithm",
		"--algorithm-language", "CSharp",
		"--data-folder", leanDataRoot,
		"--results-destination-folder", outputDir,
	)
	command.Dir = filepath.Dir(launcher)
	command.Env = append(os.Environ(), "FUNDLIVE_LEAN_JOB_DIR="+workDir)
	return command.CombinedOutput()
}

func readLeanResult(outputDir, logSummary string) (service.LeanBacktestResult, error) {
	var selected string
	var selectedSize int64
	err := filepath.WalkDir(outputDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			return walkErr
		}
		info, infoErr := entry.Info()
		if infoErr == nil && info.Size() > selectedSize {
			selected, selectedSize = path, info.Size()
		}
		return nil
	})
	if err != nil {
		return service.LeanBacktestResult{}, err
	}
	if selected == "" {
		return service.LeanBacktestResult{}, fmt.Errorf("Lean produced no JSON result")
	}
	payload, err := os.ReadFile(selected)
	if err != nil {
		return service.LeanBacktestResult{}, err
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(payload, &root); err != nil {
		return service.LeanBacktestResult{}, fmt.Errorf("decode Lean result: %w", err)
	}
	result := service.LeanBacktestResult{LogSummary: logSummary}
	result.Metrics = firstRaw(root, "Statistics", "statistics", "RuntimeStatistics", "runtimeStatistics")
	result.EquityCurve = firstRaw(root, "Charts", "charts")
	result.Trades = firstRaw(root, "Orders", "orders", "TotalPerformance", "totalPerformance")
	result.Benchmarks = result.EquityCurve
	if len(result.Metrics) == 0 {
		result.Metrics = payload
	}
	return result, nil
}

func firstRaw(values map[string]json.RawMessage, keys ...string) json.RawMessage {
	for _, key := range keys {
		if value := values[key]; len(value) > 0 {
			return value
		}
	}
	return nil
}

func quantDragonflyConfig(config *appconfig.Config) (string, string) {
	url := "redis://127.0.0.1:16380/0"
	prefix := "fundlive"
	if config != nil {
		if config.Cache.RedisURL != "" {
			url = config.Cache.RedisURL
		}
		if config.Cache.KeyPrefix != "" {
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

func leanEngineVersion() string { return envOrDefault("LEAN_ENGINE_VERSION", defaultLeanCommit) }

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func envDurationMinutes(name string, fallback int) time.Duration {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(name)))
	if err != nil || value <= 0 {
		value = fallback
	}
	return time.Duration(value) * time.Minute
}

func hostname() string {
	name, err := os.Hostname()
	if err != nil || strings.TrimSpace(name) == "" {
		return "lean-worker"
	}
	return name
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[len(value)-limit:]
}
