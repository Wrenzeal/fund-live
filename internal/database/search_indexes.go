package database

var fundSearchIndexStatements = []string{
	`CREATE EXTENSION IF NOT EXISTS pg_trgm`,
	`CREATE INDEX IF NOT EXISTS idx_funds_id_pattern ON funds (id varchar_pattern_ops)`,
	`CREATE INDEX IF NOT EXISTS idx_funds_name_trgm ON funds USING gin (name gin_trgm_ops)`,
	`CREATE INDEX IF NOT EXISTS idx_funds_manager_trgm ON funds USING gin (manager gin_trgm_ops)`,
}
