// Package repository contains data persistence implementations.
package repository

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/shopspring/decimal"
)

// MemoryFundRepository is an in-memory implementation of FundRepository.
// In production, this would be replaced with a database implementation.
type MemoryFundRepository struct {
	mu       sync.RWMutex
	funds    map[string]*domain.Fund
	holdings map[string][]domain.StockHolding
	history  map[string][]domain.FundHistory
}

// NewMemoryFundRepository creates a new in-memory fund repository with sample data.
func NewMemoryFundRepository() *MemoryFundRepository {
	repo := &MemoryFundRepository{
		funds:    make(map[string]*domain.Fund),
		holdings: make(map[string][]domain.StockHolding),
		history:  make(map[string][]domain.FundHistory),
	}
	repo.seedSampleData()
	return repo
}

// seedSampleData initializes the repository with sample fund data.
func (r *MemoryFundRepository) seedSampleData() {
	// Sample Fund 1: 易方达蓝筹精选混合
	r.funds["005827"] = &domain.Fund{
		ID:          "005827",
		Name:        "易方达蓝筹精选混合",
		Type:        "hybrid",
		Manager:     "张坤",
		Company:     "易方达基金",
		NetAssetVal: decimal.NewFromFloat(1.9856),
		TotalScale:  decimal.NewFromFloat(567.89),
	}
	r.holdings["005827"] = []domain.StockHolding{
		{StockCode: "600519", StockName: "贵州茅台", Exchange: domain.ExchangeSH, HoldingRatio: decimal.NewFromFloat(9.85)},
		{StockCode: "000858", StockName: "五粮液", Exchange: domain.ExchangeSZ, HoldingRatio: decimal.NewFromFloat(7.23)},
		{StockCode: "000568", StockName: "泸州老窖", Exchange: domain.ExchangeSZ, HoldingRatio: decimal.NewFromFloat(6.54)},
		{StockCode: "600036", StockName: "招商银行", Exchange: domain.ExchangeSH, HoldingRatio: decimal.NewFromFloat(5.87)},
		{StockCode: "601318", StockName: "中国平安", Exchange: domain.ExchangeSH, HoldingRatio: decimal.NewFromFloat(5.42)},
		{StockCode: "000333", StockName: "美的集团", Exchange: domain.ExchangeSZ, HoldingRatio: decimal.NewFromFloat(4.89)},
		{StockCode: "600887", StockName: "伊利股份", Exchange: domain.ExchangeSH, HoldingRatio: decimal.NewFromFloat(4.56)},
		{StockCode: "002304", StockName: "洋河股份", Exchange: domain.ExchangeSZ, HoldingRatio: decimal.NewFromFloat(4.21)},
		{StockCode: "600276", StockName: "恒瑞医药", Exchange: domain.ExchangeSH, HoldingRatio: decimal.NewFromFloat(3.98)},
		{StockCode: "000651", StockName: "格力电器", Exchange: domain.ExchangeSZ, HoldingRatio: decimal.NewFromFloat(3.76)},
	}

	// Sample Fund 2: 中欧医疗健康混合
	r.funds["003095"] = &domain.Fund{
		ID:          "003095",
		Name:        "中欧医疗健康混合A",
		Type:        "hybrid",
		Manager:     "葛兰",
		Company:     "中欧基金",
		NetAssetVal: decimal.NewFromFloat(1.5432),
		TotalScale:  decimal.NewFromFloat(456.78),
	}
	r.holdings["003095"] = []domain.StockHolding{
		{StockCode: "603259", StockName: "药明康德", Exchange: domain.ExchangeSH, HoldingRatio: decimal.NewFromFloat(9.42)},
		{StockCode: "300760", StockName: "迈瑞医疗", Exchange: domain.ExchangeSZ, HoldingRatio: decimal.NewFromFloat(8.36)},
		{StockCode: "600276", StockName: "恒瑞医药", Exchange: domain.ExchangeSH, HoldingRatio: decimal.NewFromFloat(7.89)},
		{StockCode: "300122", StockName: "智飞生物", Exchange: domain.ExchangeSZ, HoldingRatio: decimal.NewFromFloat(6.54)},
		{StockCode: "300015", StockName: "爱尔眼科", Exchange: domain.ExchangeSZ, HoldingRatio: decimal.NewFromFloat(5.98)},
		{StockCode: "000661", StockName: "长春高新", Exchange: domain.ExchangeSZ, HoldingRatio: decimal.NewFromFloat(5.43)},
		{StockCode: "002821", StockName: "凯莱英", Exchange: domain.ExchangeSZ, HoldingRatio: decimal.NewFromFloat(4.87)},
		{StockCode: "300529", StockName: "健帆生物", Exchange: domain.ExchangeSZ, HoldingRatio: decimal.NewFromFloat(4.32)},
		{StockCode: "002007", StockName: "华兰生物", Exchange: domain.ExchangeSZ, HoldingRatio: decimal.NewFromFloat(3.76)},
		{StockCode: "300003", StockName: "乐普医疗", Exchange: domain.ExchangeSZ, HoldingRatio: decimal.NewFromFloat(3.21)},
	}

	// Sample Fund 3: 诺安成长混合
	r.funds["320007"] = &domain.Fund{
		ID:          "320007",
		Name:        "诺安成长混合",
		Type:        "hybrid",
		Manager:     "蔡嵩松",
		Company:     "诺安基金",
		NetAssetVal: decimal.NewFromFloat(1.2345),
		TotalScale:  decimal.NewFromFloat(234.56),
	}
	r.holdings["320007"] = []domain.StockHolding{
		{StockCode: "002371", StockName: "北方华创", Exchange: domain.ExchangeSZ, HoldingRatio: decimal.NewFromFloat(9.98)},
		{StockCode: "688041", StockName: "海光信息", Exchange: domain.ExchangeSH, HoldingRatio: decimal.NewFromFloat(9.12)},
		{StockCode: "688256", StockName: "寒武纪", Exchange: domain.ExchangeSH, HoldingRatio: decimal.NewFromFloat(8.76)},
		{StockCode: "603501", StockName: "韦尔股份", Exchange: domain.ExchangeSH, HoldingRatio: decimal.NewFromFloat(7.54)},
		{StockCode: "002049", StockName: "紫光国微", Exchange: domain.ExchangeSZ, HoldingRatio: decimal.NewFromFloat(6.89)},
		{StockCode: "688012", StockName: "中微公司", Exchange: domain.ExchangeSH, HoldingRatio: decimal.NewFromFloat(6.23)},
		{StockCode: "688981", StockName: "中芯国际", Exchange: domain.ExchangeSH, HoldingRatio: decimal.NewFromFloat(5.67)},
		{StockCode: "300661", StockName: "圣邦股份", Exchange: domain.ExchangeSZ, HoldingRatio: decimal.NewFromFloat(4.98)},
		{StockCode: "603986", StockName: "兆易创新", Exchange: domain.ExchangeSH, HoldingRatio: decimal.NewFromFloat(4.32)},
		{StockCode: "688008", StockName: "澜起科技", Exchange: domain.ExchangeSH, HoldingRatio: decimal.NewFromFloat(3.87)},
	}
}

// GetFundByID retrieves a fund by its ID.
func (r *MemoryFundRepository) GetFundByID(ctx context.Context, fundID string) (*domain.Fund, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if fund, ok := r.funds[fundID]; ok {
		return fund, nil
	}
	return nil, nil
}

// GetFundsByIDs retrieves multiple funds keyed by fund ID.
func (r *MemoryFundRepository) GetFundsByIDs(ctx context.Context, fundIDs []string) (map[string]*domain.Fund, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*domain.Fund)
	for _, fundID := range fundIDs {
		if fund, ok := r.funds[fundID]; ok {
			copyFund := *fund
			result[fundID] = &copyFund
		}
	}
	return result, nil
}

// SearchFunds searches for funds by name or code.
func (r *MemoryFundRepository) SearchFunds(ctx context.Context, query string, limit int) ([]*domain.Fund, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query = strings.ToLower(query)
	var results []*domain.Fund

	for _, fund := range r.funds {
		if strings.Contains(strings.ToLower(fund.ID), query) ||
			strings.Contains(strings.ToLower(fund.Name), query) ||
			strings.Contains(strings.ToLower(fund.Manager), query) {
			results = append(results, fund)
			if len(results) >= limit {
				break
			}
		}
	}

	return results, nil
}

// GetFundHoldings retrieves the top holdings for a fund.
func (r *MemoryFundRepository) GetFundHoldings(ctx context.Context, fundID string) ([]domain.StockHolding, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if holdings, ok := r.holdings[fundID]; ok {
		return holdings, nil
	}
	return nil, nil
}

// SaveFund saves or updates a fund.
func (r *MemoryFundRepository) SaveFund(ctx context.Context, fund *domain.Fund) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.funds[fund.ID] = fund
	return nil
}

// SaveHoldings saves the holdings for a fund.
func (r *MemoryFundRepository) SaveHoldings(ctx context.Context, fundID string, holdings []domain.StockHolding) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.holdings[fundID] = holdings
	return nil
}

// SaveTimeSeriesPoint saves a time series data point (in-memory, not persisted).
func (r *MemoryFundRepository) SaveTimeSeriesPoint(ctx context.Context, point *domain.TimeSeriesPoint, fundID string) error {
	// In-memory repository doesn't persist time series to database
	// Time series is handled by ValuationService's in-memory storage
	return nil
}

// ReplaceTimeSeriesByDate is a no-op for the in-memory repository.
func (r *MemoryFundRepository) ReplaceTimeSeriesByDate(ctx context.Context, fundID string, date time.Time, points []domain.TimeSeriesPoint) error {
	return nil
}

// GetTimeSeriesByDate retrieves time series data for a fund on a specific date.
func (r *MemoryFundRepository) GetTimeSeriesByDate(ctx context.Context, fundID string, date time.Time) ([]domain.TimeSeriesPoint, error) {
	// In-memory repository returns empty - time series is managed by ValuationService
	return []domain.TimeSeriesPoint{}, nil
}

// SaveFundHistory saves a daily official NAV snapshot in memory.
func (r *MemoryFundRepository) SaveFundHistory(ctx context.Context, history *domain.FundHistory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	records := r.history[history.FundID]
	replaced := false
	for i := range records {
		if records[i].Date == history.Date {
			records[i] = *history
			replaced = true
			break
		}
	}
	if !replaced {
		records = append(records, *history)
	}
	r.history[history.FundID] = records
	return nil
}

// GetLatestFundHistory retrieves the latest official NAV snapshot from memory.
func (r *MemoryFundRepository) GetLatestFundHistory(ctx context.Context, fundID string) (*domain.FundHistory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	records := r.history[fundID]
	if len(records) == 0 {
		return nil, nil
	}

	latest := records[0]
	for _, record := range records[1:] {
		if record.Date > latest.Date {
			latest = record
		}
	}
	copyRecord := latest
	return &copyRecord, nil
}

// GetLatestFundHistoriesByFundIDs retrieves the latest official NAV snapshots from memory.
func (r *MemoryFundRepository) GetLatestFundHistoriesByFundIDs(ctx context.Context, fundIDs []string) (map[string]*domain.FundHistory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*domain.FundHistory)
	for _, fundID := range fundIDs {
		records := r.history[fundID]
		if len(records) == 0 {
			continue
		}

		latest := records[0]
		for _, record := range records[1:] {
			if record.Date > latest.Date {
				latest = record
			}
		}

		copyRecord := latest
		result[fundID] = &copyRecord
	}
	return result, nil
}
