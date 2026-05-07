'use client'

import Link from 'next/link'
import { AlertTriangle, ArrowRight, Gauge, ShieldAlert, Sparkles, TrendingUp, Zap } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { FundAnalysis } from '@/hooks/use-fund-data'
import { LoadingSpinner } from '@/components/loading-indicator'

interface FundAnalysisCardProps {
  analysis?: FundAnalysis
  fundId?: string
  isLoading?: boolean
}

function riskLevelLabel(level?: string) {
  switch (level) {
    case 'low':
      return '低风险'
    case 'medium':
      return '中风险'
    case 'high':
      return '高风险'
    default:
      return '风险待判定'
  }
}

function confidenceLabel(level?: string) {
  switch (level) {
    case 'high':
      return '覆盖较高'
    case 'medium':
      return '覆盖一般'
    case 'low':
      return '覆盖有限'
    default:
      return '覆盖未知'
  }
}

function recommendationTone(code: 'increase' | 'hold' | 'decrease') {
  switch (code) {
    case 'increase':
      return {
        label: '加仓',
        bar: 'from-rose-500 via-fuchsia-500 to-pink-500',
        text: 'text-rose-200',
        soft: 'bg-rose-500/10 border-rose-500/20',
      }
    case 'decrease':
      return {
        label: '减仓',
        bar: 'from-emerald-500 via-teal-500 to-cyan-500',
        text: 'text-emerald-200',
        soft: 'bg-emerald-500/10 border-emerald-500/20',
      }
    default:
      return {
        label: '持有',
        bar: 'from-slate-500 via-slate-400 to-cyan-400',
        text: 'text-slate-200',
        soft: 'bg-slate-500/10 border-slate-500/20',
      }
  }
}

function scoreTone(score: number) {
  if (score >= 70) {
    return 'border-cyan-500/20 bg-cyan-500/10 text-cyan-100'
  }
  if (score >= 50) {
    return 'border-amber-500/20 bg-amber-500/10 text-amber-100'
  }
  return 'border-rose-500/20 bg-rose-500/10 text-rose-100'
}

function impactTone(impact?: string) {
  switch (impact) {
    case 'positive':
      return 'border-emerald-500/20 bg-emerald-500/10 text-emerald-100'
    case 'negative':
      return 'border-rose-500/20 bg-rose-500/10 text-rose-100'
    default:
      return 'border-cyan-500/20 bg-cyan-500/10 text-cyan-100'
  }
}

function impactLabel(impact?: string) {
  switch (impact) {
    case 'positive':
      return '正向'
    case 'negative':
      return '偏负向'
    default:
      return '中性'
  }
}

function eventScopeLabel(scope?: string) {
  switch (scope) {
    case 'disclosure':
      return '披露'
    case 'methodology':
      return '口径'
    case 'holding':
      return '重仓股'
    case 'exposure':
      return '主线'
    case 'fund':
      return '基金'
    case 'macro':
      return '宏观'
    case 'index':
      return '指数'
    default:
      return ''
  }
}

function eventStrengthLabel(strength?: string) {
  switch (strength) {
    case 'high':
      return '强'
    case 'medium':
      return '中'
    case 'low':
      return '弱'
    default:
      return ''
  }
}

function eventHorizonLabel(horizon?: string) {
  switch (horizon) {
    case 'intraday':
      return '盘中'
    case 'current':
      return '当前'
    case 'quarterly':
      return '季度'
    case 'medium_term':
      return '中期'
    default:
      return ''
  }
}

function formatScore(value?: string) {
  const parsed = Number.parseFloat(value || '')
  if (Number.isNaN(parsed)) {
    return '--'
  }
  return parsed.toFixed(1)
}

function formatPercentValue(value?: string) {
  const parsed = Number.parseFloat(value || '')
  if (Number.isNaN(parsed)) {
    return '--'
  }
  return `${parsed.toFixed(1)}%`
}

function formatWeightHint(value?: string) {
  const parsed = Number.parseFloat(value || '')
  if (Number.isNaN(parsed) || parsed <= 0) {
    return ''
  }
  return `${parsed.toFixed(1)}%`
}

export function FundAnalysisCard({ analysis, fundId, isLoading = false }: FundAnalysisCardProps) {
  if (!analysis && !isLoading) {
    return null
  }

  if (!analysis) {
    return (
      <section className="glass rounded-3xl p-5 sm:p-6">
        <div className="rounded-3xl border border-[var(--card-border)] bg-[var(--card-bg)]/40 px-5 py-10">
          <LoadingSpinner size="md" text="量化分析加载中..." />
        </div>
      </section>
    )
  }

  const increase = Number.parseFloat(analysis.increase_percent || '0')
  const hold = Number.parseFloat(analysis.hold_percent || '0')
  const decrease = Number.parseFloat(analysis.decrease_percent || '0')
  const totalScore = Number.parseFloat(analysis.total_score || '0')
  const recommendationItems = [
    { code: 'increase' as const, value: increase },
    { code: 'hold' as const, value: hold },
    { code: 'decrease' as const, value: decrease },
  ]
  const dominantRecommendation = recommendationItems
    .slice()
    .sort((left, right) => right.value - left.value)[0]?.code || 'hold'
  const dominantTone = recommendationTone(dominantRecommendation)

  const safeWidth = (value: number) => Number.isFinite(value) ? Math.max(value, 0) : 0

  return (
    <section className="glass rounded-3xl p-5 sm:p-6">
      <div className="rounded-3xl border border-[var(--card-border)] bg-[var(--card-bg)]/40 p-5">
        <div className="flex items-start gap-3">
          <div className="rounded-2xl bg-fuchsia-500/15 p-3">
            <Gauge className="h-5 w-5 text-fuchsia-200" />
          </div>
          <div>
            <div className="text-sm font-semibold text-theme-primary">量化分析</div>
            <div className="text-xs text-theme-muted">
              第一版以规则打分为主，先输出建议分布与结构化解释。
            </div>
          </div>
        </div>

        <div className="mt-4 flex flex-wrap gap-2">
          <span className="rounded-full border border-[var(--card-border)] bg-[var(--input-bg)]/60 px-3 py-1 text-xs text-theme-secondary">
            分析口径：<span className="font-medium text-theme-primary">{analysis.analysis_basis}</span>
          </span>
          <span className={cn('rounded-full border px-3 py-1 text-xs', analysis.confidence === 'low' ? 'border-amber-500/20 bg-amber-500/10 text-amber-100' : 'border-cyan-500/20 bg-cyan-500/10 text-cyan-100')}>
            识别覆盖：{confidenceLabel(analysis.confidence)}
          </span>
          <span className={cn('rounded-full border px-3 py-1 text-xs', analysis.risk_level === 'high' ? 'border-rose-500/20 bg-rose-500/10 text-rose-100' : analysis.risk_level === 'medium' ? 'border-amber-500/20 bg-amber-500/10 text-amber-100' : 'border-emerald-500/20 bg-emerald-500/10 text-emerald-100')}>
            风险等级：{riskLevelLabel(analysis.risk_level)}
          </span>
          {analysis.latest_holding_period && (
            <span className="rounded-full border border-[var(--card-border)] bg-[var(--input-bg)]/60 px-3 py-1 text-xs text-theme-secondary">
              最新持仓披露：<span className="font-medium text-theme-primary">{analysis.latest_holding_period}</span>
            </span>
          )}
        </div>
      </div>

      <div className="mt-5 rounded-3xl border border-[var(--card-border)] bg-[var(--card-bg)]/28 p-5">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <div className="text-sm font-semibold text-theme-primary">六维模块评分</div>
            <div className="mt-1 text-xs text-theme-muted">
              趋势、结构、热度、风险、性价比与事件共同组成当前量化建议。
            </div>
          </div>
          <div className="text-xs text-theme-muted">
            当前模块分数是基础版规则结果，后续会继续用样本池校准权重、阈值和事件影响。
          </div>
        </div>

        <div className="mt-4 grid gap-3 md:grid-cols-2 2xl:grid-cols-3">
          {analysis.module_scores.map((module) => {
            const score = Number.parseFloat(module.score || '0')
            return (
              <div
                key={module.code}
                className={cn('rounded-2xl border p-4 min-h-[96px]', scoreTone(score))}
              >
                <div className="flex items-start justify-between gap-4">
                  <div className="text-sm font-semibold">{module.name}</div>
                  <div className="shrink-0 text-right">
                    <div className="text-lg font-bold">{formatScore(module.score)}</div>
                    <div className="mt-1 text-[11px] tracking-[0.18em] text-theme-muted">SCORE</div>
                  </div>
                </div>
                {module.summary && (
                  <div className="mt-3 text-xs leading-5 text-theme-secondary">{module.summary}</div>
                )}
              </div>
            )
          })}
        </div>
      </div>

      <div className="mt-6 grid gap-4 xl:grid-cols-2">
        <div className="rounded-2xl border border-fuchsia-500/20 bg-fuchsia-500/10 p-5 xl:col-span-2">
          <div className="mb-3 flex items-center gap-2 text-sm font-semibold text-fuchsia-100">
            <Zap className="h-4 w-4" />
            事件分析
          </div>
          <div className="grid gap-3 xl:grid-cols-2">
            {analysis.event_impacts.map((event) => (
              <div key={event.code} className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/40 p-4">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="text-sm font-semibold text-theme-primary">{event.title}</div>
                    <div className="mt-2 flex flex-wrap gap-2">
                      {event.target_scope && (
                        <span className="rounded-full border border-[var(--card-border)] bg-[var(--input-bg)]/55 px-2.5 py-1 text-[11px] text-theme-secondary">
                          {eventScopeLabel(event.target_scope)}
                        </span>
                      )}
                      {event.horizon && (
                        <span className="rounded-full border border-[var(--card-border)] bg-[var(--input-bg)]/55 px-2.5 py-1 text-[11px] text-theme-secondary">
                          {eventHorizonLabel(event.horizon)}
                        </span>
                      )}
                      {event.strength && (
                        <span className="rounded-full border border-[var(--card-border)] bg-[var(--input-bg)]/55 px-2.5 py-1 text-[11px] text-theme-secondary">
                          强度：{eventStrengthLabel(event.strength)}
                        </span>
                      )}
                      {event.weight_hint && (
                        <span className="rounded-full border border-[var(--card-border)] bg-[var(--input-bg)]/55 px-2.5 py-1 text-[11px] text-theme-secondary">
                          权重提示：{formatWeightHint(event.weight_hint)}
                        </span>
                      )}
                    </div>
                    <div className="mt-2 text-xs leading-5 text-theme-secondary">{event.summary}</div>
                    {event.related_symbols && event.related_symbols.length > 0 && (
                      <div className="mt-2 text-[11px] text-theme-muted">
                        相关标的：{event.related_symbols.join(' / ')}
                      </div>
                    )}
                  </div>
                  <span className={cn('rounded-full border px-2.5 py-1 text-[11px]', impactTone(event.impact))}>
                    {impactLabel(event.impact)}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="rounded-2xl border border-cyan-500/20 bg-cyan-500/10 p-5">
          <div className="mb-3 flex items-center gap-2 text-sm font-semibold text-cyan-100">
            <TrendingUp className="h-4 w-4" />
            主要支撑理由
          </div>
          <div className="space-y-2">
            {analysis.reasons.map((reason, index) => (
              <div key={`${reason}-${index}`} className="flex gap-3 text-sm text-theme-secondary">
                <span className="mt-1 inline-block h-1.5 w-1.5 shrink-0 rounded-full bg-cyan-300" />
                <span>{reason}</span>
              </div>
            ))}
          </div>
        </div>

        <div className="rounded-2xl border border-amber-500/20 bg-amber-500/10 p-5">
          <div className="mb-3 flex items-center gap-2 text-sm font-semibold text-amber-100">
            <ShieldAlert className="h-4 w-4" />
            风险与限制
          </div>
          <div className="space-y-2">
            {analysis.warnings.map((warning, index) => (
              <div key={`${warning}-${index}`} className="flex gap-3 text-sm text-theme-secondary">
                <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-amber-200" />
                <span>{warning}</span>
              </div>
            ))}
          </div>
        </div>
      </div>

      <div className="mt-5 rounded-3xl border border-[var(--card-border)] bg-[var(--card-bg)]/34 p-5">
        <div className="text-xs font-medium tracking-[0.18em] text-theme-muted">综合结论</div>

        <div className="mt-4 grid gap-4 xl:grid-cols-[minmax(16rem,20rem)_minmax(0,1fr)] xl:items-center">
          <div className="rounded-[1.75rem] border border-fuchsia-400/20 bg-gradient-to-br from-fuchsia-500/12 via-[var(--input-bg)]/65 to-cyan-500/10 px-6 py-5 shadow-[0_18px_45px_rgba(12,18,44,0.35)] ring-1 ring-white/5">
            <div className="text-[11px] tracking-[0.28em] text-theme-muted">TOTAL SCORE</div>
            <div className="mt-3 flex items-end gap-3">
              <div className={cn('text-7xl font-black tracking-tight drop-shadow-[0_8px_30px_rgba(15,23,42,0.35)] sm:text-8xl', totalScore >= 65 ? 'text-up' : totalScore <= 40 ? 'text-down' : 'text-theme-primary')}>
                {formatScore(analysis.total_score)}
              </div>
              <div className="pb-3 text-sm text-theme-muted/75 sm:text-base">/ 100</div>
            </div>
          </div>

          <div className={cn('rounded-2xl border px-4 py-4 text-sm', dominantTone.soft)}>
            <div className="mb-2 flex items-center gap-2 text-theme-primary">
              <Sparkles className="h-4 w-4" />
              <span className="font-medium">基础结论</span>
            </div>
            <div className="text-sm leading-6 text-theme-secondary sm:text-base">
              {analysis.summary}
            </div>
          </div>
        </div>

        <div className="mt-4 grid gap-3 md:grid-cols-3">
          {recommendationItems.map((item) => {
            const tone = recommendationTone(item.code)
            return (
              <div key={item.code} className={cn('rounded-2xl border px-4 py-4', tone.soft)}>
                <div className="flex items-center justify-between gap-3 text-sm">
                  <span className="text-theme-secondary">{tone.label}</span>
                  <span className={cn('font-semibold', tone.text)}>{formatPercentValue(String(item.value))}</span>
                </div>
                <div className="mt-3 h-2.5 overflow-hidden rounded-full bg-[var(--input-bg)]">
                  <div
                    className={cn('h-full rounded-full bg-gradient-to-r transition-all duration-500', tone.bar)}
                    style={{ width: `${safeWidth(item.value)}%` }}
                  />
                </div>
              </div>
            )
          })}
        </div>

        {analysis.confidence === 'low' && (
          <div className="mt-4 rounded-2xl border border-amber-500/20 bg-amber-500/10 px-4 py-3 text-sm text-amber-100">
            当前基础版量化分析仍明显依赖持仓覆盖率、行业/主题识别与盘中走势；若持仓披露偏旧或未归类占比偏高，应把结果当作辅助参考。
          </div>
        )}

        {fundId && (
          <div className="mt-4">
            <Link
              href={`/analysis/${fundId}`}
              className={cn(
                'group relative flex w-full flex-col items-start gap-3 overflow-hidden rounded-2xl border border-fuchsia-500/25',
                'bg-gradient-to-r from-fuchsia-500/15 via-cyan-500/10 to-sky-500/15 px-4 py-4',
                'transition-all duration-300 hover:-translate-y-0.5 hover:border-fuchsia-400/45 hover:shadow-[0_18px_40px_rgba(34,211,238,0.12)]',
                'active:scale-[0.99] sm:flex-row sm:items-center sm:justify-between sm:gap-4 sm:px-5'
              )}
            >
              <span className="pointer-events-none absolute inset-0 -translate-x-full bg-gradient-to-r from-transparent via-white/10 to-transparent transition-transform duration-700 group-hover:translate-x-full" />
              <div className="relative z-10 min-w-0 flex-1">
                <div className="text-sm font-semibold text-theme-primary sm:text-base">进入完整量化看板</div>
                <div className="mt-1 text-xs leading-5 text-theme-secondary sm:text-sm">
                  查看完整结构、事件影响、持仓明细与方法说明
                </div>
              </div>
              <div className="relative z-10 flex h-11 w-11 shrink-0 self-end items-center justify-center rounded-2xl border border-cyan-400/25 bg-cyan-400/10 text-cyan-100 transition-all duration-300 group-hover:translate-x-1 group-hover:bg-cyan-400/15 sm:self-auto">
                <ArrowRight className="h-5 w-5" />
              </div>
            </Link>
          </div>
        )}
      </div>
    </section>
  )
}
