'use client'

import { useState } from 'react'
import { Trash2 } from 'lucide-react'
import { useMarketTradingState } from '@/hooks/use-market-status'
import { useFund, type FundEstimate } from '@/hooks/use-fund-data'
import { cn } from '@/lib/utils'
import type { HoldingEntry } from '@/hooks/use-user-portfolio'

interface HoldingFundRowProps {
  holding: HoldingEntry
  metricScope?: 'official' | 'estimate'
  estimate?: FundEstimate | null
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

function formatMetricCurrency(amount?: string) {
  if (!amount) {
    return '--'
  }

  const value = Number.parseFloat(amount)
  if (Number.isNaN(value)) {
    return '--'
  }

  return new Intl.NumberFormat('zh-CN', {
    style: 'currency',
    currency: 'CNY',
    maximumFractionDigits: 2,
  }).format(value)
}

function formatNumberCurrency(value?: number) {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return '--'
  }

  return new Intl.NumberFormat('zh-CN', {
    style: 'currency',
    currency: 'CNY',
    maximumFractionDigits: 2,
  }).format(value)
}

function formatPercentValue(value?: string) {
  if (!value) {
    return '--'
  }

  const parsed = Number.parseFloat(value)
  if (Number.isNaN(parsed)) {
    return '--'
  }

  return `${parsed >= 0 ? '+' : ''}${parsed.toFixed(2)}%`
}

function formatNumberPercent(value?: number) {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return '--'
  }

  return `${value >= 0 ? '+' : ''}${value.toFixed(2)}%`
}

function parseMetricNumber(value?: string) {
  if (!value) {
    return null
  }

  const parsed = Number.parseFloat(value)
  return Number.isNaN(parsed) ? null : parsed
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

export function HoldingFundRow({ holding, metricScope = 'official', estimate: providedEstimate, onRemove }: HoldingFundRowProps) {
  const [isRemoving, setIsRemoving] = useState(false)
  const { session } = useMarketTradingState()
  const isCallAuction = session === 'call_auction'
  const { fund } = useFund(holding.fund_id)
  const estimate = providedEstimate ?? null
  const fundName = holding.fund?.name || fund?.name || estimate?.fund_name || holding.fund_id
  const tradeAtLabel = formatTradeAt(holding.trade_at)
  const estimateDelta = isCallAuction ? { text: '-', isPositive: false } : formatEstimatedDelta(holding.amount, estimate?.change_percent)
  const confirmedDateLabel = holding.confirmed_nav_date || holding.as_of_date
  const sharesNumber = parseMetricNumber(holding.shares)
  const estimateNavNumber = parseMetricNumber(estimate?.estimate_nav)
  const prevNavNumber = parseMetricNumber(estimate?.prev_nav)
  const estimateChangePercentNumber = parseMetricNumber(estimate?.change_percent)
  const hasEstimatedHoldingMetrics = !isCallAuction &&
    typeof sharesNumber === 'number' &&
    sharesNumber > 0 &&
    typeof estimateNavNumber === 'number' &&
    typeof prevNavNumber === 'number' &&
    prevNavNumber > 0
  const estimatedCurrentMarketValue = hasEstimatedHoldingMetrics ? sharesNumber * estimateNavNumber : null
  const estimatedTodayProfit = hasEstimatedHoldingMetrics ? sharesNumber * (estimateNavNumber - prevNavNumber) : null
  const isOfficialScope = metricScope === 'official'
  const currentMarketValueLabel = isOfficialScope ? '最新官方市值' : '盘中预估市值'
  const profitLabel = isOfficialScope ? '今日官方盈亏' : '盘中预估盈亏'
  const changeLabel = isOfficialScope ? '今日官方涨跌幅' : '盘中预估涨跌幅'
  const currentMarketValueText = isOfficialScope
    ? (holding.real_metrics_ready ? formatMetricCurrency(holding.current_market_value) : '--')
    : formatNumberCurrency(estimatedCurrentMarketValue ?? undefined)
  const todayProfitText = isOfficialScope
    ? (holding.real_metrics_ready ? formatMetricCurrency(holding.today_profit) : '--')
    : hasEstimatedHoldingMetrics
      ? formatNumberCurrency(estimatedTodayProfit ?? undefined)
      : (!isCallAuction && estimate?.change_percent ? estimateDelta.text : '--')
  const todayChangePercentText = isOfficialScope
    ? (holding.real_metrics_ready ? formatPercentValue(holding.today_change_percent) : '--')
    : formatNumberPercent(estimateChangePercentNumber ?? undefined)
  const realMetricTone = (() => {
    if (isOfficialScope && holding.real_metrics_ready) {
      const profit = Number.parseFloat(holding.today_profit || '0')
      return profit >= 0 ? 'text-up' : 'text-down'
    }
    if (hasEstimatedHoldingMetrics && typeof estimatedTodayProfit === 'number') {
      return estimatedTodayProfit >= 0 ? 'text-up' : 'text-down'
    }
    if (!isCallAuction && estimate?.change_percent) {
      return estimateDelta.isPositive ? 'text-up' : 'text-down'
    }
    return 'text-theme-primary'
  })()
  const changeTone = (() => {
    if (isOfficialScope && holding.real_metrics_ready) {
      const change = Number.parseFloat(holding.today_change_percent || '0')
      return change >= 0 ? 'text-up' : 'text-down'
    }
    if (typeof estimateChangePercentNumber === 'number') {
      return estimateChangePercentNumber >= 0 ? 'text-up' : 'text-down'
    }
    return 'text-theme-primary'
  })()
  const marketValueNote = isOfficialScope
    ? holding.real_metrics_ready
      ? `${holding.actual_date || '最新'} 官方净值口径`
      : holding.real_metrics_message || '待今日官方净值同步后展示'
    : hasEstimatedHoldingMetrics
      ? `按 ${holding.shares || '--'} 份与盘中预估净值估算`
      : '待确认净值补齐后展示'
  const profitNote = isOfficialScope
    ? `${holding.actual_date || '最新'} 官方净值口径`
    : hasEstimatedHoldingMetrics
      ? '盘中预估，按确认份额与预估净值计算'
      : !isCallAuction && estimate?.change_percent
        ? `按本金口径估算 ${estimateDelta.text}`
        : '待确认份额补齐后展示'
  const officialProfitNote = holding.real_metrics_ready
    ? `已按 ${holding.actual_date || '最新'} 官方净值结算`
    : holding.real_metrics_message || '待官方净值同步后展示'
  const displayedProfitNote = isOfficialScope ? officialProfitNote : profitNote
  const changeNote = isOfficialScope
    ? (holding.real_metrics_ready ? '按最新官方涨跌幅展示' : '真实涨跌幅待同步')
    : estimate?.change_percent
      ? '盘中预估涨跌幅'
      : '待确认份额补齐后展示'

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
    <div className="grid gap-4 rounded-[28px] border border-[var(--card-border)] p-5 glass lg:grid-cols-[minmax(0,1.25fr)_0.8fr_0.85fr_0.85fr_0.7fr_0.7fr_auto] lg:items-center">
      <div className="min-w-0">
        <div className="truncate text-base font-semibold text-theme-primary">{fundName}</div>
        <div className="mt-1 text-xs text-theme-muted">{holding.fund_id}</div>
        {holding.note && <div className="mt-2 text-xs text-theme-secondary">{holding.note}</div>}
        {!holding.real_metrics_ready && (
          <div className="mt-2 text-xs text-theme-muted">
            {holding.real_metrics_message || '待真实净值同步后展示真实市值与盈亏。'}
          </div>
        )}
      </div>

      <div>
        <div className="text-xs text-theme-muted">持仓本金</div>
        <div className="mt-1 text-lg font-semibold text-theme-primary">{formatAmount(holding.amount)}</div>
      </div>

      <div>
        <div className="text-xs text-theme-muted">{currentMarketValueLabel}</div>
        <div className="mt-1 text-lg font-semibold text-theme-primary">
          {currentMarketValueText}
        </div>
        <div className="mt-1 text-xs text-theme-muted">{marketValueNote}</div>
      </div>

      <div>
        <div className="text-xs text-theme-muted">{profitLabel}</div>
        <div className={cn('mt-1 text-lg font-semibold', realMetricTone)}>
          {todayProfitText}
        </div>
        <div className={cn('mt-1 text-xs', isOfficialScope || !estimate?.change_percent ? 'text-theme-muted' : estimateDelta.isPositive ? 'text-up' : 'text-down')}>
          {displayedProfitNote}
        </div>
      </div>

      <div>
        <div className="text-xs text-theme-muted">{changeLabel}</div>
        <div className={cn('mt-1 text-lg font-semibold', changeTone)}>
          {todayChangePercentText}
        </div>
        <div className="mt-1 text-xs text-theme-muted">{changeNote}</div>
      </div>

      <div>
        <div className="text-xs text-theme-muted">确认净值日</div>
        <div className="mt-1 text-sm font-medium text-theme-primary">{confirmedDateLabel}</div>
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
