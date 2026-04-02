'use client'

import useSWR, { SWRConfiguration } from 'swr'
import { useMarketTradingState } from './use-market-status'

// API 基础 URL
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || ''

// 通用 fetcher
async function fetcher<T>(url: string): Promise<T> {
    const res = await fetch(url)
    if (!res.ok) {
        throw new Error(`API error: ${res.status}`)
    }
    const json = await res.json()
    if (!json.success) {
        throw new Error(json.error?.message || 'Unknown error')
    }
    return json.data
}

// Types
export interface Fund {
    id: string
    name: string
    type: string
    manager: string
    company: string
    nav: string
    scale: string
    updated_at: string
}

export interface HoldingDetail {
    stock_code: string
    stock_name: string
    holding_ratio: string
    stock_change: string
    contribution: string
    current_price: string
    prev_close: string
}

export interface FundEstimate {
    fund_id: string
    fund_name: string
    estimate_nav: string
    prev_nav: string
    change_percent: string
    change_amount: string
    total_hold_ratio: string
    holding_details: HoldingDetail[]
    calculated_at: string
    data_source: string
}

export interface TimeSeriesPoint {
    timestamp: string
    change_percent: string
    estimate_nav: string
}

// 默认 SWR 配置
const DEFAULT_TRADING_INTERVAL = 10000  // 交易时段 10秒刷新
const DEFAULT_CLOSED_INTERVAL = 0       // 休市时不刷新 (0 = disabled)

/**
 * useFundEstimate - 获取基金实时估值
 * 
 * 根据市场状态智能调整刷新频率:
 * - 交易时段: 每 10 秒自动刷新
 * - 休市时段: 仅获取一次，不自动刷新
 */
export function useFundEstimate(fundId: string | null, options?: SWRConfiguration) {
    const { isTrading } = useMarketTradingState()

    // 根据交易状态动态设置刷新间隔
    const refreshInterval = isTrading ? DEFAULT_TRADING_INTERVAL : DEFAULT_CLOSED_INTERVAL

    const swrKey = fundId ? `${API_BASE_URL}/api/v1/fund/${fundId}/estimate` : null

    const { data, error, isLoading, isValidating, mutate } = useSWR<FundEstimate>(
        swrKey,
        fetcher,
        {
            refreshInterval,
            revalidateOnFocus: isTrading, // 仅交易时段在 focus 时刷新
            revalidateOnReconnect: isTrading,
            keepPreviousData: true, // 🔑 关键: 保持旧数据，避免 UI 闪烁
            dedupingInterval: 5000, // 5秒内相同请求去重
            errorRetryCount: 3,
            ...options,
        }
    )

    return {
        estimate: data,
        isLoading,
        isValidating, // 后台刷新中
        isError: !!error,
        error,
        mutate, // 手动刷新
        isTrading,
        refreshInterval,
    }
}

/**
 * useFund - 获取基金基本信息
 */
export function useFund(fundId: string | null) {
    const swrKey = fundId ? `${API_BASE_URL}/api/v1/fund/${fundId}` : null

    const { data, error, isLoading } = useSWR<Fund>(
        swrKey,
        fetcher,
        {
            revalidateOnFocus: false,
            dedupingInterval: 60000, // 基金信息1分钟缓存
        }
    )

    return {
        fund: data,
        isLoading,
        isError: !!error,
        error,
    }
}

/**
 * useFundHoldings - 获取基金持仓
 */
export function useFundHoldings(fundId: string | null) {
    const swrKey = fundId ? `${API_BASE_URL}/api/v1/fund/${fundId}/holdings` : null

    const { data, error, isLoading } = useSWR<{ fund: Fund; holdings: HoldingDetail[] }>(
        swrKey,
        fetcher,
        {
            revalidateOnFocus: false,
            dedupingInterval: 60000,
        }
    )

    return {
        fund: data?.fund,
        holdings: data?.holdings || [],
        isLoading,
        isError: !!error,
    }
}

/**
 * Time Series API Response with market context
 */
export interface TimeSeriesResponse {
    points: TimeSeriesPoint[]
    display_date: string
    is_trading: boolean
    is_historical: boolean
    session: 'pre_market' | 'morning' | 'lunch_break' | 'afternoon' | 'after_hours' | 'weekend' | 'holiday'
    last_trading_day: string
}

/**
 * useTimeSeries - 获取分时数据
 * 
 * Now returns additional context:
 * - displayDate: The date of the data being shown
 * - isHistorical: Whether showing previous trading day data
 * - session: Current market session
 */
export function useTimeSeries(fundId: string | null) {
    const { isTrading } = useMarketTradingState()

    const refreshInterval = isTrading ? 30000 : 0 // 交易时段30秒刷新

    const swrKey = fundId ? `${API_BASE_URL}/api/v1/fund/${fundId}/timeseries` : null

    const { data, error, isLoading } = useSWR<TimeSeriesResponse>(
        swrKey,
        fetcher,
        {
            refreshInterval,
            keepPreviousData: true,
            revalidateOnFocus: false,
        }
    )

    return {
        timeSeries: data?.points || [],
        displayDate: data?.display_date || '',
        isHistorical: data?.is_historical || false,
        session: data?.session || 'after_hours',
        lastTradingDay: data?.last_trading_day || '',
        isLoading,
        isError: !!error,
        isTrading,
    }
}

/**
 * useFundSearch - 搜索基金
 */
export function useFundSearch(query: string) {
    // 仅当 query 长度 >= 2 时搜索
    const shouldSearch = query.trim().length >= 2
    const swrKey = shouldSearch
        ? `${API_BASE_URL}/api/v1/fund/search?q=${encodeURIComponent(query)}`
        : null

    const { data, error, isLoading } = useSWR<Fund[]>(
        swrKey,
        fetcher,
        {
            revalidateOnFocus: false,
            dedupingInterval: 1000,
        }
    )

    return {
        results: data || [],
        isLoading: shouldSearch && isLoading,
        isError: !!error,
    }
}
