package database

var fundCatalogStatusMigrationStatements = []string{
	`ALTER TABLE funds ADD COLUMN IF NOT EXISTS catalog_status varchar(32)`,
	`UPDATE funds SET catalog_status = 'active' WHERE catalog_status IS NULL OR catalog_status = ''`,
	`ALTER TABLE funds ALTER COLUMN catalog_status SET DEFAULT 'active'`,
	`ALTER TABLE funds ALTER COLUMN catalog_status SET NOT NULL`,
	`ALTER TABLE funds ADD COLUMN IF NOT EXISTS catalog_synced_at timestamptz`,
	`CREATE INDEX IF NOT EXISTS idx_funds_catalog_status ON funds (catalog_status)`,
}
