-- Existing FundLive database only. Back up first; do not re-import the empty-database schema.
-- Apply before deploying the opening-position backend. Safe to run again.
BEGIN;
ALTER TABLE tb_user_fund_holding
  ADD COLUMN IF NOT EXISTS entry_type varchar(16) NOT NULL DEFAULT 'buy';
INSERT INTO schema_migrations (id)
  VALUES ('20261103_holding_opening_positions') ON CONFLICT (id) DO NOTHING;
COMMIT;
