import type { FundAnalysis, FundHoldingRecord, FundSectorSnapshot, FundThemeSnapshot } from '@/hooks/use-fund-data'
import type {
  HoldingAggregateEntry,
  HoldingEntry,
  HoldingEstimateAggregateMetrics,
} from '@/hooks/use-user-portfolio'
import { dominantAnalysisRecommendation } from '@/lib/fund-analysis-display'
import {
  aggregateChangeValue,
  aggregateProfitValue,
  parseOptionalNumber,
  type HoldingMetricScope,
} from '@/lib/holding-display'
import { resolveHoldingSourceLabel } from '@/lib/holding-sources'

export type InsightTone = 'good' | 'info' | 'warning' | 'danger'
export type ReconciliationSeverity = 'ok' | 'watch' | 'review' | 'pending'

export interface HoldingReconciliationItem {
  id: string
  fundID: string
  fundName: string
  amount: number
  impliedPrincipal: number | null
  difference: number | null
  differencePercent: number | null
  severity: ReconciliationSeverity
  manual: boolean
  sourceLabel?: string
  reason: string
}

export interface HoldingReconciliationSummary {
  total: number
  manualCount: number
  pendingCount: number
  watchCount: number
  reviewCount: number
  okCount: number
  items: HoldingReconciliationItem[]
}

export interface PortfolioHealthSignal {
  id: string
  title: string
  description: string
  tone: InsightTone
  metric?: string
  action?: string
}

export interface PortfolioHealthSummary {
  score: number
  tone: InsightTone
  title: string
  description: string
  signals: PortfolioHealthSignal[]
}

export interface HoldingExposureSnapshot {
  sectorSnapshot?: FundSectorSnapshot
  themeSnapshot?: FundThemeSnapshot
}

export interface HoldingReminderItem {
  id: string
  title: string
  description: string
  tone: InsightTone
  metric?: string
  action?: string
}

export interface HoldingReminderSummary {
  urgentCount: number
  reminders: HoldingReminderItem[]
}

function parseNumber(value?: string) {
  const parsed = Number.parseFloat(value || '')
  return Number.isFinite(parsed) ? parsed : null
}

function fundNameFromHolding(holding: HoldingEntry) {
  return holding.fund?.name || holding.fund_id
}

function fundNameFromAggregate(aggregate: HoldingAggregateEntry) {
  return aggregate.fund?.name || aggregate.fund_id
}

function severityRank(severity: ReconciliationSeverity) {
  switch (severity) {
    case 'review':
      return 4
    case 'watch':
      return 3
    case 'pending':
      return 2
    case 'ok':
    default:
      return 1
  }
}

export function buildHoldingReconciliationSummary(holdings: HoldingEntry[]): HoldingReconciliationSummary {
  const items = holdings.map<HoldingReconciliationItem>((holding) => {
    const amount = parseNumber(holding.amount) ?? 0
    const shares = parseNumber(holding.shares)
    const confirmedNav = parseNumber(holding.confirmed_nav)
    const manual = holding.manual_confirmation === true
    const sourceLabel = resolveHoldingSourceLabel(holding.source_platform, holding.source_label)
    const sourceText = sourceLabel || '外部平台'

    if (!shares || !confirmedNav || amount <= 0) {
      return {
        id: holding.id,
        fundID: holding.fund_id,
        fundName: fundNameFromHolding(holding),
        amount,
        impliedPrincipal: null,
        difference: null,
        differencePercent: null,
        severity: 'pending',
        manual,
        sourceLabel,
        reason: '确认份额或确认净值暂未齐备，无法做本金与份额口径对账。',
      }
    }

    const impliedPrincipal = shares * confirmedNav
    const difference = amount - impliedPrincipal
    const differencePercent = amount > 0 ? (difference / amount) * 100 : 0
    const absDifference = Math.abs(difference)
    const absDifferencePercent = Math.abs(differencePercent)
    const tolerance = Math.max(1, amount * 0.001)

    if (absDifference <= tolerance) {
      return {
        id: holding.id,
        fundID: holding.fund_id,
        fundName: fundNameFromHolding(holding),
        amount,
        impliedPrincipal,
        difference,
        differencePercent,
        severity: 'ok',
        manual,
        sourceLabel,
        reason: manual ? `已按${sourceText}校正，份额 × 净值 与本金基本一致。` : '系统自动确认口径与本金基本一致。',
      }
    }

    if (absDifferencePercent <= 0.5) {
      return {
        id: holding.id,
        fundID: holding.fund_id,
        fundName: fundNameFromHolding(holding),
        amount,
        impliedPrincipal,
        difference,
        differencePercent,
        severity: 'watch',
        manual,
        sourceLabel,
        reason: '存在小幅差异，常见原因是平台四舍五入、手续费或确认净值精度不同。',
      }
    }

    return {
      id: holding.id,
      fundID: holding.fund_id,
      fundName: fundNameFromHolding(holding),
      amount,
      impliedPrincipal,
      difference,
      differencePercent,
      severity: 'review',
      manual,
      sourceLabel,
      reason: '本金与“份额 × 确认净值”差异偏大，建议核对确认日期、份额或是否存在分红 / 赎回 / 平台迁移口径。',
    }
  })

  return {
    total: holdings.length,
    manualCount: items.filter((item) => item.manual).length,
    pendingCount: items.filter((item) => item.severity === 'pending').length,
    watchCount: items.filter((item) => item.severity === 'watch').length,
    reviewCount: items.filter((item) => item.severity === 'review').length,
    okCount: items.filter((item) => item.severity === 'ok').length,
    items: items.sort((left, right) => {
      const severityDiff = severityRank(right.severity) - severityRank(left.severity)
      if (severityDiff !== 0) {
        return severityDiff
      }
      return Math.abs(right.differencePercent ?? 0) - Math.abs(left.differencePercent ?? 0)
    }),
  }
}

function formatPercentMetric(value: number) {
  return `${value.toFixed(1)}%`
}

function formatMoneyMetric(value: number) {
  return new Intl.NumberFormat('zh-CN', {
    style: 'currency',
    currency: 'CNY',
    maximumFractionDigits: 0,
  }).format(value)
}

function signalWeight(tone: InsightTone) {
  switch (tone) {
    case 'danger':
      return 24
    case 'warning':
      return 14
    case 'info':
      return 5
    case 'good':
    default:
      return -4
  }
}

function addExposure(bucket: Map<string, number>, name: string | undefined, amount: number) {
  const normalizedName = (name || '').trim()
  if (!normalizedName || amount <= 0) {
    return
  }
  bucket.set(normalizedName, (bucket.get(normalizedName) ?? 0) + amount)
}

function topExposure(bucket: Map<string, number>) {
  return Array.from(bucket.entries()).sort((left, right) => right[1] - left[1])[0] ?? null
}

function primaryStyleLabel(
  aggregate: HoldingAggregateEntry,
  snapshot?: HoldingExposureSnapshot,
  analysis?: FundAnalysis | null,
) {
  const fund = aggregate.fund
  const text = [
    fund?.name,
    fund?.type,
    fund?.category_name,
    snapshot?.sectorSnapshot?.primary_sector_name,
    snapshot?.themeSnapshot?.primary_theme_name,
    analysis?.analysis_type,
  ].filter(Boolean).join(' ').toLowerCase()

  if (/qdii|海外|纳斯达克|标普|恒生|港股|美股|全球/.test(text)) {
    return '海外 / QDII'
  }
  if (/医药|医疗|生物|创新药/.test(text)) {
    return '医药健康'
  }
  if (/科技|芯片|半导体|计算机|ai|人工智能|通信|电子|算力|云计算/.test(text)) {
    return '科技成长'
  }
  if (/红利|高股息|银行|金融|保险|证券|非银/.test(text)) {
    return '红利金融'
  }
  if (/消费|白酒|食品|饮料|家电/.test(text)) {
    return '消费升级'
  }
  if (/新能源|光伏|电池|电力设备/.test(text)) {
    return '新能源'
  }
  if (/债|货币|固收|短融/.test(text)) {
    return '稳健固收'
  }
  return '宽基 / 均衡'
}

function analysisRelatedSymbols(analysis?: FundAnalysis | null) {
  const symbols = new Set<string>()
  for (const event of analysis?.event_impacts ?? []) {
    for (const symbol of event.related_symbols ?? []) {
      const normalized = symbol.trim()
      if (normalized) {
        symbols.add(normalized)
      }
    }
  }
  for (const evidence of analysis?.primary_evidence ?? []) {
    for (const symbol of evidence.related_symbols ?? []) {
      const normalized = symbol.trim()
      if (normalized) {
        symbols.add(normalized)
      }
    }
  }
  return Array.from(symbols)
}

export function buildPortfolioHealthSummary(params: {
  aggregates: HoldingAggregateEntry[]
  holdings: HoldingEntry[]
  analysesByFundID: Record<string, FundAnalysis | null>
  aggregateMetrics: Record<string, HoldingEstimateAggregateMetrics>
  metricScope: HoldingMetricScope
  exposureSnapshots?: Record<string, HoldingExposureSnapshot>
  topHoldingsByFundID?: Record<string, FundHoldingRecord[]>
}): PortfolioHealthSummary {
  const {
    aggregates,
    holdings,
    analysesByFundID,
    aggregateMetrics,
    metricScope,
    exposureSnapshots = {},
    topHoldingsByFundID = {},
  } = params
  if (holdings.length === 0 || aggregates.length === 0) {
    return {
      score: 0,
      tone: 'info',
      title: '等待第一笔持仓',
      description: '记录持仓后会自动生成组合体检。',
      signals: [],
    }
  }

  const totalPrincipal = aggregates.reduce((sum, aggregate) => sum + (parseOptionalNumber(aggregate.total_principal) ?? 0), 0)
  const sortedByPrincipal = aggregates
    .slice()
    .sort((left, right) => (parseOptionalNumber(right.total_principal) ?? 0) - (parseOptionalNumber(left.total_principal) ?? 0))
  const largest = sortedByPrincipal[0]
  const largestPrincipal = parseOptionalNumber(largest?.total_principal) ?? 0
  const largestShare = totalPrincipal > 0 ? (largestPrincipal / totalPrincipal) * 100 : 0
  const topThreePrincipal = sortedByPrincipal.slice(0, 3).reduce((sum, aggregate) => sum + (parseOptionalNumber(aggregate.total_principal) ?? 0), 0)
  const topThreeShare = totalPrincipal > 0 ? (topThreePrincipal / totalPrincipal) * 100 : 0
  const incompleteCount = holdings.filter((holding) => !holding.shares || !holding.confirmed_nav_date || !holding.real_metrics_ready).length
  const manualCount = holdings.filter((holding) => holding.manual_confirmation).length
  const multiLotCount = aggregates.filter((aggregate) => aggregate.holding_count > 1).length
  const highRiskAggregates = aggregates.filter((aggregate) => analysesByFundID[aggregate.fund_id]?.risk_level === 'high')
  const decreaseAggregates = aggregates.filter((aggregate) => {
    const analysis = analysesByFundID[aggregate.fund_id]
    return analysis ? dominantAnalysisRecommendation(analysis) === 'decrease' : false
  })
  const profitEntries = aggregates
    .map((aggregate) => ({
      aggregate,
      profit: aggregateProfitValue(aggregate, aggregateMetrics[aggregate.fund_id], metricScope),
    }))
    .filter((entry): entry is { aggregate: HoldingAggregateEntry; profit: number } => entry.profit !== null)
    .sort((left, right) => Math.abs(right.profit) - Math.abs(left.profit))
  const sectorExposure = new Map<string, number>()
  const themeExposure = new Map<string, number>()
  const styleExposure = new Map<string, number>()
  const primaryThemeGroups = new Map<string, { amount: number; count: number }>()
  const relatedSymbolOwners = new Map<string, Set<string>>()
  const topStockOwners = new Map<string, { name: string; owners: Set<string>; weightedExposure: number }>()

  for (const aggregate of aggregates) {
    const principal = parseOptionalNumber(aggregate.total_principal) ?? 0
    const snapshot = exposureSnapshots[aggregate.fund_id]
    const sectorBreakdown = snapshot?.sectorSnapshot?.breakdown ?? []
    const themeBreakdown = snapshot?.themeSnapshot?.breakdown ?? []

    if (sectorBreakdown.length > 0) {
      for (const item of sectorBreakdown) {
        addExposure(sectorExposure, item.sector_name, principal * ((parseOptionalNumber(item.weight_percent) ?? 0) / 100))
      }
    } else {
      addExposure(sectorExposure, snapshot?.sectorSnapshot?.primary_sector_name, principal)
    }

    if (themeBreakdown.length > 0) {
      for (const item of themeBreakdown) {
        addExposure(themeExposure, item.theme_name, principal * ((parseOptionalNumber(item.weight_percent) ?? 0) / 100))
      }
    } else {
      addExposure(themeExposure, snapshot?.themeSnapshot?.primary_theme_name, principal)
    }

    const styleLabel = primaryStyleLabel(aggregate, snapshot, analysesByFundID[aggregate.fund_id])
    addExposure(styleExposure, styleLabel, principal)

    const primaryTheme = snapshot?.themeSnapshot?.primary_theme_name || snapshot?.sectorSnapshot?.primary_sector_name
    if (primaryTheme && principal > 0) {
      const current = primaryThemeGroups.get(primaryTheme) ?? { amount: 0, count: 0 }
      primaryThemeGroups.set(primaryTheme, {
        amount: current.amount + principal,
        count: current.count + 1,
      })
    }

    for (const symbol of analysisRelatedSymbols(analysesByFundID[aggregate.fund_id])) {
      if (!relatedSymbolOwners.has(symbol)) {
        relatedSymbolOwners.set(symbol, new Set())
      }
      relatedSymbolOwners.get(symbol)?.add(aggregate.fund_id)
    }

    for (const holding of (topHoldingsByFundID[aggregate.fund_id] ?? []).slice(0, 10)) {
      const stockName = holding.stock_name?.trim()
      const stockCode = holding.stock_code?.trim()
      if (!stockName && !stockCode) {
        continue
      }
      const key = stockCode || stockName || ''
      const ratio = parseOptionalNumber(holding.holding_ratio) ?? 0
      const current = topStockOwners.get(key) ?? {
        name: stockName || stockCode || key,
        owners: new Set<string>(),
        weightedExposure: 0,
      }
      current.owners.add(aggregate.fund_id)
      current.weightedExposure += principal * Math.max(0, ratio) / 100
      topStockOwners.set(key, current)
    }
  }

  const signals: PortfolioHealthSignal[] = []

  if (largest && largestShare >= 50) {
    signals.push({
      id: 'largest-position-danger',
      title: '单只基金仓位过高',
      description: `${fundNameFromAggregate(largest)} 占总本金约 ${formatPercentMetric(largestShare)}，单一基金波动会明显影响组合。`,
      tone: 'danger',
      metric: formatPercentMetric(largestShare),
      action: '可优先复核这只基金的量化风险和行业暴露。',
    })
  } else if (largest && largestShare >= 30) {
    signals.push({
      id: 'largest-position-warning',
      title: '单只基金占比较高',
      description: `${fundNameFromAggregate(largest)} 占总本金约 ${formatPercentMetric(largestShare)}，建议确认这是主动配置而不是历史遗留。`,
      tone: 'warning',
      metric: formatPercentMetric(largestShare),
      action: '可用“仓位最大优先”排序继续查看。',
    })
  } else {
    signals.push({
      id: 'largest-position-good',
      title: '单只基金集中度可控',
      description: largest ? `最大单只基金占比约 ${formatPercentMetric(largestShare)}。` : '暂无过度集中的单只基金。',
      tone: 'good',
    })
  }

  if (topThreeShare >= 80 && aggregates.length >= 3) {
    signals.push({
      id: 'top-three-concentration',
      title: '前三仓位较集中',
      description: `前三只基金合计占总本金约 ${formatPercentMetric(topThreeShare)}，组合收益主要由少数基金决定。`,
      tone: topThreeShare >= 90 ? 'danger' : 'warning',
      metric: formatPercentMetric(topThreeShare),
      action: '后续可结合行业 / 主题暴露看是否重复。',
    })
  }

  if (highRiskAggregates.length > 0) {
    signals.push({
      id: 'high-risk-analysis',
      title: '存在高风险量化标签',
      description: `${highRiskAggregates.length} 只基金当前量化风险为高，建议优先查看事件和反方证据。`,
      tone: 'danger',
      metric: `${highRiskAggregates.length} 只`,
      action: '可切换“风险等级优先”排序。',
    })
  } else if (decreaseAggregates.length > 0) {
    signals.push({
      id: 'decrease-analysis',
      title: '存在风险偏高信号',
      description: `${decreaseAggregates.length} 只基金当前结构更偏谨慎。`,
      tone: 'warning',
      metric: `${decreaseAggregates.length} 只`,
      action: '可切换“风险等级优先”排序。',
    })
  }

  if (incompleteCount > 0) {
    signals.push({
      id: 'incomplete-data',
      title: '仍有持仓待补齐',
      description: `${incompleteCount}/${holdings.length} 笔持仓缺少确认份额、确认净值或最新官方口径。`,
      tone: incompleteCount === holdings.length ? 'danger' : 'warning',
      metric: `${incompleteCount} 笔`,
      action: '可打开“只看待补齐”快速处理。',
    })
  } else {
    signals.push({
      id: 'complete-data',
      title: '持仓口径已就绪',
      description: '所有分笔都具备确认份额与当前口径，可用于组合级追踪。',
      tone: 'good',
    })
  }

  if (manualCount > 0) {
    const sourceLabels = Array.from(new Set(
      holdings
        .filter((holding) => holding.manual_confirmation)
        .map((holding) => resolveHoldingSourceLabel(holding.source_platform, holding.source_label))
        .filter(Boolean)
    ))
    const sourceText = sourceLabels.length > 0 ? sourceLabels.slice(0, 3).join(' / ') : '支付宝 / 微信等外部平台'
    signals.push({
      id: 'manual-correction',
      title: '存在平台校正记录',
      description: `${manualCount} 笔持仓已按${sourceText}校正，系统不会自动覆盖这些确认口径。`,
      tone: 'info',
      metric: `${manualCount} 笔`,
      action: '建议保留来源和备注，方便后续对账或筛选流水。',
    })
  }

  if (multiLotCount > 0) {
    signals.push({
      id: 'multi-lot',
      title: '存在多笔同基金记录',
      description: `${multiLotCount} 只基金由多笔记录组成，适合保留分笔，方便后续区分补仓 / 定投 / 校正。`,
      tone: 'info',
      metric: `${multiLotCount} 只`,
    })
  }

  if (profitEntries.length > 0) {
    const lead = profitEntries[0]
    signals.push({
      id: 'profit-driver',
      title: lead.profit >= 0 ? '今日主要贡献来源' : '今日主要拖累来源',
      description: `${fundNameFromAggregate(lead.aggregate)} 当前${lead.profit >= 0 ? '贡献' : '拖累'}约 ${formatMoneyMetric(Math.abs(lead.profit))}。`,
      tone: lead.profit >= 0 ? 'good' : 'warning',
      metric: `${lead.profit >= 0 ? '+' : '-'}${formatMoneyMetric(Math.abs(lead.profit))}`,
      action: metricScope === 'official' ? '官方口径下的主要波动来源。' : '盘中预估口径，夜间会被官方净值覆盖。',
    })
  }

  const largestSector = topExposure(sectorExposure)
  if (largestSector && totalPrincipal > 0) {
    const sectorShare = (largestSector[1] / totalPrincipal) * 100
    if (sectorShare >= 45) {
      signals.push({
        id: 'sector-concentration',
        title: '行业集中度偏高',
        description: `${largestSector[0]} 相关暴露约占本金 ${formatPercentMetric(sectorShare)}，行业事件会更容易影响组合净值。`,
        tone: sectorShare >= 60 ? 'danger' : 'warning',
        metric: formatPercentMetric(sectorShare),
        action: '后续可结合最近行业事件和持仓重仓股一起复核。',
      })
    } else {
      signals.push({
        id: 'sector-balanced',
        title: '行业暴露相对分散',
        description: `当前最大行业暴露为 ${largestSector[0]}，约 ${formatPercentMetric(sectorShare)}。`,
        tone: 'good',
      })
    }
  }

  const largestTheme = topExposure(themeExposure)
  if (largestTheme && totalPrincipal > 0) {
    const themeShare = (largestTheme[1] / totalPrincipal) * 100
    if (themeShare >= 40) {
      signals.push({
        id: 'theme-concentration',
        title: '主题暴露需要留意',
        description: `${largestTheme[0]} 主题约占本金 ${formatPercentMetric(themeShare)}，主题行情或政策变化会集中反映到组合。`,
        tone: themeShare >= 55 ? 'danger' : 'warning',
        metric: formatPercentMetric(themeShare),
      })
    }
  }

  const duplicateTheme = Array.from(primaryThemeGroups.entries())
    .filter(([, group]) => group.count >= 2 && totalPrincipal > 0 && (group.amount / totalPrincipal) * 100 >= 35)
    .sort((left, right) => right[1].amount - left[1].amount)[0]
  if (duplicateTheme) {
    signals.push({
      id: 'duplicate-exposure',
      title: '同类基金重复暴露',
      description: `${duplicateTheme[1].count} 只基金主暴露都指向 ${duplicateTheme[0]}，合计约 ${formatPercentMetric((duplicateTheme[1].amount / totalPrincipal) * 100)}。`,
      tone: 'warning',
      metric: `${duplicateTheme[1].count} 只`,
      action: '如果不是有意加仓同一主线，可考虑对比费率、规模、跟踪误差或基金经理差异。',
    })
  }

  const overlappedTopStocks = Array.from(topStockOwners.entries())
    .filter(([, overlap]) => overlap.owners.size >= 2)
    .sort((left, right) => right[1].weightedExposure - left[1].weightedExposure)
    .slice(0, 3)
  if (overlappedTopStocks.length > 0) {
    const leadingNames = overlappedTopStocks.map(([, overlap]) => overlap.name).join('、')
    const totalWeightedExposure = overlappedTopStocks.reduce((sum, [, overlap]) => sum + overlap.weightedExposure, 0)
    signals.push({
      id: 'top-holding-overlap',
      title: 'Top10 重仓股穿透重叠',
      description: `基金持仓明细显示 ${leadingNames} 同时出现在多只基金 Top10 中，说明组合可能在底层股票上重复暴露。`,
      tone: totalWeightedExposure / Math.max(totalPrincipal, 1) >= 0.12 ? 'warning' : 'info',
      metric: formatMoneyMetric(totalWeightedExposure),
      action: '这是基于基金持仓明细的穿透结果，比单纯事件标的提示更可信；可结合仓位决定是否需要分散。',
    })
  }

  const overlappedSymbols = Array.from(relatedSymbolOwners.entries())
    .filter(([, owners]) => owners.size >= 2)
    .slice(0, 3)
  if (overlappedTopStocks.length === 0 && overlappedSymbols.length > 0) {
    signals.push({
      id: 'related-symbol-overlap',
      title: '重仓 / 事件标的有重叠',
      description: `量化事件中重复出现 ${overlappedSymbols.map(([symbol]) => symbol).join('、')}，说明多只基金可能受同一批股票事件影响。`,
      tone: 'info',
      metric: `${overlappedSymbols.length} 个`,
      action: '这是缺少完整 Top10 重叠时的补充信号，优先级低于基金持仓明细穿透。',
    })
  }

  const largestStyle = topExposure(styleExposure)
  if (largestStyle && totalPrincipal > 0) {
    signals.push({
      id: 'style-portrait',
      title: '组合风格画像',
      description: `当前组合最主要风格是 ${largestStyle[0]}，约 ${formatPercentMetric((largestStyle[1] / totalPrincipal) * 100)}。`,
      tone: 'info',
      metric: largestStyle[0],
    })
  }

  const penalty = signals.reduce((sum, signal) => sum + signalWeight(signal.tone), 0)
  const score = Math.max(0, Math.min(100, Math.round(88 - penalty)))
  const tone: InsightTone = score >= 80 ? 'good' : score >= 65 ? 'info' : score >= 45 ? 'warning' : 'danger'
  const title = tone === 'good'
    ? '组合状态较稳'
    : tone === 'info'
      ? '组合可继续观察'
      : tone === 'warning'
        ? '组合有需要复核的地方'
        : '组合需要优先复核'

  return {
    score,
    tone,
    title,
    description: `基于 ${holdings.length} 笔持仓、${aggregates.length} 只基金的仓位、量化风险、数据就绪度和今日波动生成。`,
    signals: signals.slice(0, 8),
  }
}

export function buildHoldingReminderSummary(params: {
  holdings: HoldingEntry[]
  aggregates: HoldingAggregateEntry[]
  analysesByFundID: Record<string, FundAnalysis | null>
  aggregateMetrics: Record<string, HoldingEstimateAggregateMetrics>
  metricScope: HoldingMetricScope
  exposureSnapshots?: Record<string, HoldingExposureSnapshot>
}): HoldingReminderSummary {
  const { holdings, aggregates, analysesByFundID, aggregateMetrics, metricScope, exposureSnapshots = {} } = params
  const reminders: HoldingReminderItem[] = []
  const incompleteCount = holdings.filter((holding) => !holding.real_metrics_ready || !holding.shares || !holding.confirmed_nav_date).length
  if (incompleteCount > 0) {
    reminders.push({
      id: 'official-nav-pending',
      title: '官方净值 / 份额仍待补齐',
      description: `${incompleteCount} 笔持仓还没有完整官方口径，夜间净值同步或手动校正后会更可信。`,
      tone: incompleteCount === holdings.length ? 'danger' : 'warning',
      metric: `${incompleteCount} 笔`,
      action: '可打开“只看待补齐”逐笔处理。',
    })
  }

  const dropEntry = aggregates
    .map((aggregate) => ({
      aggregate,
      change: aggregateChangeValue(aggregate, aggregateMetrics[aggregate.fund_id], metricScope),
    }))
    .filter((entry): entry is { aggregate: HoldingAggregateEntry; change: number } => entry.change !== null)
    .sort((left, right) => left.change - right.change)[0]
  if (dropEntry && dropEntry.change <= -3) {
    reminders.push({
      id: 'large-drop',
      title: '单只持仓跌幅触发关注',
      description: `${fundNameFromAggregate(dropEntry.aggregate)} 当前${metricScope === 'official' ? '官方' : '盘中预估'}涨跌幅约 ${dropEntry.change.toFixed(2)}%。`,
      tone: dropEntry.change <= -5 ? 'danger' : 'warning',
      metric: `${dropEntry.change.toFixed(2)}%`,
      action: '建议结合量化风险、事件和持仓仓位决定是否复核。',
    })
  }

  const riskFunds = aggregates.filter((aggregate) => {
    const analysis = analysesByFundID[aggregate.fund_id]
    return analysis?.risk_level === 'high' || (analysis ? dominantAnalysisRecommendation(analysis) === 'decrease' : false)
  })
  if (riskFunds.length > 0) {
    reminders.push({
      id: 'analysis-risk',
      title: '量化风险信号需要复核',
      description: `${riskFunds.length} 只基金存在高风险或风险偏高结论，优先查看反方证据和最近事件。`,
      tone: 'warning',
      metric: `${riskFunds.length} 只`,
      action: '可切换“风险等级优先”排序。',
    })
  }

  const staleManual = holdings.filter((holding) => {
    if (!holding.manual_confirmation) {
      return false
    }
    const parsed = new Date(holding.updated_at)
    if (Number.isNaN(parsed.getTime())) {
      return false
    }
    return Date.now() - parsed.getTime() > 1000 * 60 * 60 * 24 * 30
  })
  if (staleManual.length > 0) {
    reminders.push({
      id: 'manual-stale',
      title: '平台校正长期未复核',
      description: `${staleManual.length} 笔校正记录超过 30 天未更新，若外部平台份额变化过，建议重新对账。`,
      tone: 'info',
      metric: `${staleManual.length} 笔`,
      action: '可在对账面板切到“已校正”查看。',
    })
  }

  const totalPrincipal = aggregates.reduce((sum, aggregate) => sum + (parseOptionalNumber(aggregate.total_principal) ?? 0), 0)
  const themeExposure = new Map<string, number>()
  for (const aggregate of aggregates) {
    const principal = parseOptionalNumber(aggregate.total_principal) ?? 0
    const snapshot = exposureSnapshots[aggregate.fund_id]
    const themeBreakdown = snapshot?.themeSnapshot?.breakdown ?? []
    if (themeBreakdown.length > 0) {
      for (const item of themeBreakdown) {
        addExposure(themeExposure, item.theme_name, principal * ((parseOptionalNumber(item.weight_percent) ?? 0) / 100))
      }
    } else {
      addExposure(themeExposure, snapshot?.themeSnapshot?.primary_theme_name || snapshot?.sectorSnapshot?.primary_sector_name, principal)
    }
  }
  const largestTheme = topExposure(themeExposure)
  const hasHeavyEvent = aggregates.some((aggregate) => {
    const analysis = analysesByFundID[aggregate.fund_id]
    return (analysis?.event_impacts ?? []).some((event) => (
      (event.target_scope === 'exposure' || event.target_scope === 'macro' || event.target_scope === 'holding') &&
      event.strength !== 'low'
    ))
  })
  if (largestTheme && totalPrincipal > 0 && (largestTheme[1] / totalPrincipal) * 100 >= 35 && hasHeavyEvent) {
    reminders.push({
      id: 'heavy-theme-event',
      title: '重仓主题存在事件信号',
      description: `${largestTheme[0]} 是当前主要暴露之一，且相关基金已有中高强度事件信号。`,
      tone: 'warning',
      metric: formatPercentMetric((largestTheme[1] / totalPrincipal) * 100),
      action: '建议优先查看该主题下基金的量化详情页。',
    })
  }

  return {
    urgentCount: reminders.filter((reminder) => reminder.tone === 'danger' || reminder.tone === 'warning').length,
    reminders: reminders.slice(0, 6),
  }
}
