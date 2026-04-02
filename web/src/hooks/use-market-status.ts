'use client'

import { useEffect, useMemo, useState } from 'react'
import useSWR, { mutate as globalMutate } from 'swr'

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || ''
const STATUS_KEY = `${API_BASE_URL}/api/v1/market/status`
const STATUS_FALLBACK_REFRESH_MS = 5 * 60 * 1000

export type MarketSession =
  | 'pre_market'
  | 'morning'
  | 'lunch_break'
  | 'afternoon'
  | 'after_hours'
  | 'weekend'
  | 'holiday'

interface ApiEnvelope<T> {
  success: boolean
  data?: T
  error?: {
    code: string
    message: string
  }
}

interface RawMarketStatus {
  is_trading: boolean
  is_trading_day: boolean
  session: MarketSession
  current_time: string
  current_date: string
  display_date: string
  previous_trading_day: string
  last_trading_day: string
  next_trading_day: string
  next_session_start?: string
  next_transition_at?: string
  time_until_next_session_seconds: number
  time_until_next_transition_seconds: number
  timezone: string
  calendar_source: string
  covered_years: number[]
}

interface RawPricingDatePreview {
  trade_at: string
  trade_date: string
  pricing_date: string
  rule: 'same_day_close' | 'next_trading_day'
  is_trading_day: boolean
  after_cutoff: boolean
  cutoff_time: string
  next_trading_day?: string
  message: string
  timezone: string
}

export interface MarketStatus {
  isTrading: boolean
  isTradingDay: boolean
  session: MarketSession
  currentTime: Date
  currentDate: string
  displayDate: string
  previousTradingDay: string
  lastTradingDay: string
  nextTradingDay: string
  nextSessionStart: Date | null
  nextTransitionAt: Date | null
  timeUntilNextSession: number
  timeUntilNextTransition: number
  timezone: string
  calendarSource: string
  coveredYears: number[]
}

export interface PricingDatePreview {
  tradeAt: Date | null
  tradeDate: string
  pricingDate: string
  rule: 'same_day_close' | 'next_trading_day'
  isTradingDay: boolean
  afterCutoff: boolean
  cutoffTime: Date | null
  nextTradingDay: string
  message: string
  timezone: string
}

let scheduledTransitionAt = ''
let scheduledTransitionTimer: number | null = null

const EMPTY_STATUS: MarketStatus = {
  isTrading: false,
  isTradingDay: false,
  session: 'pre_market',
  currentTime: new Date(0),
  currentDate: '',
  displayDate: '',
  previousTradingDay: '',
  lastTradingDay: '',
  nextTradingDay: '',
  nextSessionStart: null,
  nextTransitionAt: null,
  timeUntilNextSession: 0,
  timeUntilNextTransition: 0,
  timezone: 'Asia/Shanghai',
  calendarSource: '',
  coveredYears: [],
}

function parseDate(value?: string): Date | null {
  if (!value) {
    return null
  }

  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return null
  }

  return parsed
}

async function fetchJSON<T>(url: string): Promise<T> {
  const res = await fetch(url)
  const json = await res.json() as ApiEnvelope<T>

  if (!res.ok || !json.success || !json.data) {
    throw new Error(json.error?.message || 'Market request failed')
  }

  return json.data
}

function normalizeMarketStatus(raw: RawMarketStatus | undefined, nowMs: number): MarketStatus {
  if (!raw) {
    return EMPTY_STATUS
  }

  const currentTime = new Date(nowMs)
  const nextSessionStart = parseDate(raw.next_session_start)
  const nextTransitionAt = parseDate(raw.next_transition_at)

  return {
    isTrading: raw.is_trading,
    isTradingDay: raw.is_trading_day,
    session: raw.session,
    currentTime,
    currentDate: raw.current_date,
    displayDate: raw.display_date,
    previousTradingDay: raw.previous_trading_day,
    lastTradingDay: raw.last_trading_day,
    nextTradingDay: raw.next_trading_day,
    nextSessionStart,
    nextTransitionAt,
    timeUntilNextSession: nextSessionStart ? Math.max(0, nextSessionStart.getTime() - currentTime.getTime()) : 0,
    timeUntilNextTransition: nextTransitionAt ? Math.max(0, nextTransitionAt.getTime() - currentTime.getTime()) : 0,
    timezone: raw.timezone,
    calendarSource: raw.calendar_source,
    coveredYears: raw.covered_years ?? [],
  }
}

function normalizePricingDatePreview(raw: RawPricingDatePreview | undefined): PricingDatePreview | null {
  if (!raw) {
    return null
  }

  return {
    tradeAt: parseDate(raw.trade_at),
    tradeDate: raw.trade_date,
    pricingDate: raw.pricing_date,
    rule: raw.rule,
    isTradingDay: raw.is_trading_day,
    afterCutoff: raw.after_cutoff,
    cutoffTime: parseDate(raw.cutoff_time),
    nextTradingDay: raw.next_trading_day || '',
    message: raw.message,
    timezone: raw.timezone,
  }
}

function useMarketStatusQuery() {
  return useSWR<RawMarketStatus>(
    STATUS_KEY,
    fetchJSON,
    {
      revalidateOnFocus: false,
      refreshInterval: STATUS_FALLBACK_REFRESH_MS,
      dedupingInterval: 1000,
    }
  )
}

function scheduleStatusRefreshAtBoundary(nextTransitionAtRaw: string | undefined, delaySeconds: number | undefined) {
  if (typeof window === 'undefined' || !nextTransitionAtRaw || typeof delaySeconds !== 'number') {
    return
  }

  const nextTransitionAt = parseDate(nextTransitionAtRaw)
  if (!nextTransitionAt) {
    return
  }

  if (scheduledTransitionAt === nextTransitionAtRaw && scheduledTransitionTimer !== null) {
    return
  }

  if (scheduledTransitionTimer !== null) {
    window.clearTimeout(scheduledTransitionTimer)
    scheduledTransitionTimer = null
  }

  scheduledTransitionAt = nextTransitionAtRaw
  const timeoutMs = Math.max(250, delaySeconds * 1000 + 250)

  scheduledTransitionTimer = window.setTimeout(() => {
    scheduledTransitionAt = ''
    scheduledTransitionTimer = null
    void globalMutate(STATUS_KEY)
  }, timeoutMs)
}

export function useMarketStatus() {
  const { data, error, isLoading, mutate } = useMarketStatusQuery()
  const [nowMs, setNowMs] = useState(() => Date.now())

  useEffect(() => {
    const intervalId = window.setInterval(() => {
      setNowMs(Date.now())
    }, 1000)

    return () => window.clearInterval(intervalId)
  }, [])

  useEffect(() => {
    scheduleStatusRefreshAtBoundary(data?.next_transition_at, data?.time_until_next_transition_seconds)
  }, [data?.next_transition_at, data?.time_until_next_transition_seconds])

  const status = useMemo(() => normalizeMarketStatus(data, nowMs), [data, nowMs])
  const mounted = Boolean(data)

  return {
    ...status,
    mounted,
    isLoading: isLoading && !mounted,
    error,
    refresh: mutate,
  }
}

export function useMarketTradingState() {
  const { data, error, isLoading, mutate } = useMarketStatusQuery()
  useEffect(() => {
    scheduleStatusRefreshAtBoundary(data?.next_transition_at, data?.time_until_next_transition_seconds)
  }, [data?.next_transition_at, data?.time_until_next_transition_seconds])

  return {
    isTrading: data?.is_trading ?? false,
    isTradingDay: data?.is_trading_day ?? false,
    session: (data?.session ?? 'pre_market') as MarketSession,
    currentDate: data?.current_date ?? '',
    nextTradingDay: data?.next_trading_day ?? '',
    previousTradingDay: data?.previous_trading_day ?? '',
    timezone: data?.timezone ?? 'Asia/Shanghai',
    isLoading,
    error,
    refresh: mutate,
  }
}

export function usePricingDatePreview(tradeAt: string | null) {
  const key = tradeAt
    ? `${API_BASE_URL}/api/v1/market/pricing-date?trade_at=${encodeURIComponent(tradeAt)}`
    : null

  const { data, error, isLoading } = useSWR<RawPricingDatePreview>(
    key,
    fetchJSON,
    {
      revalidateOnFocus: false,
      keepPreviousData: true,
      dedupingInterval: 1000,
    }
  )

  return {
    preview: normalizePricingDatePreview(data),
    isLoading,
    isError: !!error,
    error,
  }
}

export function getSessionLabel(session: MarketSession): string {
  const labels: Record<MarketSession, string> = {
    pre_market: '盘前',
    morning: '上午盘',
    lunch_break: '午间休市',
    afternoon: '下午盘',
    after_hours: '已收盘',
    weekend: '周末休市',
    holiday: '节假日休市',
  }

  return labels[session]
}

export function formatTimeUntil(ms: number): string {
  if (ms <= 0) return '即将开始'

  const seconds = Math.floor(ms / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)

  if (days > 0) {
    return `${days}天${hours % 24}小时`
  }
  if (hours > 0) {
    return `${hours}小时${minutes % 60}分钟`
  }
  if (minutes > 0) {
    return `${minutes}分${seconds % 60}秒`
  }
  return `${seconds}秒`
}
