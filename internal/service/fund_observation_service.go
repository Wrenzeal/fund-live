package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/go-resty/resty/v2"
	"github.com/shopspring/decimal"
)

const observationWindowDays = 45

var observationTitleTags = regexp.MustCompile(`<[^>]*>`)

var observationLocation = time.FixedZone("CST", 8*60*60)

// ObservationHoldingsResolver returns the fund code from which holdings were obtained.
type ObservationHoldingsResolver interface {
	GetHoldingsWithFallback(context.Context, string, string) ([]domain.StockHolding, string, error)
}

// ObservationNoticeBatch preserves failed requests alongside successful results.
type ObservationNoticeBatch struct {
	Notices            []domain.FundObservationNotice
	Requests           int
	SuccessfulRequests int
	Err                error
}

func (b *ObservationNoticeBatch) recordRequest(err error) {
	b.Requests++
	if err == nil {
		b.SuccessfulRequests++
	} else {
		b.Err = errors.Join(b.Err, err)
	}
}

func (b *ObservationNoticeBatch) append(other ObservationNoticeBatch) {
	b.Notices = append(b.Notices, other.Notices...)
	b.Requests += other.Requests
	b.SuccessfulRequests += other.SuccessfulRequests
	b.Err = errors.Join(b.Err, other.Err)
}

type fundObservationNoticeLoader interface {
	LoadFundObservationNotices(context.Context, string, string, time.Time) ObservationNoticeBatch
}

type holdingObservationNoticeLoader interface {
	LoadHoldingObservationNotices(context.Context, []domain.StockHolding, time.Time) ObservationNoticeBatch
}

type FundObservationService struct {
	fundRepo         domain.FundRepository
	holdingsResolver ObservationHoldingsResolver
	fundNotices      fundObservationNoticeLoader
	holdingNotices   holdingObservationNoticeLoader
	timeout          time.Duration
}

func NewFundObservationService(fundRepo domain.FundRepository, resolver ObservationHoldingsResolver) *FundObservationService {
	return &FundObservationService{
		fundRepo: fundRepo, holdingsResolver: resolver,
		fundNotices: defaultFundNoticeSource, holdingNotices: defaultHoldingNewsSource,
		timeout: 15 * time.Second,
	}
}

func (s *FundObservationService) Observe(ctx context.Context, fundID string, now time.Time) (*domain.Fund, *domain.FundObservation, error) {
	if s == nil || s.fundRepo == nil {
		return nil, nil, errors.New("observation repository unavailable")
	}
	if now.IsZero() {
		now = time.Now()
	}
	now = now.In(observationLocation)
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	fund, err := s.fundRepo.GetFundByID(ctx, fundID)
	if err != nil || fund == nil {
		return fund, nil, err
	}
	observation := &domain.FundObservation{
		CheckedAt: now, WindowDays: observationWindowDays,
		DisclosedWeightPercent: decimal.Zero, HoldingsStatus: "missing",
		Notices: []domain.FundObservationNotice{}, Sources: []domain.FundObservationSource{},
		Notes: []string{
			"公开持仓为最近披露样本，不代表实时或全部持仓；报告期距今天数不等于公告发布日期。",
			"公告仅保留来源标题与日期，不据此判断涨跌或给出买卖建议。",
			"基金公告读取各分类最新一页；持仓公告仅覆盖前三大沪深持仓的近期披露，结果不代表全量公告。",
		},
	}
	// Fund notices start independently so unavailable holdings cannot hide them.
	fundResults := startObservationLoad(func() ObservationNoticeBatch {
		return s.fundNotices.LoadFundObservationNotices(ctx, fundID, "fund", now)
	})
	holdings, sourceCode, holdingsErr := s.loadHoldings(ctx, fund)
	if holdingsErr != nil {
		observation.HoldingsStatus = "unavailable"
		log.Printf("Fund observation holdings unavailable for %s: %v", fundID, holdingsErr)
	} else if len(holdings) > 0 {
		observation.HoldingsStatus = "available"
		observation.HoldingsCount = len(holdings)
		observation.HoldingsSource = "direct_holdings"
		if sourceCode != "" && sourceCode != fundID {
			observation.HoldingsSource = "target_etf_fallback"
			observation.Notes = append(observation.Notes, "持仓来自目标 ETF，披露覆盖和报告期对应目标 ETF，不等于联接基金实际仓位。")
		}
		for _, holding := range holdings {
			if holding.HoldingRatio.IsPositive() {
				observation.DisclosedWeightPercent = observation.DisclosedWeightPercent.Add(holding.HoldingRatio)
			}
		}
		observation.LatestHoldingPeriod, observation.HoldingAgeDays = observationHoldingPeriod(holdings, now)
		if observation.HoldingAgeDays == nil {
			observation.Notes = append(observation.Notes, "持仓缺少有效报告期，无法确认距今天数。")
		}
	}
	var holdingResults, targetResults <-chan ObservationNoticeBatch
	if holdingsErr == nil && len(holdings) > 0 {
		holdingResults = startObservationLoad(func() ObservationNoticeBatch {
			return s.holdingNotices.LoadHoldingObservationNotices(ctx, holdings, now)
		})
	}
	if holdingsErr == nil && isObservationFundCode(sourceCode) && sourceCode != fundID {
		targetResults = startObservationLoad(func() ObservationNoticeBatch {
			return s.fundNotices.LoadFundObservationNotices(ctx, sourceCode, "target_etf", now)
		})
	}
	fundBatch := receiveObservationLoad(ctx, fundResults)
	if targetResults != nil {
		fundBatch.append(receiveObservationLoad(ctx, targetResults))
	}
	observation.Sources = append(observation.Sources, observationSourceState("fund_notices", fundBatch, now))
	observation.Notices = append(observation.Notices, fundBatch.Notices...)
	if holdingsErr != nil {
		observation.Sources = append(observation.Sources, domain.FundObservationSource{Code: "holding_notices", Status: "unavailable", CheckedAt: now, Message: "持仓暂不可用，无法检查重仓股公告；基金公告仍独立读取。"})
	} else if holdingResults != nil {
		batch := receiveObservationLoad(ctx, holdingResults)
		state := observationSourceState("holding_notices", batch, now)
		if batch.Requests == 0 {
			state.Message = "当前公告源仅覆盖沪深持仓，其他市场暂未接入。"
		}
		observation.Sources = append(observation.Sources, state)
		observation.Notices = append(observation.Notices, batch.Notices...)
	} else {
		observation.Sources = append(observation.Sources, domain.FundObservationSource{Code: "holding_notices", Status: "not_applicable", CheckedAt: now, Message: "尚无可用公开持仓；基金公告仍独立读取。"})
	}
	observation.Notices = normalizeObservationNotices(observation.Notices, now)
	return fund, observation, nil
}

func (s *FundObservationService) loadHoldings(ctx context.Context, fund *domain.Fund) ([]domain.StockHolding, string, error) {
	if s.holdingsResolver != nil {
		return s.holdingsResolver.GetHoldingsWithFallback(ctx, fund.ID, fund.Name)
	}
	holdings, err := s.fundRepo.GetFundHoldings(ctx, fund.ID)
	return holdings, fund.ID, err
}

func startObservationLoad(load func() ObservationNoticeBatch) <-chan ObservationNoticeBatch {
	result := make(chan ObservationNoticeBatch, 1)
	go func() { result <- load() }()
	return result
}

func receiveObservationLoad(ctx context.Context, result <-chan ObservationNoticeBatch) ObservationNoticeBatch {
	select {
	case batch := <-result:
		return batch
	default:
	}
	select {
	case batch := <-result:
		return batch
	case <-ctx.Done():
		// Let the context-aware HTTP source report pages completed before cancellation.
		cleanup := time.NewTimer(100 * time.Millisecond)
		defer cleanup.Stop()
		select {
		case batch := <-result:
			return batch
		case <-cleanup.C:
			return ObservationNoticeBatch{Requests: 1, Err: ctx.Err()}
		}
	}
}

func observationSourceState(code string, batch ObservationNoticeBatch, now time.Time) domain.FundObservationSource {
	state := domain.FundObservationSource{Code: code, CheckedAt: now, Status: "ok"}
	switch {
	case batch.Requests == 0 && batch.Err == nil:
		state.Status = "not_applicable"
	case batch.SuccessfulRequests == 0:
		state.Status, state.Message = "unavailable", "公告来源暂不可用，请稍后重试；不能据此判断没有公告。"
	case batch.Err != nil || batch.SuccessfulRequests < batch.Requests:
		state.Status, state.Message = "partial", "部分公告请求未能完成，当前结果可能不完整，可稍后重试。"
	}
	if batch.Err != nil {
		log.Printf("Fund observation source %s: %v", code, batch.Err)
	}
	return state
}

func normalizeObservationNotices(notices []domain.FundObservationNotice, now time.Time) []domain.FundObservationNotice {
	now = now.In(observationLocation)
	cutoff := observationCalendarDay(now).AddDate(0, 0, -observationWindowDays)
	result := make([]domain.FundObservationNotice, 0, len(notices))
	seen := make(map[string]struct{}, len(notices))
	seenIDs := make(map[string]struct{}, len(notices))
	for _, item := range notices {
		item.Title = strings.TrimSpace(html.UnescapeString(observationTitleTags.ReplaceAllString(item.Title, "")))
		published, ok := observationNoticeDate(item.PublishedAt, now.Location())
		if !ok || published.After(now) || published.Before(cutoff) || item.Title == "" || !observationURLValid(item.SourceURL) {
			continue
		}
		if item.Scope != "fund" && item.Scope != "holding" && item.Scope != "target_etf" {
			continue
		}
		item.PublishedAt = published.In(observationLocation).Format(time.DateOnly)
		key := strings.Join([]string{item.Scope, item.RelatedSymbol, item.PublishedAt, item.Title, item.SourceName}, "|")
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		if strings.TrimSpace(item.ID) == "" {
			item.ID = fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
		}
		idKey := strings.Join([]string{item.Scope, item.RelatedSymbol, item.SourceName, item.ID}, "|")
		if _, exists := seenIDs[idKey]; exists {
			continue
		}
		seenIDs[idKey] = struct{}{}
		result = append(result, item)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].PublishedAt == result[j].PublishedAt {
			return result[i].ID < result[j].ID
		}
		return result[i].PublishedAt > result[j].PublishedAt
	})
	return result
}

func observationNoticeDate(value string, location *time.Location) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", time.DateOnly} {
		if parsed, err := time.ParseInLocation(layout, strings.TrimSpace(value), location); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func observationHoldingPeriod(holdings []domain.StockHolding, now time.Time) (string, *int) {
	now = observationCalendarDay(now)
	var latest time.Time
	label := ""
	for _, holding := range holdings {
		period := strings.TrimSpace(holding.ReportingPeriod)
		date, err := time.ParseInLocation(time.DateOnly, period, now.Location())
		if err != nil {
			cleaned := strings.ToUpper(strings.ReplaceAll(period, "-", ""))
			if len(cleaned) != 6 || cleaned[4] != 'Q' || cleaned[5] < '1' || cleaned[5] > '4' {
				continue
			}
			year, err := strconv.Atoi(cleaned[:4])
			if err != nil || year < 1900 {
				continue
			}
			quarter := int(cleaned[5] - '0')
			date = time.Date(year, time.Month(quarter*3+1), 0, 0, 0, 0, 0, now.Location())
		}
		if date.After(now) || !date.After(latest) {
			continue
		}
		latest, label = date, period
	}
	if latest.IsZero() {
		return "", nil
	}
	age := int(now.Sub(latest).Hours() / 24)
	return label, &age
}

func observationCalendarDay(value time.Time) time.Time {
	value = value.In(observationLocation)
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, observationLocation)
}

func isObservationFundCode(code string) bool {
	if len(code) != 6 {
		return false
	}
	for _, character := range code {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func observationURLValid(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && (parsed.Scheme == "https" || parsed.Scheme == "http") && parsed.Host != "" && parsed.User == nil
}

// Reuse transports and request builders, without the scored-event caches or classifiers.
func observationHTTPClient(base *resty.Client) *resty.Client {
	client := resty.NewWithClient(base.GetClient()).SetHeader("User-Agent", "Mozilla/5.0")
	return client.OnAfterResponse(func(_ *resty.Client, response *resty.Response) error {
		if response.StatusCode() < 200 || response.StatusCode() >= 300 {
			return fmt.Errorf("notice source HTTP %d", response.StatusCode())
		}
		payload := strings.TrimSpace(response.String())
		if strings.HasPrefix(payload, "jQuery(") && strings.HasSuffix(payload, ")") {
			payload = payload[len("jQuery(") : len(payload)-1]
		}
		if strings.Contains(response.Request.URL, "/topSearch/") {
			var items []json.RawMessage
			if err := json.Unmarshal([]byte(payload), &items); err != nil || items == nil {
				return errors.New("invalid stock lookup response")
			}
			return nil
		}
		var object map[string]json.RawMessage
		if err := json.Unmarshal([]byte(payload), &object); err != nil {
			return fmt.Errorf("invalid notice response: %w", err)
		}
		key := "Data"
		if strings.Contains(response.Request.URL, "/hisAnnouncement/") {
			key = "announcements"
		}
		if _, exists := object[key]; !exists {
			return errors.New("notice response missing data field")
		}
		return nil
	})
}
