'use client'

import { useMemo } from 'react'
import { formatPercent, formatCurrency, cn } from '@/lib/utils'
import { TrendingUp, TrendingDown, Minus, RefreshCw } from 'lucide-react'
import type { FundEstimate, Fund } from '@/hooks/use-fund-data'

interface EstimateCardProps {
    fundId?: string
    estimate: FundEstimate | undefined
    fund: Fund | undefined
    isLoading: boolean
    isCallAuction?: boolean
    isValidating?: boolean
    lastUpdated?: Date | null
    className?: string
}

function formatNavCurrency(value?: string) {
    if (value === undefined || value === null || value === '') {
        return '--'
    }
    const parsed = Number.parseFloat(value)
    if (!Number.isFinite(parsed) || parsed <= 0) {
        return '--'
    }
    return formatCurrency(value)
}

export function EstimateCard({
    fundId,
    estimate,
    fund,
    isLoading,
    isCallAuction = false,
    isValidating,
    lastUpdated,
    className
}: EstimateCardProps) {
    const officialClose = estimate?.official_close
    const officialCloseChangeInfo = useMemo(() => formatPercent(officialClose?.daily_return), [officialClose?.daily_return])

    const changeInfo = useMemo(() =>
        isCallAuction
            ? { text: '--', isPositive: false }
            : formatPercent(estimate?.change_percent),
        [estimate?.change_percent, isCallAuction]
    )

    // 使用 CSS 变量实现主题感知的颜色，但缺失数据必须保持中性，不渲染为 0 或正收益。
    const change = Number.parseFloat(estimate?.change_percent || '')
    const hasChange = !isCallAuction && Number.isFinite(change)
    const isPositive = hasChange && change >= 0
    const isNegative = hasChange && change < 0
    const TrendIcon = isPositive ? TrendingUp : isNegative ? TrendingDown : Minus
    const fundCode = fund?.id || estimate?.fund_id || fundId
    const displayName = fund?.name || estimate?.fund_name || (isLoading ? '基金信息加载中' : fundCode ? `基金 ${fundCode}` : '基金信息待同步')
    const metaText = [
            fund?.manager ? `基金经理: ${fund.manager}` : '',
            fund?.company ? fund.company : '',
        ].filter(Boolean).join(' · ')

    return (
        <div
            className={cn(
                'relative overflow-hidden rounded-xl border border-[var(--card-border)] bg-[var(--card-bg)] p-5 sm:p-7',
                className
            )}
        >
            <div className="relative z-10">
                {/* Fund name and info */}
                <div className="mb-6">
                    <h2 className="text-2xl font-bold text-theme-primary">
                        {displayName}
                    </h2>
                    {fundCode && (
                        <p className="text-sm text-theme-secondary mt-1">
                            基金代码：{fundCode}
                        </p>
                    )}
                    {(fund?.category_name || fund?.type) && <p className="mt-1 text-sm text-theme-secondary">{fund?.category_name || ({ hybrid: '混合型', stock: '股票型', bond: '债券型', index: '指数型', money: '货币型', qdii: 'QDII', feeder: 'ETF 联接', fof: 'FOF' } as Record<string, string>)[fund?.type || ''] || fund?.type}</p>}
                    {metaText && (
                        <p className="text-sm text-theme-secondary mt-1">
                            {metaText}
                        </p>
                    )}
                    {isCallAuction && (
                        <p className="mt-2 text-sm text-theme-secondary">集合竞价中 · 等待 09:30 开盘后更新基金估值</p>
                    )}
                </div>

                {/* Main change display */}
                <div className="flex items-start gap-3 sm:items-center sm:gap-4 mb-6">
                    <TrendIcon
                        className={cn('h-10 w-10 shrink-0 sm:h-16 sm:w-16', isPositive ? 'text-up' : isNegative ? 'text-down' : 'text-theme-muted')}
                        strokeWidth={2.5}
                    />
                    <div className="min-w-0">
                        <div className={cn(
                            'text-4xl sm:text-5xl font-semibold leading-none tracking-tight tabular-nums transition-colors duration-160',
                            isPositive ? 'text-up' : isNegative ? 'text-down' : 'text-theme-muted'
                        )}>
                            {isLoading && !estimate && !isCallAuction ? (
                                <RefreshCw className="h-10 w-10 animate-spin sm:h-16 sm:w-16" />
                            ) : (
                                changeInfo.text
                            )}
                        </div>
                        <div className="mt-2 flex flex-wrap items-center gap-2 text-sm text-theme-secondary sm:text-lg">
                            {isCallAuction ? '等待开盘' : '实时预估涨跌幅'}
                            {/* 后台刷新指示器 */}
                            {isValidating && !isCallAuction && (
                                <RefreshCw className="w-4 h-4 animate-spin text-theme-muted" />
                            )}
                        </div>
                    </div>
                </div>

                {/* NAV info */}
                <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 sm:gap-6">
                    <div className="rounded-xl border border-[var(--card-border)] bg-[var(--input-bg)]/55 p-3 sm:p-4">
                        <div className="text-sm text-theme-muted">预估净值</div>
                        <div className="text-lg sm:text-2xl font-bold text-theme-primary mt-1">
                            {isCallAuction ? '--' : isLoading && !estimate ? '正在读取' : formatNavCurrency(estimate?.estimate_nav)}
                        </div>
                    </div>
                    <div className="rounded-xl border border-[var(--card-border)] bg-[var(--input-bg)]/55 p-3 sm:p-4">
                        <div className="text-sm text-theme-muted">昨日净值</div>
                        <div className="text-lg sm:text-2xl font-bold text-theme-primary mt-1">
                            {isCallAuction ? '--' : isLoading && !estimate ? '正在读取' : formatNavCurrency(estimate?.prev_nav)}
                        </div>
                    </div>
                </div>

                {/* Update time */}
                <div className="mt-4 text-xs text-theme-muted flex flex-wrap items-center gap-2">
                    {lastUpdated && !isCallAuction && (
                        <>
                            <span className="inline-block w-2 h-2 rounded-full bg-success-soft animate-pulse" />
                            <span>更新: {lastUpdated.toLocaleTimeString('zh-CN')}</span>
                            <span>·</span>
                        </>
                    )}
                            <span>来源：{isCallAuction ? '--' : estimate?.data_source || (isLoading ? '同步中' : '暂不可用')}</span>
                    {estimate?.total_hold_ratio && !isCallAuction && (
                        <>
                            <span>·</span>
                            <span>覆盖率: {parseFloat(estimate.total_hold_ratio).toFixed(1)}%</span>
                        </>
                    )}
                </div>

                {!isCallAuction && officialClose?.display_status === 'pending' && (
                    <div className="mt-4 rounded-xl border border-warning/20 bg-warning/10 px-4 py-3 text-sm text-warning">
                        {officialClose.message || '真实涨跌情况稍后更新'}
                    </div>
                )}

                {!isCallAuction && officialClose?.display_status === 'ready' && (
                    <div className="mt-4 rounded-xl border border-accent/20 bg-accent/10 px-4 py-3">
                        <div className="text-xs tracking-[0.18em] text-theme-muted">
                            {officialClose.date ? `${officialClose.date} 官方收盘结果` : '官方收盘结果'}
                        </div>
                        <div className="mt-2 flex flex-wrap items-end justify-between gap-3">
                            <div className={cn(
                                'text-2xl font-bold',
                                officialCloseChangeInfo.isPositive ? 'text-up' : 'text-down'
                            )}>
                                {officialCloseChangeInfo.text}
                            </div>
                            {officialClose.net_asset_val && (
                                <div className="text-sm text-theme-secondary">
                                    官方净值：{formatNavCurrency(officialClose.net_asset_val)}
                                </div>
                            )}
                        </div>
                    </div>
                )}
            </div>
        </div>
    )
}
