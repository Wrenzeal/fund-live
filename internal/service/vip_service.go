package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/RomaticDOG/fund/internal/trading"
)

var (
	ErrVIPMembershipRequired   = errors.New("vip membership required")
	ErrVIPQuotaExceeded        = errors.New("vip quota exhausted")
	ErrVIPInvalidBillingCycle  = errors.New("invalid vip billing cycle")
	ErrVIPInvalidTaskInput     = errors.New("invalid vip task input")
	ErrVIPReportNotFound       = errors.New("vip report not found")
	ErrVIPOrderNotFound        = errors.New("vip order not found")
	ErrVIPPaymentNotConfigured = errors.New("vip payment is not configured")
)

type VIPServiceImpl struct {
	repo            domain.VIPRepository
	now             func() time.Time
	wechatPay       WeChatPayClient
	wechatPayConfig WeChatPayConfig
}

func NewVIPService(repo domain.VIPRepository) *VIPServiceImpl {
	return &VIPServiceImpl{
		repo: repo,
		now:  time.Now,
	}
}

func (s *VIPServiceImpl) SetWeChatPayClient(client WeChatPayClient, config WeChatPayConfig) {
	s.wechatPay = client
	s.wechatPayConfig = config
}

func (s *VIPServiceImpl) GetMembership(ctx context.Context, userID string) (*domain.VIPMembershipState, error) {
	membership, err := s.repo.GetMembership(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.toMembershipState(membership), nil
}

func (s *VIPServiceImpl) ActivatePreviewMembership(ctx context.Context, userID string, cycle domain.VIPBillingCycle) (*domain.VIPMembershipState, error) {
	cycle = normalizeVIPBillingCycle(cycle)
	if cycle == "" {
		return nil, ErrVIPInvalidBillingCycle
	}

	now := s.now().In(tradingLocation())
	expiresAt := now.Add(30 * 24 * time.Hour)
	if cycle == domain.VIPBillingCycleYearly {
		expiresAt = now.Add(365 * 24 * time.Hour)
	}

	existing, err := s.repo.GetMembership(ctx, userID)
	if err != nil {
		return nil, err
	}

	membership := &domain.VIPMembership{
		ID:           generateID("vipm"),
		UserID:       userID,
		PlanCode:     domain.VIPPlanCode,
		PlanName:     domain.VIPPlanName,
		BillingCycle: cycle,
		ActivatedAt:  now,
		ExpiresAt:    expiresAt,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if existing != nil {
		membership.ID = existing.ID
		membership.CreatedAt = existing.CreatedAt
	}

	if err := s.repo.SaveMembership(ctx, membership); err != nil {
		return nil, err
	}

	return s.toMembershipState(membership), nil
}

func (s *VIPServiceImpl) ResetPreview(ctx context.Context, userID string) error {
	if err := s.repo.DeleteOrders(ctx, userID); err != nil {
		return err
	}
	if err := s.repo.DeleteTasks(ctx, userID); err != nil {
		return err
	}
	if err := s.repo.DeleteReports(ctx, userID); err != nil {
		return err
	}
	if err := s.repo.DeleteDailyUsages(ctx, userID); err != nil {
		return err
	}
	if err := s.repo.DeleteMembership(ctx, userID); err != nil {
		return err
	}
	return nil
}

func (s *VIPServiceImpl) CreateOrder(ctx context.Context, userID string, input domain.VIPOrderCreateInput) (*domain.VIPOrderView, error) {
	cycle := normalizeVIPBillingCycle(input.BillingCycle)
	if cycle == "" {
		return nil, ErrVIPInvalidBillingCycle
	}
	if s.wechatPay == nil || !s.wechatPayConfig.IsCreateConfigured() {
		return nil, ErrVIPPaymentNotConfigured
	}

	now := s.now().In(tradingLocation())
	order := buildVIPOrder(userID, cycle, now)

	createResult, err := s.wechatPay.CreateNativeOrder(ctx, WeChatNativeOrderInput{
		OutTradeNo:  order.OrderNo,
		Description: order.Description,
		AmountFen:   order.AmountFen,
		Currency:    order.Currency,
		ExpiresAt:   *order.ExpiresAt,
	})
	if err != nil {
		order.Status = domain.VIPOrderStatusFailed
		order.ErrorMessage = err.Error()
		order.UpdatedAt = now
		if saveErr := s.repo.SaveOrder(ctx, order); saveErr != nil {
			return nil, saveErr
		}
		return nil, err
	}

	order.CodeURL = createResult.CodeURL
	order.Status = domain.VIPOrderStatusPendingPayment
	order.UpdatedAt = now
	if err := s.repo.SaveOrder(ctx, order); err != nil {
		return nil, err
	}

	return s.toOrderView(order), nil
}

func (s *VIPServiceImpl) GetOrder(ctx context.Context, userID, orderID string) (*domain.VIPOrderView, error) {
	record, err := s.repo.GetOrderByID(ctx, strings.TrimSpace(orderID))
	if err != nil {
		return nil, err
	}
	if record == nil || record.UserID != strings.TrimSpace(userID) {
		return nil, ErrVIPOrderNotFound
	}

	now := s.now().In(tradingLocation())
	if record.Status == domain.VIPOrderStatusPendingPayment && s.wechatPay != nil && s.wechatPayConfig.IsQueryConfigured() {
		if result, queryErr := s.wechatPay.QueryOrderByOutTradeNo(ctx, record.OrderNo); queryErr == nil && result != nil {
			previousStatus := record.Status
			if result.Status != record.Status || result.TransactionID != record.WechatTransactionID {
				record.Status = result.Status
				record.WechatTransactionID = result.TransactionID
				record.PaidAt = result.SuccessTime
				record.UpdatedAt = now
				if saveErr := s.repo.SaveOrder(ctx, record); saveErr != nil {
					return nil, saveErr
				}
			}
			if previousStatus != domain.VIPOrderStatusPaid && record.Status == domain.VIPOrderStatusPaid {
				if err := s.activatePaidMembershipForOrder(ctx, record, result.SuccessTime); err != nil {
					return nil, err
				}
			}
		}
	}

	return s.toOrderView(record), nil
}

func (s *VIPServiceImpl) HandleWeChatPayNotify(ctx context.Context, headers map[string]string, body []byte) error {
	if s.wechatPay == nil || !s.wechatPayConfig.IsNotifyConfigured() {
		return ErrVIPPaymentNotConfigured
	}

	notify, err := s.wechatPay.ParsePaymentNotify(headers, body)
	if err != nil {
		return err
	}

	order, err := s.repo.GetOrderByOrderNo(ctx, notify.OrderNo)
	if err != nil {
		return err
	}
	if order == nil {
		return ErrVIPOrderNotFound
	}
	alreadyPaid := order.Status == domain.VIPOrderStatusPaid && order.WechatTransactionID != "" && order.WechatTransactionID == notify.TransactionID

	order.NotifyID = notify.NotifyID
	order.NotifyPayload = notify.RawPayload
	order.WechatTransactionID = firstNonEmpty(notify.TransactionID, order.WechatTransactionID)
	order.PaidAt = notify.SuccessTime
	order.Status = notify.Status
	order.UpdatedAt = s.now().In(tradingLocation())
	if err := s.repo.SaveOrder(ctx, order); err != nil {
		return err
	}

	if notify.Status == domain.VIPOrderStatusPaid && !alreadyPaid {
		if err := s.activatePaidMembershipForOrder(ctx, order, notify.SuccessTime); err != nil {
			return err
		}
	}

	return nil
}

func (s *VIPServiceImpl) GetQuota(ctx context.Context, userID string) (*domain.VIPQuotaStatus, error) {
	now := s.now().In(tradingLocation())
	usageDate := now.Format("2006-01-02")

	membership, err := s.repo.GetMembership(ctx, userID)
	if err != nil {
		return nil, err
	}
	usage, err := s.repo.GetDailyUsage(ctx, userID, usageDate)
	if err != nil {
		return nil, err
	}

	sectorUsed := 0
	portfolioUsed := 0
	if usage != nil {
		sectorUsed = usage.SectorAnalysisUsed
		portfolioUsed = usage.PortfolioAnalysisUsed
	}

	sectorLimit := 0
	portfolioLimit := 0
	if membership != nil && membership.IsActive(now) {
		sectorLimit = domain.VIPDailySectorAnalysisLimit
		portfolioLimit = domain.VIPDailyPortfolioAnalysisLimit
	}

	return &domain.VIPQuotaStatus{
		UsageDate:                  usageDate,
		SectorAnalysisLimit:        sectorLimit,
		SectorAnalysisUsed:         sectorUsed,
		SectorAnalysisRemaining:    max(0, sectorLimit-sectorUsed),
		PortfolioAnalysisLimit:     portfolioLimit,
		PortfolioAnalysisUsed:      portfolioUsed,
		PortfolioAnalysisRemaining: max(0, portfolioLimit-portfolioUsed),
	}, nil
}

func (s *VIPServiceImpl) CreateTask(ctx context.Context, userID string, input domain.VIPTaskCreateInput) (*domain.VIPTaskCreateResult, error) {
	taskType := normalizeVIPTaskType(input.Type)
	targetType := normalizeVIPTargetType(input.TargetType)
	targetID := strings.TrimSpace(input.TargetID)
	targetName := strings.TrimSpace(input.TargetName)
	if taskType == "" || targetType == "" || targetID == "" || targetName == "" {
		return nil, ErrVIPInvalidTaskInput
	}

	now := s.now().In(tradingLocation())
	membership, err := s.repo.GetMembership(ctx, userID)
	if err != nil {
		return nil, err
	}
	if membership == nil || !membership.IsActive(now) {
		return nil, ErrVIPMembershipRequired
	}

	quota, err := s.GetQuota(ctx, userID)
	if err != nil {
		return nil, err
	}
	switch taskType {
	case domain.VIPTaskTypeSectorAnalysis:
		if quota.SectorAnalysisRemaining <= 0 {
			return nil, ErrVIPQuotaExceeded
		}
	case domain.VIPTaskTypePortfolioAnalysis:
		if quota.PortfolioAnalysisRemaining <= 0 {
			return nil, ErrVIPQuotaExceeded
		}
	}

	usage, err := s.repo.GetDailyUsage(ctx, userID, quota.UsageDate)
	if err != nil {
		return nil, err
	}
	if usage == nil {
		usage = &domain.VIPDailyUsage{
			UserID:    userID,
			UsageDate: quota.UsageDate,
			CreatedAt: now,
			UpdatedAt: now,
		}
	}
	switch taskType {
	case domain.VIPTaskTypeSectorAnalysis:
		usage.SectorAnalysisUsed++
	case domain.VIPTaskTypePortfolioAnalysis:
		usage.PortfolioAnalysisUsed++
	}
	usage.UpdatedAt = now
	if err := s.repo.SaveDailyUsage(ctx, usage); err != nil {
		return nil, err
	}

	templateID := resolveVIPTemplateID(taskType, targetName)
	reportTemplate, ok := cloneVIPReportTemplate(templateID)
	if !ok {
		return nil, fmt.Errorf("vip report template not found: %s", templateID)
	}

	taskID := generateID("vipt")
	reportID := generateID("vipr")
	startedAt := now.Add(3 * time.Second)
	completedAt := now.Add(9 * time.Second)

	report := overrideVIPReportTemplate(reportTemplate, reportID, targetName, completedAt)
	payload, err := json.Marshal(report)
	if err != nil {
		return nil, fmt.Errorf("marshal vip report: %w", err)
	}

	if err := s.repo.SaveReport(ctx, &domain.VIPStoredReport{
		ID:          reportID,
		UserID:      userID,
		TaskID:      taskID,
		PayloadJSON: string(payload),
		GeneratedAt: completedAt,
		CreatedAt:   now,
		UpdatedAt:   now,
		Sources:     append([]domain.VIPReportSource(nil), report.Sources...),
	}); err != nil {
		return nil, err
	}

	if err := s.repo.SaveTask(ctx, &domain.VIPTaskRecord{
		ID:               taskID,
		UserID:           userID,
		Type:             taskType,
		TargetType:       targetType,
		TargetID:         targetID,
		TargetName:       targetName,
		Status:           domain.VIPTaskStatusQueued,
		ProgressText:     queuedVIPProgressText,
		TemplateReportID: templateID,
		ReportID:         reportID,
		CreatedAt:        now,
		StartedAt:        &startedAt,
		CompletedAt:      &completedAt,
		UpdatedAt:        now,
	}); err != nil {
		return nil, err
	}

	return &domain.VIPTaskCreateResult{TaskID: taskID}, nil
}

func (s *VIPServiceImpl) ListTasks(ctx context.Context, userID string) ([]domain.VIPTaskView, error) {
	records, err := s.repo.ListTasks(ctx, userID)
	if err != nil {
		return nil, err
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].CreatedAt.After(records[j].CreatedAt)
	})

	now := s.now().In(tradingLocation())
	views := make([]domain.VIPTaskView, 0, len(records))
	for _, record := range records {
		views = append(views, deriveVIPTaskView(record, now))
	}
	return views, nil
}

func (s *VIPServiceImpl) GetReportByID(ctx context.Context, userID string, reportID string) (*domain.VIPReport, error) {
	reportID = strings.TrimSpace(reportID)
	if reportID == "" {
		return nil, ErrVIPReportNotFound
	}

	if template, ok := cloneVIPReportTemplate(reportID); ok {
		return template, nil
	}

	record, err := s.repo.GetReportByID(ctx, reportID)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, ErrVIPReportNotFound
	}
	if record.UserID != "" && strings.TrimSpace(userID) != record.UserID {
		return nil, ErrVIPReportNotFound
	}

	var report domain.VIPReport
	if err := json.Unmarshal([]byte(record.PayloadJSON), &report); err != nil {
		return nil, fmt.Errorf("unmarshal vip report: %w", err)
	}
	report.ID = record.ID
	if report.GeneratedAt == "" {
		report.GeneratedAt = record.GeneratedAt.In(tradingLocation()).Format(time.RFC3339)
	}
	if len(record.Sources) > 0 {
		report.Sources = append([]domain.VIPReportSource(nil), record.Sources...)
	}
	return &report, nil
}

const (
	queuedVIPProgressText    = "已提交，正在整理分析对象与数据上下文"
	runningVIPProgressText   = "正在整合宏观、政策、财报与市场走势信息"
	completedVIPProgressText = "报告已生成，可查看完整内容"
)

func deriveVIPTaskView(record domain.VIPTaskRecord, now time.Time) domain.VIPTaskView {
	status := record.Status
	progressText := record.ProgressText
	startedAt := ""
	completedAt := ""
	reportID := ""

	switch {
	case record.FailedAt != nil && !record.FailedAt.After(now):
		status = domain.VIPTaskStatusFailed
		if progressText == "" {
			progressText = "任务执行失败，请稍后重试。"
		}
	case record.CompletedAt != nil && !record.CompletedAt.After(now):
		status = domain.VIPTaskStatusCompleted
		progressText = completedVIPProgressText
		completedAt = record.CompletedAt.In(tradingLocation()).Format(time.RFC3339)
		if record.StartedAt != nil {
			startedAt = record.StartedAt.In(tradingLocation()).Format(time.RFC3339)
		}
		reportID = record.ReportID
	case record.StartedAt != nil && !record.StartedAt.After(now):
		status = domain.VIPTaskStatusRunning
		progressText = runningVIPProgressText
		startedAt = record.StartedAt.In(tradingLocation()).Format(time.RFC3339)
	default:
		status = domain.VIPTaskStatusQueued
		progressText = queuedVIPProgressText
	}

	return domain.VIPTaskView{
		ID:           record.ID,
		Type:         record.Type,
		TargetType:   record.TargetType,
		TargetID:     record.TargetID,
		TargetName:   record.TargetName,
		CreatedAt:    record.CreatedAt.In(tradingLocation()).Format(time.RFC3339),
		Status:       status,
		StartedAt:    startedAt,
		CompletedAt:  completedAt,
		ProgressText: progressText,
		ReportID:     reportID,
	}
}

func (s *VIPServiceImpl) toMembershipState(membership *domain.VIPMembership) *domain.VIPMembershipState {
	state := &domain.VIPMembershipState{
		IsVIP:        false,
		PlanCode:     domain.VIPPlanCode,
		PlanName:     domain.VIPPlanName,
		BillingCycle: domain.VIPBillingCycleMonthly,
		ActivatedAt:  "",
		ExpiresAt:    "",
	}

	if membership == nil {
		return state
	}

	state.PlanCode = membership.PlanCode
	state.PlanName = membership.PlanName
	state.BillingCycle = membership.BillingCycle
	state.ActivatedAt = membership.ActivatedAt.In(tradingLocation()).Format(time.RFC3339)
	state.ExpiresAt = membership.ExpiresAt.In(tradingLocation()).Format(time.RFC3339)
	state.IsVIP = membership.IsActive(s.now().In(tradingLocation()))
	if !state.IsVIP {
		state.ActivatedAt = ""
		state.ExpiresAt = ""
	}

	return state
}

func normalizeVIPBillingCycle(cycle domain.VIPBillingCycle) domain.VIPBillingCycle {
	switch domain.VIPBillingCycle(strings.TrimSpace(string(cycle))) {
	case domain.VIPBillingCycleMonthly:
		return domain.VIPBillingCycleMonthly
	case domain.VIPBillingCycleYearly:
		return domain.VIPBillingCycleYearly
	default:
		return ""
	}
}

func normalizeVIPTaskType(taskType domain.VIPTaskType) domain.VIPTaskType {
	switch domain.VIPTaskType(strings.TrimSpace(string(taskType))) {
	case domain.VIPTaskTypeSectorAnalysis:
		return domain.VIPTaskTypeSectorAnalysis
	case domain.VIPTaskTypePortfolioAnalysis:
		return domain.VIPTaskTypePortfolioAnalysis
	default:
		return ""
	}
}

func normalizeVIPTargetType(targetType domain.VIPTargetType) domain.VIPTargetType {
	switch domain.VIPTargetType(strings.TrimSpace(string(targetType))) {
	case domain.VIPTargetTypeWatchlistGroup:
		return domain.VIPTargetTypeWatchlistGroup
	case domain.VIPTargetTypeWatchlistAll:
		return domain.VIPTargetTypeWatchlistAll
	case domain.VIPTargetTypeHoldingsAll:
		return domain.VIPTargetTypeHoldingsAll
	default:
		return ""
	}
}

func tradingLocation() *time.Location {
	return trading.TradingLocation()
}

func buildVIPOrder(userID string, cycle domain.VIPBillingCycle, now time.Time) *domain.VIPOrder {
	amountFen := int64(3900)
	description := "FundLive VIP 月度会员"
	if cycle == domain.VIPBillingCycleYearly {
		amountFen = 39900
		description = "FundLive VIP 年度会员"
	}

	expiresAt := now.Add(30 * time.Minute)
	return &domain.VIPOrder{
		ID:             generateID("vipo"),
		UserID:         userID,
		OrderNo:        generateID("vip"),
		PlanCode:       domain.VIPPlanCode,
		PlanName:       domain.VIPPlanName,
		BillingCycle:   cycle,
		AmountFen:      amountFen,
		Currency:       "CNY",
		Status:         domain.VIPOrderStatusPendingPayment,
		PaymentChannel: domain.VIPPaymentChannelWeChatPay,
		PaymentScene:   domain.VIPPaymentSceneNative,
		Description:    description,
		ExpiresAt:      &expiresAt,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func (s *VIPServiceImpl) activatePaidMembershipForOrder(ctx context.Context, order *domain.VIPOrder, paidAt *time.Time) error {
	if order == nil {
		return nil
	}

	now := s.now().In(tradingLocation())
	activationTime := now
	if paidAt != nil && !paidAt.IsZero() {
		activationTime = paidAt.In(tradingLocation())
	}

	existing, err := s.repo.GetMembership(ctx, order.UserID)
	if err != nil {
		return err
	}

	startFrom := activationTime
	activatedAt := activationTime
	if existing != nil {
		if existing.IsActive(activationTime) && existing.ExpiresAt.After(startFrom) {
			startFrom = existing.ExpiresAt
		}
		if existing.IsActive(activationTime) && !existing.ActivatedAt.IsZero() {
			activatedAt = existing.ActivatedAt
		}
	}

	expiresAt := startFrom.Add(vipMembershipDuration(order.BillingCycle))
	membership := &domain.VIPMembership{
		ID:           generateID("vipm"),
		UserID:       order.UserID,
		PlanCode:     order.PlanCode,
		PlanName:     order.PlanName,
		BillingCycle: order.BillingCycle,
		ActivatedAt:  activatedAt,
		ExpiresAt:    expiresAt,
		CreatedAt:    activationTime,
		UpdatedAt:    activationTime,
	}
	if existing != nil {
		membership.ID = existing.ID
		membership.CreatedAt = existing.CreatedAt
	}

	return s.repo.SaveMembership(ctx, membership)
}

func vipMembershipDuration(cycle domain.VIPBillingCycle) time.Duration {
	if cycle == domain.VIPBillingCycleYearly {
		return 365 * 24 * time.Hour
	}
	return 30 * 24 * time.Hour
}

func (s *VIPServiceImpl) toOrderView(order *domain.VIPOrder) *domain.VIPOrderView {
	if order == nil {
		return nil
	}

	view := &domain.VIPOrderView{
		ID:                  order.ID,
		OrderNo:             order.OrderNo,
		PlanCode:            order.PlanCode,
		PlanName:            order.PlanName,
		BillingCycle:        order.BillingCycle,
		AmountFen:           order.AmountFen,
		Currency:            order.Currency,
		Status:              order.Status,
		PaymentChannel:      order.PaymentChannel,
		PaymentScene:        order.PaymentScene,
		Description:         order.Description,
		CodeURL:             order.CodeURL,
		WechatTransactionID: order.WechatTransactionID,
		ErrorCode:           order.ErrorCode,
		ErrorMessage:        order.ErrorMessage,
		CreatedAt:           order.CreatedAt.In(tradingLocation()).Format(time.RFC3339),
		UpdatedAt:           order.UpdatedAt.In(tradingLocation()).Format(time.RFC3339),
	}
	if order.ExpiresAt != nil && !order.ExpiresAt.IsZero() {
		view.ExpiresAt = order.ExpiresAt.In(tradingLocation()).Format(time.RFC3339)
	}
	if order.PaidAt != nil && !order.PaidAt.IsZero() {
		view.PaidAt = order.PaidAt.In(tradingLocation()).Format(time.RFC3339)
	}
	return view
}
