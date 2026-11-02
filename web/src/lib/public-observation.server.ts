import 'server-only'

import { cache } from 'react'
import { PRODUCTION_API_BASE_URL, resolveBackendBaseUrl } from '@/lib/api-base-url'
import { isObservationIndexable, observationSourceUrl, type FundNotice, type ObservationPayload, type ObservationSource } from '@/lib/fund-observation'

type PublicObservationResult =
  | { status: 'available'; data: ObservationPayload; indexable: boolean }
  | { status: 'missing' }
  | { status: 'unavailable' }

function record(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function text(value: unknown, max = 500) {
  return typeof value === 'string' ? value.trim().slice(0, max) : ''
}

function timestamp(value: unknown) {
  return typeof value === 'string' && /^\d{4}-\d{2}-\d{2}(?:T.*)?$/.test(value) && Number.isFinite(new Date(value).getTime())
}

// Validate anonymous API data before it becomes initial HTML, metadata or JSON-LD.
function observationPayload(value: unknown, id: string): ObservationPayload | undefined {
  if (!record(value) || !record(value.fund) || !record(value.observation)) return
  const fund = value.fund
  const data = value.observation
  if (fund.id !== id || !text(fund.name) || !timestamp(data.checked_at)
    || !Number.isInteger(data.window_days) || Number(data.window_days) < 1 || Number(data.window_days) > 90
    || !['available', 'missing', 'unavailable'].includes(String(data.holdings_status))
    || !Array.isArray(data.notices) || !Array.isArray(data.sources) || !Array.isArray(data.notes)
    || !Number.isInteger(data.holdings_count) || Number(data.holdings_count) < 0
    || typeof data.disclosed_weight_percent !== 'string' || !/^\d+(?:\.\d+)?$/.test(data.disclosed_weight_percent)
    || Number(data.disclosed_weight_percent) > 100) return

  const sources: ObservationSource[] = []
  for (const source of data.sources) {
    if (!record(source) || !['fund_notices', 'holding_notices'].includes(String(source.code))
      || !['ok', 'partial', 'unavailable', 'not_applicable'].includes(String(source.status)) || !timestamp(source.checked_at)) return
    if (sources.some(item => item.code === source.code)) return
    sources.push({ code: source.code as ObservationSource['code'], status: source.status as ObservationSource['status'], checked_at: String(source.checked_at), message: text(source.message) })
  }
  const notices: FundNotice[] = []
  const now = Date.now()
  const today = new Date(now).toLocaleDateString('sv-SE', { timeZone: 'Asia/Shanghai' })
  for (const notice of data.notices) {
    if (!record(notice) || !text(notice.id) || !text(notice.title, 2000) || !timestamp(notice.published_at)
      || !['fund', 'holding', 'target_etf'].includes(String(notice.scope))) return
    const date = new Date(String(notice.published_at))
    if (date.toLocaleDateString('sv-SE', { timeZone: 'Asia/Shanghai' }) > today || now - date.getTime() > (Number(data.window_days) + 1) * 86400000) return
    if (notices.some(item => item.id === notice.id)) continue
    notices.push({
      id: text(notice.id), title: text(notice.title, 2000), scope: notice.scope as FundNotice['scope'],
      published_at: String(notice.published_at), source_name: text(notice.source_name),
      source_url: observationSourceUrl(text(notice.source_url, 4000)) || '',
      related_symbol: text(notice.related_symbol), related_name: text(notice.related_name),
    })
  }
  notices.sort((left, right) => new Date(right.published_at).getTime() - new Date(left.published_at).getTime())
  return {
    fund: { id, name: text(fund.name), type: text(fund.type), company: text(fund.company), manager: text(fund.manager), nav: text(fund.nav), scale: text(fund.scale), updated_at: text(fund.updated_at) },
    observation: {
      checked_at: String(data.checked_at), window_days: Number(data.window_days),
      latest_holding_period: text(data.latest_holding_period),
      holding_age_days: typeof data.holding_age_days === 'number' && Number.isInteger(data.holding_age_days) && data.holding_age_days >= 0 ? data.holding_age_days : null,
      disclosed_weight_percent: data.disclosed_weight_percent, holdings_count: Number(data.holdings_count),
      holdings_source: text(data.holdings_source), holdings_status: data.holdings_status as ObservationPayload['observation']['holdings_status'],
      sources, notices, notes: data.notes.filter((note): note is string => typeof note === 'string').map(note => note.slice(0, 2000)),
    },
  }
}

// React cache deduplicates metadata/page/directory work within one render. The
// Next fetch cache holds anonymous responses for 10 minutes, never user cookies.
export const getPublicObservation = cache(async (id: string): Promise<PublicObservationResult> => {
  if (!/^\d{6}$/.test(id)) return { status: 'missing' }
  const base = resolveBackendBaseUrl(process.env.BACKEND_URL, process.env.VERCEL ? PRODUCTION_API_BASE_URL : 'http://127.0.0.1:8080')
  try {
    const response = await fetch(`${base}/api/v1/fund/${id}/observation`, {
      headers: { Accept: 'application/json' }, credentials: 'omit', redirect: 'error',
      signal: AbortSignal.timeout(17_000), next: { revalidate: 600 },
    })
    const envelope: unknown = await response.json()
    if (!record(envelope)) return { status: 'unavailable' }
    if (response.status === 404 && envelope.success === false && record(envelope.error) && envelope.error.code === 'FUND_NOT_FOUND') return { status: 'missing' }
    if (!response.ok || envelope.success !== true) return { status: 'unavailable' }
    const data = observationPayload(envelope.data, id)
    return data ? { status: 'available', data, indexable: isObservationIndexable(data) } : { status: 'unavailable' }
  } catch {
    return { status: 'unavailable' }
  }
})

// Existing search examples, not a ranking or exhaustive catalogue. Only live,
// validated, complete snapshots are published as directory/sitemap entries.
const directoryIds = ['005827', '003095', '320007']

export async function getPublicObservationDirectory() {
  const results = await Promise.all(directoryIds.map(id => getPublicObservation(id)))
  return results.flatMap(result => result.status === 'available' && result.indexable ? [result.data] : [])
}
