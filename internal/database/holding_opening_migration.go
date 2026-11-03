package database

var holdingOpeningMigrationStatements = []string{
	`ALTER TABLE tb_user_fund_holding ADD COLUMN IF NOT EXISTS entry_type varchar(16) NOT NULL DEFAULT 'buy'`,
}
