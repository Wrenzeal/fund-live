'use client'

import Link from 'next/link'
import { ArrowUpRight, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { useMarketTradingState } from '@/hooks/use-market-status'
import { useFund, useFundEstimate, useTimeSeries } from '@/hooks/use-fund-data'
import { FundMiniTrend } from '@/components/fund-mini-trend'
import { cn, formatPercent } from '@/lib/utils'

interface WatchlistFundCardProps {
  fundId: string
  onRemove: () => Promise<void> | void
}

export function WatchlistFundCard({ fundId, onRemove }: WatchlistFundCardProps) {
  const [isRemoving, setIsRemoving] = useState(false)
  const { session } = useMarketTradingState()
  const isCallAuction = session === 'call_auction'
  const { estimate, isLoading } = useFundEstimate(isCallAuction ? null : fundId)
  const { fund } = useFund(fundId)
  const { timeSeries } = useTimeSeries(isCallAuction ? null : fundId)

  const fundName = fund?.name || estimate?.fund_name || fundId
  const percent = isCallAuction ? { text: '-', isPositive: false } : formatPercent(estimate?.change_percent)

  const handleRemove = async () => {
    if (isRemoving) {
      return
    }

    setIsRemoving(true)

    try {
      await new Promise((resolve) => window.setTimeout(resolve, 180))
      await Promise.resolve(onRemove())
    } catch (error) {
      console.error('Failed to remove watchlist fund', error)
      setIsRemoving(false)
    }
  }

  return (
    <div className="rounded-[28px] border border-[var(--card-border)] p-5 glass">
      <div className="mb-4 flex items-start justify-between gap-4">
        <div className="min-w-0">
          <div className="truncate text-base font-semibold text-theme-primary">{fundName}</div>
          <div className="mt-1 text-xs text-theme-muted">{fundId}</div>
        </div>

        <button
          type="button"
          onClick={() => void handleRemove()}
          disabled={isRemoving}
          className={cn(
            'group relative overflow-hidden rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] p-2 text-theme-muted transition-all duration-200',
            'hover:-translate-y-0.5 hover:border-rose-400/45 hover:bg-rose-500/12 hover:text-rose-200 active:scale-[0.95]',
            'disabled:cursor-not-allowed disabled:opacity-80',
            isRemoving && 'danger-button-pop border-rose-400/45 bg-rose-500/14 text-rose-100'
          )}
          aria-label={`从自选中移除 ${fundName}`}
        >
          <span className="action-button-shine" />
          <Trash2
            className={cn(
              'relative z-10 h-4 w-4 transition-transform duration-300 group-hover:-rotate-12 group-hover:scale-110',
              isRemoving && 'danger-button-icon'
            )}
          />
        </button>
      </div>

      <FundMiniTrend timeSeries={timeSeries} isPositive={percent.isPositive} isCallAuction={isCallAuction} />

      <div className="mt-4 flex items-end justify-between gap-4">
        <div>
          <div className="text-xs text-theme-muted">实时预估涨跌幅</div>
          <div className={cn('mt-1 text-2xl font-black', percent.isPositive ? 'text-up' : 'text-down')}>
            {isCallAuction ? '-' : isLoading ? '--' : percent.text}
          </div>
        </div>

        <Link
          href={`/?fund=${fundId}`}
          className={cn(
            'group relative inline-flex items-center gap-1.5 overflow-hidden rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2 text-xs text-theme-secondary transition-all duration-200',
            'hover:-translate-y-0.5 hover:border-cyan-400/40 hover:bg-cyan-400/10 hover:text-theme-primary hover:shadow-[0_12px_24px_rgba(34,211,238,0.10)] active:scale-[0.97]'
          )}
        >
          <span className="action-button-shine" />
          <span className="relative z-10">查看详情</span>
          <ArrowUpRight className="relative z-10 h-3.5 w-3.5 transition-transform duration-300 group-hover:translate-x-0.5 group-hover:-translate-y-0.5" />
        </Link>
      </div>
    </div>
  )
}
