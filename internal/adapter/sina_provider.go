// Package adapter contains implementations for external data sources.
package adapter

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/go-resty/resty/v2"
	"github.com/shopspring/decimal"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// SinaFinanceProvider implements the QuoteProvider interface using Sina Finance API.
type SinaFinanceProvider struct {
	client  *resty.Client
	baseURL string
}

// NewSinaFinanceProvider creates a new SinaFinanceProvider instance.
func NewSinaFinanceProvider() *SinaFinanceProvider {
	client := resty.New().
		SetTimeout(10 * time.Second).
		SetRetryCount(3).
		SetRetryWaitTime(500 * time.Millisecond)

	return &SinaFinanceProvider{
		client:  client,
		baseURL: "https://hq.sinajs.cn",
	}
}

// GetName returns the provider name.
func (s *SinaFinanceProvider) GetName() string {
	return "SinaFinance"
}

// buildSinaSymbol converts stock code to Sina format.
// Shanghai stocks: sh600519 (6开头, 68科创板)
// Shenzhen stocks: sz000858 (0开头, 3创业板)
// Beijing stocks: bj920982 (4, 8, 9开头的北交所/新三板)
func buildSinaSymbol(stockCode string) string {
	if len(stockCode) < 1 {
		return "sz" + stockCode
	}

	firstChar := stockCode[0]
	switch {
	case firstChar == '6': // Shanghai: 60xxxx, 68xxxx (科创板)
		return "sh" + stockCode
	case firstChar == '4' || firstChar == '8' || firstChar == '9':
		// Beijing Stock Exchange / NEEQ (新三板/北交所)
		// 43xxxx, 83xxxx, 87xxxx (新三板)
		// 82xxxx, 92xxxx (北交所)
		return "bj" + stockCode
	default: // Shenzhen: 00xxxx, 30xxxx (创业板)
		return "sz" + stockCode
	}
}

// GetRealTimeQuotes fetches detailed quote information for the given stock codes.
// Sina API returns data in format:
// var hq_str_sh600519="贵州茅台,1950.00,1940.00,1955.00,1960.00,1945.00,1954.00,1955.00,10000000,19500000000,..."
func (s *SinaFinanceProvider) GetRealTimeQuotes(ctx context.Context, stockCodes []string) (map[string]domain.StockQuote, error) {
	if len(stockCodes) == 0 {
		return make(map[string]domain.StockQuote), nil
	}

	// Build symbol list
	symbols := make([]string, len(stockCodes))
	for i, code := range stockCodes {
		symbols[i] = buildSinaSymbol(code)
	}
	symbolList := strings.Join(symbols, ",")

	// Make HTTP request
	resp, err := s.client.R().
		SetContext(ctx).
		SetHeader("Referer", "https://finance.sina.com.cn").
		SetQueryParam("list", symbolList).
		Get(s.baseURL + "/list=" + symbolList)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch quotes from Sina: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("sina API returned status %d", resp.StatusCode())
	}

	// Decode GBK to UTF-8
	body, err := decodeGBK(resp.Body())
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return s.parseResponse(string(body), stockCodes)
}

// decodeGBK converts GBK encoded bytes to UTF-8.
func decodeGBK(data []byte) ([]byte, error) {
	reader := transform.NewReader(strings.NewReader(string(data)), simplifiedchinese.GBK.NewDecoder())
	result, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// parseResponse parses the Sina API response.
// Format: var hq_str_sh600519="name,open,prevClose,current,high,low,bid,ask,volume,amount,...";
// Fields: 名称,今开,昨收,现价,最高,最低,买一价,卖一价,成交量,成交额,...
// Note: Beijing (bj) stocks may have different field order
func (s *SinaFinanceProvider) parseResponse(body string, stockCodes []string) (map[string]domain.StockQuote, error) {
	result := make(map[string]domain.StockQuote)

	// Regex to extract: var hq_str_<symbol>="<data>";
	// Support sh, sz, bj prefixes
	re := regexp.MustCompile(`var hq_str_([a-z]{2}\d+)="([^"]*)";`)
	matches := re.FindAllStringSubmatch(body, -1)

	for _, match := range matches {
		if len(match) != 3 {
			continue
		}

		symbol := match[1]
		data := match[2]

		// Skip suspended stocks (empty data)
		if data == "" {
			continue
		}

		fields := strings.Split(data, ",")

		// Determine the exchange and minimum field count
		exchange := symbol[:2]
		minFields := 32 // sh, sz stocks

		if exchange == "bj" {
			// Beijing stocks may have different format
			// Some only have ~20 fields
			minFields = 6 // At least: name, open, prev_close, current, high, low
		}

		if len(fields) < minFields {
			continue
		}

		// Extract stock code (remove exchange prefix)
		stockCode := symbol[2:]

		quote, err := s.parseQuoteByExchange(stockCode, fields, exchange)
		if err != nil {
			continue // Skip malformed data
		}

		result[stockCode] = quote
	}

	return result, nil
}

// parseQuoteByExchange parses quote data based on exchange type.
// Different exchanges may have different field layouts.
func (s *SinaFinanceProvider) parseQuoteByExchange(stockCode string, fields []string, exchange string) (domain.StockQuote, error) {
	// Beijing stocks may have different field format
	if exchange == "bj" {
		return s.parseBJQuote(stockCode, fields)
	}
	// Default: sh, sz stocks
	return s.parseQuote(stockCode, fields)
}

// parseBJQuote parses Beijing Stock Exchange quote data.
// BJE stocks have a different field layout from SH/SZ.
// Format may vary, but commonly: name,open,prevClose,current,high,low,...
func (s *SinaFinanceProvider) parseBJQuote(stockCode string, fields []string) (domain.StockQuote, error) {
	quote := domain.StockQuote{
		StockCode: stockCode,
		StockName: strings.TrimSpace(fields[0]),
		UpdatedAt: time.Now(),
	}

	// BJE format is similar to SH/SZ for the main fields
	// Try to parse with same indices first
	if len(fields) >= 6 {
		quote.OpenPrice, _ = parseDecimal(fields[1])
		quote.PrevClose, _ = parseDecimal(fields[2])
		quote.CurrentPrice, _ = parseDecimal(fields[3])
		quote.HighPrice, _ = parseDecimal(fields[4])
		quote.LowPrice, _ = parseDecimal(fields[5])
	}

	// Try to get volume and turnover if available
	if len(fields) >= 10 {
		quote.Volume, _ = parseDecimal(fields[8])
		quote.Turnover, _ = parseDecimal(fields[9])
	}

	quote.CurrentPrice = firstNonZeroDecimal(quote.CurrentPrice, quote.OpenPrice, quote.PrevClose)
	if quote.HighPrice.IsZero() {
		quote.HighPrice = quote.CurrentPrice
	}
	if quote.LowPrice.IsZero() {
		quote.LowPrice = quote.CurrentPrice
	}

	// Calculate change percent and amount
	if !quote.PrevClose.IsZero() && !quote.CurrentPrice.IsZero() {
		quote.ChangeAmount = quote.CurrentPrice.Sub(quote.PrevClose)
		quote.ChangePercent = quote.ChangeAmount.Div(quote.PrevClose).Mul(decimal.NewFromInt(100)).Round(4)
	}

	return quote, nil
}

// parseQuote parses a single stock's quote data (SH/SZ exchanges).
func (s *SinaFinanceProvider) parseQuote(stockCode string, fields []string) (domain.StockQuote, error) {
	// Field indices (0-indexed)
	// 0: 股票名称
	// 1: 今日开盘价
	// 2: 昨日收盘价
	// 3: 当前价格
	// 4: 今日最高价
	// 5: 今日最低价
	// 8: 成交量(股)
	// 9: 成交额(元)
	// 30: 日期
	// 31: 时间

	quote := domain.StockQuote{
		StockCode: stockCode,
		StockName: strings.TrimSpace(fields[0]),
		UpdatedAt: time.Now(),
	}

	var err error

	// Parse prices
	quote.OpenPrice, err = parseDecimal(fields[1])
	if err != nil {
		return quote, err
	}

	quote.PrevClose, err = parseDecimal(fields[2])
	if err != nil {
		return quote, err
	}

	quote.CurrentPrice, err = parseDecimal(fields[3])
	if err != nil {
		return quote, err
	}

	quote.HighPrice, err = parseDecimal(fields[4])
	if err != nil {
		return quote, err
	}

	quote.LowPrice, err = parseDecimal(fields[5])
	if err != nil {
		return quote, err
	}

	quote.Volume, err = parseDecimal(fields[8])
	if err != nil {
		return quote, err
	}

	quote.Turnover, err = parseDecimal(fields[9])
	if err != nil {
		return quote, err
	}

	bidPrice, err := parseDecimal(fields[6])
	if err != nil {
		return quote, err
	}

	askPrice, err := parseDecimal(fields[7])
	if err != nil {
		return quote, err
	}

	quote.CurrentPrice = firstNonZeroDecimal(quote.CurrentPrice, bidPrice, askPrice, quote.OpenPrice, quote.PrevClose)
	if quote.HighPrice.IsZero() {
		quote.HighPrice = quote.CurrentPrice
	}
	if quote.LowPrice.IsZero() {
		quote.LowPrice = quote.CurrentPrice
	}

	// Calculate change percent and amount
	if !quote.PrevClose.IsZero() {
		quote.ChangeAmount = quote.CurrentPrice.Sub(quote.PrevClose)
		quote.ChangePercent = quote.ChangeAmount.Div(quote.PrevClose).Mul(decimal.NewFromInt(100)).Round(4)
	}

	return quote, nil
}

// parseDecimal safely parses a string to decimal.
func parseDecimal(s string) (decimal.Decimal, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" {
		return decimal.Zero, nil
	}
	return decimal.NewFromString(s)
}

func firstNonZeroDecimal(values ...decimal.Decimal) decimal.Decimal {
	for _, value := range values {
		if !value.IsZero() {
			return value
		}
	}
	return decimal.Zero
}
