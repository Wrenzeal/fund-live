package database

var fundAnalysisSnapshotMigrationStatements = []string{
	`CREATE TABLE IF NOT EXISTS fund_analysis_snapshots (
		fund_id varchar(10) PRIMARY KEY,
		analysis_type varchar(32) NOT NULL,
		analysis_basis varchar(64) NOT NULL,
		total_score numeric(8,4) NOT NULL DEFAULT 0,
		confidence varchar(16) NOT NULL DEFAULT '',
		risk_level varchar(16) NOT NULL DEFAULT '',
		increase_percent numeric(8,4) NOT NULL DEFAULT 0,
		hold_percent numeric(8,4) NOT NULL DEFAULT 0,
		decrease_percent numeric(8,4) NOT NULL DEFAULT 0,
		latest_holding_period varchar(16),
		summary text,
		generated_at timestamptz NOT NULL,
		analysis_json jsonb NOT NULL,
		created_at timestamptz DEFAULT now(),
		updated_at timestamptz DEFAULT now()
	)`,
	`CREATE INDEX IF NOT EXISTS idx_fund_analysis_snapshots_generated_at ON fund_analysis_snapshots (generated_at DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_fund_analysis_snapshots_risk_score ON fund_analysis_snapshots (risk_level, total_score)`,
	`CREATE INDEX IF NOT EXISTS idx_fund_analysis_snapshots_increase_score ON fund_analysis_snapshots (increase_percent DESC, total_score DESC)`,
}
