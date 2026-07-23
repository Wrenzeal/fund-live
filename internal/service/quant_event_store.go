package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const defaultEventTimelineLimit = 50

// QuantEventStore keeps source events versioned and addressable by the time at
// which FundLive first knew each version.
type QuantEventStore struct {
	db *gorm.DB
}

func NewQuantEventStore(db *gorm.DB) *QuantEventStore {
	if db == nil {
		return nil
	}
	return &QuantEventStore{db: db}
}

func (s *QuantEventStore) SaveImpacts(ctx context.Context, fundID string, impacts []domain.FundAnalysisEventImpact, observedAt time.Time) error {
	if s == nil || s.db == nil || strings.TrimSpace(fundID) == "" || len(impacts) == 0 {
		return nil
	}
	if observedAt.IsZero() {
		observedAt = time.Now()
	}
	impacts = normalizeCurrentEventMetadata(impacts, observedAt)

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, impact := range impacts {
			if err := saveQuantEventImpact(tx, strings.TrimSpace(fundID), impact, observedAt); err != nil {
				return err
			}
		}
		return nil
	})
}

func saveQuantEventImpact(tx *gorm.DB, fundID string, impact domain.FundAnalysisEventImpact, observedAt time.Time) error {
	payload, contentHash, err := quantEventContent(impact)
	if err != nil {
		return err
	}

	var event database.QuantEvent
	err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&event, "id = ?", impact.EventID).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if err == gorm.ErrRecordNotFound {
		event = database.QuantEvent{
			ID:              impact.EventID,
			EventType:       impact.EventType,
			TargetScope:     impact.TargetScope,
			CurrentVersion:  1,
			FirstKnownAt:    observedAt,
			FirstIngestedAt: observedAt,
		}
		if err := tx.Create(&event).Error; err != nil {
			return err
		}
	} else {
		var current database.QuantEventVersion
		if err := tx.Where("event_id = ? AND version = ?", event.ID, event.CurrentVersion).First(&current).Error; err != nil {
			return err
		}
		if current.ContentHash != contentHash {
			event.CurrentVersion++
			event.EventType = impact.EventType
			event.TargetScope = impact.TargetScope
			if err := tx.Save(&event).Error; err != nil {
				return err
			}
		} else {
			return upsertQuantEventMapping(tx, event.ID, fundID, impact)
		}
	}

	knownAt := observedAt
	knownAtBasis := strings.TrimSpace(impact.KnownAtBasis)
	if impact.KnownAt != nil && !impact.KnownAt.IsZero() {
		knownAt = *impact.KnownAt
	}
	if knownAtBasis == "" {
		knownAtBasis = "first_seen"
	}
	ingestedAt := observedAt
	if impact.IngestedAt != nil && !impact.IngestedAt.IsZero() {
		ingestedAt = *impact.IngestedAt
	}

	version := database.QuantEventVersion{
		EventID:           event.ID,
		Version:           event.CurrentVersion,
		ContentHash:       contentHash,
		Status:            impact.EventStatus,
		Title:             impact.Title,
		Summary:           impact.Summary,
		Impact:            impact.Impact,
		Strength:          impact.Strength,
		Horizon:           impact.Horizon,
		ExpectedAt:        impact.ExpectedAt,
		AnnouncedAt:       impact.AnnouncedAt,
		EffectiveAt:       impact.EffectiveAt,
		ExpiresAt:         impact.ExpiresAt,
		KnownAt:           knownAt,
		IngestedAt:        ingestedAt,
		KnownAtBasis:      knownAtBasis,
		SourceTier:        impact.SourceTier,
		SourceName:        impact.SourceName,
		SourceURL:         impact.SourceURL,
		SourcePublishedAt: parseAnalysisEventDate(impact.SourcePublishedAt),
		SourceConfidence:  impact.SourceConfidence,
		RawPayload:        payload,
	}
	if err := tx.Create(&version).Error; err != nil {
		return err
	}
	return upsertQuantEventMapping(tx, event.ID, fundID, impact)
}

func upsertQuantEventMapping(tx *gorm.DB, eventID, fundID string, impact domain.FundAnalysisEventImpact) error {
	weight := decimal.Zero
	if impact.WeightHint != nil {
		weight = *impact.WeightHint
	}
	mapping := database.QuantEventAsset{
		EventID:      eventID,
		AssetType:    "fund",
		AssetCode:    fundID,
		MappingBasis: strings.TrimSpace(impact.MappingBasis),
		WeightHint:   weight,
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "event_id"}, {Name: "asset_type"}, {Name: "asset_code"}},
		DoUpdates: clause.AssignmentColumns([]string{"mapping_basis", "weight_hint"}),
	}).Create(&mapping).Error
}

func quantEventContent(impact domain.FundAnalysisEventImpact) (json.RawMessage, string, error) {
	stable := impact
	stable.KnownAt = nil
	stable.IngestedAt = nil
	stable.Version = 0
	payload, err := json.Marshal(stable)
	if err != nil {
		return nil, "", fmt.Errorf("marshal quant event: %w", err)
	}
	sum := sha256.Sum256(payload)
	return payload, fmt.Sprintf("%x", sum[:]), nil
}

func (s *QuantEventStore) ListAsOf(ctx context.Context, fundID string, asOf time.Time, status string, limit int) ([]domain.FundAnalysisEventImpact, error) {
	if s == nil || s.db == nil || strings.TrimSpace(fundID) == "" {
		return nil, nil
	}
	if asOf.IsZero() {
		asOf = time.Now()
	}
	if limit <= 0 || limit > 200 {
		limit = defaultEventTimelineLimit
	}

	query := `
		SELECT DISTINCT ON (v.event_id) v.*
		FROM quant_event_versions v
		JOIN quant_event_assets a ON a.event_id = v.event_id
		WHERE a.asset_type = 'fund' AND a.asset_code = ? AND v.known_at <= ?`
	args := []interface{}{strings.TrimSpace(fundID), asOf}
	query += ` ORDER BY v.event_id, v.known_at DESC, v.version DESC`
	outerFilter := ""
	if status = strings.TrimSpace(status); status != "" {
		outerFilter = ` WHERE latest.status = ?`
		args = append(args, status)
	}

	var versions []database.QuantEventVersion
	if err := s.db.WithContext(ctx).Raw(`SELECT * FROM (`+query+`) latest`+outerFilter+` ORDER BY known_at DESC, id DESC LIMIT ?`, append(args, limit)...).Scan(&versions).Error; err != nil {
		return nil, err
	}
	result := make([]domain.FundAnalysisEventImpact, 0, len(versions))
	for _, version := range versions {
		var impact domain.FundAnalysisEventImpact
		if err := json.Unmarshal(version.RawPayload, &impact); err != nil {
			continue
		}
		impact.EventID = version.EventID
		impact.Version = version.Version
		impact.EventStatus = version.Status
		impact.ExpectedAt = version.ExpectedAt
		impact.AnnouncedAt = version.AnnouncedAt
		impact.EffectiveAt = version.EffectiveAt
		impact.ExpiresAt = version.ExpiresAt
		knownAt := version.KnownAt
		ingestedAt := version.IngestedAt
		impact.KnownAt = &knownAt
		impact.IngestedAt = &ingestedAt
		impact.KnownAtBasis = version.KnownAtBasis
		impact.SourceTier = version.SourceTier
		result = append(result, impact)
	}
	return result, nil
}
