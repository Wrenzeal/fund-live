package service

import (
	"context"
	"testing"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/repository"
)

func TestIssueServiceCreateAndUpdate(t *testing.T) {
	repo := repository.NewMemoryIssueRepository()
	svc := NewIssueService(repo)

	user := &domain.User{
		ID:          "usr_test",
		Email:       "admin@example.com",
		DisplayName: "Admin",
		IsAdmin:     true,
	}

	created, err := svc.CreateIssue(context.Background(), user, domain.IssueCreateInput{
		Title: "首页在 Safari 下布局错位",
		Body:  "打开首页后，顶部卡片会发生横向溢出。",
		Type:  domain.IssueTypeBug,
	})
	if err != nil {
		t.Fatalf("CreateIssue() error = %v", err)
	}
	if created.Status != domain.IssueStatusPending {
		t.Fatalf("CreateIssue() status = %q, want %q", created.Status, domain.IssueStatusPending)
	}
	if created.CreatedByDisplayName != "Admin" {
		t.Fatalf("CreateIssue() display name = %q", created.CreatedByDisplayName)
	}

	updated, err := svc.UpdateIssueStatus(context.Background(), created.ID, domain.IssueStatusAccepted)
	if err != nil {
		t.Fatalf("UpdateIssueStatus() error = %v", err)
	}
	if updated.Status != domain.IssueStatusAccepted {
		t.Fatalf("UpdateIssueStatus() status = %q, want %q", updated.Status, domain.IssueStatusAccepted)
	}
}

func TestIssueServiceRejectsInvalidType(t *testing.T) {
	repo := repository.NewMemoryIssueRepository()
	svc := NewIssueService(repo)

	_, err := svc.CreateIssue(context.Background(), &domain.User{
		ID:          "usr_test",
		Email:       "user@example.com",
		DisplayName: "User",
	}, domain.IssueCreateInput{
		Title: "标题",
		Body:  "详情",
		Type:  domain.IssueType("unknown"),
	})
	if err == nil {
		t.Fatal("CreateIssue() error = nil, want invalid type error")
	}
}
