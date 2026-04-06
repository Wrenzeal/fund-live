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
		ID:             "20260405_vip_tables",
		RequiredTables: []string{},
		Statements:     vipTableMigrationStatements,
	},
	{
		ID:             "20260405_vip_payment_orders",
		RequiredTables: []string{},
		Statements:     vipPaymentMigrationStatements,
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
