package service

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/go-resty/resty/v2"
)

const (
	cninfoTopSearchURL         = "https://www.cninfo.com.cn/new/information/topSearch/query"
	cninfoAnnouncementQueryURL = "https://www.cninfo.com.cn/new/hisAnnouncement/query"
)

type cninfoSearchResult struct {
	Code  string `json:"code"`
	OrgID string `json:"orgId"`
	Name  string `json:"zwjc"`
}

type cninfoAnnouncement struct {
	SecCode           string `json:"secCode"`
	SecName           string `json:"secName"`
	OrgID             string `json:"orgId"`
	AnnouncementID    string `json:"announcementId"`
	AnnouncementTitle string `json:"announcementTitle"`
	AnnouncementTime  int64  `json:"announcementTime"`
	AdjunctURL        string `json:"adjunctUrl"`
}

type cninfoAnnouncementResponse struct {
	Announcements []cninfoAnnouncement `json:"announcements"`
}

type cachedHoldingEvents struct {
	expiresAt time.Time
	events    []domain.FundAnalysisEventImpact
}

type HoldingNewsSource struct {
	client *resty.Client
	ttl    time.Duration

	mu         sync.Mutex
	orgCache   map[string]cninfoSearchResult
	eventCache map[string]cachedHoldingEvents
}

func NewHoldingNewsSource() *HoldingNewsSource {
	client := resty.New().
		SetTimeout(20*time.Second).
		SetRetryCount(2).
		SetRetryWaitTime(500*time.Millisecond).
		SetHeader("User-Agent", "Mozilla/5.0").
		SetHeader("Referer", "https://www.cninfo.com.cn/new/commonUrl/pageOfSearch?url=disclosure/list/search").
		SetHeader("X-Requested-With", "XMLHttpRequest")

	return &HoldingNewsSource{
		client:     client,
		ttl:        3 * time.Hour,
		orgCache:   make(map[string]cninfoSearchResult),
		eventCache: make(map[string]cachedHoldingEvents),
	}
}

var defaultHoldingNewsSource = NewHoldingNewsSource()

func LoadCurrentHoldingNewsEvents(ctx context.Context, holdings []domain.StockHolding, now time.Time) []domain.FundAnalysisEventImpact {
	return defaultHoldingNewsSource.LoadCurrentHoldingNewsEvents(ctx, holdings, now)
}

func (s *HoldingNewsSource) LoadCurrentHoldingNewsEvents(ctx context.Context, holdings []domain.StockHolding, now time.Time) []domain.FundAnalysisEventImpact {
	selected := selectAnnouncementTargetHoldings(holdings, 3)
	if len(selected) == 0 {
		return nil
	}

	events := make([]domain.FundAnalysisEventImpact, 0, 3)
	for _, holding := range selected {
		itemEvents, err := s.loadHoldingEvents(ctx, holding, now)
		if err != nil {
			continue
		}
		events = append(events, itemEvents...)
	}

	sort.Slice(events, func(i, j int) bool {
		return eventImpactRank(events[i]) > eventImpactRank(events[j])
	})
	if len(events) > 3 {
		events = events[:3]
	}
	return events
}

func (s *HoldingNewsSource) loadHoldingEvents(ctx context.Context, holding domain.StockHolding, now time.Time) ([]domain.FundAnalysisEventImpact, error) {
	cacheKey := strings.TrimSpace(holding.StockCode)
	if cacheKey == "" {
		return nil, nil
	}

	s.mu.Lock()
	if cached, ok := s.eventCache[cacheKey]; ok && cached.expiresAt.After(now) {
		events := append([]domain.FundAnalysisEventImpact(nil), cached.events...)
		s.mu.Unlock()
		return events, nil
	}
	s.mu.Unlock()

	searchItem, err := s.resolveCninfoStock(ctx, holding.StockCode)
	if err != nil || searchItem.OrgID == "" {
		return nil, err
	}

	announcements, err := s.fetchRecentAnnouncements(ctx, holding, searchItem.OrgID, now)
	if err != nil {
		return nil, err
	}
	if len(announcements) == 0 {
		return nil, nil
	}

	events := make([]domain.FundAnalysisEventImpact, 0, len(announcements))
	for _, item := range announcements {
		if impact, ok := announcementToImpact(item, holding); ok {
			events = append(events, impact)
		}
	}
	if len(events) == 0 {
		return nil, nil
	}

	s.mu.Lock()
	s.eventCache[cacheKey] = cachedHoldingEvents{
		expiresAt: now.Add(s.ttl),
		events:    append([]domain.FundAnalysisEventImpact(nil), events...),
	}
	s.mu.Unlock()
	return events, nil
}

func (s *HoldingNewsSource) resolveCninfoStock(ctx context.Context, stockCode string) (cninfoSearchResult, error) {
	stockCode = strings.TrimSpace(stockCode)
	if stockCode == "" {
		return cninfoSearchResult{}, nil
	}

	s.mu.Lock()
	if item, ok := s.orgCache[stockCode]; ok {
		s.mu.Unlock()
		return item, nil
	}
	s.mu.Unlock()

	var result []cninfoSearchResult
	_, err := s.client.R().
		SetContext(ctx).
		SetFormData(map[string]string{
			"keyWord": stockCode,
			"maxNum":  "10",
			"plate":   "",
		}).
		SetResult(&result).
		Post(cninfoTopSearchURL)
	if err != nil {
		return cninfoSearchResult{}, err
	}

	for _, item := range result {
		if strings.TrimSpace(item.Code) == stockCode && strings.TrimSpace(item.OrgID) != "" {
			s.mu.Lock()
			s.orgCache[stockCode] = item
			s.mu.Unlock()
			return item, nil
		}
	}
	return cninfoSearchResult{}, nil
}

func (s *HoldingNewsSource) fetchRecentAnnouncements(ctx context.Context, holding domain.StockHolding, orgID string, now time.Time) ([]cninfoAnnouncement, error) {
	seDateStart := now.AddDate(0, 0, -45).Format("2006-01-02")
	seDateEnd := now.Format("2006-01-02")

	payload := map[string]string{
		"pageNum":   "1",
		"pageSize":  "6",
		"column":    "szse",
		"tabName":   "fulltext",
		"plate":     "",
		"stock":     strings.TrimSpace(holding.StockCode) + "," + strings.TrimSpace(orgID),
		"searchkey": "",
		"secid":     "",
		"category":  "category_yjygjxz_szsh;category_ndbg_szsh;category_bndbg_szsh;category_yjdbg_szsh;category_sjdbg_szsh;category_rcjy_szsh;category_gszl_szsh;",
		"trade":     "",
		"seDate":    seDateStart + "~" + seDateEnd,
		"sortName":  "",
		"sortType":  "",
		"isHLtitle": "true",
	}

	var response cninfoAnnouncementResponse
	_, err := s.client.R().
		SetContext(ctx).
		SetFormData(payload).
		SetResult(&response).
		Post(cninfoAnnouncementQueryURL)
	if err != nil {
		return nil, err
	}
	return response.Announcements, nil
}

func selectAnnouncementTargetHoldings(holdings []domain.StockHolding, limit int) []domain.StockHolding {
	if len(holdings) == 0 || limit <= 0 {
		return nil
	}
	filtered := make([]domain.StockHolding, 0, len(holdings))
	for _, holding := range holdings {
		code := strings.TrimSpace(holding.StockCode)
		if len(code) != 6 {
			continue
		}
		switch holding.Exchange {
		case domain.ExchangeSH, domain.ExchangeSZ:
			filtered = append(filtered, holding)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].HoldingRatio.GreaterThan(filtered[j].HoldingRatio)
	})
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered
}

func announcementToImpact(item cninfoAnnouncement, holding domain.StockHolding) (domain.FundAnalysisEventImpact, bool) {
	title := strings.TrimSpace(item.AnnouncementTitle)
	if title == "" {
		return domain.FundAnalysisEventImpact{}, false
	}

	impact, strength, accepted := classifyAnnouncementTitle(title)
	if !accepted {
		return domain.FundAnalysisEventImpact{}, false
	}

	date := ""
	if item.AnnouncementTime > 0 {
		date = time.UnixMilli(item.AnnouncementTime).In(time.Local).Format("2006-01-02")
	}
	summary := title
	if date != "" {
		summary = "近期开启事件：" + date + " 发布《" + title + "》。"
	}

	return domain.FundAnalysisEventImpact{
		Code:              "holding_current_notice_" + strings.TrimSpace(holding.StockCode) + "_" + strings.TrimSpace(item.AnnouncementID),
		Title:             compressAnnouncementTitle(holding.StockName, title),
		Impact:            impact,
		Summary:           summary,
		TargetScope:       "holding",
		Strength:          strength,
		Horizon:           "current",
		RelatedSymbols:    compactSymbols(holding.StockCode),
		WeightHint:        decimalPointerFromValue(holding.HoldingRatio),
		SourceName:        "巨潮资讯",
		SourceURL:         "https://static.cninfo.com.cn/" + strings.TrimLeft(item.AdjunctURL, "/"),
		SourcePublishedAt: date,
		SourceConfidence:  "high",
		SourceTier:        "official",
	}, true
}

func classifyAnnouncementTitle(title string) (impact string, strength string, accepted bool) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", "", false
	}

	negativeStrong := []string{"风险提示", "减持", "处罚", "问询", "诉讼", "仲裁", "预亏", "首亏", "续亏", "终止", "失败", "异常波动", "下滑", "下降", "未中标", "解约"}
	positiveStrong := []string{"预增", "增长", "扭亏", "回购", "增持", "中标", "签署", "获批", "订单", "进展公告", "战略合作", "业绩快报"}
	neutralMedium := []string{"季度报告", "年度报告", "半年度报告", "业绩说明会", "业绩预告", "经营情况", "调研活动", "投资者关系活动"}

	for _, keyword := range negativeStrong {
		if strings.Contains(title, keyword) {
			return "negative", "high", true
		}
	}
	for _, keyword := range positiveStrong {
		if strings.Contains(title, keyword) {
			return "positive", "high", true
		}
	}
	for _, keyword := range neutralMedium {
		if strings.Contains(title, keyword) {
			return "neutral", "medium", true
		}
	}
	return "", "", false
}

func compressAnnouncementTitle(stockName, title string) string {
	stockName = strings.TrimSpace(stockName)
	title = strings.TrimSpace(title)
	if stockName != "" && strings.Contains(title, stockName) {
		title = strings.ReplaceAll(title, stockName, "")
		title = strings.TrimLeft(title, "关于")
		title = strings.TrimSpace(title)
	}
	if stockName == "" {
		return limitTextForImpact(title, 22)
	}
	return limitTextForImpact(stockName+title, 24)
}

func limitTextForImpact(raw string, max int) string {
	runes := []rune(strings.TrimSpace(raw))
	if len(runes) <= max {
		return string(runes)
	}
	return string(runes[:max]) + "…"
}

func eventImpactRank(event domain.FundAnalysisEventImpact) int {
	score := 0
	switch strings.TrimSpace(event.Impact) {
	case "negative":
		score += 30
	case "positive":
		score += 20
	default:
		score += 10
	}
	switch strings.TrimSpace(event.Strength) {
	case "high":
		score += 20
	case "medium":
		score += 10
	}
	if strings.TrimSpace(event.Horizon) == "current" {
		score += 10
	}
	if strings.TrimSpace(event.TargetScope) == "holding" {
		score += 5
	}
	if event.WeightHint != nil {
		weight := decimalToFloat(*event.WeightHint)
		switch {
		case weight >= 8:
			score += 12
		case weight >= 5:
			score += 8
		case weight >= 3:
			score += 4
		}
	}
	return score
}
