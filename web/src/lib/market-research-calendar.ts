import holidays from './market-research-holidays.json'
import type { ResearchSourceMeta } from '../hooks/use-market-research'

const OFFSET = 8 * 60 * 60 * 1000
const holidayDates = new Set(Object.values(holidays.years).flat())

export const marketCalendarSource = holidays.source

export function beijingDate(now: Date): string {
  return new Date(now.getTime() + OFFSET).toISOString().slice(0, 10)
}

export function beijingTimestamp(now: Date): string {
  return new Date(now.getTime() + OFFSET).toISOString().slice(0, 19) + '+08:00'
}

function localMinutes(now: Date): number {
  const shifted = new Date(now.getTime() + OFFSET)
  return shifted.getUTCHours() * 60 + shifted.getUTCMinutes()
}

export function validDate(date: string): boolean {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(date)) return false
  const parsed = new Date(`${date}T00:00:00Z`)
  return Number.isFinite(parsed.getTime()) && parsed.toISOString().slice(0, 10) === date
}

export function isTradingDay(date: string): boolean {
  if (!validDate(date)) return false
  const day = new Date(`${date}T00:00:00Z`).getUTCDay()
  return day !== 0 && day !== 6 && !holidayDates.has(date)
}

export function previousTradingDate(now: Date): string {
  const candidate = new Date(`${beijingDate(now)}T00:00:00Z`)
  for (let i = 0; i < 370; i++) {
    candidate.setUTCDate(candidate.getUTCDate() - 1)
    const date = candidate.toISOString().slice(0, 10)
    if (isTradingDay(date)) return date
  }
  throw new Error('无法确定上一交易日')
}

export function marketPhase(now: Date): string {
  const date = beijingDate(now)
  const day = new Date(`${date}T00:00:00Z`).getUTCDay()
  if (day === 0 || day === 6) return 'weekend'
  if (holidayDates.has(date)) return 'holiday'
  const minutes = localMinutes(now)
  if (minutes < 9 * 60) return 'pre_market'
  if (minutes < 9 * 60 + 30) return 'call_auction'
  if (minutes < 11 * 60 + 30) return 'morning'
  if (minutes < 13 * 60) return 'lunch_break'
  if (minutes < 15 * 60) return 'afternoon'
  return 'after_hours'
}

export function expectedDataDate(now: Date): string {
  return isTradingDay(beijingDate(now)) && localMinutes(now) >= 9 * 60 + 30
    ? beijingDate(now) : previousTradingDate(now)
}

export function dayCompleted(date: string, now: Date): boolean {
  if (!validDate(date)) return false
  const today = beijingDate(now)
  return date < today || date === today && localMinutes(now) >= 15 * 60 + 5
}

// A pre-close snapshot remains unfinished if its refresh fails across the close boundary.
export function historyAsOf(meta: ResearchSourceMeta, now: Date): Date {
  const fetched = new Date(meta.fetched_at)
  return Number.isFinite(fetched.getTime()) && fetched < now && !dayCompleted(beijingDate(fetched), fetched)
    ? fetched : now
}

export function dataPointWarning(date: string, time: string, now: Date): string {
  const expected = expectedDataDate(now)
  if (!validDate(date) || date !== expected) return `数据日期为 ${date || '未知'}，当前应观察 ${expected}；保留旧数据`
  if (!/^([01]\d|2[0-3]):[0-5]\d(?::[0-5]\d)?$/.test(time)) return '最后更新时间无效'
  const point = new Date(`${date}T${time.length === 5 ? `${time}:00` : time}+08:00`)
  if (point.getTime() > now.getTime() + 60_000) return '公开源返回未来时间，无法确认数据有效性'
  const phase = marketPhase(now)
  let reference = new Date(`${date}T15:00:00+08:00`).getTime()
  if (phase === 'morning' || phase === 'afternoon') reference = now.getTime()
  if (phase === 'lunch_break') reference = new Date(`${date}T11:30:00+08:00`).getTime()
  if (reference - point.getTime() > 180_000) return `最后点为 ${date} ${time}，落后于应有时点；公开源可能延迟`
  return ''
}
