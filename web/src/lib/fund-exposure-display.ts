// Exposure coverage describes mapped disclosed holdings, not prediction accuracy.
export function coverageLevelLabel(level?: string, labels?: Partial<Record<'high' | 'medium' | 'low' | 'unknown', string>>) {
  switch (level) {
    case 'high': return labels?.high || '覆盖较高'
    case 'medium': return labels?.medium || '覆盖一般'
    case 'low': return labels?.low || '覆盖有限'
    default: return labels?.unknown || '覆盖未知'
  }
}
