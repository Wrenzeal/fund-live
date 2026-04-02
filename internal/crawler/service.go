package crawler

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/go-resty/resty/v2"
	"github.com/shopspring/decimal"
)

// CrawlService provides fund data crawling capabilities.
type CrawlService struct {
	client         *resty.Client
	apiParser      *EastmoneyAPIParser
	holdingsParser *EastmoneyHoldingsParser
	maxConcurrency int
	requestDelay   time.Duration
	debug          bool
}

// CrawlResult holds the result of crawling a single fund.
type CrawlResult struct {
	Fund     *domain.Fund
	Holdings []domain.StockHolding
	Error    error
}

// NewCrawlService creates a new crawl service.
func NewCrawlService(maxConcurrency int) *CrawlService {
	client := resty.New().
		SetTimeout(60*time.Second). // Increased from 30s for slow responses
		SetRetryCount(3).
		SetRetryWaitTime(2*time.Second).     // Increased from 1s
		SetRetryMaxWaitTime(10*time.Second). // Max wait between retries
		SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36").
		SetHeader("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8").
		SetHeader("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8").
		SetHeader("Referer", "http://fund.eastmoney.com/")

	return &CrawlService{
		client:         client,
		apiParser:      NewEastmoneyAPIParser(),
		holdingsParser: NewEastmoneyHoldingsParser(),
		maxConcurrency: maxConcurrency,
		requestDelay:   500 * time.Millisecond,
		debug:          os.Getenv("DEBUG") == "1",
	}
}

// SetDebug enables or disables debug logging.
func (s *CrawlService) SetDebug(enabled bool) {
	s.debug = enabled
}

// FetchFundData fetches fund information and holdings for a single fund.
func (s *CrawlService) FetchFundData(ctx context.Context, fundCode string) (*domain.Fund, []domain.StockHolding, error) {
	var fund *domain.Fund
	var holdings []domain.StockHolding
	var fundErr, holdingsErr error

	var wg sync.WaitGroup
	wg.Add(2)

	// Fetch fund info
	go func() {
		defer wg.Done()
		fund, fundErr = s.fetchFundInfo(ctx, fundCode)
	}()

	// Fetch holdings
	go func() {
		defer wg.Done()
		holdings, holdingsErr = s.fetchHoldings(ctx, fundCode)
	}()

	wg.Wait()

	if fundErr != nil {
		return nil, nil, fmt.Errorf("fetch fund info failed: %w", fundErr)
	}
	if holdingsErr != nil {
		// Holdings failure is not critical, return fund with empty holdings
		if s.debug {
			log.Printf("[DEBUG] Holdings fetch failed for %s: %v", fundCode, holdingsErr)
		}
		return fund, nil, nil
	}

	return fund, holdings, nil
}

// FetchLatestFundHistory fetches the latest official NAV snapshot for a fund.
func (s *CrawlService) FetchLatestFundHistory(ctx context.Context, fundCode string) (*domain.FundHistory, error) {
	url := fmt.Sprintf("http://fund.eastmoney.com/pingzhongdata/%s.js", fundCode)

	resp, err := s.client.R().
		SetContext(ctx).
		Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}

	detail, err := s.apiParser.ParsePingzhongJS(string(resp.Body()), fundCode)
	if err != nil {
		return nil, fmt.Errorf("parse JS failed: %w", err)
	}

	if detail.NAV.IsZero() || detail.NAVDate == "" {
		return nil, fmt.Errorf("latest NAV not found for fund %s", fundCode)
	}

	dailyReturn := decimal.Zero
	if !detail.PreviousNAV.IsZero() {
		dailyReturn = detail.NAV.Sub(detail.PreviousNAV).
			Div(detail.PreviousNAV).
			Mul(decimal.NewFromInt(100)).
			Round(4)
	}

	return &domain.FundHistory{
		FundID:      fundCode,
		Date:        detail.NAVDate,
		NetAssetVal: detail.NAV,
		AccumVal:    detail.AccumNAV,
		DailyReturn: dailyReturn,
		CreatedAt:   time.Now(),
	}, nil
}

// fetchFundInfo fetches fund information from Eastmoney pingzhongdata JS.
func (s *CrawlService) fetchFundInfo(ctx context.Context, fundCode string) (*domain.Fund, error) {
	url := fmt.Sprintf("http://fund.eastmoney.com/pingzhongdata/%s.js", fundCode)

	resp, err := s.client.R().
		SetContext(ctx).
		Get(url)

	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}

	content := string(resp.Body())

	if s.debug {
		log.Printf("[DEBUG] Fund JS response length: %d bytes", len(content))
	}

	// Parse using the API parser
	detail, err := s.apiParser.ParsePingzhongJS(content, fundCode)
	if err != nil {
		return nil, fmt.Errorf("parse JS failed: %w", err)
	}

	fund := s.apiParser.ToFund(detail)
	return fund, nil
}

// fetchHoldings fetches top 10 holdings from Eastmoney.
func (s *CrawlService) fetchHoldings(ctx context.Context, fundCode string) ([]domain.StockHolding, error) {
	url := fmt.Sprintf("http://fundf10.eastmoney.com/FundArchivesDatas.aspx?type=jjcc&code=%s&topline=10", fundCode)

	resp, err := s.client.R().
		SetContext(ctx).
		Get(url)

	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}

	body := resp.Body()

	// Try GBK to UTF-8 conversion
	utf8Body, convErr := GBKToUTF8(body)
	if convErr != nil {
		utf8Body = body
	}

	content := string(utf8Body)

	if s.debug {
		log.Printf("[DEBUG] Holdings response length: %d bytes", len(content))
		// Save raw response for debugging
		if len(content) < 5000 {
			log.Printf("[DEBUG] Holdings content: %s", content[:min(500, len(content))])
		}
	}

	// Parse the HTML content
	rawHoldings, reportPeriod, err := s.holdingsParser.ParseHoldingsHTML(content)
	if err != nil {
		return nil, fmt.Errorf("parse HTML failed: %w", err)
	}

	if s.debug {
		log.Printf("[DEBUG] Parsed %d raw holdings, report period: %s", len(rawHoldings), reportPeriod)
	}

	// Convert to domain holdings
	holdings := s.holdingsParser.ToStockHoldings(rawHoldings)

	return holdings, nil
}

// BatchFetchFundData fetches data for multiple funds with concurrency control.
func (s *CrawlService) BatchFetchFundData(ctx context.Context, fundCodes []string) map[string]*CrawlResult {
	results := make(map[string]*CrawlResult)
	var mu sync.Mutex

	sem := make(chan struct{}, s.maxConcurrency)
	var wg sync.WaitGroup

	for _, code := range fundCodes {
		wg.Add(1)
		go func(fundCode string) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			select {
			case <-ctx.Done():
				mu.Lock()
				results[fundCode] = &CrawlResult{Error: ctx.Err()}
				mu.Unlock()
				return
			default:
			}

			fund, holdings, err := s.FetchFundData(ctx, fundCode)

			mu.Lock()
			results[fundCode] = &CrawlResult{
				Fund:     fund,
				Holdings: holdings,
				Error:    err,
			}
			mu.Unlock()

			time.Sleep(s.requestDelay)
		}(code)
	}

	wg.Wait()
	return results
}

// PrintResults prints the crawl results for debugging.
// For large result sets (>50), only prints a summary.
func (s *CrawlService) PrintResults(results map[string]*CrawlResult) {
	// For large result sets, only print summary
	if len(results) > 50 {
		success, failed := 0, 0
		for _, result := range results {
			if result.Error == nil && result.Fund != nil {
				success++
			} else {
				failed++
			}
		}
		log.Printf("📊 Results summary: %d success, %d failed (total: %d)", success, failed, len(results))
		return
	}

	// For smaller sets, print details
	for code, result := range results {
		if result.Error != nil {
			log.Printf("❌ [%s] Error: %v", code, result.Error)
			continue
		}

		if result.Fund == nil {
			log.Printf("❌ [%s] Fund is nil", code)
			continue
		}

		log.Printf("✅ [%s] %s", code, result.Fund.Name)
		log.Printf("   基金经理: %s", result.Fund.Manager)
		log.Printf("   基金公司: %s", result.Fund.Company)
		log.Printf("   最新净值: %s", result.Fund.NetAssetVal.String())
		log.Printf("   基金规模: %s 亿元", result.Fund.TotalScale.String())
		log.Printf("   前十大重仓股 (%d只):", len(result.Holdings))

		for i, h := range result.Holdings {
			if i >= 10 {
				break
			}
			log.Printf("      %d. %s (%s) - 占比 %.2f%%",
				i+1, h.StockName, h.StockCode, h.HoldingRatio.InexactFloat64())
		}
		log.Println()
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
