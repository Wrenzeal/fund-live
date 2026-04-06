package database

var adminUserMigrationStatements = []string{
	`ALTER TABLE tb_user ADD COLUMN IF NOT EXISTS is_admin boolean NOT NULL DEFAULT false`,
	`CREATE INDEX IF NOT EXISTS idx_tb_user_is_admin ON tb_user (is_admin)`,
}

var issueMigrationStatements = []string{
	`CREATE EXTENSION IF NOT EXISTS pg_trgm`,
	`CREATE TABLE IF NOT EXISTS issues (
		id varchar(40) PRIMARY KEY,
		title varchar(200) NOT NULL,
		body text NOT NULL,
		type varchar(32) NOT NULL,
		status varchar(32) NOT NULL,
		created_by_user_id varchar(40) NOT NULL,
		created_by_display_name varchar(120) NOT NULL,
		created_at timestamptz NOT NULL DEFAULT now(),
		updated_at timestamptz NOT NULL DEFAULT now()
	)`,
	`CREATE INDEX IF NOT EXISTS idx_issues_type ON issues (type)`,
	`CREATE INDEX IF NOT EXISTS idx_issues_status ON issues (status)`,
	`CREATE INDEX IF NOT EXISTS idx_issues_created_at ON issues (created_at DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_issues_created_by_user_id ON issues (created_by_user_id)`,
	`CREATE INDEX IF NOT EXISTS idx_issues_title_trgm ON issues USING gin (title gin_trgm_ops)`,
	`CREATE INDEX IF NOT EXISTS idx_issues_body_trgm ON issues USING gin (body gin_trgm_ops)`,
}

var announcementMigrationStatements = []string{
	`CREATE TABLE IF NOT EXISTS announcements (
		id varchar(40) PRIMARY KEY,
		title varchar(200) NOT NULL,
		summary varchar(500) NOT NULL,
		content text NOT NULL,
		source_type varchar(32) NOT NULL,
		source_ref varchar(128) NOT NULL DEFAULT '',
		published_at timestamptz NOT NULL,
		created_at timestamptz NOT NULL DEFAULT now(),
		updated_at timestamptz NOT NULL DEFAULT now()
	)`,
	`CREATE INDEX IF NOT EXISTS idx_announcements_published_at ON announcements (published_at DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_announcements_source_type ON announcements (source_type)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS uq_announcements_changelog_ref ON announcements (source_ref) WHERE source_type = 'changelog'`,
	`CREATE TABLE IF NOT EXISTS announcement_reads (
		id bigserial PRIMARY KEY,
		announcement_id varchar(40) NOT NULL,
		user_id varchar(40) NOT NULL,
		read_at timestamptz NOT NULL,
		created_at timestamptz NOT NULL DEFAULT now(),
		CONSTRAINT uq_announcement_reads_user_announcement UNIQUE (user_id, announcement_id)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_announcement_reads_user_id ON announcement_reads (user_id)`,
	`CREATE INDEX IF NOT EXISTS idx_announcement_reads_announcement_id ON announcement_reads (announcement_id)`,
}
