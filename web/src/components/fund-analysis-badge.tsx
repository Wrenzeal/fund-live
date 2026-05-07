'use client'

import { cn } from '@/lib/utils'
import type { FundAnalysis } from '@/hooks/use-fund-data'

interface FundAnalysisBadgeProps {
  analysis?: FundAnalysis | null
  compact?: boolean
  showScore?: boolean
}

function parsePercent(value?: string) {
  const parsed = Number.parseFloat(value || '')
  return Number.isNaN(parsed) ? 0 : parsed
}

function parseScore(value?: string) {
  const parsed = Number.parseFloat(value || '')
  return Number.isNaN(parsed) ? '--' : parsed.toFixed(1)
}

function recommendationMeta(analysis?: FundAnalysis | null) {
  if (!analysis) {
    return null
  }

  const increase = parsePercent(analysis.increase_percent)
  const hold = parsePercent(analysis.hold_percent)
  const decrease = parsePercent(analysis.decrease_percent)

  if (increase >= hold && increase >= decrease) {
    return {
      label: '偏加仓',
      tone: 'border-rose-400/35 bg-rose-500/12 text-rose-100',
    }
  }
  if (decrease >= increase && decrease >= hold) {
    return {
      label: '偏减仓',
      tone: 'border-emerald-400/35 bg-emerald-500/12 text-emerald-100',
    }
  }
  return {
    label: '偏持有',
    tone: 'border-slate-400/35 bg-slate-500/12 text-slate-100',
  }
}

function riskMeta(level?: string) {
  switch (level) {
    case 'low':
      return {
        label: '低风险',
        tone: 'border-emerald-400/25 bg-emerald-500/10 text-emerald-100',
      }
    case 'high':
      return {
        label: '高风险',
        tone: 'border-amber-400/25 bg-amber-500/10 text-amber-100',
      }
    default:
      return {
        label: '中风险',
        tone: 'border-cyan-400/25 bg-cyan-500/10 text-cyan-100',
      }
  }
}

export function FundAnalysisBadge({ analysis, compact = false, showScore = false }: FundAnalysisBadgeProps) {
  const recommendation = recommendationMeta(analysis)
  if (!recommendation || !analysis) {
    return null
  }

  const risk = riskMeta(analysis.risk_level)
  const score = parseScore(analysis.total_score)

  return (
    <div className={cn('flex flex-wrap items-center gap-2', compact && 'gap-1.5')}>
      <span
        className={cn(
          'rounded-full border px-2.5 py-1 text-[11px] font-medium',
          compact ? 'px-2 py-0.5 text-[10px]' : '',
          recommendation.tone
        )}
      >
        {recommendation.label}
      </span>
      <span
        className={cn(
          'rounded-full border px-2.5 py-1 text-[11px] font-medium',
          compact ? 'px-2 py-0.5 text-[10px]' : '',
          risk.tone
        )}
      >
        {risk.label}
      </span>
      {showScore && (
        <span className={cn(
          'rounded-full border border-[var(--input-border)] bg-[var(--input-bg)] px-2.5 py-1 text-[11px] text-theme-secondary',
          compact ? 'px-2 py-0.5 text-[10px]' : ''
        )}>
          评分 {score}
        </span>
      )}
    </div>
  )
}
