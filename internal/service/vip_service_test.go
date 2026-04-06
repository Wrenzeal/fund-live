package service

import (
	"context"
	"testing"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/repository"
	"github.com/RomaticDOG/fund/internal/trading"
)

type mockWeChatPayClient struct {
	createResult *WeChatNativeOrderResult
	createErr    error
	queryResult  *WeChatOrderStatusResult
	queryErr     error
	notifyResult *WeChatPaymentNotifyResult
	notifyErr    error
}

func (m *mockWeChatPayClient) CreateNativeOrder(ctx context.Context, input WeChatNativeOrderInput) (*WeChatNativeOrderResult, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return m.createResult, nil
}

func (m *mockWeChatPayClient) QueryOrderByOutTradeNo(ctx context.Context, outTradeNo string) (*WeChatOrderStatusResult, error) {
	if m.queryErr != nil {
		return nil, m.queryErr
	}
	return m.queryResult, nil
}

func (m *mockWeChatPayClient) ParsePaymentNotify(headers map[string]string, body []byte) (*WeChatPaymentNotifyResult, error) {
	if m.notifyErr != nil {
		return nil, m.notifyErr
	}
	return m.notifyResult, nil
}

func TestVIPServiceActivatePreviewMembershipAndQuota(t *testing.T) {
	repo := repository.NewMemoryVIPRepository()
	service := NewVIPService(repo)
	now := time.Date(2026, time.April, 5, 10, 0, 0, 0, trading.TradingLocation())
	service.now = func() time.Time { return now }

	membership, err := service.ActivatePreviewMembership(context.Background(), "user-1", domain.VIPBillingCycleYearly)
	if err != nil {
		t.Fatalf("ActivatePreviewMembership() error = %v", err)
	}
	if !membership.IsVIP {
		t.Fatalf("membership should be active")
	}
	if membership.BillingCycle != domain.VIPBillingCycleYearly {
		t.Fatalf("billing cycle = %q, want yearly", membership.BillingCycle)
	}

	quota, err := service.GetQuota(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("GetQuota() error = %v", err)
	}
	if quota.SectorAnalysisRemaining != domain.VIPDailySectorAnalysisLimit {
		t.Fatalf("sector remaining = %d, want %d", quota.SectorAnalysisRemaining, domain.VIPDailySectorAnalysisLimit)
	}
	if quota.PortfolioAnalysisRemaining != domain.VIPDailyPortfolioAnalysisLimit {
		t.Fatalf("portfolio remaining = %d, want %d", quota.PortfolioAnalysisRemaining, domain.VIPDailyPortfolioAnalysisLimit)
	}
}

func TestVIPServiceCreateTaskConsumesQuotaAndTransitions(t *testing.T) {
	repo := repository.NewMemoryVIPRepository()
	service := NewVIPService(repo)
	baseNow := time.Date(2026, time.April, 5, 10, 0, 0, 0, trading.TradingLocation())
	service.now = func() time.Time { return baseNow }

	if _, err := service.ActivatePreviewMembership(context.Background(), "user-1", domain.VIPBillingCycleMonthly); err != nil {
		t.Fatalf("ActivatePreviewMembership() error = %v", err)
	}

	result, err := service.CreateTask(context.Background(), "user-1", domain.VIPTaskCreateInput{
		Type:       domain.VIPTaskTypeSectorAnalysis,
		TargetType: domain.VIPTargetTypeWatchlistGroup,
		TargetID:   "group-1",
		TargetName: "智能制造主题自选分组",
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if result.TaskID == "" {
		t.Fatalf("expected task id")
	}

	quota, err := service.GetQuota(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("GetQuota() error = %v", err)
	}
	if quota.SectorAnalysisRemaining != domain.VIPDailySectorAnalysisLimit-1 {
		t.Fatalf("sector remaining = %d, want %d", quota.SectorAnalysisRemaining, domain.VIPDailySectorAnalysisLimit-1)
	}

	tasks, err := service.ListTasks(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("tasks len = %d, want 1", len(tasks))
	}
	if tasks[0].Status != domain.VIPTaskStatusQueued {
		t.Fatalf("initial status = %q, want queued", tasks[0].Status)
	}
	if tasks[0].ReportID != "" {
		t.Fatalf("initial report id should be empty")
	}

	service.now = func() time.Time { return baseNow.Add(4 * time.Second) }
	tasks, err = service.ListTasks(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if tasks[0].Status != domain.VIPTaskStatusRunning {
		t.Fatalf("running status = %q, want running", tasks[0].Status)
	}

	service.now = func() time.Time { return baseNow.Add(10 * time.Second) }
	tasks, err = service.ListTasks(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if tasks[0].Status != domain.VIPTaskStatusCompleted {
		t.Fatalf("completed status = %q, want completed", tasks[0].Status)
	}
	if tasks[0].ReportID == "" {
		t.Fatalf("completed task should expose report id")
	}

	report, err := service.GetReportByID(context.Background(), "user-1", tasks[0].ReportID)
	if err != nil {
		t.Fatalf("GetReportByID() error = %v", err)
	}
	if report == nil || report.TargetName != "智能制造主题自选分组" {
		t.Fatalf("report target name = %+v", report)
	}
}

func TestVIPServiceReturnsPublicSampleReportWithoutAuth(t *testing.T) {
	repo := repository.NewMemoryVIPRepository()
	service := NewVIPService(repo)

	report, err := service.GetReportByID(context.Background(), "", defaultVIPPortfolioReportID)
	if err != nil {
		t.Fatalf("GetReportByID() error = %v", err)
	}
	if report == nil {
		t.Fatalf("expected sample report")
	}
	if report.ID != defaultVIPPortfolioReportID {
		t.Fatalf("report id = %q, want %q", report.ID, defaultVIPPortfolioReportID)
	}
}

func TestVIPServiceCreateOrderAndPromoteMembershipOnQuery(t *testing.T) {
	repo := repository.NewMemoryVIPRepository()
	service := NewVIPService(repo)
	baseNow := time.Date(2026, time.April, 5, 12, 0, 0, 0, trading.TradingLocation())
	service.now = func() time.Time { return baseNow }
	service.SetWeChatPayClient(&mockWeChatPayClient{
		createResult: &WeChatNativeOrderResult{CodeURL: "weixin://pay/mock-qr"},
		queryResult: &WeChatOrderStatusResult{
			Status:        domain.VIPOrderStatusPaid,
			TradeState:    "SUCCESS",
			TransactionID: "wx_tx_001",
			SuccessTime:   ptrTime(baseNow.Add(2 * time.Minute)),
		},
	}, WeChatPayConfig{
		Enabled:                     true,
		AppID:                       "wx-app-id",
		MerchantID:                  "mch-id",
		MerchantCertificateSerialNo: "serial-001",
		MerchantPrivateKeyPath:      "/tmp/mock.key",
		NotifyURL:                   "https://example.com/notify",
	})

	order, err := service.CreateOrder(context.Background(), "user-1", domain.VIPOrderCreateInput{
		BillingCycle: domain.VIPBillingCycleMonthly,
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}
	if order == nil || order.CodeURL != "weixin://pay/mock-qr" {
		t.Fatalf("order = %+v", order)
	}
	if order.Status != domain.VIPOrderStatusPendingPayment {
		t.Fatalf("order status = %q, want pending_payment", order.Status)
	}

	service.now = func() time.Time { return baseNow.Add(3 * time.Minute) }
	queriedOrder, err := service.GetOrder(context.Background(), "user-1", order.ID)
	if err != nil {
		t.Fatalf("GetOrder() error = %v", err)
	}
	if queriedOrder.Status != domain.VIPOrderStatusPaid {
		t.Fatalf("queried order status = %q, want paid", queriedOrder.Status)
	}

	membership, err := service.GetMembership(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("GetMembership() error = %v", err)
	}
	if membership == nil || !membership.IsVIP {
		t.Fatalf("membership = %+v, want active", membership)
	}
}

func TestVIPServiceHandleWeChatNotifyIsIdempotent(t *testing.T) {
	repo := repository.NewMemoryVIPRepository()
	service := NewVIPService(repo)
	baseNow := time.Date(2026, time.April, 5, 12, 0, 0, 0, trading.TradingLocation())
	service.now = func() time.Time { return baseNow }
	service.SetWeChatPayClient(&mockWeChatPayClient{
		createResult: &WeChatNativeOrderResult{CodeURL: "weixin://pay/mock-qr"},
		notifyResult: &WeChatPaymentNotifyResult{
			OrderNo:       "vip_manual",
			TransactionID: "wx_tx_notify",
			Status:        domain.VIPOrderStatusPaid,
			TradeState:    "SUCCESS",
			SuccessTime:   ptrTime(baseNow.Add(time.Minute)),
			NotifyID:      "notify-001",
			RawPayload:    `{"trade_state":"SUCCESS"}`,
		},
	}, WeChatPayConfig{
		Enabled:                     true,
		APIV3Key:                    "12345678901234567890123456789012",
		PlatformPublicKeyPath:       "/tmp/mock.pub",
		PlatformSerialNo:            "platform-001",
		MerchantID:                  "mch-id",
		MerchantPrivateKeyPath:      "/tmp/mock.key",
		NotifyURL:                   "https://example.com/notify",
		AppID:                       "wx-app-id",
		MerchantCertificateSerialNo: "serial-001",
	})

	now := baseNow
	order := &domain.VIPOrder{
		ID:             "order-1",
		UserID:         "user-1",
		OrderNo:        "vip_manual",
		PlanCode:       domain.VIPPlanCode,
		PlanName:       domain.VIPPlanName,
		BillingCycle:   domain.VIPBillingCycleMonthly,
		AmountFen:      3900,
		Currency:       "CNY",
		Status:         domain.VIPOrderStatusPendingPayment,
		PaymentChannel: domain.VIPPaymentChannelWeChatPay,
		PaymentScene:   domain.VIPPaymentSceneNative,
		Description:    "FundLive VIP 月度会员",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := repo.SaveOrder(context.Background(), order); err != nil {
		t.Fatalf("SaveOrder() error = %v", err)
	}

	if err := service.HandleWeChatPayNotify(context.Background(), map[string]string{}, []byte(`{}`)); err != nil {
		t.Fatalf("HandleWeChatPayNotify() error = %v", err)
	}

	firstMembership, err := repo.GetMembership(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("GetMembership() error = %v", err)
	}
	if firstMembership == nil {
		t.Fatalf("expected membership after first notify")
	}
	firstExpiresAt := firstMembership.ExpiresAt

	service.now = func() time.Time { return baseNow.Add(2 * time.Minute) }
	if err := service.HandleWeChatPayNotify(context.Background(), map[string]string{}, []byte(`{}`)); err != nil {
		t.Fatalf("HandleWeChatPayNotify() second call error = %v", err)
	}

	secondMembership, err := repo.GetMembership(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("GetMembership() error = %v", err)
	}
	if !secondMembership.ExpiresAt.Equal(firstExpiresAt) {
		t.Fatalf("membership expiry changed on duplicate notify: first=%s second=%s", firstExpiresAt, secondMembership.ExpiresAt)
	}
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
