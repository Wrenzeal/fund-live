package service

import (
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
)

func LoadIndexLayerEvents(
	now time.Time,
	fund *domain.Fund,
	holdingsSource string,
	sectorSnapshot *domain.FundSectorSnapshot,
	themeSnapshot *domain.FundThemeSnapshot,
) []domain.FundAnalysisEventImpact {
	if now.IsZero() {
		now = time.Now()
	}
	if !isIndexLikeFund(fund, holdingsSource) {
		return nil
	}

	nextWindow := nextIndexRebalanceWindow(now)
	daysUntil := int(nextWindow.Sub(now).Hours() / 24)
	if daysUntil < 0 || daysUntil > 60 {
		return nil
	}

	indexName := deriveIndexEventName(fund, sectorSnapshot, themeSnapshot)
	expectedAt := nextWindow
	return []domain.FundAnalysisEventImpact{
		{
			Code:         "index_rebalance_window_" + nextWindow.Format("20060102"),
			Title:        indexName + "调样窗口临近",
			Impact:       "neutral",
			Summary:      "指数层面通常会在样本调整窗口附近更新成分股与权重；若近期跟踪指数进入调样窗口，结构解释需留意成分与权重变化。",
			TargetScope:  "index",
			Strength:     indexWindowStrength(daysUntil),
			Horizon:      "current",
			EventType:    "index_rebalance",
			EventStatus:  "expected",
			ExpectedAt:   &expectedAt,
			SourceTier:   "heuristic",
			KnownAtBasis: "derived_calendar",
		},
	}
}

func isIndexLikeFund(fund *domain.Fund, holdingsSource string) bool {
	if fund == nil {
		return false
	}
	name := strings.ToLower(strings.TrimSpace(fund.Name))
	fundType := strings.ToLower(strings.TrimSpace(fund.Type))
	if strings.Contains(fundType, "index") || strings.Contains(fundType, "指数") {
		return true
	}
	if strings.Contains(name, "指数") || strings.Contains(name, "etf") || strings.Contains(name, "联接") {
		return true
	}
	return strings.TrimSpace(holdingsSource) == SectorSourceTargetETFFallback
}

func deriveIndexEventName(fund *domain.Fund, sectorSnapshot *domain.FundSectorSnapshot, themeSnapshot *domain.FundThemeSnapshot) string {
	if hasRecognizedTheme(themeSnapshot) && strings.TrimSpace(themeSnapshot.PrimaryThemeName) != "" {
		return strings.TrimSpace(themeSnapshot.PrimaryThemeName)
	}
	if hasRecognizedSector(sectorSnapshot) && strings.TrimSpace(sectorSnapshot.PrimarySectorName) != "" {
		return strings.TrimSpace(sectorSnapshot.PrimarySectorName)
	}
	if fund != nil {
		name := strings.TrimSpace(fund.Name)
		name = strings.ReplaceAll(name, "ETF联接", "")
		name = strings.ReplaceAll(name, "ETF", "")
		name = strings.ReplaceAll(name, "指数", "")
		name = strings.ReplaceAll(name, "联接", "")
		name = strings.TrimSpace(name)
		if name != "" {
			return name
		}
	}
	return "跟踪指数"
}

func nextIndexRebalanceWindow(now time.Time) time.Time {
	loc := now.Location()
	year := now.Year()
	candidates := []time.Time{
		secondFriday(year, time.June, loc),
		secondFriday(year, time.December, loc),
		secondFriday(year+1, time.June, loc),
	}
	for _, candidate := range candidates {
		if candidate.After(now.Add(-7 * 24 * time.Hour)) {
			return candidate
		}
	}
	return candidates[len(candidates)-1]
}

func secondFriday(year int, month time.Month, loc *time.Location) time.Time {
	d := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	for d.Weekday() != time.Friday {
		d = d.AddDate(0, 0, 1)
	}
	return d.AddDate(0, 0, 7)
}

func indexWindowStrength(daysUntil int) string {
	switch {
	case daysUntil <= 14:
		return "high"
	case daysUntil <= 35:
		return "medium"
	default:
		return "low"
	}
}
