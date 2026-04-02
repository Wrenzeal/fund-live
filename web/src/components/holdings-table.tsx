'use client'

import { useMemo } from 'react'
import { formatPercent, formatCurrency, formatRatio, cn } from '@/lib/utils'
import { TrendingUp, TrendingDown, Minus } from 'lucide-react'
import type { HoldingDetail, FundEstimate } from '@/hooks/use-fund-data'

interface HoldingsTableProps {
    estimate?: FundEstimate
    className?: string
}

export function HoldingsTable({ estimate, className }: HoldingsTableProps) {
    const holdings = useMemo(() =>
        estimate?.holding_details || [],
        [estimate?.holding_details]
    )

    if (holdings.length === 0) {
        return (
            <div className={cn('glass rounded-2xl p-6', className)}>
                <h3 className="text-lg font-semibold text-theme-primary mb-4">持仓明细</h3>
                <p className="text-theme-muted text-center py-8">暂无持仓数据</p>
            </div>
        )
    }

    return (
        <div className={cn('glass rounded-2xl p-6', className)}>
            <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-theme-primary">前十大重仓股</h3>
                <span className="text-xs text-theme-muted">
                    合计占比: {formatRatio(estimate?.total_hold_ratio)}
                </span>
            </div>

            <div className="overflow-x-auto">
                <table className="w-full">
                    <thead>
                        <tr className="text-xs text-theme-muted border-b border-[var(--card-border)]">
                            <th className="text-left py-3 px-2">股票名称</th>
                            <th className="text-right py-3 px-2">代码</th>
                            <th className="text-right py-3 px-2">持仓占比</th>
                            <th className="text-right py-3 px-2">现价</th>
                            <th className="text-right py-3 px-2">涨跌幅</th>
                            <th className="text-right py-3 px-2">贡献</th>
                        </tr>
                    </thead>
                    <tbody>
                        {holdings.map((holding, index) => (
                            <HoldingRow key={holding.stock_code} holding={holding} index={index} />
                        ))}
                    </tbody>
                </table>
            </div>

            {/* Legend */}
            <div className="mt-4 pt-4 border-t border-[var(--card-border)] text-xs text-theme-muted">
                <p>
                    💡 <strong>贡献</strong> = 个股涨跌幅 × 持仓占比 / 100，表示该股对基金整体涨跌的影响
                </p>
            </div>
        </div>
    )
}

// 单独的行组件，使用 memo 优化
function HoldingRow({ holding, index }: { holding: HoldingDetail; index: number }) {
    const changeInfo = formatPercent(holding.stock_change)
    const change = parseFloat(holding.stock_change || '0')
    const contribution = parseFloat(holding.contribution || '0')

    const TrendIcon = change > 0 ? TrendingUp : change < 0 ? TrendingDown : Minus

    // 使用 CSS 变量实现主题感知
    const isPositive = change > 0
    const isNegative = change < 0
    const contribPositive = contribution > 0
    const contribNegative = contribution < 0

    return (
        <tr className="border-b border-[var(--card-border)] hover:bg-[var(--card-bg)] transition-colors">
            <td className="py-3 px-2">
                <div className="flex items-center gap-2">
                    <span className="text-xs text-theme-muted w-4">{index + 1}</span>
                    <span className="font-medium text-theme-primary">{holding.stock_name}</span>
                </div>
            </td>
            <td className="text-right py-3 px-2 text-theme-secondary font-mono text-sm">
                {holding.stock_code}
            </td>
            <td className="text-right py-3 px-2 text-theme-secondary">
                {formatRatio(holding.holding_ratio)}
            </td>
            <td className="text-right py-3 px-2 text-theme-primary font-mono">
                {formatCurrency(holding.current_price).replace('¥', '')}
            </td>
            <td className={cn(
                'text-right py-3 px-2 font-medium',
                isPositive && 'text-up',
                isNegative && 'text-down',
                !isPositive && !isNegative && 'text-theme-muted'
            )}>
                <div className="flex items-center justify-end gap-1">
                    <TrendIcon className="w-3 h-3" />
                    {changeInfo.text}
                </div>
            </td>
            <td className={cn(
                'text-right py-3 px-2 font-medium',
                contribPositive && 'text-up',
                contribNegative && 'text-down',
                !contribPositive && !contribNegative && 'text-theme-muted'
            )}>
                {contribution >= 0 ? '+' : ''}{contribution.toFixed(4)}%
            </td>
        </tr>
    )
}
