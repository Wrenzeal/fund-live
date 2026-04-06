package domain

import (
	"context"
	"time"
)

type VIPBillingCycle string

const (
	VIPBillingCycleMonthly VIPBillingCycle = "monthly"
	VIPBillingCycleYearly  VIPBillingCycle = "yearly"
)

type VIPTaskType string

const (
	VIPTaskTypeSectorAnalysis    VIPTaskType = "sector_analysis"
	VIPTaskTypePortfolioAnalysis VIPTaskType = "portfolio_analysis"
)

type VIPTargetType string

const (
	VIPTargetTypeWatchlistGroup VIPTargetType = "watchlist_group"
	VIPTargetTypeWatchlistAll   VIPTargetType = "watchlist_all"
	VIPTargetTypeHoldingsAll    VIPTargetType = "holdings_all"
)

type VIPTaskStatus string

const (
	VIPTaskStatusQueued    VIPTaskStatus = "queued"
	VIPTaskStatusRunning   VIPTaskStatus = "running"
	VIPTaskStatusCompleted VIPTaskStatus = "completed"
	VIPTaskStatusFailed    VIPTaskStatus = "failed"
)

type VIPRiskLevel string

const (
	VIPRiskLevelLow    VIPRiskLevel = "low"
	VIPRiskLevelMedium VIPRiskLevel = "medium"
	VIPRiskLevelHigh   VIPRiskLevel = "high"
)

type VIPPaymentChannel string

const (
	VIPPaymentChannelWeChatPay VIPPaymentChannel = "wechat_pay"
)

type VIPPaymentScene string

const (
	VIPPaymentSceneNative VIPPaymentScene = "native"
)

type VIPOrderStatus string

const (
	VIPOrderStatusPendingPayment VIPOrderStatus = "pending_payment"
	VIPOrderStatusPaid           VIPOrderStatus = "paid"
	VIPOrderStatusClosed         VIPOrderStatus = "closed"
	VIPOrderStatusFailed         VIPOrderStatus = "failed"
)

type VIPSourceType string

const (
	VIPSourceTypeNews     VIPSourceType = "news"
	VIPSourceTypePolicy   VIPSourceType = "policy"
	VIPSourceTypeEarnings VIPSourceType = "earnings"
	VIPSourceTypeMarket   VIPSourceType = "market"
)

const (
	VIPPlanCode                    = "vip"
	VIPPlanName                    = "FundLive VIP"
	VIPDailySectorAnalysisLimit    = 2
	VIPDailyPortfolioAnalysisLimit = 2
)

type VIPMembership struct {
	ID           string
	UserID       string
	PlanCode     string
	PlanName     string
	BillingCycle VIPBillingCycle
	ActivatedAt  time.Time
	ExpiresAt    time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (m *VIPMembership) IsActive(now time.Time) bool {
	if m == nil {
		return false
	}
	return now.Before(m.ExpiresAt)
}

type VIPMembershipState struct {
	IsVIP        bool            `json:"is_vip"`
	PlanCode     string          `json:"plan_code"`
	PlanName     string          `json:"plan_name"`
	BillingCycle VIPBillingCycle `json:"billing_cycle"`
	ActivatedAt  string          `json:"activated_at"`
	ExpiresAt    string          `json:"expires_at"`
}

type VIPDailyUsage struct {
	UserID                string
	UsageDate             string
	SectorAnalysisUsed    int
	PortfolioAnalysisUsed int
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type VIPQuotaStatus struct {
	UsageDate                  string `json:"usage_date"`
	SectorAnalysisLimit        int    `json:"sector_analysis_limit"`
	SectorAnalysisUsed         int    `json:"sector_analysis_used"`
	SectorAnalysisRemaining    int    `json:"sector_analysis_remaining"`
	PortfolioAnalysisLimit     int    `json:"portfolio_analysis_limit"`
	PortfolioAnalysisUsed      int    `json:"portfolio_analysis_used"`
	PortfolioAnalysisRemaining int    `json:"portfolio_analysis_remaining"`
}

type VIPTaskRecord struct {
	ID               string
	UserID           string
	Type             VIPTaskType
	TargetType       VIPTargetType
	TargetID         string
	TargetName       string
	Status           VIPTaskStatus
	ProgressText     string
	TemplateReportID string
	ReportID         string
	CreatedAt        time.Time
	StartedAt        *time.Time
	CompletedAt      *time.Time
	FailedAt         *time.Time
	UpdatedAt        time.Time
}

type VIPTaskView struct {
	ID           string        `json:"id"`
	Type         VIPTaskType   `json:"type"`
	TargetType   VIPTargetType `json:"target_type"`
	TargetID     string        `json:"target_id"`
	TargetName   string        `json:"target_name"`
	CreatedAt    string        `json:"created_at"`
	Status       VIPTaskStatus `json:"status"`
	StartedAt    string        `json:"started_at,omitempty"`
	CompletedAt  string        `json:"completed_at,omitempty"`
	ProgressText string        `json:"progress_text"`
	ReportID     string        `json:"report_id,omitempty"`
}

type VIPReportSource struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Type        VIPSourceType `json:"type"`
	Publisher   string        `json:"publisher"`
	PublishedAt string        `json:"published_at"`
	URL         string        `json:"url"`
	Snippet     string        `json:"snippet"`
}

type VIPAdvice struct {
	Action        string   `json:"action"`
	PositionRange string   `json:"position_range"`
	Conditions    []string `json:"conditions"`
}

type VIPNarrativeSection struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Bullets []string `json:"bullets"`
}

type VIPEarningsCompany struct {
	Name string `json:"name"`
	Note string `json:"note"`
}

type VIPEarningsSection struct {
	Title     string               `json:"title"`
	Companies []VIPEarningsCompany `json:"companies"`
}

type VIPReportSummary struct {
	Headline string   `json:"headline"`
	Bullets  []string `json:"bullets"`
}

type VIPReport struct {
	ID               string              `json:"id"`
	Type             VIPTaskType         `json:"type"`
	Title            string              `json:"title"`
	TargetName       string              `json:"target_name"`
	GeneratedAt      string              `json:"generated_at"`
	CoverageWindow   string              `json:"coverage_window"`
	RiskLevel        VIPRiskLevel        `json:"riskLevel"`
	Summary          VIPReportSummary    `json:"summary"`
	Advice           VIPAdvice           `json:"advice"`
	Macro            VIPNarrativeSection `json:"macro"`
	Policy           VIPNarrativeSection `json:"policy"`
	Earnings         VIPEarningsSection  `json:"earnings"`
	Market           VIPNarrativeSection `json:"market"`
	Risks            []string            `json:"risks"`
	Sources          []VIPReportSource   `json:"sources"`
	FooterDisclaimer string              `json:"footerDisclaimer"`
}

type VIPStoredReport struct {
	ID          string
	UserID      string
	TaskID      string
	PayloadJSON string
	GeneratedAt time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Sources     []VIPReportSource
}

type VIPTaskCreateInput struct {
	Type       VIPTaskType   `json:"type"`
	TargetType VIPTargetType `json:"target_type"`
	TargetID   string        `json:"target_id"`
	TargetName string        `json:"target_name"`
}

type VIPTaskCreateResult struct {
	TaskID string `json:"task_id"`
}

type VIPOrder struct {
	ID                  string
	UserID              string
	OrderNo             string
	PlanCode            string
	PlanName            string
	BillingCycle        VIPBillingCycle
	AmountFen           int64
	Currency            string
	Status              VIPOrderStatus
	PaymentChannel      VIPPaymentChannel
	PaymentScene        VIPPaymentScene
	Description         string
	CodeURL             string
	WechatTransactionID string
	WechatPrepayID      string
	ErrorCode           string
	ErrorMessage        string
	NotifyID            string
	NotifyPayload       string
	ExpiresAt           *time.Time
	PaidAt              *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type VIPOrderView struct {
	ID                  string            `json:"id"`
	OrderNo             string            `json:"order_no"`
	PlanCode            string            `json:"plan_code"`
	PlanName            string            `json:"plan_name"`
	BillingCycle        VIPBillingCycle   `json:"billing_cycle"`
	AmountFen           int64             `json:"amount_fen"`
	Currency            string            `json:"currency"`
	Status              VIPOrderStatus    `json:"status"`
	PaymentChannel      VIPPaymentChannel `json:"payment_channel"`
	PaymentScene        VIPPaymentScene   `json:"payment_scene"`
	Description         string            `json:"description"`
	CodeURL             string            `json:"code_url,omitempty"`
	WechatTransactionID string            `json:"wechat_transaction_id,omitempty"`
	ErrorCode           string            `json:"error_code,omitempty"`
	ErrorMessage        string            `json:"error_message,omitempty"`
	ExpiresAt           string            `json:"expires_at,omitempty"`
	PaidAt              string            `json:"paid_at,omitempty"`
	CreatedAt           string            `json:"created_at"`
	UpdatedAt           string            `json:"updated_at"`
}

type VIPOrderCreateInput struct {
	BillingCycle VIPBillingCycle `json:"billing_cycle"`
}

type VIPRepository interface {
	GetMembership(ctx context.Context, userID string) (*VIPMembership, error)
	SaveMembership(ctx context.Context, membership *VIPMembership) error
	DeleteMembership(ctx context.Context, userID string) error
	GetDailyUsage(ctx context.Context, userID, usageDate string) (*VIPDailyUsage, error)
	SaveDailyUsage(ctx context.Context, usage *VIPDailyUsage) error
	DeleteDailyUsages(ctx context.Context, userID string) error
	SaveTask(ctx context.Context, task *VIPTaskRecord) error
	ListTasks(ctx context.Context, userID string) ([]VIPTaskRecord, error)
	DeleteTasks(ctx context.Context, userID string) error
	SaveReport(ctx context.Context, report *VIPStoredReport) error
	GetReportByID(ctx context.Context, reportID string) (*VIPStoredReport, error)
	DeleteReports(ctx context.Context, userID string) error
	SaveOrder(ctx context.Context, order *VIPOrder) error
	GetOrderByID(ctx context.Context, orderID string) (*VIPOrder, error)
	GetOrderByOrderNo(ctx context.Context, orderNo string) (*VIPOrder, error)
	DeleteOrders(ctx context.Context, userID string) error
}

type VIPService interface {
	GetMembership(ctx context.Context, userID string) (*VIPMembershipState, error)
	ActivatePreviewMembership(ctx context.Context, userID string, cycle VIPBillingCycle) (*VIPMembershipState, error)
	ResetPreview(ctx context.Context, userID string) error
	GetQuota(ctx context.Context, userID string) (*VIPQuotaStatus, error)
	CreateTask(ctx context.Context, userID string, input VIPTaskCreateInput) (*VIPTaskCreateResult, error)
	ListTasks(ctx context.Context, userID string) ([]VIPTaskView, error)
	GetReportByID(ctx context.Context, userID string, reportID string) (*VIPReport, error)
	CreateOrder(ctx context.Context, userID string, input VIPOrderCreateInput) (*VIPOrderView, error)
	GetOrder(ctx context.Context, userID, orderID string) (*VIPOrderView, error)
	HandleWeChatPayNotify(ctx context.Context, headers map[string]string, body []byte) error
}
