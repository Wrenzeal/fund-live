import type { Fund } from '@/hooks/use-fund-data'

export interface FundNotice {
  id: string
  title: string
  scope: 'fund' | 'holding' | 'target_etf'
  published_at: string
  source_name: string
  source_url: string
  related_symbol?: string
  related_name?: string
}

export interface ObservationSource {
  code: 'fund_notices' | 'holding_notices'
  status: 'ok' | 'partial' | 'unavailable' | 'not_applicable'
  checked_at: string
  message?: string
}

export interface FundObservation {
  checked_at: string
  window_days: number
  latest_holding_period: string
  holding_age_days: number | null
  disclosed_weight_percent: string
  holdings_count: number
  holdings_source: string
  holdings_status: 'available' | 'missing' | 'unavailable'
  notices: FundNotice[]
  sources: ObservationSource[]
  notes: string[]
}

export interface ObservationPayload {
  fund: Fund
  observation: FundObservation
}

export function formatObservationDate(value: string, withTime = false) {
  if (!value) return '未提供'
  const date = new Date(value)
  if (!Number.isFinite(date.getTime())) return '未提供'
  const day = date.toLocaleDateString('sv-SE', { timeZone: 'Asia/Shanghai' })
  return withTime ? `${day} ${date.toLocaleTimeString('zh-CN', { timeZone: 'Asia/Shanghai', hour: '2-digit', minute: '2-digit', hour12: false })}` : day
}

export function formatReportPeriod(value: string) {
  const quarter = value.match(/^(\d{4})-?Q([1-4])$/i)
  if (quarter) return `${quarter[1]} 年第 ${quarter[2]} 季度`
  const date = formatObservationDate(value)
  return date === '未提供' ? value || date : date
}

export function observationSourceUrl(value: string) {
  try {
    const url = new URL(value)
    return ['https:', 'http:'].includes(url.protocol) && !url.username && !url.password ? url.href : undefined
  } catch {
    return undefined
  }
}

export function isObservationIndexable(payload: ObservationPayload, now = Date.now()) {
  const data = payload.observation
  const checked = new Date(data.checked_at).getTime()
  const isFresh = (value: number) => Number.isFinite(value) && value <= now + 60_000 && now - value < 24 * 60 * 60 * 1000
  const fresh = isFresh(checked) && data.sources.every(source => isFresh(new Date(source.checked_at).getTime()))
  const fundSource = data.sources.find(source => source.code === 'fund_notices')
  const holdingsSource = data.sources.find(source => source.code === 'holding_notices')
  return fresh && fundSource?.status === 'ok' && Boolean(holdingsSource && ['ok', 'not_applicable'].includes(holdingsSource.status))
    && data.holdings_status !== 'unavailable'
    && (data.holdings_status !== 'available' || Boolean(data.latest_holding_period && data.holdings_source))
    && data.notices.every(notice => Boolean(notice.source_name && observationSourceUrl(notice.source_url)))
}
