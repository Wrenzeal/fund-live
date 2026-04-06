package service

import (
	"testing"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
)

func TestParseAnnouncementsFromChangelog(t *testing.T) {
	raw := `
# Changelog

## [Unreleased]

## [2026.4.6] - 2026-04-06

### Added
- 新增公开 Issue 页面
- 新增公告历史页

### Changed
- 登录后支持未读公告弹窗

## [2026.4.5] - 2026-04-05

- 另一条更新
`

	items, err := parseAnnouncementsFromChangelog(raw, time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("parseAnnouncementsFromChangelog() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("parseAnnouncementsFromChangelog() len = %d, want 2", len(items))
	}
	if items[0].SourceType != domain.AnnouncementSourceChangelog {
		t.Fatalf("first SourceType = %q", items[0].SourceType)
	}
	if items[0].SourceRef != "2026.4.6" {
		t.Fatalf("first SourceRef = %q, want 2026.4.6", items[0].SourceRef)
	}
	if items[0].Title == "" || items[0].Summary == "" || items[0].Content == "" {
		t.Fatalf("first announcement should have title/summary/content: %#v", items[0])
	}
}

func TestDeriveAnnouncementSummary(t *testing.T) {
	summary := deriveAnnouncementSummary("- 新增公开 Issue 页面\n- 支持管理员改状态")
	if summary == "" {
		t.Fatal("deriveAnnouncementSummary() returned empty summary")
	}
}
