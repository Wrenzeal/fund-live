package service

import (
	"context"
	"testing"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/repository"
	"github.com/shopspring/decimal"
)

type stubOfficialNAVHistoryCrawler struct {
	histories map[string][]domain.FundHistory
}

func (s stubOfficialNAVHistoryCrawler) FetchRecentFundHistory(ctx context.Context, fundCode string, limit int) ([]domain.FundHistory, error) {
	histories := append([]domain.FundHistory(nil), s.histories[fundCode]...)
	if limit > 0 && len(histories) > limit {
		histories = histories[len(histories)-limit:]
	}
	return histories, nil
}

func TestOfficialNAVSyncServiceNextRunAt(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewOfficialNAVSyncService(fundRepo, userRepo, userRepo, userRepo)

	beforeRun := time.Date(2026, 3, 31, 22, 15, 0, 0, officialNAVSyncLocation)
	next := service.nextRunAt(beforeRun)
	if next.Format(time.RFC3339) != "2026-03-31T23:00:00+08:00" {
		t.Fatalf("next run before cutoff = %s", next.Format(time.RFC3339))
	}

	afterRun := time.Date(2026, 3, 31, 23, 15, 0, 0, officialNAVSyncLocation)
	next = service.nextRunAt(afterRun)
	if next.Format(time.RFC3339) != "2026-04-01T23:00:00+08:00" {
		t.Fatalf("next run after cutoff = %s", next.Format(time.RFC3339))
	}
}

func TestOfficialNAVSyncServiceExpectedOfficialNAVDate(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewOfficialNAVSyncService(fundRepo, userRepo, userRepo, userRepo)

	tests := []struct {
		name string
		now  time.Time
		want string
	}{
		{
			name: "trading day before nightly publish window uses previous trading day",
			now:  time.Date(2026, 3, 31, 15, 30, 0, 0, officialNAVSyncLocation),
			want: "2026-03-30",
		},
		{
			name: "trading day after nightly sync hour expects current trading day",
			now:  time.Date(2026, 3, 31, 23, 15, 0, 0, officialNAVSyncLocation),
			want: "2026-03-31",
		},
		{
			name: "weekend uses previous trading day",
			now:  time.Date(2026, 4, 4, 23, 15, 0, 0, officialNAVSyncLocation),
			want: "2026-04-03",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := service.expectedOfficialNAVDate(tt.now); got != tt.want {
				t.Fatalf("expectedOfficialNAVDate() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestOfficialNAVSyncServiceShouldSyncImmediatelyWhenCurrentTradingDayMissing(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewOfficialNAVSyncService(fundRepo, userRepo, userRepo, userRepo)

	if err := userRepo.SaveFundHolding(context.Background(), &domain.UserFundHolding{
		ID:       "ufh_1",
		UserID:   "user-1",
		FundID:   "005827",
		Amount:   decimal.RequireFromString("1000"),
		AsOfDate: "2026-03-31",
	}); err != nil {
		t.Fatalf("SaveFundHolding() error = %v", err)
	}

	service.now = func() time.Time {
		return time.Date(2026, 3, 31, 23, 15, 0, 0, officialNAVSyncLocation)
	}

	if !service.shouldSyncImmediately(context.Background()) {
		t.Fatalf("shouldSyncImmediately() = false, want true")
	}
}

func TestOfficialNAVSyncServiceSkipsImmediateSyncWhenAlreadyUpToDate(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewOfficialNAVSyncService(fundRepo, userRepo, userRepo, userRepo)

	if err := userRepo.SaveFundHolding(context.Background(), &domain.UserFundHolding{
		ID:       "ufh_1",
		UserID:   "user-1",
		FundID:   "005827",
		Amount:   decimal.RequireFromString("1000"),
		AsOfDate: "2026-03-31",
	}); err != nil {
		t.Fatalf("SaveFundHolding() error = %v", err)
	}
	if err := fundRepo.SaveFundHistory(context.Background(), &domain.FundHistory{
		FundID:      "005827",
		Date:        "2026-03-31",
		NetAssetVal: decimal.RequireFromString("1.8000"),
		AccumVal:    decimal.RequireFromString("2.1000"),
		DailyReturn: decimal.RequireFromString("1.2345"),
		CreatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("SaveFundHistory() error = %v", err)
	}

	service.now = func() time.Time {
		return time.Date(2026, 3, 31, 23, 15, 0, 0, officialNAVSyncLocation)
	}

	if service.shouldSyncImmediately(context.Background()) {
		t.Fatalf("shouldSyncImmediately() = true, want false")
	}
}

func TestOfficialNAVSyncServiceSkipsImmediateSyncWhenHistoryIsNewerThanExpected(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewOfficialNAVSyncService(fundRepo, userRepo, userRepo, userRepo)

	if err := userRepo.SaveFundHolding(context.Background(), &domain.UserFundHolding{
		ID:       "ufh_1",
		UserID:   "user-1",
		FundID:   "005827",
		Amount:   decimal.RequireFromString("1000"),
		AsOfDate: "2026-03-31",
	}); err != nil {
		t.Fatalf("SaveFundHolding() error = %v", err)
	}
	if err := fundRepo.SaveFundHistory(context.Background(), &domain.FundHistory{
		FundID:      "005827",
		Date:        "2026-03-31",
		NetAssetVal: decimal.RequireFromString("1.8000"),
		AccumVal:    decimal.RequireFromString("2.1000"),
		DailyReturn: decimal.RequireFromString("1.2345"),
		CreatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("SaveFundHistory() error = %v", err)
	}

	service.now = func() time.Time {
		return time.Date(2026, 3, 31, 15, 30, 0, 0, officialNAVSyncLocation)
	}

	if service.shouldSyncImmediately(context.Background()) {
		t.Fatalf("shouldSyncImmediately() = true, want false when latest history is newer than expected date")
	}
}

func TestOfficialNAVSyncServiceSkipsImmediateSyncBeforePublishWhenPreviousTradingDayPresent(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewOfficialNAVSyncService(fundRepo, userRepo, userRepo, userRepo)

	if err := userRepo.SaveFundHolding(context.Background(), &domain.UserFundHolding{
		ID:       "ufh_1",
		UserID:   "user-1",
		FundID:   "005827",
		Amount:   decimal.RequireFromString("1000"),
		AsOfDate: "2026-03-31",
	}); err != nil {
		t.Fatalf("SaveFundHolding() error = %v", err)
	}
	if err := fundRepo.SaveFundHistory(context.Background(), &domain.FundHistory{
		FundID:      "005827",
		Date:        "2026-03-30",
		NetAssetVal: decimal.RequireFromString("1.8000"),
		AccumVal:    decimal.RequireFromString("2.1000"),
		DailyReturn: decimal.RequireFromString("1.2345"),
		CreatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("SaveFundHistory() error = %v", err)
	}

	service.now = func() time.Time {
		return time.Date(2026, 3, 31, 15, 30, 0, 0, officialNAVSyncLocation)
	}

	if service.shouldSyncImmediately(context.Background()) {
		t.Fatalf("shouldSyncImmediately() = true, want false before official NAV publish window")
	}
}

func TestOfficialNAVSyncServiceSyncsImmediatelyBeforePublishWhenPreviousTradingDayMissing(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewOfficialNAVSyncService(fundRepo, userRepo, userRepo, userRepo)

	if err := userRepo.SaveFundHolding(context.Background(), &domain.UserFundHolding{
		ID:       "ufh_1",
		UserID:   "user-1",
		FundID:   "005827",
		Amount:   decimal.RequireFromString("1000"),
		AsOfDate: "2026-03-31",
	}); err != nil {
		t.Fatalf("SaveFundHolding() error = %v", err)
	}

	service.now = func() time.Time {
		return time.Date(2026, 3, 31, 15, 30, 0, 0, officialNAVSyncLocation)
	}

	if !service.shouldSyncImmediately(context.Background()) {
		t.Fatalf("shouldSyncImmediately() = false, want true when previous official NAV is missing")
	}
}

func TestOfficialNAVSyncServiceSkipsImmediateSyncOnWeekend(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewOfficialNAVSyncService(fundRepo, userRepo, userRepo, userRepo)

	service.now = func() time.Time {
		return time.Date(2026, 4, 4, 23, 15, 0, 0, officialNAVSyncLocation)
	}

	if service.shouldSyncImmediately(context.Background()) {
		t.Fatalf("shouldSyncImmediately() = true, want false on weekend")
	}
}

func TestOfficialNAVSyncServiceSyncOnceBackfillsAfterCrawlerGroupCompletes(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewOfficialNAVSyncService(fundRepo, userRepo, userRepo, userRepo)
	service.crawler = stubOfficialNAVHistoryCrawler{
		histories: map[string][]domain.FundHistory{
			"005827": {
				{
					FundID:      "005827",
					Date:        "2026-03-31",
					NetAssetVal: decimal.RequireFromString("2.0000"),
					AccumVal:    decimal.RequireFromString("2.0000"),
					DailyReturn: decimal.RequireFromString("1.0000"),
					CreatedAt:   time.Now(),
				},
			},
		},
	}

	if err := fundRepo.SaveFund(context.Background(), &domain.Fund{
		ID:          "005827",
		Name:        "测试基金",
		NetAssetVal: decimal.RequireFromString("1.0000"),
		UpdatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("SaveFund() error = %v", err)
	}
	if err := userRepo.SaveFundHolding(context.Background(), &domain.UserFundHolding{
		ID:       "ufh_1",
		UserID:   "user-1",
		FundID:   "005827",
		Amount:   decimal.RequireFromString("1000"),
		AsOfDate: "2026-03-31",
	}); err != nil {
		t.Fatalf("SaveFundHolding() error = %v", err)
	}

	if err := service.SyncOnce(context.Background()); err != nil {
		t.Fatalf("SyncOnce() error = %v", err)
	}

	holdings, err := userRepo.ListFundHoldings(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListFundHoldings() error = %v", err)
	}
	if len(holdings) != 1 {
		t.Fatalf("holdings len = %d, want 1", len(holdings))
	}
	if holdings[0].ConfirmedNavDate != "2026-03-31" {
		t.Fatalf("confirmed nav date = %s, want 2026-03-31", holdings[0].ConfirmedNavDate)
	}
	if holdings[0].Shares.String() != "500" {
		t.Fatalf("shares = %s, want 500", holdings[0].Shares.String())
	}
}

func TestOfficialNAVSyncServiceSkipsManualHoldingConfirmation(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewOfficialNAVSyncService(fundRepo, userRepo, userRepo, userRepo)

	if err := userRepo.SaveFundHolding(context.Background(), &domain.UserFundHolding{
		ID:                 "ufh_manual",
		UserID:             "user-1",
		FundID:             "005827",
		Amount:             decimal.RequireFromString("1000"),
		AsOfDate:           "2026-03-31",
		ManualConfirmation: true,
	}); err != nil {
		t.Fatalf("SaveFundHolding() error = %v", err)
	}
	if err := fundRepo.SaveFundHistory(context.Background(), &domain.FundHistory{
		FundID:      "005827",
		Date:        "2026-03-31",
		NetAssetVal: decimal.RequireFromString("2.0000"),
		AccumVal:    decimal.RequireFromString("2.0000"),
		DailyReturn: decimal.RequireFromString("1.0000"),
		CreatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("SaveFundHistory() error = %v", err)
	}

	backfilledCount, err := service.backfillHoldingConfirmations(context.Background())
	if err != nil {
		t.Fatalf("backfillHoldingConfirmations() error = %v", err)
	}
	if backfilledCount != 0 {
		t.Fatalf("backfilledCount = %d, want 0", backfilledCount)
	}

	holdings, err := userRepo.ListFundHoldings(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListFundHoldings() error = %v", err)
	}
	if len(holdings) != 1 {
		t.Fatalf("holdings len = %d, want 1", len(holdings))
	}
	if holdings[0].Shares.GreaterThan(decimal.Zero) || holdings[0].ConfirmedNav.GreaterThan(decimal.Zero) || holdings[0].ConfirmedNavDate != "" {
		t.Fatalf("manual holding was unexpectedly backfilled: %+v", holdings[0])
	}
}

func TestOfficialNAVSyncServiceBackfillsHoldingConfirmationWhenHistoryExists(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewOfficialNAVSyncService(fundRepo, userRepo, userRepo, userRepo)

	if err := userRepo.SaveFundHolding(context.Background(), &domain.UserFundHolding{
		ID:       "ufh_1",
		UserID:   "user-1",
		FundID:   "005827",
		Amount:   decimal.RequireFromString("1000"),
		AsOfDate: "2026-03-31",
	}); err != nil {
		t.Fatalf("SaveFundHolding() error = %v", err)
	}
	if err := fundRepo.SaveFundHistory(context.Background(), &domain.FundHistory{
		FundID:      "005827",
		Date:        "2026-03-31",
		NetAssetVal: decimal.RequireFromString("2.0000"),
		AccumVal:    decimal.RequireFromString("2.0000"),
		DailyReturn: decimal.RequireFromString("1.0000"),
		CreatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("SaveFundHistory() error = %v", err)
	}

	backfilledCount, err := service.backfillHoldingConfirmations(context.Background())
	if err != nil {
		t.Fatalf("backfillHoldingConfirmations() error = %v", err)
	}
	if backfilledCount != 1 {
		t.Fatalf("backfilledCount = %d, want 1", backfilledCount)
	}

	holdings, err := userRepo.ListFundHoldings(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListFundHoldings() error = %v", err)
	}
	if len(holdings) != 1 {
		t.Fatalf("holdings len = %d, want 1", len(holdings))
	}
	if holdings[0].ConfirmedNavDate != "2026-03-31" {
		t.Fatalf("confirmed nav date = %s, want 2026-03-31", holdings[0].ConfirmedNavDate)
	}
	if holdings[0].ConfirmedNav.String() != "2" {
		t.Fatalf("confirmed nav = %s, want 2", holdings[0].ConfirmedNav.String())
	}
	if holdings[0].Shares.String() != "500" {
		t.Fatalf("shares = %s, want 500", holdings[0].Shares.String())
	}
}

func TestOfficialNAVSyncServiceListSyncFundIDsMergesTrackedSources(t *testing.T) {
	fundRepo := repository.NewMemoryFundRepository()
	userRepo := repository.NewMemoryUserRepository()
	service := NewOfficialNAVSyncService(fundRepo, userRepo, userRepo, userRepo)

	if err := userRepo.SaveFundHolding(context.Background(), &domain.UserFundHolding{
		ID:       "ufh_1",
		UserID:   "user-1",
		FundID:   "005827",
		Amount:   decimal.RequireFromString("1000"),
		AsOfDate: "2026-03-31",
	}); err != nil {
		t.Fatalf("SaveFundHolding() error = %v", err)
	}
	if err := userRepo.SaveFavoriteFund(context.Background(), &domain.UserFavoriteFund{
		UserID: "user-1",
		FundID: "003095",
	}); err != nil {
		t.Fatalf("SaveFavoriteFund() error = %v", err)
	}
	group := domain.UserWatchlistGroup{
		ID:        "watchlist-group-1",
		UserID:    "user-1",
		Name:      "观察",
		Accent:    "cyan",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := userRepo.SaveWatchlistGroup(context.Background(), &group); err != nil {
		t.Fatalf("SaveWatchlistGroup() error = %v", err)
	}
	if err := userRepo.SaveWatchlistFund(context.Background(), &domain.UserWatchlistFund{
		GroupID: group.ID,
		FundID:  "005827",
	}); err != nil {
		t.Fatalf("SaveWatchlistFund() duplicate error = %v", err)
	}
	if err := userRepo.SaveWatchlistFund(context.Background(), &domain.UserWatchlistFund{
		GroupID: group.ID,
		FundID:  "320007",
	}); err != nil {
		t.Fatalf("SaveWatchlistFund() error = %v", err)
	}

	fundIDs, err := service.listSyncFundIDs(context.Background())
	if err != nil {
		t.Fatalf("listSyncFundIDs() error = %v", err)
	}
	want := []string{"003095", "005827", "320007"}
	if len(fundIDs) != len(want) {
		t.Fatalf("fundIDs = %+v, want %+v", fundIDs, want)
	}
	for index := range want {
		if fundIDs[index] != want[index] {
			t.Fatalf("fundIDs = %+v, want %+v", fundIDs, want)
		}
	}
}
