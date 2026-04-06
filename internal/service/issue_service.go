package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
)

var (
	ErrIssueNotFound       = errors.New("issue not found")
	ErrIssueInvalidType    = errors.New("invalid issue type")
	ErrIssueInvalidStatus  = errors.New("invalid issue status")
	ErrIssueInvalidContent = errors.New("issue title and body are required")
)

type IssueServiceImpl struct {
	repo domain.IssueRepository
	now  func() time.Time
}

func NewIssueService(repo domain.IssueRepository) *IssueServiceImpl {
	return &IssueServiceImpl{
		repo: repo,
		now:  time.Now,
	}
}

func (s *IssueServiceImpl) ListPublicIssues(ctx context.Context, params domain.IssueSearchParams) ([]domain.Issue, error) {
	params.Query = strings.TrimSpace(params.Query)
	params.Type = normalizeIssueType(params.Type)
	if raw := strings.TrimSpace(string(params.Status)); raw != "" {
		params.Status = normalizeIssueStatus(domain.IssueStatus(raw))
	}
	return s.repo.ListPublicIssues(ctx, params)
}

func (s *IssueServiceImpl) GetIssueByID(ctx context.Context, issueID string) (*domain.Issue, error) {
	issueID = strings.TrimSpace(issueID)
	if issueID == "" {
		return nil, ErrIssueNotFound
	}

	issue, err := s.repo.GetIssueByID(ctx, issueID)
	if err != nil {
		return nil, err
	}
	if issue == nil {
		return nil, ErrIssueNotFound
	}
	return issue, nil
}

func (s *IssueServiceImpl) CreateIssue(ctx context.Context, user *domain.User, input domain.IssueCreateInput) (*domain.Issue, error) {
	if user == nil {
		return nil, ErrInvalidSession
	}

	issueType := normalizeIssueType(input.Type)
	if issueType == "" {
		return nil, ErrIssueInvalidType
	}

	title := strings.TrimSpace(input.Title)
	body := strings.TrimSpace(input.Body)
	if title == "" || body == "" {
		return nil, ErrIssueInvalidContent
	}

	now := s.now()
	issue := &domain.Issue{
		ID:                   generateID("iss"),
		Title:                title,
		Body:                 body,
		Type:                 issueType,
		Status:               domain.IssueStatusPending,
		CreatedByUserID:      user.ID,
		CreatedByDisplayName: strings.TrimSpace(firstNonEmpty(user.DisplayName, user.Email)),
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if err := s.repo.SaveIssue(ctx, issue); err != nil {
		return nil, err
	}
	return issue, nil
}

func (s *IssueServiceImpl) UpdateIssueStatus(ctx context.Context, issueID string, status domain.IssueStatus) (*domain.Issue, error) {
	normalizedStatus := normalizeIssueStatus(status)
	if normalizedStatus == "" {
		return nil, ErrIssueInvalidStatus
	}

	issue, err := s.repo.GetIssueByID(ctx, strings.TrimSpace(issueID))
	if err != nil {
		return nil, err
	}
	if issue == nil {
		return nil, ErrIssueNotFound
	}

	issue.Status = normalizedStatus
	issue.UpdatedAt = s.now()
	if err := s.repo.SaveIssue(ctx, issue); err != nil {
		return nil, err
	}
	return issue, nil
}

func normalizeIssueType(issueType domain.IssueType) domain.IssueType {
	switch domain.IssueType(strings.TrimSpace(string(issueType))) {
	case domain.IssueTypeBug:
		return domain.IssueTypeBug
	case domain.IssueTypeFeature:
		return domain.IssueTypeFeature
	case domain.IssueTypeImprovement:
		return domain.IssueTypeImprovement
	default:
		return ""
	}
}

func normalizeIssueStatus(status domain.IssueStatus) domain.IssueStatus {
	switch domain.IssueStatus(strings.TrimSpace(string(status))) {
	case domain.IssueStatusPending:
		return domain.IssueStatusPending
	case domain.IssueStatusAccepted:
		return domain.IssueStatusAccepted
	case domain.IssueStatusCompleted:
		return domain.IssueStatusCompleted
	default:
		return ""
	}
}
