'use client'

import { useMemo } from 'react'
import useSWR, { mutate as mutatePortfolioCache } from 'swr'
import { useMarketTradingState } from '@/hooks/use-market-status'
import type { Fund, FundEstimate } from '@/hooks/use-fund-data'
import type { HoldingSourceFilter } from '@/lib/holding-sources'
import { API_BASE_URL } from '@/lib/api-base-url'


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
	entry_type?: 'buy' | 'opening'
	real_metrics_stale?: boolean
  id: string
  fund_id: string
  amount: string
  shares?: string
  confirmed_nav?: string
  confirmed_nav_date?: string
  manual_confirmation?: boolean
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
  source_platform?: string
  source_label?: string
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

export type HoldingTransactionType = 'buy' | 'opening' | 'correction' | 'delete' | 'sell' | 'dividend' | 'adjustment'
export type HoldingTransactionStatusFilter = 'all' | 'active' | 'voided'

export interface HoldingTransactionFilters {
  fundID?: string
  type?: HoldingTransactionType | 'all'
  status?: HoldingTransactionStatusFilter
  sourcePlatform?: HoldingSourceFilter
  keyword?: string
  startDate?: string
  endDate?: string
  offset?: number
  limit?: number
}

export interface HoldingTransactionEntry {
  id: string
  user_id: string
  holding_id?: string
  fund_id: string
  type: HoldingTransactionType
  amount: string
  shares?: string
  confirmed_nav?: string
  confirmed_nav_date?: string
  manual_confirmation?: boolean
  trade_at?: string
  as_of_date?: string
  note?: string
  source_platform?: string
  source_label?: string
  metadata?: Record<string, string>
  voided?: boolean
  voided_at?: string
  void_reason?: string
  created_at: string
  fund?: Fund
}

export interface HoldingTransactionRollbackField {
  field: string
  label: string
  current_value?: string
  rollback_value?: string
  delta?: string
  direction?: string
}

export interface HoldingTransactionRollbackPreview {
  transaction: HoldingTransactionEntry
  current_holding?: HoldingEntry
  preview_only: boolean
  can_apply_automatically: boolean
  state: string
  title: string
  summary: string
  suggested_action: string
  affected_fields: HoldingTransactionRollbackField[]
  warnings?: string[]
}

export interface HoldingTransactionRollbackApplyResult {
  transaction: HoldingTransactionEntry
  current_holding?: HoldingEntry
  preview: HoldingTransactionRollbackPreview
  applied: boolean
  holding_removed?: boolean
  holding_restored?: boolean
  message: string
}

export interface HoldingTransactionDetail {
  transaction: HoldingTransactionEntry
  current_holding?: HoldingEntry
  rollback_preview?: HoldingTransactionRollbackPreview
  related_transactions?: HoldingTransactionEntry[]
  subsequent_transactions?: HoldingTransactionEntry[]
  impact_chain?: string[]
}

interface HoldingsResponse {
  items: HoldingEntry[]
  aggregates: HoldingAggregateEntry[]
  summary: HoldingSummary
}

export interface CreateHoldingBatchItem {
	entry_type?: 'buy' | 'opening'
	shares?: string
	market_value?: string
	holding_profit?: string
	snapshot_date?: string
  fund_id: string
  amount?: string
  trade_at?: string
  as_of_date?: string
  note?: string
  source_platform?: string
  source_label?: string
}

export interface HoldingBatchCreateFailure {
  index: number
  fund_id?: string
  code: string
  message: string
}

export interface HoldingBatchCreateResult {
  total: number
  created_count: number
  failed_count: number
  created: HoldingEntry[]
  failed?: HoldingBatchCreateFailure[]
}

export interface UpdateHoldingPayload {
	snapshot_date?: string
  amount: string
  shares?: string
  confirmed_nav?: string
  confirmed_nav_date?: string
  trade_at?: string
  note?: string
  source_platform?: string
  source_label?: string
}

export interface SellHoldingPayload {
  amount?: string
  shares?: string
  trade_at?: string
  note?: string
  sell_all?: boolean
}

export interface DividendHoldingPayload {
  amount: string
  shares?: string
  trade_at?: string
  note?: string
  reinvest?: boolean
  source_platform?: string
  source_label?: string
}

export interface AdjustHoldingSharesPayload {
  shares_delta?: string
  target_shares?: string
  confirmed_nav?: string
  confirmed_nav_date?: string
  trade_at?: string
  note?: string
  source_platform?: string
  source_label?: string
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
  const json = (await res.json()) as ApiEnvelope<T>
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

  const json = (await res.json()) as ApiEnvelope<T>
  if (!res.ok || !json.success) {
    throw new Error(json.error?.message || 'Request failed')
  }

  return json.data ?? null
}

export async function deleteHoldingTransaction(transactionID: string) {
  const prefix = `${API_BASE_URL}/api/v1/user/holdings/transactions`
  const detailKey = `${prefix}/${encodeURIComponent(transactionID)}`
  await request(`/api/v1/user/holdings/transactions/${encodeURIComponent(transactionID)}`, { method: 'DELETE' })
  // Remove the deleted entry immediately from every cached filter/page and invalidate detail previews.
  const transactionKey = (key: unknown) => typeof key === 'string' && (key === prefix || key.startsWith(prefix + '?') || key.startsWith(prefix + '/'))
  await mutatePortfolioCache(transactionKey, (cached: unknown) => Array.isArray(cached) ? (cached as HoldingTransactionEntry[]).filter(item => item.id !== transactionID) : undefined, { revalidate: false })
  // A failed refresh must not report a successful deletion as a failed delete.
  void mutatePortfolioCache(key => transactionKey(key) && key !== detailKey).catch(() => undefined)
}

function parseDecimal(value?: string) {
  if (!value) {
    return null
  }

  const parsed = Number.parseFloat(value)
  return Number.isFinite(parsed) ? parsed : null
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
    }),
  )

  return Object.fromEntries(entries)
}

function buildHoldingTransactionsPath(filters: HoldingTransactionFilters = {}) {
  const params = new URLSearchParams()
  params.set('limit', String(filters.limit ?? 12))

  if (filters.fundID && filters.fundID !== 'all') {
    params.set('fund_id', filters.fundID)
  }
  if (filters.type && filters.type !== 'all') {
    params.set('type', filters.type)
  }
  if (filters.status === 'active') {
    params.set('voided', 'false')
  } else if (filters.status === 'voided') {
    params.set('voided', 'true')
  }
  if (filters.sourcePlatform && filters.sourcePlatform !== 'all') {
    params.set('source_platform', filters.sourcePlatform)
  }
  if (filters.keyword?.trim()) {
    params.set('keyword', filters.keyword.trim())
  }
  if (filters.startDate) {
    params.set('start_date', filters.startDate)
  }
  if (filters.endDate) {
    params.set('end_date', filters.endDate)
  }
  if (typeof filters.offset === 'number' && filters.offset > 0) {
    params.set('offset', String(filters.offset))
  }

  return `${API_BASE_URL}/api/v1/user/holdings/transactions?${params.toString()}`
}

export function useHoldingEstimateMetrics(aggregates: HoldingAggregateEntry[]) {
  const { isTrading } = useMarketTradingState()
  const fundIDs = useMemo(
    () => Array.from(new Set(aggregates.map((aggregate) => aggregate.fund_id).filter(Boolean))).sort(),
    [aggregates],
  )

  const { data: estimatesByFundID = {} } = useSWR<Record<string, FundEstimate | null>>(
    fundIDs.length > 0 ? ['user-holding-estimates', fundIDs.join(',')] : null,
    () => fetchHoldingEstimates(fundIDs),
    {
      revalidateOnFocus: isTrading,
      dedupingInterval: 5000,
      refreshInterval: isTrading ? 30000 : 0,
    },
  )

  const aggregateMetrics = useMemo(() => {
    return aggregates.reduce<Record<string, HoldingEstimateAggregateMetrics>>((result, aggregate) => {
      const estimate = estimatesByFundID[aggregate.fund_id] ?? null
      const confirmedShares = parseDecimal(aggregate.confirmed_shares)
      const confirmedPrincipal = parseDecimal(aggregate.confirmed_principal)
      const estimateNav = parseDecimal(estimate?.estimate_nav)
      const prevNav = parseDecimal(estimate?.prev_nav)

      const previewReady =
        typeof confirmedShares === 'number' &&
        confirmedShares > 0 &&
        typeof confirmedPrincipal === 'number' &&
        typeof estimateNav === 'number' && estimateNav > 0

      const previewCurrentMarketValue = previewReady ? confirmedShares * estimateNav : null
      const previewTodayProfit = previewReady && prevNav !== null && prevNav > 0 ? confirmedShares * (estimateNav - prevNav) : null
      const previewPreviousMarketValue = previewReady && prevNav !== null && prevNav > 0 ? confirmedShares * prevNav : null
      const previewTodayChangePercent =
        previewTodayProfit !== null && previewPreviousMarketValue !== null && previewPreviousMarketValue > 0
          ? ((previewTodayProfit ?? 0) / (previewPreviousMarketValue ?? 1)) * 100
          : null


      result[aggregate.fund_id] = {
        fund_id: aggregate.fund_id,
        estimate,
        preview_ready: previewReady,
        fallback_ready: false,
        preview_current_market_value: previewCurrentMarketValue?.toFixed(2),
        preview_today_profit: previewTodayProfit?.toFixed(2),
        preview_today_change_percent:
          typeof previewTodayChangePercent === 'number' ? previewTodayChangePercent.toFixed(4) : undefined,
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
    let dailyReadyCount = 0
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
      const dailyProfit = parseDecimal(metrics.preview_today_profit)
      if (dailyProfit !== null) {
        totalTodayProfit += dailyProfit
        dailyReadyCount++
      }

      const estimate = metrics.estimate
      const confirmedShares = parseDecimal(aggregate.confirmed_shares) ?? 0
      const prevNav = parseDecimal(estimate?.prev_nav) ?? 0
      totalPreviousMarketValue += confirmedShares * prevNav
    }

    const fullyCovered = readyCount === aggregates.length && aggregates.every(item => item.confirmed_holding_count === item.holding_count)
    const metricsScope = readyCount === 0 ? 'none' : fullyCovered ? 'full' : 'partial'

    return {
      metrics_scope: metricsScope,
      ready_count: readyCount,
      total_count: aggregates.length,
      ready_principal: readyPrincipal.toFixed(2),
      total_current_market_value: readyCount > 0 ? totalCurrentMarketValue.toFixed(2) : undefined,
      total_today_profit: dailyReadyCount > 0 ? totalTodayProfit.toFixed(2) : undefined,
      total_today_change_percent:
        readyCount > 0 && totalPreviousMarketValue > 0
          ? ((totalTodayProfit / totalPreviousMarketValue) * 100).toFixed(4)
          : undefined,
      message:
        readyCount === 0
          ? '待确认份额补齐后展示盘中预估总览。'
          : !fullyCovered
            ? `当前已按确认份额汇总 ${readyCount}/${aggregates.length} 只基金的盘中预估，仍有分笔待补齐份额。`
            : '盘中预估总览已按全部已确认份额汇总。',
    }
  }, [aggregateMetrics, aggregates])

  return {
    estimatesByFundID,
    aggregateMetrics,
    summary,
  }
}

export function useUserWatchlistGroups(userID: string | null) {
  const { data: watchlistGroups = [], isLoading: isWatchlistLoading, mutate: mutateWatchlistGroups } = useSWR<WatchlistGroup[]>(
    userID ? `${API_BASE_URL}/api/v1/user/watchlist/groups` : null,
    fetcher,
    {
      revalidateOnFocus: false,
      dedupingInterval: 5000,
    },
  )

  return {
    watchlistGroups,
    isWatchlistLoading,
    mutateWatchlistGroups,
    totalWatchlistFunds: watchlistGroups.reduce((sum, group) => sum + group.funds.length, 0),
  }
}

export function useUserPortfolio(userID: string | null, transactionFilters: HoldingTransactionFilters = {}) {
  const {
    fundID: transactionFundID,
    type: transactionType,
    status: transactionStatus,
    sourcePlatform: transactionSourcePlatform,
    keyword: transactionKeyword,
    startDate: transactionStartDate,
    endDate: transactionEndDate,
    offset: transactionOffset,
    limit: transactionLimit,
  } = transactionFilters
  const holdingTransactionsPath = userID
    ? buildHoldingTransactionsPath({
        fundID: transactionFundID,
        type: transactionType,
        status: transactionStatus,
        sourcePlatform: transactionSourcePlatform,
        keyword: transactionKeyword,
        startDate: transactionStartDate,
        endDate: transactionEndDate,
        offset: transactionOffset,
        limit: transactionLimit,
      })
    : null

  const { watchlistGroups, isWatchlistLoading, mutateWatchlistGroups } = useUserWatchlistGroups(userID)

  const { data: holdingsPayload, isLoading: isHoldingsLoading, error: holdingsError, mutate: mutateHoldings } = useSWR<HoldingsResponse>(
    userID ? `${API_BASE_URL}/api/v1/user/holdings` : null,
    fetcher,
    {
      revalidateOnFocus: false,
      dedupingInterval: 3000,
      refreshInterval: payload => payload?.items?.some(item => !item.real_metrics_ready || item.real_metrics_stale) ? 5000 : 60000,
    },
  )

  const { data: holdingTransactions = [], mutate: mutateHoldingTransactions } = useSWR<HoldingTransactionEntry[]>(
    holdingTransactionsPath,
    fetcher,
    {
      revalidateOnFocus: false,
      dedupingInterval: 5000,
    },
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
    isWatchlistLoading,
    holdings,
    isHoldingsLoading,
    holdingsError: holdingsError as Error | undefined,
    refreshHoldings: mutateHoldings,
    holdingAggregates,
    holdingTransactions,
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

      await Promise.all([mutateWatchlistGroups(), mutateHoldings(), mutateHoldingTransactions()])
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
      await Promise.allSettled([mutateHoldings(), mutateHoldingTransactions()])
    },
    addOpeningHolding: async (payload: CreateHoldingBatchItem) => {
      if (!userID) throw new Error('请先登录。')
      const result = await request<HoldingEntry>('/api/v1/user/holdings', { method: 'POST', body: JSON.stringify({ ...payload, entry_type: 'opening' }) })
      await Promise.allSettled([mutateHoldings(), mutateHoldingTransactions()])
      return result
    },
    updateHolding: async (holdingID: string, payload: UpdateHoldingPayload) => {
      if (!userID) return
      await request(`/api/v1/user/holdings/${holdingID}`, {
        method: 'PUT',
        body: JSON.stringify(payload),
      })
      await Promise.all([mutateHoldings(), mutateHoldingTransactions()])
    },
    sellHolding: async (holdingID: string, payload: SellHoldingPayload) => {
      if (!userID) return
      await request(`/api/v1/user/holdings/${holdingID}/sell`, {
        method: 'POST',
        body: JSON.stringify(payload),
      })
      await Promise.all([mutateHoldings(), mutateHoldingTransactions()])
    },
    recordHoldingDividend: async (holdingID: string, payload: DividendHoldingPayload) => {
      if (!userID) return
      await request(`/api/v1/user/holdings/${holdingID}/dividend`, {
        method: 'POST',
        body: JSON.stringify(payload),
      })
      await Promise.all([mutateHoldings(), mutateHoldingTransactions()])
    },
    adjustHoldingShares: async (holdingID: string, payload: AdjustHoldingSharesPayload) => {
      if (!userID) return
      await request(`/api/v1/user/holdings/${holdingID}/adjustment`, {
        method: 'POST',
        body: JSON.stringify(payload),
      })
      await Promise.all([mutateHoldings(), mutateHoldingTransactions()])
    },
    removeHolding: async (holdingID: string) => {
      if (!userID) return
      await request(`/api/v1/user/holdings/${holdingID}`, {
        method: 'DELETE',
      })
      await Promise.all([mutateHoldings(), mutateHoldingTransactions()])
    },
    voidHoldingTransaction: async (transactionID: string, reason: string) => {
      if (!userID) return
      await request(`/api/v1/user/holdings/transactions/${transactionID}/void`, {
        method: 'POST',
        body: JSON.stringify({ reason }),
      })
      await mutateHoldingTransactions()
    },
    deleteHoldingTransaction: async (transactionID: string) => {
      if (!userID) throw new Error('请先登录后再删除流水。')
      await deleteHoldingTransaction(transactionID)
    },
    previewHoldingTransactionRollback: async (transactionID: string) => {
      if (!userID) return null
      return request<HoldingTransactionRollbackPreview>(
        `/api/v1/user/holdings/transactions/${transactionID}/rollback-preview`,
      )
    },
    applyHoldingTransactionRollback: async (transactionID: string, reason: string) => {
      if (!userID) return null
      const result = await request<HoldingTransactionRollbackApplyResult>(
        `/api/v1/user/holdings/transactions/${transactionID}/rollback-apply`,
        {
          method: 'POST',
          body: JSON.stringify({ reason }),
        },
      )
      await Promise.all([mutateHoldings(), mutateHoldingTransactions()])
      return result
    },
    addHoldingsBatch: async (items: CreateHoldingBatchItem[]) => {
      if (!userID) return null
      const result = await request<HoldingBatchCreateResult>('/api/v1/user/holdings/batch', {
        method: 'POST',
        body: JSON.stringify({ items }),
      })
      await Promise.allSettled([mutateHoldings(), mutateHoldingTransactions()])
      return result
    },
    getHoldingTransactionDetail: async (transactionID: string) => {
      if (!userID) return null
      return request<HoldingTransactionDetail>(`/api/v1/user/holdings/transactions/${transactionID}`)
    },
  }
}

export function useHoldingTransactionDetail(userID: string | null, transactionID: string | null) {
  const { data, error, isLoading, mutate } = useSWR<HoldingTransactionDetail>(
    userID && transactionID
      ? `${API_BASE_URL}/api/v1/user/holdings/transactions/${transactionID}`
      : null,
    fetcher,
    {
      revalidateOnFocus: false,
      dedupingInterval: 5000,
    },
  )

  return {
    detail: data ?? null,
    error,
    isLoading,
    refresh: mutate,
  }
}
