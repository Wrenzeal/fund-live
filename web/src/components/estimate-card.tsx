'use client'

import { useMemo } from 'react'
import { formatPercent, formatCurrency, cn } from '@/lib/utils'
import { TrendingUp, TrendingDown, Minus, RefreshCw } from 'lucide-react'
import type { FundEstimate, Fund } from '@/hooks/use-fund-data'

interface EstimateCardProps {
    estimate: FundEstimate | undefined
    fund: Fund | undefined
    isLoading: boolean
    isCallAuction?: boolean
    isValidating?: boolean
    lastUpdated?: Date | null
    className?: string
}

export function EstimateCard({
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
            ? { text: '-', isPositive: false }
            : formatPercent(estimate?.change_percent),
        [estimate?.change_percent, isCallAuction]
    )

    // 使用 CSS 变量实现主题感知的颜色
    const change = parseFloat(estimate?.change_percent || '0')
    const isPositive = change >= 0
    const TrendIcon = change > 0 ? TrendingUp : change < 0 ? TrendingDown : Minus

    return (
        <div
            className={cn(
                'relative overflow-hidden rounded-3xl p-8 glass',
                // 动态边框颜色
                isPositive
                    ? 'border-[var(--accent-up)]/30'
                    : 'border-[var(--accent-down)]/30',
                className
            )}
            style={{
                // 使用 CSS 变量确保主题感知
                background: isPositive
                    ? 'linear-gradient(135deg, rgba(var(--accent-up-rgb, 244, 63, 94), 0.1), rgba(var(--accent-up-rgb, 244, 63, 94), 0.05))'
                    : 'linear-gradient(135deg, rgba(var(--accent-down-rgb, 16, 185, 129), 0.1), rgba(var(--accent-down-rgb, 16, 185, 129), 0.05))',
            }}
        >
            <div className="relative z-10">
                {/* Fund name and info */}
                <div className="mb-6">
                    <h2 className="text-2xl font-bold text-theme-primary">
                        {isCallAuction ? '集合竞价中' : estimate?.fund_name || fund?.name || '选择基金'}
                    </h2>
                    {!isCallAuction && (estimate?.fund_id || fund?.id) && (
                        <p className="text-sm text-theme-secondary mt-1">
                            基金代码：{estimate?.fund_id || fund?.id}
                        </p>
                    )}
                    <p className="text-sm text-theme-secondary mt-1">
                        {isCallAuction
                            ? '等待 09:30 开盘后更新基金估值数据'
                            : [
                                fund?.manager ? `基金经理: ${fund.manager}` : '',
                                fund?.company ? fund.company : '',
                            ].filter(Boolean).join(' · ')}
                    </p>
                </div>

                {/* Main change display */}
                <div className="flex items-center gap-4 mb-6">
                    <TrendIcon
                        className={cn('w-16 h-16', isPositive ? 'text-up' : 'text-down')}
                        strokeWidth={2.5}
                    />
                    <div>
                        <div className={cn(
                            'text-6xl sm:text-7xl font-black tracking-tight transition-all duration-300',
                            isPositive ? 'text-up' : 'text-down'
                        )}>
                            {isLoading && !estimate && !isCallAuction ? (
                                <RefreshCw className="w-16 h-16 animate-spin" />
                            ) : (
                                changeInfo.text
                            )}
                        </div>
                        <div className="text-lg text-theme-secondary mt-1 flex items-center gap-2">
                            {isCallAuction ? '等待开盘' : '实时预估涨跌幅'}
                            {/* 后台刷新指示器 */}
                            {isValidating && !isCallAuction && (
                                <RefreshCw className="w-4 h-4 animate-spin text-theme-muted" />
                            )}
                        </div>
                    </div>
                </div>

                {/* NAV info */}
                <div className="grid grid-cols-2 gap-4 sm:gap-6">
                    <div className="glass rounded-xl p-4">
                        <div className="text-sm text-theme-muted">预估净值</div>
                        <div className="text-xl sm:text-2xl font-bold text-theme-primary mt-1">
                            {isCallAuction ? '-' : formatCurrency(estimate?.estimate_nav)}
                        </div>
                    </div>
                    <div className="glass rounded-xl p-4">
                        <div className="text-sm text-theme-muted">昨日净值</div>
                        <div className="text-xl sm:text-2xl font-bold text-theme-primary mt-1">
                            {isCallAuction ? '-' : formatCurrency(estimate?.prev_nav)}
                        </div>
                    </div>
                </div>

                {/* Update time */}
                <div className="mt-4 text-xs text-theme-muted flex flex-wrap items-center gap-2">
                    {lastUpdated && !isCallAuction && (
                        <>
                            <span className="inline-block w-2 h-2 rounded-full bg-green-500 animate-pulse" />
                            <span>更新: {lastUpdated.toLocaleTimeString('zh-CN')}</span>
                            <span>·</span>
                        </>
                    )}
                    <span>数据源: {isCallAuction ? '-' : estimate?.data_source || 'N/A'}</span>
                    {estimate?.total_hold_ratio && !isCallAuction && (
                        <>
                            <span>·</span>
                            <span>覆盖率: {parseFloat(estimate.total_hold_ratio).toFixed(1)}%</span>
                        </>
                    )}
                </div>

                {!isCallAuction && officialClose?.display_status === 'pending' && (
                    <div className="mt-4 rounded-2xl border border-amber-500/20 bg-amber-500/10 px-4 py-3 text-sm text-amber-100">
                        {officialClose.message || '真实涨跌情况稍后更新'}
                    </div>
                )}

                {!isCallAuction && officialClose?.display_status === 'ready' && (
                    <div className="mt-4 rounded-2xl border border-cyan-500/20 bg-cyan-500/10 px-4 py-3">
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
                                    官方净值：{formatCurrency(officialClose.net_asset_val)}
                                </div>
                            )}
                        </div>
                    </div>
                )}
            </div>
        </div>
    )
}
