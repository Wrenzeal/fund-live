'use client'

import { useEffect, useId, useRef, useState } from 'react'
import { cn } from '@/lib/utils'

type AnimatedScoreGaugeProps = {
  value?: string | number
  label?: string
  variant?: 'hero' | 'summary'
  className?: string
}

function parseScore(value?: string | number) {
  const parsed = typeof value === 'number' ? value : Number.parseFloat(value || '')
  return Number.isFinite(parsed) ? parsed : 0
}

function formatScore(value?: string | number) {
  const parsed = typeof value === 'number' ? value : Number.parseFloat(value || '')
  return Number.isFinite(parsed) ? parsed.toFixed(1) : '--'
}

function scoreTone(score: number) {
  if (score >= 75) {
    return {
      label: '强势',
      from: '#fb7185',
      via: '#d946ef',
      to: '#22d3ee',
      glow: 'rgba(217,70,239,.22)',
      text: 'text-fuchsia-100',
      chip: 'border-fuchsia-400/25 bg-fuchsia-500/10',
    }
  }

  if (score >= 55) {
    return {
      label: '均衡',
      from: '#38bdf8',
      via: '#22d3ee',
      to: '#14b8a6',
      glow: 'rgba(34,211,238,.2)',
      text: 'text-cyan-100',
      chip: 'border-cyan-400/25 bg-cyan-500/10',
    }
  }

  return {
    label: '谨慎',
    from: '#fbbf24',
    via: '#fb7185',
    to: '#a78bfa',
    glow: 'rgba(251,191,36,.18)',
    text: 'text-amber-100',
    chip: 'border-amber-400/25 bg-amber-500/10',
  }
}

export function AnimatedScoreGauge({
  value,
  label = 'SCORE',
  variant = 'hero',
  className,
}: AnimatedScoreGaugeProps) {
  const [rootRef, hasEnteredView] = useGaugeReveal<HTMLDivElement>()
  const [draw, setDraw] = useState(false)
  const gradientId = useId().replace(/:/g, '')
  const score = parseScore(value)
  const normalized = Math.max(0, Math.min(score, 100))
  const displayValue = formatScore(value)
  const tone = scoreTone(normalized)
  const isHero = variant === 'hero'
  const center = 112
  const radius = isHero ? 78 : 74
  const strokeWidth = isHero ? 18 : 20
  const circumference = 2 * Math.PI * radius
  const activeLength = circumference * (normalized / 100)
  const endAngle = -Math.PI / 2 + (normalized / 100) * Math.PI * 2
  const endPoint = {
    x: center + Math.cos(endAngle) * radius,
    y: center + Math.sin(endAngle) * radius,
  }
  const tickMarks = Array.from({ length: 32 }, (_, index) => {
    const angle = -Math.PI / 2 + (index * Math.PI * 2) / 32
    const outer = isHero ? 101 : 100
    const inner = index % 4 === 0 ? 94 : 97
    return {
      key: index,
      x1: center + Math.cos(angle) * inner,
      y1: center + Math.sin(angle) * inner,
      x2: center + Math.cos(angle) * outer,
      y2: center + Math.sin(angle) * outer,
      strong: index % 4 === 0,
    }
  })

  useEffect(() => {
    if (!hasEnteredView) {
      return
    }

    const frame = window.requestAnimationFrame(() => setDraw(true))
    return () => window.cancelAnimationFrame(frame)
  }, [hasEnteredView])

  return (
    <div
      ref={rootRef}
      className={cn(
        'relative mx-auto flex items-center justify-center',
        isHero ? 'h-64 w-64' : 'h-36 w-36',
        className
      )}
      aria-label={`量化总分 ${displayValue} 分`}
      role="img"
    >
      <div
        className={cn(
          'absolute inset-0 rounded-full blur-2xl transition-opacity duration-700',
          draw ? 'opacity-100' : 'opacity-20'
        )}
        style={{ background: `radial-gradient(circle, ${tone.glow}, transparent 66%)` }}
      />
      <div className="absolute inset-3 rounded-full border border-white/10 bg-[var(--card-bg)]/15 shadow-[inset_0_0_46px_rgba(255,255,255,0.04)]" />
      <div className={cn('absolute inset-1 rounded-full border border-cyan-300/10', isHero && 'animate-[spin_22s_linear_infinite]')} />
      <div className={cn('absolute inset-7 rounded-full border border-dashed border-white/10', isHero && 'animate-[spin_28s_linear_infinite_reverse]')} />

      <svg viewBox="0 0 224 224" className="relative h-full w-full -rotate-90 drop-shadow-[0_18px_38px_rgba(34,211,238,0.14)]">
        <defs>
          <linearGradient id={`${gradientId}-score`} x1="20%" y1="0%" x2="90%" y2="100%">
            <stop offset="0%" stopColor={tone.from} />
            <stop offset="52%" stopColor={tone.via} />
            <stop offset="100%" stopColor={tone.to} />
          </linearGradient>
        </defs>

        {tickMarks.map((tick) => (
          <line
            key={tick.key}
            x1={tick.x1}
            y1={tick.y1}
            x2={tick.x2}
            y2={tick.y2}
            stroke={tick.strong ? 'rgba(226,232,240,.3)' : 'rgba(148,163,184,.16)'}
            strokeWidth={tick.strong ? 1.6 : 1}
            strokeLinecap="round"
          />
        ))}

        <circle
          cx={center}
          cy={center}
          r={radius}
          fill="none"
          stroke="rgba(148,163,184,.14)"
          strokeWidth={strokeWidth}
        />
        <circle
          cx={center}
          cy={center}
          r={radius}
          fill="none"
          stroke={`url(#${gradientId}-score)`}
          strokeWidth={strokeWidth}
          strokeLinecap="round"
          strokeDasharray={`${draw ? activeLength : 0} ${circumference}`}
          className="transition-[stroke-dasharray] duration-1000 ease-out"
        />
        <circle
          cx={endPoint.x}
          cy={endPoint.y}
          r={isHero ? 5 : 4.5}
          fill={tone.to}
          className={cn('transition-opacity delay-700 duration-500', draw ? 'opacity-100' : 'opacity-0')}
        />
      </svg>

      <div className="absolute inset-0 flex items-center justify-center">
        <div className={cn(
          'flex flex-col items-center justify-center rounded-full border border-[var(--card-border)] bg-[var(--card-bg)]/88 text-center shadow-[0_18px_44px_rgba(0,0,0,0.18)] backdrop-blur',
          isHero ? 'h-36 w-36' : 'h-20 w-20'
        )}>
          <div className={cn('tracking-[0.22em] text-theme-muted', isHero ? 'text-[10px]' : 'text-[8px]')}>
            {label}
          </div>
          <div className={cn('font-black leading-none text-theme-primary', isHero ? 'mt-2 text-6xl' : 'mt-1 text-3xl')}>
            {displayValue}
          </div>
          <div className={cn('mt-1 font-medium text-theme-muted', isHero ? 'text-xs' : 'text-[9px]')}>
            / 100
          </div>
          <div className={cn(
            'mt-2 rounded-full border px-2 py-0.5 font-semibold',
            tone.chip,
            tone.text,
            isHero ? 'text-[11px]' : 'text-[9px]'
          )}>
            {tone.label}
          </div>
        </div>
      </div>
    </div>
  )
}

function useGaugeReveal<T extends HTMLElement>() {
  const ref = useRef<T | null>(null)
  const [hasEnteredView, setHasEnteredView] = useState(false)

  useEffect(() => {
    if (hasEnteredView) {
      return
    }

    const element = ref.current
    if (!element) {
      return
    }

    if (typeof IntersectionObserver === 'undefined') {
      const frame = window.requestAnimationFrame(() => setHasEnteredView(true))
      return () => window.cancelAnimationFrame(frame)
    }

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting || entry.intersectionRatio > 0) {
          setHasEnteredView(true)
          observer.disconnect()
        }
      },
      {
        rootMargin: '0px 0px -10% 0px',
        threshold: 0.22,
      }
    )

    observer.observe(element)
    return () => observer.disconnect()
  }, [hasEnteredView])

  return [ref, hasEnteredView] as const
}
