package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/database"
	"github.com/RomaticDOG/fund/internal/domain"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	SectorSourceDirectHoldings    = "direct_holdings"
	SectorSourceTargetETFFallback = "target_etf_fallback"
	SectorSourceQDIIHoldings      = "qdii_holdings"
	ThemeSourceDirectHoldings     = "direct_holdings"
	ThemeSourceTargetETFFallback  = "target_etf_fallback"
	ThemeSourceQDIIHoldings       = "qdii_holdings"

	SectorConfidenceHigh   = "high"
	SectorConfidenceMedium = "medium"
	SectorConfidenceLow    = "low"
)

type FundSectorStore struct {
	db *gorm.DB
}

func NewFundSectorStore(db *gorm.DB) *FundSectorStore {
	return &FundSectorStore{db: db}
}

type InstrumentRef struct {
	Code     string
	Exchange domain.Exchange
}

type seededInstrumentSector struct {
	Code       string
	Exchange   string
	SectorCode string
}

type seededInstrumentTheme struct {
	Code      string
	Exchange  string
	ThemeCode string
	Weight    decimal.Decimal
}

type sectorAgg struct {
	code   string
	weight decimal.Decimal
}

type themeAgg struct {
	code   string
	weight decimal.Decimal
}

var defaultFundSectors = []database.FundSector{
	{Code: "semiconductor", Name: "半导体", Level: 1, SortOrder: 10, IsEnabled: true},
	{Code: "communications_equipment", Name: "通信设备", Level: 1, SortOrder: 15, IsEnabled: true},
	{Code: "consumer_electronics", Name: "消费电子", Level: 1, SortOrder: 20, IsEnabled: true},
	{Code: "internet_platform", Name: "互联网平台", Level: 1, SortOrder: 30, IsEnabled: true},
	{Code: "internet_ecommerce", Name: "互联网电商", Level: 1, SortOrder: 40, IsEnabled: true},
	{Code: "software_cloud", Name: "软件云计算", Level: 1, SortOrder: 50, IsEnabled: true},
	{Code: "streaming_media", Name: "流媒体娱乐", Level: 1, SortOrder: 60, IsEnabled: true},
	{Code: "new_energy_auto", Name: "新能源车", Level: 1, SortOrder: 70, IsEnabled: true},
	{Code: "banking", Name: "银行", Level: 1, SortOrder: 75, IsEnabled: true},
	{Code: "brokerage_finance", Name: "券商与非银金融", Level: 1, SortOrder: 77, IsEnabled: true},
	{Code: "insurance", Name: "保险", Level: 1, SortOrder: 79, IsEnabled: true},
	{Code: "liquor", Name: "白酒", Level: 1, SortOrder: 80, IsEnabled: true},
	{Code: "food_beverage", Name: "食品饮料", Level: 1, SortOrder: 85, IsEnabled: true},
	{Code: "media_advertising", Name: "传媒广告", Level: 1, SortOrder: 90, IsEnabled: true},
	{Code: "oil_gas_energy", Name: "石油能源", Level: 1, SortOrder: 100, IsEnabled: true},
	{Code: "healthcare_service", Name: "医疗服务", Level: 1, SortOrder: 110, IsEnabled: true},
	{Code: "machinery_equipment", Name: "机械设备", Level: 1, SortOrder: 115, IsEnabled: true},
	{Code: "real_estate_reits", Name: "地产REITs", Level: 1, SortOrder: 118, IsEnabled: true},
	{Code: "consumer_service", Name: "消费服务", Level: 1, SortOrder: 120, IsEnabled: true},
	{Code: "resources_cycle", Name: "资源周期", Level: 1, SortOrder: 130, IsEnabled: true},
	{Code: "utilities_power", Name: "公用事业", Level: 1, SortOrder: 140, IsEnabled: true},
	{Code: "agriculture", Name: "农业", Level: 1, SortOrder: 150, IsEnabled: true},
	{Code: "other_equity", Name: "未归类权益", Level: 1, SortOrder: 999, IsEnabled: true},
}

var defaultFundCategories = []database.FundCategory{
	{Code: "stock", Name: "股票型", Description: "以股票资产为主的基金", SortOrder: 10, IsEnabled: true},
	{Code: "hybrid", Name: "混合型", Description: "股票/债券灵活配置基金", SortOrder: 20, IsEnabled: true},
	{Code: "bond", Name: "债券型", Description: "以债券资产为主的基金", SortOrder: 30, IsEnabled: true},
	{Code: "index", Name: "指数型", Description: "指数跟踪基金", SortOrder: 40, IsEnabled: true},
	{Code: "money", Name: "货币型", Description: "货币市场基金", SortOrder: 50, IsEnabled: true},
	{Code: "qdii", Name: "QDII", Description: "投资海外市场的基金", SortOrder: 60, IsEnabled: true},
	{Code: "feeder", Name: "ETF联接", Description: "ETF 联接基金", SortOrder: 70, IsEnabled: true},
	{Code: "fof", Name: "FOF", Description: "基金中基金", SortOrder: 80, IsEnabled: true},
	{Code: "commodity", Name: "商品/期货", Description: "商品或期货主题基金", SortOrder: 90, IsEnabled: true},
	{Code: "other", Name: "其他", Description: "其他类型基金", SortOrder: 999, IsEnabled: true},
}

var defaultFundThemes = []database.FundTheme{
	{Code: "semiconductor_chip", Name: "半导体芯片", Level: 1, SortOrder: 5, IsEnabled: true},
	{Code: "ai_application", Name: "AI应用", Level: 1, SortOrder: 10, IsEnabled: true},
	{Code: "computing_power", Name: "算力", Level: 1, SortOrder: 20, IsEnabled: true},
	{Code: "cpo_optical_module", Name: "CPO/光模块", Level: 1, SortOrder: 30, IsEnabled: true},
	{Code: "platform_internet", Name: "平台互联网", Level: 1, SortOrder: 35, IsEnabled: true},
	{Code: "commercial_aerospace", Name: "商业航天", Level: 1, SortOrder: 40, IsEnabled: true},
	{Code: "satellite_internet", Name: "卫星互联网", Level: 1, SortOrder: 50, IsEnabled: true},
	{Code: "robotics", Name: "机器人", Level: 1, SortOrder: 60, IsEnabled: true},
	{Code: "data_infrastructure", Name: "数据基础设施", Level: 1, SortOrder: 70, IsEnabled: true},
	{Code: "military_electronics", Name: "军工电子", Level: 1, SortOrder: 80, IsEnabled: true},
	{Code: "healthcare", Name: "医疗健康", Level: 1, SortOrder: 85, IsEnabled: true},
	{Code: "low_altitude_economy", Name: "低空经济", Level: 1, SortOrder: 90, IsEnabled: true},
	{Code: "smart_driving", Name: "智能驾驶", Level: 1, SortOrder: 100, IsEnabled: true},
	{Code: "innovative_medicine", Name: "创新药", Level: 1, SortOrder: 110, IsEnabled: true},
	{Code: "energy_storage", Name: "储能", Level: 1, SortOrder: 120, IsEnabled: true},
	{Code: "gaming_entertainment", Name: "游戏娱乐", Level: 1, SortOrder: 130, IsEnabled: true},
	{Code: "consumption_upgrade", Name: "消费升级", Level: 1, SortOrder: 135, IsEnabled: true},
	{Code: "photovoltaic", Name: "光伏", Level: 1, SortOrder: 140, IsEnabled: true},
	{Code: "wind_power", Name: "风电", Level: 1, SortOrder: 150, IsEnabled: true},
	{Code: "power_grid", Name: "电网设备", Level: 1, SortOrder: 160, IsEnabled: true},
	{Code: "network_security", Name: "网络安全", Level: 1, SortOrder: 170, IsEnabled: true},
	{Code: "financials", Name: "金融", Level: 1, SortOrder: 175, IsEnabled: true},
	{Code: "dividend_value", Name: "高股息红利", Level: 1, SortOrder: 178, IsEnabled: true},
	{Code: "financial_it", Name: "金融科技", Level: 1, SortOrder: 180, IsEnabled: true},
	{Code: "biotech", Name: "生物科技", Level: 1, SortOrder: 190, IsEnabled: true},
	{Code: "communications_5g", Name: "5G通信", Level: 1, SortOrder: 200, IsEnabled: true},
	{Code: "resources_cycle", Name: "资源周期", Level: 1, SortOrder: 210, IsEnabled: true},
	{Code: "reits_real_estate", Name: "地产REITs", Level: 1, SortOrder: 220, IsEnabled: true},
	{Code: "other_theme", Name: "未归类主题", Level: 1, SortOrder: 999, IsEnabled: true},
}

var defaultInstrumentSectorMappings = []seededInstrumentSector{
	{Code: "600519", Exchange: "SH", SectorCode: "liquor"},
	{Code: "000858", Exchange: "SZ", SectorCode: "liquor"},
	{Code: "000568", Exchange: "SZ", SectorCode: "liquor"},
	{Code: "600809", Exchange: "SH", SectorCode: "liquor"},
	{Code: "002027", Exchange: "SZ", SectorCode: "media_advertising"},
	{Code: "00700", Exchange: "HK", SectorCode: "internet_platform"},
	{Code: "09988", Exchange: "HK", SectorCode: "internet_platform"},
	{Code: "09987", Exchange: "HK", SectorCode: "consumer_service"},
	{Code: "00883", Exchange: "HK", SectorCode: "oil_gas_energy"},
	{Code: "06618", Exchange: "HK", SectorCode: "healthcare_service"},

	{Code: "688012", Exchange: "SH", SectorCode: "semiconductor"},
	{Code: "002371", Exchange: "SZ", SectorCode: "semiconductor"},
	{Code: "688256", Exchange: "SH", SectorCode: "semiconductor"},
	{Code: "688981", Exchange: "SH", SectorCode: "semiconductor"},
	{Code: "688041", Exchange: "SH", SectorCode: "semiconductor"},
	{Code: "688072", Exchange: "SH", SectorCode: "semiconductor"},
	{Code: "300604", Exchange: "SZ", SectorCode: "semiconductor"},
	{Code: "688120", Exchange: "SH", SectorCode: "semiconductor"},
	{Code: "688361", Exchange: "SH", SectorCode: "semiconductor"},
	{Code: "300346", Exchange: "SZ", SectorCode: "semiconductor"},
	{Code: "603986", Exchange: "SH", SectorCode: "semiconductor"},
	{Code: "688525", Exchange: "SH", SectorCode: "semiconductor"},
	{Code: "301308", Exchange: "SZ", SectorCode: "semiconductor"},
	{Code: "688766", Exchange: "SH", SectorCode: "semiconductor"},
	{Code: "688031", Exchange: "SH", SectorCode: "software_cloud"},
	{Code: "001309", Exchange: "SZ", SectorCode: "semiconductor"},
	{Code: "300223", Exchange: "SZ", SectorCode: "semiconductor"},
	{Code: "688627", Exchange: "SH", SectorCode: "semiconductor"},
	{Code: "603929", Exchange: "SH", SectorCode: "semiconductor"},
	{Code: "300308", Exchange: "SZ", SectorCode: "communications_equipment"},
	{Code: "300502", Exchange: "SZ", SectorCode: "communications_equipment"},
	{Code: "300394", Exchange: "SZ", SectorCode: "communications_equipment"},
	{Code: "002281", Exchange: "SZ", SectorCode: "communications_equipment"},
	{Code: "000988", Exchange: "SZ", SectorCode: "communications_equipment"},
	{Code: "603083", Exchange: "SH", SectorCode: "communications_equipment"},
	{Code: "600030", Exchange: "SH", SectorCode: "brokerage_finance"},
	{Code: "601688", Exchange: "SH", SectorCode: "brokerage_finance"},
	{Code: "601211", Exchange: "SH", SectorCode: "brokerage_finance"},
	{Code: "600999", Exchange: "SH", SectorCode: "brokerage_finance"},
	{Code: "601066", Exchange: "SH", SectorCode: "brokerage_finance"},
	{Code: "600036", Exchange: "SH", SectorCode: "banking"},
	{Code: "601398", Exchange: "SH", SectorCode: "banking"},
	{Code: "601288", Exchange: "SH", SectorCode: "banking"},
	{Code: "601166", Exchange: "SH", SectorCode: "banking"},
	{Code: "600919", Exchange: "SH", SectorCode: "banking"},
	{Code: "601328", Exchange: "SH", SectorCode: "banking"},
	{Code: "601318", Exchange: "SH", SectorCode: "insurance"},
	{Code: "601601", Exchange: "SH", SectorCode: "insurance"},
	{Code: "601628", Exchange: "SH", SectorCode: "insurance"},
	{Code: "600111", Exchange: "SH", SectorCode: "resources_cycle"},
	{Code: "601899", Exchange: "SH", SectorCode: "resources_cycle"},
	{Code: "603993", Exchange: "SH", SectorCode: "resources_cycle"},
	{Code: "600900", Exchange: "SH", SectorCode: "utilities_power"},
	{Code: "003816", Exchange: "SZ", SectorCode: "utilities_power"},
	{Code: "180101", Exchange: "SZ", SectorCode: "real_estate_reits"},
	{Code: "508000", Exchange: "SH", SectorCode: "real_estate_reits"},
	{Code: "300498", Exchange: "SZ", SectorCode: "agriculture"},

	{Code: "NVDA", Exchange: "US", SectorCode: "semiconductor"},
	{Code: "AAPL", Exchange: "US", SectorCode: "consumer_electronics"},
	{Code: "GOOG", Exchange: "US", SectorCode: "internet_platform"},
	{Code: "TSLA", Exchange: "US", SectorCode: "new_energy_auto"},
	{Code: "MSFT", Exchange: "US", SectorCode: "software_cloud"},
	{Code: "AVGO", Exchange: "US", SectorCode: "semiconductor"},
	{Code: "AMZN", Exchange: "US", SectorCode: "internet_ecommerce"},
	{Code: "NFLX", Exchange: "US", SectorCode: "streaming_media"},
	{Code: "META", Exchange: "US", SectorCode: "internet_platform"},
	{Code: "MRVL", Exchange: "US", SectorCode: "semiconductor"},
}

var defaultInstrumentSectorMapByKey = func() map[string]string {
	result := make(map[string]string, len(defaultInstrumentSectorMappings))
	for _, item := range defaultInstrumentSectorMappings {
		result[sectorInstrumentKey(item.Code, item.Exchange)] = item.SectorCode
	}
	return result
}()

var defaultInstrumentThemeMappings = []seededInstrumentTheme{
	{Code: "300308", Exchange: "SZ", ThemeCode: "cpo_optical_module", Weight: decimal.NewFromInt(1)},
	{Code: "300502", Exchange: "SZ", ThemeCode: "cpo_optical_module", Weight: decimal.NewFromInt(1)},
	{Code: "300394", Exchange: "SZ", ThemeCode: "cpo_optical_module", Weight: decimal.NewFromInt(1)},
	{Code: "002281", Exchange: "SZ", ThemeCode: "cpo_optical_module", Weight: decimal.NewFromInt(1)},
	{Code: "000988", Exchange: "SZ", ThemeCode: "cpo_optical_module", Weight: decimal.NewFromInt(1)},
	{Code: "603083", Exchange: "SH", ThemeCode: "cpo_optical_module", Weight: decimal.NewFromInt(1)},

	{Code: "002230", Exchange: "SZ", ThemeCode: "ai_application", Weight: decimal.NewFromInt(1)},
	{Code: "688111", Exchange: "SH", ThemeCode: "ai_application", Weight: decimal.NewFromInt(1)},
	{Code: "300229", Exchange: "SZ", ThemeCode: "ai_application", Weight: decimal.NewFromInt(1)},
	{Code: "600570", Exchange: "SH", ThemeCode: "ai_application", Weight: decimal.NewFromInt(1)},

	{Code: "601138", Exchange: "SH", ThemeCode: "computing_power", Weight: decimal.NewFromInt(1)},
	{Code: "000977", Exchange: "SZ", ThemeCode: "computing_power", Weight: decimal.NewFromInt(1)},
	{Code: "603019", Exchange: "SH", ThemeCode: "computing_power", Weight: decimal.NewFromInt(1)},
	{Code: "000938", Exchange: "SZ", ThemeCode: "computing_power", Weight: decimal.NewFromInt(1)},
	{Code: "NVDA", Exchange: "US", ThemeCode: "computing_power", Weight: decimal.NewFromInt(1)},
	{Code: "AMD", Exchange: "US", ThemeCode: "computing_power", Weight: decimal.NewFromInt(1)},

	{Code: "600118", Exchange: "SH", ThemeCode: "commercial_aerospace", Weight: decimal.NewFromInt(1)},
	{Code: "601698", Exchange: "SH", ThemeCode: "commercial_aerospace", Weight: decimal.NewFromInt(1)},
	{Code: "300762", Exchange: "SZ", ThemeCode: "commercial_aerospace", Weight: decimal.NewFromInt(1)},
	{Code: "600879", Exchange: "SH", ThemeCode: "military_electronics", Weight: decimal.NewFromInt(1)},
	{Code: "600990", Exchange: "SH", ThemeCode: "military_electronics", Weight: decimal.NewFromInt(1)},
	{Code: "002414", Exchange: "SZ", ThemeCode: "military_electronics", Weight: decimal.NewFromInt(1)},

	{Code: "601698", Exchange: "SH", ThemeCode: "satellite_internet", Weight: decimal.NewFromInt(1)},
	{Code: "300762", Exchange: "SZ", ThemeCode: "satellite_internet", Weight: decimal.NewFromInt(1)},

	{Code: "300024", Exchange: "SZ", ThemeCode: "robotics", Weight: decimal.NewFromInt(1)},
	{Code: "002747", Exchange: "SZ", ThemeCode: "robotics", Weight: decimal.NewFromInt(1)},
	{Code: "688017", Exchange: "SH", ThemeCode: "robotics", Weight: decimal.NewFromInt(1)},
	{Code: "300607", Exchange: "SZ", ThemeCode: "robotics", Weight: decimal.NewFromInt(1)},

	{Code: "603881", Exchange: "SH", ThemeCode: "data_infrastructure", Weight: decimal.NewFromInt(1)},
	{Code: "300738", Exchange: "SZ", ThemeCode: "data_infrastructure", Weight: decimal.NewFromInt(1)},
	{Code: "300383", Exchange: "SZ", ThemeCode: "data_infrastructure", Weight: decimal.NewFromInt(1)},

	{Code: "002594", Exchange: "SZ", ThemeCode: "smart_driving", Weight: decimal.NewFromInt(1)},
	{Code: "300496", Exchange: "SZ", ThemeCode: "smart_driving", Weight: decimal.NewFromInt(1)},
	{Code: "688208", Exchange: "SH", ThemeCode: "smart_driving", Weight: decimal.NewFromInt(1)},

	{Code: "300274", Exchange: "SZ", ThemeCode: "energy_storage", Weight: decimal.NewFromInt(1)},
	{Code: "300750", Exchange: "SZ", ThemeCode: "energy_storage", Weight: decimal.NewFromInt(1)},
	{Code: "688063", Exchange: "SH", ThemeCode: "energy_storage", Weight: decimal.NewFromInt(1)},

	{Code: "002625", Exchange: "SZ", ThemeCode: "low_altitude_economy", Weight: decimal.NewFromInt(1)},
	{Code: "300719", Exchange: "SZ", ThemeCode: "low_altitude_economy", Weight: decimal.NewFromInt(1)},
	{Code: "300696", Exchange: "SZ", ThemeCode: "low_altitude_economy", Weight: decimal.NewFromInt(1)},

	{Code: "688235", Exchange: "SH", ThemeCode: "innovative_medicine", Weight: decimal.NewFromInt(1)},
	{Code: "600276", Exchange: "SH", ThemeCode: "innovative_medicine", Weight: decimal.NewFromInt(1)},
	{Code: "300122", Exchange: "SZ", ThemeCode: "innovative_medicine", Weight: decimal.NewFromInt(1)},

	{Code: "002555", Exchange: "SZ", ThemeCode: "gaming_entertainment", Weight: decimal.NewFromInt(1)},
	{Code: "002602", Exchange: "SZ", ThemeCode: "gaming_entertainment", Weight: decimal.NewFromInt(1)},
	{Code: "300413", Exchange: "SZ", ThemeCode: "gaming_entertainment", Weight: decimal.NewFromInt(1)},

	{Code: "601012", Exchange: "SH", ThemeCode: "photovoltaic", Weight: decimal.NewFromInt(1)},
	{Code: "600438", Exchange: "SH", ThemeCode: "photovoltaic", Weight: decimal.NewFromInt(1)},
	{Code: "002459", Exchange: "SZ", ThemeCode: "photovoltaic", Weight: decimal.NewFromInt(1)},
	{Code: "688599", Exchange: "SH", ThemeCode: "photovoltaic", Weight: decimal.NewFromInt(1)},
	{Code: "605117", Exchange: "SH", ThemeCode: "photovoltaic", Weight: decimal.NewFromInt(1)},
	{Code: "603806", Exchange: "SH", ThemeCode: "photovoltaic", Weight: decimal.NewFromInt(1)},

	{Code: "601615", Exchange: "SH", ThemeCode: "wind_power", Weight: decimal.NewFromInt(1)},
	{Code: "002202", Exchange: "SZ", ThemeCode: "wind_power", Weight: decimal.NewFromInt(1)},
	{Code: "002531", Exchange: "SZ", ThemeCode: "wind_power", Weight: decimal.NewFromInt(1)},

	{Code: "600406", Exchange: "SH", ThemeCode: "power_grid", Weight: decimal.NewFromInt(1)},
	{Code: "000400", Exchange: "SZ", ThemeCode: "power_grid", Weight: decimal.NewFromInt(1)},
	{Code: "600312", Exchange: "SH", ThemeCode: "power_grid", Weight: decimal.NewFromInt(1)},
	{Code: "600089", Exchange: "SH", ThemeCode: "power_grid", Weight: decimal.NewFromInt(1)},

	{Code: "688561", Exchange: "SH", ThemeCode: "network_security", Weight: decimal.NewFromInt(1)},
	{Code: "300454", Exchange: "SZ", ThemeCode: "network_security", Weight: decimal.NewFromInt(1)},
	{Code: "002439", Exchange: "SZ", ThemeCode: "network_security", Weight: decimal.NewFromInt(1)},
	{Code: "300188", Exchange: "SZ", ThemeCode: "network_security", Weight: decimal.NewFromInt(1)},

	{Code: "300059", Exchange: "SZ", ThemeCode: "financial_it", Weight: decimal.NewFromInt(1)},
	{Code: "300033", Exchange: "SZ", ThemeCode: "financial_it", Weight: decimal.NewFromInt(1)},
	{Code: "603383", Exchange: "SH", ThemeCode: "financial_it", Weight: decimal.NewFromInt(1)},
	{Code: "600570", Exchange: "SH", ThemeCode: "financial_it", Weight: decimal.NewFromFloat(0.5)},

	{Code: "603259", Exchange: "SH", ThemeCode: "biotech", Weight: decimal.NewFromInt(1)},
	{Code: "300760", Exchange: "SZ", ThemeCode: "biotech", Weight: decimal.NewFromInt(1)},
	{Code: "09926", Exchange: "HK", ThemeCode: "biotech", Weight: decimal.NewFromInt(1)},

	{Code: "000063", Exchange: "SZ", ThemeCode: "communications_5g", Weight: decimal.NewFromInt(1)},
	{Code: "002463", Exchange: "SZ", ThemeCode: "communications_5g", Weight: decimal.NewFromInt(1)},
	{Code: "603228", Exchange: "SH", ThemeCode: "communications_5g", Weight: decimal.NewFromInt(1)},
	{Code: "300308", Exchange: "SZ", ThemeCode: "communications_5g", Weight: decimal.NewFromFloat(0.5)},
	{Code: "300502", Exchange: "SZ", ThemeCode: "communications_5g", Weight: decimal.NewFromFloat(0.5)},
}

var defaultInstrumentThemeMapByKey = func() map[string][]seededInstrumentTheme {
	result := make(map[string][]seededInstrumentTheme)
	for _, item := range defaultInstrumentThemeMappings {
		key := sectorInstrumentKey(item.Code, item.Exchange)
		result[key] = append(result[key], item)
	}
	return result
}()

func SeedDefaultFundSectorData(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}

	if err := db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "code"}},
		UpdateAll: true,
	}).Create(&defaultFundCategories).Error; err != nil {
		return err
	}

	if err := db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "code"}},
		UpdateAll: true,
	}).Create(&defaultFundSectors).Error; err != nil {
		return err
	}
	if err := db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "code"}},
		UpdateAll: true,
	}).Create(&defaultFundThemes).Error; err != nil {
		return err
	}

	mappings := make([]database.InstrumentSectorMap, 0, len(defaultInstrumentSectorMappings))
	for _, item := range defaultInstrumentSectorMappings {
		mappings = append(mappings, database.InstrumentSectorMap{
			InstrumentCode: item.Code,
			Exchange:       item.Exchange,
			SectorCode:     item.SectorCode,
			Source:         "seed",
		})
	}
	if err := db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "instrument_code"}, {Name: "exchange"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"sector_code",
			"source",
			"updated_at",
		}),
	}).Create(&mappings).Error; err != nil {
		return err
	}

	themeMappings := make([]database.InstrumentThemeMap, 0, len(defaultInstrumentThemeMappings))
	for _, item := range defaultInstrumentThemeMappings {
		themeMappings = append(themeMappings, database.InstrumentThemeMap{
			InstrumentCode: item.Code,
			Exchange:       item.Exchange,
			ThemeCode:      item.ThemeCode,
			Source:         "seed",
			Weight:         item.Weight,
		})
	}

	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "instrument_code"}, {Name: "exchange"}, {Name: "theme_code"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"source",
			"weight",
			"updated_at",
		}),
	}).Create(&themeMappings).Error
}

func (s *FundSectorStore) ResolveFundCategory(ctx context.Context, fund *domain.Fund, snapshot *domain.FundSectorSnapshot) (*domain.FundCategory, error) {
	if fund == nil {
		return nil, nil
	}

	override, err := s.getClassificationOverride(ctx, fund.ID)
	if err != nil {
		return nil, err
	}

	categoryCode := deriveFundCategoryCode(fund)
	if override != nil && strings.TrimSpace(override.CategoryCode) != "" {
		categoryCode = strings.TrimSpace(override.CategoryCode)
	}
	categoryNameByCode, err := s.loadCategoryNames(ctx)
	if err != nil {
		return nil, err
	}
	name := categoryNameByCode[categoryCode]
	if name == "" {
		categoryCode = "other"
		name = categoryNameByCode[categoryCode]
	}

	if s != nil && s.db != nil && fund.CategoryCode != categoryCode {
		if err := s.db.WithContext(ctx).Model(&database.Fund{}).Where("id = ?", fund.ID).Update("category_code", categoryCode).Error; err != nil {
			return nil, err
		}
	}

	return &domain.FundCategory{
		Code:      categoryCode,
		Name:      name,
		SortOrder: categorySortOrder(categoryCode),
	}, nil
}

func (s *FundSectorStore) UpsertFromHoldings(ctx context.Context, fundID string, holdings []domain.StockHolding, source string) (*domain.FundSectorSnapshot, error) {
	snapshot, err := s.BuildSnapshot(ctx, fundID, holdings, source)
	if err != nil || snapshot == nil || s == nil || s.db == nil {
		return snapshot, err
	}

	asOfDate, err := time.Parse("2006-01-02", snapshot.AsOfDate)
	if err != nil {
		return nil, err
	}

	record := &database.FundSectorSnapshot{
		FundID:            snapshot.FundID,
		AsOfDate:          asOfDate,
		PrimarySectorCode: snapshot.PrimarySectorCode,
		Source:            snapshot.Source,
		Confidence:        snapshot.Confidence,
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "fund_id"}, {Name: "as_of_date"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"primary_sector_code",
				"source",
				"confidence",
				"updated_at",
			}),
		}).Create(record).Error; err != nil {
			return err
		}

		if err := tx.Where("fund_id = ? AND as_of_date = ?", snapshot.FundID, asOfDate).Delete(&database.FundSectorBreakdown{}).Error; err != nil {
			return err
		}

		if len(snapshot.Breakdown) == 0 {
			return nil
		}

		breakdown := make([]database.FundSectorBreakdown, 0, len(snapshot.Breakdown))
		for _, item := range snapshot.Breakdown {
			breakdown = append(breakdown, database.FundSectorBreakdown{
				FundID:        snapshot.FundID,
				AsOfDate:      asOfDate,
				SectorCode:    item.SectorCode,
				WeightPercent: item.WeightPercent,
				Rank:          item.Rank,
			})
		}
		return tx.Create(&breakdown).Error
	}); err != nil {
		return nil, err
	}

	if _, err := s.UpsertThemeFromHoldings(ctx, fundID, holdings, source); err != nil {
		return nil, err
	}

	return snapshot, nil
}

func (s *FundSectorStore) UpsertThemeFromHoldings(ctx context.Context, fundID string, holdings []domain.StockHolding, source string) (*domain.FundThemeSnapshot, error) {
	snapshot, err := s.BuildThemeSnapshot(ctx, fundID, holdings, source)
	if err != nil || snapshot == nil || s == nil || s.db == nil {
		return snapshot, err
	}

	asOfDate, err := time.Parse("2006-01-02", snapshot.AsOfDate)
	if err != nil {
		return nil, err
	}

	record := &database.FundThemeSnapshot{
		FundID:           snapshot.FundID,
		AsOfDate:         asOfDate,
		PrimaryThemeCode: snapshot.PrimaryThemeCode,
		Source:           snapshot.Source,
		Confidence:       snapshot.Confidence,
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "fund_id"}, {Name: "as_of_date"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"primary_theme_code",
				"source",
				"confidence",
				"updated_at",
			}),
		}).Create(record).Error; err != nil {
			return err
		}

		if err := tx.Where("fund_id = ? AND as_of_date = ?", snapshot.FundID, asOfDate).Delete(&database.FundThemeBreakdown{}).Error; err != nil {
			return err
		}

		if len(snapshot.Breakdown) == 0 {
			return nil
		}

		breakdown := make([]database.FundThemeBreakdown, 0, len(snapshot.Breakdown))
		for _, item := range snapshot.Breakdown {
			breakdown = append(breakdown, database.FundThemeBreakdown{
				FundID:        snapshot.FundID,
				AsOfDate:      asOfDate,
				ThemeCode:     item.ThemeCode,
				WeightPercent: item.WeightPercent,
				Rank:          item.Rank,
			})
		}
		return tx.Create(&breakdown).Error
	}); err != nil {
		return nil, err
	}

	return snapshot, nil
}

func (s *FundSectorStore) BuildSnapshot(ctx context.Context, fundID string, holdings []domain.StockHolding, source string) (*domain.FundSectorSnapshot, error) {
	if !hasEffectiveHoldings(holdings) {
		return nil, nil
	}

	mappingByKey, err := s.loadInstrumentMappings(ctx, holdings)
	if err != nil {
		return nil, err
	}

	sectorTotals := make(map[string]decimal.Decimal)
	totalRatio := decimal.Zero
	for _, holding := range holdings {
		if !holding.HoldingRatio.GreaterThan(decimal.Zero) {
			continue
		}
		sectorCode := inferFundSectorCode(holding, mappingByKey)
		sectorTotals[sectorCode] = sectorTotals[sectorCode].Add(holding.HoldingRatio)
		totalRatio = totalRatio.Add(holding.HoldingRatio)
	}

	if len(sectorTotals) == 0 {
		return nil, nil
	}

	sectorNameByCode, err := s.loadSectorNames(ctx)
	if err != nil {
		return nil, err
	}

	aggregated := make([]sectorAgg, 0, len(sectorTotals))
	for code, weight := range sectorTotals {
		aggregated = append(aggregated, sectorAgg{code: code, weight: weight})
	}
	sort.Slice(aggregated, func(i, j int) bool {
		if aggregated[i].weight.Equal(aggregated[j].weight) {
			return aggregated[i].code < aggregated[j].code
		}
		return aggregated[i].weight.GreaterThan(aggregated[j].weight)
	})

	primary := pickPrimarySectorAgg(aggregated)
	mappedRatio := mappedSectorRatio(aggregated)
	breakdownOrder := reorderSectorAggs(aggregated, primary.code)
	breakdown := make([]domain.FundSectorBreakdown, 0, minInt(3, len(breakdownOrder)))
	for idx, item := range breakdownOrder {
		if idx >= 3 {
			break
		}
		breakdown = append(breakdown, domain.FundSectorBreakdown{
			SectorCode:    item.code,
			SectorName:    sectorNameByCode[item.code],
			WeightPercent: item.weight,
			Rank:          idx + 1,
		})
	}

	confidence := classificationConfidenceFromMappedRatio(mappedRatio)

	return &domain.FundSectorSnapshot{
		FundID:            fundID,
		AsOfDate:          resolveSectorAsOfDate(holdings),
		PrimarySectorCode: primary.code,
		PrimarySectorName: sectorNameByCode[primary.code],
		Source:            normalizeSectorSource(source),
		Confidence:        confidence,
		Breakdown:         breakdown,
	}, nil
}

func (s *FundSectorStore) BuildThemeSnapshot(ctx context.Context, fundID string, holdings []domain.StockHolding, source string) (*domain.FundThemeSnapshot, error) {
	if !hasEffectiveHoldings(holdings) {
		return nil, nil
	}

	mappingByKey, err := s.loadThemeMappings(ctx, holdings)
	if err != nil {
		return nil, err
	}

	themeTotals := make(map[string]decimal.Decimal)
	totalRatio := decimal.Zero
	for _, holding := range holdings {
		if !holding.HoldingRatio.GreaterThan(decimal.Zero) {
			continue
		}
		themeCodes := inferFundThemeCodes(holding, mappingByKey)
		if len(themeCodes) == 0 {
			themeCodes = []string{"other_theme"}
		}
		shareWeight := decimal.NewFromInt(1).DivRound(decimal.NewFromInt(int64(len(themeCodes))), 8)
		for _, themeCode := range themeCodes {
			themeTotals[themeCode] = themeTotals[themeCode].Add(holding.HoldingRatio.Mul(shareWeight))
		}
		totalRatio = totalRatio.Add(holding.HoldingRatio)
	}

	if len(themeTotals) == 0 {
		return nil, nil
	}

	themeNameByCode, err := s.loadThemeNames(ctx)
	if err != nil {
		return nil, err
	}

	aggregated := make([]themeAgg, 0, len(themeTotals))
	for code, weight := range themeTotals {
		aggregated = append(aggregated, themeAgg{code: code, weight: weight})
	}
	sort.Slice(aggregated, func(i, j int) bool {
		if aggregated[i].weight.Equal(aggregated[j].weight) {
			return aggregated[i].code < aggregated[j].code
		}
		return aggregated[i].weight.GreaterThan(aggregated[j].weight)
	})

	primary := pickPrimaryThemeAgg(aggregated)
	mappedRatio := mappedThemeRatio(aggregated)
	breakdownOrder := reorderThemeAggs(aggregated, primary.code)
	breakdown := make([]domain.FundThemeBreakdown, 0, minInt(3, len(breakdownOrder)))
	for idx, item := range breakdownOrder {
		if idx >= 3 {
			break
		}
		breakdown = append(breakdown, domain.FundThemeBreakdown{
			ThemeCode:     item.code,
			ThemeName:     themeNameByCode[item.code],
			WeightPercent: item.weight,
			Rank:          idx + 1,
		})
	}

	confidence := classificationConfidenceFromMappedRatio(mappedRatio)

	return &domain.FundThemeSnapshot{
		FundID:           fundID,
		AsOfDate:         resolveSectorAsOfDate(holdings),
		PrimaryThemeCode: primary.code,
		PrimaryThemeName: themeNameByCode[primary.code],
		Source:           normalizeThemeSource(source),
		Confidence:       confidence,
		Breakdown:        breakdown,
	}, nil
}

func (s *FundSectorStore) GetLatestSnapshot(ctx context.Context, fundID string) (*domain.FundSectorSnapshot, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}

	var snapshot database.FundSectorSnapshot
	result := s.db.WithContext(ctx).Where("fund_id = ?", fundID).Order("as_of_date DESC").First(&snapshot)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}

	var breakdownRecords []database.FundSectorBreakdown
	if err := s.db.WithContext(ctx).
		Where("fund_id = ? AND as_of_date = ?", fundID, snapshot.AsOfDate).
		Order("rank ASC").
		Find(&breakdownRecords).Error; err != nil {
		return nil, err
	}

	sectorNameByCode, err := s.loadSectorNames(ctx)
	if err != nil {
		return nil, err
	}
	override, err := s.getClassificationOverride(ctx, fundID)
	if err != nil {
		return nil, err
	}

	primarySectorCode := snapshot.PrimarySectorCode
	if override != nil && strings.TrimSpace(override.PrimarySectorCode) != "" {
		primarySectorCode = strings.TrimSpace(override.PrimarySectorCode)
	}

	breakdown := make([]domain.FundSectorBreakdown, 0, len(breakdownRecords))
	for _, item := range breakdownRecords {
		sectorCode := item.SectorCode
		if override != nil && strings.TrimSpace(override.PrimarySectorCode) != "" && item.Rank == 1 {
			sectorCode = strings.TrimSpace(override.PrimarySectorCode)
		}
		breakdown = append(breakdown, domain.FundSectorBreakdown{
			SectorCode:    sectorCode,
			SectorName:    sectorNameByCode[sectorCode],
			WeightPercent: item.WeightPercent,
			Rank:          item.Rank,
		})
	}

	return &domain.FundSectorSnapshot{
		FundID:            snapshot.FundID,
		AsOfDate:          snapshot.AsOfDate.Format("2006-01-02"),
		PrimarySectorCode: primarySectorCode,
		PrimarySectorName: sectorNameByCode[primarySectorCode],
		Source:            snapshot.Source,
		Confidence:        snapshot.Confidence,
		Breakdown:         breakdown,
	}, nil
}

func (s *FundSectorStore) GetLatestThemeSnapshot(ctx context.Context, fundID string) (*domain.FundThemeSnapshot, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}

	var snapshot database.FundThemeSnapshot
	result := s.db.WithContext(ctx).Where("fund_id = ?", fundID).Order("as_of_date DESC").First(&snapshot)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}

	var breakdownRecords []database.FundThemeBreakdown
	if err := s.db.WithContext(ctx).
		Where("fund_id = ? AND as_of_date = ?", fundID, snapshot.AsOfDate).
		Order("rank ASC").
		Find(&breakdownRecords).Error; err != nil {
		return nil, err
	}

	themeNameByCode, err := s.loadThemeNames(ctx)
	if err != nil {
		return nil, err
	}

	breakdown := make([]domain.FundThemeBreakdown, 0, len(breakdownRecords))
	for _, item := range breakdownRecords {
		breakdown = append(breakdown, domain.FundThemeBreakdown{
			ThemeCode:     item.ThemeCode,
			ThemeName:     themeNameByCode[item.ThemeCode],
			WeightPercent: item.WeightPercent,
			Rank:          item.Rank,
		})
	}

	return &domain.FundThemeSnapshot{
		FundID:           snapshot.FundID,
		AsOfDate:         snapshot.AsOfDate.Format("2006-01-02"),
		PrimaryThemeCode: snapshot.PrimaryThemeCode,
		PrimaryThemeName: themeNameByCode[snapshot.PrimaryThemeCode],
		Source:           snapshot.Source,
		Confidence:       snapshot.Confidence,
		Breakdown:        breakdown,
	}, nil
}

func (s *FundSectorStore) loadInstrumentMappings(ctx context.Context, holdings []domain.StockHolding) (map[string]string, error) {
	result := make(map[string]string)
	if s == nil || s.db == nil {
		return result, nil
	}

	codes := make([]string, 0, len(holdings))
	exchanges := make([]string, 0, len(holdings))
	for _, holding := range holdings {
		codes = append(codes, strings.ToUpper(strings.TrimSpace(holding.StockCode)))
		exchanges = append(exchanges, string(holding.Exchange))
	}

	var mappings []database.InstrumentSectorMap
	if err := s.db.WithContext(ctx).
		Where("instrument_code IN ? AND exchange IN ?", uniqueStrings(codes), uniqueStrings(exchanges)).
		Find(&mappings).Error; err != nil {
		return nil, err
	}

	for _, item := range mappings {
		result[sectorInstrumentKey(item.InstrumentCode, item.Exchange)] = item.SectorCode
	}
	return result, nil
}

func (s *FundSectorStore) loadThemeMappings(ctx context.Context, holdings []domain.StockHolding) (map[string][]database.InstrumentThemeMap, error) {
	result := make(map[string][]database.InstrumentThemeMap)
	if s == nil || s.db == nil {
		return result, nil
	}

	codes := make([]string, 0, len(holdings))
	exchanges := make([]string, 0, len(holdings))
	for _, holding := range holdings {
		codes = append(codes, strings.ToUpper(strings.TrimSpace(holding.StockCode)))
		exchanges = append(exchanges, string(holding.Exchange))
	}

	var mappings []database.InstrumentThemeMap
	if err := s.db.WithContext(ctx).
		Where("instrument_code IN ? AND exchange IN ?", uniqueStrings(codes), uniqueStrings(exchanges)).
		Find(&mappings).Error; err != nil {
		return nil, err
	}

	for _, item := range mappings {
		key := sectorInstrumentKey(item.InstrumentCode, item.Exchange)
		result[key] = append(result[key], item)
	}
	return result, nil
}

func (s *FundSectorStore) loadSectorNames(ctx context.Context) (map[string]string, error) {
	result := make(map[string]string)
	for _, sector := range defaultFundSectors {
		result[sector.Code] = sector.Name
	}
	if s == nil || s.db == nil {
		return result, nil
	}

	var sectors []database.FundSector
	if err := s.db.WithContext(ctx).Where("is_enabled = ?", true).Find(&sectors).Error; err != nil {
		return nil, err
	}
	for _, sector := range sectors {
		result[sector.Code] = sector.Name
	}
	return result, nil
}

func (s *FundSectorStore) loadCategoryNames(ctx context.Context) (map[string]string, error) {
	result := make(map[string]string)
	for _, category := range defaultFundCategories {
		result[category.Code] = category.Name
	}
	if s == nil || s.db == nil {
		return result, nil
	}

	var categories []database.FundCategory
	if err := s.db.WithContext(ctx).Where("is_enabled = ?", true).Find(&categories).Error; err != nil {
		return nil, err
	}
	for _, category := range categories {
		result[category.Code] = category.Name
	}
	return result, nil
}

func (s *FundSectorStore) loadThemeNames(ctx context.Context) (map[string]string, error) {
	result := make(map[string]string)
	for _, theme := range defaultFundThemes {
		result[theme.Code] = theme.Name
	}
	if s == nil || s.db == nil {
		return result, nil
	}

	var themes []database.FundTheme
	if err := s.db.WithContext(ctx).Where("is_enabled = ?", true).Find(&themes).Error; err != nil {
		return nil, err
	}
	for _, theme := range themes {
		result[theme.Code] = theme.Name
	}
	return result, nil
}

func (s *FundSectorStore) getClassificationOverride(ctx context.Context, fundID string) (*database.FundClassificationOverride, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}

	var override database.FundClassificationOverride
	result := s.db.WithContext(ctx).Where("fund_id = ?", fundID).First(&override)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &override, nil
}

func inferFundSectorCode(holding domain.StockHolding, mappingByKey map[string]string) string {
	key := sectorInstrumentKey(holding.StockCode, string(holding.Exchange))
	if code, ok := mappingByKey[key]; ok {
		return code
	}
	if code, ok := defaultInstrumentSectorMapByKey[key]; ok {
		return code
	}

	name := strings.ToLower(strings.TrimSpace(holding.StockName))
	switch {
	case containsSectorKeyword(name, "光模块", "光通信", "通信设备", "通信", "光器件", "天孚", "中际", "新易盛", "剑桥"):
		return "communications_equipment"
	case containsSectorKeyword(name, "芯片", "半导体", "存储"):
		return "semiconductor"
	case containsSectorKeyword(name, "银行"):
		return "banking"
	case containsSectorKeyword(name, "证券", "券商", "东方财富", "中信建投", "广发证券", "国泰君安", "华泰证券", "中金公司"):
		return "brokerage_finance"
	case containsSectorKeyword(name, "保险", "平安", "人寿", "太保", "新华保险"):
		return "insurance"
	case containsSectorKeyword(name, "软件", "云", "数据"):
		return "software_cloud"
	case containsSectorKeyword(name, "白酒", "茅台", "五粮液", "汾酒", "老窖"):
		return "liquor"
	case containsSectorKeyword(name, "食品", "饮料", "乳业", "调味", "啤酒", "伊利", "海天", "双汇", "安井"):
		return "food_beverage"
	case containsSectorKeyword(name, "石油", "海洋石油", "油气"):
		return "oil_gas_energy"
	case containsSectorKeyword(name, "有色", "铜", "铝", "黄金", "矿业", "稀土", "紫金", "洛阳钼业", "山东黄金", "北方稀土"):
		return "resources_cycle"
	case containsSectorKeyword(name, "健康", "医疗", "医药", "生物"):
		return "healthcare_service"
	case containsSectorKeyword(name, "机械", "机床", "重工", "自动化", "机器人", "工程机械", "电梯", "叉车"):
		return "machinery_equipment"
	case containsSectorKeyword(name, "地产", "物业", "reit", "reits", "万科", "保利发展", "招商蛇口", "华润置地"):
		return "real_estate_reits"
	case containsSectorKeyword(name, "农业", "牧业", "种业", "养殖", "猪", "饲料", "农发", "隆平"):
		return "agriculture"
	case containsSectorKeyword(name, "亚马逊", "京东", "电商"):
		return "internet_ecommerce"
	case containsSectorKeyword(name, "腾讯", "阿里", "谷歌", "meta", "facebook", "平台"):
		return "internet_platform"
	case containsSectorKeyword(name, "苹果", "消费电子"):
		return "consumer_electronics"
	case containsSectorKeyword(name, "奈飞", "视频", "流媒体"):
		return "streaming_media"
	case containsSectorKeyword(name, "特斯拉", "新能源车", "汽车"):
		return "new_energy_auto"
	case containsSectorKeyword(name, "分众", "传媒", "广告"):
		return "media_advertising"
	case containsSectorKeyword(name, "电力", "火电", "水电", "燃气", "公用事业", "长江电力", "华能", "国电", "三峡能源"):
		return "utilities_power"
	default:
		return "other_equity"
	}
}

func inferFundThemeCodes(holding domain.StockHolding, mappingByKey map[string][]database.InstrumentThemeMap) []string {
	key := sectorInstrumentKey(holding.StockCode, string(holding.Exchange))
	if mappings, ok := mappingByKey[key]; ok && len(mappings) > 0 {
		result := make([]string, 0, len(mappings))
		for _, item := range mappings {
			result = append(result, item.ThemeCode)
		}
		return uniqueStrings(result)
	}
	if mappings, ok := defaultInstrumentThemeMapByKey[key]; ok && len(mappings) > 0 {
		result := make([]string, 0, len(mappings))
		for _, item := range mappings {
			result = append(result, item.ThemeCode)
		}
		return uniqueStrings(result)
	}

	name := strings.ToLower(strings.TrimSpace(holding.StockName))
	switch {
	case containsSectorKeyword(name, "芯片", "半导体", "存储", "eda", "封测", "晶圆"):
		return []string{"semiconductor_chip"}
	case containsSectorKeyword(name, "cpo", "光模块", "光通信", "光器件"):
		return []string{"cpo_optical_module"}
	case containsSectorKeyword(name, "互联网", "平台", "腾讯", "阿里", "美团", "京东", "拼多多", "网易"):
		return []string{"platform_internet"}
	case containsSectorKeyword(name, "ai", "aigc", "大模型", "智能办公", "金山办公", "科大讯飞", "应用"):
		return []string{"ai_application"}
	case containsSectorKeyword(name, "算力", "服务器", "gpu", "英伟达", "浪潮", "曙光", "工业富联", "idc"):
		return []string{"computing_power"}
	case containsSectorKeyword(name, "商业航天", "卫星", "火箭", "航天", "卫通"):
		return []string{"commercial_aerospace", "satellite_internet"}
	case containsSectorKeyword(name, "低空", "无人机", "飞行汽车", "evtol", "通航"):
		return []string{"low_altitude_economy"}
	case containsSectorKeyword(name, "智驾", "自动驾驶", "激光雷达", "车路协同", "智能驾驶"):
		return []string{"smart_driving"}
	case containsSectorKeyword(name, "机器人", "自动化", "谐波", "拓斯达", "埃斯顿"):
		return []string{"robotics"}
	case containsSectorKeyword(name, "数据港", "光环新网", "奥飞数据", "数据基础", "数据要素"):
		return []string{"data_infrastructure"}
	case containsSectorKeyword(name, "军工电子", "航天电子", "四创电子", "高德红外"):
		return []string{"military_electronics"}
	case containsSectorKeyword(name, "医疗", "医药", "生物", "器械", "药明", "迈瑞", "恒瑞", "百济", "康方"):
		return []string{"healthcare"}
	case containsSectorKeyword(name, "创新药", "单抗", "adc", "百济", "恒瑞", "药明", "cro"):
		return []string{"innovative_medicine"}
	case containsSectorKeyword(name, "储能", "电池", "宁德时代", "阳光电源", "逆变器"):
		return []string{"energy_storage"}
	case containsSectorKeyword(name, "游戏", "娱乐", "传媒", "恺英", "三七", "完美世界", "吉比特"):
		return []string{"gaming_entertainment"}
	case containsSectorKeyword(name, "白酒", "食品", "饮料", "家电", "免税", "酒店", "旅游", "零售", "乳业"):
		return []string{"consumption_upgrade"}
	case containsSectorKeyword(name, "光伏", "逆变器", "组件", "硅料", "隆基", "通威", "晶澳", "天合"):
		return []string{"photovoltaic"}
	case containsSectorKeyword(name, "风电", "海风", "明阳", "金风", "风塔", "风机"):
		return []string{"wind_power"}
	case containsSectorKeyword(name, "电网", "特高压", "国电南瑞", "许继", "平高", "特变"):
		return []string{"power_grid"}
	case containsSectorKeyword(name, "网络安全", "安全", "奇安信", "深信服", "启明星辰"):
		return []string{"network_security"}
	case containsSectorKeyword(name, "银行", "证券", "券商", "保险", "平安", "东方财富", "中信证券"):
		return []string{"financials", "dividend_value"}
	case containsSectorKeyword(name, "金融科技", "证券软件", "同花顺", "东方财富", "恒生电子", "金融it"):
		return []string{"financial_it"}
	case containsSectorKeyword(name, "生物", "药明", "百济", "康方", "迈瑞", "生命科技", "biotech"):
		return []string{"biotech"}
	case containsSectorKeyword(name, "5g", "通信", "中兴", "沪电", "天线", "基站", "射频"):
		return []string{"communications_5g"}
	case containsSectorKeyword(name, "有色", "铜", "铝", "黄金", "煤炭", "石油", "矿业", "稀土"):
		return []string{"resources_cycle"}
	case containsSectorKeyword(name, "地产", "物业", "reit", "reits"):
		return []string{"reits_real_estate"}
	default:
		if sectorCode := inferFundSectorCode(holding, nil); sectorCode != "other_equity" {
			if sectorThemes := fallbackThemeCodesFromSector(sectorCode); len(sectorThemes) > 0 {
				return sectorThemes
			}
		}
		return []string{"other_theme"}
	}
}

func deriveFundCategoryCode(fund *domain.Fund) string {
	if fund == nil {
		return "other"
	}

	fundType := strings.ToLower(strings.TrimSpace(fund.Type))
	name := strings.ToLower(strings.TrimSpace(fund.Name))
	switch {
	case strings.Contains(name, "联接"):
		return "feeder"
	case strings.Contains(fundType, "qdii") || strings.Contains(name, "qdii"):
		return "qdii"
	case strings.Contains(fundType, "fof") || strings.Contains(name, "fof"):
		return "fof"
	case strings.Contains(fundType, "商品") || strings.Contains(name, "期货") || strings.Contains(name, "黄金") || strings.Contains(name, "白银"):
		return "commodity"
	case fundType == "money" || strings.Contains(fundType, "货币"):
		return "money"
	case fundType == "bond" || strings.Contains(fundType, "债"):
		return "bond"
	case fundType == "index" || strings.Contains(fundType, "指数") || strings.Contains(name, "etf"):
		return "index"
	case fundType == "stock" || strings.Contains(fundType, "股票"):
		return "stock"
	case fundType == "hybrid" || strings.Contains(fundType, "混合"):
		return "hybrid"
	default:
		return "other"
	}
}

func categorySortOrder(code string) int {
	for _, category := range defaultFundCategories {
		if category.Code == code {
			return category.SortOrder
		}
	}
	return 999
}

func resolveSectorAsOfDate(holdings []domain.StockHolding) string {
	for _, holding := range holdings {
		trimmed := strings.TrimSpace(holding.ReportingPeriod)
		if trimmed == "" {
			continue
		}
		if _, err := time.Parse("2006-01-02", trimmed); err == nil {
			return trimmed
		}
	}
	return time.Now().In(tradingLocation()).Format("2006-01-02")
}

func normalizeSectorSource(source string) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return SectorSourceDirectHoldings
	}
	return source
}

func normalizeThemeSource(source string) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return ThemeSourceDirectHoldings
	}
	return source
}

func classificationConfidenceFromMappedRatio(mappedRatio decimal.Decimal) string {
	switch {
	case mappedRatio.GreaterThanOrEqual(decimal.NewFromInt(50)):
		return SectorConfidenceHigh
	case mappedRatio.GreaterThanOrEqual(decimal.NewFromInt(20)):
		return SectorConfidenceMedium
	default:
		return SectorConfidenceLow
	}
}

func mappedSectorRatio(aggregated []sectorAgg) decimal.Decimal {
	total := decimal.Zero
	for _, item := range aggregated {
		if item.code == "other_equity" {
			continue
		}
		total = total.Add(item.weight)
	}
	return total
}

func mappedThemeRatio(aggregated []themeAgg) decimal.Decimal {
	total := decimal.Zero
	for _, item := range aggregated {
		if item.code == "other_theme" {
			continue
		}
		total = total.Add(item.weight)
	}
	return total
}

func pickPrimarySectorAgg(aggregated []sectorAgg) sectorAgg {
	for _, item := range aggregated {
		if item.code != "other_equity" {
			return item
		}
	}
	return aggregated[0]
}

func pickPrimaryThemeAgg(aggregated []themeAgg) themeAgg {
	for _, item := range aggregated {
		if item.code != "other_theme" {
			return item
		}
	}
	return aggregated[0]
}

func reorderSectorAggs(aggregated []sectorAgg, primaryCode string) []sectorAgg {
	if len(aggregated) <= 1 {
		return aggregated
	}
	result := make([]sectorAgg, 0, len(aggregated))
	for _, item := range aggregated {
		if item.code == primaryCode {
			result = append(result, item)
			break
		}
	}
	for _, item := range aggregated {
		if item.code == primaryCode {
			continue
		}
		result = append(result, item)
	}
	return result
}

func reorderThemeAggs(aggregated []themeAgg, primaryCode string) []themeAgg {
	if len(aggregated) <= 1 {
		return aggregated
	}
	result := make([]themeAgg, 0, len(aggregated))
	for _, item := range aggregated {
		if item.code == primaryCode {
			result = append(result, item)
			break
		}
	}
	for _, item := range aggregated {
		if item.code == primaryCode {
			continue
		}
		result = append(result, item)
	}
	return result
}

func fallbackThemeCodesFromSector(sectorCode string) []string {
	switch strings.TrimSpace(sectorCode) {
	case "semiconductor":
		return []string{"semiconductor_chip"}
	case "communications_equipment":
		return []string{"communications_5g"}
	case "healthcare_service":
		return []string{"healthcare"}
	case "consumer_electronics", "consumer_service", "internet_ecommerce", "food_beverage", "liquor":
		return []string{"consumption_upgrade"}
	case "internet_platform", "software_cloud", "streaming_media":
		return []string{"platform_internet"}
	case "banking", "brokerage_finance", "insurance":
		return []string{"financials"}
	case "resources_cycle", "oil_gas_energy":
		return []string{"resources_cycle"}
	case "real_estate_reits":
		return []string{"reits_real_estate"}
	case "utilities_power":
		return []string{"dividend_value"}
	default:
		return nil
	}
}

func sectorInstrumentKey(code, exchange string) string {
	return strings.ToUpper(strings.TrimSpace(code)) + "|" + strings.ToUpper(strings.TrimSpace(exchange))
}

func containsSectorKeyword(value string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(value, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
