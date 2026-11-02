'use client'

import { useState, type ReactNode } from 'react'
import Link from 'next/link'
import { ChevronDown } from 'lucide-react'
import { FundHistoryTrend } from '@/components/fund-history-trend'
import type { FundHistorySeries } from '@/hooks/use-fund-data'
import type { HoldingAggregateEntry, HoldingEstimateAggregateMetrics } from '@/hooks/use-user-portfolio'
import { resolveAggregateMetrics, type HoldingMetricScope } from '@/lib/holding-display'
import { holdingMoney, holdingTone } from '@/components/holdings-summary-metrics'
import { cn } from '@/lib/utils'

export function HoldingAggregateRow({ aggregate, metricScope, estimateMetrics, history, isHistoryLoading = false, children }: {
  aggregate: HoldingAggregateEntry; metricScope: HoldingMetricScope; estimateMetrics?: HoldingEstimateAggregateMetrics
  history?: FundHistorySeries; isHistoryLoading?: boolean; children?: ReactNode
}) {
  const [isExpanded, setIsExpanded] = useState(false)
  const metrics = resolveAggregateMetrics(aggregate, estimateMetrics, metricScope)
  const name = aggregate.fund?.name || aggregate.fund_id
  return <article data-testid="holding-aggregate-row" data-fund-id={aggregate.fund_id} className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]">
    <div className="grid gap-5 p-5 lg:grid-cols-[minmax(0,1.4fr)_1fr_1fr_1fr_auto] lg:items-center">
      <div className="min-w-0"><h3 className="break-words text-base font-semibold text-theme-primary">{name}</h3><p className="mt-1 text-sm text-theme-secondary">{aggregate.fund_id} · {aggregate.holding_count} 笔</p>{metrics.partial && <p className="mt-2 text-sm text-[var(--accent-secondary)]">仅显示已计价部分</p>}</div>
      <dl className="grid grid-cols-3 gap-3 lg:col-span-3 lg:contents">
        <div><dt className="text-xs text-theme-secondary">{metricScope === 'official' ? '持仓市值' : '预估市值'}</dt><dd className="mt-2 break-words text-lg font-semibold tabular-nums text-theme-primary">{holdingMoney(metrics.marketValue)}</dd></div>
        <div><dt className="text-xs text-theme-secondary">持有盈亏</dt><dd className={cn('mt-2 break-words text-lg font-semibold tabular-nums', holdingTone(metrics.holdingProfit))}>{holdingMoney(metrics.holdingProfit, true)}</dd></div>
        <div><dt className="text-xs text-theme-secondary">当日变动{metricScope === 'estimate' ? '预估' : ''}</dt><dd className={cn('mt-2 break-words text-lg font-semibold tabular-nums', holdingTone(metrics.todayProfit))}>{holdingMoney(metrics.todayProfit, true)}</dd></div>
      </dl>
      <button type="button" aria-expanded={isExpanded} aria-label={`${isExpanded ? '收起' : '查看'} ${name} 明细`} onClick={() => setIsExpanded(value => !value)} className="inline-flex min-h-11 items-center justify-center gap-2 rounded-lg px-3 text-sm text-theme-secondary hover:bg-[var(--input-bg)]">明细<ChevronDown className={cn('h-4 w-4 transition-transform', isExpanded && 'rotate-180')} /></button>
    </div>
    {isExpanded && <div className="space-y-4 border-t border-[var(--card-border)] p-4 sm:p-5">
      <div className="flex flex-wrap items-center justify-between gap-3 text-sm text-theme-secondary"><span>剩余成本 {holdingMoney(Number(aggregate.total_principal))}{metrics.partial ? ` · 已计价成本 ${holdingMoney(metrics.principal)}` : ''}</span><Link href={`/observations/${aggregate.fund_id}`} className="text-[var(--accent-primary)] underline underline-offset-4">公告与资料</Link></div>
      <details className="text-sm text-theme-secondary"><summary className="min-h-11 cursor-pointer py-3">近 {history?.days || 15} 日官方净值</summary><FundHistoryTrend points={history?.points || []} days={history?.days || 15} compact isLoading={isHistoryLoading} /></details>
      <h4 className="text-sm font-semibold text-theme-primary">分笔记录与操作</h4>{children}
    </div>}
  </article>
}
