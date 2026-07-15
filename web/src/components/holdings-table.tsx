'use client'

import { useMemo } from 'react'
import { formatPercent, formatCurrency, formatRatio, cn } from '@/lib/utils'
import { TrendingUp, TrendingDown, Minus } from 'lucide-react'
import type { FundEstimate, FundHoldingsDisplayItem } from '@/hooks/use-fund-data'
import { Disclosure } from '@/components/ui/disclosure'

interface HoldingsTableProps {
    estimate?: FundEstimate
    displayLevel?: 'stock_layer' | 'target_layer'
    items?: FundHoldingsDisplayItem[]
    lookthroughAvailable?: boolean
    isLoading?: boolean
    isCallAuction?: boolean
    className?: string
}

interface DisplayHoldingRow {
    item_type: 'stock' | 'target_fund'
    target_type?: 'etf_fund' | 'fund' | 'index'
    code: string
    name: string
    holding_ratio?: string
    weight_percent?: string
    current_price?: string
    stock_change?: string
    contribution?: string
    source?: string
}

export function HoldingsTable({
    estimate,
    displayLevel = 'stock_layer',
    items: displayItems = [],
    lookthroughAvailable = false,
    isLoading = false,
    isCallAuction = false,
    className,
}: HoldingsTableProps) {
    const rows = useMemo<DisplayHoldingRow[]>(() => {
        if (displayLevel === 'target_layer') {
            return displayItems.map((item) => ({
                item_type: item.item_type,
                target_type: item.target_type,
                code: item.code,
                name: item.name,
                weight_percent: item.weight_percent,
                source: item.source,
            }))
        }

        const estimateByCode = Object.fromEntries(
            (estimate?.holding_details || []).map((holding) => [holding.stock_code, holding])
        )

        return displayItems.map((item) => {
            const estimateItem = estimateByCode[item.code]
            return {
                item_type: item.item_type,
                code: item.code,
                name: item.name,
                holding_ratio: item.holding_ratio,
                weight_percent: item.weight_percent,
                current_price: estimateItem?.current_price,
                stock_change: estimateItem?.stock_change,
                contribution: estimateItem?.contribution,
                source: item.source,
            }
        })
    }, [displayItems, displayLevel, estimate?.holding_details])

    if (rows.length === 0) {
        return (
            <div className={cn('glass rounded-2xl p-4 sm:p-6', className)}>
                <h3 className="mb-4 text-lg font-semibold text-theme-primary">
                    {displayLevel === 'target_layer' ? '追踪目标' : '持仓明细'}
                </h3>
                <HoldingsSystemState
                    code={isLoading ? 'HOLDINGS_LINKING' : 'DATA_LINK_OFFLINE'}
                    text={isLoading ? '正在整理持仓数据' : '暂无持仓数据'}
                    scanning={isLoading}
                />
            </div>
        )
    }

    const totalHoldingRatio = rows.reduce((sum, holding) => {
        const ratio = parseOptionalNumber(holding.holding_ratio)
        return Number.isFinite(ratio) ? sum + ratio : sum
    }, 0)
    const hasTotalHoldingRatio = rows.some((holding) => Number.isFinite(parseOptionalNumber(holding.holding_ratio)))

    return (
        <div className={cn('glass rounded-2xl p-4 sm:p-6', className)}>
            <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div>
                    <h3 className="text-lg font-semibold text-theme-primary">
                        {displayLevel === 'target_layer' ? '追踪目标' : '重仓股明细'}
                    </h3>
                    <p className="mt-1 text-xs text-theme-muted">
                        {displayLevel === 'target_layer'
                            ? `默认展示下一层目标，共识别 ${rows.length} 个目标`
                            : isCallAuction
                                ? `集合竞价中，已展示 ${rows.length} 只固定持仓信息`
                                : `当前参与估值展示 ${rows.length} / 10 只`}
                    </p>
                </div>
                <span className="text-xs text-theme-muted sm:text-right">
                    {displayLevel === 'target_layer'
                        ? (lookthroughAvailable ? '可继续穿透估值' : '当前无穿透估值')
                        : `合计占比: ${isCallAuction ? hasTotalHoldingRatio ? formatRatio(totalHoldingRatio.toString()) : '--' : formatRatio(estimate?.total_hold_ratio)}`}
                </span>
            </div>

            <div className="overflow-x-auto [-webkit-overflow-scrolling:touch]">
                <table className="w-full min-w-[42rem]">
                    <thead>
                        {displayLevel === 'target_layer' ? (
                            <tr className="text-xs text-theme-muted border-b border-[var(--card-border)]">
                                <th className="text-left py-3 px-2">目标名称</th>
                                <th className="text-right py-3 px-2">代码</th>
                                <th className="text-right py-3 px-2">类型</th>
                                <th className="text-right py-3 px-2">权重</th>
                                <th className="text-right py-3 px-2">说明</th>
                            </tr>
                        ) : (
                            <tr className="text-xs text-theme-muted border-b border-[var(--card-border)]">
                                <th className="text-left py-3 px-2">股票名称</th>
                                <th className="text-right py-3 px-2">代码</th>
                                <th className="text-right py-3 px-2">持仓占比</th>
                                <th className="text-right py-3 px-2">现价</th>
                                <th className="text-right py-3 px-2">涨跌幅</th>
                                <th className="text-right py-3 px-2">贡献</th>
                            </tr>
                        )}
                    </thead>
                    <tbody>
                        {rows.map((holding, index) => (
                            <HoldingRow
                                key={`${holding.item_type}:${holding.code || holding.name}:${index}`}
                                holding={holding}
                                index={index}
                                displayLevel={displayLevel}
                                isCallAuction={isCallAuction}
                            />
                        ))}
                    </tbody>
                </table>
            </div>
            <div className="mt-2 text-[11px] text-theme-muted sm:hidden">横向滑动查看更多列</div>

            <div className="mt-4">
              <Disclosure summary="查看持仓口径">
                {displayLevel === 'target_layer' ? (
                    <p>默认展示基金的下一层追踪目标；底层股票只用于估值计算。</p>
                ) : isCallAuction ? (
                    <p>集合竞价阶段只展示名称、代码和占比；涨跌幅、贡献等字段在开盘后更新。</p>
                ) : (
                    <p><strong>贡献</strong> = 个股涨跌幅 × 持仓占比 / 100，表示该股对基金整体涨跌的影响。</p>
                )}
              </Disclosure>
            </div>
        </div>
    )
}

// 单独的行组件，使用 memo 优化
function HoldingRow({
    holding,
    index,
    displayLevel,
    isCallAuction = false,
}: {
    holding: DisplayHoldingRow
    index: number
    displayLevel: 'stock_layer' | 'target_layer'
    isCallAuction?: boolean
}) {
    const changeInfo = formatPercent(holding.stock_change)
    const change = parseOptionalNumber(holding.stock_change)
    const contribution = parseOptionalNumber(holding.contribution)

    const TrendIcon = change > 0 ? TrendingUp : change < 0 ? TrendingDown : Minus

    // 使用 CSS 变量实现主题感知
    const isPositive = change > 0
    const isNegative = change < 0
    const contribPositive = contribution > 0
    const contribNegative = contribution < 0
    const contributionText = Number.isFinite(contribution) ? `${contribution >= 0 ? '+' : ''}${contribution.toFixed(4)}%` : '--'

    if (displayLevel === 'target_layer') {
        return (
            <tr className="border-b border-[var(--card-border)] hover:bg-[var(--card-bg)] transition-colors">
                <td className="py-3 px-2">
                    <div className="flex items-center gap-2">
                        <span className="text-xs text-theme-muted w-4">{index + 1}</span>
                        <span className="font-medium text-theme-primary">{holding.name}</span>
                    </div>
                </td>
                <td className="text-right py-3 px-2 text-theme-secondary font-mono text-sm">
                    {holding.code || '--'}
                </td>
                <td className="text-right py-3 px-2 text-theme-secondary">
                    {holding.target_type === 'etf_fund'
                        ? 'ETF'
                        : holding.target_type === 'index'
                            ? '指数'
                            : '基金'}
                </td>
                <td className="text-right py-3 px-2 text-theme-secondary">
                    {holding.weight_percent ? formatRatio(holding.weight_percent) : '--'}
                </td>
                <td className="text-right py-3 px-2 text-theme-muted">
                    下一层目标
                </td>
            </tr>
        )
    }

    return (
        <tr className="border-b border-[var(--card-border)] hover:bg-[var(--card-bg)] transition-colors">
            <td className="py-3 px-2">
                <div className="flex items-center gap-2">
                    <span className="text-xs text-theme-muted w-4">{index + 1}</span>
                    <span className="font-medium text-theme-primary">{holding.name}</span>
                </div>
            </td>
            <td className="text-right py-3 px-2 text-theme-secondary font-mono text-sm">
                {holding.code}
            </td>
            <td className="text-right py-3 px-2 text-theme-secondary">
                {formatRatio(holding.holding_ratio)}
            </td>
            <td className="text-right py-3 px-2 text-theme-primary font-mono">
                {isCallAuction ? '--' : formatCurrency(holding.current_price).replace('¥', '')}
            </td>
            <td className={cn(
                'text-right py-3 px-2 font-medium',
                isPositive && 'text-up',
                isNegative && 'text-down',
                !isPositive && !isNegative && 'text-theme-muted'
            )}>
                {isCallAuction ? (
                    <span>--</span>
                ) : (
                    <div className="flex items-center justify-end gap-1">
                        <TrendIcon className="w-3 h-3" />
                        {changeInfo.text}
                    </div>
                )}
            </td>
            <td className={cn(
                'text-right py-3 px-2 font-medium',
                contribPositive && 'text-up',
                contribNegative && 'text-down',
                !contribPositive && !contribNegative && 'text-theme-muted'
            )}>
                {isCallAuction ? '--' : contributionText}
            </td>
        </tr>
    )
}

function parseOptionalNumber(value?: string) {
    if (value === undefined || value === null || value === '') {
        return Number.NaN
    }
    const parsed = Number.parseFloat(value)
    return Number.isFinite(parsed) ? parsed : Number.NaN
}

function HoldingsSystemState({
  code,
  text,
  scanning = false,
}: {
    code: string
    text: string
    scanning?: boolean
}) {
    return (
        <div role="status" data-diagnostic-code={code} data-loading={scanning || undefined} className="rounded-2xl border border-dashed border-[var(--card-border)] bg-[var(--card-bg)]/25 px-4 py-8 text-center text-sm text-theme-muted">
            <div>{text}</div>
            <div className="mt-2">{scanning ? '数据到达后会自动更新。' : '可以稍后重试。'}</div>
        </div>
    )
}
