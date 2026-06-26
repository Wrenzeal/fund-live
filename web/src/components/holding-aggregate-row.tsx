'use client'

import { useState, type ReactNode } from 'react'
import { ChevronDown, ChevronUp, Layers3 } from 'lucide-react'
import { FundAnalysisBadge } from '@/components/fund-analysis-badge'
import { FundHistoryTrend } from '@/components/fund-history-trend'
import { FundAnalysisEventHint } from '@/components/fund-analysis-event-hint'
import type { FundAnalysis, FundHistorySeries } from '@/hooks/use-fund-data'
import { cn } from '@/lib/utils'
import type {
  HoldingAggregateEntry,
  HoldingEstimateAggregateMetrics,
} from '@/hooks/use-user-portfolio'

interface HoldingAggregateRowProps {
  aggregate: HoldingAggregateEntry
  metricScope: 'official' | 'estimate'
  estimateMetrics?: HoldingEstimateAggregateMetrics
  analysis?: FundAnalysis | null
  history?: FundHistorySeries
  isHistoryLoading?: boolean
  children?: ReactNode
}

function formatMoney(value?: string) {
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

function formatPercent(value?: string) {
  if (!value) {
    return '--'
  }

  const parsed = Number.parseFloat(value)
  if (Number.isNaN(parsed)) {
    return '--'
  }

  return `${parsed >= 0 ? '+' : ''}${parsed.toFixed(2)}%`
}

export function HoldingAggregateRow({ aggregate, metricScope, estimateMetrics, analysis, history, isHistoryLoading = false, children }: HoldingAggregateRowProps) {
  const [isExpanded, setIsExpanded] = useState(false)
  const fundName = aggregate.fund?.name || aggregate.fund_id
  const hasChildren = Boolean(children)
  const isOfficialScope = metricScope === 'official'
  const shouldUseOfficialValues = isOfficialScope && aggregate.real_metrics_ready

  const currentMarketValueText = shouldUseOfficialValues
    ? (aggregate.real_metrics_ready_count > 0 ? formatMoney(aggregate.official_current_market_value) : '--')
    : (estimateMetrics?.preview_ready ? formatMoney(estimateMetrics.preview_current_market_value) : '--')

  const todayProfitText = shouldUseOfficialValues
    ? (aggregate.real_metrics_ready_count > 0 ? formatMoney(aggregate.official_today_profit) : '--')
    : estimateMetrics?.preview_ready
      ? formatMoney(estimateMetrics.preview_today_profit)
      : estimateMetrics?.fallback_ready
        ? formatMoney(estimateMetrics.fallback_today_profit)
        : '--'

  const todayChangePercentText = shouldUseOfficialValues
    ? (aggregate.real_metrics_ready_count > 0 ? formatPercent(aggregate.official_today_change_percent) : '--')
    : estimateMetrics?.estimate?.change_percent
      ? formatPercent(estimateMetrics.preview_today_change_percent || estimateMetrics.estimate.change_percent)
      : '--'

  const todayProfitTone = (() => {
    const reference = shouldUseOfficialValues
      ? aggregate.official_today_profit
      : estimateMetrics?.preview_ready
        ? estimateMetrics.preview_today_profit
        : estimateMetrics?.fallback_today_profit
    const parsed = Number.parseFloat(reference || '')
    if (Number.isNaN(parsed)) {
      return 'text-theme-primary'
    }
    return parsed >= 0 ? 'text-up' : 'text-down'
  })()

  const todayChangeTone = (() => {
    const reference = shouldUseOfficialValues
      ? aggregate.official_today_change_percent
      : estimateMetrics?.preview_today_change_percent || estimateMetrics?.estimate?.change_percent
    const parsed = Number.parseFloat(reference || '')
    if (Number.isNaN(parsed)) {
      return 'text-theme-primary'
    }
    return parsed >= 0 ? 'text-up' : 'text-down'
  })()

  const valueNote = shouldUseOfficialValues
    ? aggregate.real_metrics_ready_count > 0
      ? `官方口径：已就绪 ${aggregate.real_metrics_ready_count}/${aggregate.holding_count} 笔`
      : aggregate.message || '待官方净值齐备'
    : estimateMetrics?.preview_ready
      ? `实时盈亏预估：已按 ${aggregate.confirmed_shares || '--'} 份估算，夜间真实涨跌会自动覆盖`
      : estimateMetrics?.fallback_ready
        ? `实时盈亏预估：仅能按本金口径提示 ${formatMoney(estimateMetrics.fallback_today_profit)}`
        : '待确认份额补齐后展示盘中预估'

  return (
    <div className="overflow-hidden rounded-[28px] border border-[var(--card-border)] glass">
      <div className="grid gap-4 p-5 lg:grid-cols-[minmax(0,1.25fr)_minmax(10rem,0.72fr)_0.8fr_0.85fr_0.85fr_auto] lg:items-center">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <div className="truncate text-base font-semibold text-theme-primary">{fundName}</div>
            <span className="rounded-full border border-cyan-400/20 bg-cyan-400/10 px-2 py-0.5 text-[11px] text-cyan-100">
              {aggregate.holding_count} 笔
            </span>
            <span className="rounded-full border border-[var(--input-border)] bg-[var(--input-bg)] px-2 py-0.5 text-[11px] text-theme-secondary">
              本金 {formatMoney(aggregate.total_principal)}
            </span>
          </div>
          <div className="mt-1 text-xs text-theme-muted">{aggregate.fund_id}</div>
          <div className="mt-2">
            <FundAnalysisBadge analysis={analysis} showScore />
          </div>
          <div className="mt-2">
            <FundAnalysisEventHint analysis={analysis} />
          </div>
          <div className="mt-2 text-xs text-theme-secondary">{valueNote}</div>
        </div>

        <div className="min-w-0">
          <div className="mb-2 text-xs text-theme-muted">近 {history?.days || 15} 日净值</div>
          <FundHistoryTrend
            points={history?.points || []}
            days={history?.days || 15}
            compact
            isLoading={isHistoryLoading}
            className="bg-[var(--input-bg)]/25"
          />
        </div>

        <div>
          <div className="text-xs text-theme-muted">{shouldUseOfficialValues ? '官方市值' : '预估市值'}</div>
          <div className="mt-1 text-lg font-semibold text-theme-primary">{currentMarketValueText}</div>
          <div className="mt-1 text-xs text-theme-muted">
            已覆盖本金 {formatMoney(shouldUseOfficialValues ? aggregate.ready_principal : aggregate.confirmed_principal)} / {formatMoney(aggregate.total_principal)}
          </div>
        </div>

        <div>
          <div className="text-xs text-theme-muted">{shouldUseOfficialValues ? '官方盈亏' : '实时盈亏预估'}</div>
          <div className={cn('mt-1 text-lg font-semibold', todayProfitTone)}>{todayProfitText}</div>
          <div className="mt-1 text-xs text-theme-muted">
            {shouldUseOfficialValues ? '按已就绪分笔汇总' : estimateMetrics?.preview_ready ? '根据基金预估涨跌幅估算' : '按当前可用数据提示'}
          </div>
        </div>

        <div>
          <div className="text-xs text-theme-muted">{shouldUseOfficialValues ? '官方涨跌幅' : '实时涨跌预估'}</div>
          <div className={cn('mt-1 text-lg font-semibold', todayChangeTone)}>{todayChangePercentText}</div>
          <div className="mt-1 text-xs text-theme-muted">
            {shouldUseOfficialValues
              ? (aggregate.metrics_scope === 'partial' ? `待补齐 ${aggregate.incomplete_holdings_count} 笔` : '夜间真实涨跌幅已同步')
              : estimateMetrics?.estimate?.change_percent
                ? '夜间真实涨跌幅同步后会自动覆盖'
                : '待确认份额补齐后展示'}
          </div>
        </div>

        {hasChildren ? (
          <button
            type="button"
            onClick={() => setIsExpanded((value) => !value)}
            className="inline-flex items-center justify-center gap-2 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2 text-sm text-theme-secondary transition-colors hover:border-cyan-400/35 hover:text-theme-primary"
          >
            <Layers3 className="h-4 w-4" />
            <span>{isExpanded ? '收起分笔' : '查看分笔'}</span>
            {isExpanded ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
          </button>
        ) : (
          <div />
        )}
      </div>

      {hasChildren && isExpanded && (
        <div className="border-t border-[var(--card-border)] bg-[var(--card-bg)]/30 p-4">
          <div className="space-y-4">{children}</div>
        </div>
      )}
    </div>
  )
}
