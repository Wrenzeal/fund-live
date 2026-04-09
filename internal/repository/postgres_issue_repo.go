package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PostgresIssueRepository struct {
	db *gorm.DB
}

func NewPostgresIssueRepository(db *gorm.DB) *PostgresIssueRepository {
	return &PostgresIssueRepository{db: db}
}

func (r *PostgresIssueRepository) ListPublicIssues(ctx context.Context, params domain.IssueSearchParams) ([]domain.Issue, error) {
	query := r.db.WithContext(ctx).Model(&database.Issue{})

	if params.Type != "" {
		query = query.Where(`type = ?`, string(params.Type))
	}
	if params.Status != "" {
		query = query.Where(`status = ?`, string(params.Status))
	}
	if keyword := strings.TrimSpace(params.Query); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where(`title ILIKE ? OR body ILIKE ?`, like, like)
	}

	var records []database.Issue
	if err := query.Order("created_at DESC").Limit(200).Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}

	result := make([]domain.Issue, 0, len(records))
	for _, record := range records {
		result = append(result, r.toDomainIssue(&record, false))
	}

	return result, nil
}

func (r *PostgresIssueRepository) GetIssueByID(ctx context.Context, issueID string) (*domain.Issue, error) {
	var record database.Issue
	if err := r.db.WithContext(ctx).First(&record, "id = ?", issueID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}

	return issueFromDBRecord(&record), nil
}

func (r *PostgresIssueRepository) SaveIssue(ctx context.Context, issue *domain.Issue) error {
	if issue == nil {
		return nil
	}

	now := time.Now()
	if issue.CreatedAt.IsZero() {
		issue.CreatedAt = now
	}
	if issue.UpdatedAt.IsZero() {
		issue.UpdatedAt = now
	}

	record := &database.Issue{
		ID:                         issue.ID,
		Title:                      issue.Title,
		Body:                       issue.Body,
		Type:                       string(issue.Type),
		Status:                     string(issue.Status),
		CreatedByUserID:            issue.CreatedByUserID,
		CreatedByDisplayName:       issue.CreatedByDisplayName,
		CreatedAt:                  issue.CreatedAt,
		UpdatedAt:                  issue.UpdatedAt,
		OfficialReplyBody:          "",
		OfficialReplyByUserID:      "",
		OfficialReplyByDisplayName: "",
	}
	if issue.OfficialReply != nil {
		record.OfficialReplyBody = issue.OfficialReply.Body
		record.OfficialReplyByUserID = issue.OfficialReply.RepliedByUserID
		record.OfficialReplyByDisplayName = issue.OfficialReply.RepliedByDisplayName
		record.OfficialReplyCreatedAt = &issue.OfficialReply.CreatedAt
		record.OfficialReplyUpdatedAt = &issue.OfficialReply.UpdatedAt
	}

	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"title",
			"body",
			"type",
			"status",
			"official_reply_body",
			"official_reply_by_user_id",
			"official_reply_by_display_name",
			"official_reply_created_at",
			"official_reply_updated_at",
			"created_by_user_id",
			"created_by_display_name",
			"updated_at",
		}),
	}).Create(record).Error; err != nil {
		return fmt.Errorf("failed to save issue: %w", err)
	}

	return nil
}

func issueFromDBRecord(record *database.Issue) *domain.Issue {
	if record == nil {
		return nil
	}

	return &domain.Issue{
		ID:                   record.ID,
		Title:                record.Title,
		Body:                 record.Body,
		Type:                 domain.IssueType(record.Type),
		Status:               domain.IssueStatus(record.Status),
		OfficialReply:        toIssueOfficialReply(record),
		CreatedByUserID:      record.CreatedByUserID,
		CreatedByDisplayName: record.CreatedByDisplayName,
		CreatedAt:            record.CreatedAt,
		UpdatedAt:            record.UpdatedAt,
	}
}

func (r *PostgresIssueRepository) toDomainIssue(record *database.Issue, includeReply bool) domain.Issue {
	issue := issueFromDBRecord(record)
	if issue == nil {
		return domain.Issue{}
	}
	if !includeReply {
		issue.OfficialReply = nil
	}
	return *issue
}

func toIssueOfficialReply(record *database.Issue) *domain.IssueOfficialReply {
	if record == nil {
		return nil
	}

	body := strings.TrimSpace(record.OfficialReplyBody)
	if body == "" {
		return nil
	}

	reply := &domain.IssueOfficialReply{
		Body:                 body,
		RepliedByUserID:      record.OfficialReplyByUserID,
		RepliedByDisplayName: record.OfficialReplyByDisplayName,
	}
	if record.OfficialReplyCreatedAt != nil {
		reply.CreatedAt = *record.OfficialReplyCreatedAt
	}
	if record.OfficialReplyUpdatedAt != nil {
		reply.UpdatedAt = *record.OfficialReplyUpdatedAt
	}
	return reply
}
