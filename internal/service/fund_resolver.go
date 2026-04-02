// Package service contains the core business logic implementations.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/domain"
	"gorm.io/gorm"
)

const defaultFailedMappingRetryCooldown = 12 * time.Hour

type fundMappingStore interface {
	GetByFeederCode(ctx context.Context, feederCode string) (*database.FundMapping, error)
	Save(ctx context.Context, mapping *database.FundMapping) error
}

type gormFundMappingStore struct {
	db *gorm.DB
}

func (s *gormFundMappingStore) GetByFeederCode(ctx context.Context, feederCode string) (*database.FundMapping, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}

	var mapping database.FundMapping
	result := s.db.WithContext(ctx).
		Where("feeder_code = ?", feederCode).
		First(&mapping)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}

	return &mapping, nil
}

func (s *gormFundMappingStore) Save(ctx context.Context, mapping *database.FundMapping) error {
	if s == nil || s.db == nil || mapping == nil {
		return nil
	}

	now := time.Now()
	record := database.FundMapping{FeederCode: mapping.FeederCode}
	return s.db.WithContext(ctx).
		Where("feeder_code = ?", mapping.FeederCode).
		Assign(map[string]interface{}{
			"feeder_name":   mapping.FeederName,
			"target_code":   mapping.TargetCode,
			"target_name":   mapping.TargetName,
			"is_resolved":   mapping.IsResolved,
			"resolved_at":   mapping.ResolvedAt,
			"resolve_error": mapping.ResolveError,
			"updated_at":    now,
		}).
		FirstOrCreate(&record).Error
}

// FundResolver handles resolution of feeder funds to their target ETFs.
// It keeps a persistent mapping and uses deterministic Eastmoney search instead
// of AI so runtime behavior stays predictable.
type FundResolver struct {
	mappingStore  fundMappingStore
	fundRepo      domain.FundRepository
	dataLoader    *FundDataLoader
	searchByQuery func(ctx context.Context, query string) ([]eastmoneySearchResult, error)
	now           func() time.Time
	retryCooldown time.Duration
}

// NewFundResolver creates a new fund resolver.
func NewFundResolver(db *gorm.DB, fundRepo domain.FundRepository) *FundResolver {
	return &FundResolver{
		mappingStore:  &gormFundMappingStore{db: db},
		fundRepo:      fundRepo,
		dataLoader:    NewFundDataLoader(fundRepo),
		searchByQuery: searchFundsByKeyword,
		now:           time.Now,
		retryCooldown: defaultFailedMappingRetryCooldown,
	}
}

// SetFundDataLoader overrides the transient fund data loader used for target ETF hydration.
func (r *FundResolver) SetFundDataLoader(loader *FundDataLoader) {
	if loader != nil {
		r.dataLoader = loader
	}
}

// IsFeederFund checks if a fund is a feeder fund (联接基金).
func IsFeederFund(fundName string) bool {
	return strings.Contains(fundName, "联接")
}

// GetHoldingsWithFallback returns holdings for a fund, with fallback to the target ETF for feeder funds.
func (r *FundResolver) GetHoldingsWithFallback(ctx context.Context, fundID string, fundName string) ([]domain.StockHolding, string, error) {
	holdings, err := r.fundRepo.GetFundHoldings(ctx, fundID)
	if err != nil {
		return nil, fundID, err
	}
	if len(holdings) > 0 {
		return holdings, fundID, nil
	}

	if !IsFeederFund(fundName) {
		return nil, fundID, nil
	}

	log.Printf("🔍 Fund %s (%s) is a feeder fund with no direct holdings, attempting to resolve target ETF...", fundID, fundName)

	cachedMapping, err := r.getCachedMapping(ctx, fundID)
	if err != nil {
		log.Printf("⚠️ Failed to load cached mapping for %s: %v", fundID, err)
	}

	if cachedMapping != nil && cachedMapping.IsResolved && cachedMapping.TargetCode != "" {
		targetCode := cachedMapping.TargetCode
		log.Printf("✅ Found existing mapping: %s -> %s", fundID, targetCode)
		holdings, err := r.fundRepo.GetFundHoldings(ctx, targetCode)
		if err != nil {
			return nil, fundID, fmt.Errorf("failed to get holdings for target ETF %s: %w", targetCode, err)
		}
		if len(holdings) > 0 {
			return holdings, targetCode, nil
		}
	}

	if cachedMapping != nil && !cachedMapping.IsResolved && !r.shouldRetryFailedMapping(cachedMapping) {
		log.Printf("ℹ️ Recent unresolved mapping cached for %s, skipping target ETF re-resolution during cooldown", fundID)
		return nil, fundID, fmt.Errorf("target ETF resolution temporarily skipped after recent failure")
	}

	targetCode, err := r.resolveAndSaveMapping(ctx, fundID, fundName)
	if err != nil {
		return nil, fundID, fmt.Errorf("failed to resolve target ETF: %w", err)
	}
	if targetCode == "" {
		return nil, fundID, fmt.Errorf("could not find target ETF for feeder fund: %s", fundName)
	}

	holdings, err = r.fundRepo.GetFundHoldings(ctx, targetCode)
	if (err != nil || len(holdings) == 0) && r.dataLoader != nil {
		_, hydratedHoldings, hydrateErr := r.dataLoader.FetchTransientFundData(ctx, targetCode)
		if hydrateErr != nil {
			log.Printf("⚠️ Failed to hydrate target ETF %s: %v", targetCode, hydrateErr)
		} else if len(hydratedHoldings) > 0 {
			holdings = hydratedHoldings
			err = nil
		}
	}

	// Some target ETFs do not expose stock holdings. The caller can then fall back to direct ETF quotes.
	if err != nil || len(holdings) == 0 {
		log.Printf("⚠️ Target ETF %s has no holdings, will fallback to direct quote", targetCode)
		return nil, targetCode, nil
	}

	return holdings, targetCode, nil
}

func (r *FundResolver) getCachedMapping(ctx context.Context, feederCode string) (*database.FundMapping, error) {
	if r.mappingStore == nil {
		return nil, nil
	}
	return r.mappingStore.GetByFeederCode(ctx, feederCode)
}

func (r *FundResolver) shouldRetryFailedMapping(mapping *database.FundMapping) bool {
	if mapping == nil || mapping.IsResolved {
		return true
	}
	if r.retryCooldown <= 0 {
		return true
	}

	lastAttempt := mapping.UpdatedAt
	if lastAttempt.IsZero() {
		return true
	}

	now := time.Now()
	if r.now != nil {
		now = r.now()
	}

	return now.After(lastAttempt.Add(r.retryCooldown))
}

// resolveAndSaveMapping resolves the target ETF via deterministic Eastmoney search and stores the mapping.
func (r *FundResolver) resolveAndSaveMapping(ctx context.Context, feederCode, feederName string) (string, error) {
	targetCode, err := r.resolveBySearch(ctx, feederCode, feederName)
	if err != nil {
		r.saveMapping(ctx, feederCode, feederName, "", err.Error())
		return "", err
	}

	r.saveMapping(ctx, feederCode, feederName, targetCode, "")
	log.Printf("✅ Resolved and saved mapping via search: %s -> %s", feederCode, targetCode)
	return targetCode, nil
}

func (r *FundResolver) resolveBySearch(ctx context.Context, feederCode, feederName string) (string, error) {
	queries := buildResolverQueries(feederName)
	searchFunc := r.searchByQuery
	if searchFunc == nil {
		searchFunc = searchFundsByKeyword
	}
	for _, query := range queries {
		results, err := searchFunc(ctx, query)
		if err != nil {
			continue
		}
		for _, candidate := range results {
			name := normalizeFundName(candidate.Name)
			if candidate.Code == feederCode {
				continue
			}
			if strings.Contains(name, "联接") {
				continue
			}
			if !strings.Contains(strings.ToUpper(name), "ETF") {
				continue
			}
			return candidate.Code, nil
		}
	}
	return "", fmt.Errorf("search fallback could not find target ETF for %s", feederName)
}

type eastmoneySearchResponse struct {
	Datas []eastmoneySearchResult `json:"Datas"`
}

type eastmoneySearchResult struct {
	Code string `json:"CODE"`
	Name string `json:"NAME"`
}

func searchFundsByKeyword(ctx context.Context, query string) ([]eastmoneySearchResult, error) {
	apiURL := fmt.Sprintf("http://fundsuggest.eastmoney.com/FundSearch/api/FundSearchAPI.ashx?m=1&key=%s", url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "http://fund.eastmoney.com/")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsed eastmoneySearchResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	return parsed.Datas, nil
}

func buildResolverQueries(feederName string) []string {
	base := normalizeFundName(feederName)
	if idx := strings.Index(base, "联接"); idx > 0 {
		base = base[:idx]
	}
	base = strings.ReplaceAll(base, "发起式", "")
	base = strings.ReplaceAll(base, "基金", "")
	if strings.Contains(base, "ETF") {
		base = base[:strings.Index(base, "ETF")+len("ETF")]
	}

	queries := []string{base}
	if !strings.Contains(strings.ToUpper(base), "ETF") {
		queries = append(queries, base+"ETF")
	}

	seen := make(map[string]struct{})
	filtered := make([]string, 0, len(queries))
	for _, q := range queries {
		q = strings.TrimSpace(q)
		if q == "" {
			continue
		}
		if _, ok := seen[q]; ok {
			continue
		}
		seen[q] = struct{}{}
		filtered = append(filtered, q)
	}
	return filtered
}

func normalizeFundName(name string) string {
	replacer := strings.NewReplacer(" ", "", "（", "", "）", "", "(", "", ")", "", "　", "")
	return replacer.Replace(strings.TrimSpace(name))
}

// saveMapping saves a fund mapping to the database.
func (r *FundResolver) saveMapping(ctx context.Context, feederCode, feederName, targetCode, errorMsg string) {
	if r.mappingStore == nil {
		return
	}

	mapping := database.FundMapping{
		FeederCode: feederCode,
		FeederName: feederName,
		TargetCode: targetCode,
		IsResolved: targetCode != "" && errorMsg == "",
	}

	if mapping.IsResolved {
		now := time.Now()
		if r.now != nil {
			now = r.now()
		}
		mapping.ResolvedAt = &now
	} else {
		mapping.ResolveError = errorMsg
	}

	if err := r.mappingStore.Save(ctx, &mapping); err != nil {
		log.Printf("⚠️ Failed to persist fund mapping %s: %v", feederCode, err)
	}
}
