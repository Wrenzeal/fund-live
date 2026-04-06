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

type PostgresAnnouncementRepository struct {
	db *gorm.DB
}

func NewPostgresAnnouncementRepository(db *gorm.DB) *PostgresAnnouncementRepository {
	return &PostgresAnnouncementRepository{db: db}
}

func (r *PostgresAnnouncementRepository) ListAnnouncements(ctx context.Context) ([]domain.Announcement, error) {
	var records []database.Announcement
	if err := r.db.WithContext(ctx).Order("published_at DESC").Limit(100).Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to list announcements: %w", err)
	}

	result := make([]domain.Announcement, 0, len(records))
	for _, record := range records {
		result = append(result, toDomainAnnouncement(record))
	}
	return result, nil
}

func (r *PostgresAnnouncementRepository) GetAnnouncementByID(ctx context.Context, announcementID string) (*domain.Announcement, error) {
	var record database.Announcement
	if err := r.db.WithContext(ctx).First(&record, "id = ?", announcementID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get announcement: %w", err)
	}

	announcement := toDomainAnnouncement(record)
	return &announcement, nil
}

func (r *PostgresAnnouncementRepository) GetAnnouncementBySource(ctx context.Context, sourceType domain.AnnouncementSourceType, sourceRef string) (*domain.Announcement, error) {
	if strings.TrimSpace(sourceRef) == "" {
		return nil, nil
	}

	var record database.Announcement
	if err := r.db.WithContext(ctx).
		Where("source_type = ? AND source_ref = ?", string(sourceType), sourceRef).
		First(&record).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get announcement by source: %w", err)
	}

	announcement := toDomainAnnouncement(record)
	return &announcement, nil
}

func (r *PostgresAnnouncementRepository) SaveAnnouncement(ctx context.Context, announcement *domain.Announcement) error {
	if announcement == nil {
		return nil
	}

	record := &database.Announcement{
		ID:          announcement.ID,
		Title:       announcement.Title,
		Summary:     announcement.Summary,
		Content:     announcement.Content,
		SourceType:  string(announcement.SourceType),
		SourceRef:   announcement.SourceRef,
		PublishedAt: announcement.PublishedAt,
		CreatedAt:   announcement.CreatedAt,
		UpdatedAt:   announcement.UpdatedAt,
	}

	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"title",
			"summary",
			"content",
			"source_type",
			"source_ref",
			"published_at",
			"updated_at",
		}),
	}).Create(record).Error; err != nil {
		return fmt.Errorf("failed to save announcement: %w", err)
	}

	return nil
}

func (r *PostgresAnnouncementRepository) ListUnreadAnnouncements(ctx context.Context, userID string, limit int) ([]domain.Announcement, error) {
	query := r.db.WithContext(ctx).
		Table("announcements").
		Select("announcements.*").
		Joins("LEFT JOIN announcement_reads ON announcement_reads.announcement_id = announcements.id AND announcement_reads.user_id = ?", userID).
		Where("announcement_reads.id IS NULL").
		Order("announcements.published_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}

	var records []database.Announcement
	if err := query.Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to list unread announcements: %w", err)
	}

	result := make([]domain.Announcement, 0, len(records))
	for _, record := range records {
		result = append(result, toDomainAnnouncement(record))
	}
	return result, nil
}

func (r *PostgresAnnouncementRepository) MarkAnnouncementRead(ctx context.Context, userID, announcementID string, readAt time.Time) error {
	record := &database.AnnouncementRead{
		UserID:         userID,
		AnnouncementID: announcementID,
		ReadAt:         readAt,
		CreatedAt:      readAt,
	}

	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "announcement_id"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"read_at"}),
	}).Create(record).Error; err != nil {
		return fmt.Errorf("failed to mark announcement read: %w", err)
	}

	return nil
}

func toDomainAnnouncement(record database.Announcement) domain.Announcement {
	return domain.Announcement{
		ID:          record.ID,
		Title:       record.Title,
		Summary:     record.Summary,
		Content:     record.Content,
		SourceType:  domain.AnnouncementSourceType(record.SourceType),
		SourceRef:   record.SourceRef,
		PublishedAt: record.PublishedAt,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}
}
