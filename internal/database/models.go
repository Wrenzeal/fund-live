// Package database 提供带有 GORM 标签的数据库模型。
// 这些模型与领域模型分离，以保持数据库关注点的隔离。
package database

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Fund 表示数据库中的基金实体。
// 对应数据库表: funds
type Fund struct {
	// ID 基金代码，如 "005827" (易方达蓝筹精选混合)
	ID string `gorm:"primaryKey;type:varchar(10)" json:"id"`

	// Name 基金简称，如 "易方达蓝筹"
	Name string `gorm:"type:varchar(100);index" json:"name"`

	// Type 基金类型，来自东方财富的分类，可能的值包括：
	//   - "股票型": 纯股票基金
	//   - "混合型-偏股": 混合型基金偏向股票
	//   - "混合型-偏债": 混合型基金偏向债券
	//   - "混合型-平衡": 混合型基金平衡配置
	//   - "混合型-灵活": 混合型基金灵活配置
	//   - "指数型-股票": 股票指数基金
	//   - "债券型-长债": 中长期债券基金
	//   - "债券型-短债": 短期债券基金
	//   - "货币型": 货币市场基金
	//   - "QDII-股票": 投资海外股票的基金
	//   - "FOF-进取": 基金中的基金-进取型
	//   - "联接基金": ETF联接基金，追踪对应的ETF
	Type string `gorm:"type:varchar(20);index" json:"type"`

	// Manager 基金经理姓名
	Manager string `gorm:"type:varchar(50)" json:"manager"`

	// Company 基金公司名称，如 "易方达基金"
	Company string `gorm:"type:varchar(100)" json:"company"`

	// NetAssetVal 最新单位净值 (NAV)
	NetAssetVal decimal.Decimal `gorm:"type:decimal(10,4)" json:"nav"`

	// TotalScale 基金规模（亿元）
	TotalScale decimal.Decimal `gorm:"type:decimal(15,4)" json:"scale"`

	// CreatedAt 记录创建时间
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// UpdatedAt 记录更新时间
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Holdings 基金持仓列表（外键关联）
	Holdings []StockHolding `gorm:"foreignKey:FundID" json:"holdings,omitempty"`
}

// TableName 指定 Fund 的表名。
func (Fund) TableName() string {
	return "funds"
}

// StockHolding 表示基金投资组合中的股票持仓。
// 对应数据库表: stock_holdings
type StockHolding struct {
	// ID 自增主键
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`

	// FundID 关联的基金代码
	FundID string `gorm:"type:varchar(10);index;not null" json:"fund_id"`

	// StockCode 股票代码，如 "600519" (贵州茅台)
	StockCode string `gorm:"type:varchar(10);index;not null" json:"stock_code"`

	// StockName 股票名称，如 "贵州茅台"
	StockName string `gorm:"type:varchar(50)" json:"stock_name"`

	// Exchange 交易所代码:
	//   - "SH": 上海证券交易所
	//   - "SZ": 深圳证券交易所
	//   - "BJ": 北京证券交易所 (北交所)
	Exchange string `gorm:"type:varchar(5)" json:"exchange"`

	// HoldingRatio 持仓占比（%），如 9.52 表示占基金净值的 9.52%
	HoldingRatio decimal.Decimal `gorm:"type:decimal(8,4)" json:"holding_ratio"`

	// HoldingShares 持股数量（股）
	HoldingShares decimal.Decimal `gorm:"type:decimal(18,2)" json:"holding_shares"`

	// MarketValue 持仓市值（元）
	MarketValue decimal.Decimal `gorm:"type:decimal(18,2)" json:"market_value"`

	// ReportingPeriod 报告期，如 "2024-Q4" 表示2024年第四季度
	ReportingPeriod string `gorm:"type:varchar(10);index" json:"reporting_period"`

	// CreatedAt 记录创建时间
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// UpdatedAt 记录更新时间
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 指定 StockHolding 的表名。
func (StockHolding) TableName() string {
	return "stock_holdings"
}

// FundTimeSeries 存储基金的盘中时间序列数据。
// 用于持久化存储，支持服务器重启后恢复数据和历史查询。
// 对应数据库表: fund_time_series
type FundTimeSeries struct {
	// ID 自增主键
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`

	// FundID 基金代码
	FundID string `gorm:"type:varchar(10);index:idx_fund_time,priority:1;not null" json:"fund_id"`

	// Date 交易日期
	Date time.Time `gorm:"type:date;index:idx_fund_time,priority:2;not null" json:"date"`

	// Time 采集时间点
	Time time.Time `gorm:"index:idx_fund_time,priority:3;not null" json:"time"`

	// ChangePercent 该时间点的预估涨跌幅（%）
	ChangePercent decimal.Decimal `gorm:"type:decimal(8,4)" json:"change_percent"`

	// EstimateNav 该时间点的预估净值
	EstimateNav decimal.Decimal `gorm:"type:decimal(10,4)" json:"estimate_nav"`

	// CreatedAt 记录创建时间
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName 指定 FundTimeSeries 的表名。
func (FundTimeSeries) TableName() string {
	return "fund_time_series"
}

// FundHistory stores official end-of-day NAV snapshots and daily returns.
type FundHistory struct {
	ID          uint            `gorm:"primaryKey;autoIncrement" json:"id"`
	FundID      string          `gorm:"type:varchar(10);index:idx_fund_date,priority:1;not null" json:"fund_id"`
	Date        time.Time       `gorm:"type:date;index:idx_fund_date,priority:2;not null" json:"date"`
	NetAssetVal decimal.Decimal `gorm:"type:decimal(10,4)" json:"net_asset_val"`
	AccumVal    decimal.Decimal `gorm:"type:decimal(10,4)" json:"accum_val"`
	DailyReturn decimal.Decimal `gorm:"type:decimal(8,4)" json:"daily_return"`
	CreatedAt   time.Time       `gorm:"autoCreateTime" json:"created_at"`
}

func (FundHistory) TableName() string {
	return "fund_history"
}

// FundValuationProfile stores custom pricing profiles for funds that cannot be valued via stock holdings.
// Typical examples are commodity and futures funds.
type FundValuationProfile struct {
	FundID            string          `gorm:"primaryKey;type:varchar(10)" json:"fund_id"`
	PricingMethod     string          `gorm:"type:varchar(50);index;not null" json:"pricing_method"`
	QuoteSource       string          `gorm:"type:varchar(50);not null" json:"quote_source"`
	UnderlyingSymbol  string          `gorm:"type:varchar(50);not null" json:"underlying_symbol"`
	UnderlyingName    string          `gorm:"type:varchar(100)" json:"underlying_name"`
	EffectiveExposure decimal.Decimal `gorm:"type:decimal(10,4);default:1.0000" json:"effective_exposure"`
	Notes             string          `gorm:"type:text" json:"notes"`
	CreatedAt         time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time       `gorm:"autoUpdateTime" json:"updated_at"`
}

func (FundValuationProfile) TableName() string {
	return "fund_valuation_profiles"
}

// FundMapping 存储联接基金与其目标 ETF 之间的映射关系。
// 对应数据库表: fund_mappings
//
// 工作流程:
//  1. 当查询联接基金（如"华宝创业板人工智能ETF联接C"）的持仓时
//  2. 系统先检查此表是否有已解析的映射
//  3. 若有，直接使用目标 ETF 的持仓数据
//  4. 若无，通过东财搜索解析后保存映射关系
type FundMapping struct {
	// ID 自增主键
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`

	// FeederCode 联接基金代码（发起方）
	FeederCode string `gorm:"type:varchar(10);uniqueIndex;not null" json:"feeder_code"`

	// FeederName 联接基金名称
	FeederName string `gorm:"type:varchar(100)" json:"feeder_name"`

	// TargetCode 目标 ETF 代码（被追踪的 ETF）
	TargetCode string `gorm:"type:varchar(10);index" json:"target_code"`

	// TargetName 目标 ETF 名称
	TargetName string `gorm:"type:varchar(100)" json:"target_name"`

	// IsResolved 是否已成功解析
	//   true: 已找到目标 ETF
	//   false: 解析失败或尚未解析
	IsResolved bool `gorm:"default:false;index" json:"is_resolved"`

	// ResolvedAt 解析成功时间（可为空）
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`

	// ResolveError 解析失败时的错误信息
	ResolveError string `gorm:"type:text" json:"resolve_error,omitempty"`

	// CreatedAt 记录创建时间
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// UpdatedAt 记录更新时间
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 指定 FundMapping 的表名。
func (FundMapping) TableName() string {
	return "fund_mappings"
}

// AllModels 返回所有数据库模型，用于自动迁移。
func AllModels() []interface{} {
	models := []interface{}{
		&Fund{},
		&StockHolding{},
		&FundTimeSeries{},
		&FundHistory{},
		&FundValuationProfile{},
		&FundMapping{},
	}
	return append(models, UserModels()...)
}

// BeforeCreate 是 Fund 的 GORM 钩子函数，用于设置默认值。
func (f *Fund) BeforeCreate(tx *gorm.DB) error {
	if f.CreatedAt.IsZero() {
		f.CreatedAt = time.Now()
	}
	return nil
}
