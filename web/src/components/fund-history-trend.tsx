'use client'

import { useId } from 'react'
import { Area, AreaChart, CartesianGrid, ReferenceLine, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import type { FundHistoryPoint } from '@/hooks/use-fund-data'
import { cn } from '@/lib/utils'

interface FundHistoryTrendProps {
  points: FundHistoryPoint[]
  days?: number
  compact?: boolean
  isLoading?: boolean
  className?: string
  title?: string
  description?: string
}

interface TrendPoint {
  date: string
  shortDate: string
  nav: number
  dailyReturn: number
}

interface HistoryTrendTooltipProps {
  active?: boolean
  payload?: Array<{
    payload: TrendPoint
  }>
}

function parseDecimal(value?: string) {
  if (!value) {
    return null
  }
  const parsed = Number.parseFloat(value)
  return Number.isFinite(parsed) ? parsed : null
}

function buildTrendData(points: FundHistoryPoint[]): TrendPoint[] {
  return points
    .map((point) => {
      const nav = parseDecimal(point.net_asset_val)
      if (nav === null) {
        return null
      }
      return {
        date: point.date,
        shortDate: point.date.slice(5),
        nav,
        dailyReturn: parseDecimal(point.daily_return) ?? 0,
      }
    })
    .filter((point): point is TrendPoint => point !== null)
}

function formatNav(value?: number) {
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    return '--'
  }
  return value.toFixed(4)
}

function formatPercent(value?: number) {
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    return '--'
  }
  return `${value >= 0 ? '+' : ''}${value.toFixed(2)}%`
}

function HistoryTrendTooltip({ active, payload }: HistoryTrendTooltipProps) {
  if (!active || !payload?.length) {
    return null
  }

  const point = payload[0]?.payload
  if (!point) {
    return null
  }

  const isUp = point.dailyReturn >= 0

  return (
    <div className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/95 px-3 py-2 text-xs shadow-[0_18px_40px_rgba(2,8,23,0.26)] backdrop-blur-xl">
      <div className="font-semibold text-theme-primary">{point.date}</div>
      <div className="mt-1 text-theme-secondary">净值 {formatNav(point.nav)}</div>
      <div className={cn('mt-1 font-semibold', isUp ? 'text-up' : 'text-down')}>
        日涨跌 {formatPercent(point.dailyReturn)}
      </div>
    </div>
  )
}

export function FundHistoryTrend({
  points,
  days = 30,
  compact = false,
  isLoading = false,
  className,
  title,
  description,
}: FundHistoryTrendProps) {
  const data = buildTrendData(points)
  const first = data[0]
  const last = data[data.length - 1]
  const change = first && last && first.nav !== 0 ? ((last.nav - first.nav) / first.nav) * 100 : 0
  const isPositive = change >= 0
  const stroke = isPositive ? 'var(--accent-up)' : 'var(--accent-down)'
  const reactID = useId().replace(/:/g, '')
  const gradientID = `fund-history-trend-${reactID}-${compact ? 'compact' : 'full'}-${isPositive ? 'up' : 'down'}`
  const minNav = data.length > 0 ? Math.min(...data.map((point) => point.nav)) : 0
  const maxNav = data.length > 0 ? Math.max(...data.map((point) => point.nav)) : 0
  const domainPadding = Math.max((maxNav - minNav) * 0.18, 0.002)

  if (data.length === 0) {
    return (
      <div className={cn('relative overflow-hidden rounded-3xl border border-dashed border-[var(--card-border)] bg-[var(--input-bg)]/35', compact ? 'h-20' : 'min-h-[16rem] p-5', className)}>
        <div className="pointer-events-none absolute inset-0 opacity-60 [background-image:linear-gradient(rgba(148,163,184,0.10)_1px,transparent_1px),linear-gradient(90deg,rgba(148,163,184,0.10)_1px,transparent_1px)] [background-size:18px_18px]" />
        <div role="status" className="relative flex h-full min-h-[inherit] items-center justify-center px-4 text-center text-sm text-theme-muted">
          <div>
            <div>{isLoading ? '历史净值正在同步' : `暂时没有近 ${days} 日官方净值`}</div>
            {!compact && <div className="mt-2">{isLoading ? '同步完成后会显示走势。' : '官方净值公布后会自动补上。'}</div>}
          </div>
        </div>
      </div>
    )
  }

  return (
    <section className={cn('relative overflow-hidden rounded-3xl border border-[var(--card-border)] bg-[var(--card-bg)]/40', compact ? 'h-20 p-0' : 'p-5 md:p-6', className)}>
      {!compact && (
        <div className="mb-5 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <div className="text-sm font-semibold text-theme-primary">{title || `近 ${days} 日官方净值走势`}</div>
            <div className="mt-1 text-xs leading-5 text-theme-muted">
              {description || '只展示每日官方收盘净值，不与盘中估值混算。'}
            </div>
          </div>
          <div className="rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2 text-right">
            <div className="text-[10px] tracking-[0.18em] text-theme-muted">区间变化</div>
            <div className={cn('mt-1 text-lg font-black', isPositive ? 'text-up' : 'text-down')}>{formatPercent(change)}</div>
          </div>
        </div>
      )}

      <div className={cn('w-full', compact ? 'h-20' : 'h-56')}>
        <ResponsiveContainer width="100%" height="100%" minWidth={compact ? 120 : 240} minHeight={compact ? 72 : 180}>
          <AreaChart data={data} margin={compact ? { top: 10, right: 0, bottom: 0, left: 0 } : { top: 10, right: 8, bottom: 0, left: 0 }}>
            <defs>
              <linearGradient id={gradientID} x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor={stroke} stopOpacity={compact ? 0.30 : 0.36} />
                <stop offset="95%" stopColor={stroke} stopOpacity={0.02} />
              </linearGradient>
            </defs>
            {!compact && <CartesianGrid stroke="var(--card-border)" strokeDasharray="3 5" vertical={false} opacity={0.72} />}
            {!compact && (
              <XAxis
                dataKey="shortDate"
                tick={{ fill: 'var(--text-muted)', fontSize: 11 }}
                tickLine={false}
                axisLine={false}
                minTickGap={24}
              />
            )}
            {!compact && (
              <YAxis
                width={48}
                tick={{ fill: 'var(--text-muted)', fontSize: 11 }}
                tickLine={false}
                axisLine={false}
                domain={[minNav - domainPadding, maxNav + domainPadding]}
                tickFormatter={(value) => Number(value).toFixed(3)}
              />
            )}
            {!compact && first && <ReferenceLine y={first.nav} stroke="var(--text-muted)" strokeDasharray="4 6" opacity={0.42} />}
            <Tooltip
              cursor={!compact ? { stroke: 'var(--text-muted)', strokeDasharray: '3 5', opacity: 0.45 } : false}
              content={<HistoryTrendTooltip />}
              wrapperStyle={{ outline: 'none' }}
              offset={12}
            />
            <Area
              type="monotone"
              dataKey="nav"
              stroke={stroke}
              fill={`url(#${gradientID})`}
              strokeWidth={compact ? 2 : 2.4}
              dot={false}
              activeDot={{
                r: compact ? 3 : 4,
                fill: stroke,
                stroke: 'rgba(255,255,255,0.86)',
                strokeWidth: 1.5,
              }}
              isAnimationActive={!compact}
              animationDuration={700}
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>

      {!compact && (
        <div className="mt-4 grid gap-3 text-xs text-theme-secondary sm:grid-cols-3">
          <div className="rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2">
            <div className="text-theme-muted">起点净值</div>
            <div className="mt-1 font-semibold text-theme-primary">{formatNav(first?.nav)}</div>
          </div>
          <div className="rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2">
            <div className="text-theme-muted">最新净值</div>
            <div className="mt-1 font-semibold text-theme-primary">{formatNav(last?.nav)}</div>
          </div>
          <div className="rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2">
            <div className="text-theme-muted">最新日期</div>
            <div className="mt-1 font-semibold text-theme-primary">{last?.date || '--'}</div>
          </div>
        </div>
      )}
    </section>
  )
}
