'use client'

import type { ReactNode } from 'react'
import Link from 'next/link'
import { ArrowRight, Gauge, ShieldAlert, Sparkles, TrendingUp } from 'lucide-react'
import { cn } from '@/lib/utils'
import {
  confidenceLevelLabel,
  dominantAnalysisRecommendation,
  eventImpactTone,
  formatAnalysisPercent,
  parseAnalysisNumber,
  riskLevelLabel,
  type FundAnalysisRecommendationCode,
} from '@/lib/fund-analysis-display'
import type { FundAnalysis } from '@/hooks/use-fund-data'
import { AnimatedScoreGauge } from '@/components/animated-score-gauge'
import { LoadingSpinner } from '@/components/loading-indicator'

interface FundAnalysisCardProps {
  analysis?: FundAnalysis
  fundId?: string
  isLoading?: boolean
}

function confidenceLabel(level?: string) {
  return confidenceLevelLabel(level, {
    high: '可信度较高',
    medium: '可信度一般',
    low: '可信度有限',
    unknown: '可信度未知',
  })
}

function recommendationTone(code: FundAnalysisRecommendationCode) {
  switch (code) {
    case 'increase':
      return {
        label: '结构偏积极',
        color: 'text-rose-100',
        soft: 'border-rose-500/20 bg-rose-500/10',
        bar: 'from-rose-500 via-fuchsia-500 to-pink-500',
        dot: 'bg-rose-300',
      }
    case 'decrease':
      return {
        label: '风险偏高',
        color: 'text-emerald-100',
        soft: 'border-emerald-500/20 bg-emerald-500/10',
        bar: 'from-emerald-500 via-teal-500 to-cyan-500',
        dot: 'bg-emerald-300',
      }
    default:
      return {
        label: '适合观察',
        color: 'text-slate-100',
        soft: 'border-slate-500/20 bg-slate-500/10',
        bar: 'from-slate-500 via-slate-400 to-cyan-400',
        dot: 'bg-slate-300',
      }
  }
}

function compactText(value: string, maxLength = 66) {
  if (value.length <= maxLength) {
    return value
  }
  return `${value.slice(0, maxLength)}…`
}

function eventSourceConfidenceLabel(level?: string) {
  switch (level) {
    case 'high':
      return '高可信'
    case 'medium':
      return '中可信'
    case 'low':
      return '低可信'
    default:
      return ''
  }
}

export function FundAnalysisCard({ analysis, fundId, isLoading = false }: FundAnalysisCardProps) {
  if (!analysis && !isLoading) {
    return null
  }

  if (!analysis) {
    return (
      <section className="glass rounded-3xl p-5 sm:p-6">
        <div className="flex min-h-[12rem] items-center justify-center rounded-3xl border border-[var(--card-border)] bg-[var(--card-bg)]/40 px-5 py-8">
          <LoadingSpinner size="md" text="量化观察加载中..." />
        </div>
      </section>
    )
  }

  const increase = parseAnalysisNumber(analysis.increase_percent)
  const hold = parseAnalysisNumber(analysis.hold_percent)
  const decrease = parseAnalysisNumber(analysis.decrease_percent)
  const recommendationItems = [
    { code: 'increase' as const, value: increase },
    { code: 'hold' as const, value: hold },
    { code: 'decrease' as const, value: decrease },
  ]
  const dominantTone = recommendationTone(dominantAnalysisRecommendation(analysis))
  const primaryPoints = [
    ...(analysis.primary_evidence || []).map((item) => item.summary || item.title),
    ...(analysis.reasons || []),
  ].filter(Boolean).slice(0, 2)
  const riskPoints = [
    ...(analysis.counter_evidence || []).map((item) => item.summary || item.title),
    ...(analysis.warnings || []),
  ].filter(Boolean).slice(0, 2)
  const radarEvents = buildSummaryRadarEvents(analysis.event_impacts || [])

  return (
    <section className="glass rounded-3xl p-5 sm:p-6">
      <div className="flex flex-col gap-5 lg:flex-row lg:items-stretch">
        <div className="flex min-h-[22rem] flex-1 flex-col justify-center rounded-3xl border border-[var(--card-border)] bg-[var(--card-bg)]/40 p-5">
          <div className="flex items-start justify-between gap-4">
            <div className="flex items-start gap-3">
              <div className="rounded-2xl bg-fuchsia-500/15 p-3">
                <Gauge className="h-5 w-5 text-fuchsia-200" />
              </div>
              <div>
                <div className="text-sm font-semibold text-theme-primary">量化观察摘要</div>
                <div className="mt-1 text-xs leading-5 text-theme-muted">
                  只展示核心结论；完整证据、规则和事件拆解请进入量化看板。
                </div>
              </div>
            </div>
            <span className={cn('shrink-0 rounded-full border px-3 py-1 text-xs', dominantTone.soft, dominantTone.color)}>
              {dominantTone.label}
            </span>
          </div>

          <div className="mt-5 grid gap-5 md:grid-cols-[9rem_minmax(0,1fr)] md:items-center">
            <AnimatedScoreGauge value={analysis.total_score} label="SCORE" variant="summary" />

            <div>
              <div className="text-base font-semibold text-theme-primary">{analysis.summary}</div>
              <div className="mt-3 flex flex-wrap gap-2">
                <span className="rounded-full border border-[var(--card-border)] bg-[var(--input-bg)]/60 px-3 py-1 text-xs text-theme-secondary">
                  {analysis.analysis_basis || '分析口径待定'}
                </span>
                <span className="rounded-full border border-cyan-500/20 bg-cyan-500/10 px-3 py-1 text-xs text-cyan-100">
                  {confidenceLabel(analysis.confidence)}
                </span>
                <span className={cn('rounded-full border px-3 py-1 text-xs', analysis.risk_level === 'high' ? 'border-rose-500/20 bg-rose-500/10 text-rose-100' : 'border-emerald-500/20 bg-emerald-500/10 text-emerald-100')}>
                  {riskLevelLabel(analysis.risk_level, '风险待判定')}
                </span>
              </div>

              <div className="mt-5 space-y-2">
                {recommendationItems.map((item) => {
                  const tone = recommendationTone(item.code)
                  return (
                    <div key={item.code} className="grid grid-cols-[5.5rem_minmax(0,1fr)_3.5rem] items-center gap-3 text-xs">
                      <span className="text-theme-muted">{tone.label}</span>
                      <div className="h-2 overflow-hidden rounded-full bg-[var(--input-bg)]">
                        <div className={cn('h-full rounded-full bg-gradient-to-r transition-all duration-700', tone.bar)} style={{ width: `${Math.max(item.value, 0)}%` }} />
                      </div>
                      <span className="text-right font-medium text-theme-primary">{formatAnalysisPercent(item.value)}</span>
                    </div>
                  )
                })}
              </div>
            </div>
          </div>
        </div>

        <div className="grid gap-4 lg:w-[22rem] lg:grid-rows-2">
          <SummaryMiniPanel
            icon={<TrendingUp className="h-4 w-4 text-cyan-200" />}
            title="看这两点"
            items={primaryPoints}
            emptyText="当前主证据不足，建议按弱观察处理。"
          />
          <SummaryMiniPanel
            icon={<ShieldAlert className="h-4 w-4 text-amber-200" />}
            title="主要限制"
            items={riskPoints}
            emptyText="暂未识别到明显反方证据。"
            amber
          />
        </div>
      </div>

      <div className="mt-4 grid gap-3 lg:grid-cols-[minmax(0,1fr)_max-content] lg:items-stretch">
        <SummaryEventRadar events={radarEvents} />

        {fundId && (
          <Link
            href={`/analysis/${fundId}`}
            className={cn(
              'group inline-flex h-full min-h-[5.5rem] items-center justify-center gap-2 rounded-2xl border border-fuchsia-500/25',
              'bg-gradient-to-r from-fuchsia-500/15 to-cyan-500/15 px-5 py-3 text-sm font-semibold text-theme-primary',
              'transition-all duration-300 hover:-translate-y-0.5 hover:border-fuchsia-400/45 hover:shadow-[0_16px_34px_rgba(34,211,238,0.12)]'
            )}
          >
            <Sparkles className="h-4 w-4 text-fuchsia-200" />
            查看完整量化看板
            <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-1" />
          </Link>
        )}
      </div>
    </section>
  )
}

function buildSummaryRadarEvents(events: FundAnalysis['event_impacts']) {
  const currentEvents = events
    .filter((event) => event.horizon === 'current' || event.horizon === 'intraday')
    .filter((event) => event.target_scope !== 'disclosure' && event.target_scope !== 'methodology')

  const picked: NonNullable<FundAnalysis['event_impacts']> = []
  const pickFirst = (predicate: (event: FundAnalysis['event_impacts'][number]) => boolean) => {
    const item = currentEvents.find((event) => !picked.includes(event) && predicate(event))
    if (item) {
      picked.push(item)
    }
  }

  pickFirst((event) => event.target_scope === 'macro' && event.code.startsWith('realtime_'))
  pickFirst((event) => event.target_scope === 'holding')
  pickFirst((event) => event.target_scope === 'exposure')
  pickFirst((event) => event.target_scope === 'macro')

  return picked.slice(0, 3)
}

function SummaryEventRadar({ events }: { events: NonNullable<FundAnalysis['event_impacts']> }) {
  if (events.length === 0) {
    return (
      <div className="flex h-full min-h-[5.5rem] items-center rounded-2xl border border-dashed border-[var(--card-border)] px-4 py-3 text-xs text-theme-muted">
        事件雷达暂无可展示的实时信号；当前结论主要来自行情、持仓和规则口径。
      </div>
    )
  }

  return (
    <div className="grid h-full min-h-[5.5rem] gap-3 md:grid-cols-3">
      {events.map((event) => (
        <div key={event.code} className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 px-4 py-3">
          <div className="mb-2 flex flex-wrap items-center gap-2 text-[11px]">
            <span className={cn('rounded-full border px-2 py-0.5', eventImpactTone(event.impact))}>
              {eventRadarScopeLabel(event.target_scope)}
            </span>
            {event.weight_hint && <span className="text-theme-muted">暴露 {event.weight_hint}%</span>}
          </div>
          <div className="line-clamp-1 text-xs font-semibold text-theme-primary">{event.title}</div>
          <div className="mt-1 line-clamp-2 text-xs leading-5 text-theme-secondary">{compactText(event.summary, 76)}</div>
          {(event.mapping_basis || event.source_name || event.source_published_at) && (
            <div className="mt-2 flex flex-wrap gap-1.5 text-[10px] leading-4 text-theme-muted">
              {event.mapping_basis && (
                <span className="rounded-full border border-[var(--card-border)] bg-[var(--input-bg)]/55 px-2 py-0.5">
                  映射：{compactText(event.mapping_basis, 30)}
                </span>
              )}
              {event.source_url ? (
                <a
                  href={event.source_url}
                  target="_blank"
                  rel="noreferrer"
                  className="rounded-full border border-cyan-500/20 bg-cyan-500/10 px-2 py-0.5 text-cyan-100 transition-colors hover:border-cyan-300/40 hover:text-cyan-50"
                >
                  来源：{event.source_name || '事件源'}
                </a>
              ) : event.source_name ? (
                <span className="rounded-full border border-[var(--card-border)] bg-[var(--input-bg)]/55 px-2 py-0.5">
                  来源：{event.source_name}
                </span>
              ) : null}
              {event.source_published_at && (
                <span className="rounded-full border border-[var(--card-border)] bg-[var(--input-bg)]/55 px-2 py-0.5">
                  {event.source_published_at}
                </span>
              )}
              {event.source_confidence && (
                <span className="rounded-full border border-[var(--card-border)] bg-[var(--input-bg)]/55 px-2 py-0.5">
                  {eventSourceConfidenceLabel(event.source_confidence)}
                </span>
              )}
            </div>
          )}
        </div>
      ))}
    </div>
  )
}

function eventRadarScopeLabel(scope?: string) {
  switch (scope) {
    case 'macro':
      return '实时宏观'
    case 'holding':
      return '持仓事件'
    case 'exposure':
      return '主线暴露'
    case 'fund':
      return '基金公告'
    case 'index':
      return '指数事件'
    default:
      return '事件'
  }
}

function SummaryMiniPanel({
  icon,
  title,
  items,
  emptyText,
  amber = false,
}: {
  icon: ReactNode
  title: string
  items: string[]
  emptyText: string
  amber?: boolean
}) {
  return (
    <div className={cn('flex h-full min-h-[10rem] flex-col rounded-3xl border p-4', amber ? 'border-amber-500/20 bg-amber-500/10' : 'border-cyan-500/20 bg-cyan-500/10')}>
      <div className="mb-3 flex items-center gap-2 text-sm font-semibold text-theme-primary">
        {icon}
        {title}
      </div>
      <div className="flex flex-1 flex-col justify-center gap-2">
        {items.length === 0 ? (
          <div className="text-xs leading-5 text-theme-muted">{emptyText}</div>
        ) : items.map((item, index) => (
          <div key={`${title}-${index}`} className="flex gap-2 text-xs leading-5 text-theme-secondary">
            <span className={cn('mt-2 h-1.5 w-1.5 shrink-0 rounded-full', amber ? 'bg-amber-200' : 'bg-cyan-200')} />
            <span>{compactText(item)}</span>
          </div>
        ))}
      </div>
    </div>
  )
}
