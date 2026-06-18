package service

import (
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
)

type macroPolicySeed struct {
	Code        string
	Title       string
	Summary     string
	Impact      string
	Strength    string
	PublishedAt string
	ExpiresAt   string
	SectorCodes []string
	ThemeCodes  []string
}

type macroMatchContext struct {
	matchedPrimaryTheme  string
	matchedPrimarySector string
	matchedName          string
	matchedScope         string
	matchedWeight        float64
	matchedPrimary       bool
}

const macroPolicyMinimumExposureWeight = 8.0

var macroPolicySeeds = []macroPolicySeed{
	{
		Code:        "macro_ic_tax_support_2026",
		Title:       "集成电路税收支持延续",
		Summary:     "近期官方继续强调集成电路产业支持，半导体主线的中期政策预期仍有支撑。",
		Impact:      "positive",
		Strength:    "medium",
		PublishedAt: "2026-04-09",
		ExpiresAt:   "2026-06-30",
		SectorCodes: []string{"semiconductor"},
		ThemeCodes:  []string{"semiconductor_chip"},
	},
	{
		Code:        "macro_ai_plus_2026",
		Title:       "AI+ 与智能终端政策继续推进",
		Summary:     "近期官方继续释放 AI+、智能终端和算力基础设施推进信号，AI 与算力主线情绪仍有支撑。",
		Impact:      "positive",
		Strength:    "medium",
		PublishedAt: "2026-03-05",
		ExpiresAt:   "2026-06-30",
		SectorCodes: []string{"software_service", "communications_equipment"},
		ThemeCodes:  []string{"computing_power", "ai_application", "data_infrastructure", "robotics"},
	},
	{
		Code:        "macro_innovative_medicine_support_2026",
		Title:       "创新药与医疗产业支持继续推进",
		Summary:     "近期医药创新、医疗服务与高端器械方向继续获得政策与产业层支持，医疗主线的中期景气预期仍有承接。",
		Impact:      "positive",
		Strength:    "medium",
		PublishedAt: "2026-03-20",
		ExpiresAt:   "2026-07-31",
		SectorCodes: []string{"healthcare_service"},
		ThemeCodes:  []string{"healthcare", "innovative_medicine", "biotech"},
	},
	{
		Code:        "macro_dividend_finance_support_2026",
		Title:       "高股息与金融权重防御属性抬升",
		Summary:     "在波动环境下，高股息、银行与券商非银板块的防御与估值修复逻辑更容易获得市场关注。",
		Impact:      "neutral",
		Strength:    "medium",
		PublishedAt: "2026-04-01",
		ExpiresAt:   "2026-07-31",
		SectorCodes: []string{"banking", "brokerage_finance", "insurance"},
		ThemeCodes:  []string{"dividend_value", "financials", "financial_it"},
	},
	{
		Code:        "macro_energy_transition_2026",
		Title:       "新能源设备与电网改造继续推进",
		Summary:     "储能、光伏、风电与电网升级仍属于中期产业投入重点，相关主线的事件密度和景气讨论仍然较高。",
		Impact:      "positive",
		Strength:    "medium",
		PublishedAt: "2026-03-18",
		ExpiresAt:   "2026-08-31",
		SectorCodes: []string{"new_energy_auto", "utilities"},
		ThemeCodes:  []string{"energy_storage", "photovoltaic", "wind_power", "power_grid"},
	},
	{
		Code:        "macro_platform_consumption_2026",
		Title:       "平台互联网与消费服务景气修复",
		Summary:     "平台互联网、消费服务与线上消费相关主线近期更容易受到政策、需求修复与经营改善预期影响。",
		Impact:      "positive",
		Strength:    "medium",
		PublishedAt: "2026-03-12",
		ExpiresAt:   "2026-07-31",
		SectorCodes: []string{"internet_platform", "internet_ecommerce", "consumer_service", "consumer_electronics"},
		ThemeCodes:  []string{"platform_internet", "consumer_upgrade", "gaming_entertainment"},
	},
	{
		Code:        "realtime_hormuz_reopening_energy_pressure_202606",
		Title:       "美伊协议推动霍尔木兹重开预期",
		Summary:     "美伊协议与霍尔木兹重开预期推动原油风险溢价回落，油气上游和资源周期主线需要观察油价中枢下修压力。",
		Impact:      "negative",
		Strength:    "high",
		PublishedAt: "2026-06-15",
		ExpiresAt:   "2026-07-31",
		SectorCodes: []string{"oil_gas_energy", "resources_cycle"},
	},
	{
		Code:        "realtime_hormuz_reopening_cost_relief_202606",
		Title:       "霍尔木兹重开预期缓解成本压力",
		Summary:     "中东能源运输风险缓和有助于压低油价和通胀风险溢价，消费、出行与部分制造链条的成本压力边际改善。",
		Impact:      "positive",
		Strength:    "medium",
		PublishedAt: "2026-06-15",
		ExpiresAt:   "2026-07-31",
		SectorCodes: []string{"consumer_service", "consumer_electronics", "internet_ecommerce", "food_beverage", "liquor", "new_energy_auto"},
		ThemeCodes:  []string{"consumer_upgrade", "gaming_entertainment", "platform_internet", "smart_driving"},
	},
	{
		Code:        "realtime_hormuz_reopening_defensive_rotation_202606",
		Title:       "地缘风险缓和后防御交易需复核",
		Summary:     "美伊冲突缓和降低避险情绪，前期因高波动受益的高股息、金融权重和防御资产需要重新观察资金拥挤度。",
		Impact:      "neutral",
		Strength:    "medium",
		PublishedAt: "2026-06-15",
		ExpiresAt:   "2026-07-31",
		SectorCodes: []string{"banking", "brokerage_finance", "insurance"},
		ThemeCodes:  []string{"dividend_value", "financials", "financial_it"},
	},
}

func LoadMacroPolicyEvents(now time.Time, sectorSnapshot *domain.FundSectorSnapshot, themeSnapshot *domain.FundThemeSnapshot) []domain.FundAnalysisEventImpact {
	if now.IsZero() {
		now = time.Now()
	}

	activeSectorCodes := make(map[string]float64)
	activeThemeCodes := make(map[string]float64)
	if sectorSnapshot != nil {
		for _, item := range sectorSnapshot.Breakdown {
			if code := strings.TrimSpace(item.SectorCode); code != "" {
				activeSectorCodes[normalizeMacroCode(code)] = decimalToFloat(item.WeightPercent)
			}
		}
	}
	if themeSnapshot != nil {
		for _, item := range themeSnapshot.Breakdown {
			if code := strings.TrimSpace(item.ThemeCode); code != "" {
				activeThemeCodes[normalizeMacroCode(code)] = decimalToFloat(item.WeightPercent)
			}
		}
	}

	results := make([]domain.FundAnalysisEventImpact, 0, 2)
	for _, item := range macroPolicySeeds {
		if !macroPolicySeedActive(item, now) {
			continue
		}
		match := matchMacroSeed(item, activeSectorCodes, activeThemeCodes, sectorSnapshot, themeSnapshot)
		if match == nil {
			continue
		}
		results = append(results, domain.FundAnalysisEventImpact{
			Code:        item.Code,
			Title:       item.Title,
			Impact:      item.Impact,
			Summary:     contextualizeMacroSummary(item, match),
			TargetScope: "macro",
			Strength:    macroEventStrength(item.Strength, match.matchedWeight),
			Horizon:     "current",
			WeightHint:  decimalPointerFromFloat(match.matchedWeight),
		})
	}
	return results
}

func macroPolicySeedActive(item macroPolicySeed, now time.Time) bool {
	if item.PublishedAt != "" {
		if publishedAt, err := time.ParseInLocation("2006-01-02", item.PublishedAt, time.Local); err == nil && now.Before(publishedAt) {
			return false
		}
	}
	if item.ExpiresAt != "" {
		if expiresAt, err := time.ParseInLocation("2006-01-02", item.ExpiresAt, time.Local); err == nil && now.After(expiresAt.Add(24*time.Hour)) {
			return false
		}
	}
	return true
}

func matchMacroSeed(
	item macroPolicySeed,
	sectorCodes map[string]float64,
	themeCodes map[string]float64,
	sectorSnapshot *domain.FundSectorSnapshot,
	themeSnapshot *domain.FundThemeSnapshot,
) *macroMatchContext {
	primaryThemeCode := ""
	primarySectorCode := ""
	if themeSnapshot != nil {
		primaryThemeCode = normalizeMacroCode(strings.TrimSpace(themeSnapshot.PrimaryThemeCode))
	}
	if sectorSnapshot != nil {
		primarySectorCode = normalizeMacroCode(strings.TrimSpace(sectorSnapshot.PrimarySectorCode))
	}
	for _, code := range item.ThemeCodes {
		normalized := normalizeMacroCode(strings.TrimSpace(code))
		if weight, ok := themeCodes[normalized]; ok && weight >= macroPolicyMinimumExposureWeight {
			context := &macroMatchContext{
				matchedName:    macroThemeName(themeSnapshot, normalized),
				matchedScope:   "theme",
				matchedWeight:  weight,
				matchedPrimary: normalized == primaryThemeCode,
			}
			if context.matchedPrimary && themeSnapshot != nil {
				context.matchedPrimaryTheme = strings.TrimSpace(themeSnapshot.PrimaryThemeName)
			}
			return context
		}
	}
	for _, code := range item.SectorCodes {
		normalized := normalizeMacroCode(strings.TrimSpace(code))
		if weight, ok := sectorCodes[normalized]; ok && weight >= macroPolicyMinimumExposureWeight {
			context := &macroMatchContext{
				matchedName:    macroSectorName(sectorSnapshot, normalized),
				matchedScope:   "sector",
				matchedWeight:  weight,
				matchedPrimary: normalized == primarySectorCode,
			}
			if context.matchedPrimary && sectorSnapshot != nil {
				context.matchedPrimarySector = strings.TrimSpace(sectorSnapshot.PrimarySectorName)
			}
			return context
		}
	}
	return nil
}

func contextualizeMacroSummary(item macroPolicySeed, match *macroMatchContext) string {
	if match == nil {
		return item.Summary
	}
	if match.matchedPrimaryTheme != "" {
		return item.Summary + " 当前主主题为" + match.matchedPrimaryTheme + "，匹配暴露约 " + macroWeightText(match.matchedWeight) + "，这类政策/产业信号与基金当前主线贴合度更高。"
	}
	if match.matchedPrimarySector != "" {
		return item.Summary + " 当前主行业为" + match.matchedPrimarySector + "，匹配暴露约 " + macroWeightText(match.matchedWeight) + "，这类政策/产业信号对组合解释更直接。"
	}
	if match.matchedName != "" {
		return item.Summary + " 当前组合中可映射到" + match.matchedName + "，匹配暴露约 " + macroWeightText(match.matchedWeight) + "；该热点仅按实际暴露权重参与解释。"
	}
	return item.Summary + " 当前组合中存在可映射暴露，匹配暴露约 " + macroWeightText(match.matchedWeight) + "；该热点仅按实际暴露权重参与解释。"
}

func normalizeMacroCode(code string) string {
	code = strings.TrimSpace(code)
	switch code {
	case "ai_compute":
		return "computing_power"
	default:
		return code
	}
}

func macroEventStrength(seedStrength string, matchedWeight float64) string {
	switch {
	case matchedWeight >= 35:
		return "high"
	case matchedWeight < 15:
		return "low"
	default:
		return strings.TrimSpace(seedStrength)
	}
}

func macroThemeName(snapshot *domain.FundThemeSnapshot, normalizedCode string) string {
	if snapshot == nil {
		return ""
	}
	if normalizeMacroCode(snapshot.PrimaryThemeCode) == normalizedCode {
		return strings.TrimSpace(snapshot.PrimaryThemeName)
	}
	for _, item := range snapshot.Breakdown {
		if normalizeMacroCode(item.ThemeCode) == normalizedCode {
			return strings.TrimSpace(item.ThemeName)
		}
	}
	return ""
}

func macroSectorName(snapshot *domain.FundSectorSnapshot, normalizedCode string) string {
	if snapshot == nil {
		return ""
	}
	if normalizeMacroCode(snapshot.PrimarySectorCode) == normalizedCode {
		return strings.TrimSpace(snapshot.PrimarySectorName)
	}
	for _, item := range snapshot.Breakdown {
		if normalizeMacroCode(item.SectorCode) == normalizedCode {
			return strings.TrimSpace(item.SectorName)
		}
	}
	return ""
}

func macroWeightText(weight float64) string {
	return decimalPointerFromFloat(weight).Round(1).StringFixed(1) + "%"
}
