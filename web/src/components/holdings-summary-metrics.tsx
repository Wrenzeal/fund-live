'use client'

import type { FundEstimate } from '@/hooks/use-fund-data'
import type { HoldingEntry } from '@/hooks/use-user-portfolio'
import { resolveHoldingMetrics, type HoldingMetricScope } from '@/lib/holding-display'
import { cn } from '@/lib/utils'

export function holdingMoney(value: number | null, signed = false) {
  if (value === null || !Number.isFinite(value)) return '--'
  const formatted = new Intl.NumberFormat('zh-CN', { style: 'currency', currency: 'CNY', minimumFractionDigits: 2, maximumFractionDigits: 2 }).format(value)
  return signed && value > 0 ? `+${formatted}` : formatted
}

export function holdingTone(value: number | null) {
  return value === null || value === 0 ? 'text-theme-primary' : value > 0 ? 'text-up' : 'text-down'
}

export function HoldingsSummaryMetrics({ holdings, estimatesByFundID, metricScope }: {
  holdings: HoldingEntry[]
  estimatesByFundID: Record<string, FundEstimate | null>
  metricScope: HoldingMetricScope
}) {
  const entries = holdings.map(holding => ({ holding, metrics: resolveHoldingMetrics(holding, estimatesByFundID[holding.fund_id], metricScope) }))
  const priced = entries.filter(({ metrics }) => metrics.marketValue !== null && metrics.principal !== null)
  const ready = priced.length > 0
  const partial = priced.length < holdings.length
  const totalValue = ready ? priced.reduce((sum, entry) => sum + entry.metrics.marketValue!, 0) : null
  const principal = ready ? priced.reduce((sum, entry) => sum + entry.metrics.principal!, 0) : null
  const profit = totalValue !== null && principal !== null ? totalValue - principal : null
  const dailyEntries = priced.filter(entry => entry.metrics.todayProfit !== null)
  const dailyProfit = dailyEntries.length > 0 ? dailyEntries.reduce((sum, entry) => sum + entry.metrics.todayProfit!, 0) : null
  const returnPercent = profit !== null && principal !== null && principal > 0 ? `${profit > 0 ? '+' : ''}${(profit / principal * 100).toFixed(2)}%` : '--'
  const dates = [...new Set(priced.map(({ holding }) => metricScope === 'official' ? holding.actual_date : estimatesByFundID[holding.fund_id]?.calculated_at?.slice(0, 10)).filter(Boolean))].sort()
  const dateLabel = dates.length === 1 ? `截至 ${dates[0]}` : dates.length > 1 ? `净值日 ${dates[0]} 至 ${dates.at(-1)}，日期不一致` : '等待净值更新'
  const totalCost = holdings.reduce((sum, holding) => sum + (Number.isFinite(Number(holding.amount)) ? Number(holding.amount) : 0), 0)

  return <section data-testid="holdings-summary" aria-label="持仓金额总览" className="rounded-xl border border-[var(--card-border)] bg-[var(--card-bg)] p-5 sm:p-6">
    <div className="flex flex-wrap items-center justify-between gap-2 text-sm text-theme-secondary">
      <span>{partial && ready ? '已计价持仓市值' : '持仓市值'}{metricScope === 'estimate' && '（预估）'}</span>
      <span>{metricScope === 'official' ? '官方净值' : '盘中预估'} · {dateLabel}</span>
    </div>
    <div className="mt-4 grid gap-5 lg:grid-cols-[1.1fr_1.4fr] lg:items-center">
    <div><div className="break-words text-4xl font-semibold tracking-tight tabular-nums text-theme-primary sm:text-5xl">{holdingMoney(totalValue)}</div>
    <div className="mt-3 text-sm text-theme-secondary">{ready ? `已计价 ${priced.length}/${holdings.length} 笔` : '尚无可计价持仓'}{partial && ` · ${holdings.length - priced.length} 笔待确认或待更新`}{priced.some(entry => entry.holding.real_metrics_stale) && ' · 部分净值待更新'}</div>
    </div>
    <dl className="grid grid-cols-2 gap-4 border-t border-[var(--card-border)] pt-4 lg:border-l lg:border-t-0 lg:pl-6 lg:pt-0">
      <div><dt className="text-sm text-theme-secondary">持有盈亏{partial && ready ? '（已计价部分）' : ''}</dt><dd className={cn('mt-2 text-2xl font-semibold tabular-nums', holdingTone(profit))}>{holdingMoney(profit, true)}</dd><dd className="mt-1 text-sm tabular-nums text-theme-secondary">{returnPercent}</dd></div>
      <div><dt className="text-sm text-theme-secondary">{metricScope === 'estimate' ? '当日变动预估' : '当日变动'}{dailyEntries.length < priced.length && ready ? '（部分）' : ''}</dt><dd className={cn('mt-2 text-2xl font-semibold tabular-nums', holdingTone(dailyProfit))}>{holdingMoney(dailyProfit, true)}</dd><dd className="mt-1 text-sm text-theme-secondary">当前持仓在对应净值日的变动</dd></div>
    </dl>
    </div>
    <div className="mt-5 flex flex-wrap justify-between gap-2 text-sm text-theme-secondary"><span>剩余成本 {holdingMoney(totalCost)}</span><span>{new Set(holdings.map(item => item.fund_id)).size} 只基金 · {holdings.length} 笔记录</span></div>
    <details className="mt-2 text-sm text-theme-secondary"><summary className="min-h-11 cursor-pointer py-3">金额怎么算？</summary><p className="max-w-3xl leading-7">持仓市值按确认份额与所选净值计算；持有盈亏只用同一已计价范围的市值减剩余成本，不包含已卖出盈亏和现金分红。未确认的资金没有混入市值。{partial && ready && `当前已计价成本 ${holdingMoney(principal)}，其余成本不参与盈亏计算。`}当日变动不是按实际成交时点结算的账户损益。</p></details>
  </section>
}
