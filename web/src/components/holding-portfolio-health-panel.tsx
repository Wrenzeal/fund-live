'use client'

import {
  Activity,
  AlertTriangle,
  CheckCircle2,
  HeartPulse,
  Info,
  ShieldAlert,
} from 'lucide-react'
import type { FundHoldingRecord } from '@/hooks/use-fund-data'
import type {
  HoldingAggregateEntry,
  HoldingEntry,
  HoldingEstimateAggregateMetrics,
} from '@/hooks/use-user-portfolio'
import {
  buildPortfolioHealthSummary,
  type InsightTone,
} from '@/lib/holding-insights'
import type { HoldingExposureSnapshot } from '@/lib/holding-insights'
import type { HoldingMetricScope } from '@/lib/holding-display'
import { cn } from '@/lib/utils'

interface HoldingPortfolioHealthPanelProps {
  compact?: boolean
  holdings: HoldingEntry[]
  aggregates: HoldingAggregateEntry[]
  aggregateMetrics: Record<string, HoldingEstimateAggregateMetrics>
  metricScope: HoldingMetricScope
  exposureSnapshots?: Record<string, HoldingExposureSnapshot>
  topHoldingsByFundID?: Record<string, FundHoldingRecord[]>
}

function toneClass(tone: InsightTone) {
  switch (tone) {
    case 'danger':
      return 'border-danger/28 bg-danger/10 text-danger'
    case 'warning':
      return 'border-warning/28 bg-warning/10 text-warning'
    case 'good':
      return 'border-success/24 bg-success/10 text-success'
    case 'info':
    default:
      return 'border-accent/24 bg-accent/10 text-accent'
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
  compact = false,
  holdings,
  aggregates,
  aggregateMetrics,
  metricScope,
  exposureSnapshots = {},
  topHoldingsByFundID = {},
}: HoldingPortfolioHealthPanelProps) {
  const summary = buildPortfolioHealthSummary({
    aggregates,
    holdings,
    aggregateMetrics,
    metricScope,
    exposureSnapshots,
    topHoldingsByFundID,
  })

  if (holdings.length === 0) {
    return null
  }

  return (
    <section className="mb-6 overflow-hidden rounded-xl border border-[var(--card-border)] bg-[var(--card-bg)]/84 p-5 glass">
      <div
        className={cn(
          'grid gap-5 xl:items-stretch',
          compact ? 'xl:grid-cols-1' : 'xl:grid-cols-[0.78fr_1.22fr]',
        )}
      >
        <div className="rounded-xl border border-accent/20 bg-accent/8 p-5">
          <div className="inline-flex items-center gap-2 rounded-full border border-accent/25 bg-accent/10 px-3 py-1 text-[11px] font-medium tracking-[0.2em] text-accent">
            <HeartPulse className="h-3.5 w-3.5" />
            组合体检
          </div>
          <div className="mt-5 flex items-end gap-3">
            <div className="text-5xl font-semibold text-theme-primary">
              {summary.reviewCount}
            </div>
            <div className="pb-1 text-sm text-theme-muted">项待复核</div>
          </div>
          <div className="mt-3 text-xl font-semibold text-theme-primary">
            {summary.title}
          </div>
          {!compact && (
            <p className="mt-2 text-sm leading-6 text-theme-secondary">
              {summary.description}
            </p>
          )}
          <div
            className={cn(
              'mt-5 inline-flex items-center gap-2 rounded-xl border px-3 py-2 text-sm font-medium',
              toneClass(summary.tone),
            )}
          >
            <Activity className="h-4 w-4" />
            {metricScope === 'official' ? '官方口径体检' : '盘中预估体检'}
          </div>
        </div>

        <div className="grid gap-3 md:grid-cols-2">
          {summary.signals.map((signal) => (
            <div
              key={signal.id}
              className={cn(
                'rounded-xl border p-4',
                toneClass(signal.tone),
              )}
            >
              <div className="flex items-start justify-between gap-3">
                <div className="flex min-w-0 items-center gap-2">
                  <span className="shrink-0">{toneIcon(signal.tone)}</span>
                  <div className="truncate text-sm font-semibold text-theme-primary">
                    {signal.title}
                  </div>
                </div>
                {signal.metric && (
                  <span className="shrink-0 rounded-full border border-[var(--card-border)] bg-surface-subtle px-2 py-0.5 text-[11px]">
                    {signal.metric}
                  </span>
                )}
              </div>
              <p className="mt-2 text-xs leading-5 text-theme-secondary">
                {signal.description}
              </p>
              {signal.action && (
                <div className="mt-3 text-xs font-medium text-theme-primary">
                  {signal.action}
                </div>
              )}
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
