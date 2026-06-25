package database

import (
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestInitDBCreatesCoreSchemaWithoutAutoMigrate(t *testing.T) {
	adminCfg := DefaultConfig()
	adminCfg.DBName = "postgres"
	adminCfg.AutoMigrate = false

	adminDB, err := gorm.Open(postgres.Open(adminCfg.DSN()), &gorm.Config{})
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}

	tempDBName := fmt.Sprintf("fund_migrate_test_%d", time.Now().UnixNano())
	if err := adminDB.Exec(`CREATE DATABASE ` + tempDBName).Error; err != nil {
		t.Skipf("create temp database failed: %v", err)
	}
	defer func() {
		_ = Close()
		_ = adminDB.Exec(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = ? AND pid <> pg_backend_pid()`, tempDBName).Error
		_ = adminDB.Exec(`DROP DATABASE IF EXISTS ` + tempDBName).Error
	}()

	cfg := DefaultConfig()
	cfg.DBName = tempDBName
	cfg.AutoMigrate = false

	db, err := InitDB(cfg, AllModels()...)
	if err != nil {
		t.Fatalf("InitDB() error = %v", err)
	}

	requiredTables := []string{
		schemaMigrationsTable,
		"funds",
		"fund_categories",
		"fund_classification_overrides",
		"fund_estimate_capabilities",
		"fund_sectors",
		"fund_themes",
		"instrument_sector_map",
		"instrument_theme_map",
		"fund_sector_snapshots",
		"fund_sector_breakdown",
		"fund_theme_snapshots",
		"fund_theme_breakdown",
		"stock_holdings",
		"fund_time_series",
		"fund_history",
		"fund_valuation_profiles",
		"fund_mappings",
		"tb_user",
		"tb_user_session",
		"tb_user_favorite_fund",
		"tb_user_watchlist_group",
		"tb_user_watchlist_fund",
		"tb_user_holding_override",
		"tb_user_fund_holding",
		"tb_user_fund_holding_transaction",
		"issues",
		"announcements",
	}

	for _, table := range requiredTables {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("expected table %s to exist after InitDB with auto_migrate=false", table)
		}
	}

	appliedMigrations := []string{
		"20260409_core_fund_tables",
		"20260409_core_user_tables",
		"20260418_fund_sector_tables",
		"20260419_fund_category_tables",
		"20260422_fund_theme_tables",
		"20260530_fund_catalog_status",
		"20260529_fund_classification_override_manual_tags",
		"20260421_fund_estimate_capability_tables",
		"20260416_user_holding_confirmation",
		"20260525_user_holding_manual_confirmation",
		"20260525_user_holding_transactions",
		"20260525_user_holding_transaction_voids",
		"20260526_user_holding_sources",
		"20260423_user_watchlist_group_ordering",
		"20260404_fund_search_indexes",
		"20260404_fund_history_unique_index",
		"20260404_fund_time_series_unique_index",
		"20260406_issue_tables",
		"20260406_announcement_tables",
	}

	for _, id := range appliedMigrations {
		applied, err := migrationApplied(db, id)
		if err != nil {
			t.Fatalf("migrationApplied(%s) error = %v", id, err)
		}
		if !applied {
			t.Fatalf("expected migration %s to be recorded as applied", id)
		}
	}
}
