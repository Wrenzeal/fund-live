package database

var fundEstimateCapabilityMigrationStatements = []string{
	`CREATE TABLE IF NOT EXISTS fund_estimate_capabilities (
		fund_id varchar(10) PRIMARY KEY REFERENCES funds(id) ON DELETE CASCADE,
		capability_status varchar(20) NOT NULL DEFAULT 'unsupported',
		capability_type varchar(50) NOT NULL DEFAULT 'unknown',
		quote_source_mode varchar(20) NOT NULL DEFAULT 'domestic',
		target_code varchar(10),
		holdings_count bigint NOT NULL DEFAULT 0,
		total_hold_ratio numeric(10,4) NOT NULL DEFAULT 0,
		has_effective_holdings boolean NOT NULL DEFAULT false,
		has_valuation_profile boolean NOT NULL DEFAULT false,
		has_target_mapping boolean NOT NULL DEFAULT false,
		checked_at timestamptz NOT NULL DEFAULT now(),
		created_at timestamptz NOT NULL DEFAULT now(),
		updated_at timestamptz NOT NULL DEFAULT now()
	)`,
	`CREATE INDEX IF NOT EXISTS idx_fund_estimate_capabilities_status ON fund_estimate_capabilities (capability_status)`,
	`CREATE INDEX IF NOT EXISTS idx_fund_estimate_capabilities_type ON fund_estimate_capabilities (capability_type)`,
	`CREATE INDEX IF NOT EXISTS idx_fund_estimate_capabilities_quote_mode ON fund_estimate_capabilities (quote_source_mode)`,
	`CREATE INDEX IF NOT EXISTS idx_fund_estimate_capabilities_checked_at ON fund_estimate_capabilities (checked_at)`,
}
