package database

import "time"

type UserMembership struct {
	ID           string    `gorm:"primaryKey;type:varchar(40)" json:"id"`
	UserID       string    `gorm:"type:varchar(40);uniqueIndex;not null" json:"user_id"`
	PlanCode     string    `gorm:"type:varchar(32);not null" json:"plan_code"`
	PlanName     string    `gorm:"type:varchar(100);not null" json:"plan_name"`
	BillingCycle string    `gorm:"type:varchar(16);not null" json:"billing_cycle"`
	ActivatedAt  time.Time `gorm:"type:timestamptz;not null" json:"activated_at"`
	ExpiresAt    time.Time `gorm:"type:timestamptz;index;not null" json:"expires_at"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (UserMembership) TableName() string {
	return "user_memberships"
}

type VIPUsageDaily struct {
	ID                    uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID                string    `gorm:"type:varchar(40);uniqueIndex:idx_vip_usage_daily_user_date,priority:1;index;not null" json:"user_id"`
	UsageDate             time.Time `gorm:"type:date;uniqueIndex:idx_vip_usage_daily_user_date,priority:2;not null" json:"usage_date"`
	SectorAnalysisUsed    int       `gorm:"default:0;not null" json:"sector_analysis_used"`
	PortfolioAnalysisUsed int       `gorm:"default:0;not null" json:"portfolio_analysis_used"`
	CreatedAt             time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt             time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (VIPUsageDaily) TableName() string {
	return "vip_usage_daily"
}

type AnalysisTask struct {
	ID               string     `gorm:"primaryKey;type:varchar(40)" json:"id"`
	UserID           string     `gorm:"type:varchar(40);index:idx_analysis_tasks_user_created,priority:1;not null" json:"user_id"`
	Type             string     `gorm:"type:varchar(32);index;not null" json:"type"`
	TargetType       string     `gorm:"type:varchar(32);not null" json:"target_type"`
	TargetID         string     `gorm:"type:varchar(80);not null" json:"target_id"`
	TargetName       string     `gorm:"type:varchar(255);not null" json:"target_name"`
	Status           string     `gorm:"type:varchar(16);index;not null" json:"status"`
	ProgressText     string     `gorm:"type:text" json:"progress_text"`
	TemplateReportID string     `gorm:"type:varchar(120)" json:"template_report_id"`
	ReportID         string     `gorm:"type:varchar(40);index" json:"report_id"`
	CreatedAt        time.Time  `gorm:"autoCreateTime;index:idx_analysis_tasks_user_created,priority:2" json:"created_at"`
	StartedAt        *time.Time `gorm:"type:timestamptz" json:"started_at,omitempty"`
	CompletedAt      *time.Time `gorm:"type:timestamptz" json:"completed_at,omitempty"`
	FailedAt         *time.Time `gorm:"type:timestamptz" json:"failed_at,omitempty"`
	UpdatedAt        time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (AnalysisTask) TableName() string {
	return "analysis_tasks"
}

type AnalysisReport struct {
	ID          string    `gorm:"primaryKey;type:varchar(40)" json:"id"`
	UserID      string    `gorm:"type:varchar(40);index;not null" json:"user_id"`
	TaskID      string    `gorm:"type:varchar(40);uniqueIndex" json:"task_id"`
	PayloadJSON string    `gorm:"type:text;not null" json:"payload_json"`
	GeneratedAt time.Time `gorm:"type:timestamptz;not null" json:"generated_at"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (AnalysisReport) TableName() string {
	return "analysis_reports"
}

type AnalysisReportSource struct {
	ID          string    `gorm:"primaryKey;type:varchar(80)" json:"id"`
	ReportID    string    `gorm:"type:varchar(40);index:idx_analysis_report_sources_report_id,not null" json:"report_id"`
	Title       string    `gorm:"type:varchar(255);not null" json:"title"`
	Type        string    `gorm:"type:varchar(32);not null" json:"type"`
	Publisher   string    `gorm:"type:varchar(255);not null" json:"publisher"`
	PublishedAt time.Time `gorm:"type:timestamptz;not null" json:"published_at"`
	URL         string    `gorm:"type:text;not null" json:"url"`
	Snippet     string    `gorm:"type:text;not null" json:"snippet"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (AnalysisReportSource) TableName() string {
	return "analysis_report_sources"
}

type VIPOrder struct {
	ID                  string     `gorm:"primaryKey;type:varchar(40)" json:"id"`
	UserID              string     `gorm:"type:varchar(40);index:idx_vip_orders_user_created,priority:1;not null" json:"user_id"`
	OrderNo             string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"order_no"`
	PlanCode            string     `gorm:"type:varchar(32);not null" json:"plan_code"`
	PlanName            string     `gorm:"type:varchar(100);not null" json:"plan_name"`
	BillingCycle        string     `gorm:"type:varchar(16);not null" json:"billing_cycle"`
	AmountFen           int64      `gorm:"not null" json:"amount_fen"`
	Currency            string     `gorm:"type:varchar(8);not null" json:"currency"`
	Status              string     `gorm:"type:varchar(24);index;not null" json:"status"`
	PaymentChannel      string     `gorm:"type:varchar(32);not null" json:"payment_channel"`
	PaymentScene        string     `gorm:"type:varchar(16);not null" json:"payment_scene"`
	Description         string     `gorm:"type:varchar(255);not null" json:"description"`
	CodeURL             string     `gorm:"type:text" json:"code_url"`
	WechatTransactionID string     `gorm:"type:varchar(64);index" json:"wechat_transaction_id"`
	WechatPrepayID      string     `gorm:"type:varchar(128)" json:"wechat_prepay_id"`
	ErrorCode           string     `gorm:"type:varchar(64)" json:"error_code"`
	ErrorMessage        string     `gorm:"type:text" json:"error_message"`
	NotifyID            string     `gorm:"type:varchar(64)" json:"notify_id"`
	NotifyPayload       string     `gorm:"type:text" json:"notify_payload"`
	ExpiresAt           *time.Time `gorm:"type:timestamptz;index" json:"expires_at,omitempty"`
	PaidAt              *time.Time `gorm:"type:timestamptz;index" json:"paid_at,omitempty"`
	CreatedAt           time.Time  `gorm:"autoCreateTime;index:idx_vip_orders_user_created,priority:2" json:"created_at"`
	UpdatedAt           time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (VIPOrder) TableName() string {
	return "vip_orders"
}
