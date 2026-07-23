package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/database"
	"gorm.io/gorm"
)

type LeanJobManifest struct {
	JobID            string               `json:"job_id"`
	Strategy         string               `json:"strategy"`
	UniverseVersion  string               `json:"universe_version"`
	SignalMode       string               `json:"signal_mode"`
	EngineVersion    string               `json:"engine_version"`
	Parameters       QuantBacktestRequest `json:"parameters"`
	Symbols          []string             `json:"symbols"`
	BenchmarkSymbols []string             `json:"benchmark_symbols"`
	SignalTiming     string               `json:"signal_timing"`
	GeneratedAt      time.Time            `json:"generated_at"`
}

type LeanBacktestResult struct {
	EngineVersion string          `json:"engine_version"`
	Metrics       json.RawMessage `json:"metrics"`
	EquityCurve   json.RawMessage `json:"equity_curve"`
	Trades        json.RawMessage `json:"trades"`
	Benchmarks    json.RawMessage `json:"benchmarks"`
	LogSummary    string          `json:"log_summary"`
}

func (s *QuantResearchStore) ClaimBacktestJob(ctx context.Context, jobID, engineVersion string) (bool, error) {
	if s == nil || s.db == nil {
		return false, fmt.Errorf("quant research store is unavailable")
	}
	now := time.Now()
	result := s.db.WithContext(ctx).Model(&database.QuantBacktestJob{}).
		Where("id = ? AND status = ?", strings.TrimSpace(jobID), "queued").
		Updates(map[string]interface{}{
			"status":         "running",
			"engine_version": strings.TrimSpace(engineVersion),
			"started_at":     now,
			"attempt_count":  gorm.Expr("attempt_count + 1"),
			"error_message":  "",
		})
	return result.RowsAffected == 1, result.Error
}

func (s *QuantResearchStore) RequeueStaleBacktestJob(ctx context.Context, jobID string, staleBefore time.Time) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("quant research store is unavailable")
	}
	return s.db.WithContext(ctx).Model(&database.QuantBacktestJob{}).
		Where("id = ? AND status = ? AND started_at < ?", strings.TrimSpace(jobID), "running", staleBefore).
		Updates(map[string]interface{}{
			"status":        "queued",
			"error_message": "recovered after stale worker claim",
			"started_at":    nil,
		}).Error
}

func (s *QuantResearchStore) CompleteBacktestJob(ctx context.Context, jobID string, result LeanBacktestResult) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("quant research store is unavailable")
	}
	now := time.Now()
	return s.db.WithContext(ctx).Model(&database.QuantBacktestJob{}).
		Where("id = ? AND status = ?", strings.TrimSpace(jobID), "running").
		Updates(map[string]interface{}{
			"status":          "completed",
			"engine_version":  result.EngineVersion,
			"metrics_json":    nullJSON(result.Metrics),
			"equity_json":     nullJSON(result.EquityCurve),
			"trades_json":     nullJSON(result.Trades),
			"benchmarks_json": nullJSON(result.Benchmarks),
			"log_summary":     result.LogSummary,
			"completed_at":    now,
		}).Error
}

func nullJSON(value json.RawMessage) interface{} {
	if len(value) == 0 {
		return nil
	}
	return value
}

func (s *QuantResearchStore) FailBacktestJob(ctx context.Context, jobID string, cause error, logSummary string) error {
	if s == nil || s.db == nil {
		return nil
	}
	message := "Lean backtest failed"
	if cause != nil {
		message = cause.Error()
	}
	now := time.Now()
	return s.db.WithContext(ctx).Model(&database.QuantBacktestJob{}).
		Where("id = ? AND status = ?", strings.TrimSpace(jobID), "running").
		Updates(map[string]interface{}{
			"status":        "failed",
			"error_message": message,
			"log_summary":   logSummary,
			"completed_at":  now,
		}).Error
}

func (s *QuantResearchStore) ExportLeanJob(ctx context.Context, jobID, root, engineVersion string) (string, *LeanJobManifest, error) {
	if s == nil || s.db == nil {
		return "", nil, fmt.Errorf("quant research store is unavailable")
	}
	job, err := s.GetBacktestJob(ctx, jobID)
	if err != nil {
		return "", nil, err
	}
	if job.Status != "running" {
		return "", nil, fmt.Errorf("job %s is not running", job.ID)
	}
	var request QuantBacktestRequest
	if err := json.Unmarshal(job.ParametersJSON, &request); err != nil {
		return "", nil, fmt.Errorf("decode backtest parameters: %w", err)
	}
	start, _ := time.Parse("2006-01-02", request.StartDate)
	end, _ := time.Parse("2006-01-02", request.EndDate)

	var members []database.QuantUniverseMember
	if err := s.db.WithContext(ctx).Where("universe_version = ?", job.UniverseVersion).Order("bucket, symbol").Find(&members).Error; err != nil {
		return "", nil, err
	}
	if len(members) == 0 {
		return "", nil, fmt.Errorf("universe %s has no members", job.UniverseVersion)
	}
	symbols := make([]string, 0, len(members))
	for _, member := range members {
		symbols = append(symbols, member.Symbol)
	}

	workDir := filepath.Join(root, job.ID)
	marketDir := filepath.Join(workDir, "data", "fundlive", "market")
	signalDir := filepath.Join(workDir, "data", "fundlive", "signals")
	outputDir := filepath.Join(workDir, "output")
	for _, directory := range []string{marketDir, signalDir, outputDir} {
		if err := os.MkdirAll(directory, 0o750); err != nil {
			return "", nil, err
		}
	}
	for _, symbol := range append(append([]string(nil), symbols...), "000300") {
		var bars []database.QuantMarketBar
		if err := s.db.WithContext(ctx).
			Where("symbol = ? AND date BETWEEN ? AND ?", symbol, start.AddDate(0, 0, -90), end.AddDate(0, 0, 25)).
			Order("date").Find(&bars).Error; err != nil {
			return "", nil, err
		}
		if len(bars) == 0 {
			return "", nil, fmt.Errorf("missing market bars for %s", symbol)
		}
		if err := writeLeanMarketCSV(filepath.Join(marketDir, symbol+".csv"), bars); err != nil {
			return "", nil, err
		}
	}
	for _, symbol := range symbols {
		var signals []database.QuantSignalHistory
		if err := s.db.WithContext(ctx).
			Where("fund_id = ? AND mode = ? AND signal_date BETWEEN ? AND ?", symbol, job.SignalMode, start, end).
			Order("signal_date").Find(&signals).Error; err != nil {
			return "", nil, err
		}
		if err := writeLeanSignalCSV(filepath.Join(signalDir, symbol+".csv"), signals); err != nil {
			return "", nil, err
		}
	}

	manifest := &LeanJobManifest{
		JobID:            job.ID,
		Strategy:         job.Strategy,
		UniverseVersion:  job.UniverseVersion,
		SignalMode:       job.SignalMode,
		EngineVersion:    engineVersion,
		Parameters:       request,
		Symbols:          symbols,
		BenchmarkSymbols: []string{"000300", "pilot_equal_weight", "cash"},
		SignalTiming:     "Friday close signal; next trading day market order",
		GeneratedAt:      time.Now(),
	}
	manifestJSON, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(filepath.Join(workDir, "job.json"), manifestJSON, 0o640); err != nil {
		return "", nil, err
	}
	return workDir, manifest, nil
}

func writeLeanMarketCSV(path string, bars []database.QuantMarketBar) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	for _, bar := range bars {
		if err := writer.Write([]string{
			bar.Date.Format("2006-01-02"), bar.Open.String(), bar.High.String(), bar.Low.String(), bar.Close.String(),
			bar.AdjustedClose.String(), bar.Volume.String(), bar.Amount.String(), bar.AdjustFactor.String(),
		}); err != nil {
			return err
		}
	}
	return writer.Error()
}

func writeLeanSignalCSV(path string, signals []database.QuantSignalHistory) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	for index, signal := range signals {
		rebalance := "0"
		if index == len(signals)-1 || !sameISOWeek(signal.SignalDate, signals[index+1].SignalDate) {
			rebalance = "1"
		}
		if err := writer.Write([]string{
			signal.SignalDate.Format("2006-01-02"), signal.TotalScore.String(), signal.ShadowEventScore.String(), signal.DecisionAt.UTC().Format(time.RFC3339Nano), rebalance,
		}); err != nil {
			return err
		}
	}
	return writer.Error()
}

func sameISOWeek(left, right time.Time) bool {
	leftYear, leftWeek := left.ISOWeek()
	rightYear, rightWeek := right.ISOWeek()
	return leftYear == rightYear && leftWeek == rightWeek
}
