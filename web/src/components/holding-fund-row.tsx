'use client'

import { useState } from 'react'
import { Trash2 } from 'lucide-react'
import { useMarketTradingState } from '@/hooks/use-market-status'
import { useFund, useFundEstimate } from '@/hooks/use-fund-data'
import { cn } from '@/lib/utils'
import type { HoldingEntry } from '@/hooks/use-user-portfolio'

interface HoldingFundRowProps {
  holding: HoldingEntry
  onRemove: () => Promise<void> | void
}

function formatAmount(amount: string) {
  const value = Number.parseFloat(amount)
  if (Number.isNaN(value)) {
    return '¥0.00'
  }
  return new Intl.NumberFormat('zh-CN', {
    style: 'currency',
    currency: 'CNY',
    maximumFractionDigits: 2,
  }).format(value)
}

function formatEstimatedDelta(amount: string, changePercent?: string) {
  const amountNumber = Number.parseFloat(amount)
  const percentNumber = Number.parseFloat(changePercent || '0')
  if (Number.isNaN(amountNumber) || Number.isNaN(percentNumber)) {
    return { text: '¥0.00', isPositive: false }
  }

  const delta = amountNumber * percentNumber / 100
  const isPositive = delta >= 0
  return {
    text: `${isPositive ? '+' : ''}${new Intl.NumberFormat('zh-CN', {
      style: 'currency',
      currency: 'CNY',
      maximumFractionDigits: 2,
    }).format(delta)}`,
    isPositive,
  }
}

function formatTradeAt(tradeAt?: string) {
  if (!tradeAt) {
    return ''
  }

  const parsed = new Date(tradeAt)
  if (Number.isNaN(parsed.getTime())) {
    return tradeAt
  }

  const formatter = new Intl.DateTimeFormat('zh-CN', {
    timeZone: 'Asia/Shanghai',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  })

  const parts = formatter.formatToParts(parsed)
  const values = Object.fromEntries(parts.map((part) => [part.type, part.value]))
  const dateLabel = `${values.month}/${values.day}`
  const timeLabel = `${values.hour}:${values.minute}`

  if (timeLabel === '14:59') {
    return `${dateLabel} 15:00前`
  }

  if (timeLabel === '15:01') {
    return `${dateLabel} 15:00后`
  }

  return formatter.format(parsed)
}

export function HoldingFundRow({ holding, onRemove }: HoldingFundRowProps) {
  const [isRemoving, setIsRemoving] = useState(false)
  const { session } = useMarketTradingState()
  const isCallAuction = session === 'call_auction'
  const { estimate } = useFundEstimate(isCallAuction ? null : holding.fund_id)
  const { fund } = useFund(holding.fund_id)
  const actualDailyReturn = holding.actual_daily_return?.trim() || ''
  const effectiveChangePercent = actualDailyReturn || estimate?.change_percent
  const delta = isCallAuction ? { text: '-', isPositive: false } : formatEstimatedDelta(holding.amount, effectiveChangePercent)
  const fundName = holding.fund?.name || fund?.name || estimate?.fund_name || holding.fund_id
  const tradeAtLabel = formatTradeAt(holding.trade_at)
  const isActualized = actualDailyReturn !== ''

  const handleRemove = async () => {
    if (isRemoving) {
      return
    }

    setIsRemoving(true)

    try {
      await new Promise((resolve) => window.setTimeout(resolve, 180))
      await Promise.resolve(onRemove())
    } catch (error) {
      console.error('Failed to remove holding', error)
      setIsRemoving(false)
    }
  }

  return (
    <div className="grid gap-4 rounded-[28px] border border-[var(--card-border)] p-5 glass lg:grid-cols-[1.4fr_0.9fr_0.9fr_0.7fr_auto] lg:items-center">
      <div className="min-w-0">
        <div className="truncate text-base font-semibold text-theme-primary">{fundName}</div>
        <div className="mt-1 text-xs text-theme-muted">{holding.fund_id}</div>
        {holding.note && <div className="mt-2 text-xs text-theme-secondary">{holding.note}</div>}
      </div>

      <div>
        <div className="text-xs text-theme-muted">持仓金额</div>
        <div className="mt-1 text-lg font-semibold text-theme-primary">{formatAmount(holding.amount)}</div>
      </div>

      <div>
        <div className="text-xs text-theme-muted">{isActualized ? '最新实际涨跌额' : '实时预估涨跌额'}</div>
        <div className={cn('mt-1 text-lg font-semibold', delta.isPositive ? 'text-up' : 'text-down')}>
          {delta.text}
        </div>
        {isActualized && holding.actual_date && (
          <div className="mt-1 text-xs text-theme-muted">已按 {holding.actual_date} 官方净值结算</div>
        )}
      </div>

      <div>
        <div className="text-xs text-theme-muted">确认净值日</div>
        <div className="mt-1 text-sm font-medium text-theme-primary">{holding.as_of_date}</div>
        {tradeAtLabel && <div className="mt-1 text-xs text-theme-muted">提交于 {tradeAtLabel}</div>}
      </div>

      <button
        type="button"
        onClick={() => void handleRemove()}
        disabled={isRemoving}
        className={cn(
          'group relative inline-flex items-center justify-center overflow-hidden rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] p-2 text-theme-muted transition-all duration-200',
          'hover:-translate-y-0.5 hover:border-rose-400/50 hover:bg-rose-500/12 hover:text-rose-300',
          'active:scale-95 disabled:cursor-not-allowed',
          isRemoving && 'holding-delete-button border-rose-400/50 bg-rose-500/16 text-rose-200'
        )}
        aria-label={`移除 ${fundName} 持仓`}
        aria-busy={isRemoving}
      >
        <span
          className={cn(
            'pointer-events-none absolute inset-0 rounded-xl bg-rose-400/0 opacity-0 transition-opacity duration-200',
            'group-hover:opacity-100',
            isRemoving && 'opacity-100'
          )}
        />
        <Trash2
          className={cn(
            'relative z-10 h-4 w-4 transition-transform duration-300',
            'group-hover:-rotate-12 group-hover:scale-110',
            isRemoving && 'holding-delete-icon'
          )}
        />
      </button>
    </div>
  )
}
