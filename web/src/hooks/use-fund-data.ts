'use client'

import { useEffect, useEffectEvent } from 'react'
import useSWR, { SWRConfiguration } from 'swr'
import { useMarketTradingState } from './use-market-status'
import { API_BASE_URL } from '@/lib/api-base-url'

// API 基础 URL

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

async function fetchEnvelopeWithTimeout<T>(url: string, timeoutMs: number): Promise<{ data: T; meta?: ResponseMeta }> {
    const controller = new AbortController()
    const timer = window.setTimeout(() => controller.abort(), timeoutMs)

    try {
        const res = await fetch(url, { signal: controller.signal })

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
    } catch (error) {
        if (error instanceof DOMException && error.name === 'AbortError') {
            throw new Error(`请求超时（${Math.round(timeoutMs / 1000)} 秒），请稍后重试`)
        }
        throw error
    } finally {
        window.clearTimeout(timer)
    }
}

async function requestEnvelope<T>(path: string, init?: RequestInit): Promise<T | null> {
    const res = await fetch(`${API_BASE_URL}${path}`, {
        credentials: 'include',
        headers: {
            'Content-Type': 'application/json',
            ...(init?.headers ?? {}),
        },
        ...init,
    })

    let json: ApiEnvelope<T> | null = null
    try {
        json = await res.json() as ApiEnvelope<T>
    } catch {
        json = null
    }

    if (!res.ok || !json?.success) {
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

    return json.data ?? null
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
    category_code?: string
    category_name?: string
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

export interface FundHoldingRecord {
    stock_code: string
    stock_name: string
    exchange?: string
    holding_ratio: string
    holding_shares?: string
    market_value?: string
    reporting_period?: string
}

export interface FundHoldingsDisplayItem {
    item_type: 'stock' | 'target_fund'
    target_type?: 'etf_fund' | 'fund' | 'index'
    code: string
    name: string
    exchange?: string
    holding_ratio?: string
    weight_percent?: string
    reporting_period?: string
    is_primary?: boolean
    source?: string
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
    official_close?: OfficialCloseInfo
}

export interface TimeSeriesPoint {
    timestamp: string
    change_percent: string
    estimate_nav: string
}

export interface OfficialCloseInfo {
    display_status: 'hidden' | 'pending' | 'ready'
    date?: string
    daily_return?: string
    net_asset_val?: string
    message?: string
}

export interface FundSectorBreakdown {
    sector_code: string
    sector_name: string
    weight_percent: string
    rank: number
}

export interface FundSectorSnapshot {
    fund_id: string
    as_of_date: string
    primary_sector_code: string
    primary_sector_name: string
    source: string
    confidence: string
    breakdown: FundSectorBreakdown[]
}

export interface FundThemeBreakdown {
    theme_code: string
    theme_name: string
    weight_percent: string
    rank: number
}

export interface FundThemeSnapshot {
    fund_id: string
    as_of_date: string
    primary_theme_code: string
    primary_theme_name: string
    source: string
    confidence: string
    breakdown: FundThemeBreakdown[]
}

export interface FundClassificationOption {
    code: string
    name: string
    description?: string
    sort_order: number
}

export interface FundClassificationOptions {
    categories: FundClassificationOption[]
    sectors: FundClassificationOption[]
    themes: FundClassificationOption[]
}

export interface FundClassificationOverride {
    fund_id: string
    category_code?: string
    category_name?: string
    primary_sector_code?: string
    primary_sector_name?: string
    primary_theme_code?: string
    primary_theme_name?: string
    manual_tags: string[]
    note?: string
    updated_by?: string
    created_at?: string
    updated_at?: string
}

export interface FundClassificationOverrideInput {
    category_code?: string
    primary_sector_code?: string
    primary_theme_code?: string
    manual_tags?: string[]
    note?: string
}

export interface FundAnalysisModuleScore {
    code: string
    name: string
    score: string
    summary?: string
}

export interface FundAnalysisEventImpact {
    code: string
    title: string
    impact: 'positive' | 'neutral' | 'negative'
    summary: string
    target_scope?: 'disclosure' | 'methodology' | 'holding' | 'exposure' | 'fund' | 'macro' | 'index'
    strength?: 'low' | 'medium' | 'high'
    horizon?: 'intraday' | 'current' | 'quarterly' | 'medium_term'
    related_symbols?: string[]
    weight_hint?: string
}

export interface FundAnalysisConfidenceFactor {
    code: string
    name: string
    level: 'high' | 'medium' | 'low'
    score: string
    summary: string
}

export interface FundAnalysisEvidenceItem {
    code: string
    title: string
    summary: string
    evidence_type: string
    source_scope?: 'disclosure' | 'methodology' | 'holding' | 'exposure' | 'fund' | 'macro' | 'index'
    impact?: 'positive' | 'neutral' | 'negative'
    strength?: 'low' | 'medium' | 'high'
    horizon?: 'intraday' | 'current' | 'quarterly' | 'medium_term'
    related_symbols?: string[]
    weight_hint?: string
}

export interface FundAnalysisAIExplanationCitation {
    code: string
    source_type: string
    source_scope?: 'disclosure' | 'methodology' | 'holding' | 'exposure' | 'fund' | 'macro' | 'index'
    title: string
    summary?: string
}

export interface FundAnalysisAIExplanationSection {
    title: string
    summary: string
    citation_codes?: string[]
}

export interface FundAnalysisAIExplanation {
    status: 'ready' | 'disabled' | 'fallback' | 'rejected' | 'failed'
    provider: string
    model?: string
    generated_at?: string
    cache_key?: string
    cache_status?: 'generated' | 'snapshot_hit' | 'not_cacheable'
    expires_at?: string
    invalidation_basis?: string[]
    rule_recommendation?: 'increase' | 'hold' | 'decrease'
    boundary_notice: string
    summary: string
    attribution?: FundAnalysisAIExplanationSection[]
    risk_notes?: FundAnalysisAIExplanationSection[]
    citations?: FundAnalysisAIExplanationCitation[]
    limitations?: string[]
}

export interface FundAnalysis {
    analysis_version: string
    analysis_type: 'direct_holdings' | 'tracked_etf' | 'qdii_holdings'
    analysis_basis: string
    as_of_time: string
    total_score: string
    confidence: 'high' | 'medium' | 'low'
    risk_level: 'low' | 'medium' | 'high'
    increase_percent: string
    hold_percent: string
    decrease_percent: string
    latest_holding_period?: string
    summary: string
    reasons: string[]
    warnings: string[]
    event_impacts: FundAnalysisEventImpact[]
    module_scores: FundAnalysisModuleScore[]
    confidence_factors?: FundAnalysisConfidenceFactor[]
    primary_evidence?: FundAnalysisEvidenceItem[]
    counter_evidence?: FundAnalysisEvidenceItem[]
    confidence_deductions?: string[]
    ai_explanation?: FundAnalysisAIExplanation
}

// 默认 SWR 配置
const DEFAULT_TRADING_INTERVAL = 30000  // 交易时段 30秒刷新
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

    const { data, error, isLoading, mutate } = useSWR<{ data: { fund: Fund; holdings: FundHoldingRecord[]; display_level?: 'stock_layer' | 'target_layer'; display_items?: FundHoldingsDisplayItem[]; lookthrough_available?: boolean }; meta?: ResponseMeta }>(
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
        displayLevel: data?.data?.display_level || 'stock_layer',
        displayItems: data?.data?.display_items || [],
        lookthroughAvailable: data?.data?.lookthrough_available || false,
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
    official_close?: OfficialCloseInfo
}

export interface FundDashboardPayload {
    fund?: Fund
    estimate?: FundEstimate
    analysis?: FundAnalysis
    sector_snapshot?: FundSectorSnapshot
    theme_snapshot?: FundThemeSnapshot
    classification_override?: FundClassificationOverride
    time_series: TimeSeriesPoint[]
    display_date: string
    is_trading: boolean
    is_historical: boolean
    session: 'pre_market' | 'morning' | 'lunch_break' | 'afternoon' | 'after_hours' | 'weekend' | 'holiday'
    last_trading_day: string
}

export interface FundAnalysisPayload {
    fund?: Fund
    analysis?: FundAnalysis
}

export interface FundAnalysisRankingItem {
    fund?: Fund
    analysis?: FundAnalysis
}

export interface FundAnalysisRankingsPayload {
    generated_at: string
    increase_ideas: FundAnalysisRankingItem[]
    watch_ideas: FundAnalysisRankingItem[]
    risk_alerts: FundAnalysisRankingItem[]
}

export interface FundExposureSnapshotEntry {
    sectorSnapshot?: FundSectorSnapshot
    themeSnapshot?: FundThemeSnapshot
    classificationOverride?: FundClassificationOverride
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
        officialClose: payload?.official_close,
        isLoading,
        isError: !!error,
        isTrading,
        isWarming,
        warmingMessage: isWarming ? error.message : '',
        mutate,
    }
}

function useFundDashboardPayload(fundId: string | null, options?: SWRConfiguration) {
    const { isTrading } = useMarketTradingState()
    const refreshInterval = isTrading ? DEFAULT_TRADING_INTERVAL : DEFAULT_CLOSED_INTERVAL
    const {
        onSuccess,
        onError,
        ...restOptions
    } = options ?? {}

    const swrKey = fundId ? `${API_BASE_URL}/api/v1/fund/${fundId}/dashboard?include_analysis=false` : null

    const { data, error, isLoading, isValidating, mutate } = useSWR<{ data: FundDashboardPayload; meta?: ResponseMeta }>(
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
            onSuccess: (payload, key, config) => {
                if (typeof onSuccess === 'function') {
                    ;(onSuccess as (data: FundDashboardPayload, key: string, config: unknown) => void)(payload.data, key, config)
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
        payload: data?.data,
        meta: data?.meta,
        isLoading,
        isValidating,
        isError: !!error,
        error,
        mutate,
        isTrading,
        refreshInterval,
        isWarming: data?.meta?.cache_status === 'warming',
    }
}

export function useFundDashboard(fundId: string | null, options?: SWRConfiguration) {
    const dashboard = useFundDashboardPayload(fundId, options)

    return {
        fund: dashboard.payload?.fund,
        estimate: dashboard.payload?.estimate,
        analysis: dashboard.payload?.analysis,
        sectorSnapshot: dashboard.payload?.sector_snapshot,
        themeSnapshot: dashboard.payload?.theme_snapshot,
        classificationOverride: dashboard.payload?.classification_override,
        timeSeries: dashboard.payload?.time_series || [],
        displayDate: dashboard.payload?.display_date || '',
        isHistorical: dashboard.payload?.is_historical || false,
        session: dashboard.payload?.session || 'after_hours',
        lastTradingDay: dashboard.payload?.last_trading_day || '',
        officialClose: dashboard.payload?.estimate?.official_close,
        cacheStatus: dashboard.meta?.cache_status || '',
        isLoading: dashboard.isLoading,
        isValidating: dashboard.isValidating,
        isError: dashboard.isError,
        error: dashboard.error,
        mutate: dashboard.mutate,
        isTrading: dashboard.isTrading,
        refreshInterval: dashboard.refreshInterval,
        isWarming: dashboard.isWarming,
    }
}

export async function fetchFundClassificationOptions(): Promise<FundClassificationOptions> {
    const data = await requestEnvelope<FundClassificationOptions>('/api/v1/admin/funds/classification-options', {
        method: 'GET',
    })
    return data ?? { categories: [], sectors: [], themes: [] }
}

export async function fetchFundClassificationOverride(fundId: string): Promise<FundClassificationOverride | null> {
    return requestEnvelope<FundClassificationOverride>(
        `/api/v1/admin/funds/${encodeURIComponent(fundId)}/classification`,
        { method: 'GET' }
    )
}

export async function updateFundClassificationOverride(
    fundId: string,
    input: FundClassificationOverrideInput
): Promise<FundClassificationOverride | null> {
    return requestEnvelope<FundClassificationOverride>(
        `/api/v1/admin/funds/${encodeURIComponent(fundId)}/classification`,
        {
            method: 'PUT',
            body: JSON.stringify(input),
        }
    )
}

async function fetchFundExposureSnapshots(fundIDs: string[]): Promise<Record<string, FundExposureSnapshotEntry>> {
    const entries = await Promise.all(
        fundIDs.map(async (fundID) => {
            try {
                const payload = await fetchEnvelope<FundDashboardPayload>(`${API_BASE_URL}/api/v1/fund/${fundID}/dashboard?include_analysis=false`)
                return [fundID, {
                    sectorSnapshot: payload.data.sector_snapshot,
                    themeSnapshot: payload.data.theme_snapshot,
                    classificationOverride: payload.data.classification_override,
                }] as const
            } catch {
                return [fundID, {}] as const
            }
        })
    )

    return Object.fromEntries(entries)
}

export function useFundExposureSnapshots(fundIDs: string[]) {
    const { isTrading } = useMarketTradingState()
    const normalizedFundIDs = [...new Set(fundIDs.filter(Boolean))].sort()

    const { data = {}, error, isLoading, isValidating } = useSWR<Record<string, FundExposureSnapshotEntry>>(
        normalizedFundIDs.length > 0 ? ['fund-exposure-snapshots', normalizedFundIDs.join(',')] : null,
        () => fetchFundExposureSnapshots(normalizedFundIDs),
        {
            refreshInterval: isTrading ? DEFAULT_TRADING_INTERVAL : DEFAULT_CLOSED_INTERVAL,
            revalidateOnFocus: false,
            dedupingInterval: 10000,
        }
    )

    return {
        exposureSnapshotsByFundID: data,
        isLoading,
        isValidating,
        isError: !!error,
    }
}

async function fetchFundTopHoldings(fundIDs: string[]): Promise<Record<string, FundHoldingRecord[]>> {
    const entries = await Promise.all(
        fundIDs.map(async (fundID) => {
            try {
                const payload = await fetchEnvelope<{ fund: Fund; holdings: FundHoldingRecord[] }>(
                    `${API_BASE_URL}/api/v1/fund/${fundID}/holdings`
                )
                return [fundID, payload.data.holdings ?? []] as const
            } catch {
                return [fundID, []] as const
            }
        })
    )

    return Object.fromEntries(entries)
}

export function useFundTopHoldings(fundIDs: string[]) {
    const normalizedFundIDs = [...new Set(fundIDs.filter(Boolean))].sort()

    const { data = {}, error, isLoading, isValidating } = useSWR<Record<string, FundHoldingRecord[]>>(
        normalizedFundIDs.length > 0 ? ['fund-top-holdings', normalizedFundIDs.join(',')] : null,
        () => fetchFundTopHoldings(normalizedFundIDs),
        {
            revalidateOnFocus: false,
            dedupingInterval: 60000,
        }
    )

    return {
        topHoldingsByFundID: data,
        isLoading,
        isValidating,
        isError: !!error,
    }
}

export function useFundAnalysis(fundId: string | null, options?: SWRConfiguration) {
    const { isTrading } = useMarketTradingState()
    const refreshInterval = isTrading ? DEFAULT_TRADING_INTERVAL : DEFAULT_CLOSED_INTERVAL
    const {
        onSuccess,
        onError,
        ...restOptions
    } = options ?? {}

    const swrKey = fundId ? `${API_BASE_URL}/api/v1/fund/${fundId}/analysis` : null

    const { data, error, isLoading, isValidating, mutate } = useSWR<{ data: FundAnalysisPayload; meta?: ResponseMeta }>(
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
            onSuccess: (payload, key, config) => {
                if (typeof onSuccess === 'function') {
                    ;(onSuccess as (data: FundAnalysisPayload, key: string, config: unknown) => void)(payload.data, key, config)
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
        analysis: data?.data?.analysis,
        cacheStatus: data?.meta?.cache_status || '',
        isLoading,
        isValidating,
        isError: !!error,
        error,
        mutate,
        isTrading,
        refreshInterval,
        isWarming: data?.meta?.cache_status === 'warming',
    }
}

async function fetchFundAnalyses(fundIDs: string[]): Promise<Record<string, FundAnalysis | null>> {
    if (fundIDs.length === 0) {
        return {}
    }
    try {
        const payload = await fetchEnvelope<{ analyses: Record<string, FundAnalysis | null> }>(
            `${API_BASE_URL}/api/v1/analysis/batch?fund_ids=${encodeURIComponent(fundIDs.join(','))}`
        )
        return payload.data?.analyses || {}
    } catch {
        const entries = await Promise.all(
            fundIDs.map(async (fundID) => {
                try {
                    const payload = await fetchEnvelope<FundAnalysisPayload>(`${API_BASE_URL}/api/v1/fund/${fundID}/analysis`)
                    return [fundID, payload.data?.analysis ?? null] as const
                } catch {
                    return [fundID, null] as const
                }
            })
        )
        return Object.fromEntries(entries)
    }
}

export function useFundAnalyses(fundIDs: string[]) {
    const { isTrading } = useMarketTradingState()
    const normalizedFundIDs = [...new Set(fundIDs.filter(Boolean))].sort()

    const { data = {}, error, isLoading, isValidating } = useSWR<Record<string, FundAnalysis | null>>(
        normalizedFundIDs.length > 0 ? ['fund-analyses', normalizedFundIDs.join(',')] : null,
        () => fetchFundAnalyses(normalizedFundIDs),
        {
            refreshInterval: isTrading ? DEFAULT_TRADING_INTERVAL : DEFAULT_CLOSED_INTERVAL,
            revalidateOnFocus: false,
            dedupingInterval: 5000,
        }
    )

    return {
        analysesByFundID: data,
        isLoading,
        isValidating,
        isError: !!error,
    }
}

export function useFundAnalysisRankings() {
    const { isTrading } = useMarketTradingState()
    const refreshInterval = isTrading ? DEFAULT_TRADING_INTERVAL : DEFAULT_CLOSED_INTERVAL

    const { data, error, isLoading, isValidating, mutate } = useSWR<{ data: FundAnalysisRankingsPayload; meta?: ResponseMeta }>(
        `${API_BASE_URL}/api/v1/analysis/rankings`,
        (url: string) => fetchEnvelopeWithTimeout<FundAnalysisRankingsPayload>(url, 15000),
        {
            refreshInterval,
            revalidateOnFocus: false,
            dedupingInterval: 5000,
        }
    )

    return {
        rankings: data?.data,
        isLoading,
        isValidating,
        isError: !!error,
        error,
        mutate,
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
