'use client'

import { useId } from 'react'
import { ResponsiveContainer, AreaChart, Area, Tooltip, YAxis } from 'recharts'
import type { TimeSeriesPoint } from '@/hooks/use-fund-data'
import { cn } from '@/lib/utils'

interface FundMiniTrendProps {
  timeSeries: TimeSeriesPoint[]
  isPositive: boolean
  isCallAuction?: boolean
  isHistorical?: boolean
  displayDate?: string
  isLoading?: boolean
  className?: string
}

interface MiniTrendTooltipProps {
  active?: boolean
  payload?: Array<{
    payload: {
      timestamp: string
      value: number
    }
  }>
}

function MiniTrendTooltip({ active, payload }: MiniTrendTooltipProps) {
  if (!active || !payload?.length) {
    return null
  }

  const value = payload[0]?.payload?.value
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return null
  }

  const isUp = value >= 0
  const label = `${isUp ? '+' : ''}${value.toFixed(2)}%`
  const timestamp = payload[0]?.payload?.timestamp
  const timeLabel = timestamp ? new Date(timestamp).toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
  }) : ''

  return (
    <div
      className={cn(
        'rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/95 px-2.5 py-1.5 text-[11px] font-semibold shadow-[0_12px_28px_rgba(2,8,23,0.22)] backdrop-blur-md',
        isUp
          ? 'text-up'
          : 'text-down'
      )}
    >
      {timeLabel && <div className="mb-0.5 font-medium text-theme-muted">{timeLabel}</div>}
      <div>{label}</div>
    </div>
  )
}

export function FundMiniTrend({
  timeSeries,
  isPositive,
  isCallAuction = false,
  isHistorical = false,
  displayDate = '',
  isLoading = false,
  className,
}: FundMiniTrendProps) {
  const data = timeSeries
    .map((point) => {
      const value = Number.parseFloat(point.change_percent)
      if (!Number.isFinite(value)) {
        return null
      }
      return {
        timestamp: point.timestamp,
        value,
      }
    })
    .filter((point): point is { timestamp: string; value: number } => point !== null)

  const latestValue = data.at(-1)?.value
  const resolvedPositive = typeof latestValue === 'number' ? latestValue >= 0 : isPositive
  const reactID = useId().replace(/:/g, '')
  const gradientID = `mini-trend-${reactID}-${resolvedPositive ? 'up' : 'down'}`
  const stroke = resolvedPositive ? 'var(--accent-up)' : 'var(--accent-down)'
  const minValue = data.length > 0 ? Math.min(...data.map((point) => point.value)) : 0
  const maxValue = data.length > 0 ? Math.max(...data.map((point) => point.value)) : 0
  const span = Math.max(maxValue - minValue, 0.08)
  const padding = span * 0.28
  const domainMin = minValue - padding
  const domainMax = maxValue + padding
  const dateLabel = displayDate ? displayDate.slice(5) : ''
  const title = isHistorical ? '上一交易日分时' : '今日分时'

  if (isCallAuction) {
    return (
      <div className={cn('flex h-24 items-center justify-center rounded-2xl border border-dashed border-[var(--card-border)] bg-[var(--input-bg)]/35 px-4 text-center', className)}>
        <div className="text-xs text-theme-muted">集合竞价中</div>
      </div>
    )
  }

  if (data.length === 0) {
    return (
      <div className={cn('relative flex h-24 items-center justify-center overflow-hidden rounded-2xl border border-dashed border-[var(--card-border)] bg-[var(--input-bg)]/25 px-4 text-center text-xs text-theme-muted', className)}>
        {isLoading && <div className="pointer-events-none absolute inset-y-0 left-0 w-1/2 animate-[history-scan_1.8s_ease-in-out_infinite] bg-gradient-to-r from-transparent via-cyan-400/10 to-transparent" />}
        <div className="relative">
          <div className="font-mono tracking-[0.16em]">{isLoading ? '[ INTRADAY_SYNC_PENDING ]' : '[ INTRADAY_NOT_READY ]'}</div>
          <div className="mt-1">分时数据暂未生成</div>
        </div>
      </div>
    )
  }

  return (
    <div className={cn('relative h-24 w-full overflow-hidden rounded-2xl border border-[var(--card-border)] bg-[var(--input-bg)]/25 px-3 py-2', className)}>
      <div className="mb-1 flex items-center justify-between gap-2 text-[10px]">
        <span className="font-semibold text-theme-secondary">{title}</span>
        {dateLabel && <span className="font-mono text-theme-muted">{dateLabel}</span>}
      </div>
      <div className="h-[4.4rem] w-full">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={data} margin={{ top: 4, right: 2, bottom: 0, left: 2 }}>
          <defs>
            <linearGradient id={gradientID} x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={stroke} stopOpacity={0.35} />
              <stop offset="95%" stopColor={stroke} stopOpacity={0.02} />
            </linearGradient>
          </defs>
          <YAxis hide domain={[domainMin, domainMax]} />
          <Tooltip
            cursor={false}
            content={<MiniTrendTooltip />}
            wrapperStyle={{ outline: 'none' }}
            offset={10}
          />
          <Area
            type="monotone"
            dataKey="value"
            stroke={stroke}
            fill={`url(#${gradientID})`}
            strokeWidth={2}
            dot={false}
            activeDot={{
              r: 4,
              fill: stroke,
              stroke: 'rgba(255,255,255,0.9)',
              strokeWidth: 1.5,
            }}
            isAnimationActive={false}
          />
        </AreaChart>
      </ResponsiveContainer>
      </div>
    </div>
  )
}
