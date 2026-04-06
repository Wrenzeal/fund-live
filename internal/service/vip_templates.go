package service

import (
	"strings"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
)

const (
	defaultVIPSectorReportID    = "sector-ai-manufacturing"
	defaultVIPMedicalReportID   = "sector-medical-innovation"
	defaultVIPPortfolioReportID = "portfolio-core-balance"
)

var defaultVIPReportDisclaimer = "投资有风险，报告内容基于公开信息与模拟数据整理，仅供参考，不构成投资建议。"

var vipSampleReports = map[string]domain.VIPReport{
	defaultVIPSectorReportID: {
		ID:             defaultVIPSectorReportID,
		Type:           domain.VIPTaskTypeSectorAnalysis,
		Title:          "AI 产业链主导板块分析",
		TargetName:     "智能制造主题自选分组",
		GeneratedAt:    "2026-04-04T10:35:00+08:00",
		CoverageWindow: "近 24 小时市场与公开资讯",
		RiskLevel:      domain.VIPRiskLevelMedium,
		Summary: domain.VIPReportSummary{
			Headline: "AI 算力与高端制造链条热度仍在，但短线交易拥挤度提升，适合偏低吸而非追高。",
			Bullets: []string{
				"外围科技资产维持偏强，风险偏好并未明显退潮。",
				"国内政策层面对先进制造和算力基础设施的支持方向仍然清晰。",
				"板块内个股财报分化扩大，强阿尔法公司与普通主题标的拉开差距。",
			},
		},
		Advice: domain.VIPAdvice{
			Action:        "低吸",
			PositionRange: "5%-10%",
			Conditions: []string{
				"若板块回撤后量能未明显失真，可分批低吸。",
				"若单日涨幅过大且北向资金流入放缓，优先观望而非追涨。",
			},
		},
		Macro: domain.VIPNarrativeSection{
			Title:   "世界局势与宏观环境",
			Content: "全球风险偏好仍围绕科技成长与制造升级展开，海外长端利率对高估值成长资产的压制边际减弱。",
			Bullets: []string{
				"美国核心科技资产维持强势，情绪外溢仍利好本地 AI 产业链。",
				"地缘摩擦尚未出现新的系统性冲击，但仍会影响高端制造出口预期。",
			},
		},
		Policy: domain.VIPNarrativeSection{
			Title:   "政策信息",
			Content: "国内政策口径仍偏向“新质生产力、先进制造、算力基础设施”，方向层面没有逆风。",
			Bullets: []string{
				"科技创新和工业升级仍是中期政策主线。",
				"若后续产业扶持细则落地，算力、设备与工业软件板块弹性可能更强。",
			},
		},
		Earnings: domain.VIPEarningsSection{
			Title: "财报与公司基本面",
			Companies: []domain.VIPEarningsCompany{
				{Name: "中际旭创", Note: "订单预期稳定，但短线市场预期已经较高。"},
				{Name: "工业富联", Note: "算力服务器业务仍是核心观察点，资金偏好较高。"},
				{Name: "海光信息", Note: "国产算力替代逻辑持续，但估值波动也更大。"},
			},
		},
		Market: domain.VIPNarrativeSection{
			Title:   "大盘与板块走势",
			Content: "市场风格仍偏成长，AI 与高端制造仍是高弹性方向，但短线拥挤度需要警惕。",
			Bullets: []string{
				"大盘成交额维持在高位区间，有利于主题方向延续。",
				"板块轮动速度加快，分化中更强调龙头质量而非泛主题覆盖。",
			},
		},
		Risks: []string{
			"若外围科技股回撤放大，高估值主题会先承压。",
			"若政策层面短期没有增量催化，板块可能进入高位震荡。",
			"若核心公司财报不及预期，板块整体风险偏好会快速降温。",
		},
		Sources: []domain.VIPReportSource{
			{
				ID:          "source-sector-1",
				Title:       "沪深两市成交额与板块强度日报",
				Type:        domain.VIPSourceTypeMarket,
				Publisher:   "FundLive Market Feed",
				PublishedAt: "2026-04-04T10:20:00+08:00",
				URL:         "https://example.com/market-daily",
				Snippet:     "成长风格继续占优，AI 与高端制造链维持高活跃度。",
			},
			{
				ID:          "source-sector-2",
				Title:       "先进制造与算力建设政策跟踪",
				Type:        domain.VIPSourceTypePolicy,
				Publisher:   "公开政策汇编",
				PublishedAt: "2026-04-03T20:00:00+08:00",
				URL:         "https://example.com/policy-ai",
				Snippet:     "政策方向延续对先进制造与算力基础设施的支持表述。",
			},
			{
				ID:          "source-sector-3",
				Title:       "算力产业链核心公司季报摘要",
				Type:        domain.VIPSourceTypeEarnings,
				Publisher:   "上市公司公告整理",
				PublishedAt: "2026-04-03T18:30:00+08:00",
				URL:         "https://example.com/earnings-ai",
				Snippet:     "板块龙头订单和利润率表现分化，市场更重视兑现能力。",
			},
		},
		FooterDisclaimer: defaultVIPReportDisclaimer,
	},
	defaultVIPMedicalReportID: {
		ID:             defaultVIPMedicalReportID,
		Type:           domain.VIPTaskTypeSectorAnalysis,
		Title:          "医药创新主导板块分析",
		TargetName:     "医药成长观察分组",
		GeneratedAt:    "2026-04-04T11:05:00+08:00",
		CoverageWindow: "近 24 小时公开资讯与板块走势",
		RiskLevel:      domain.VIPRiskLevelMedium,
		Summary: domain.VIPReportSummary{
			Headline: "医药板块仍处于修复区间，政策与估值具备支撑，但财报兑现决定反弹持续性。",
			Bullets: []string{
				"创新药与器械方向存在资金回流，但节奏偏慢。",
				"行业政策环境较前期更温和，估值压制有所缓解。",
				"板块更适合中期跟踪，短线不宜过度激进。",
			},
		},
		Advice: domain.VIPAdvice{
			Action:        "观望",
			PositionRange: "0%-5%",
			Conditions: []string{
				"若后续财报持续改善，可逐步提升跟踪仓位。",
				"若板块出现放量突破，再考虑从观望转向低吸。",
			},
		},
		Macro: domain.VIPNarrativeSection{
			Title:   "世界局势与宏观环境",
			Content: "全球风险偏好修复对医药成长有边际利好，但医药仍更多取决于自身产业与政策预期。",
			Bullets: []string{
				"外部宏观压力缓和后，成长行业估值修复空间略有释放。",
			},
		},
		Policy: domain.VIPNarrativeSection{
			Title:   "政策信息",
			Content: "政策环境已从极度压缩估值的阶段转向更稳定的观察期，行业情绪修复仍需时间。",
			Bullets: []string{
				"集采与监管预期边际稳定。",
				"创新支持政策仍是中长期重要变量。",
			},
		},
		Earnings: domain.VIPEarningsSection{
			Title: "财报与公司基本面",
			Companies: []domain.VIPEarningsCompany{
				{Name: "药明康德", Note: "订单与外部预期仍是市场焦点。"},
				{Name: "迈瑞医疗", Note: "稳健基本面仍是板块重要压舱石。"},
				{Name: "恒瑞医药", Note: "创新药管线兑现将影响估值修复斜率。"},
			},
		},
		Market: domain.VIPNarrativeSection{
			Title:   "大盘与板块走势",
			Content: "医药板块处于修复但未完全转强状态，短线更适合等待确认信号。",
			Bullets: []string{
				"板块量能仍弱于高热度科技赛道。",
				"低位反弹后的持续性仍需等待验证。",
			},
		},
		Risks: []string{
			"若医药财报继续弱于预期，反弹节奏会受阻。",
			"若市场风险偏好重新集中到高弹性赛道，医药可能再度边缘化。",
		},
		Sources: []domain.VIPReportSource{
			{
				ID:          "source-med-1",
				Title:       "医药板块交易热度追踪",
				Type:        domain.VIPSourceTypeMarket,
				Publisher:   "FundLive Market Feed",
				PublishedAt: "2026-04-04T10:50:00+08:00",
				URL:         "https://example.com/market-med",
				Snippet:     "板块回暖但量能尚未充分放大，风格偏修复。",
			},
			{
				ID:          "source-med-2",
				Title:       "创新药政策环境观察",
				Type:        domain.VIPSourceTypePolicy,
				Publisher:   "公开政策汇编",
				PublishedAt: "2026-04-03T19:15:00+08:00",
				URL:         "https://example.com/policy-med",
				Snippet:     "行业政策环境趋稳，创新支持方向仍是长期变量。",
			},
		},
		FooterDisclaimer: defaultVIPReportDisclaimer,
	},
	defaultVIPPortfolioReportID: {
		ID:             defaultVIPPortfolioReportID,
		Type:           domain.VIPTaskTypePortfolioAnalysis,
		Title:          "核心组合平衡分析报告",
		TargetName:     "全部持仓组合",
		GeneratedAt:    "2026-04-04T13:45:00+08:00",
		CoverageWindow: "近 24 小时市场、财报与公开资讯",
		RiskLevel:      domain.VIPRiskLevelMedium,
		Summary: domain.VIPReportSummary{
			Headline: "组合当前处于“成长驱动 + 医药修复”并存阶段，适合维持平衡偏积极而非极端押注。",
			Bullets: []string{
				"成长基金贡献弹性，但波动也更大。",
				"医药仓位为组合提供一定风格平衡，但修复节奏仍偏慢。",
				"大盘成交活跃有利于组合维持弹性，但短线仍需警惕高位回撤。",
			},
		},
		Advice: domain.VIPAdvice{
			Action:        "低吸",
			PositionRange: "5%-10%",
			Conditions: []string{
				"若市场维持高成交且成长方向未明显退潮，可分批增配。",
				"若核心基金连续快速拉升，则优先等待回撤后再补仓。",
			},
		},
		Macro: domain.VIPNarrativeSection{
			Title:   "世界局势与宏观环境",
			Content: "全球风险资产情绪对成长方向依然友好，但组合层面不宜忽视突发风险事件对波动的放大。",
			Bullets: []string{
				"外围成长资产偏强是组合风险偏好的正向变量。",
				"突发地缘与利率波动仍是组合回撤的主要外部触发器。",
			},
		},
		Policy: domain.VIPNarrativeSection{
			Title:   "政策信息",
			Content: "国内政策主线仍围绕先进制造、科技创新和稳增长，组合中的成长资产仍具备政策环境支撑。",
			Bullets: []string{
				"科技与制造方向中期逻辑未破坏。",
				"医药和消费方向更多依赖盈利修复与风格轮动。",
			},
		},
		Earnings: domain.VIPEarningsSection{
			Title: "财报与公司基本面",
			Companies: []domain.VIPEarningsCompany{
				{Name: "工业富联", Note: "成长仓位的关键景气风向标。"},
				{Name: "药明康德", Note: "医药仓位的修复预期核心样本。"},
				{Name: "贵州茅台", Note: "若组合含消费核心资产，其稳健性有助于平衡成长波动。"},
			},
		},
		Market: domain.VIPNarrativeSection{
			Title:   "大盘与板块走势",
			Content: "市场当前偏成长风格，组合层面应接受“收益弹性提升但短线回撤变大”的现实。",
			Bullets: []string{
				"指数层面偏强，但板块分化明显。",
				"组合更适合通过结构平衡而非单赛道重仓提升体验。",
			},
		},
		Risks: []string{
			"若组合成长仓位过高，会放大日内波动。",
			"若医药修复不及预期，组合平衡作用会弱化。",
			"若市场成交额回落，组合弹性会明显下降。",
		},
		Sources: []domain.VIPReportSource{
			{
				ID:          "source-port-1",
				Title:       "组合持仓风格拆解",
				Type:        domain.VIPSourceTypeMarket,
				Publisher:   "FundLive Portfolio Feed",
				PublishedAt: "2026-04-04T13:20:00+08:00",
				URL:         "https://example.com/portfolio-style",
				Snippet:     "组合当前成长暴露高于防御暴露，医药承担风格平衡作用。",
			},
			{
				ID:          "source-port-2",
				Title:       "宏观与政策环境周观察",
				Type:        domain.VIPSourceTypePolicy,
				Publisher:   "公开政策汇编",
				PublishedAt: "2026-04-03T21:00:00+08:00",
				URL:         "https://example.com/policy-weekly",
				Snippet:     "政策仍偏支持科技与制造升级，稳增长环境未发生明显逆转。",
			},
			{
				ID:          "source-port-3",
				Title:       "核心持仓公司财报摘要",
				Type:        domain.VIPSourceTypeEarnings,
				Publisher:   "上市公司公告整理",
				PublishedAt: "2026-04-03T18:00:00+08:00",
				URL:         "https://example.com/portfolio-earnings",
				Snippet:     "组合中的核心样本公司基本面分化，景气判断仍需逐项跟踪。",
			},
		},
		FooterDisclaimer: defaultVIPReportDisclaimer,
	},
}

func cloneVIPReportTemplate(templateID string) (*domain.VIPReport, bool) {
	report, ok := vipSampleReports[templateID]
	if !ok {
		return nil, false
	}

	copyReport := report
	copyReport.Summary.Bullets = append([]string(nil), report.Summary.Bullets...)
	copyReport.Advice.Conditions = append([]string(nil), report.Advice.Conditions...)
	copyReport.Macro.Bullets = append([]string(nil), report.Macro.Bullets...)
	copyReport.Policy.Bullets = append([]string(nil), report.Policy.Bullets...)
	copyReport.Market.Bullets = append([]string(nil), report.Market.Bullets...)
	copyReport.Risks = append([]string(nil), report.Risks...)
	copyReport.Earnings.Companies = append([]domain.VIPEarningsCompany(nil), report.Earnings.Companies...)
	copyReport.Sources = append([]domain.VIPReportSource(nil), report.Sources...)
	return &copyReport, true
}

func resolveVIPTemplateID(taskType domain.VIPTaskType, targetName string) string {
	switch taskType {
	case domain.VIPTaskTypeSectorAnalysis:
		if containsMedicalKeyword(targetName) {
			return defaultVIPMedicalReportID
		}
		return defaultVIPSectorReportID
	default:
		return defaultVIPPortfolioReportID
	}
}

func containsMedicalKeyword(targetName string) bool {
	return containsAny(targetName, []string{"医", "药", "医疗"})
}

func containsAny(target string, keywords []string) bool {
	for _, keyword := range keywords {
		if keyword != "" && len(target) > 0 && strings.Contains(target, keyword) {
			return true
		}
	}
	return false
}

func overrideVIPReportTemplate(base *domain.VIPReport, reportID, targetName string, generatedAt time.Time) *domain.VIPReport {
	if base == nil {
		return nil
	}

	copyReport := *base
	copyReport.ID = reportID
	if targetName != "" {
		copyReport.TargetName = targetName
	}
	copyReport.GeneratedAt = generatedAt.In(tradingLocation()).Format(time.RFC3339)
	return &copyReport
}
