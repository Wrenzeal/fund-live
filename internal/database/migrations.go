package database

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

const schemaMigrationsTable = "schema_migrations"

type sqlMigration struct {
	ID             string
	RequiredTables []string
	Statements     []string
}

var managedMigrations = []sqlMigration{
	{
		ID:             "20260409_core_fund_tables",
		RequiredTables: []string{},
		Statements:     coreFundTableMigrationStatements,
	},
	{
		ID:             "20260409_core_user_tables",
		RequiredTables: []string{},
		Statements:     coreUserTableMigrationStatements,
	},
	{
		ID:             "20260418_fund_sector_tables",
		RequiredTables: []string{},
		Statements:     fundSectorMigrationStatements,
	},
	{
		ID:             "20260419_fund_category_tables",
		RequiredTables: []string{"funds"},
		Statements:     fundCategoryMigrationStatements,
	},
	{
		ID:             "20260422_fund_theme_tables",
		RequiredTables: []string{"funds"},
		Statements:     fundThemeMigrationStatements,
	},
	{
		ID:             "20260530_fund_catalog_status",
		RequiredTables: []string{"funds"},
		Statements:     fundCatalogStatusMigrationStatements,
	},
	{
		ID:             "20260529_fund_classification_override_manual_tags",
		RequiredTables: []string{"fund_classification_overrides"},
		Statements:     fundClassificationOverrideManualTagsMigrationStatements,
	},
	{
		ID:             "20260421_fund_estimate_capability_tables",
		RequiredTables: []string{"funds"},
		Statements:     fundEstimateCapabilityMigrationStatements,
	},
	{
		ID:             "20260427_fund_holding_history",
		RequiredTables: []string{"funds"},
		Statements:     fundHoldingHistoryMigrationStatements,
	},
	{
		ID:             "20260428_fund_analysis_snapshots",
		RequiredTables: []string{"funds"},
		Statements:     fundAnalysisSnapshotMigrationStatements,
	},
	{
		ID:             "20260416_user_holding_confirmation",
		RequiredTables: []string{"tb_user_fund_holding"},
		Statements:     userHoldingConfirmationMigrationStatements,
	},
	{
		ID:             "20260525_user_holding_manual_confirmation",
		RequiredTables: []string{"tb_user_fund_holding"},
		Statements:     userHoldingManualConfirmationMigrationStatements,
	},
	{
		ID:             "20260525_user_holding_transactions",
		RequiredTables: []string{"tb_user_fund_holding"},
		Statements:     userHoldingTransactionMigrationStatements,
	},
	{
		ID:             "20260525_user_holding_transaction_voids",
		RequiredTables: []string{"tb_user_fund_holding_transaction"},
		Statements:     userHoldingTransactionVoidMigrationStatements,
	},
	{
		ID:             "20260526_user_holding_sources",
		RequiredTables: []string{"tb_user_fund_holding", "tb_user_fund_holding_transaction"},
		Statements:     userHoldingSourceMigrationStatements,
	},
	{
		ID:             "20260423_user_watchlist_group_ordering",
		RequiredTables: []string{"tb_user_watchlist_group"},
		Statements:     watchlistGroupOrderingMigrationStatements,
	},
	{
		ID:             "20260406_user_admin_flag",
		RequiredTables: []string{"tb_user"},
		Statements:     adminUserMigrationStatements,
	},
	{
		ID:             "20260406_issue_tables",
		RequiredTables: []string{},
		Statements:     issueMigrationStatements,
	},
	{
		ID:             "20260408_issue_official_reply",
		RequiredTables: []string{"issues"},
		Statements:     issueOfficialReplyMigrationStatements,
	},
	{
		ID:             "20260406_announcement_tables",
		RequiredTables: []string{},
		Statements:     announcementMigrationStatements,
	},
	{
		ID:             "20260404_fund_search_indexes",
		RequiredTables: []string{"funds"},
		Statements:     fundSearchIndexStatements,
	},
	{
		ID:             "20260404_fund_history_unique_index",
		RequiredTables: []string{"fund_history"},
		Statements: []string{
			`CREATE UNIQUE INDEX IF NOT EXISTS uq_fund_history_fund_id_date ON fund_history (fund_id, date)`,
		},
	},
	{
		ID:             "20260404_fund_time_series_unique_index",
		RequiredTables: []string{"fund_time_series"},
		Statements: []string{
			`DELETE FROM fund_time_series a USING fund_time_series b WHERE a.id < b.id AND a.fund_id = b.fund_id AND a."time" = b."time"`,
			`CREATE UNIQUE INDEX IF NOT EXISTS uq_fund_time_series_fund_id_time ON fund_time_series (fund_id, "time")`,
		},
	},
}

func RunDatabaseMigrations(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}

	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id varchar(128) PRIMARY KEY,
			applied_at timestamptz NOT NULL DEFAULT now()
		)
	`).Error; err != nil {
		return fmt.Errorf("create schema migrations table: %w", err)
	}

	for _, migration := range managedMigrations {
		applied, err := migrationApplied(db, migration.ID)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		if !migrationTablesReady(db, migration.RequiredTables) {
			log.Printf("ℹ️ Skipping migration %s because required tables are not ready", migration.ID)
			continue
		}
		if err := applyMigration(db, migration); err != nil {
			return err
		}
	}

	return nil
}

func migrationApplied(db *gorm.DB, id string) (bool, error) {
	var count int64
	if err := db.Table(schemaMigrationsTable).Where("id = ?", id).Count(&count).Error; err != nil {
		return false, fmt.Errorf("check migration %s: %w", id, err)
	}
	return count > 0, nil
}

func migrationTablesReady(db *gorm.DB, tables []string) bool {
	for _, table := range tables {
		if !db.Migrator().HasTable(table) {
			return false
		}
	}
	return true
}

func applyMigration(db *gorm.DB, migration sqlMigration) error {
	for _, stmt := range migration.Statements {
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("apply migration %s: %w", migration.ID, err)
		}
	}

	if err := db.Exec(
		`INSERT INTO schema_migrations (id) VALUES (?) ON CONFLICT (id) DO NOTHING`,
		migration.ID,
	).Error; err != nil {
		return fmt.Errorf("record migration %s: %w", migration.ID, err)
	}

	log.Printf("✅ Database migration applied: %s", migration.ID)
	return nil
}
