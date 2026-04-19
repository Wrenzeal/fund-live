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

var defaultFundSectors = []database.FundSector{
	{Code: "semiconductor", Name: "半导体", Level: 1, SortOrder: 10, IsEnabled: true},
	{Code: "consumer_electronics", Name: "消费电子", Level: 1, SortOrder: 20, IsEnabled: true},
	{Code: "internet_platform", Name: "互联网平台", Level: 1, SortOrder: 30, IsEnabled: true},
	{Code: "internet_ecommerce", Name: "互联网电商", Level: 1, SortOrder: 40, IsEnabled: true},
	{Code: "software_cloud", Name: "软件云计算", Level: 1, SortOrder: 50, IsEnabled: true},
	{Code: "streaming_media", Name: "流媒体娱乐", Level: 1, SortOrder: 60, IsEnabled: true},
	{Code: "new_energy_auto", Name: "新能源车", Level: 1, SortOrder: 70, IsEnabled: true},
	{Code: "liquor", Name: "白酒", Level: 1, SortOrder: 80, IsEnabled: true},
	{Code: "media_advertising", Name: "传媒广告", Level: 1, SortOrder: 90, IsEnabled: true},
	{Code: "oil_gas_energy", Name: "石油能源", Level: 1, SortOrder: 100, IsEnabled: true},
	{Code: "healthcare_service", Name: "医疗服务", Level: 1, SortOrder: 110, IsEnabled: true},
	{Code: "consumer_service", Name: "消费服务", Level: 1, SortOrder: 120, IsEnabled: true},
	{Code: "other_equity", Name: "其他权益", Level: 1, SortOrder: 999, IsEnabled: true},
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

	mappings := make([]database.InstrumentSectorMap, 0, len(defaultInstrumentSectorMappings))
	for _, item := range defaultInstrumentSectorMappings {
		mappings = append(mappings, database.InstrumentSectorMap{
			InstrumentCode: item.Code,
			Exchange:       item.Exchange,
			SectorCode:     item.SectorCode,
			Source:         "seed",
		})
	}

	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "instrument_code"}, {Name: "exchange"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"sector_code",
			"source",
			"updated_at",
		}),
	}).Create(&mappings).Error
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

	type sectorAgg struct {
		code   string
		weight decimal.Decimal
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

	breakdown := make([]domain.FundSectorBreakdown, 0, minInt(3, len(aggregated)))
	for idx, item := range aggregated {
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

	confidence := SectorConfidenceMedium
	if totalRatio.GreaterThan(decimal.NewFromInt(50)) {
		confidence = SectorConfidenceHigh
	}
	if len(aggregated) == 1 && aggregated[0].code == "other_equity" {
		confidence = SectorConfidenceLow
	}

	return &domain.FundSectorSnapshot{
		FundID:            fundID,
		AsOfDate:          resolveSectorAsOfDate(holdings),
		PrimarySectorCode: aggregated[0].code,
		PrimarySectorName: sectorNameByCode[aggregated[0].code],
		Source:            normalizeSectorSource(source),
		Confidence:        confidence,
		Breakdown:         breakdown,
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
	case containsSectorKeyword(name, "芯片", "半导体", "存储"):
		return "semiconductor"
	case containsSectorKeyword(name, "软件", "云", "数据"):
		return "software_cloud"
	case containsSectorKeyword(name, "白酒", "茅台", "五粮液", "汾酒", "老窖"):
		return "liquor"
	case containsSectorKeyword(name, "石油", "海洋石油", "油气"):
		return "oil_gas_energy"
	case containsSectorKeyword(name, "健康", "医疗", "医药", "生物"):
		return "healthcare_service"
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
	default:
		return "other_equity"
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
