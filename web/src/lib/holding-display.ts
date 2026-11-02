import type { FundEstimate } from '@/hooks/use-fund-data'
import type { HoldingAggregateEntry, HoldingEntry, HoldingEstimateAggregateMetrics } from '@/hooks/use-user-portfolio'

export const BEIJING_OFFSET = '+08:00'

export type TradeTiming = 'before_close' | 'after_close'
export type HoldingMetricScope = 'official' | 'estimate'
export type HoldingSortMode = 'default' | 'principal_desc' | 'profit_asc' | 'profit_desc' | 'change_asc' | 'change_desc' | 'count_desc' | 'recent_desc'
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
  return Number.isFinite(parsed) ? parsed : null
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

export interface HoldingDisplayMetrics {
  marketValue: number | null
  principal: number | null
  holdingProfit: number | null
  todayProfit: number | null
  change: number | null
  ready: boolean
  partial: boolean
}

type HoldingMetricInput = Pick<HoldingEntry, 'amount'> & Partial<HoldingEntry>
type AggregateMetricInput = Partial<HoldingAggregateEntry>
type EstimateInput = Partial<FundEstimate> | null | undefined

export function resolveHoldingMetrics(holding: HoldingMetricInput, estimate: EstimateInput, scope: HoldingMetricScope): HoldingDisplayMetrics {
  const principal = parseOptionalNumber(holding.amount)
  let marketValue: number | null = null
  let todayProfit: number | null = null
  let change: number | null = null
  if (scope === 'official' && holding.real_metrics_ready) {
    marketValue = parseOptionalNumber(holding.current_market_value)
    todayProfit = parseOptionalNumber(holding.today_profit)
    change = parseOptionalNumber(holding.today_change_percent)
  } else if (scope === 'estimate') {
    const shares = parseOptionalNumber(holding.shares)
    const nav = parseOptionalNumber(estimate?.estimate_nav)
    const previous = parseOptionalNumber(estimate?.prev_nav)
    if (holding.confirmed_nav_date && shares !== null && shares > 0 && nav !== null && nav > 0) {
      marketValue = shares * nav
      if (previous !== null && previous > 0) {
        todayProfit = Number((shares * (nav - previous)).toFixed(8))
        change = Number(((nav / previous - 1) * 100).toFixed(8))
      }
    }
  }
  return { marketValue, principal, todayProfit, change,
    holdingProfit: marketValue !== null && principal !== null ? marketValue - principal : null,
    ready: marketValue !== null && principal !== null, partial: false }
}

export function resolveAggregateMetrics(aggregate: AggregateMetricInput, estimate: Partial<HoldingEstimateAggregateMetrics> | undefined, scope: HoldingMetricScope): HoldingDisplayMetrics {
  const available = scope === 'official'
    ? (aggregate.real_metrics_ready_count ?? 0) > 0 || aggregate.real_metrics_ready === true || aggregate.metrics_scope === 'full'
    : estimate?.preview_ready === true
  const partial = scope === 'official'
    ? available && (aggregate.metrics_scope === 'partial' || (aggregate.real_metrics_ready_count ?? aggregate.holding_count ?? 0) < (aggregate.holding_count ?? 0))
    : available && (aggregate.confirmed_holding_count ?? 0) < (aggregate.holding_count ?? 0)
  const marketValue = !available ? null : parseOptionalNumber(scope === 'official' ? aggregate.official_current_market_value : estimate?.preview_current_market_value)
  const principal = scope === 'official'
    ? parseOptionalNumber(partial ? aggregate.ready_principal : aggregate.ready_principal ?? aggregate.total_principal)
    : parseOptionalNumber(estimate?.confirmed_principal ?? aggregate.confirmed_principal)
  const todayProfit = !available ? null : parseOptionalNumber(scope === 'official' ? aggregate.official_today_profit : estimate?.preview_today_profit)
  const change = !available ? null : parseOptionalNumber(scope === 'official' ? aggregate.official_today_change_percent : estimate?.preview_today_change_percent)
  return { marketValue, principal, todayProfit, change, partial,
    holdingProfit: marketValue !== null && principal !== null ? marketValue - principal : null,
    ready: marketValue !== null && principal !== null }
}

export function aggregateProfitValue(aggregate: AggregateMetricInput, estimate: Partial<HoldingEstimateAggregateMetrics> | undefined, scope: HoldingMetricScope) {
  return resolveAggregateMetrics(aggregate, estimate, scope).todayProfit
}
export function aggregateChangeValue(aggregate: AggregateMetricInput, estimate: Partial<HoldingEstimateAggregateMetrics> | undefined, scope: HoldingMetricScope) {
  return resolveAggregateMetrics(aggregate, estimate, scope).change
}
export function detailProfitValue(holding: HoldingMetricInput, estimate: EstimateInput, scope: HoldingMetricScope) {
  return resolveHoldingMetrics(holding, estimate, scope).todayProfit
}
export function detailChangeValue(holding: HoldingMetricInput, estimate: EstimateInput, scope: HoldingMetricScope) {
  return resolveHoldingMetrics(holding, estimate, scope).change
}

export function timestampValue(value?: string) {
  if (!value) {
    return null
  }

  const parsed = new Date(value).getTime()
  return Number.isFinite(parsed) ? parsed : null
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
  aggregate: AggregateMetricInput,
  estimate: Partial<HoldingEstimateAggregateMetrics> | undefined,
  scope: HoldingMetricScope,
  filter: HoldingFilterMode,
) {
  const metrics = resolveAggregateMetrics(aggregate, estimate, scope)
  if (filter === 'all') return true
  if (filter === 'ready') return metrics.ready && !metrics.partial
  if (filter === 'partial') return !metrics.ready || metrics.partial
  if (filter === 'single') return (aggregate.holding_count ?? 0) <= 1
  if (filter === 'multiple') return (aggregate.holding_count ?? 0) > 1
  return metrics.todayProfit !== null && (filter === 'profit' ? metrics.todayProfit > 0 : metrics.todayProfit < 0)
}

export function matchesHoldingFilter(
  holding: HoldingMetricInput,
  estimate: EstimateInput,
  scope: HoldingMetricScope,
  filter: HoldingFilterMode,
  sameFundHoldingCount = 1,
) {
  const metrics = resolveHoldingMetrics(holding, estimate, scope)
  if (filter === 'all') return true
  if (filter === 'ready') return metrics.ready
  if (filter === 'partial') return !metrics.ready
  if (filter === 'single') return sameFundHoldingCount <= 1
  if (filter === 'multiple') return sameFundHoldingCount > 1
  return metrics.todayProfit !== null && (filter === 'profit' ? metrics.todayProfit > 0 : metrics.todayProfit < 0)
}
