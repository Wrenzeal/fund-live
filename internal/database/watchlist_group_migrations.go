package database

var watchlistGroupOrderingMigrationStatements = []string{
	`ALTER TABLE tb_user_watchlist_group ADD COLUMN IF NOT EXISTS sort_order integer`,
	`WITH ranked AS (
		SELECT id, ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY created_at DESC, id ASC) AS rn
		FROM tb_user_watchlist_group
		WHERE sort_order IS NULL
	)
	UPDATE tb_user_watchlist_group AS g
	SET sort_order = ranked.rn
	FROM ranked
	WHERE g.id = ranked.id`,
	`ALTER TABLE tb_user_watchlist_group ALTER COLUMN sort_order SET DEFAULT 0`,
	`UPDATE tb_user_watchlist_group SET sort_order = 0 WHERE sort_order IS NULL`,
	`ALTER TABLE tb_user_watchlist_group ALTER COLUMN sort_order SET NOT NULL`,
	`CREATE INDEX IF NOT EXISTS idx_tb_user_watchlist_group_user_sort_order ON tb_user_watchlist_group (user_id, sort_order)`,
}
