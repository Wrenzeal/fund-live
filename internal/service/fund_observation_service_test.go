package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/go-resty/resty/v2"
	"github.com/shopspring/decimal"
)

func TestObservationNormalizesDatesTitlesAndDuplicates(t *testing.T) {
	now := time.Date(2026, 10, 16, 10, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	base := domain.FundObservationNotice{ID: "one", Title: "  关于<em>召开</em>股东大会的公告 &amp; 补充说明  ", Scope: "holding", PublishedAt: "2026-10-15", SourceName: "巨潮资讯", SourceURL: "https://static.cninfo.com.cn/notice.pdf", RelatedSymbol: "600519"}
	duplicate := base
	duplicate.ID = "duplicate-category"
	today := base
	today.ID, today.Title, today.PublishedAt = "today", "完整标题不应截断或判断方向", "2026-10-16"
	invalid := []domain.FundObservationNotice{}
	for _, date := range []string{"", "not-a-date", "2026-02-30", "2026-10-17", "2026-10-16T12:00:00+08:00", "2026-08-31"} {
		item := base
		item.ID, item.PublishedAt = date, date
		invalid = append(invalid, item)
	}
	notices := normalizeObservationNotices(append([]domain.FundObservationNotice{base, duplicate, today}, invalid...), now)
	if len(notices) != 2 || notices[0].ID != "today" || notices[1].Title != "关于召开股东大会的公告 & 补充说明" {
		t.Fatalf("normalized notices = %#v", notices)
	}
	boundary := base
	boundary.PublishedAt = "2026-09-01"
	if got := normalizeObservationNotices([]domain.FundObservationNotice{boundary}, now); len(got) != 1 {
		t.Fatalf("45-day boundary should be included: %#v", got)
	}
}

func TestObservationWindowUsesShanghaiCalendarOnUTCHost(t *testing.T) {
	// The host is still on October 15, but the market calendar is October 16.
	now := time.Date(2026, 10, 15, 18, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		date string
		want bool
	}{
		{"2026-09-01", true},
		{"2026-09-01T00:00:00+08:00", true},
		{"2026-08-31T23:59:59+08:00", false},
		{"2026-10-16", true},
		{"2026-10-16T03:00:00+08:00", false},
		{"2026-10-17", false},
	} {
		notice := domain.FundObservationNotice{Title: "公告", Scope: "fund", PublishedAt: tc.date, SourceName: "来源", SourceURL: "https://example.test/notice"}
		if got := normalizeObservationNotices([]domain.FundObservationNotice{notice}, now); (len(got) == 1) != tc.want {
			t.Errorf("date=%s got=%v want included=%v", tc.date, got, tc.want)
		}
	}
	period, age := observationHoldingPeriod([]domain.StockHolding{{ReportingPeriod: "2026-10-15"}}, now)
	if period != "2026-10-15" || age == nil || *age != 1 {
		t.Fatalf("calendar holding age=%s %v", period, age)
	}
}

func TestObservationNoticeIDsAreStableAndDeduplicated(t *testing.T) {
	now := time.Date(2026, 10, 16, 10, 0, 0, 0, observationLocation)
	base := domain.FundObservationNotice{ID: "same-source-id", Title: "公告标题", Scope: "fund", PublishedAt: "2026-10-15", SourceName: "来源", SourceURL: "https://example.test/notice"}
	duplicate := base
	duplicate.Title = "不同分类返回的同一公告"
	missingID := base
	missingID.ID, missingID.Title = "", "缺少上游编号的公告"
	got := normalizeObservationNotices([]domain.FundObservationNotice{base, duplicate, missingID}, now)
	if len(got) != 2 || got[0].ID == "" || got[1].ID == "" || got[0].ID == got[1].ID {
		t.Fatalf("notice IDs=%#v", got)
	}
}

type observationRepoStub struct {
	domain.FundRepository
	fund        *domain.Fund
	holdings    []domain.StockHolding
	fundErr     error
	holdingsErr error
}

func (r observationRepoStub) GetFundByID(context.Context, string) (*domain.Fund, error) {
	return r.fund, r.fundErr
}

func (r observationRepoStub) GetFundHoldings(context.Context, string) ([]domain.StockHolding, error) {
	return r.holdings, r.holdingsErr
}

type observationResolverStub struct {
	holdings []domain.StockHolding
	source   string
	err      error
}

func (r observationResolverStub) GetHoldingsWithFallback(context.Context, string, string) ([]domain.StockHolding, string, error) {
	return r.holdings, r.source, r.err
}

type observationFundLoaderFunc func(context.Context, string, string, time.Time) ObservationNoticeBatch

func (f observationFundLoaderFunc) LoadFundObservationNotices(ctx context.Context, code, scope string, now time.Time) ObservationNoticeBatch {
	return f(ctx, code, scope, now)
}

type observationHoldingLoaderFunc func(context.Context, []domain.StockHolding, time.Time) ObservationNoticeBatch

func (f observationHoldingLoaderFunc) LoadHoldingObservationNotices(ctx context.Context, holdings []domain.StockHolding, now time.Time) ObservationNoticeBatch {
	return f(ctx, holdings, now)
}

func observationTestService(repo observationRepoStub) *FundObservationService {
	s := NewFundObservationService(repo, nil)
	s.fundNotices = observationFundLoaderFunc(func(context.Context, string, string, time.Time) ObservationNoticeBatch {
		return ObservationNoticeBatch{Requests: 4, SuccessfulRequests: 4}
	})
	s.holdingNotices = observationHoldingLoaderFunc(func(context.Context, []domain.StockHolding, time.Time) ObservationNoticeBatch {
		return ObservationNoticeBatch{}
	})
	return s
}

func TestObservationMissingHoldingsStillReturnsFundNotices(t *testing.T) {
	now := time.Date(2026, 10, 16, 10, 0, 0, 0, time.UTC)
	s := observationTestService(observationRepoStub{fund: &domain.Fund{ID: "000001"}, holdingsErr: errors.New("private database diagnostic")})
	s.fundNotices = observationFundLoaderFunc(func(context.Context, string, string, time.Time) ObservationNoticeBatch {
		return ObservationNoticeBatch{Requests: 4, SuccessfulRequests: 4, Notices: []domain.FundObservationNotice{{ID: "fund", Title: "关于召开股东大会的公告", Scope: "fund", PublishedAt: "2026-10-15", SourceName: "东方财富基金公告", SourceURL: "https://fundf10.eastmoney.com/jjgg_000001.html"}}}
	})
	_, observation, err := s.Observe(context.Background(), "000001", now)
	if err != nil || observation.HoldingsStatus != "unavailable" || len(observation.Notices) != 1 {
		t.Fatalf("observation=%#v err=%v", observation, err)
	}
	if observation.HoldingAgeDays != nil || observation.Sources[0].Status != "ok" || observation.Sources[1].Status != "unavailable" {
		t.Fatalf("unexpected status: %#v", observation)
	}
	if strings.Contains(observation.Sources[1].Message, "private database") {
		t.Fatal("internal diagnostic leaked to user message")
	}
}

func TestObservationReportsRawCoverageAndTargetSource(t *testing.T) {
	now := time.Date(2026, 10, 16, 10, 0, 0, 0, time.UTC)
	s := observationTestService(observationRepoStub{fund: &domain.Fund{ID: "000001"}})
	s.holdingsResolver = observationResolverStub{source: "510300", holdings: []domain.StockHolding{
		{StockCode: "600519", Exchange: domain.ExchangeSH, HoldingRatio: decimal.RequireFromString("8.25"), ReportingPeriod: "2026Q2"},
		{StockCode: "000858", Exchange: domain.ExchangeSZ, HoldingRatio: decimal.RequireFromString("3.75"), ReportingPeriod: "2026Q2"},
	}}
	s.holdingNotices = observationHoldingLoaderFunc(func(context.Context, []domain.StockHolding, time.Time) ObservationNoticeBatch {
		return ObservationNoticeBatch{Requests: 2, SuccessfulRequests: 1, Err: errors.New("one upstream failed")}
	})
	_, observation, err := s.Observe(context.Background(), "000001", now)
	if err != nil || observation.DisclosedWeightPercent.String() != "12" || observation.HoldingsCount != 2 || observation.HoldingsSource != "target_etf_fallback" || observation.LatestHoldingPeriod != "2026Q2" || observation.HoldingAgeDays == nil || *observation.HoldingAgeDays != 108 {
		t.Fatalf("observation=%#v err=%v", observation, err)
	}
	if observation.Sources[1].Status != "partial" {
		t.Fatalf("expected partial source: %#v", observation.Sources)
	}
}

func TestObservationSourceStatesAndUnknownPeriod(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		batch      ObservationNoticeBatch
	}{
		{"empty success", "ok", ObservationNoticeBatch{Requests: 1, SuccessfulRequests: 1}},
		{"failed", "unavailable", ObservationNoticeBatch{Requests: 1, Err: errors.New("token=do-not-display")}},
		{"partial", "partial", ObservationNoticeBatch{Requests: 2, SuccessfulRequests: 1, Err: errors.New("one failure")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := observationTestService(observationRepoStub{fund: &domain.Fund{ID: "000001"}, holdings: []domain.StockHolding{{StockCode: "AAPL", Exchange: domain.ExchangeUS, HoldingRatio: decimal.NewFromInt(7), ReportingPeriod: "unknown"}}})
			s.fundNotices = observationFundLoaderFunc(func(context.Context, string, string, time.Time) ObservationNoticeBatch { return tc.batch })
			_, observation, err := s.Observe(context.Background(), "000001", time.Now())
			if err != nil || observation.Sources[0].Status != tc.want || observation.Sources[1].Status != "not_applicable" || observation.HoldingAgeDays != nil {
				t.Fatalf("observation=%#v err=%v", observation, err)
			}
		})
	}
}

func TestObservationTimeoutIsExplicit(t *testing.T) {
	s := observationTestService(observationRepoStub{fund: &domain.Fund{ID: "000001"}})
	s.timeout = 10 * time.Millisecond
	s.fundNotices = observationFundLoaderFunc(func(ctx context.Context, _ string, _ string, _ time.Time) ObservationNoticeBatch {
		<-ctx.Done()
		return ObservationNoticeBatch{Requests: 1, Err: ctx.Err()}
	})
	_, observation, err := s.Observe(context.Background(), "000001", time.Now())
	if err != nil || observation.Sources[0].Status != "unavailable" {
		t.Fatalf("observation=%#v err=%v", observation, err)
	}
}

func TestObservationTimeoutKeepsCompletedPagesAsPartial(t *testing.T) {
	s := observationTestService(observationRepoStub{fund: &domain.Fund{ID: "000001"}})
	s.timeout = 10 * time.Millisecond
	s.fundNotices = observationFundLoaderFunc(func(ctx context.Context, _ string, _ string, _ time.Time) ObservationNoticeBatch {
		<-ctx.Done()
		return ObservationNoticeBatch{Requests: 2, SuccessfulRequests: 1, Err: ctx.Err()}
	})
	_, observation, err := s.Observe(context.Background(), "000001", time.Now())
	if err != nil || observation.Sources[0].Status != "partial" {
		t.Fatalf("completed page lost at timeout: observation=%#v err=%v", observation, err)
	}
}

func TestObservationReportPeriodRejectsFutureAndUsesCalendarDates(t *testing.T) {
	now := time.Date(2026, 10, 16, 10, 0, 0, 0, time.UTC)
	period, age := observationHoldingPeriod([]domain.StockHolding{{ReportingPeriod: "2026Q4"}, {ReportingPeriod: "2026Q0"}, {ReportingPeriod: "not-a-period"}}, now)
	if period != "" || age != nil {
		t.Fatalf("future or invalid report was treated as current: %s %v", period, age)
	}
	period, age = observationHoldingPeriod([]domain.StockHolding{{ReportingPeriod: "2026-09-30"}, {ReportingPeriod: "2026Q2"}}, now)
	if period != "2026-09-30" || age == nil || *age != 16 {
		t.Fatalf("actual period/age = %s %v", period, age)
	}
}

func TestRawFundObservationKeepsUnclassifiedTitlesAndFailureEvidence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("type") == "4" {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"Data":[]}`))
			return
		}
		_, _ = w.Write([]byte(`jQuery({"Data":[{"ID":"AN202610150001","TITLE":"关于召开股东大会的完整公告标题","PUBLISHDATEDesc":"2026-10-15"}]})`))
	}))
	defer server.Close()
	client := resty.New()
	client.SetTransport(observationRewriteTransport{base: server.Client().Transport, target: server.URL})
	source := &FundNoticeSource{client: client}
	batch := source.LoadFundObservationNotices(context.Background(), "000001", "fund", time.Now())
	if batch.Requests != 4 || batch.SuccessfulRequests != 3 || batch.Err == nil || len(batch.Notices) != 3 || batch.Notices[0].Title != "关于召开股东大会的完整公告标题" {
		t.Fatalf("raw batch=%#v", batch)
	}
}

func TestRawFundObservationDistinguishesEmptyAndMalformedEnvelopes(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		success    bool
	}{
		{"empty", `{"Data":[]}`, true},
		{"missing field", `{}`, false},
		{"wrong data type", `{"Data":{"message":"failure"}}`, false},
		{"null envelope", `null`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			client := resty.New().SetTransport(observationRewriteTransport{base: server.Client().Transport, target: server.URL})
			batch := (&FundNoticeSource{client: client}).LoadFundObservationNotices(context.Background(), "000001", "fund", time.Now())
			if (batch.SuccessfulRequests == 4 && batch.Err == nil) != tc.success {
				t.Fatalf("batch=%#v success=%v", batch, tc.success)
			}
		})
	}
}

type observationRewriteTransport struct {
	base   http.RoundTripper
	target string
}

func (t observationRewriteTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	copy := r.Clone(r.Context())
	parsed, err := http.NewRequestWithContext(r.Context(), r.Method, t.target+r.URL.RequestURI(), r.Body)
	if err != nil {
		return nil, err
	}
	copy.URL = parsed.URL
	return t.base.RoundTrip(copy)
}

func TestRawHoldingObservationUnsupportedAndMalformedResponses(t *testing.T) {
	source := NewHoldingNewsSource()
	if batch := source.LoadHoldingObservationNotices(context.Background(), []domain.StockHolding{{StockCode: "AAPL", Exchange: domain.ExchangeUS}}, time.Now()); batch.Requests != 0 || batch.Err != nil {
		t.Fatalf("unsupported market should be not applicable: %#v", batch)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"unexpected":"not a successful empty result"}`))
	}))
	defer server.Close()
	source.client.SetTransport(observationRewriteTransport{base: server.Client().Transport, target: server.URL})
	batch := source.LoadHoldingObservationNotices(context.Background(), []domain.StockHolding{{StockCode: "600519", Exchange: domain.ExchangeSH}}, time.Now())
	if batch.Requests != 1 || batch.SuccessfulRequests != 0 || batch.Err == nil {
		t.Fatalf("malformed response must stay unavailable: %#v", batch)
	}
}

func TestRawHoldingObservationRetainsGenericTitleAndOriginalDocument(t *testing.T) {
	now := time.Date(2026, 10, 16, 10, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "topSearch") {
			_, _ = w.Write([]byte(`[{"code":"600519","orgId":"organization"}]`))
			return
		}
		_, _ = w.Write([]byte(`{"announcements":[{"announcementId":"notice","announcementTitle":"关于召开股东大会的完整公告标题","announcementTime":1792036800000,"adjunctUrl":"finalpage/2026-10-15/notice.PDF"}]}`))
	}))
	defer server.Close()
	source := NewHoldingNewsSource()
	source.client.SetTransport(observationRewriteTransport{base: server.Client().Transport, target: server.URL})
	batch := source.LoadHoldingObservationNotices(context.Background(), []domain.StockHolding{{StockCode: "600519", StockName: "贵州茅台", Exchange: domain.ExchangeSH}}, now)
	if batch.Err != nil || batch.SuccessfulRequests != 1 || len(batch.Notices) != 1 || batch.Notices[0].Title != "关于召开股东大会的完整公告标题" || batch.Notices[0].SourceURL != "https://static.cninfo.com.cn/finalpage/2026-10-15/notice.PDF" {
		t.Fatalf("raw holding batch = %#v", batch)
	}
}
