'use client'

import type { ReactNode } from 'react'
import Link from 'next/link'
import { ArrowRight, ShieldAlert, Sparkles, Target } from 'lucide-react'
import { AnimatedScoreGauge } from '@/components/animated-score-gauge'
import { AnalysisEventTraceMeta } from '@/components/analysis-event-trace-meta'
import { FundAnalysisDecisionList } from '@/components/fund-analysis-decision-points'
import { LoadingSpinner } from '@/components/loading-indicator'
import type { FundAnalysis } from '@/hooks/use-fund-data'
import { buildFundAnalysisDecision, type FundAnalysisDecisionPoint } from '@/lib/fund-analysis-decision'
import {
  eventImpactTone,
  formatAnalysisPercent,
  parseAnalysisNumber,
  type FundAnalysisRecommendationCode,
} from '@/lib/fund-analysis-display'
import { cn } from '@/lib/utils'

interface FundAnalysisCardProps {
  analysis?: FundAnalysis
  fundId?: string
  isLoading?: boolean
}

function recommendationTone(code: FundAnalysisRecommendationCode) {
  switch (code) {
    case 'increase':
      return {
        label: '结构偏积极',
        color: 'text-rose-100',
        soft: 'border-rose-500/20 bg-rose-500/10',
        bar: 'from-rose-500 via-fuchsia-500 to-pink-500',
      }
    case 'decrease':
      return {
        label: '风险偏高',
        color: 'text-emerald-100',
        soft: 'border-emerald-500/20 bg-emerald-500/10',
        bar: 'from-emerald-500 via-teal-500 to-cyan-500',
      }
    default:
      return {
        label: '适合观察',
        color: 'text-slate-100',
        soft: 'border-slate-500/20 bg-slate-500/10',
        bar: 'from-slate-500 via-slate-400 to-cyan-400',
      }
  }
}

function compactText(value: string, maxLength = 66) {
  if (value.length <= maxLength) {
    return value
  }
  return `${value.slice(0, maxLength)}…`
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

  const decision = buildFundAnalysisDecision(analysis)
  if (!decision) {
    return null
  }

  const increase = parseAnalysisNumber(analysis.increase_percent)
  const hold = parseAnalysisNumber(analysis.hold_percent)
  const decrease = parseAnalysisNumber(analysis.decrease_percent)
  const recommendationItems = [
    { code: 'increase' as const, value: increase },
    { code: 'hold' as const, value: hold },
    { code: 'decrease' as const, value: decrease },
  ]
  const dominantTone = recommendationTone(decision.result.code)
  const radarEvents = buildSummaryRadarEvents(analysis.event_impacts || [])

  return (
    <section className="glass scroll-mt-36 rounded-3xl p-5 sm:p-6 md:scroll-mt-24">
      <div className="space-y-5">
        <div className="relative overflow-hidden rounded-3xl border border-[var(--card-border)] bg-[var(--card-bg)]/42 p-5">
          <div className="pointer-events-none absolute -right-16 -top-20 h-44 w-44 rounded-full bg-cyan-500/12 blur-3xl" />
          <div className="pointer-events-none absolute -bottom-20 left-8 h-40 w-40 rounded-full bg-sky-500/10 blur-3xl" />

          <div className="relative grid items-center gap-5 lg:grid-cols-[minmax(14rem,0.44fr)_minmax(0,0.56fr)]">
            <div className="rounded-[2rem] border border-cyan-500/14 bg-cyan-500/5 px-4 py-5">
              <AnimatedScoreGauge
                value={analysis.total_score}
                label="SCORE"
                variant="summary"
                className="h-44 w-44 sm:h-48 sm:w-48"
              />
            </div>

            <div className="min-w-0">
              <div className="mb-3 flex flex-wrap items-center gap-2">
                <span className={cn('rounded-full border px-3 py-1 text-xs', dominantTone.soft, dominantTone.color)}>
                  {decision.result.label}
                </span>
                <span className="rounded-full border border-[var(--card-border)] bg-[var(--input-bg)]/60 px-3 py-1 text-xs text-theme-secondary">
                  {decision.basisLabel}
                </span>
                <span className="rounded-full border border-cyan-500/20 bg-cyan-500/10 px-3 py-1 text-xs text-cyan-100">
                  {decision.confidenceLabel}
                </span>
              </div>

              <h2 className="text-xl font-black leading-tight text-theme-primary md:text-2xl">
                {decision.result.summary}
              </h2>
              <p className="mt-3 max-w-2xl text-sm leading-6 text-theme-secondary">
                先看评分，再看方向、风险和证据数量，结论与原因保持同一阅读路径。
              </p>

              <div className="mt-5 grid gap-3 sm:grid-cols-3">
                <DecisionMetric label="主方向" value={`${decision.result.label} ${decision.result.percentLabel}`} />
                <DecisionMetric label="风险" value={decision.riskLabel} />
                <DecisionMetric label="证据" value={decision.evidenceCountLabel} />
              </div>
            </div>
          </div>
        </div>

        <div className="grid items-stretch gap-4 xl:grid-cols-[minmax(18rem,0.74fr)_minmax(0,1.26fr)]">
          {decision.topSignal && (
            <div className="flex h-full flex-col rounded-3xl border border-[var(--card-border)] bg-[var(--card-bg)]/32 p-4">
              <div className="text-[11px] tracking-[0.18em] text-theme-muted">最重要信号</div>
              <div className="mt-2 text-sm font-semibold text-theme-primary">{decision.topSignal.title}</div>
              <div className="mt-1 text-xs leading-5 text-theme-secondary">
                {decision.topSignal.summary}
              </div>
              <div className="mt-auto flex flex-wrap gap-2 pt-4 text-[10px] text-theme-muted">
                <span className="rounded-full border border-[var(--card-border)] bg-[var(--input-bg)]/55 px-2.5 py-1">
                  {decision.topSignal.sourceLabel}
                </span>
                {decision.topSignal.metaLabel && (
                  <span className="rounded-full border border-[var(--card-border)] bg-[var(--input-bg)]/55 px-2.5 py-1">
                    {decision.topSignal.metaLabel}
                  </span>
                )}
                <span className="rounded-full border border-[var(--card-border)] bg-[var(--input-bg)]/55 px-2.5 py-1">
                  {decision.confidenceLabel}
                </span>
              </div>
            </div>
          )}
          <RecommendationDistribution items={recommendationItems} />
        </div>

        <div className="grid gap-4 lg:grid-cols-2">
          <NarrativePanel
            icon={<Target className="h-4 w-4 text-cyan-200" />}
            title="主要依据"
            points={decision.mainReasons}
            emptyText="主要依据不足，按低置信度观察。"
            className="h-full"
          />
          <NarrativePanel
            icon={<ShieldAlert className="h-4 w-4 text-amber-200" />}
            title="需要注意什么"
            points={decision.riskReasons.slice(0, 2)}
            emptyText="暂未识别到明显反方证据。"
            warning
            className="h-full"
          />
        </div>
      </div>

      <div className="mt-4 grid gap-3 lg:grid-cols-[minmax(0,1fr)_max-content] lg:items-stretch">
        <SummaryEventRadar events={radarEvents} />

        {fundId && (
          <Link
            href={`/analysis/${fundId}`}
            className={cn(
              'group inline-flex h-full min-h-[5.5rem] items-center justify-center gap-2 rounded-2xl border border-cyan-500/25',
              'bg-cyan-500/10 px-5 py-3 text-sm font-semibold text-theme-primary',
              'transition-all duration-300 hover:-translate-y-0.5 hover:border-cyan-300/45 hover:shadow-[0_16px_34px_rgba(34,211,238,0.12)]'
            )}
          >
            <Sparkles className="h-4 w-4 text-cyan-200" />
            查看完整量化看板
            <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-1" />
          </Link>
        )}
      </div>
    </section>
  )
}

function DecisionMetric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 px-3 py-3">
      <div className="text-[11px] text-theme-muted">{label}</div>
      <div className="mt-1 truncate text-sm font-semibold text-theme-primary">{value}</div>
    </div>
  )
}

function NarrativePanel({
  icon,
  title,
  points,
  emptyText,
  warning = false,
  className,
}: {
  icon: ReactNode
  title: string
  points: FundAnalysisDecisionPoint[]
  emptyText: string
  warning?: boolean
  className?: string
}) {
  return (
    <div className={cn('rounded-3xl border p-4', warning ? 'border-amber-500/18 bg-amber-500/5' : 'border-cyan-500/18 bg-cyan-500/5', className)}>
      <div className="mb-3 flex items-center gap-2 text-sm font-semibold text-theme-primary">
        {icon}
        {title}
      </div>
      <FundAnalysisDecisionList points={points} emptyText={emptyText} compact />
    </div>
  )
}

function RecommendationDistribution({
  items,
}: {
  items: Array<{ code: FundAnalysisRecommendationCode; value: number }>
}) {
  return (
    <div className="rounded-3xl border border-[var(--card-border)] bg-[var(--card-bg)]/32 p-4">
      <div className="mb-4 flex flex-wrap items-end justify-between gap-3">
        <div>
          <div className="text-sm font-semibold text-theme-primary">建议分布</div>
          <div className="mt-1 text-xs text-theme-muted">把评分拆成积极、观察和风险三类权重</div>
        </div>
        <div className="rounded-full border border-[var(--card-border)] bg-[var(--input-bg)]/55 px-3 py-1 text-[11px] text-theme-muted">
          RULE WEIGHT
        </div>
      </div>

      <div className="grid gap-3 md:grid-cols-3">
        {items.map((item) => {
          const tone = recommendationTone(item.code)
          return (
            <div key={item.code} className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 px-3 py-3">
              <div className="mb-2 flex items-center justify-between gap-2 text-xs">
                <span className="text-theme-muted">{tone.label}</span>
                <span className="font-semibold text-theme-primary">{formatAnalysisPercent(item.value)}</span>
              </div>
              <div className="h-1.5 overflow-hidden rounded-full bg-[var(--input-bg)]">
                <div
                  className={cn('h-full rounded-full bg-gradient-to-r transition-all duration-700', tone.bar)}
                  style={{ width: `${Math.max(item.value, 0)}%` }}
                />
              </div>
            </div>
          )
        })}
      </div>
    </div>
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
        暂无匹配的实时事件；结论主要来自行情、持仓和规则观察。
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
          <AnalysisEventTraceMeta trace={event} dense />
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
