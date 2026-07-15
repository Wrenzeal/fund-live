'use client'

import { formatRatio } from '@/lib/utils'
import type { FundHoldingRecord } from '@/hooks/use-fund-data'

interface TargetETFHoldingsCardProps {
  targetName?: string
  holdings: FundHoldingRecord[]
}

export function TargetETFHoldingsCard({ targetName, holdings }: TargetETFHoldingsCardProps) {
  const topHoldings = holdings.slice(0, 10)

  return (
    <div className="glass rounded-2xl p-6">
      <div className="mb-4 flex items-center justify-between gap-4">
        <div>
          <h3 className="text-lg font-semibold text-theme-primary">目标 ETF 持仓</h3>
          <p className="mt-1 text-xs text-theme-muted">
            {targetName ? `展示 ${targetName} 当前可获取的持仓明细` : '展示当前可获取的目标 ETF 持仓明细'}
          </p>
        </div>
        <span className="text-xs text-theme-muted">
          共 {holdings.length} 条
        </span>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full">
          <thead>
            <tr className="border-b border-[var(--card-border)] text-xs text-theme-muted">
              <th className="px-2 py-3 text-left">股票名称</th>
              <th className="px-2 py-3 text-right">代码</th>
              <th className="px-2 py-3 text-right">持仓占比</th>
              <th className="px-2 py-3 text-right">报告期</th>
            </tr>
          </thead>
          <tbody>
            {topHoldings.map((holding, index) => (
              <tr key={`${holding.stock_code}:${index}`} className="border-b border-[var(--card-border)] transition-colors hover:bg-[var(--card-bg)] last:border-b-0">
                <td className="px-2 py-3">
                  <div className="flex items-center gap-2">
                    <span className="w-4 text-xs text-theme-muted">{index + 1}</span>
                    <span className="font-medium text-theme-primary">{holding.stock_name}</span>
                  </div>
                </td>
                <td className="px-2 py-3 text-right font-mono text-sm text-theme-secondary">{holding.stock_code}</td>
                <td className="px-2 py-3 text-right text-theme-secondary">{formatRatio(holding.holding_ratio)}</td>
                <td className="px-2 py-3 text-right text-theme-muted">{holding.reporting_period || '--'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="mt-4 border-t border-[var(--card-border)] pt-4 text-sm leading-6 text-theme-muted">
        目标 ETF 持仓用于说明联接基金的跟踪对象和底层暴露，不代表联接基金的实时涨跌幅。
      </div>
    </div>
  )
}
