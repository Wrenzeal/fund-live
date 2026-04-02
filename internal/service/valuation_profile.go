package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/shopspring/decimal"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	PricingMethodFuturesUnderlying = "futures_underlying"
	QuoteSourceSinaFuturesCN       = "sina_futures_cn"
)

type ValuationProfileStore struct {
	db *gorm.DB
}

type futuresQuote struct {
	Name            string
	CurrentPrice    decimal.Decimal
	PrevSettlement  decimal.Decimal
	SettlementPrice decimal.Decimal
}

func NewValuationProfileStore(db *gorm.DB) *ValuationProfileStore {
	return &ValuationProfileStore{db: db}
}

func (s *ValuationProfileStore) GetByFundID(ctx context.Context, fundID string) (*database.FundValuationProfile, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}

	var profile database.FundValuationProfile
	result := s.db.WithContext(ctx).First(&profile, "fund_id = ?", fundID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &profile, nil
}

func SeedDefaultValuationProfiles(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}

	profiles := []database.FundValuationProfile{
		{
			FundID:            "161226",
			PricingMethod:     PricingMethodFuturesUnderlying,
			QuoteSource:       QuoteSourceSinaFuturesCN,
			UnderlyingSymbol:  "AG0",
			UnderlyingName:    "上期所白银主力合约",
			EffectiveExposure: decimal.NewFromInt(1),
			Notes:             "国投瑞银白银期货(LOF)A",
		},
		{
			FundID:            "019005",
			PricingMethod:     PricingMethodFuturesUnderlying,
			QuoteSource:       QuoteSourceSinaFuturesCN,
			UnderlyingSymbol:  "AG0",
			UnderlyingName:    "上期所白银主力合约",
			EffectiveExposure: decimal.NewFromInt(1),
			Notes:             "国投瑞银白银期货(LOF)C",
		},
	}

	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "fund_id"}},
		UpdateAll: true,
	}).Create(&profiles).Error
}

func isCommodityOrFuturesFund(fund *domain.Fund) bool {
	if fund == nil {
		return false
	}
	name := strings.ToLower(fund.Name)
	typ := strings.ToLower(fund.Type)
	return strings.Contains(name, "期货") || strings.Contains(name, "白银") || strings.Contains(name, "黄金") || strings.Contains(typ, "商品")
}

func (s *ValuationServiceImpl) calculateEstimateFromValuationProfile(ctx context.Context, fund *domain.Fund) (*domain.FundEstimate, bool, error) {
	if s.profileStore == nil || fund == nil {
		return nil, false, nil
	}

	profile, err := s.profileStore.GetByFundID(ctx, fund.ID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to load valuation profile: %w", err)
	}
	if profile == nil {
		if isCommodityOrFuturesFund(fund) {
			return nil, true, fmt.Errorf("pricing profile not configured for commodity/futures fund: %s", fund.ID)
		}
		return nil, false, nil
	}

	switch profile.PricingMethod {
	case PricingMethodFuturesUnderlying:
		estimate, err := calculateEstimateFromDomesticFutures(ctx, fund, profile)
		if err != nil {
			return nil, true, err
		}
		return estimate, true, nil
	default:
		return nil, true, fmt.Errorf("unsupported pricing method %s for fund %s", profile.PricingMethod, fund.ID)
	}
}

func calculateEstimateFromDomesticFutures(ctx context.Context, fund *domain.Fund, profile *database.FundValuationProfile) (*domain.FundEstimate, error) {
	quote, err := fetchDomesticFuturesQuote(ctx, profile.UnderlyingSymbol)
	if err != nil {
		return nil, err
	}
	if quote.PrevSettlement.IsZero() {
		return nil, fmt.Errorf("missing previous settlement for futures symbol %s", profile.UnderlyingSymbol)
	}

	exposure := profile.EffectiveExposure
	if exposure.IsZero() {
		exposure = decimal.NewFromInt(1)
	}

	hundred := decimal.NewFromInt(100)
	changePercent := quote.CurrentPrice.Sub(quote.PrevSettlement).Div(quote.PrevSettlement).Mul(hundred)
	changePercent = changePercent.Mul(exposure).Round(4)

	estimateNav := decimal.Zero
	if !fund.NetAssetVal.IsZero() {
		estimateNav = fund.NetAssetVal.Mul(decimal.NewFromInt(1).Add(changePercent.Div(hundred))).Round(4)
	}

	detailName := profile.UnderlyingName
	if detailName == "" {
		detailName = quote.Name
	}
	if detailName == "" {
		detailName = profile.UnderlyingSymbol
	}

	return &domain.FundEstimate{
		FundID:         fund.ID,
		FundName:       fund.Name,
		EstimateNav:    estimateNav,
		PrevNav:        fund.NetAssetVal,
		ChangePercent:  changePercent,
		ChangeAmount:   estimateNav.Sub(fund.NetAssetVal).Round(4),
		TotalHoldRatio: decimal.NewFromInt(100),
		HoldingDetails: []domain.HoldingDetail{{
			StockCode:    profile.UnderlyingSymbol,
			StockName:    detailName,
			HoldingRatio: decimal.NewFromInt(100),
			StockChange:  changePercent,
			Contribution: changePercent,
			CurrentPrice: quote.CurrentPrice,
			PrevClose:    quote.PrevSettlement,
		}},
		CalculatedAt: time.Now(),
		DataSource:   fmt.Sprintf("%s:%s", profile.QuoteSource, detailName),
	}, nil
}

func fetchDomesticFuturesQuote(ctx context.Context, symbol string) (*futuresQuote, error) {
	url := fmt.Sprintf("https://hq.sinajs.cn/list=nf_%s", symbol)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Referer", "https://finance.sina.com.cn")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	decoded := transform.NewReader(resp.Body, simplifiedchinese.GBK.NewDecoder())
	bodyBytes, err := io.ReadAll(decoded)
	if err != nil {
		return nil, err
	}
	body := string(bodyBytes)

	prefix := fmt.Sprintf("var hq_str_nf_%s=\"", symbol)
	start := strings.Index(body, prefix)
	if start == -1 {
		return nil, fmt.Errorf("unexpected futures quote response for %s", symbol)
	}
	payload := body[start+len(prefix):]
	end := strings.Index(payload, "\";")
	if end == -1 {
		return nil, fmt.Errorf("malformed futures quote payload for %s", symbol)
	}
	fields := strings.Split(payload[:end], ",")
	if len(fields) < 11 {
		return nil, fmt.Errorf("insufficient futures quote fields for %s", symbol)
	}

	current, err := decimal.NewFromString(fields[5])
	if err != nil {
		return nil, fmt.Errorf("parse futures current price: %w", err)
	}
	prevSettlement, err := decimal.NewFromString(fields[10])
	if err != nil {
		return nil, fmt.Errorf("parse futures previous settlement: %w", err)
	}
	settlement, _ := decimal.NewFromString(fields[9])

	return &futuresQuote{
		Name:            strings.TrimSpace(fields[0]),
		CurrentPrice:    current,
		PrevSettlement:  prevSettlement,
		SettlementPrice: settlement,
	}, nil
}
