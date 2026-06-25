'use client'

import { AlertTriangle, CheckCircle2, Database, FileSearch, Target } from 'lucide-react'
import { AnalysisEventTraceMeta } from '@/components/analysis-event-trace-meta'
import type { FundAnalysisDecisionPoint, FundAnalysisDecisionTone } from '@/lib/fund-analysis-decision'
import { cn } from '@/lib/utils'

interface FundAnalysisDecisionPointCardProps {
  point: FundAnalysisDecisionPoint
  index?: number
  compact?: boolean
}

interface FundAnalysisDecisionListProps {
  points: FundAnalysisDecisionPoint[]
  emptyText: string
  compact?: boolean
}

function toneClasses(tone: FundAnalysisDecisionTone) {
  switch (tone) {
    case 'positive':
      return {
        shell: 'border-rose-500/22 bg-rose-500/9',
        icon: 'bg-rose-500/12 text-rose-100',
        label: 'border-rose-500/20 bg-rose-500/10 text-rose-100',
      }
    case 'negative':
      return {
        shell: 'border-emerald-500/22 bg-emerald-500/9',
        icon: 'bg-emerald-500/12 text-emerald-100',
        label: 'border-emerald-500/20 bg-emerald-500/10 text-emerald-100',
      }
    case 'warning':
      return {
        shell: 'border-amber-500/24 bg-amber-500/10',
        icon: 'bg-amber-500/12 text-amber-100',
        label: 'border-amber-500/20 bg-amber-500/10 text-amber-100',
      }
    default:
      return {
        shell: 'border-cyan-500/20 bg-cyan-500/8',
        icon: 'bg-cyan-500/12 text-cyan-100',
        label: 'border-cyan-500/20 bg-cyan-500/10 text-cyan-100',
      }
  }
}

function roleIcon(point: FundAnalysisDecisionPoint) {
  if (point.role === 'risk') {
    return <AlertTriangle className="h-4 w-4" />
  }
  if (point.role === 'data') {
    return <Database className="h-4 w-4" />
  }
  if (point.role === 'primary') {
    return <Target className="h-4 w-4" />
  }
  if (point.sourceLabel.includes('事件')) {
    return <FileSearch className="h-4 w-4" />
  }
  return <CheckCircle2 className="h-4 w-4" />
}

export function FundAnalysisDecisionPointCard({ point, index, compact = false }: FundAnalysisDecisionPointCardProps) {
  const tone = toneClasses(point.tone)

  return (
    <article
      className={cn(
        'rounded-2xl border transition-transform duration-300 hover:-translate-y-0.5',
        tone.shell,
        compact ? 'p-3' : 'p-4'
      )}
    >
      <div className="flex items-start gap-3">
        <div className={cn('flex shrink-0 items-center justify-center rounded-xl', tone.icon, compact ? 'h-8 w-8' : 'h-9 w-9')}>
          {index !== undefined ? <span className="text-xs font-black">{index + 1}</span> : roleIcon(point)}
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <h3 className={cn('font-semibold text-theme-primary', compact ? 'text-xs' : 'text-sm')}>{point.title}</h3>
            <span className={cn('rounded-full border px-2 py-0.5 text-[10px]', tone.label)}>{point.sourceLabel}</span>
            {point.metaLabel && (
              <span className="rounded-full border border-[var(--card-border)] bg-[var(--input-bg)]/55 px-2 py-0.5 text-[10px] text-theme-muted">
                {point.metaLabel}
              </span>
            )}
          </div>
          <p className={cn('mt-2 text-theme-secondary', compact ? 'line-clamp-2 text-xs leading-5' : 'text-sm leading-6')}>
            {point.summary}
          </p>
          {point.trace && <AnalysisEventTraceMeta trace={point.trace} dense className="mt-2" />}
        </div>
      </div>
    </article>
  )
}

export function FundAnalysisDecisionList({ points, emptyText, compact = false }: FundAnalysisDecisionListProps) {
  if (points.length === 0) {
    return (
      <div className="rounded-2xl border border-dashed border-[var(--card-border)] bg-[var(--card-bg)]/25 px-4 py-5 text-sm text-theme-muted">
        {emptyText}
      </div>
    )
  }

  return (
    <div className={cn('space-y-3', compact && 'space-y-2')}>
      {points.map((point, index) => (
        <FundAnalysisDecisionPointCard key={`${point.id}-${index}`} point={point} index={index} compact={compact} />
      ))}
    </div>
  )
}
