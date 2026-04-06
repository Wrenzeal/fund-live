'use client'

import { useEffect, useEffectEvent } from 'react'
import useSWR, { SWRConfiguration } from 'swr'
import { useMarketTradingState } from './use-market-status'

// API 基础 URL
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || ''

export interface ResponseMeta {
    data_source?: string
    cache_status?: string
}

interface ApiEnvelope<T> {
    success: boolean
    data?: T
    error?: {
        code?: string
        message?: string
    }
    meta?: ResponseMeta
}

export class ApiRequestError extends Error {
    code: string
    status: number
    retryAfterSeconds: number
    meta?: ResponseMeta

    constructor(message: string, options: {
        code?: string
        status: number
        retryAfterSeconds?: number
        meta?: ResponseMeta
    }) {
        super(message)
        this.name = 'ApiRequestError'
        this.code = options.code || 'UNKNOWN_ERROR'
        this.status = options.status
        this.retryAfterSeconds = options.retryAfterSeconds ?? 0
        this.meta = options.meta
    }
}

function parseRetryAfter(res: Response): number {
    const raw = res.headers.get('Retry-After')
    if (!raw) {
        return 0
    }

    const parsed = Number.parseInt(raw, 10)
    if (!Number.isFinite(parsed) || parsed <= 0) {
        return 0
    }
    return parsed
}

async function fetchEnvelope<T>(url: string): Promise<{ data: T; meta?: ResponseMeta }> {
    const res = await fetch(url)

    let json: ApiEnvelope<T> | null = null
    try {
        json = await res.json() as ApiEnvelope<T>
    } catch {
        json = null
    }

    if (!res.ok || !json?.success || typeof json.data === 'undefined') {
        throw new ApiRequestError(
            json?.error?.message || `API error: ${res.status}`,
            {
                code: json?.error?.code,
                status: res.status,
                retryAfterSeconds: parseRetryAfter(res),
                meta: json?.meta,
            }
        )
    }

    return {
        data: json.data,
        meta: json.meta,
    }
}

function getRetryDelayMs(error: unknown, fallbackMs: number) {
    if (error instanceof ApiRequestError && error.retryAfterSeconds > 0) {
        return error.retryAfterSeconds * 1000
    }
    return fallbackMs
}

function scheduleRetry(
    error: unknown,
    retryCount: number,
    revalidate: (options: { retryCount: number }) => void,
    maxRetryCount: number,
    fallbackMs: number
) {
    if (retryCount >= maxRetryCount) {
        return
    }

    const delayMs = getRetryDelayMs(error, fallbackMs)
    window.setTimeout(() => {
        revalidate({ retryCount: retryCount + 1 })
    }, delayMs)
}

export function isFundDataWarmingError(error: unknown): error is ApiRequestError {
    return error instanceof ApiRequestError && error.code === 'FUND_DATA_WARMING'
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
    const {
        onSuccess,
        onError,
        ...restOptions
    } = options ?? {}

    // 根据交易状态动态设置刷新间隔
    const refreshInterval = isTrading ? DEFAULT_TRADING_INTERVAL : DEFAULT_CLOSED_INTERVAL

    const swrKey = fundId ? `${API_BASE_URL}/api/v1/fund/${fundId}/estimate` : null

    const { data, error, isLoading, isValidating, mutate } = useSWR<{ data: FundEstimate; meta?: ResponseMeta }>(
        swrKey,
        fetchEnvelope,
        {
            refreshInterval,
            revalidateOnFocus: isTrading, // 仅交易时段在 focus 时刷新
            revalidateOnReconnect: isTrading,
            keepPreviousData: true, // 🔑 关键: 保持旧数据，避免 UI 闪烁
            dedupingInterval: 5000, // 5秒内相同请求去重
            shouldRetryOnError: false,
            onErrorRetry: (retryError, _key, _config, revalidate, { retryCount }) => {
                if (isFundDataWarmingError(retryError)) {
                    scheduleRetry(retryError, retryCount, revalidate, 12, 5000)
                    return
                }

                scheduleRetry(retryError, retryCount, revalidate, 3, 5000)
            },
            onSuccess: (payload, key, config) => {
                if (typeof onSuccess === 'function') {
                    ;(onSuccess as (data: FundEstimate, key: string, config: unknown) => void)(payload.data, key, config)
                }
            },
            onError: (requestError, key, config) => {
                if (typeof onError === 'function') {
                    ;(onError as (error: unknown, key: string, config: unknown) => void)(requestError, key, config)
                }
            },
            ...restOptions,
        }
    )

    const estimate = data?.data
    const isWarming = isFundDataWarmingError(error)
    const warmingMessage = isWarming ? error.message : ''
    const triggerRetry = useEffectEvent(() => {
        void mutate()
    })

    useEffect(() => {
        if (!isWarming) {
            return
        }

        const delayMs = getRetryDelayMs(error, 5000)
        const timer = window.setTimeout(() => {
            triggerRetry()
        }, delayMs)

        return () => window.clearTimeout(timer)
    }, [error, isWarming])

    return {
        estimate,
        isLoading,
        isValidating, // 后台刷新中
        isError: !!error,
        error,
        mutate, // 手动刷新
        isTrading,
        refreshInterval,
        isWarming,
        warmingMessage,
        retryAfterSeconds: error instanceof ApiRequestError ? error.retryAfterSeconds : 0,
    }
}

/**
 * useFund - 获取基金基本信息
 */
export function useFund(fundId: string | null) {
    const swrKey = fundId ? `${API_BASE_URL}/api/v1/fund/${fundId}` : null

    const { data, error, isLoading, mutate } = useSWR<{ data: Fund; meta?: ResponseMeta }>(
        swrKey,
        fetchEnvelope,
        {
            revalidateOnFocus: false,
            dedupingInterval: 60000, // 基金信息1分钟缓存
        }
    )

    const triggerRetry = useEffectEvent(() => {
        void mutate()
    })

    useEffect(() => {
        if (data?.meta?.cache_status !== 'warming') {
            return
        }

        const timer = window.setTimeout(() => {
            triggerRetry()
        }, 5000)

        return () => window.clearTimeout(timer)
    }, [data?.meta?.cache_status])

    return {
        fund: data?.data,
        cacheStatus: data?.meta?.cache_status || '',
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

    const { data, error, isLoading, mutate } = useSWR<{ data: { fund: Fund; holdings: HoldingDetail[] }; meta?: ResponseMeta }>(
        swrKey,
        fetchEnvelope,
        {
            revalidateOnFocus: false,
            dedupingInterval: 60000,
        }
    )

    const triggerRetry = useEffectEvent(() => {
        void mutate()
    })

    useEffect(() => {
        if (data?.meta?.cache_status !== 'warming') {
            return
        }

        const timer = window.setTimeout(() => {
            triggerRetry()
        }, 5000)

        return () => window.clearTimeout(timer)
    }, [data?.meta?.cache_status])

    return {
        fund: data?.data?.fund,
        holdings: data?.data?.holdings || [],
        cacheStatus: data?.meta?.cache_status || '',
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

    const { data, error, isLoading, mutate } = useSWR<{ data: TimeSeriesResponse; meta?: ResponseMeta }>(
        swrKey,
        fetchEnvelope,
        {
            refreshInterval,
            keepPreviousData: true,
            revalidateOnFocus: false,
            shouldRetryOnError: false,
            onErrorRetry: (retryError, _key, _config, revalidate, { retryCount }) => {
                if (isFundDataWarmingError(retryError)) {
                    scheduleRetry(retryError, retryCount, revalidate, 12, 5000)
                    return
                }

                scheduleRetry(retryError, retryCount, revalidate, 3, 5000)
            },
        }
    )

    const payload = data?.data
    const isWarming = isFundDataWarmingError(error)
    const triggerRetry = useEffectEvent(() => {
        void mutate()
    })

    useEffect(() => {
        if (!isWarming) {
            return
        }

        const delayMs = getRetryDelayMs(error, 5000)
        const timer = window.setTimeout(() => {
            triggerRetry()
        }, delayMs)

        return () => window.clearTimeout(timer)
    }, [error, isWarming])

    return {
        timeSeries: payload?.points || [],
        displayDate: payload?.display_date || '',
        isHistorical: payload?.is_historical || false,
        session: payload?.session || 'after_hours',
        lastTradingDay: payload?.last_trading_day || '',
        isLoading,
        isError: !!error,
        isTrading,
        isWarming,
        warmingMessage: isWarming ? error.message : '',
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

    const { data, error, isLoading } = useSWR<{ data: Fund[]; meta?: ResponseMeta }>(
        swrKey,
        fetchEnvelope,
        {
            revalidateOnFocus: false,
            dedupingInterval: 1000,
        }
    )

    return {
        results: data?.data || [],
        isLoading: shouldSearch && isLoading,
        isError: !!error,
    }
}
