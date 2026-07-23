package database

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

// QuantEvent identifies one logical event. Revisions live in QuantEventVersion
// so an as-of query never depends on a row that was overwritten later.
type QuantEvent struct {
	ID              string    `gorm:"primaryKey;type:char(64)" json:"id"`
	EventType       string    `gorm:"type:varchar(48);index;not null" json:"event_type"`
	TargetScope     string    `gorm:"type:varchar(24);index;not null" json:"target_scope"`
	CurrentVersion  int       `gorm:"not null;default:1" json:"current_version"`
	FirstKnownAt    time.Time `gorm:"index;not null" json:"first_known_at"`
	FirstIngestedAt time.Time `gorm:"not null" json:"first_ingested_at"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (QuantEvent) TableName() string { return "quant_events" }

type QuantEventVersion struct {
	ID                int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	EventID           string          `gorm:"type:char(64);uniqueIndex:uq_quant_event_version,priority:1;index;not null" json:"event_id"`
	Version           int             `gorm:"uniqueIndex:uq_quant_event_version,priority:2;not null" json:"version"`
	ContentHash       string          `gorm:"type:char(64);index;not null" json:"content_hash"`
	Status            string          `gorm:"type:varchar(20);index;not null" json:"status"`
	Title             string          `gorm:"type:text;not null" json:"title"`
	Summary           string          `gorm:"type:text" json:"summary"`
	Impact            string          `gorm:"type:varchar(16)" json:"impact"`
	Strength          string          `gorm:"type:varchar(16)" json:"strength"`
	Horizon           string          `gorm:"type:varchar(24)" json:"horizon"`
	ExpectedAt        *time.Time      `gorm:"index" json:"expected_at,omitempty"`
	AnnouncedAt       *time.Time      `gorm:"index" json:"announced_at,omitempty"`
	EffectiveAt       *time.Time      `gorm:"index" json:"effective_at,omitempty"`
	ExpiresAt         *time.Time      `gorm:"index" json:"expires_at,omitempty"`
	KnownAt           time.Time       `gorm:"index;not null" json:"known_at"`
	IngestedAt        time.Time       `gorm:"index;not null" json:"ingested_at"`
	KnownAtBasis      string          `gorm:"type:varchar(32);not null" json:"known_at_basis"`
	SourceTier        string          `gorm:"type:varchar(16);index;not null" json:"source_tier"`
	SourceName        string          `gorm:"type:varchar(120)" json:"source_name"`
	SourceURL         string          `gorm:"type:text" json:"source_url"`
	SourcePublishedAt *time.Time      `gorm:"index" json:"source_published_at,omitempty"`
	SourceConfidence  string          `gorm:"type:varchar(16)" json:"source_confidence"`
	RawPayload        json.RawMessage `gorm:"type:jsonb;serializer:json;not null" json:"raw_payload"`
	CreatedAt         time.Time       `gorm:"autoCreateTime" json:"created_at"`
}

func (QuantEventVersion) TableName() string { return "quant_event_versions" }

type QuantEventAsset struct {
	ID           int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	EventID      string          `gorm:"type:char(64);uniqueIndex:uq_quant_event_asset,priority:1;index;not null" json:"event_id"`
	AssetType    string          `gorm:"type:varchar(24);uniqueIndex:uq_quant_event_asset,priority:2;index;not null" json:"asset_type"`
	AssetCode    string          `gorm:"type:varchar(32);uniqueIndex:uq_quant_event_asset,priority:3;index;not null" json:"asset_code"`
	MappingBasis string          `gorm:"type:varchar(64)" json:"mapping_basis"`
	WeightHint   decimal.Decimal `gorm:"type:decimal(10,4)" json:"weight_hint"`
	CreatedAt    time.Time       `gorm:"autoCreateTime" json:"created_at"`
}

func (QuantEventAsset) TableName() string { return "quant_event_assets" }

type QuantInstrument struct {
	Symbol        string     `gorm:"primaryKey;type:varchar(32)" json:"symbol"`
	Name          string     `gorm:"type:varchar(120);not null" json:"name"`
	Exchange      string     `gorm:"type:varchar(16);index;not null" json:"exchange"`
	AssetClass    string     `gorm:"type:varchar(24);index;not null" json:"asset_class"`
	UniverseGroup string     `gorm:"type:varchar(32);index" json:"universe_group"`
	TrackingIndex string     `gorm:"type:varchar(32);index" json:"tracking_index"`
	ListedAt      *time.Time `gorm:"type:date" json:"listed_at,omitempty"`
	DelistedAt    *time.Time `gorm:"type:date" json:"delisted_at,omitempty"`
	Source        string     `gorm:"type:varchar(32);not null" json:"source"`
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (QuantInstrument) TableName() string { return "quant_instruments" }

type QuantUniverseMember struct {
	ID              int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UniverseVersion string    `gorm:"type:varchar(32);uniqueIndex:uq_quant_universe_member,priority:1;index;not null" json:"universe_version"`
	Symbol          string    `gorm:"type:varchar(32);uniqueIndex:uq_quant_universe_member,priority:2;index;not null" json:"symbol"`
	Bucket          string    `gorm:"type:varchar(32);index;not null" json:"bucket"`
	IncludedAt      time.Time `gorm:"not null" json:"included_at"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (QuantUniverseMember) TableName() string { return "quant_universe_members" }

type QuantMarketBar struct {
	ID            int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	Symbol        string          `gorm:"type:varchar(32);uniqueIndex:uq_quant_market_bar,priority:1;index;not null" json:"symbol"`
	Date          time.Time       `gorm:"type:date;uniqueIndex:uq_quant_market_bar,priority:2;index;not null" json:"date"`
	Open          decimal.Decimal `gorm:"type:decimal(18,6);not null" json:"open"`
	High          decimal.Decimal `gorm:"type:decimal(18,6);not null" json:"high"`
	Low           decimal.Decimal `gorm:"type:decimal(18,6);not null" json:"low"`
	Close         decimal.Decimal `gorm:"type:decimal(18,6);not null" json:"close"`
	AdjustedClose decimal.Decimal `gorm:"type:decimal(18,6);not null" json:"adjusted_close"`
	Volume        decimal.Decimal `gorm:"type:decimal(24,4);not null" json:"volume"`
	Amount        decimal.Decimal `gorm:"type:decimal(24,4);not null" json:"amount"`
	AdjustFactor  decimal.Decimal `gorm:"type:decimal(20,10);not null;default:1" json:"adjust_factor"`
	Suspended     bool            `gorm:"not null;default:false" json:"suspended"`
	Source        string          `gorm:"type:varchar(32);not null" json:"source"`
	IngestedAt    time.Time       `gorm:"index;not null" json:"ingested_at"`
	CreatedAt     time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time       `gorm:"autoUpdateTime" json:"updated_at"`
}

func (QuantMarketBar) TableName() string { return "quant_market_bars" }

type QuantCorporateAction struct {
	ID          int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	Symbol      string          `gorm:"type:varchar(32);uniqueIndex:uq_quant_corporate_action,priority:1;index;not null" json:"symbol"`
	ActionDate  time.Time       `gorm:"type:date;uniqueIndex:uq_quant_corporate_action,priority:2;index;not null" json:"action_date"`
	ActionType  string          `gorm:"type:varchar(24);uniqueIndex:uq_quant_corporate_action,priority:3;not null" json:"action_type"`
	CashAmount  decimal.Decimal `gorm:"type:decimal(18,6)" json:"cash_amount"`
	SplitFactor decimal.Decimal `gorm:"type:decimal(20,10)" json:"split_factor"`
	Source      string          `gorm:"type:varchar(32);not null" json:"source"`
	KnownAt     time.Time       `gorm:"index;not null" json:"known_at"`
	CreatedAt   time.Time       `gorm:"autoCreateTime" json:"created_at"`
}

func (QuantCorporateAction) TableName() string { return "quant_corporate_actions" }

type QuantTradingDay struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Market    string    `gorm:"type:varchar(16);uniqueIndex:uq_quant_trading_day,priority:1;index;not null" json:"market"`
	Date      time.Time `gorm:"type:date;uniqueIndex:uq_quant_trading_day,priority:2;index;not null" json:"date"`
	IsOpen    bool      `gorm:"index;not null" json:"is_open"`
	Reason    string    `gorm:"type:varchar(80)" json:"reason,omitempty"`
	Source    string    `gorm:"type:varchar(32);not null" json:"source"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (QuantTradingDay) TableName() string { return "quant_trading_calendar" }

type QuantSignalHistory struct {
	ID               int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	FundID           string          `gorm:"type:varchar(10);uniqueIndex:uq_quant_signal_day,priority:1;index;not null" json:"fund_id"`
	AnalysisVersion  string          `gorm:"type:varchar(32);uniqueIndex:uq_quant_signal_day,priority:2;not null" json:"analysis_version"`
	SignalDate       time.Time       `gorm:"type:date;uniqueIndex:uq_quant_signal_day,priority:3;index;not null" json:"signal_date"`
	Mode             string          `gorm:"type:varchar(32);uniqueIndex:uq_quant_signal_day,priority:4;index;not null" json:"mode"`
	DecisionAt       time.Time       `gorm:"index;not null" json:"decision_at"`
	DataCutoffAt     time.Time       `gorm:"index;not null" json:"data_cutoff_at"`
	TotalScore       decimal.Decimal `gorm:"type:decimal(8,4);not null" json:"total_score"`
	ShadowEventScore decimal.Decimal `gorm:"type:decimal(8,4)" json:"shadow_event_score"`
	InputJSON        json.RawMessage `gorm:"type:jsonb;serializer:json;not null" json:"input_json"`
	OutputJSON       json.RawMessage `gorm:"type:jsonb;serializer:json;not null" json:"output_json"`
	EventIDs         json.RawMessage `gorm:"type:jsonb;serializer:json;not null" json:"event_ids"`
	CreatedAt        time.Time       `gorm:"autoCreateTime" json:"created_at"`
}

func (QuantSignalHistory) TableName() string { return "quant_signal_history" }

type QuantBacktestJob struct {
	ID              string          `gorm:"primaryKey;type:char(32)" json:"id"`
	IdempotencyKey  string          `gorm:"type:char(64);uniqueIndex;not null" json:"-"`
	Status          string          `gorm:"type:varchar(20);index;not null" json:"status"`
	Strategy        string          `gorm:"type:varchar(48);not null" json:"strategy"`
	UniverseVersion string          `gorm:"type:varchar(32);index;not null" json:"universe_version"`
	SignalMode      string          `gorm:"type:varchar(32);index;not null" json:"signal_mode"`
	Engine          string          `gorm:"type:varchar(24);not null" json:"engine"`
	EngineVersion   string          `gorm:"type:varchar(80)" json:"engine_version"`
	ParametersJSON  json.RawMessage `gorm:"type:jsonb;serializer:json;not null" json:"parameters,omitempty"`
	MetricsJSON     json.RawMessage `gorm:"type:jsonb;serializer:json" json:"metrics,omitempty"`
	EquityJSON      json.RawMessage `gorm:"type:jsonb;serializer:json" json:"equity_curve,omitempty"`
	TradesJSON      json.RawMessage `gorm:"type:jsonb;serializer:json" json:"trades,omitempty"`
	BenchmarksJSON  json.RawMessage `gorm:"type:jsonb;serializer:json" json:"benchmarks,omitempty"`
	LogSummary      string          `gorm:"type:text" json:"-"`
	ErrorMessage    string          `gorm:"type:text" json:"error_message,omitempty"`
	AttemptCount    int             `gorm:"not null;default:0" json:"attempt_count"`
	CreatedAt       time.Time       `gorm:"autoCreateTime;index" json:"created_at"`
	StartedAt       *time.Time      `json:"started_at,omitempty"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty"`
	UpdatedAt       time.Time       `gorm:"autoUpdateTime" json:"updated_at"`
}

func (QuantBacktestJob) TableName() string { return "quant_backtest_jobs" }
