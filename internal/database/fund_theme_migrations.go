package database

var fundThemeMigrationStatements = []string{
	`CREATE TABLE IF NOT EXISTS fund_themes (
		code varchar(50) PRIMARY KEY,
		name varchar(100) NOT NULL,
		parent_code varchar(50),
		level integer NOT NULL DEFAULT 1,
		sort_order integer NOT NULL DEFAULT 0,
		is_enabled boolean NOT NULL DEFAULT true,
		created_at timestamptz DEFAULT now(),
		updated_at timestamptz DEFAULT now()
	)`,
	`CREATE INDEX IF NOT EXISTS idx_fund_themes_parent_code ON fund_themes (parent_code)`,
	`CREATE TABLE IF NOT EXISTS instrument_theme_map (
		id bigserial PRIMARY KEY,
		instrument_code varchar(20) NOT NULL,
		exchange varchar(8) NOT NULL,
		theme_code varchar(50) NOT NULL,
		source varchar(50) NOT NULL,
		weight numeric(8,4) NOT NULL DEFAULT 1.0000,
		created_at timestamptz DEFAULT now(),
		updated_at timestamptz DEFAULT now()
	)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_instrument_theme_map_code_exchange_theme ON instrument_theme_map (instrument_code, exchange, theme_code)`,
	`CREATE INDEX IF NOT EXISTS idx_instrument_theme_map_theme_code ON instrument_theme_map (theme_code)`,
	`CREATE TABLE IF NOT EXISTS fund_theme_snapshots (
		id bigserial PRIMARY KEY,
		fund_id varchar(10) NOT NULL,
		as_of_date date NOT NULL,
		primary_theme_code varchar(50) NOT NULL,
		source varchar(50) NOT NULL,
		confidence varchar(20) NOT NULL,
		created_at timestamptz DEFAULT now(),
		updated_at timestamptz DEFAULT now()
	)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_fund_theme_snapshot_fund_date ON fund_theme_snapshots (fund_id, as_of_date)`,
	`CREATE INDEX IF NOT EXISTS idx_fund_theme_snapshot_primary_theme_code ON fund_theme_snapshots (primary_theme_code)`,
	`CREATE TABLE IF NOT EXISTS fund_theme_breakdown (
		id bigserial PRIMARY KEY,
		fund_id varchar(10) NOT NULL,
		as_of_date date NOT NULL,
		theme_code varchar(50) NOT NULL,
		weight_percent numeric(8,4) NOT NULL,
		rank integer NOT NULL DEFAULT 0,
		created_at timestamptz DEFAULT now(),
		updated_at timestamptz DEFAULT now()
	)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_fund_theme_breakdown_fund_date_theme ON fund_theme_breakdown (fund_id, as_of_date, theme_code)`,
	`CREATE INDEX IF NOT EXISTS idx_fund_theme_breakdown_rank ON fund_theme_breakdown (fund_id, as_of_date, rank)`,
}
