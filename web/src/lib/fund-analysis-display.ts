export type FundAnalysisRecommendationCode = 'increase' | 'hold' | 'decrease'

export const FUND_ANALYSIS_INCREASE_THRESHOLD = 55
export const FUND_ANALYSIS_DECREASE_THRESHOLD = 60

export function parseAnalysisNumber(value?: string | number): number {
  const parsed = typeof value === 'number' ? value : Number.parseFloat(value || '')
  return Number.isFinite(parsed) ? parsed : 0
}

export function formatAnalysisPercent(value?: string | number): string {
  const parsed = parseAnalysisNumber(value)
  return `${parsed.toFixed(1)}%`
}

export function formatAnalysisScore(value?: string | number): string {
  const parsed = typeof value === 'number' ? value : Number.parseFloat(value || '')
  return Number.isFinite(parsed) ? parsed.toFixed(1) : '--'
}

export function dominantAnalysisRecommendation(analysis?: { increase_percent?: string; decrease_percent?: string } | null): FundAnalysisRecommendationCode {
  const increase = parseAnalysisNumber(analysis?.increase_percent)
  const decrease = parseAnalysisNumber(analysis?.decrease_percent)

  if (increase >= FUND_ANALYSIS_INCREASE_THRESHOLD) {
    return 'increase'
  }
  if (decrease >= FUND_ANALYSIS_DECREASE_THRESHOLD) {
    return 'decrease'
  }
  return 'hold'
}

export function riskLevelLabel(level?: string, unknownLabel = '待判断'): string {
  switch (level) {
    case 'low':
      return '低风险'
    case 'medium':
      return '中风险'
    case 'high':
      return '高风险'
    default:
      return unknownLabel
  }
}

export function confidenceLevelLabel(level?: string, labels?: Partial<Record<'high' | 'medium' | 'low' | 'unknown', string>>): string {
  switch (level) {
    case 'high':
      return labels?.high || '覆盖较高'
    case 'medium':
      return labels?.medium || '覆盖一般'
    case 'low':
      return labels?.low || '覆盖有限'
    default:
      return labels?.unknown || '覆盖未知'
  }
}

export function eventHorizonLabel(horizon?: string): string {
  switch (horizon) {
    case 'intraday':
      return '盘中'
    case 'current':
      return '当前'
    case 'quarterly':
      return '季报变化'
    case 'medium_term':
      return '中期'
    default:
      return '事件'
  }
}

export function eventStrengthLabel(strength?: string): string {
  switch (strength) {
    case 'high':
      return '高强度'
    case 'medium':
      return '中强度'
    case 'low':
      return '低强度'
    default:
      return '普通'
  }
}

export function eventImpactLabel(impact?: string): string {
  switch (impact) {
    case 'positive':
      return '偏正向'
    case 'negative':
      return '偏负向'
    default:
      return '中性'
  }
}

export function eventImpactTone(impact?: string): string {
  switch (impact) {
    case 'positive':
      return 'border-emerald-500/25 bg-emerald-500/10 text-emerald-100'
    case 'negative':
      return 'border-rose-500/25 bg-rose-500/10 text-rose-100'
    default:
      return 'border-cyan-500/25 bg-cyan-500/10 text-cyan-100'
  }
}

export function eventTimelineRank(event: { horizon?: string; strength?: string }): number {
  const horizonRank = event.horizon === 'intraday'
    ? 0
    : event.horizon === 'current'
      ? 1
      : event.horizon === 'quarterly'
        ? 2
        : 3
  const strengthRank = event.strength === 'high'
    ? 0
    : event.strength === 'medium'
      ? 1
      : 2
  return horizonRank * 10 + strengthRank
}
