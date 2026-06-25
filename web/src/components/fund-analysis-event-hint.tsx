'use client'

import type { FundAnalysis } from '@/hooks/use-fund-data'
import { buildFundAnalysisDecision } from '@/lib/fund-analysis-decision'

interface FundAnalysisEventHintProps {
  analysis?: FundAnalysis | null
  compact?: boolean
}

export function FundAnalysisEventHint({ analysis, compact = false }: FundAnalysisEventHintProps) {
  const decision = buildFundAnalysisDecision(analysis)
  const signal = decision?.topSignal
  if (!signal) {
    return null
  }

  return (
    <div className={compact ? 'text-[11px] text-theme-muted' : 'text-xs text-theme-secondary'}>
      <span className="font-medium text-theme-primary">量化原因：</span>
      <span>{signal.title}</span>
      {!compact && <span className="text-theme-muted">，{signal.sourceLabel}</span>}
    </div>
  )
}
