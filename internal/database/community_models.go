package database

import "time"

type Issue struct {
	ID                   string    `gorm:"primaryKey;type:varchar(40)" json:"id"`
	Title                string    `gorm:"type:varchar(200);not null" json:"title"`
	Body                 string    `gorm:"type:text;not null" json:"body"`
	Type                 string    `gorm:"type:varchar(32);index;not null" json:"type"`
	Status               string    `gorm:"type:varchar(32);index;not null" json:"status"`
	CreatedByUserID      string    `gorm:"type:varchar(40);index;not null" json:"created_by_user_id"`
	CreatedByDisplayName string    `gorm:"type:varchar(120);not null" json:"created_by_display_name"`
	CreatedAt            time.Time `gorm:"autoCreateTime;index" json:"created_at"`
	UpdatedAt            time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Issue) TableName() string {
	return "issues"
}

type Announcement struct {
	ID          string    `gorm:"primaryKey;type:varchar(40)" json:"id"`
	Title       string    `gorm:"type:varchar(200);not null" json:"title"`
	Summary     string    `gorm:"type:varchar(500);not null" json:"summary"`
	Content     string    `gorm:"type:text;not null" json:"content"`
	SourceType  string    `gorm:"type:varchar(32);index;not null" json:"source_type"`
	SourceRef   string    `gorm:"type:varchar(128);index" json:"source_ref"`
	PublishedAt time.Time `gorm:"type:timestamptz;index;not null" json:"published_at"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Announcement) TableName() string {
	return "announcements"
}

type AnnouncementRead struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	AnnouncementID string    `gorm:"type:varchar(40);index:idx_announcement_reads_user_announcement,priority:2;uniqueIndex:idx_announcement_reads_user_announcement,priority:2;not null" json:"announcement_id"`
	UserID         string    `gorm:"type:varchar(40);index:idx_announcement_reads_user_announcement,priority:1;uniqueIndex:idx_announcement_reads_user_announcement,priority:1;index;not null" json:"user_id"`
	ReadAt         time.Time `gorm:"type:timestamptz;not null" json:"read_at"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (AnnouncementRead) TableName() string {
	return "announcement_reads"
}
