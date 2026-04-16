'use client'

import useSWR from 'swr'
import type { Fund } from '@/hooks/use-fund-data'

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
  total_current_market_value?: string
  total_today_profit?: string
  total_today_change_percent?: string
  real_metrics_ready: boolean
  real_metrics_ready_count: number
  total_holdings: number
  message?: string
}

interface HoldingsResponse {
  items: HoldingEntry[]
  summary: HoldingSummary
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
  const holdingSummary = holdingsPayload?.summary ?? {
    total_principal: '0',
    real_metrics_ready: false,
    real_metrics_ready_count: 0,
    total_holdings: 0,
    message: '',
  }

  return {
    watchlistGroups,
    holdings,
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
