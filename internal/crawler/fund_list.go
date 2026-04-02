// Package crawler provides fund data crawling capabilities.
package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/go-resty/resty/v2"
)

// FundListItem represents a fund in the fund list.
type FundListItem struct {
	Code     string `json:"code"`     // 基金代码
	Name     string `json:"name"`     // 基金名称
	Type     string `json:"type"`     // 基金类型
	FullName string `json:"fullName"` // 全称
}

// FundListCrawler fetches the complete list of funds from Eastmoney.
type FundListCrawler struct {
	client *resty.Client
	debug  bool
}

// NewFundListCrawler creates a new fund list crawler.
func NewFundListCrawler() *FundListCrawler {
	client := resty.New().
		SetTimeout(60000000000). // 60 seconds
		SetRetryCount(3).
		SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36").
		SetHeader("Referer", "http://fund.eastmoney.com/")

	return &FundListCrawler{
		client: client,
		debug:  false,
	}
}

// SetDebug enables debug logging.
func (c *FundListCrawler) SetDebug(enabled bool) {
	c.debug = enabled
}

// FetchAllFunds fetches the complete list of all funds from Eastmoney.
// This returns thousands of funds from the market.
func (c *FundListCrawler) FetchAllFunds(ctx context.Context) ([]FundListItem, error) {
	// Eastmoney fund list API - returns all funds
	// Format: var r = [["000001","HXCZHH","华夏成长混合","混合型-偏股","HUAXIACHENGZHANGHUNHE"],...]
	url := "http://fund.eastmoney.com/js/fundcode_search.js"

	resp, err := c.client.R().
		SetContext(ctx).
		Get(url)

	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}

	content := string(resp.Body())

	if c.debug {
		log.Printf("[DEBUG] Fund list response: %d bytes", len(content))
	}

	// Parse the JavaScript array
	funds, err := c.parseFundList(content)
	if err != nil {
		return nil, fmt.Errorf("parse fund list failed: %w", err)
	}

	if c.debug {
		log.Printf("[DEBUG] Parsed %d funds from list", len(funds))
	}

	return funds, nil
}

// FetchStockFunds fetches only stock-type funds (股票型 + 混合型).
// These are the most relevant for real-time valuation.
func (c *FundListCrawler) FetchStockFunds(ctx context.Context) ([]FundListItem, error) {
	allFunds, err := c.FetchAllFunds(ctx)
	if err != nil {
		return nil, err
	}

	// Filter for stock and hybrid funds
	var stockFunds []FundListItem
	for _, fund := range allFunds {
		if strings.Contains(fund.Type, "股票") || strings.Contains(fund.Type, "混合") {
			stockFunds = append(stockFunds, fund)
		}
	}

	if c.debug {
		log.Printf("[DEBUG] Filtered to %d stock/hybrid funds", len(stockFunds))
	}

	return stockFunds, nil
}

// FetchPopularFunds fetches a curated list of popular funds.
// This is a smaller subset suitable for initial testing.
func (c *FundListCrawler) FetchPopularFunds(ctx context.Context) ([]FundListItem, error) {
	// Popular fund codes (manually curated)
	popularCodes := map[string]bool{
		"005827": true, // 易方达蓝筹精选混合
		"003095": true, // 中欧医疗健康混合
		"320007": true, // 诺安成长混合
		"161725": true, // 招商中证白酒指数
		"110011": true, // 易方达中小盘混合
		"000968": true, // 广发聚富混合
		"519736": true, // 交银成长混合
		"260108": true, // 景顺长城新兴成长混合
		"001938": true, // 中欧时代先锋股票
		"001156": true, // 景顺长城鼎益混合
		"110022": true, // 易方达消费行业股票
		"519674": true, // 银河创新成长混合
		"161005": true, // 富国天惠成长混合
		"001714": true, // 工银前沿医疗股票
		"000991": true, // 工银战略转型股票
		"001875": true, // 前海开源国家比较优势混合
		"001102": true, // 前海开源股息率100强股票
		"110003": true, // 易方达上证50指数
		"001644": true, // 汇添富文体娱乐混合
		"000961": true, // 天弘沪深300ETF联接
	}

	allFunds, err := c.FetchAllFunds(ctx)
	if err != nil {
		return nil, err
	}

	var popularFunds []FundListItem
	for _, fund := range allFunds {
		if popularCodes[fund.Code] {
			popularFunds = append(popularFunds, fund)
		}
	}

	return popularFunds, nil
}

// parseFundList parses the JavaScript fund list response.
// Format: var r = [["000001","HXCZHH","华夏成长混合","混合型-偏股","HUAXIACHENGZHANGHUNHE"],...]
func (c *FundListCrawler) parseFundList(content string) ([]FundListItem, error) {
	// Extract the JSON array from JavaScript
	re := regexp.MustCompile(`var\s+r\s*=\s*(\[.+\])`)
	matches := re.FindStringSubmatch(content)
	if len(matches) < 2 {
		return nil, fmt.Errorf("cannot find fund array in response")
	}

	jsonStr := matches[1]

	// Parse as array of arrays
	var rawData [][]string
	if err := json.Unmarshal([]byte(jsonStr), &rawData); err != nil {
		return nil, fmt.Errorf("JSON unmarshal failed: %w", err)
	}

	funds := make([]FundListItem, 0, len(rawData))
	for _, item := range rawData {
		if len(item) >= 4 {
			funds = append(funds, FundListItem{
				Code:     item[0],
				Name:     item[2],
				Type:     item[3],
				FullName: item[2],
			})
		}
	}

	return funds, nil
}
