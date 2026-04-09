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
		"issues",
		"announcements",
		"user_memberships",
		"vip_orders",
	}

	for _, table := range requiredTables {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("expected table %s to exist after InitDB with auto_migrate=false", table)
		}
	}

	appliedMigrations := []string{
		"20260409_core_fund_tables",
		"20260409_core_user_tables",
		"20260404_fund_search_indexes",
		"20260404_fund_history_unique_index",
		"20260404_fund_time_series_unique_index",
		"20260406_issue_tables",
		"20260406_announcement_tables",
		"20260405_vip_tables",
		"20260405_vip_payment_orders",
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
