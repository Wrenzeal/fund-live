package database

var userHoldingConfirmationMigrationStatements = []string{
	`ALTER TABLE tb_user_fund_holding ADD COLUMN IF NOT EXISTS shares numeric(18,6)`,
	`ALTER TABLE tb_user_fund_holding ADD COLUMN IF NOT EXISTS confirmed_nav numeric(18,6)`,
	`ALTER TABLE tb_user_fund_holding ADD COLUMN IF NOT EXISTS confirmed_nav_date date`,
}
