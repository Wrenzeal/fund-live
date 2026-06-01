package database

var fundSectorMigrationStatements = []string{
	`CREATE TABLE IF NOT EXISTS fund_sectors (
		code varchar(50) PRIMARY KEY,
		name varchar(100) NOT NULL,
		parent_code varchar(50),
		level integer NOT NULL DEFAULT 1,
		sort_order integer NOT NULL DEFAULT 0,
		is_enabled boolean NOT NULL DEFAULT true,
		created_at timestamptz DEFAULT now(),
		updated_at timestamptz DEFAULT now()
	)`,
	`CREATE INDEX IF NOT EXISTS idx_fund_sectors_parent_code ON fund_sectors (parent_code)`,
	`CREATE TABLE IF NOT EXISTS instrument_sector_map (
		id bigserial PRIMARY KEY,
		instrument_code varchar(20) NOT NULL,
		exchange varchar(8) NOT NULL,
		sector_code varchar(50) NOT NULL,
		source varchar(50) NOT NULL,
		created_at timestamptz DEFAULT now(),
		updated_at timestamptz DEFAULT now()
	)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_instrument_sector_map_code_exchange ON instrument_sector_map (instrument_code, exchange)`,
	`CREATE INDEX IF NOT EXISTS idx_instrument_sector_map_sector_code ON instrument_sector_map (sector_code)`,
	`CREATE TABLE IF NOT EXISTS fund_sector_snapshots (
		id bigserial PRIMARY KEY,
		fund_id varchar(10) NOT NULL,
		as_of_date date NOT NULL,
		primary_sector_code varchar(50) NOT NULL,
		source varchar(50) NOT NULL,
		confidence varchar(20) NOT NULL,
		created_at timestamptz DEFAULT now(),
		updated_at timestamptz DEFAULT now()
	)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_fund_sector_snapshot_fund_date ON fund_sector_snapshots (fund_id, as_of_date)`,
	`CREATE INDEX IF NOT EXISTS idx_fund_sector_snapshot_primary_sector_code ON fund_sector_snapshots (primary_sector_code)`,
	`CREATE TABLE IF NOT EXISTS fund_sector_breakdown (
		id bigserial PRIMARY KEY,
		fund_id varchar(10) NOT NULL,
		as_of_date date NOT NULL,
		sector_code varchar(50) NOT NULL,
		weight_percent numeric(8,4) NOT NULL,
		rank integer NOT NULL DEFAULT 0,
		created_at timestamptz DEFAULT now(),
		updated_at timestamptz DEFAULT now()
	)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_fund_sector_breakdown_fund_date_sector ON fund_sector_breakdown (fund_id, as_of_date, sector_code)`,
	`CREATE INDEX IF NOT EXISTS idx_fund_sector_breakdown_rank ON fund_sector_breakdown (fund_id, as_of_date, rank)`,
}

var fundCategoryMigrationStatements = []string{
	`ALTER TABLE funds ADD COLUMN IF NOT EXISTS category_code varchar(50)`,
	`CREATE INDEX IF NOT EXISTS idx_funds_category_code ON funds (category_code)`,
	`CREATE TABLE IF NOT EXISTS fund_categories (
		code varchar(50) PRIMARY KEY,
		name varchar(100) NOT NULL,
		description text,
		sort_order integer NOT NULL DEFAULT 0,
		is_enabled boolean NOT NULL DEFAULT true,
		created_at timestamptz DEFAULT now(),
		updated_at timestamptz DEFAULT now()
	)`,
	`CREATE TABLE IF NOT EXISTS fund_classification_overrides (
		id bigserial PRIMARY KEY,
		fund_id varchar(10) NOT NULL UNIQUE,
		category_code varchar(50),
		primary_sector_code varchar(50),
		primary_theme_code varchar(50),
		manual_tags_json text,
		sector_tags_json text,
		note text,
		updated_by varchar(100),
		created_at timestamptz DEFAULT now(),
		updated_at timestamptz DEFAULT now()
	)`,
	`ALTER TABLE fund_classification_overrides ADD COLUMN IF NOT EXISTS primary_theme_code varchar(50)`,
	`ALTER TABLE fund_classification_overrides ADD COLUMN IF NOT EXISTS manual_tags_json text`,
	`CREATE INDEX IF NOT EXISTS idx_fund_classification_overrides_category_code ON fund_classification_overrides (category_code)`,
	`CREATE INDEX IF NOT EXISTS idx_fund_classification_overrides_primary_sector_code ON fund_classification_overrides (primary_sector_code)`,
	`CREATE INDEX IF NOT EXISTS idx_fund_classification_overrides_primary_theme_code ON fund_classification_overrides (primary_theme_code)`,
}

var fundClassificationOverrideManualTagsMigrationStatements = []string{
	`ALTER TABLE fund_classification_overrides ADD COLUMN IF NOT EXISTS primary_theme_code varchar(50)`,
	`ALTER TABLE fund_classification_overrides ADD COLUMN IF NOT EXISTS manual_tags_json text`,
	`CREATE INDEX IF NOT EXISTS idx_fund_classification_overrides_primary_theme_code ON fund_classification_overrides (primary_theme_code)`,
}
