// Package adapter contains implementations for external data sources.
package adapter

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/go-resty/resty/v2"
	"github.com/shopspring/decimal"
)

// TencentQuoteProvider implements QuoteProvider using the qt.gtimg.cn quote endpoint.
type TencentQuoteProvider struct {
	client  *resty.Client
	baseURL string
}

// NewTencentQuoteProvider creates a new Tencent quote provider.
func NewTencentQuoteProvider() *TencentQuoteProvider {
	client := resty.New().
		SetTimeout(10 * time.Second).
		SetRetryCount(3).
		SetRetryWaitTime(500 * time.Millisecond)

	return &TencentQuoteProvider{
		client:  client,
		baseURL: "https://qt.gtimg.cn",
	}
}

// GetName returns the provider name.
func (t *TencentQuoteProvider) GetName() string {
	return "tencent"
}

// GetRealTimeQuotes fetches real-time quote snapshots for the given stock codes.
func (t *TencentQuoteProvider) GetRealTimeQuotes(ctx context.Context, stockCodes []string) (map[string]domain.StockQuote, error) {
	if len(stockCodes) == 0 {
		return map[string]domain.StockQuote{}, nil
	}

	symbols := make([]string, 0, len(stockCodes))
	for _, code := range stockCodes {
		symbols = append(symbols, buildTencentSymbol(code))
	}

	resp, err := t.client.R().
		SetContext(ctx).
		SetHeader("Referer", "https://gu.qq.com/").
		SetQueryParam("q", strings.Join(symbols, ",")).
		Get(t.baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quotes from Tencent: %w", err)
	}
	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("tencent quote API returned status %d", resp.StatusCode())
	}

	body, err := decodeGBK(resp.Body())
	if err != nil {
		return nil, fmt.Errorf("failed to decode Tencent quote response: %w", err)
	}

	return t.parseResponse(string(body))
}

func buildTencentSymbol(stockCode string) string {
	if len(stockCode) < 1 {
		return "sz" + stockCode
	}
	if isUSQuoteCode(stockCode) {
		return "us" + strings.ToUpper(stockCode)
	}
	if len(stockCode) == 5 {
		return "hk" + stockCode
	}

	firstChar := stockCode[0]
	switch {
	case firstChar == '5' || firstChar == '6' || firstChar == '9':
		return "sh" + stockCode
	case firstChar == '4' || firstChar == '8':
		return "bj" + stockCode
	default:
		return "sz" + stockCode
	}
}

func (t *TencentQuoteProvider) parseResponse(body string) (map[string]domain.StockQuote, error) {
	result := make(map[string]domain.StockQuote)

	re := regexp.MustCompile(`v_([a-z]{2}[A-Za-z0-9\._]+)="([^"]*)";`)
	matches := re.FindAllStringSubmatch(body, -1)

	for _, match := range matches {
		if len(match) != 3 {
			continue
		}

		fields := strings.Split(match[2], "~")
		if len(fields) < 6 {
			continue
		}

		symbol := match[1]
		quote, err := t.parseQuote(symbol, fields)
		if err != nil {
			continue
		}
		result[quote.StockCode] = quote
	}

	return result, nil
}

func (t *TencentQuoteProvider) parseQuote(symbol string, fields []string) (domain.StockQuote, error) {
	if strings.HasPrefix(symbol, "us") {
		return t.parseUSQuote(fields)
	}

	quote := domain.StockQuote{
		StockName: strings.TrimSpace(fields[1]),
		StockCode: strings.TrimSpace(fields[2]),
		UpdatedAt: time.Now(),
	}

	var err error
	quote.CurrentPrice, err = parseDecimal(fields[3])
	if err != nil {
		return quote, err
	}
	quote.PrevClose, err = parseDecimal(fields[4])
	if err != nil {
		return quote, err
	}
	quote.OpenPrice, err = parseDecimal(fields[5])
	if err != nil {
		return quote, err
	}

	if len(fields) > 33 {
		quote.HighPrice, _ = parseDecimal(fields[33])
	}
	if len(fields) > 34 {
		quote.LowPrice, _ = parseDecimal(fields[34])
	}
	if len(fields) > 6 {
		quote.Volume, _ = parseDecimal(fields[6])
	}
	if len(fields) > 37 {
		quote.Turnover, _ = parseDecimal(fields[37])
	}

	quote.CurrentPrice = firstNonZeroDecimal(quote.CurrentPrice, quote.OpenPrice, quote.PrevClose)
	if quote.HighPrice.IsZero() {
		quote.HighPrice = quote.CurrentPrice
	}
	if quote.LowPrice.IsZero() {
		quote.LowPrice = quote.CurrentPrice
	}

	if !quote.PrevClose.IsZero() && !quote.CurrentPrice.IsZero() {
		quote.ChangeAmount = quote.CurrentPrice.Sub(quote.PrevClose)
		quote.ChangePercent = quote.ChangeAmount.Div(quote.PrevClose).Mul(decimal.NewFromInt(100)).Round(4)
	}

	return quote, nil
}

func (t *TencentQuoteProvider) parseUSQuote(fields []string) (domain.StockQuote, error) {
	stockCode := strings.TrimSpace(fields[2])
	if idx := strings.Index(stockCode, "."); idx > 0 {
		stockCode = stockCode[:idx]
	}

	quote := domain.StockQuote{
		StockName: strings.TrimSpace(fields[1]),
		StockCode: strings.ToUpper(stockCode),
		UpdatedAt: time.Now(),
	}

	var err error
	quote.CurrentPrice, err = parseDecimal(fields[3])
	if err != nil {
		return quote, err
	}
	quote.PrevClose, err = parseDecimal(fields[4])
	if err != nil {
		return quote, err
	}
	quote.OpenPrice, err = parseDecimal(fields[5])
	if err != nil {
		return quote, err
	}

	if len(fields) > 33 {
		quote.HighPrice, _ = parseDecimal(fields[33])
	}
	if len(fields) > 34 {
		quote.LowPrice, _ = parseDecimal(fields[34])
	}
	if len(fields) > 36 {
		quote.Volume, _ = parseDecimal(fields[36])
	}
	if len(fields) > 37 {
		quote.Turnover, _ = parseDecimal(fields[37])
	}
	if len(fields) > 31 {
		quote.ChangeAmount, _ = parseDecimal(fields[31])
	}
	if len(fields) > 32 {
		quote.ChangePercent, _ = parseDecimal(fields[32])
	}

	quote.CurrentPrice = firstNonZeroDecimal(quote.CurrentPrice, quote.OpenPrice, quote.PrevClose)
	if quote.HighPrice.IsZero() {
		quote.HighPrice = quote.CurrentPrice
	}
	if quote.LowPrice.IsZero() {
		quote.LowPrice = quote.CurrentPrice
	}
	if quote.ChangeAmount.IsZero() && !quote.PrevClose.IsZero() {
		quote.ChangeAmount = quote.CurrentPrice.Sub(quote.PrevClose)
	}
	if quote.ChangePercent.IsZero() && !quote.PrevClose.IsZero() {
		quote.ChangePercent = quote.ChangeAmount.Div(quote.PrevClose).Mul(decimal.NewFromInt(100)).Round(4)
	}

	return quote, nil
}
