package domain

import (
	"context"
	"time"
)

type IssueType string

const (
	IssueTypeBug         IssueType = "bug"
	IssueTypeFeature     IssueType = "feature"
	IssueTypeImprovement IssueType = "improvement"
)

type IssueStatus string

const (
	IssueStatusPending   IssueStatus = "pending"
	IssueStatusAccepted  IssueStatus = "accepted"
	IssueStatusCompleted IssueStatus = "completed"
)

type Issue struct {
	ID                   string      `json:"id"`
	Title                string      `json:"title"`
	Body                 string      `json:"body"`
	Type                 IssueType   `json:"type"`
	Status               IssueStatus `json:"status"`
	CreatedByUserID      string      `json:"created_by_user_id"`
	CreatedByDisplayName string      `json:"created_by_display_name"`
	CreatedAt            time.Time   `json:"created_at"`
	UpdatedAt            time.Time   `json:"updated_at"`
}

type IssueSearchParams struct {
	Query  string
	Type   IssueType
	Status IssueStatus
}

type IssueCreateInput struct {
	Title string    `json:"title"`
	Body  string    `json:"body"`
	Type  IssueType `json:"type"`
}

type IssueStatusUpdateInput struct {
	Status IssueStatus `json:"status"`
}

type AnnouncementSourceType string

const (
	AnnouncementSourceManual    AnnouncementSourceType = "manual"
	AnnouncementSourceChangelog AnnouncementSourceType = "changelog"
)

type Announcement struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Summary     string                 `json:"summary"`
	Content     string                 `json:"content"`
	SourceType  AnnouncementSourceType `json:"source_type"`
	SourceRef   string                 `json:"source_ref"`
	PublishedAt time.Time              `json:"published_at"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Read        bool                   `json:"read,omitempty"`
}

type AnnouncementCreateInput struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Content string `json:"content"`
}

type IssueRepository interface {
	ListPublicIssues(ctx context.Context, params IssueSearchParams) ([]Issue, error)
	GetIssueByID(ctx context.Context, issueID string) (*Issue, error)
	SaveIssue(ctx context.Context, issue *Issue) error
}

type AnnouncementRepository interface {
	ListAnnouncements(ctx context.Context) ([]Announcement, error)
	GetAnnouncementByID(ctx context.Context, announcementID string) (*Announcement, error)
	GetAnnouncementBySource(ctx context.Context, sourceType AnnouncementSourceType, sourceRef string) (*Announcement, error)
	SaveAnnouncement(ctx context.Context, announcement *Announcement) error
	ListUnreadAnnouncements(ctx context.Context, userID string, limit int) ([]Announcement, error)
	MarkAnnouncementRead(ctx context.Context, userID, announcementID string, readAt time.Time) error
}

type IssueService interface {
	ListPublicIssues(ctx context.Context, params IssueSearchParams) ([]Issue, error)
	GetIssueByID(ctx context.Context, issueID string) (*Issue, error)
	CreateIssue(ctx context.Context, user *User, input IssueCreateInput) (*Issue, error)
	UpdateIssueStatus(ctx context.Context, issueID string, status IssueStatus) (*Issue, error)
}

type AnnouncementService interface {
	ListAnnouncements(ctx context.Context) ([]Announcement, error)
	GetAnnouncementByID(ctx context.Context, announcementID string) (*Announcement, error)
	ListUnreadAnnouncements(ctx context.Context, userID string, limit int) ([]Announcement, error)
	MarkAnnouncementRead(ctx context.Context, userID, announcementID string) error
	CreateAnnouncement(ctx context.Context, input AnnouncementCreateInput) (*Announcement, error)
	ImportAnnouncementsFromChangelog(ctx context.Context) (int, error)
}
