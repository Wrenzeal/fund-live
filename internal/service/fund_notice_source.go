package service

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/go-resty/resty/v2"
)

const eastmoneyFundNoticeURL = "https://api.fund.eastmoney.com/f10/JJGG"

type eastmoneyFundNoticeResponse struct {
	Data []eastmoneyFundNoticeItem `json:"Data"`
}

type eastmoneyFundNoticeItem struct {
	FundCode       string `json:"FUNDCODE"`
	Title          string `json:"TITLE"`
	ShortTitle     string `json:"ShortTitle"`
	NewCategory    string `json:"NEWCATEGORY"`
	PublishDate    string `json:"PUBLISHDATEDesc"`
	AttachmentType string `json:"ATTACHTYPE"`
	ID             string `json:"ID"`
}

type cachedFundNoticeEvents struct {
	expiresAt time.Time
	events    []domain.FundAnalysisEventImpact
}

type FundNoticeSource struct {
	client *resty.Client
	ttl    time.Duration

	mu         sync.Mutex
	eventCache map[string]cachedFundNoticeEvents
}

func NewFundNoticeSource() *FundNoticeSource {
	client := resty.New().
		SetTimeout(15*time.Second).
		SetRetryCount(2).
		SetRetryWaitTime(500*time.Millisecond).
		SetHeader("User-Agent", "Mozilla/5.0")

	return &FundNoticeSource{
		client:     client,
		ttl:        3 * time.Hour,
		eventCache: make(map[string]cachedFundNoticeEvents),
	}
}

var defaultFundNoticeSource = NewFundNoticeSource()

func LoadCurrentFundNoticeEvents(ctx context.Context, fundCode string, now time.Time) []domain.FundAnalysisEventImpact {
	return defaultFundNoticeSource.LoadCurrentFundNoticeEvents(ctx, fundCode, now)
}

func (s *FundNoticeSource) LoadCurrentFundNoticeEvents(ctx context.Context, fundCode string, now time.Time) []domain.FundAnalysisEventImpact {
	fundCode = strings.TrimSpace(fundCode)
	if fundCode == "" {
		return nil
	}

	s.mu.Lock()
	if cached, ok := s.eventCache[fundCode]; ok && cached.expiresAt.After(now) {
		events := append([]domain.FundAnalysisEventImpact(nil), cached.events...)
		s.mu.Unlock()
		return events
	}
	s.mu.Unlock()

	types := []string{"3", "4", "5", "6"}
	events := make([]domain.FundAnalysisEventImpact, 0, 4)
	for _, typeCode := range types {
		noticeItems, err := s.fetchNoticePage(ctx, fundCode, typeCode)
		if err != nil {
			continue
		}
		for _, item := range noticeItems {
			if impact, ok := fundNoticeToImpact(item, fundCode, now); ok {
				events = append(events, impact)
			}
		}
	}

	events = dedupeFundNoticeEvents(events)
	if len(events) > 3 {
		events = events[:3]
	}

	s.mu.Lock()
	s.eventCache[fundCode] = cachedFundNoticeEvents{
		expiresAt: now.Add(s.ttl),
		events:    append([]domain.FundAnalysisEventImpact(nil), events...),
	}
	s.mu.Unlock()
	return events
}

func (s *FundNoticeSource) fetchNoticePage(ctx context.Context, fundCode, typeCode string) ([]eastmoneyFundNoticeItem, error) {
	resp, err := s.client.R().
		SetContext(ctx).
		SetHeader("Referer", "https://fundf10.eastmoney.com/jjgg_"+fundCode+".html").
		SetQueryParams(map[string]string{
			"callback":  "jQuery",
			"fundcode":  fundCode,
			"pageIndex": "1",
			"pageSize":  "8",
			"type":      typeCode,
		}).
		Get(eastmoneyFundNoticeURL)
	if err != nil {
		return nil, err
	}

	payload := strings.TrimSpace(resp.String())
	if payload == "" {
		return nil, nil
	}
	if idx := strings.Index(payload, "("); idx >= 0 && strings.HasSuffix(payload, ")") {
		payload = payload[idx+1 : len(payload)-1]
	}

	var parsed eastmoneyFundNoticeResponse
	if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
		return nil, err
	}
	return parsed.Data, nil
}

func fundNoticeToImpact(item eastmoneyFundNoticeItem, fundCode string, now time.Time) (domain.FundAnalysisEventImpact, bool) {
	title := strings.TrimSpace(item.Title)
	if title == "" {
		return domain.FundAnalysisEventImpact{}, false
	}
	if item.PublishDate != "" {
		if publishedAt, err := time.ParseInLocation("2006-01-02", item.PublishDate, time.Local); err == nil {
			if now.Sub(publishedAt) > 45*24*time.Hour {
				return domain.FundAnalysisEventImpact{}, false
			}
		}
	}

	impact, strength, accepted := classifyFundNotice(title, item.NewCategory)
	if !accepted {
		return domain.FundAnalysisEventImpact{}, false
	}

	summary := title
	if item.PublishDate != "" {
		summary = "基金近期事件：" + item.PublishDate + " 发布《" + title + "》。"
	}

	return domain.FundAnalysisEventImpact{
		Code:              "fund_notice_" + fundCode + "_" + strings.TrimSpace(item.ID),
		Title:             limitTextForImpact(trimFundNoticeTitle(title), 24),
		Impact:            impact,
		Summary:           summary,
		TargetScope:       "fund",
		Strength:          strength,
		Horizon:           "current",
		SourceName:        "东方财富基金公告",
		SourceURL:         "https://fundf10.eastmoney.com/jjgg_" + fundCode + ".html",
		SourcePublishedAt: strings.TrimSpace(item.PublishDate),
		SourceConfidence:  "medium",
		SourceTier:        "official_aggregator",
	}, true
}

func classifyFundNotice(title, category string) (string, string, bool) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", "", false
	}

	negativeKeywords := []string{"风险提示", "暂停申购", "暂停赎回", "暂停大额", "清算", "终止", "基金经理变更", "基金经理离任", "异常波动", "溢价风险"}
	positiveKeywords := []string{"分红", "收益分配", "增加", "新增", "恢复申购", "恢复赎回", "开放申购", "开放赎回", "费率优惠", "份额折算"}
	neutralKeywords := []string{"季度报告", "年度报告", "半年度报告", "产品资料概要", "更新招募说明书", "流动性服务商", "申购赎回代理券商", "暂停客服电话服务", "高级管理人员变更", "上市交易提示"}

	for _, keyword := range negativeKeywords {
		if strings.Contains(title, keyword) {
			return "negative", "medium", true
		}
	}
	for _, keyword := range positiveKeywords {
		if strings.Contains(title, keyword) {
			return "positive", "medium", true
		}
	}
	for _, keyword := range neutralKeywords {
		if strings.Contains(title, keyword) {
			strength := "low"
			if strings.Contains(title, "季度报告") || strings.Contains(title, "年度报告") || strings.Contains(title, "半年度报告") {
				strength = "medium"
			}
			return "neutral", strength, true
		}
	}

	switch strings.TrimSpace(category) {
	case "3":
		return "neutral", "medium", true
	case "4", "5", "6":
		return "neutral", "low", true
	default:
		return "", "", false
	}
}

func trimFundNoticeTitle(title string) string {
	replacements := []string{
		"鹏华基金管理有限公司",
		"关于",
		"鹏华国证半导体芯片交易型开放式指数证券投资基金",
		"鹏华国证半导体芯片ETF",
	}
	trimmed := strings.TrimSpace(title)
	for _, item := range replacements {
		trimmed = strings.ReplaceAll(trimmed, item, "")
	}
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return strings.TrimSpace(title)
	}
	return trimmed
}

func dedupeFundNoticeEvents(events []domain.FundAnalysisEventImpact) []domain.FundAnalysisEventImpact {
	if len(events) == 0 {
		return nil
	}
	result := make([]domain.FundAnalysisEventImpact, 0, len(events))
	seen := make(map[string]struct{}, len(events))
	for _, event := range events {
		key := strings.TrimSpace(event.Title)
		if key == "" {
			key = strings.TrimSpace(event.Code)
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, event)
	}
	return result
}
