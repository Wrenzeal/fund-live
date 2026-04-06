package repository

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
)

type MemoryIssueRepository struct {
	mu     sync.RWMutex
	issues map[string]domain.Issue
}

func NewMemoryIssueRepository() *MemoryIssueRepository {
	return &MemoryIssueRepository{
		issues: make(map[string]domain.Issue),
	}
}

func (r *MemoryIssueRepository) ListPublicIssues(ctx context.Context, params domain.IssueSearchParams) ([]domain.Issue, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query := strings.ToLower(strings.TrimSpace(params.Query))
	result := make([]domain.Issue, 0, len(r.issues))
	for _, issue := range r.issues {
		if params.Type != "" && issue.Type != params.Type {
			continue
		}
		if params.Status != "" && issue.Status != params.Status {
			continue
		}
		if query != "" {
			title := strings.ToLower(issue.Title)
			body := strings.ToLower(issue.Body)
			if !strings.Contains(title, query) && !strings.Contains(body, query) {
				continue
			}
		}
		result = append(result, issue)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].ID > result[j].ID
		}
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	return result, nil
}

func (r *MemoryIssueRepository) GetIssueByID(ctx context.Context, issueID string) (*domain.Issue, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	issue, ok := r.issues[issueID]
	if !ok {
		return nil, nil
	}
	copyIssue := issue
	return &copyIssue, nil
}

func (r *MemoryIssueRepository) SaveIssue(ctx context.Context, issue *domain.Issue) error {
	if issue == nil {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	copyIssue := *issue
	if copyIssue.CreatedAt.IsZero() {
		copyIssue.CreatedAt = now
	}
	if copyIssue.UpdatedAt.IsZero() {
		copyIssue.UpdatedAt = now
	}
	r.issues[copyIssue.ID] = copyIssue
	return nil
}
