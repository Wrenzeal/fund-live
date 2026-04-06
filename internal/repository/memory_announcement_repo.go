package repository

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
)

type MemoryAnnouncementRepository struct {
	mu            sync.RWMutex
	announcements map[string]domain.Announcement
	reads         map[string]map[string]time.Time
}

func NewMemoryAnnouncementRepository() *MemoryAnnouncementRepository {
	return &MemoryAnnouncementRepository{
		announcements: make(map[string]domain.Announcement),
		reads:         make(map[string]map[string]time.Time),
	}
}

func (r *MemoryAnnouncementRepository) ListAnnouncements(ctx context.Context) ([]domain.Announcement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]domain.Announcement, 0, len(r.announcements))
	for _, announcement := range r.announcements {
		result = append(result, announcement)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].PublishedAt.Equal(result[j].PublishedAt) {
			return result[i].ID > result[j].ID
		}
		return result[i].PublishedAt.After(result[j].PublishedAt)
	})
	return result, nil
}

func (r *MemoryAnnouncementRepository) GetAnnouncementByID(ctx context.Context, announcementID string) (*domain.Announcement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	announcement, ok := r.announcements[announcementID]
	if !ok {
		return nil, nil
	}
	copyAnnouncement := announcement
	return &copyAnnouncement, nil
}

func (r *MemoryAnnouncementRepository) GetAnnouncementBySource(ctx context.Context, sourceType domain.AnnouncementSourceType, sourceRef string) (*domain.Announcement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sourceRef = strings.TrimSpace(sourceRef)
	for _, announcement := range r.announcements {
		if announcement.SourceType == sourceType && announcement.SourceRef == sourceRef {
			copyAnnouncement := announcement
			return &copyAnnouncement, nil
		}
	}
	return nil, nil
}

func (r *MemoryAnnouncementRepository) SaveAnnouncement(ctx context.Context, announcement *domain.Announcement) error {
	if announcement == nil {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	copyAnnouncement := *announcement
	if copyAnnouncement.CreatedAt.IsZero() {
		copyAnnouncement.CreatedAt = now
	}
	if copyAnnouncement.UpdatedAt.IsZero() {
		copyAnnouncement.UpdatedAt = now
	}
	r.announcements[copyAnnouncement.ID] = copyAnnouncement
	return nil
}

func (r *MemoryAnnouncementRepository) ListUnreadAnnouncements(ctx context.Context, userID string, limit int) ([]domain.Announcement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	readMap := r.reads[userID]
	result := make([]domain.Announcement, 0)
	for _, announcement := range r.announcements {
		if readMap != nil {
			if _, ok := readMap[announcement.ID]; ok {
				continue
			}
		}
		result = append(result, announcement)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].PublishedAt.Equal(result[j].PublishedAt) {
			return result[i].ID > result[j].ID
		}
		return result[i].PublishedAt.After(result[j].PublishedAt)
	})

	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (r *MemoryAnnouncementRepository) MarkAnnouncementRead(ctx context.Context, userID, announcementID string, readAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.reads[userID]; !ok {
		r.reads[userID] = make(map[string]time.Time)
	}
	r.reads[userID][announcementID] = readAt
	return nil
}
