package database

var userHoldingConfirmationMigrationStatements = []string{
	`ALTER TABLE tb_user_fund_holding ADD COLUMN IF NOT EXISTS shares numeric(18,6)`,
	`ALTER TABLE tb_user_fund_holding ADD COLUMN IF NOT EXISTS confirmed_nav numeric(18,6)`,
	`ALTER TABLE tb_user_fund_holding ADD COLUMN IF NOT EXISTS confirmed_nav_date date`,
}

var userHoldingManualConfirmationMigrationStatements = []string{
	`ALTER TABLE tb_user_fund_holding ADD COLUMN IF NOT EXISTS manual_confirmation boolean`,
	`UPDATE tb_user_fund_holding SET manual_confirmation = false WHERE manual_confirmation IS NULL`,
	`ALTER TABLE tb_user_fund_holding ALTER COLUMN manual_confirmation SET DEFAULT false`,
	`ALTER TABLE tb_user_fund_holding ALTER COLUMN manual_confirmation SET NOT NULL`,
}

var userHoldingTransactionMigrationStatements = []string{
	`CREATE TABLE IF NOT EXISTS tb_user_fund_holding_transaction (
		id varchar(40) PRIMARY KEY,
		user_id varchar(40) NOT NULL,
		holding_id varchar(40),
		fund_id varchar(10) NOT NULL,
		type varchar(24) NOT NULL,
		amount numeric(18,2) NOT NULL,
		shares numeric(18,6),
		confirmed_nav numeric(18,6),
		confirmed_nav_date date,
		manual_confirmation boolean NOT NULL DEFAULT false,
		trade_at timestamptz,
		as_of_date date,
		note text,
		metadata text NOT NULL DEFAULT '{}',
		created_at timestamptz NOT NULL DEFAULT now()
	)`,
	`CREATE INDEX IF NOT EXISTS idx_user_fund_holding_tx_user_created ON tb_user_fund_holding_transaction (user_id, created_at DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_tb_user_fund_holding_tx_holding_id ON tb_user_fund_holding_transaction (holding_id)`,
	`CREATE INDEX IF NOT EXISTS idx_tb_user_fund_holding_tx_fund_id ON tb_user_fund_holding_transaction (fund_id)`,
	`CREATE INDEX IF NOT EXISTS idx_tb_user_fund_holding_tx_type ON tb_user_fund_holding_transaction (type)`,
	`CREATE INDEX IF NOT EXISTS idx_tb_user_fund_holding_tx_trade_at ON tb_user_fund_holding_transaction (trade_at)`,
}

var userHoldingTransactionVoidMigrationStatements = []string{
	`ALTER TABLE tb_user_fund_holding_transaction ADD COLUMN IF NOT EXISTS voided boolean`,
	`UPDATE tb_user_fund_holding_transaction SET voided = false WHERE voided IS NULL`,
	`ALTER TABLE tb_user_fund_holding_transaction ALTER COLUMN voided SET DEFAULT false`,
	`ALTER TABLE tb_user_fund_holding_transaction ALTER COLUMN voided SET NOT NULL`,
	`ALTER TABLE tb_user_fund_holding_transaction ADD COLUMN IF NOT EXISTS voided_at timestamptz`,
	`ALTER TABLE tb_user_fund_holding_transaction ADD COLUMN IF NOT EXISTS void_reason text`,
	`CREATE INDEX IF NOT EXISTS idx_tb_user_fund_holding_tx_voided ON tb_user_fund_holding_transaction (voided)`,
}

var userHoldingSourceMigrationStatements = []string{
	`ALTER TABLE tb_user_fund_holding ADD COLUMN IF NOT EXISTS source_platform varchar(32)`,
	`ALTER TABLE tb_user_fund_holding ADD COLUMN IF NOT EXISTS source_label varchar(64)`,
	`ALTER TABLE tb_user_fund_holding_transaction ADD COLUMN IF NOT EXISTS source_platform varchar(32)`,
	`ALTER TABLE tb_user_fund_holding_transaction ADD COLUMN IF NOT EXISTS source_label varchar(64)`,
	`CREATE INDEX IF NOT EXISTS idx_tb_user_fund_holding_source_platform ON tb_user_fund_holding (source_platform)`,
	`CREATE INDEX IF NOT EXISTS idx_tb_user_fund_holding_tx_source_platform ON tb_user_fund_holding_transaction (source_platform)`,
}
