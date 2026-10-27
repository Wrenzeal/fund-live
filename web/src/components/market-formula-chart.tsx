'use client'

import { Line, LineChart, ReferenceLine, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import { MarketChartFrame, marketChartTooltip, useStableChartData } from './market-chart-motion'
import type { WorkspaceCurve, WorkspaceResult } from '@/lib/market-workspace-types'

const COLORS = ['var(--accent-primary)', '#e6a23c', '#c97db8']

export function MarketFormulaChart({ result, normalized = false, curves = [] }: { result: WorkspaceResult; normalized?: boolean; curves?: WorkspaceCurve[] }) {
  const ids = curves.length ? curves.map(curve => curve.id) : [...new Set(result.rows.flatMap(row => Object.keys(row.values)))].slice(0, 3)
  const baseRow = result.rows.find(row => ids.every(id => typeof row.values[id] === 'number' && Number.isFinite(row.values[id]) && row.values[id] > 0))
  const rows = useStableChartData(result.rows.map(row => {
    const point: Record<string, string | number | null | boolean> = { date: row.date, signal: row.signal }
    ids.forEach(id => {
      const value = row.values[id]
      const base = baseRow?.values[id]
      point[id] = typeof value !== 'number' || !Number.isFinite(value) ? null : normalized ? typeof base === 'number' && base !== 0 ? value / base * 100 : null : value
    })
    return point
  }))
  if (!rows.length || !ids.length) return <div role="status" className="flex h-72 items-center justify-center text-sm text-theme-muted">暂无公式结果</div>
  if (normalized && !baseRow) return <div role="status" className="flex h-72 items-center justify-center px-4 text-center text-sm text-theme-muted">没有所有曲线同时有效且大于零的共同日期，无法归一化；含零或负值的相对强弱公式适合原始值显示。</div>
  const names = ids.map(id => curves.find(curve => curve.id === id)?.name || id)
  return <div className="space-y-2">
    <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-theme-secondary">
      {names.map((name, index) => <span key={ids[index]} className="inline-flex items-center gap-2"><span className="h-0.5 w-4" style={{ background: COLORS[index] }} />{name}</span>)}
      <span className="text-theme-muted">{normalized ? `共同日期${baseRow?.date}基期100` : '原始公式值'}</span>
    </div>
    <MarketChartFrame label={normalized ? '公式归一化走势' : '公式原始走势'}>{motion => (
      <ResponsiveContainer width="100%" height="100%" minWidth={0} initialDimension={{ width: 640, height: 288 }}>
        <LineChart data={rows} margin={{ top: 12, right: 12, left: 0, bottom: 4 }}>
          <XAxis dataKey="date" tick={{ fill: 'var(--text-muted)', fontSize: 11 }} tickFormatter={(v: string) => v.slice(5)} minTickGap={35} tickLine={false} axisLine={false} />
          <YAxis width={58} domain={['auto', 'auto']} tick={{ fill: 'var(--text-muted)', fontSize: 11 }} tickLine={false} axisLine={false} />
          <Tooltip {...marketChartTooltip} formatter={value => typeof value === 'number' ? value.toFixed(3) : value} />
          {normalized && <ReferenceLine y={100} stroke="var(--text-muted)" strokeDasharray="4 4" />}
          {ids.map((id, index) => <Line key={id} name={names[index]} dataKey={id} stroke={COLORS[index]} strokeWidth={2} dot={(props: { cx?: number; cy?: number; payload?: { signal?: boolean } }) => props.payload?.signal && props.cx !== undefined && props.cy !== undefined ? <circle cx={props.cx} cy={props.cy} r={4} fill={COLORS[index]} stroke="var(--background)" strokeWidth={1} /> : null} connectNulls={false} {...motion} strokeLinecap="round" strokeLinejoin="round" animateNewValues={false} activeDot={{ r: 4, strokeWidth: 2, fill: 'var(--background)' }} />)}
          {rows.filter(row => row.signal).map(row => <ReferenceLine key={String(row.date)} x={String(row.date)} stroke="var(--accent-up)" strokeDasharray="3 4" />)}
        </LineChart>
      </ResponsiveContainer>
    )}</MarketChartFrame>
  </div>
}
