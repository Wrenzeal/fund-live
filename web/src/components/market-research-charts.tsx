'use client'

import { CartesianGrid, Line, LineChart, ReferenceLine, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import { MarketChartFrame, MarketChartLoading, MarketChartEndDot, marketChartTooltip, useStableChartData } from './market-chart-motion'
import type { ResearchBar, ResearchInstrumentDetail } from '@/hooks/use-market-research'

const COLORS = ['var(--accent-primary)', '#e6a23c', '#c97db8']

export function researchDayChartData(bars: ResearchBar[], limit: number) {
  const sorted = [...bars].sort((a, b) => a.date.localeCompare(b.date))
  return sorted.map((bar, index) => {
    const average = (period: number) => {
      if (index + 1 < period) return null
      const window = sorted.slice(index + 1 - period, index + 1)
      return window.reduce((sum, item) => sum + item.close, 0) / period
    }
    return { ...bar, ma5: average(5), ma20: average(20), ma60: average(60) }
  }).slice(-limit)
}

export function researchComparisonData(details: ResearchInstrumentDetail[], limit: number) {
  if (details.length < 2) return []
  const maps = details.map((detail) => new Map(detail.bars
    .filter((bar) => Number.isFinite(bar.close) && bar.close > 0)
    .map((bar) => [bar.date, bar.close])))
  const dates = [...maps[0].keys()].filter((date) => maps.every((map) => map.has(date))).sort().slice(-limit)
  if (dates.length < 2) return []
  const baselines = maps.map((map) => map.get(dates[0])!)
  return dates.map((date) => {
    const point: Record<string, string | number> = { date }
    details.forEach((detail, index) => { point[detail.instrument.symbol] = maps[index].get(date)! / baselines[index] * 100 })
    return point
  })
}

const chartTicks = { fill: 'var(--text-muted)', fontSize: 11 }
const initialChartDimension = { width: 640, height: 288 }

export function MarketDayChart({ bars, limit, loading }: { bars: ResearchBar[]; limit: number; loading?: boolean }) {
  const points = useStableChartData(researchDayChartData(bars, limit))
  if (!points.length) return <MarketChartLoading loading={loading} text="暂无日线数据" />
  return (
    <MarketChartFrame label={`最近 ${points.length} 个交易日日线及均线`}>{motion => (
      <ResponsiveContainer width="100%" height="100%" minWidth={0} initialDimension={initialChartDimension}>
        <LineChart data={points} margin={{ top: 12, right: 12, left: 0, bottom: 4 }}>
          <CartesianGrid stroke="var(--card-border)" vertical={false} strokeDasharray="3 5" />
          <XAxis dataKey="date" tick={chartTicks} tickFormatter={(value: string) => value.slice(5)} minTickGap={35} tickLine={false} axisLine={false} />
          <YAxis width={60} domain={['auto', 'auto']} tick={chartTicks} tickFormatter={(value: number) => value.toFixed(value < 10 ? 3 : 0)} tickLine={false} axisLine={false} />
          <Tooltip {...marketChartTooltip} formatter={(value) => typeof value === 'number' ? value.toFixed(3) : value} />
          <Line name="收盘价" dataKey="close" stroke="var(--accent-primary)" strokeWidth={2.2} dot={(props) => <MarketChartEndDot {...props} lastIndex={points.length - 1} color="var(--accent-primary)" />} {...motion} strokeLinecap="round" strokeLinejoin="round" animateNewValues={false} activeDot={{ r: 4, strokeWidth: 2, fill: 'var(--background)' }} />
          <Line name="MA5" dataKey="ma5" stroke="#e6a23c" strokeWidth={1.2} dot={false} {...motion} strokeLinecap="round" strokeLinejoin="round" animateNewValues={false} activeDot={{ r: 4, strokeWidth: 2, fill: 'var(--background)' }} />
          <Line name="MA20" dataKey="ma20" stroke="#c97db8" strokeWidth={1.2} dot={false} {...motion} strokeLinecap="round" strokeLinejoin="round" animateNewValues={false} activeDot={{ r: 4, strokeWidth: 2, fill: 'var(--background)' }} />
          <Line name="MA60" dataKey="ma60" stroke="#72bfa3" strokeWidth={1.2} dot={false} {...motion} strokeLinecap="round" strokeLinejoin="round" animateNewValues={false} activeDot={{ r: 4, strokeWidth: 2, fill: 'var(--background)' }} />
        </LineChart>
      </ResponsiveContainer>
    )}</MarketChartFrame>
  )
}

export function MarketMinuteChart({ minute, previousClose, loading }: { minute: ResearchInstrumentDetail['minute']; previousClose?: number; loading?: boolean }) {
  const points = useStableChartData(minute?.points || [])
  if (!minute?.points.length) return <MarketChartLoading loading={loading} text="暂无分时数据" />
  return (
    <MarketChartFrame label={`${minute.date} 分时走势`}>{motion => (
      <ResponsiveContainer width="100%" height="100%" minWidth={0} initialDimension={initialChartDimension}>
        <LineChart data={points} margin={{ top: 12, right: 12, left: 0, bottom: 4 }}>
          <CartesianGrid stroke="var(--card-border)" vertical={false} strokeDasharray="3 5" />
          <XAxis dataKey="time" tick={chartTicks} minTickGap={45} tickLine={false} axisLine={false} />
          <YAxis width={60} domain={['auto', 'auto']} tick={chartTicks} tickFormatter={(value: number) => value.toFixed(value < 10 ? 3 : 0)} tickLine={false} axisLine={false} />
          <Tooltip {...marketChartTooltip} labelFormatter={(label) => `${minute.date} ${label}`} formatter={(value) => typeof value === 'number' ? value.toFixed(3) : value} />
          {previousClose !== undefined && Number.isFinite(previousClose) && previousClose > 0 && <ReferenceLine y={previousClose} className="market-minute-previous-close" stroke="var(--text-muted)" strokeDasharray="4 4" />}
          <Line name="价格" dataKey="price" stroke="var(--accent-primary)" strokeWidth={2.2} dot={(props) => <MarketChartEndDot {...props} lastIndex={points.length - 1} color="var(--accent-primary)" />} {...motion} strokeLinecap="round" strokeLinejoin="round" animateNewValues={false} activeDot={{ r: 4, strokeWidth: 2, fill: 'var(--background)' }} />
        </LineChart>
      </ResponsiveContainer>
    )}</MarketChartFrame>
  )
}

export function MarketComparisonChart({ details, limit, loading }: { details: ResearchInstrumentDetail[]; limit: number; loading?: boolean }) {
  const points = useStableChartData(researchComparisonData(details, limit))
  if (!points.length) return <MarketChartLoading loading={loading} text="暂无足够的共同交易日数据" />
  return (
    <>
      <div className="mb-2 flex flex-wrap gap-x-5 gap-y-1 text-xs text-theme-secondary">
        {details.map((detail, index) => <span key={detail.instrument.symbol} className="inline-flex items-center gap-2"><span className="h-0.5 w-4" style={{ background: COLORS[index] }} />{detail.instrument.name}</span>)}
        <span className="text-theme-muted">{points[0].date} — {points[points.length - 1].date} · 基期100</span>
      </div>
      <MarketChartFrame label="共同交易日归一化走势对比">{motion => (
        <ResponsiveContainer width="100%" height="100%" minWidth={0} initialDimension={initialChartDimension}>
          <LineChart data={points} margin={{ top: 12, right: 12, left: 0, bottom: 4 }}>
            <CartesianGrid stroke="var(--card-border)" vertical={false} strokeDasharray="3 5" />
            <XAxis dataKey="date" tick={chartTicks} tickFormatter={(value: string) => value.slice(5)} minTickGap={35} tickLine={false} axisLine={false} />
            <YAxis width={52} domain={['auto', 'auto']} tick={chartTicks} tickFormatter={(value: number) => value.toFixed(1)} tickLine={false} axisLine={false} />
            <Tooltip {...marketChartTooltip} formatter={(value) => typeof value === 'number' ? value.toFixed(2) : value} />
            <ReferenceLine y={100} stroke="var(--text-muted)" strokeDasharray="4 4" />
            {details.map((detail, index) => <Line key={detail.instrument.symbol} name={detail.instrument.name} dataKey={detail.instrument.symbol} stroke={COLORS[index]} strokeWidth={2.2} dot={(props) => <MarketChartEndDot {...props} lastIndex={points.length - 1} color={COLORS[index]} />} {...motion} strokeLinecap="round" strokeLinejoin="round" animateNewValues={false} activeDot={{ r: 4, strokeWidth: 2, fill: 'var(--background)' }} />)}
          </LineChart>
        </ResponsiveContainer>
      )}</MarketChartFrame>
    </>
  )
}
