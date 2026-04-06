package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
)

var (
	ErrAnnouncementNotFound       = errors.New("announcement not found")
	ErrAnnouncementInvalidContent = errors.New("announcement title and content are required")
)

var changelogVersionPattern = regexp.MustCompile(`^## \[(.+?)\](?: - (\d{4}-\d{2}-\d{2}))?\s*$`)

type AnnouncementServiceImpl struct {
	repo domain.AnnouncementRepository
	now  func() time.Time
}

func NewAnnouncementService(repo domain.AnnouncementRepository) *AnnouncementServiceImpl {
	return &AnnouncementServiceImpl{
		repo: repo,
		now:  time.Now,
	}
}

func (s *AnnouncementServiceImpl) ListAnnouncements(ctx context.Context) ([]domain.Announcement, error) {
	return s.repo.ListAnnouncements(ctx)
}

func (s *AnnouncementServiceImpl) GetAnnouncementByID(ctx context.Context, announcementID string) (*domain.Announcement, error) {
	announcementID = strings.TrimSpace(announcementID)
	if announcementID == "" {
		return nil, ErrAnnouncementNotFound
	}

	announcement, err := s.repo.GetAnnouncementByID(ctx, announcementID)
	if err != nil {
		return nil, err
	}
	if announcement == nil {
		return nil, ErrAnnouncementNotFound
	}
	return announcement, nil
}

func (s *AnnouncementServiceImpl) ListUnreadAnnouncements(ctx context.Context, userID string, limit int) ([]domain.Announcement, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return []domain.Announcement{}, nil
	}

	items, err := s.repo.ListUnreadAnnouncements(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].Read = false
	}
	return items, nil
}

func (s *AnnouncementServiceImpl) MarkAnnouncementRead(ctx context.Context, userID, announcementID string) error {
	userID = strings.TrimSpace(userID)
	announcementID = strings.TrimSpace(announcementID)
	if userID == "" || announcementID == "" {
		return ErrAnnouncementNotFound
	}

	announcement, err := s.repo.GetAnnouncementByID(ctx, announcementID)
	if err != nil {
		return err
	}
	if announcement == nil {
		return ErrAnnouncementNotFound
	}

	return s.repo.MarkAnnouncementRead(ctx, userID, announcementID, s.now())
}

func (s *AnnouncementServiceImpl) CreateAnnouncement(ctx context.Context, input domain.AnnouncementCreateInput) (*domain.Announcement, error) {
	title := strings.TrimSpace(input.Title)
	content := strings.TrimSpace(input.Content)
	if title == "" || content == "" {
		return nil, ErrAnnouncementInvalidContent
	}

	now := s.now()
	summary := strings.TrimSpace(input.Summary)
	if summary == "" {
		summary = deriveAnnouncementSummary(content)
	}

	announcement := &domain.Announcement{
		ID:          generateID("ann"),
		Title:       title,
		Summary:     summary,
		Content:     content,
		SourceType:  domain.AnnouncementSourceManual,
		SourceRef:   "",
		PublishedAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.SaveAnnouncement(ctx, announcement); err != nil {
		return nil, err
	}
	return announcement, nil
}

func (s *AnnouncementServiceImpl) ImportAnnouncementsFromChangelog(ctx context.Context) (int, error) {
	changelogPath, err := resolveChangelogPath()
	if err != nil {
		return 0, err
	}

	raw, err := os.ReadFile(changelogPath)
	if err != nil {
		return 0, fmt.Errorf("read changelog: %w", err)
	}

	items, err := parseAnnouncementsFromChangelog(string(raw), s.now())
	if err != nil {
		return 0, err
	}

	imported := 0
	for _, item := range items {
		existing, err := s.repo.GetAnnouncementBySource(ctx, domain.AnnouncementSourceChangelog, item.SourceRef)
		if err != nil {
			return imported, err
		}

		now := s.now()
		if existing != nil {
			item.ID = existing.ID
			item.CreatedAt = existing.CreatedAt
			item.UpdatedAt = now
		} else {
			item.ID = generateID("ann")
			item.CreatedAt = now
			item.UpdatedAt = now
		}

		if err := s.repo.SaveAnnouncement(ctx, &item); err != nil {
			return imported, err
		}
		imported++
	}

	return imported, nil
}

func parseAnnouncementsFromChangelog(raw string, now time.Time) ([]domain.Announcement, error) {
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")

	currentVersion := ""
	currentDate := ""
	currentContent := make([]string, 0)
	announcements := make([]domain.Announcement, 0)

	flush := func() error {
		version := strings.TrimSpace(currentVersion)
		if version == "" || strings.EqualFold(version, "Unreleased") {
			currentContent = currentContent[:0]
			return nil
		}

		content := strings.TrimSpace(strings.Join(currentContent, "\n"))
		currentContent = currentContent[:0]
		if content == "" {
			return nil
		}

		publishedAt := now
		if currentDate != "" {
			parsed, err := time.ParseInLocation("2006-01-02", currentDate, tradingLocation())
			if err == nil {
				publishedAt = parsed
			}
		}

		announcements = append(announcements, domain.Announcement{
			Title:       fmt.Sprintf("FundLive 更新 %s", version),
			Summary:     deriveAnnouncementSummary(content),
			Content:     content,
			SourceType:  domain.AnnouncementSourceChangelog,
			SourceRef:   version,
			PublishedAt: publishedAt,
		})
		return nil
	}

	for _, line := range lines {
		matches := changelogVersionPattern.FindStringSubmatch(line)
		if len(matches) > 0 {
			if err := flush(); err != nil {
				return nil, err
			}
			currentVersion = matches[1]
			if len(matches) > 2 {
				currentDate = matches[2]
			} else {
				currentDate = ""
			}
			continue
		}

		if currentVersion != "" {
			currentContent = append(currentContent, line)
		}
	}

	if err := flush(); err != nil {
		return nil, err
	}

	return announcements, nil
}

func deriveAnnouncementSummary(content string) string {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		trimmed = strings.TrimPrefix(trimmed, "- ")
		trimmed = strings.TrimPrefix(trimmed, "* ")
		trimmed = strings.TrimSpace(trimmed)
		trimmed = strings.Trim(trimmed, "*")
		if trimmed != "" {
			return limitText(trimmed, 120)
		}
	}
	return "FundLive 发布了新的更新公告。"
}

func limitText(raw string, max int) string {
	if max <= 0 {
		return raw
	}

	runes := []rune(raw)
	if len(runes) <= max {
		return raw
	}
	return string(runes[:max]) + "…"
}

func resolveChangelogPath() (string, error) {
	candidates := []string{
		"CHANGELOG.md",
		filepath.Join("..", "CHANGELOG.md"),
		filepath.Join("..", "..", "CHANGELOG.md"),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("CHANGELOG.md not found")
}
