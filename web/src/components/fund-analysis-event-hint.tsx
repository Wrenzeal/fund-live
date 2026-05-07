'use client'

import type { FundAnalysis } from '@/hooks/use-fund-data'

interface FundAnalysisEventHintProps {
  analysis?: FundAnalysis | null
  compact?: boolean
}

function pickPrimaryEvent(analysis?: FundAnalysis | null) {
  if (!analysis?.event_impacts?.length) {
    return null
  }
  return analysis.event_impacts.find((item) => item.code !== 'analysis_basis') || analysis.event_impacts[0]
}

export function FundAnalysisEventHint({ analysis, compact = false }: FundAnalysisEventHintProps) {
  const event = pickPrimaryEvent(analysis)
  if (!event) {
    return null
  }

  return (
    <div className={compact ? 'text-[11px] text-theme-muted' : 'text-xs text-theme-secondary'}>
      <span className="font-medium text-theme-primary">事件关注：</span>
      <span>{event.title || event.summary}</span>
    </div>
  )
}
