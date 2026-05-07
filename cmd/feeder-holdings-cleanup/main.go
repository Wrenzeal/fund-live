package main

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/database"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type residualRow struct {
	FeederCode         string
	FeederName         string
	TargetCode         string
	TargetName         string
	FeederRows         int64
	FeederTotalRatio   float64
	FeederLatestPeriod string
	TargetRows         int64
	TargetTotalRatio   float64
	TargetLatestPeriod string
	Recommended        bool
	Reason             string
}

type backupRow struct {
	FundID          string
	StockCode       string
	StockName       string
	Exchange        string
	HoldingRatio    string
	HoldingShares   string
	MarketValue     string
	ReportingPeriod string
	CreatedAt       string
	UpdatedAt       string
}

const defaultOutputDir = ".omx/backups/manual"

func main() {
	action := flag.String("action", "scan", "one of: scan, backup, cleanup")
	outputDir := flag.String("output-dir", defaultOutputDir, "directory for generated artifacts")
	inputFile := flag.String("input-file", "", "input whitelist csv for backup/cleanup actions")
	execute := flag.Bool("execute", false, "for cleanup: actually delete holdings instead of dry-run")
	flag.Parse()

	cfg := database.DefaultConfig()
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}

	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		log.Fatalf("mkdir output dir: %v", err)
	}

	switch strings.ToLower(strings.TrimSpace(*action)) {
	case "scan":
		if err := runScan(db, *outputDir); err != nil {
			log.Fatal(err)
		}
	case "backup":
		if err := runBackup(db, *outputDir, *inputFile); err != nil {
			log.Fatal(err)
		}
	case "cleanup":
		if err := runCleanup(db, *inputFile, *execute); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("unsupported action: %s", *action)
	}
}

func runScan(db *gorm.DB, outputDir string) error {
	rows, err := scanResidualRows(db)
	if err != nil {
		return err
	}

	day := shanghaiNow().Format("2006-01-02")
	auditPath := filepath.Join(outputDir, fmt.Sprintf("feeder-residual-holdings-audit-%s.csv", day))
	whitelistPath := filepath.Join(outputDir, fmt.Sprintf("feeder-residual-holdings-whitelist-%s.csv", day))

	if err := writeResidualAuditCSV(auditPath, rows); err != nil {
		return err
	}
	if err := writeWhitelistCSV(whitelistPath, rows); err != nil {
		return err
	}

	var lt1, mid, ge20, recommended int
	for _, row := range rows {
		switch {
		case row.FeederTotalRatio < 1:
			lt1++
		case row.FeederTotalRatio < 20:
			mid++
		default:
			ge20++
		}
		if row.Recommended {
			recommended++
		}
	}

	fmt.Printf("scan complete\n")
	fmt.Printf("residual_feeders=%d\n", len(rows))
	fmt.Printf("residual_lt1=%d residual_1to20=%d residual_ge20=%d\n", lt1, mid, ge20)
	fmt.Printf("recommended_candidates=%d\n", recommended)
	fmt.Printf("audit_csv=%s\n", auditPath)
	fmt.Printf("whitelist_csv=%s\n", whitelistPath)
	return nil
}

func runBackup(db *gorm.DB, outputDir, inputFile string) error {
	if inputFile == "" {
		return errors.New("backup requires -input-file")
	}

	codes, err := readSelectedCodes(inputFile)
	if err != nil {
		return err
	}
	if len(codes) == 0 {
		return fmt.Errorf("no selected feeder codes found in %s", inputFile)
	}

	var rows []backupRow
	err = db.Raw(`
SELECT
  fund_id,
  stock_code,
  stock_name,
  exchange,
  holding_ratio::text AS holding_ratio,
  holding_shares::text AS holding_shares,
  market_value::text AS market_value,
  reporting_period,
  created_at::text AS created_at,
  updated_at::text AS updated_at
FROM stock_holdings
WHERE fund_id IN ?
ORDER BY fund_id ASC, reporting_period DESC, holding_ratio DESC, stock_code ASC
`, codes).Scan(&rows).Error
	if err != nil {
		return fmt.Errorf("query backup rows: %w", err)
	}

	day := shanghaiNow().Format("2006-01-02")
	backupPath := filepath.Join(outputDir, fmt.Sprintf("feeder-residual-holdings-backup-%s.csv", day))
	file, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("create backup file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	if err := writer.Write([]string{
		"fund_id", "stock_code", "stock_name", "exchange", "holding_ratio",
		"holding_shares", "market_value", "reporting_period", "created_at", "updated_at",
	}); err != nil {
		return err
	}
	for _, row := range rows {
		if err := writer.Write([]string{
			row.FundID,
			row.StockCode,
			row.StockName,
			row.Exchange,
			row.HoldingRatio,
			row.HoldingShares,
			row.MarketValue,
			row.ReportingPeriod,
			row.CreatedAt,
			row.UpdatedAt,
		}); err != nil {
			return err
		}
	}

	fmt.Printf("backup complete\nselected_codes=%d\nbacked_up_rows=%d\nbackup_csv=%s\n", len(codes), len(rows), backupPath)
	return nil
}

func runCleanup(db *gorm.DB, inputFile string, execute bool) error {
	if inputFile == "" {
		return errors.New("cleanup requires -input-file")
	}

	codes, err := readSelectedCodes(inputFile)
	if err != nil {
		return err
	}
	if len(codes) == 0 {
		return fmt.Errorf("no selected feeder codes found in %s", inputFile)
	}

	var count int64
	if err := db.Raw(`SELECT COUNT(*) FROM stock_holdings WHERE fund_id IN ?`, codes).Scan(&count).Error; err != nil {
		return fmt.Errorf("count cleanup rows: %w", err)
	}

	if !execute {
		fmt.Printf("cleanup dry-run\nselected_codes=%d\nrows_to_delete=%d\n", len(codes), count)
		return nil
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		return tx.Exec(`DELETE FROM stock_holdings WHERE fund_id IN ?`, codes).Error
	}); err != nil {
		return fmt.Errorf("execute cleanup: %w", err)
	}

	fmt.Printf("cleanup executed\nselected_codes=%d\ndeleted_rows=%d\n", len(codes), count)
	return nil
}

func scanResidualRows(db *gorm.DB) ([]residualRow, error) {
	rows := []residualRow{}
	err := db.Raw(`
WITH feeder AS (
  SELECT fund_id, COUNT(*) AS feeder_rows, COALESCE(SUM(holding_ratio),0) AS feeder_total_ratio, MAX(reporting_period) AS feeder_latest_period
  FROM stock_holdings
  GROUP BY fund_id
),
target AS (
  SELECT fund_id, COUNT(*) AS target_rows, COALESCE(SUM(holding_ratio),0) AS target_total_ratio, MAX(reporting_period) AS target_latest_period
  FROM stock_holdings
  GROUP BY fund_id
)
SELECT
  fm.feeder_code,
  COALESCE(ff.name, fm.feeder_name) AS feeder_name,
  fm.target_code,
  COALESCE(tf.name, fm.target_name) AS target_name,
  COALESCE(feeder.feeder_rows, 0) AS feeder_rows,
  COALESCE(feeder.feeder_total_ratio, 0) AS feeder_total_ratio,
  COALESCE(feeder.feeder_latest_period, '') AS feeder_latest_period,
  COALESCE(target.target_rows, 0) AS target_rows,
  COALESCE(target.target_total_ratio, 0) AS target_total_ratio,
  COALESCE(target.target_latest_period, '') AS target_latest_period
FROM fund_mappings fm
LEFT JOIN funds ff ON ff.id = fm.feeder_code
LEFT JOIN funds tf ON tf.id = fm.target_code
LEFT JOIN feeder ON feeder.fund_id = fm.feeder_code
LEFT JOIN target ON target.fund_id = fm.target_code
WHERE fm.is_resolved = true AND fm.target_code <> '' AND COALESCE(feeder.feeder_total_ratio, 0) > 0
ORDER BY feeder.feeder_total_ratio DESC, fm.feeder_code ASC
`).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("scan residual rows: %w", err)
	}

	for i := range rows {
		rows[i].Recommended, rows[i].Reason = classifyRecommendation(rows[i])
	}
	return rows, nil
}

func classifyRecommendation(row residualRow) (bool, string) {
	switch {
	case row.FeederTotalRatio < 1:
		return true, "residual_ratio_lt_1"
	case row.FeederTotalRatio <= 5 &&
		row.FeederLatestPeriod != "" &&
		row.TargetLatestPeriod != "" &&
		row.FeederLatestPeriod < row.TargetLatestPeriod:
		return true, "stale_and_low_coverage"
	default:
		return false, ""
	}
}

func writeResidualAuditCSV(path string, rows []residualRow) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create audit csv: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{
		"feeder_code", "feeder_name", "target_code", "target_name",
		"feeder_rows", "feeder_total_ratio", "feeder_latest_period",
		"target_rows", "target_total_ratio", "target_latest_period",
		"recommended", "reason",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	for _, row := range rows {
		if err := writer.Write([]string{
			row.FeederCode,
			row.FeederName,
			row.TargetCode,
			row.TargetName,
			strconv.FormatInt(row.FeederRows, 10),
			fmt.Sprintf("%.2f", row.FeederTotalRatio),
			row.FeederLatestPeriod,
			strconv.FormatInt(row.TargetRows, 10),
			fmt.Sprintf("%.2f", row.TargetTotalRatio),
			row.TargetLatestPeriod,
			strconv.FormatBool(row.Recommended),
			row.Reason,
		}); err != nil {
			return err
		}
	}
	return nil
}

func writeWhitelistCSV(path string, rows []residualRow) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create whitelist csv: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{
		"selected_for_cleanup", "feeder_code", "feeder_name", "target_code", "target_name",
		"feeder_total_ratio", "feeder_rows", "feeder_latest_period", "reason",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	recommended := make([]residualRow, 0, len(rows))
	for _, row := range rows {
		if row.Recommended {
			recommended = append(recommended, row)
		}
	}
	sort.Slice(recommended, func(i, j int) bool {
		if recommended[i].FeederTotalRatio != recommended[j].FeederTotalRatio {
			return recommended[i].FeederTotalRatio < recommended[j].FeederTotalRatio
		}
		return recommended[i].FeederCode < recommended[j].FeederCode
	})

	for _, row := range recommended {
		if err := writer.Write([]string{
			"true",
			row.FeederCode,
			row.FeederName,
			row.TargetCode,
			row.TargetName,
			fmt.Sprintf("%.2f", row.FeederTotalRatio),
			strconv.FormatInt(row.FeederRows, 10),
			row.FeederLatestPeriod,
			row.Reason,
		}); err != nil {
			return err
		}
	}
	return nil
}

func readSelectedCodes(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open whitelist csv: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read whitelist csv: %w", err)
	}
	if len(records) <= 1 {
		return nil, nil
	}

	headerIndex := make(map[string]int, len(records[0]))
	for idx, col := range records[0] {
		headerIndex[strings.TrimSpace(col)] = idx
	}
	selectedIdx, okSelected := headerIndex["selected_for_cleanup"]
	codeIdx, okCode := headerIndex["feeder_code"]
	if !okSelected || !okCode {
		return nil, fmt.Errorf("whitelist csv missing required columns")
	}

	codes := make([]string, 0, len(records)-1)
	for _, row := range records[1:] {
		if len(row) <= codeIdx || len(row) <= selectedIdx {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(row[selectedIdx]), "true") {
			codes = append(codes, strings.TrimSpace(row[codeIdx]))
		}
	}
	return codes, nil
}

func shanghaiNow() time.Time {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.Now()
	}
	return time.Now().In(loc)
}
