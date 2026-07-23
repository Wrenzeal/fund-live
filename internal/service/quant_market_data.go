package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/database"
	"github.com/go-resty/resty/v2"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const eastmoneyDailyKLineURL = "https://push2his.eastmoney.com/api/qt/stock/kline/get"

type QuantMarketDataProvider interface {
	FetchDailyBars(ctx context.Context, symbol string, start, end time.Time) ([]database.QuantMarketBar, error)
}

type EastmoneyQuantMarketDataProvider struct {
	client *resty.Client
}

func NewEastmoneyQuantMarketDataProvider() *EastmoneyQuantMarketDataProvider {
	return &EastmoneyQuantMarketDataProvider{client: resty.New().
		SetTimeout(20*time.Second).
		SetRetryCount(2).
		SetRetryWaitTime(500*time.Millisecond).
		SetHeader("User-Agent", "Mozilla/5.0")}
}

type eastmoneyKLineResponse struct {
	Data *struct {
		Code   string   `json:"code"`
		Name   string   `json:"name"`
		KLines []string `json:"klines"`
	} `json:"data"`
}

type parsedQuantKLine struct {
	Date   time.Time
	Open   decimal.Decimal
	Close  decimal.Decimal
	High   decimal.Decimal
	Low    decimal.Decimal
	Volume decimal.Decimal
	Amount decimal.Decimal
}

func (p *EastmoneyQuantMarketDataProvider) FetchDailyBars(ctx context.Context, symbol string, start, end time.Time) ([]database.QuantMarketBar, error) {
	if p == nil || p.client == nil {
		return nil, fmt.Errorf("market data provider is unavailable")
	}
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}
	if start.IsZero() {
		start = time.Now().AddDate(-5, 0, 0)
	}
	if end.IsZero() {
		end = time.Now()
	}
	raw, err := p.fetchKLines(ctx, symbol, start, end, 0)
	if err != nil {
		return nil, err
	}
	adjusted, err := p.fetchKLines(ctx, symbol, start, end, 1)
	if err != nil {
		return nil, err
	}
	adjustedByDate := make(map[string]parsedQuantKLine, len(adjusted))
	for _, item := range adjusted {
		adjustedByDate[item.Date.Format("2006-01-02")] = item
	}
	now := time.Now()
	result := make([]database.QuantMarketBar, 0, len(raw))
	for _, item := range raw {
		adjustedItem, ok := adjustedByDate[item.Date.Format("2006-01-02")]
		if !ok {
			continue
		}
		factor := decimal.NewFromInt(1)
		if !item.Close.IsZero() {
			factor = adjustedItem.Close.Div(item.Close)
		}
		result = append(result, database.QuantMarketBar{
			Symbol:        symbol,
			Date:          item.Date,
			Open:          item.Open,
			High:          item.High,
			Low:           item.Low,
			Close:         item.Close,
			AdjustedClose: adjustedItem.Close,
			Volume:        item.Volume,
			Amount:        item.Amount,
			AdjustFactor:  factor,
			Source:        "eastmoney",
			IngestedAt:    now,
		})
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no daily bars returned for %s", symbol)
	}
	return result, nil
}

func (p *EastmoneyQuantMarketDataProvider) fetchKLines(ctx context.Context, symbol string, start, end time.Time, adjustment int) ([]parsedQuantKLine, error) {
	var response eastmoneyKLineResponse
	resp, err := p.client.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{
			"secid":   eastmoneySecurityID(symbol),
			"fields1": "f1,f2,f3,f4,f5,f6",
			"fields2": "f51,f52,f53,f54,f55,f56,f57,f58,f59,f60,f61",
			"klt":     "101",
			"fqt":     strconv.Itoa(adjustment),
			"beg":     start.Format("20060102"),
			"end":     end.Format("20060102"),
			"lmt":     "100000",
		}).
		SetResult(&response).
		Get(eastmoneyDailyKLineURL)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return nil, fmt.Errorf("eastmoney returned status %d", resp.StatusCode())
	}
	if response.Data == nil {
		var raw eastmoneyKLineResponse
		if unmarshalErr := json.Unmarshal(resp.Body(), &raw); unmarshalErr == nil {
			response = raw
		}
	}
	if response.Data == nil {
		return nil, fmt.Errorf("eastmoney returned no data for %s", symbol)
	}
	result := make([]parsedQuantKLine, 0, len(response.Data.KLines))
	for _, line := range response.Data.KLines {
		parsed, parseErr := parseEastmoneyQuantKLine(line)
		if parseErr == nil {
			result = append(result, parsed)
		}
	}
	return result, nil
}

func eastmoneySecurityID(symbol string) string {
	symbol = strings.TrimSpace(symbol)
	if strings.HasPrefix(symbol, "5") || symbol == "000300" {
		return "1." + symbol
	}
	return "0." + symbol
}

func parseEastmoneyQuantKLine(line string) (parsedQuantKLine, error) {
	parts := strings.Split(strings.TrimSpace(line), ",")
	if len(parts) < 7 {
		return parsedQuantKLine{}, fmt.Errorf("invalid kline field count")
	}
	date, err := time.ParseInLocation("2006-01-02", parts[0], time.FixedZone("CST", 8*60*60))
	if err != nil {
		return parsedQuantKLine{}, err
	}
	values := make([]decimal.Decimal, 6)
	for i := 0; i < 6; i++ {
		value, parseErr := decimal.NewFromString(strings.TrimSpace(parts[i+1]))
		if parseErr != nil {
			return parsedQuantKLine{}, parseErr
		}
		values[i] = value
	}
	return parsedQuantKLine{Date: date, Open: values[0], Close: values[1], High: values[2], Low: values[3], Volume: values[4], Amount: values[5]}, nil
}

func (s *QuantResearchStore) UpsertMarketBars(ctx context.Context, bars []database.QuantMarketBar) error {
	if s == nil || s.db == nil || len(bars) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "symbol"}, {Name: "date"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"open", "high", "low", "close", "adjusted_close", "volume", "amount", "adjust_factor", "suspended", "source", "ingested_at", "updated_at",
		}),
	}).CreateInBatches(bars, 500).Error
}

func (s *QuantResearchStore) RebuildTradingCalendar(ctx context.Context, start, end time.Time) error {
	if s == nil || s.db == nil {
		return nil
	}
	var openDates []time.Time
	if err := s.db.WithContext(ctx).Model(&database.QuantMarketBar{}).
		Where("symbol = ? AND date BETWEEN ? AND ?", "510300", start, end).
		Order("date").Pluck("date", &openDates).Error; err != nil {
		return err
	}
	if len(openDates) == 0 {
		return fmt.Errorf("cannot build CN trading calendar without 510300 observations")
	}
	open := make(map[string]struct{}, len(openDates))
	for _, date := range openDates {
		open[date.Format("2006-01-02")] = struct{}{}
	}
	records := make([]database.QuantTradingDay, 0, int(end.Sub(start).Hours()/24)+1)
	for date := start; !date.After(end); date = date.AddDate(0, 0, 1) {
		_, isOpen := open[date.Format("2006-01-02")]
		reason := ""
		if !isOpen {
			reason = "closed"
		}
		records = append(records, database.QuantTradingDay{Market: "CN", Date: date, IsOpen: isOpen, Reason: reason, Source: "510300_observed"})
	}
	if len(records) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "market"}, {Name: "date"}},
		DoUpdates: clause.AssignmentColumns([]string{"is_open", "reason", "source", "updated_at"}),
	}).CreateInBatches(records, 500).Error
}

func (s *QuantResearchStore) InferAdjustmentActions(ctx context.Context, symbol string) error {
	if s == nil || s.db == nil {
		return nil
	}
	var bars []database.QuantMarketBar
	if err := s.db.WithContext(ctx).Where("symbol = ?", strings.TrimSpace(symbol)).Order("date").Find(&bars).Error; err != nil {
		return err
	}
	actions := make([]database.QuantCorporateAction, 0)
	for i := 1; i < len(bars); i++ {
		if bars[i-1].AdjustFactor.IsZero() || bars[i].AdjustFactor.Equal(bars[i-1].AdjustFactor) {
			continue
		}
		ratio := bars[i].AdjustFactor.Div(bars[i-1].AdjustFactor)
		if ratio.Sub(decimal.NewFromInt(1)).Abs().LessThan(decimal.NewFromFloat(0.000001)) {
			continue
		}
		actions = append(actions, database.QuantCorporateAction{
			Symbol:      symbol,
			ActionDate:  bars[i].Date,
			ActionType:  "adjustment",
			SplitFactor: ratio,
			Source:      "eastmoney_factor_inference",
			KnownAt:     bars[i].Date.Add(24 * time.Hour),
		})
	}
	if len(actions) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "symbol"}, {Name: "action_date"}, {Name: "action_type"}},
		DoUpdates: clause.AssignmentColumns([]string{"split_factor", "source", "known_at"}),
	}).Create(&actions).Error
}

func isRecordNotFound(err error) bool { return err == gorm.ErrRecordNotFound }
