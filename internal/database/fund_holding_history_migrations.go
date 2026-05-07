package database

var fundHoldingHistoryMigrationStatements = []string{
	`CREATE TABLE IF NOT EXISTS stock_holding_history (
		id bigserial PRIMARY KEY,
		fund_id varchar(10) NOT NULL,
		stock_code varchar(10) NOT NULL,
		stock_name varchar(50),
		exchange varchar(5),
		holding_ratio numeric(8,4),
		holding_shares numeric(18,2),
		market_value numeric(18,2),
		reporting_period varchar(10) NOT NULL,
		source varchar(50),
		created_at timestamptz DEFAULT now(),
		updated_at timestamptz DEFAULT now()
	)`,
	`CREATE INDEX IF NOT EXISTS idx_stock_holding_history_fund_period ON stock_holding_history (fund_id, reporting_period)`,
	`CREATE INDEX IF NOT EXISTS idx_stock_holding_history_stock_code ON stock_holding_history (stock_code)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS uq_stock_holding_history_fund_stock_period ON stock_holding_history (fund_id, stock_code, reporting_period)`,
}
