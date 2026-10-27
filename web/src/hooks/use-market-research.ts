'use client'

import useSWR from 'swr'

export type ResearchSourceMeta = { source: string; fetched_at: string; stale: boolean; error?: string }
export type ResearchInstrument = { symbol: string; name: string; group: string }
export type ResearchQuote = ResearchInstrument & {
  price: number
  prev_close: number
  open: number
  high: number
  low: number
  change_pct: number
  timestamp: string
  delayed: boolean
}
export type ResearchBar = { date: string; open: number; close: number; high: number; low: number; volume: number }
export type ResearchOverview = {
  generated_at: string
  instruments: ResearchInstrument[]
  quotes: ResearchQuote[]
  quote_meta: ResearchSourceMeta
  breadth: { advancing: number; declining: number; unchanged: number; turnover_yuan: number } | null
  breadth_meta: ResearchSourceMeta
  news: { title: string; url: string; date: string; related: boolean }[]
  news_meta: ResearchSourceMeta
  market_phase: string
}
export type ResearchInstrumentDetail = {
  instrument: ResearchInstrument
  quote: ResearchQuote | null
  quote_meta: ResearchSourceMeta
  bars: ResearchBar[]
  history_meta: ResearchSourceMeta
  technical: {
    ma5: number | null
    ma20: number | null
    ma60: number | null
    rsi14: number | null
    previous_low: number | null
    provisional: boolean
    conditions: { label: string; detail: string; tone: string }[]
  } | null
  minute: { date: string; points: { time: string; price: number }[] } | null
  minute_meta: ResearchSourceMeta
}
export type ResearchFlows = {
  rows: { code: string; name: string; change_pct: number; net_inflow_yuan: number; timestamp: string }[]
  meta: ResearchSourceMeta
  method: string
}
export type ResearchRule = 'ma20_up' | 'ma20_down' | 'rsi_recovery' | 'previous_low_break'
export type ResearchStudy = {
  symbol: string
  rule: ResearchRule
  direction: string
  holding: number
  fee_bps: number
  sample_count: number
  pending_count: number
  direction_win_rate: number | null
  mean_return_pct: number | null
  mean_net_return_pct: number | null
  worst_drawdown_pct: number | null
  baseline_mean_return_pct: number | null
  events: {
    signal_date: string
    entry_date: string
    exit_date: string
    return_pct: number
    direction_pct: number
    net_return_pct: number | null
    drawdown_pct: number
  }[]
  last_completed_date: string
  method: string
  history_meta: ResearchSourceMeta
}

const ROOT = '/api/v1/market/research'
const ACTIVE_PHASES = new Set(['morning', 'afternoon', 'call_auction', 'trading'])

export function researchRefreshInterval(phase?: string) {
  return ACTIVE_PHASES.has(phase || '') ? 30_000 : 300_000
}

async function fetchResearch<T>(url: string): Promise<T> {
  const response = await fetch(url, { signal: AbortSignal.timeout(25_000) })
  let envelope: { success: boolean; data?: T; error?: { message?: string } }
  try {
    envelope = await response.json()
  } catch {
    throw new Error('数据服务返回异常，请稍后重试')
  }
  if (!response.ok || !envelope.success || envelope.data === undefined) {
    throw new Error(envelope.error?.message || `数据请求失败（${response.status}）`)
  }
  return envelope.data
}

const sharedOptions = {
  dedupingInterval: 15_000,
  errorRetryCount: 2,
  errorRetryInterval: 15_000,
  refreshWhenHidden: false,
  refreshWhenOffline: false,
  revalidateOnFocus: true,
}

export function useMarketResearchOverview() {
  return useSWR<ResearchOverview, Error>(`${ROOT}/overview`, fetchResearch<ResearchOverview>, {
    ...sharedOptions,
    refreshInterval: (data) => researchRefreshInterval(data?.market_phase),
  })
}

export function useMarketResearchInstrument(symbol: string | null, phase?: string) {
  return useSWR<ResearchInstrumentDetail, Error>(
    symbol ? `${ROOT}/instrument?${new URLSearchParams({ symbol })}` : null,
    fetchResearch<ResearchInstrumentDetail>,
    { ...sharedOptions, refreshInterval: researchRefreshInterval(phase) },
  )
}

export function useMarketResearchStudy(parameters: { symbol: string; rule: ResearchRule; holding: number; fee_bps: number } | null) {
  return useSWR<ResearchStudy, Error>(
    parameters ? `${ROOT}/study?${new URLSearchParams({
      symbol: parameters.symbol,
      rule: parameters.rule,
      holding: String(parameters.holding),
      fee_bps: String(parameters.fee_bps),
    })}` : null,
    fetchResearch<ResearchStudy>,
    { ...sharedOptions, refreshInterval: 0, revalidateOnFocus: false },
  )
}

export function useMarketResearchFlows(phase?: string) {
  return useSWR<ResearchFlows, Error>(`${ROOT}/flows`, fetchResearch<ResearchFlows>, {
    ...sharedOptions, refreshInterval: researchRefreshInterval(phase),
  })
}
