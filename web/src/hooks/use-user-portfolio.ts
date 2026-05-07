'use client'

import { useMemo } from 'react'
import useSWR from 'swr'
import { useMarketTradingState } from '@/hooks/use-market-status'
import type { Fund, FundEstimate } from '@/hooks/use-fund-data'

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || ''

export interface WatchlistFundEntry {
  fund_id: string
  created_at: string
  fund?: Fund
}

export interface WatchlistGroup {
  id: string
  name: string
  description: string
  accent: string
  sort_order: number
  created_at: string
  updated_at: string
  funds: WatchlistFundEntry[]
}

export interface HoldingEntry {
  id: string
  fund_id: string
  amount: string
  shares?: string
  confirmed_nav?: string
  confirmed_nav_date?: string
  trade_at?: string
  as_of_date: string
  actual_date?: string
  actual_nav?: string
  actual_daily_return?: string
  current_market_value?: string
  today_profit?: string
  today_change_percent?: string
  real_metrics_ready: boolean
  real_metrics_message?: string
  note: string
  created_at: string
  updated_at: string
  fund?: Fund
}

export interface HoldingSummary {
  total_principal: string
  ready_principal?: string
  total_current_market_value?: string
  total_today_profit?: string
  total_today_change_percent?: string
  metrics_scope?: 'none' | 'partial' | 'full'
  real_metrics_ready: boolean
  real_metrics_ready_count: number
  total_holdings: number
  incomplete_holdings_count?: number
  message?: string
}

export interface HoldingAggregateEntry {
  fund_id: string
  holding_count: number
  confirmed_holding_count: number
  real_metrics_ready_count: number
  incomplete_holdings_count: number
  total_principal: string
  confirmed_principal?: string
  ready_principal?: string
  confirmed_shares?: string
  official_current_market_value?: string
  official_today_profit?: string
  official_today_change_percent?: string
  metrics_scope?: 'none' | 'partial' | 'full'
  real_metrics_ready: boolean
  message?: string
  fund?: Fund
}

interface HoldingsResponse {
  items: HoldingEntry[]
  aggregates: HoldingAggregateEntry[]
  summary: HoldingSummary
}

export interface HoldingEstimateAggregateMetrics {
  fund_id: string
  estimate?: FundEstimate | null
  preview_ready: boolean
  fallback_ready: boolean
  preview_current_market_value?: string
  preview_today_profit?: string
  preview_today_change_percent?: string
  fallback_today_profit?: string
  confirmed_principal?: string
  confirmed_shares?: string
}

export interface HoldingEstimateSummary {
  metrics_scope: 'none' | 'partial' | 'full'
  ready_count: number
  total_count: number
  ready_principal: string
  total_current_market_value?: string
  total_today_profit?: string
  total_today_change_percent?: string
  message?: string
}

interface ApiEnvelope<T> {
  success: boolean
  data?: T
  error?: {
    code: string
    message: string
  }
}

async function fetcher<T>(url: string): Promise<T> {
  const res = await fetch(url, {
    credentials: 'include',
  })
  const json = await res.json() as ApiEnvelope<T>
  if (!res.ok || !json.success || !json.data) {
    throw new Error(json.error?.message || 'Failed to fetch user portfolio data')
  }
  return json.data
}

async function request<T>(path: string, init?: RequestInit): Promise<T | null> {
  const res = await fetch(`${API_BASE_URL}${path}`, {
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
    ...init,
  })

  const json = await res.json() as ApiEnvelope<T>
  if (!res.ok || !json.success) {
    throw new Error(json.error?.message || 'Request failed')
  }

  return json.data ?? null
}

function parseDecimal(value?: string) {
  if (!value) {
    return null
  }

  const parsed = Number.parseFloat(value)
  return Number.isNaN(parsed) ? null : parsed
}

async function fetchHoldingEstimates(fundIDs: string[]): Promise<Record<string, FundEstimate | null>> {
  const entries = await Promise.all(
    fundIDs.map(async (fundID) => {
      try {
        const estimate = await fetcher<FundEstimate>(`${API_BASE_URL}/api/v1/fund/${fundID}/estimate`)
        return [fundID, estimate] as const
      } catch {
        return [fundID, null] as const
      }
    })
  )

  return Object.fromEntries(entries)
}

export function useHoldingEstimateMetrics(aggregates: HoldingAggregateEntry[]) {
  const { isTrading } = useMarketTradingState()
  const fundIDs = useMemo(
    () => Array.from(new Set(aggregates.map((aggregate) => aggregate.fund_id).filter(Boolean))).sort(),
    [aggregates]
  )

  const { data: estimatesByFundID = {} } = useSWR<Record<string, FundEstimate | null>>(
    fundIDs.length > 0 ? ['user-holding-estimates', fundIDs.join(',')] : null,
    () => fetchHoldingEstimates(fundIDs),
    {
      revalidateOnFocus: isTrading,
      dedupingInterval: 5000,
      refreshInterval: isTrading ? 30000 : 0,
    }
  )

  const aggregateMetrics = useMemo(() => {
    return aggregates.reduce<Record<string, HoldingEstimateAggregateMetrics>>((result, aggregate) => {
      const estimate = estimatesByFundID[aggregate.fund_id] ?? null
      const confirmedShares = parseDecimal(aggregate.confirmed_shares)
      const confirmedPrincipal = parseDecimal(aggregate.confirmed_principal)
      const totalPrincipal = parseDecimal(aggregate.total_principal) ?? 0
      const estimateNav = parseDecimal(estimate?.estimate_nav)
      const prevNav = parseDecimal(estimate?.prev_nav)
      const changePercent = parseDecimal(estimate?.change_percent)

      const previewReady = typeof confirmedShares === 'number' &&
        confirmedShares > 0 &&
        typeof estimateNav === 'number' &&
        typeof prevNav === 'number' &&
        prevNav > 0

      const previewCurrentMarketValue = previewReady ? confirmedShares * estimateNav : null
      const previewTodayProfit = previewReady ? confirmedShares * (estimateNav - prevNav) : null
      const previewPreviousMarketValue = previewReady ? confirmedShares * prevNav : null
      const previewTodayChangePercent = previewReady && prevNav > 0
        ? ((previewTodayProfit ?? 0) / (previewPreviousMarketValue ?? 1)) * 100
        : changePercent

      const fallbackTodayProfit = !previewReady && typeof changePercent === 'number'
        ? totalPrincipal * changePercent / 100
        : null

      result[aggregate.fund_id] = {
        fund_id: aggregate.fund_id,
        estimate,
        preview_ready: previewReady,
        fallback_ready: fallbackTodayProfit !== null,
        preview_current_market_value: previewCurrentMarketValue?.toFixed(2),
        preview_today_profit: previewTodayProfit?.toFixed(2),
        preview_today_change_percent: typeof previewTodayChangePercent === 'number'
          ? previewTodayChangePercent.toFixed(4)
          : undefined,
        fallback_today_profit: fallbackTodayProfit?.toFixed(2),
        confirmed_principal: confirmedPrincipal?.toFixed(2),
        confirmed_shares: confirmedShares?.toFixed(6),
      }

      return result
    }, {})
  }, [aggregates, estimatesByFundID])

  const summary = useMemo<HoldingEstimateSummary>(() => {
    if (aggregates.length === 0) {
      return {
        metrics_scope: 'none',
        ready_count: 0,
        total_count: 0,
        ready_principal: '0.00',
      }
    }

    let readyCount = 0
    let readyPrincipal = 0
    let totalCurrentMarketValue = 0
    let totalTodayProfit = 0
    let totalPreviousMarketValue = 0

    for (const aggregate of aggregates) {
      const metrics = aggregateMetrics[aggregate.fund_id]
      const confirmedPrincipal = parseDecimal(aggregate.confirmed_principal) ?? 0
      if (!metrics?.preview_ready) {
        continue
      }

      readyCount++
      readyPrincipal += confirmedPrincipal
      totalCurrentMarketValue += parseDecimal(metrics.preview_current_market_value) ?? 0
      totalTodayProfit += parseDecimal(metrics.preview_today_profit) ?? 0

      const estimate = metrics.estimate
      const confirmedShares = parseDecimal(aggregate.confirmed_shares) ?? 0
      const prevNav = parseDecimal(estimate?.prev_nav) ?? 0
      totalPreviousMarketValue += confirmedShares * prevNav
    }

    const metricsScope = readyCount === 0
      ? 'none'
      : readyCount === aggregates.length
        ? 'full'
        : 'partial'

    return {
      metrics_scope: metricsScope,
      ready_count: readyCount,
      total_count: aggregates.length,
      ready_principal: readyPrincipal.toFixed(2),
      total_current_market_value: readyCount > 0 ? totalCurrentMarketValue.toFixed(2) : undefined,
      total_today_profit: readyCount > 0 ? totalTodayProfit.toFixed(2) : undefined,
      total_today_change_percent: readyCount > 0 && totalPreviousMarketValue > 0
        ? ((totalTodayProfit / totalPreviousMarketValue) * 100).toFixed(4)
        : undefined,
      message: readyCount === 0
        ? '待确认份额补齐后展示盘中预估总览。'
        : readyCount < aggregates.length
          ? `当前已按确认份额汇总 ${readyCount}/${aggregates.length} 只基金的盘中预估，剩余基金待补齐份额。`
          : '盘中预估总览已按全部已确认份额汇总。',
    }
  }, [aggregateMetrics, aggregates])

  return {
    estimatesByFundID,
    aggregateMetrics,
    summary,
  }
}

export function useUserPortfolio(userID: string | null) {
  const { data: watchlistGroups = [], mutate: mutateWatchlistGroups } = useSWR<WatchlistGroup[]>(
    userID ? `${API_BASE_URL}/api/v1/user/watchlist/groups` : null,
    fetcher,
    {
      revalidateOnFocus: false,
      dedupingInterval: 5000,
    }
  )

  const { data: holdingsPayload, mutate: mutateHoldings } = useSWR<HoldingsResponse>(
    userID ? `${API_BASE_URL}/api/v1/user/holdings` : null,
    fetcher,
    {
      revalidateOnFocus: false,
      dedupingInterval: 5000,
    }
  )

  const holdings = holdingsPayload?.items ?? []
  const holdingAggregates = holdingsPayload?.aggregates ?? []
  const holdingSummary = holdingsPayload?.summary ?? {
    total_principal: '0',
    ready_principal: '0',
    metrics_scope: 'none' as const,
    real_metrics_ready: false,
    real_metrics_ready_count: 0,
    total_holdings: 0,
    incomplete_holdings_count: 0,
    message: '',
  }

  return {
    watchlistGroups,
    holdings,
    holdingAggregates,
    holdingSummary,
    totalWatchlistFunds: watchlistGroups.reduce((sum, group) => sum + group.funds.length, 0),
    seedDemoData: async () => {
      if (!userID) {
        return
      }

      if (watchlistGroups.length === 0) {
        const coreGroup = await request<{ id: string }>('/api/v1/user/watchlist/groups', {
          method: 'POST',
          body: JSON.stringify({
            name: '核心观察',
            description: '长期跟踪的大盘与核心风格基金。',
          }),
        })

        const themeGroup = await request<{ id: string }>('/api/v1/user/watchlist/groups', {
          method: 'POST',
          body: JSON.stringify({
            name: '主题轮动',
            description: '更关注波段与赛道轮动机会。',
          }),
        })

        if (coreGroup?.id) {
          await request(`/api/v1/user/watchlist/groups/${coreGroup.id}/funds`, {
            method: 'POST',
            body: JSON.stringify({ fund_id: '005827' }),
          })
          await request(`/api/v1/user/watchlist/groups/${coreGroup.id}/funds`, {
            method: 'POST',
            body: JSON.stringify({ fund_id: '003095' }),
          })
        }

        if (themeGroup?.id) {
          await request(`/api/v1/user/watchlist/groups/${themeGroup.id}/funds`, {
            method: 'POST',
            body: JSON.stringify({ fund_id: '320007' }),
          })
        }
      }

      if (holdings.length === 0) {
        await request('/api/v1/user/holdings', {
          method: 'POST',
          body: JSON.stringify({
            fund_id: '005827',
            amount: '50000',
            trade_at: '2026-03-27T14:20:00+08:00',
            note: '长期底仓',
          }),
        })
        await request('/api/v1/user/holdings', {
          method: 'POST',
          body: JSON.stringify({
            fund_id: '003095',
            amount: '28000',
            trade_at: '2026-03-30T15:18:00+08:00',
            note: '医药主题仓位',
          }),
        })
      }

      await Promise.all([mutateWatchlistGroups(), mutateHoldings()])
    },
    createGroup: async (name: string, description: string) => {
      if (!userID) return
      await request('/api/v1/user/watchlist/groups', {
        method: 'POST',
        body: JSON.stringify({ name, description }),
      })
      await mutateWatchlistGroups()
    },
    updateGroup: async (groupID: string, name: string, description: string, accent: string) => {
      if (!userID) return
      await request(`/api/v1/user/watchlist/groups/${groupID}`, {
        method: 'PUT',
        body: JSON.stringify({ name, description, accent }),
      })
      await mutateWatchlistGroups()
    },
    reorderGroups: async (groupIDs: string[]) => {
      if (!userID) return
      await request('/api/v1/user/watchlist/groups/reorder', {
        method: 'PUT',
        body: JSON.stringify({ group_ids: groupIDs }),
      })
      await mutateWatchlistGroups()
    },
    deleteGroup: async (groupID: string) => {
      if (!userID) return
      await request(`/api/v1/user/watchlist/groups/${groupID}`, {
        method: 'DELETE',
      })
      await mutateWatchlistGroups()
    },
    addFundToGroup: async (groupID: string, fundID: string) => {
      if (!userID) return
      await request(`/api/v1/user/watchlist/groups/${groupID}/funds`, {
        method: 'POST',
        body: JSON.stringify({ fund_id: fundID }),
      })
      await mutateWatchlistGroups()
    },
    removeFundFromGroup: async (groupID: string, fundID: string) => {
      if (!userID) return
      await request(`/api/v1/user/watchlist/groups/${groupID}/funds/${fundID}`, {
        method: 'DELETE',
      })
      await mutateWatchlistGroups()
    },
    addHolding: async (fundID: string, amount: string, tradeAt: string, note: string) => {
      if (!userID) return
      const normalizedFundID = fundID.trim()
      if (!normalizedFundID) {
        throw new Error('请先从搜索结果中选择基金')
      }

      await request('/api/v1/user/holdings', {
        method: 'POST',
        body: JSON.stringify({
          fund_id: normalizedFundID,
          amount,
          trade_at: tradeAt,
          note,
        }),
      })
      await mutateHoldings()
    },
    removeHolding: async (holdingID: string) => {
      if (!userID) return
      await request(`/api/v1/user/holdings/${holdingID}`, {
        method: 'DELETE',
      })
      await mutateHoldings()
    },
  }
}
