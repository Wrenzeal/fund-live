'use client'

import { Activity, AlertTriangle, CheckCircle2, HeartPulse, Info, ShieldAlert } from 'lucide-react'
import type { FundAnalysis, FundHoldingRecord } from '@/hooks/use-fund-data'
import type { HoldingAggregateEntry, HoldingEntry, HoldingEstimateAggregateMetrics } from '@/hooks/use-user-portfolio'
import { buildPortfolioHealthSummary, type InsightTone } from '@/lib/holding-insights'
import type { HoldingExposureSnapshot } from '@/lib/holding-insights'
import type { HoldingMetricScope } from '@/lib/holding-display'
import { cn } from '@/lib/utils'

interface HoldingPortfolioHealthPanelProps {
  holdings: HoldingEntry[]
  aggregates: HoldingAggregateEntry[]
  analysesByFundID: Record<string, FundAnalysis | null>
  aggregateMetrics: Record<string, HoldingEstimateAggregateMetrics>
  metricScope: HoldingMetricScope
  exposureSnapshots?: Record<string, HoldingExposureSnapshot>
  topHoldingsByFundID?: Record<string, FundHoldingRecord[]>
}

function toneClass(tone: InsightTone) {
  switch (tone) {
    case 'danger':
      return 'border-rose-400/28 bg-rose-500/10 text-rose-100'
    case 'warning':
      return 'border-amber-400/28 bg-amber-400/10 text-amber-100'
    case 'good':
      return 'border-emerald-400/24 bg-emerald-400/10 text-emerald-100'
    case 'info':
    default:
      return 'border-cyan-400/24 bg-cyan-400/10 text-cyan-100'
  }
}

function toneIcon(tone: InsightTone) {
  switch (tone) {
    case 'danger':
      return <ShieldAlert className="h-4 w-4" />
    case 'warning':
      return <AlertTriangle className="h-4 w-4" />
    case 'good':
      return <CheckCircle2 className="h-4 w-4" />
    case 'info':
    default:
      return <Info className="h-4 w-4" />
  }
}

export function HoldingPortfolioHealthPanel({
  holdings,
  aggregates,
  analysesByFundID,
  aggregateMetrics,
  metricScope,
  exposureSnapshots = {},
  topHoldingsByFundID = {},
}: HoldingPortfolioHealthPanelProps) {
  const summary = buildPortfolioHealthSummary({
    aggregates,
    holdings,
    analysesByFundID,
    aggregateMetrics,
    metricScope,
    exposureSnapshots,
    topHoldingsByFundID,
  })

  if (holdings.length === 0) {
    return null
  }

  return (
    <section className="mb-6 overflow-hidden rounded-[30px] border border-[var(--card-border)] bg-[var(--card-bg)]/84 p-5 glass">
      <div className="grid gap-5 xl:grid-cols-[0.78fr_1.22fr] xl:items-stretch">
        <div className="rounded-[26px] border border-cyan-400/20 bg-cyan-400/8 p-5">
          <div className="inline-flex items-center gap-2 rounded-full border border-cyan-400/25 bg-cyan-400/10 px-3 py-1 text-[11px] font-medium tracking-[0.2em] text-cyan-200">
            <HeartPulse className="h-3.5 w-3.5" />
            组合体检
          </div>
          <div className="mt-5 flex items-end gap-3">
            <div className="text-5xl font-black text-theme-primary">{summary.score}</div>
            <div className="pb-1 text-sm text-theme-muted">/ 100</div>
          </div>
          <div className="mt-3 text-xl font-black text-theme-primary">{summary.title}</div>
          <p className="mt-2 text-sm leading-6 text-theme-secondary">{summary.description}</p>
          <div className={cn('mt-5 inline-flex items-center gap-2 rounded-2xl border px-3 py-2 text-sm font-medium', toneClass(summary.tone))}>
            <Activity className="h-4 w-4" />
            {metricScope === 'official' ? '官方口径体检' : '盘中预估体检'}
          </div>
        </div>

        <div className="grid gap-3 md:grid-cols-2">
          {summary.signals.map((signal) => (
            <div key={signal.id} className={cn('rounded-[24px] border p-4', toneClass(signal.tone))}>
              <div className="flex items-start justify-between gap-3">
                <div className="flex min-w-0 items-center gap-2">
                  <span className="shrink-0">{toneIcon(signal.tone)}</span>
                  <div className="truncate text-sm font-semibold text-theme-primary">{signal.title}</div>
                </div>
                {signal.metric && <span className="shrink-0 rounded-full border border-white/15 bg-white/8 px-2 py-0.5 text-[11px]">{signal.metric}</span>}
              </div>
              <p className="mt-2 text-xs leading-5 text-theme-secondary">{signal.description}</p>
              {signal.action && <div className="mt-3 text-xs font-medium text-theme-primary">{signal.action}</div>}
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
