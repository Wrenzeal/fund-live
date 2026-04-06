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
		result = append(result, domain.Issue{
			ID:                   record.ID,
			Title:                record.Title,
			Body:                 record.Body,
			Type:                 domain.IssueType(record.Type),
			Status:               domain.IssueStatus(record.Status),
			CreatedByUserID:      record.CreatedByUserID,
			CreatedByDisplayName: record.CreatedByDisplayName,
			CreatedAt:            record.CreatedAt,
			UpdatedAt:            record.UpdatedAt,
		})
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

	return &domain.Issue{
		ID:                   record.ID,
		Title:                record.Title,
		Body:                 record.Body,
		Type:                 domain.IssueType(record.Type),
		Status:               domain.IssueStatus(record.Status),
		CreatedByUserID:      record.CreatedByUserID,
		CreatedByDisplayName: record.CreatedByDisplayName,
		CreatedAt:            record.CreatedAt,
		UpdatedAt:            record.UpdatedAt,
	}, nil
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
		ID:                   issue.ID,
		Title:                issue.Title,
		Body:                 issue.Body,
		Type:                 string(issue.Type),
		Status:               string(issue.Status),
		CreatedByUserID:      issue.CreatedByUserID,
		CreatedByDisplayName: issue.CreatedByDisplayName,
		CreatedAt:            issue.CreatedAt,
		UpdatedAt:            issue.UpdatedAt,
	}

	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"title",
			"body",
			"type",
			"status",
			"created_by_user_id",
			"created_by_display_name",
			"updated_at",
		}),
	}).Create(record).Error; err != nil {
		return fmt.Errorf("failed to save issue: %w", err)
	}

	return nil
}
