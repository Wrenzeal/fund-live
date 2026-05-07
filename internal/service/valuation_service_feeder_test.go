package service

import (
	"context"
	"testing"
	"time"

	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/repository"
	"github.com/shopspring/decimal"
)

type feederQuoteProvider struct {
	quotes map[string]domain.StockQuote
}

func (p feederQuoteProvider) GetRealTimeQuotes(ctx context.Context, stockCodes []string) (map[string]domain.StockQuote, error) {
	result := make(map[string]domain.StockQuote, len(stockCodes))
	for _, code := range stockCodes {
		if quote, ok := p.quotes[code]; ok {
			result[code] = quote
		}
	}
	return result, nil
}

func (p feederQuoteProvider) GetName() string {
	return "feeder-test"
}

func TestCalculateEstimateForFeederFundPrefersTargetETFQuote(t *testing.T) {
	repo := repository.NewMemoryFundRepository()
	if err := repo.SaveFund(context.Background(), &domain.Fund{
		ID:          "023408",
		Name:        "华宝创业板人工智能ETF发起式联接C",
		Type:        "index",
		NetAssetVal: decimal.RequireFromString("2.2911"),
		UpdatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("SaveFund(feeder) error = %v", err)
	}
	if err := repo.SaveHoldings(context.Background(), "023408", []domain.StockHolding{
		{
			StockCode:    "300502",
			StockName:    "新易盛",
			Exchange:     domain.ExchangeSZ,
			HoldingRatio: decimal.RequireFromString("0.04"),
		},
		{
			StockCode:    "300394",
			StockName:    "天孚通信",
			Exchange:     domain.ExchangeSZ,
			HoldingRatio: decimal.RequireFromString("0.02"),
		},
	}); err != nil {
		t.Fatalf("SaveHoldings(feeder residual) error = %v", err)
	}
	if err := repo.SaveFund(context.Background(), &domain.Fund{
		ID:          "159363",
		Name:        "创业板人工智能ETF华宝",
		Type:        "index",
		NetAssetVal: decimal.RequireFromString("1.0523"),
		UpdatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("SaveFund(target) error = %v", err)
	}
	if err := repo.SaveHoldings(context.Background(), "159363", []domain.StockHolding{
		{
			StockCode:    "300502",
			StockName:    "新易盛",
			Exchange:     domain.ExchangeSZ,
			HoldingRatio: decimal.RequireFromString("15.44"),
		},
		{
			StockCode:    "300394",
			StockName:    "天孚通信",
			Exchange:     domain.ExchangeSZ,
			HoldingRatio: decimal.RequireFromString("6.82"),
		},
	}); err != nil {
		t.Fatalf("SaveHoldings(target) error = %v", err)
	}

	store := &stubFundMappingStore{
		mapping: &database.FundMapping{
			FeederCode: "023408",
			FeederName: "华宝创业板人工智能ETF发起式联接C",
			TargetCode: "159363",
			TargetName: "创业板人工智能ETF华宝",
			IsResolved: true,
		},
	}
	resolver := &FundResolver{
		mappingStore: store,
		fundRepo:     repo,
	}

	provider := feederQuoteProvider{
		quotes: map[string]domain.StockQuote{
			"159363": {
				StockCode:     "159363",
				StockName:     "创业板人工智能ETF华宝",
				CurrentPrice:  decimal.RequireFromString("1.251"),
				PrevClose:     decimal.RequireFromString("1.313"),
				ChangePercent: decimal.RequireFromString("-4.7220"),
				ChangeAmount:  decimal.RequireFromString("-0.062"),
			},
			"300502": {
				StockCode:     "300502",
				StockName:     "新易盛",
				CurrentPrice:  decimal.RequireFromString("530.99"),
				PrevClose:     decimal.RequireFromString("608.28"),
				ChangePercent: decimal.RequireFromString("-12.7063"),
				ChangeAmount:  decimal.RequireFromString("-77.29"),
			},
			"300394": {
				StockCode:     "300394",
				StockName:     "天孚通信",
				CurrentPrice:  decimal.RequireFromString("317.22"),
				PrevClose:     decimal.RequireFromString("350.09"),
				ChangePercent: decimal.RequireFromString("-9.3890"),
				ChangeAmount:  decimal.RequireFromString("-32.87"),
			},
		},
	}

	service := NewValuationService(repo, provider, noopCacheRepository{})
	service.SetFundResolver(resolver)

	estimate, err := service.CalculateEstimate(context.Background(), "023408")
	if err != nil {
		t.Fatalf("CalculateEstimate() error = %v", err)
	}

	if estimate.ChangePercent.String() != "-4.722" {
		t.Fatalf("change percent = %s, want -4.722", estimate.ChangePercent.String())
	}
	if estimate.TotalHoldRatio.String() != "100" {
		t.Fatalf("total hold ratio = %s, want 100", estimate.TotalHoldRatio.String())
	}
	if estimate.DataSource != "追踪目标ETF(sina): 创业板人工智能ETF华宝" {
		t.Fatalf("data source = %q", estimate.DataSource)
	}
	if len(estimate.HoldingDetails) != 1 || estimate.HoldingDetails[0].StockCode != "159363" {
		t.Fatalf("holding details = %+v, want virtual ETF quote detail", estimate.HoldingDetails)
	}
}

func TestCalculateEstimateForFeederFundFallsBackToAlternateQuoteSource(t *testing.T) {
	repo := repository.NewMemoryFundRepository()
	if err := repo.SaveFund(context.Background(), &domain.Fund{
		ID:          "020465",
		Name:        "招商中证半导体产业ETF发起式联接C",
		Type:        "index",
		NetAssetVal: decimal.RequireFromString("1.5000"),
		UpdatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("SaveFund(feeder) error = %v", err)
	}
	if err := repo.SaveFund(context.Background(), &domain.Fund{
		ID:          "561980",
		Name:        "半导体设备ETF招商",
		Type:        "index",
		NetAssetVal: decimal.RequireFromString("2.3100"),
		UpdatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("SaveFund(target) error = %v", err)
	}
	if err := repo.SaveHoldings(context.Background(), "561980", []domain.StockHolding{
		{
			StockCode:    "688072",
			StockName:    "拓荆科技",
			Exchange:     domain.ExchangeSH,
			HoldingRatio: decimal.RequireFromString("13.50"),
		},
	}); err != nil {
		t.Fatalf("SaveHoldings(target) error = %v", err)
	}

	store := &stubFundMappingStore{
		mapping: &database.FundMapping{
			FeederCode: "020465",
			FeederName: "招商中证半导体产业ETF发起式联接C",
			TargetCode: "561980",
			TargetName: "半导体设备ETF招商",
			IsResolved: true,
		},
	}
	resolver := &FundResolver{
		mappingStore: store,
		fundRepo:     repo,
	}

	sinaProvider := feederQuoteProvider{
		quotes: map[string]domain.StockQuote{},
	}
	tencentProvider := feederQuoteProvider{
		quotes: map[string]domain.StockQuote{
			"561980": {
				StockCode:     "561980",
				StockName:     "半导体设备ETF招商",
				CurrentPrice:  decimal.RequireFromString("2.368"),
				PrevClose:     decimal.RequireFromString("2.310"),
				ChangePercent: decimal.RequireFromString("2.5108"),
				ChangeAmount:  decimal.RequireFromString("0.058"),
			},
		},
	}

	service := NewValuationService(repo, sinaProvider, noopCacheRepository{})
	service.SetQuoteProvider(domain.QuoteSourceTencent, tencentProvider)
	service.SetFundResolver(resolver)

	estimate, err := service.CalculateEstimate(domain.WithQuoteSource(context.Background(), domain.QuoteSourceSina), "020465")
	if err != nil {
		t.Fatalf("CalculateEstimate() error = %v", err)
	}

	if estimate.ChangePercent.String() != "2.5108" {
		t.Fatalf("change percent = %s, want 2.5108", estimate.ChangePercent.String())
	}
	if estimate.TotalHoldRatio.String() != "100" {
		t.Fatalf("total hold ratio = %s, want 100", estimate.TotalHoldRatio.String())
	}
	if estimate.DataSource != "追踪目标ETF(tencent): 半导体设备ETF招商" {
		t.Fatalf("data source = %q", estimate.DataSource)
	}
	if len(estimate.HoldingDetails) != 1 || estimate.HoldingDetails[0].StockCode != "561980" {
		t.Fatalf("holding details = %+v, want virtual ETF quote detail", estimate.HoldingDetails)
	}
}
