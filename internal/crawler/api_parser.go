package crawler

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/shopspring/decimal"
)

// EastmoneyAPIParser provides alternative parsing using Eastmoney's JSON APIs.
type EastmoneyAPIParser struct{}

// NewEastmoneyAPIParser creates a new API parser.
func NewEastmoneyAPIParser() *EastmoneyAPIParser {
	return &EastmoneyAPIParser{}
}

// FundDetailFromJS extracts fund details from the pingzhongdata JS file.
// This is a more robust parser that handles various edge cases.
type FundDetailFromJS struct {
	Code        string
	Name        string
	Type        string
	ManagerID   string
	ManagerName string
	Company     string
	NAV         decimal.Decimal
	PreviousNAV decimal.Decimal
	AccumNAV    decimal.Decimal
	NAVDate     string
}

// ParsePingzhongJS parses the pingzhongdata/{code}.js response.
// This file contains multiple var assignments with fund data.
func (p *EastmoneyAPIParser) ParsePingzhongJS(jsContent string, fundCode string) (*FundDetailFromJS, error) {
	result := &FundDetailFromJS{
		Code: fundCode,
	}

	// Extract fund name: var fS_name = "xxx";
	if name := p.extractVar(jsContent, "fS_name"); name != "" {
		result.Name = name
	}

	// Extract company short name: var fS_companySname = "xxx";
	if company := p.extractVar(jsContent, "fS_companySname"); company != "" {
		result.Company = company
	}

	// Extract latest NAV from Data_netWorthTrend
	// Format: var Data_netWorthTrend = [{x:1704038400000,y:1.2345,...},...];
	navTrendRe := regexp.MustCompile(`var\s+Data_netWorthTrend\s*=\s*(\[[\s\S]*?\])\s*;`)
	if matches := navTrendRe.FindStringSubmatch(jsContent); len(matches) > 1 {
		nav, prevNav, navDate := p.parseNavTrend(matches[1])
		result.NAV = nav
		result.PreviousNAV = prevNav
		result.NAVDate = navDate
	}

	accWorthRe := regexp.MustCompile(`var\s+Data_ACWorthTrend\s*=\s*(\[[\s\S]*?\])\s*;`)
	if matches := accWorthRe.FindStringSubmatch(jsContent); len(matches) > 1 {
		result.AccumNAV = p.parseAccumulatedWorthTrend(matches[1])
	}

	// Extract fund manager: var Data_currentFundManager = [{id:"xxx",name:"xxx",pic:"xxx"...}];
	managerRe := regexp.MustCompile(`var\s+Data_currentFundManager\s*=\s*(\[[\s\S]*?\])\s*;`)
	if matches := managerRe.FindStringSubmatch(jsContent); len(matches) > 1 {
		id, name := p.parseManager(matches[1])
		result.ManagerID = id
		result.ManagerName = name
	}

	// Infer fund type from name
	result.Type = p.inferFundType(result.Name)

	return result, nil
}

// extractVar extracts a simple var assignment value.
func (p *EastmoneyAPIParser) extractVar(content, varName string) string {
	// Pattern: var varName = "value";
	pattern := fmt.Sprintf(`var\s+%s\s*=\s*"([^"]+)"`, regexp.QuoteMeta(varName))
	re := regexp.MustCompile(pattern)
	if matches := re.FindStringSubmatch(content); len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// parseNavTrend parses the nav trend array and returns the latest NAV.
func (p *EastmoneyAPIParser) parseNavTrend(jsonArray string) (decimal.Decimal, decimal.Decimal, string) {
	type navPoint struct {
		X int64   `json:"x"`
		Y float64 `json:"y"`
	}

	var points []navPoint
	if err := json.Unmarshal([]byte(jsonArray), &points); err != nil {
		return decimal.Zero, decimal.Zero, ""
	}

	if len(points) == 0 {
		return decimal.Zero, decimal.Zero, ""
	}

	// Get the last point (most recent)
	lastPoint := points[len(points)-1]
	nav := decimal.NewFromFloat(lastPoint.Y)
	prevNav := decimal.Zero
	if len(points) > 1 {
		prevNav = decimal.NewFromFloat(points[len(points)-2].Y)
	}

	// Convert timestamp (milliseconds) to date
	t := time.Unix(lastPoint.X/1000, 0)
	date := t.Format("2006-01-02")

	return nav, prevNav, date
}

func (p *EastmoneyAPIParser) parseAccumulatedWorthTrend(jsonArray string) decimal.Decimal {
	var points [][]float64
	if err := json.Unmarshal([]byte(jsonArray), &points); err != nil {
		return decimal.Zero
	}

	if len(points) == 0 || len(points[len(points)-1]) < 2 {
		return decimal.Zero
	}

	return decimal.NewFromFloat(points[len(points)-1][1])
}

// parseManager parses the fund manager JSON array.
func (p *EastmoneyAPIParser) parseManager(jsonArray string) (string, string) {
	type manager struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	var managers []manager
	if err := json.Unmarshal([]byte(jsonArray), &managers); err != nil {
		return "", ""
	}

	if len(managers) == 0 {
		return "", ""
	}

	// Join multiple manager names
	names := make([]string, len(managers))
	for i, m := range managers {
		names[i] = m.Name
	}

	return managers[0].ID, strings.Join(names, "、")
}

// inferFundType infers fund type from name.
func (p *EastmoneyAPIParser) inferFundType(name string) string {
	nameLower := strings.ToLower(name)

	switch {
	case strings.Contains(nameLower, "qdii"):
		return "qdii"
	case strings.Contains(nameLower, "股票"):
		return "stock"
	case strings.Contains(nameLower, "债券") || strings.Contains(nameLower, "债"):
		return "bond"
	case strings.Contains(nameLower, "混合"):
		return "hybrid"
	case strings.Contains(nameLower, "货币"):
		return "money"
	case strings.Contains(nameLower, "指数") || strings.Contains(nameLower, "etf"):
		return "index"
	default:
		return "hybrid"
	}
}

// ToFund converts FundDetailFromJS to domain.Fund.
func (p *EastmoneyAPIParser) ToFund(detail *FundDetailFromJS) *domain.Fund {
	return &domain.Fund{
		ID:          detail.Code,
		Name:        detail.Name,
		Type:        detail.Type,
		Manager:     detail.ManagerName,
		Company:     detail.Company,
		NetAssetVal: detail.NAV,
		TotalScale:  decimal.Zero, // Scale requires separate API
		UpdatedAt:   time.Now(),
	}
}

// HoldingFromAPI represents holding data from Eastmoney API.
type HoldingFromAPI struct {
	StockCode    string `json:"GPDM,omitempty"` // 股票代码
	StockName    string `json:"GPJC,omitempty"` // 股票简称
	HoldingRatio string `json:"JZBL,omitempty"` // 占净值比例
	SharesWan    string `json:"CYGS,omitempty"` // 持有股数(万股)
	MarketWan    string `json:"CYSZ,omitempty"` // 持有市值(万元)
}

// ParseAlternativeHoldings attempts to parse holdings from alternative formats.
func (p *EastmoneyAPIParser) ParseAlternativeHoldings(content string) ([]HoldingFromAPI, error) {
	// Try to parse as JSON array directly
	var holdings []HoldingFromAPI
	if err := json.Unmarshal([]byte(content), &holdings); err == nil {
		return holdings, nil
	}

	// Try to extract from arryContent variable
	// Format: var arryContent = [{...},...];
	arrayRe := regexp.MustCompile(`var\s+arryContent\s*=\s*(\[[\s\S]*?\])\s*;`)
	if matches := arrayRe.FindStringSubmatch(content); len(matches) > 1 {
		if err := json.Unmarshal([]byte(matches[1]), &holdings); err == nil {
			return holdings, nil
		}
	}

	return nil, fmt.Errorf("could not parse holdings from content")
}

// ConvertAPIHoldings converts API holdings to domain holdings.
func (p *EastmoneyAPIParser) ConvertAPIHoldings(apiHoldings []HoldingFromAPI, reportPeriod string) []domain.StockHolding {
	result := make([]domain.StockHolding, 0, len(apiHoldings))

	for _, h := range apiHoldings {
		holding := domain.StockHolding{
			StockCode:       h.StockCode,
			StockName:       h.StockName,
			ReportingPeriod: reportPeriod,
		}

		// Infer exchange
		if len(h.StockCode) == 5 {
			holding.Exchange = domain.ExchangeHK
		} else if len(h.StockCode) == 6 {
			prefix := h.StockCode[:2]
			if prefix == "60" || prefix == "68" {
				holding.Exchange = domain.ExchangeSH
			} else {
				holding.Exchange = domain.ExchangeSZ
			}
		}

		// Parse ratio
		ratioStr := strings.ReplaceAll(h.HoldingRatio, "%", "")
		if ratio, err := decimal.NewFromString(ratioStr); err == nil {
			holding.HoldingRatio = ratio
		}

		// Parse shares (万股 -> 股)
		if shares, err := decimal.NewFromString(h.SharesWan); err == nil {
			holding.HoldingShares = shares.Mul(decimal.NewFromInt(10000))
		}

		// Parse market value (万元 -> 元)
		if value, err := decimal.NewFromString(h.MarketWan); err == nil {
			holding.MarketValue = value.Mul(decimal.NewFromInt(10000))
		}

		result = append(result, holding)
	}

	return result
}
