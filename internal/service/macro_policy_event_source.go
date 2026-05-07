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
}

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
}

func LoadMacroPolicyEvents(now time.Time, sectorSnapshot *domain.FundSectorSnapshot, themeSnapshot *domain.FundThemeSnapshot) []domain.FundAnalysisEventImpact {
	if now.IsZero() {
		now = time.Now()
	}

	activeSectorCodes := make(map[string]struct{})
	activeThemeCodes := make(map[string]struct{})
	if sectorSnapshot != nil {
		if code := strings.TrimSpace(sectorSnapshot.PrimarySectorCode); code != "" {
			activeSectorCodes[normalizeMacroCode(code)] = struct{}{}
		}
		for _, item := range sectorSnapshot.Breakdown {
			if code := strings.TrimSpace(item.SectorCode); code != "" {
				activeSectorCodes[normalizeMacroCode(code)] = struct{}{}
			}
		}
	}
	if themeSnapshot != nil {
		if code := strings.TrimSpace(themeSnapshot.PrimaryThemeCode); code != "" {
			activeThemeCodes[normalizeMacroCode(code)] = struct{}{}
		}
		for _, item := range themeSnapshot.Breakdown {
			if code := strings.TrimSpace(item.ThemeCode); code != "" {
				activeThemeCodes[normalizeMacroCode(code)] = struct{}{}
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
			Strength:    item.Strength,
			Horizon:     "current",
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
	sectorCodes map[string]struct{},
	themeCodes map[string]struct{},
	sectorSnapshot *domain.FundSectorSnapshot,
	themeSnapshot *domain.FundThemeSnapshot,
) *macroMatchContext {
	context := &macroMatchContext{}
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
		if _, ok := themeCodes[normalized]; ok {
			if normalized == primaryThemeCode && themeSnapshot != nil {
				context.matchedPrimaryTheme = strings.TrimSpace(themeSnapshot.PrimaryThemeName)
			}
			return context
		}
	}
	for _, code := range item.SectorCodes {
		normalized := normalizeMacroCode(strings.TrimSpace(code))
		if _, ok := sectorCodes[normalized]; ok {
			if normalized == primarySectorCode && sectorSnapshot != nil {
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
		return item.Summary + " 当前主主题为" + match.matchedPrimaryTheme + "，这类政策/产业信号与基金当前主线贴合度更高。"
	}
	if match.matchedPrimarySector != "" {
		return item.Summary + " 当前主行业为" + match.matchedPrimarySector + "，这类政策/产业信号对组合解释更直接。"
	}
	return item.Summary
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
