import type { FundAnalysis } from '@/hooks/use-fund-data'
import type { FundEstimate } from '@/hooks/use-fund-data'
import type { HoldingAggregateEntry, HoldingEntry, HoldingEstimateAggregateMetrics } from '@/hooks/use-user-portfolio'
import { dominantAnalysisRecommendation, parseAnalysisNumber } from '@/lib/fund-analysis-display'

export const BEIJING_OFFSET = '+08:00'

export type TradeTiming = 'before_close' | 'after_close'
export type HoldingMetricScope = 'official' | 'estimate'
export type HoldingSortMode = 'default' | 'analysis_recommendation' | 'analysis_risk' | 'principal_desc' | 'profit_asc' | 'profit_desc' | 'change_asc' | 'change_desc' | 'count_desc' | 'recent_desc'
export type HoldingFilterMode = 'all' | 'profit' | 'loss' | 'ready' | 'partial' | 'single' | 'multiple'

export function buildTradeAtValue(date: string, tradeTiming: TradeTiming) {
  if (!date) {
    return ''
  }

  const marker = tradeTiming === 'before_close' ? '14:59:00' : '15:01:00'
  return `${date}T${marker}${BEIJING_OFFSET}`
}

export function formatTradeDateLabel(date: string) {
  if (!date) {
    return '选择交易日期'
  }

  const parsed = new Date(`${date}T12:00:00${BEIJING_OFFSET}`)
  if (Number.isNaN(parsed.getTime())) {
    return date
  }

  return new Intl.DateTimeFormat('zh-CN', {
    timeZone: 'Asia/Shanghai',
    month: 'long',
    day: 'numeric',
    weekday: 'short',
  }).format(parsed)
}

export function formatTradeTimingLabel(tradeTiming: TradeTiming) {
  return tradeTiming === 'after_close' ? '15:00 后' : '15:00 前'
}

export function resolveTradeTimingFromServerClock(currentTime: Date): TradeTiming {
  const beijingTime = currentTime.toLocaleTimeString('en-GB', {
    timeZone: 'Asia/Shanghai',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  })

  return beijingTime >= '15:00' ? 'after_close' : 'before_close'
}

export function formatSummaryMoney(value?: string) {
  if (!value) {
    return '--'
  }

  const parsed = Number.parseFloat(value)
  if (Number.isNaN(parsed)) {
    return '--'
  }

  return new Intl.NumberFormat('zh-CN', {
    style: 'currency',
    currency: 'CNY',
    maximumFractionDigits: 2,
  }).format(parsed)
}

export function formatSummaryPercent(value?: string) {
  if (!value) {
    return '--'
  }

  const parsed = Number.parseFloat(value)
  if (Number.isNaN(parsed)) {
    return '--'
  }

  return `${parsed >= 0 ? '+' : ''}${parsed.toFixed(2)}%`
}

export function parseOptionalNumber(value?: string) {
  if (!value) {
    return null
  }

  const parsed = Number.parseFloat(value)
  return Number.isNaN(parsed) ? null : parsed
}

export function compareOptionalNumbers(left: number | null, right: number | null, direction: 'asc' | 'desc') {
  const leftMissing = left === null
  const rightMissing = right === null

  if (leftMissing && rightMissing) {
    return 0
  }
  if (leftMissing) {
    return 1
  }
  if (rightMissing) {
    return -1
  }

  return direction === 'asc' ? left - right : right - left
}

export function analysisRecommendationWeight(analysis?: Pick<FundAnalysis, 'increase_percent' | 'hold_percent' | 'decrease_percent'> | null) {
  if (!analysis) {
    return -1
  }

  const increase = parseAnalysisNumber(analysis.increase_percent)
  const hold = parseAnalysisNumber(analysis.hold_percent)
  const decrease = parseAnalysisNumber(analysis.decrease_percent)
  const dominant = dominantAnalysisRecommendation(analysis)

  if (dominant === 'increase') {
    return 3_000 + increase
  }
  if (dominant === 'hold') {
    return 2_000 + hold
  }
  return 1_000 + decrease
}

export function analysisRiskWeight(analysis?: Pick<FundAnalysis, 'risk_level' | 'total_score'> | null) {
  if (!analysis) {
    return -1
  }

  const score = parseAnalysisNumber(analysis.total_score)
  switch (analysis.risk_level) {
    case 'high':
      return 3_000 + (100 - score)
    case 'medium':
      return 2_000 + (100 - score)
    case 'low':
      return 1_000 + (100 - score)
    default:
      return score
  }
}

export function aggregateProfitValue(
  aggregate: Pick<HoldingAggregateEntry, 'official_today_profit'>,
  estimateMetrics: Pick<HoldingEstimateAggregateMetrics, 'preview_ready' | 'fallback_ready' | 'preview_today_profit' | 'fallback_today_profit'> | undefined,
  metricScope: HoldingMetricScope
) {
  if (metricScope === 'official') {
    return parseOptionalNumber(aggregate.official_today_profit)
  }

  if (estimateMetrics?.preview_ready) {
    return parseOptionalNumber(estimateMetrics.preview_today_profit)
  }
  if (estimateMetrics?.fallback_ready) {
    return parseOptionalNumber(estimateMetrics.fallback_today_profit)
  }
  return null
}

export function aggregateChangeValue(
  aggregate: Pick<HoldingAggregateEntry, 'official_today_change_percent'>,
  estimateMetrics: Pick<HoldingEstimateAggregateMetrics, 'preview_ready' | 'preview_today_change_percent' | 'estimate'> | undefined,
  metricScope: HoldingMetricScope
) {
  if (metricScope === 'official') {
    return parseOptionalNumber(aggregate.official_today_change_percent)
  }

  if (estimateMetrics?.preview_ready) {
    return parseOptionalNumber(estimateMetrics.preview_today_change_percent)
  }
  return parseOptionalNumber(estimateMetrics?.estimate?.change_percent)
}

export function detailProfitValue(
  holding: Pick<HoldingEntry, 'today_profit' | 'amount'>,
  estimate: Pick<FundEstimate, 'change_percent'> | null | undefined,
  metricScope: HoldingMetricScope
) {
  if (metricScope === 'official') {
    return parseOptionalNumber(holding.today_profit)
  }

  const changePercent = parseOptionalNumber(estimate?.change_percent)
  const amount = parseOptionalNumber(holding.amount)
  if (changePercent === null || amount === null) {
    return null
  }

  return amount * changePercent / 100
}

export function detailChangeValue(
  holding: Pick<HoldingEntry, 'today_change_percent'>,
  estimate: Pick<FundEstimate, 'change_percent'> | null | undefined,
  metricScope: HoldingMetricScope
) {
  if (metricScope === 'official') {
    return parseOptionalNumber(holding.today_change_percent)
  }
  return parseOptionalNumber(estimate?.change_percent)
}

export function timestampValue(value?: string) {
  if (!value) {
    return null
  }

  const parsed = new Date(value).getTime()
  return Number.isNaN(parsed) ? null : parsed
}

export function aggregateLatestUpdatedAt(holdings: Array<Pick<HoldingEntry, 'updated_at' | 'created_at'>>) {
  return holdings.reduce<number | null>((current, holding) => {
    const next = timestampValue(holding.updated_at || holding.created_at)
    if (next === null) {
      return current
    }
    return current === null ? next : Math.max(current, next)
  }, null)
}

export function isHoldingIncomplete(holding: Pick<HoldingEntry, 'shares' | 'confirmed_nav_date' | 'real_metrics_ready'>) {
  return !holding.shares || !holding.confirmed_nav_date || holding.real_metrics_ready === false
}

export function isAggregateIncomplete(aggregate: Pick<HoldingAggregateEntry, 'incomplete_holdings_count' | 'metrics_scope' | 'real_metrics_ready'>) {
  return (aggregate.incomplete_holdings_count ?? 0) > 0 ||
    aggregate.metrics_scope === 'partial' ||
    aggregate.metrics_scope === 'none' ||
    aggregate.real_metrics_ready === false
}

export function incompleteAggregateChildren<T extends Pick<HoldingEntry, 'shares' | 'confirmed_nav_date' | 'real_metrics_ready'>>(holdings: T[], showIncompleteOnly: boolean): T[] {
  return showIncompleteOnly ? holdings.filter(isHoldingIncomplete) : holdings
}

export function matchesAggregateFilter(
  aggregate: Pick<HoldingAggregateEntry, 'real_metrics_ready' | 'metrics_scope' | 'holding_count' | 'official_today_profit' | 'incomplete_holdings_count'>,
  estimateMetrics: Pick<HoldingEstimateAggregateMetrics, 'preview_ready' | 'fallback_ready' | 'preview_today_profit' | 'fallback_today_profit'> | undefined,
  metricScope: HoldingMetricScope,
  filterMode: HoldingFilterMode
) {
  if (filterMode === 'all') {
    return true
  }

  if (filterMode === 'ready') {
    return metricScope === 'official'
      ? aggregate.real_metrics_ready === true || aggregate.metrics_scope === 'full'
      : estimateMetrics?.preview_ready === true
  }
  if (filterMode === 'partial') {
    return metricScope === 'official'
      ? isAggregateIncomplete(aggregate)
      : estimateMetrics?.preview_ready !== true
  }
  if (filterMode === 'single') {
    return (aggregate.holding_count ?? 0) <= 1
  }
  if (filterMode === 'multiple') {
    return (aggregate.holding_count ?? 0) > 1
  }

  const profit = aggregateProfitValue(aggregate, estimateMetrics, metricScope)
  if (profit === null) {
    return false
  }
  return filterMode === 'profit' ? profit > 0 : profit < 0
}

export function matchesHoldingFilter(
  holding: Pick<HoldingEntry, 'real_metrics_ready' | 'today_profit' | 'amount' | 'shares' | 'confirmed_nav_date'>,
  estimate: Pick<FundEstimate, 'change_percent'> | null | undefined,
  metricScope: HoldingMetricScope,
  filterMode: HoldingFilterMode,
  sameFundHoldingCount = 1
) {
  if (filterMode === 'all') {
    return true
  }

  if (filterMode === 'ready') {
    return metricScope === 'official'
      ? holding.real_metrics_ready === true
      : detailProfitValue(holding, estimate, metricScope) !== null
  }
  if (filterMode === 'partial') {
    return metricScope === 'official'
      ? isHoldingIncomplete(holding)
      : detailProfitValue(holding, estimate, metricScope) === null
  }
  if (filterMode === 'single') {
    return sameFundHoldingCount <= 1
  }
  if (filterMode === 'multiple') {
    return sameFundHoldingCount > 1
  }

  const profit = detailProfitValue(holding, estimate, metricScope)
  if (profit === null) {
    return false
  }
  return filterMode === 'profit' ? profit > 0 : profit < 0
}
