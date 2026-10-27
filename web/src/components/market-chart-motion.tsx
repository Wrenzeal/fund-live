'use client'

import { useCallback, useEffect, useId, useMemo, useRef, useState, useSyncExternalStore, type ReactNode } from 'react'

const reducedMotionQuery = '(prefers-reduced-motion: reduce)'
function subscribeMotion(callback: () => void) {
  const query = window.matchMedia(reducedMotionQuery)
  query.addEventListener('change', callback)
  return () => query.removeEventListener('change', callback)
}
const getReducedMotion = () => window.matchMedia(reducedMotionQuery).matches
const getServerMotion = () => true

// API refresh timestamps can change while every plotted value stays the same.
// Retain equal data references so Recharts does not animate an unchanged snapshot.
export function useStableChartData<T>(data: T[]): T[] {
  const signature = JSON.stringify(data)
  return useMemo(() => JSON.parse(signature) as T[], [signature])
}

type ChartMotion = {
  isAnimationActive: boolean
  animationDuration: number
  animationEasing: 'ease-in-out'
  onAnimationStart: () => void
  onAnimationEnd: () => void
}

export function MarketChartFrame({ label, children }: { label: string; children: (motion: ChartMotion) => ReactNode }) {
  const chartId = useId()
  const frame = useRef<HTMLDivElement>(null)
  const [entered, setEntered] = useState(false)
  const reduced = useSyncExternalStore(subscribeMotion, getReducedMotion, getServerMotion)
  // Animation callbacks must not rerender Recharts: that can restart its geometry animation.
  const start = useCallback(() => { const element = document.getElementById(chartId); if (element && !getReducedMotion()) element.dataset.motionState = 'drawing' }, [chartId])
  const end = useCallback(() => { const element = document.getElementById(chartId); if (element) element.dataset.motionState = getReducedMotion() ? 'reduced' : 'settled' }, [chartId])
  useEffect(() => {
    const element = frame.current
    if (!element) return
    if (!('IntersectionObserver' in window)) {
      const handle = requestAnimationFrame(() => setEntered(true))
      return () => cancelAnimationFrame(handle)
    }
    const observer = new IntersectionObserver(entries => {
      if (entries.some(entry => entry.isIntersecting)) { setEntered(true); observer.disconnect() }
    }, { rootMargin: '48px', threshold: 0.05 })
    observer.observe(element)
    return () => observer.disconnect()
  }, [])
  const motion = useMemo<ChartMotion>(() => ({
    isAnimationActive: !reduced,
    animationDuration: 560,
    animationEasing: 'ease-in-out',
    onAnimationStart: start,
    onAnimationEnd: end,
  }), [reduced, start, end])
  return <div id={chartId} ref={frame} className="research-chart-frame h-72 w-full" aria-label={label} data-motion-state={!entered ? 'waiting' : reduced ? 'reduced' : 'settled'}>
    {entered ? <div className="research-chart-surface h-full w-full">{children(motion)}</div> : <div className="research-chart-placeholder h-full w-full" aria-hidden="true" />}
  </div>
}

export function MarketChartLoading({ loading, text }: { loading?: boolean; text: string }) {
  return <div role="status" className={`relative flex h-72 items-center justify-center overflow-hidden rounded-md text-sm text-theme-muted ${loading ? 'research-chart-placeholder skeleton-block' : ''}`}>
    <span className="relative z-10 rounded-md bg-[var(--background)]/80 px-3 py-2">{loading ? '正在读取走势…' : text}</span>
  </div>
}

export const marketChartTooltip = {
  contentStyle: { background: 'var(--background)', border: '1px solid var(--card-border)', borderRadius: 10, color: 'var(--text-primary)', fontSize: 12, boxShadow: '0 6px 24px rgb(0 0 0 / 8%)', padding: '10px 14px' },
  isAnimationActive: false,
  cursor: { stroke: 'var(--text-muted)', strokeDasharray: '3 4', strokeOpacity: 0.5 },
}

export function MarketChartEndDot({ cx, cy, index, lastIndex, color }: { cx?: number; cy?: number; index?: number; lastIndex: number; color: string }) {
  if (index !== lastIndex || cx === undefined || cy === undefined || !Number.isFinite(cx) || !Number.isFinite(cy)) return null
  return <circle className="research-chart-end-dot" cx={cx} cy={cy} r={3} fill="var(--background)" stroke={color} strokeWidth={1.8} />
}
