package service

import (
	"context"
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
)

var ErrOpeningNAVUnavailable = errors.New("该净值日暂无可用官方净值，请选择已公布日期或改用持有份额录入")
var ErrInvalidOpeningPosition = errors.New("请提供有效的已有持仓份额和剩余成本，或同一净值日的市值与持有盈亏")

type holdingBatchHistoryKey struct{}

func holdingEntryType(value string) string {
	if value == "opening" {
		return value
	}
	return "buy"
}

func (s *UserPreferenceService) prepareHoldingBatchHistory(ctx context.Context, inputs []domain.CreateFundHoldingInput) context.Context {
	if s.holdingHistorySync == nil {
		return ctx
	}
	dates := make(map[string][]string)
	for _, input := range inputs {
		if input.EntryType == "opening" && input.Shares != "" {
			continue
		}
		fund, err := s.fundRepo.GetFundByID(ctx, strings.TrimSpace(input.FundID))
		if err != nil || fund == nil {
			continue
		}
		date := input.SnapshotDate
		if input.EntryType != "opening" {
			trade, err := parseHoldingTradeAt(input.TradeAt)
			if err != nil {
				continue
			}
			date = resolveHoldingPricingDate(trade).Format("2006-01-02")
		}
		if date != "" && date <= expectedOfficialHistoryDate(time.Now()) {
			dates[fund.ID] = append(dates[fund.ID], date)
		}
	}
	bounded, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	results := make(map[string]error)
	var mu sync.Mutex
	group, groupCtx := errgroup.WithContext(bounded)
	group.SetLimit(4)
	for fundID, required := range dates {
		group.Go(func() error {
			err := s.ensureHoldingHistory(groupCtx, fundID, required)
			mu.Lock()
			results[fundID] = err
			mu.Unlock()
			return nil
		})
	}
	_ = group.Wait()
	return context.WithValue(ctx, holdingBatchHistoryKey{}, results)
}

func (s *UserPreferenceService) SetHoldingHistorySync(sync func(context.Context, string, []string) error) {
	s.holdingHistorySync = sync
}

func (s *UserPreferenceService) requestHoldingHistory(fundID string, dates []string, cooldown ...time.Duration) {
	if s.holdingHistorySync == nil {
		return
	}
	key := fundID + "|" + strings.Join(dates, ",")
	now := time.Now()
	interval := time.Minute
	if len(cooldown) > 0 {
		interval = cooldown[0]
	}
	s.navRequestsMu.Lock()
	if s.navRequests == nil {
		s.navRequests = make(map[string]time.Time)
	}
	if previous := s.navRequests[key]; now.Sub(previous) < interval {
		s.navRequestsMu.Unlock()
		return
	}
	s.navRequests[key] = now
	s.navRequestsMu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := s.holdingHistorySync(ctx, fundID, dates); err != nil {
			log.Printf("Holding NAV refresh %s: %v", fundID, err)
		}
	}()
}

func (s *UserPreferenceService) ensureHoldingHistory(ctx context.Context, fundID string, dates []string) error {
	if attempted, ok := ctx.Value(holdingBatchHistoryKey{}).(map[string]error); ok {
		if err, found := attempted[fundID]; found {
			return err
		}
	}
	if s.holdingHistorySync == nil {
		return nil
	}
	bounded, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	return s.holdingHistorySync(bounded, fundID, dates)
}

func (s *UserPreferenceService) CreateFundHoldingWithInput(ctx context.Context, userID string, input domain.CreateFundHoldingInput) (*domain.UserFundHoldingDetail, error) {
	return s.createFundHolding(ctx, userID, input)
}

func parseSnapshotDate(raw string) (string, error) {
	date, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(raw), holdingTradeLocation)
	if err != nil || date.Format("2006-01-02") > time.Now().In(holdingTradeLocation).Format("2006-01-02") {
		return "", ErrInvalidHoldingDate
	}
	return date.Format("2006-01-02"), nil
}

func (s *UserPreferenceService) openingDetail(ctx context.Context, holding *domain.UserFundHolding, fund *domain.Fund) (*domain.UserFundHoldingDetail, error) {
	latest, err := s.fundRepo.GetLatestFundHistory(ctx, holding.FundID)
	if err != nil {
		return nil, err
	}
	detail, _ := buildUserFundHoldingDetail(*holding, fund, latest, expectedOfficialHistoryDate(time.Now()))
	return &detail, nil
}

func (s *UserPreferenceService) createOpeningHolding(ctx context.Context, userID string, input domain.CreateFundHoldingInput) (*domain.UserFundHoldingDetail, error) {
	fundID := strings.TrimSpace(input.FundID)
	fund, err := s.fundRepo.GetFundByID(ctx, fundID)
	if err != nil {
		return nil, err
	}
	if fund == nil {
		return nil, ErrFundNotFound
	}
	date, err := parseSnapshotDate(input.SnapshotDate)
	if err != nil {
		return nil, err
	}
	var amount, shares decimal.Decimal
	if strings.TrimSpace(input.Shares) != "" {
		if input.MarketValue != "" || input.HoldingProfit != "" {
			return nil, ErrInvalidOpeningPosition
		}
		amount, err = parsePositiveDecimal(input.Amount, ErrInvalidOpeningPosition)
		if err != nil {
			return nil, err
		}
		shares, err = parsePositiveDecimal(input.Shares, ErrInvalidOpeningPosition)
		if err != nil {
			return nil, err
		}
	} else {
		market, parseErr := parsePositiveDecimal(input.MarketValue, ErrInvalidOpeningPosition)
		if parseErr != nil {
			return nil, parseErr
		}
		profit, parseErr := decimal.NewFromString(strings.TrimSpace(input.HoldingProfit))
		if parseErr != nil || strings.TrimSpace(input.HoldingProfit) == "" {
			return nil, ErrInvalidOpeningPosition
		}
		amount = market.Sub(profit).RoundBank(2)
		if amount.Sign() <= 0 || strings.TrimSpace(input.Amount) != "" {
			return nil, ErrInvalidOpeningPosition
		}
		if err := s.ensureHoldingHistory(ctx, fundID, []string{date}); err != nil {
			return nil, ErrOpeningNAVUnavailable
		}
		history, err := s.lookupFundHistory(ctx, fundID, date)
		if err != nil {
			return nil, err
		}
		if history == nil || history.NetAssetVal.Sign() <= 0 {
			return nil, ErrOpeningNAVUnavailable
		}
		shares = market.DivRound(history.NetAssetVal, 6)
	}
	if shares.Round(6).Sign() <= 0 {
		return nil, ErrInvalidOpeningPosition
	}
	source, label, err := normalizeHoldingSource(input.SourcePlatform, input.SourceLabel)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	holding := &domain.UserFundHolding{ID: generateID("ufh"), UserID: userID, FundID: fundID, EntryType: "opening", Amount: amount, Shares: shares.Round(6), ManualConfirmation: true, AsOfDate: date, Note: strings.TrimSpace(input.Note), SourcePlatform: source, SourceLabel: label, CreatedAt: now, UpdatedAt: now}
	activity := newFundHoldingTransaction(domain.UserFundHoldingTransactionOpening, holding, "登记已有持仓", map[string]string{"source": "opening_position", "snapshot_date": date})
	detail, err := s.openingDetail(ctx, holding, fund)
	if err != nil {
		return nil, err
	}
	if err := s.fundHoldingRepo.CreateFundHoldingWithTransaction(ctx, holding, activity); err != nil {
		return nil, err
	}
	s.requestHoldingHistory(fundID, nil)
	return detail, nil
}

func (s *UserPreferenceService) updateOpeningHolding(ctx context.Context, holding *domain.UserFundHolding, fund *domain.Fund, input domain.UpdateFundHoldingInput) (*domain.UserFundHoldingDetail, error) {
	amount, err := parsePositiveDecimal(input.Amount, ErrInvalidOpeningPosition)
	if err != nil {
		return nil, err
	}
	shares, err := parsePositiveDecimal(input.Shares, ErrInvalidOpeningPosition)
	if err != nil {
		return nil, err
	}
	updated := *holding
	updated.Amount = amount
	updated.Shares = shares.Round(6)
	updated.Note = strings.TrimSpace(input.Note)
	if input.SnapshotDate != "" {
		updated.AsOfDate, err = parseSnapshotDate(input.SnapshotDate)
		if err != nil {
			return nil, err
		}
	}
	updated.SourcePlatform, updated.SourceLabel, err = normalizeHoldingSource(input.SourcePlatform, input.SourceLabel)
	if err != nil {
		return nil, err
	}
	updated.UpdatedAt = time.Now()
	if err := s.fundHoldingRepo.SaveFundHolding(ctx, &updated); err != nil {
		return nil, err
	}
	if err := s.recordFundHoldingTransaction(ctx, domain.UserFundHoldingTransactionCorrection, &updated, "校正已有持仓", buildHoldingCorrectionMetadata(holding, &updated)); err != nil {
		return nil, err
	}
	return s.openingDetail(ctx, &updated, fund)
}
