'use client'

import { ResponsiveContainer, AreaChart, Area, Tooltip } from 'recharts'
import type { TimeSeriesPoint } from '@/hooks/use-fund-data'
import { cn } from '@/lib/utils'

interface FundMiniTrendProps {
  timeSeries: TimeSeriesPoint[]
  isPositive: boolean
  isCallAuction?: boolean
}

interface MiniTrendTooltipProps {
  active?: boolean
  payload?: Array<{
    payload: {
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

  return (
    <div
      className={cn(
        'rounded-full border px-2.5 py-1 text-[11px] font-semibold shadow-[0_12px_28px_rgba(2,8,23,0.34)] backdrop-blur-md',
        isUp
          ? 'border-rose-400/35 bg-rose-500/18 text-rose-50'
          : 'border-emerald-400/35 bg-emerald-500/18 text-emerald-50'
      )}
    >
      {label}
    </div>
  )
}

export function FundMiniTrend({ timeSeries, isPositive, isCallAuction = false }: FundMiniTrendProps) {
  const data = timeSeries.map((point) => ({
    value: Number.parseFloat(point.change_percent),
  }))

  const gradientID = isPositive ? 'mini-trend-up' : 'mini-trend-down'
  const stroke = isPositive ? 'var(--accent-up)' : 'var(--accent-down)'

  if (isCallAuction) {
    return (
      <div className="flex h-20 items-center justify-center rounded-2xl border border-dashed border-[var(--card-border)] bg-[var(--input-bg)]/35 px-4 text-center">
        <div className="text-xs text-theme-muted">集合竞价中</div>
      </div>
    )
  }

  if (data.length === 0) {
    return (
      <div className="flex h-20 items-center justify-center rounded-2xl border border-dashed border-[var(--card-border)] text-xs text-theme-muted">
        暂无走势
      </div>
    )
  }

  return (
    <div className="h-20 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={data}>
          <defs>
            <linearGradient id={gradientID} x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={stroke} stopOpacity={0.35} />
              <stop offset="95%" stopColor={stroke} stopOpacity={0.02} />
            </linearGradient>
          </defs>
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
  )
}
