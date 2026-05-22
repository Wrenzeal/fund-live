'use client'

import { useEffect, useId, useRef, useState, type ReactNode } from 'react'
import Link from 'next/link'
import { AlertTriangle, ArrowLeft, BarChart3, CalendarClock, FileText, Gauge, Layers3, PieChart, ShieldCheck, ShieldAlert, Sparkles, Target, TimerReset, Zap } from 'lucide-react'
import { AnimatedScoreGauge } from '@/components/animated-score-gauge'
import { EstimateCard } from '@/components/estimate-card'
import { FundSectorCard } from '@/components/fund-sector-card'
import { HoldingsTable } from '@/components/holdings-table'
import { TargetETFHoldingsCard } from '@/components/target-etf-holdings-card'
import { useFundAnalysis, useFundDashboard, useFundHoldings, type FundAnalysis, type FundAnalysisEventImpact, type FundAnalysisModuleScore } from '@/hooks/use-fund-data'
import { cn } from '@/lib/utils'
import {
  confidenceLevelLabel,
  eventHorizonLabel,
  eventImpactLabel,
  eventImpactTone,
  eventStrengthLabel,
  eventTimelineRank,
  formatAnalysisPercent,
  formatAnalysisScore,
  parseAnalysisNumber,
  riskLevelLabel,
} from '@/lib/fund-analysis-display'

export function AnalysisBoardPageClient({ fundId }: { fundId: string }) {
  const {
    fund,
    estimate,
    sectorSnapshot,
    themeSnapshot,
    displayDate,
    isHistorical,
    officialClose,
    isLoading,
    isValidating,
  } = useFundDashboard(fundId)
  const {
    analysis,
    isLoading: isAnalysisLoading,
  } = useFundAnalysis(fundId)

  const {
    holdings: resolvedHoldings,
    displayItems,
    displayLevel,
    lookthroughAvailable,
  } = useFundHoldings(fundId)

  const lastUpdated = estimate?.calculated_at ? new Date(estimate.calculated_at) : null
  const timelineEvents = (analysis?.event_impacts || []).slice().sort((left, right) => eventTimelineRank(left) - eventTimelineRank(right))
  const quarterlyEvents = (analysis?.event_impacts || []).filter((item) => item.horizon === 'quarterly')
  const exposureEvents = (analysis?.event_impacts || []).filter((item) => item.target_scope === 'exposure')
  const riskModules = (analysis?.module_scores || []).filter((item) => isRiskModule(item.code))
  const positiveModules = (analysis?.module_scores || []).filter((item) => !isRiskModule(item.code))
  const recommendationItems = buildRecommendationItems(analysis)

  return (
    <main className="min-h-screen">
      <div className="container mx-auto px-4 py-6 md:py-8">
        <header className="mb-6 flex flex-col gap-4 rounded-3xl border border-[var(--card-border)] bg-[var(--card-bg)]/40 p-5 md:flex-row md:items-center md:justify-between">
          <div>
            <Link
              href={fundId ? `/?fund=${fundId}` : '/'}
              className="mb-3 inline-flex items-center gap-2 text-sm text-theme-secondary transition-colors hover:text-theme-primary"
            >
              <ArrowLeft className="h-4 w-4" />
              返回基金详情
            </Link>
            <div className="text-2xl font-bold text-theme-primary">完整量化看板</div>
            <div className="mt-2 text-sm text-theme-secondary">
              {fund?.name ? `${fund.name}（${fund.id}）` : `基金 ${fundId}`} ·
              {' '}用于承载更完整的结构、事件与风险拆解
            </div>
          </div>
        </header>

        <div className="grid items-stretch gap-4 md:grid-cols-2 xl:grid-cols-4">
          <OverviewCard
            icon={<BarChart3 className="h-4 w-4 text-cyan-200" />}
            title="当前分析口径"
            value={analysis?.analysis_basis || '--'}
            description={analysis?.analysis_type === 'tracked_etf' ? '联接基金优先复用目标 ETF 口径' : '当前量化分析以单快照规则打分为主'}
          />
          <OverviewCard
            icon={<ShieldCheck className="h-4 w-4 text-emerald-200" />}
            title="风险与覆盖"
            value={`${riskLevelLabel(analysis?.risk_level)} / ${confidenceLevelLabel(analysis?.confidence)}`}
            description={analysis?.latest_holding_period ? `最新持仓披露：${analysis.latest_holding_period}` : '当前未获取有效持仓披露'}
          />
          <OverviewCard
            icon={<Layers3 className="h-4 w-4 text-fuchsia-200" />}
            title="主线结构"
            value={themeSnapshot?.primary_theme_name || sectorSnapshot?.primary_sector_name || fund?.category_name || '--'}
            description={themeSnapshot?.primary_theme_name && sectorSnapshot?.primary_sector_name ? `${sectorSnapshot.primary_sector_name} / ${themeSnapshot.primary_theme_name}` : '主暴露结构会继续细化到季报变化层'}
          />
          <OverviewCard
            icon={<FileText className="h-4 w-4 text-amber-200" />}
            title="当前阶段"
            value="基础版"
            description="已覆盖总分、建议分布、事件影响、持仓分类；后续继续补季报变化与详细时间线"
          />
        </div>

        <div className="mt-6 space-y-6">
          <AnalysisHeroVisual
            analysis={analysis}
            isLoading={isAnalysisLoading}
            sectorName={sectorSnapshot?.primary_sector_name}
            themeName={themeSnapshot?.primary_theme_name}
          />

          <div className="grid items-stretch gap-6 xl:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)]">
            <RecommendationMixPanel items={recommendationItems} />
            <ModuleRadarPanel modules={analysis?.module_scores || []} />
          </div>

          <EvidenceFocusGrid analysis={analysis} />

          <div className="grid items-stretch gap-6 xl:grid-cols-[minmax(0,1.05fr)_minmax(22rem,0.95fr)]">
            <EventSignalBoard events={timelineEvents} />
            <QuarterlyDiffCard events={quarterlyEvents} />
          </div>

          <div className="grid items-stretch gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(20rem,0.95fr)]">
            <RiskBreakdownCard analysis={analysis} riskModules={riskModules} />
            <StructureComparisonCard
              sectorSnapshot={sectorSnapshot}
              themeSnapshot={themeSnapshot}
              exposureEvents={exposureEvents}
              positiveModules={positiveModules}
            />
          </div>

          <div className="grid items-stretch gap-6 xl:grid-cols-[repeat(auto-fit,minmax(20rem,1fr))]">
            <EstimateCard
              estimate={estimate}
              fund={fund}
              isLoading={isLoading}
              isValidating={isValidating}
              lastUpdated={lastUpdated}
              className="h-full"
            />
            <FundSectorCard
              fund={fund}
              sectorSnapshot={sectorSnapshot}
              themeSnapshot={themeSnapshot}
            />

            <section className="glass flex h-full flex-col rounded-3xl p-6">
              <div className="mb-4 text-sm font-semibold text-theme-primary">结构与数据视角</div>
              <div className="flex flex-1 flex-col justify-center space-y-3 text-sm text-theme-secondary">
                <MetaRow label="展示层级" value={displayLevel === 'target_layer' ? '下一层追踪目标' : '股票持仓层'} />
                <MetaRow label="可穿透估值" value={lookthroughAvailable ? '是' : '否'} />
                <MetaRow label="分时日期" value={displayDate || '--'} />
                <MetaRow label="历史数据" value={isHistorical ? '是' : '否'} />
                <MetaRow label="官方收盘结果" value={officialClose?.display_status === 'ready' ? '已就绪' : officialClose?.display_status === 'pending' ? '待同步' : '隐藏'} />
                <div className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-4 text-xs leading-6 text-theme-muted">
                  当前完整量化看板页主要负责把“评分 + 结构 + 事件 + 持仓”串起来。后续如果事件层继续扩展，这里会承载时间线、季报变化与更细的风险拆解。
                </div>
              </div>
            </section>
          </div>

          <HoldingsTable
            estimate={estimate}
            displayLevel={displayLevel}
            items={displayItems}
            lookthroughAvailable={lookthroughAvailable}
          />

          {displayLevel === 'target_layer' && resolvedHoldings.length > 0 && (
            <TargetETFHoldingsCard
              targetName={displayItems[0]?.name}
              holdings={resolvedHoldings}
            />
          )}

          <section className="glass rounded-3xl p-6">
            <div className="mb-4 text-sm font-semibold text-theme-primary">方法与限制</div>
            <div className="grid gap-4 xl:grid-cols-2">
              <div className="rounded-2xl border border-cyan-500/20 bg-cyan-500/10 p-4">
                <div className="mb-2 text-sm font-medium text-cyan-100">当前方法</div>
                <div className="space-y-2 text-sm text-theme-secondary">
                  <p>1. 先基于 dashboard 单快照统一估值、分时、分类与持仓口径。</p>
                  <p>2. 再按趋势 / 结构 / 热度 / 风险 / 性价比 / 事件六模块打分。</p>
                  <p>3. 最后把综合偏向映射成加 / 平 / 减分布，而不是直接给单句买卖结论。</p>
                </div>
              </div>
              <div className="rounded-2xl border border-amber-500/20 bg-amber-500/10 p-4">
                <div className="mb-2 text-sm font-medium text-amber-100">当前限制</div>
                <div className="space-y-2 text-sm text-theme-secondary">
                  <p>1. 事件模块还未接外部新闻、政策、财报与基金公告数据源。</p>
                  <p>2. 六模块权重仍在样本池校准阶段，部分 ETF 与低覆盖主动基金还需继续调参。</p>
                  <p>3. 目前以详情页展示为主，后续才会扩展到持仓页、自选页与排行榜。</p>
                </div>
              </div>
            </div>
          </section>
        </div>
      </div>
    </main>
  )
}

type RecommendationItem = {
  code: 'increase' | 'hold' | 'decrease'
  label: string
  value: number
  gradient: string
  tone: string
  colors: {
    from: string
    via: string
    to: string
  }
}

function buildRecommendationItems(analysis?: FundAnalysis): RecommendationItem[] {
  return [
    {
      code: 'increase',
      label: '结构偏积极',
      value: parseAnalysisNumber(analysis?.increase_percent),
      gradient: 'from-rose-500 via-fuchsia-500 to-pink-500',
      tone: 'border-rose-500/20 bg-rose-500/10 text-rose-100',
      colors: {
        from: '#fb7185',
        via: '#d946ef',
        to: '#ec4899',
      },
    },
    {
      code: 'hold',
      label: '适合观察',
      value: parseAnalysisNumber(analysis?.hold_percent),
      gradient: 'from-slate-500 via-cyan-400 to-sky-400',
      tone: 'border-cyan-500/20 bg-cyan-500/10 text-cyan-100',
      colors: {
        from: '#64748b',
        via: '#22d3ee',
        to: '#38bdf8',
      },
    },
    {
      code: 'decrease',
      label: '风险偏高',
      value: parseAnalysisNumber(analysis?.decrease_percent),
      gradient: 'from-emerald-500 via-teal-500 to-cyan-500',
      tone: 'border-emerald-500/20 bg-emerald-500/10 text-emerald-100',
      colors: {
        from: '#10b981',
        via: '#14b8a6',
        to: '#06b6d4',
      },
    },
  ]
}

function dominantRecommendationItem(items: RecommendationItem[]) {
  const increase = items.find((item) => item.code === 'increase')
  const hold = items.find((item) => item.code === 'hold')
  const decrease = items.find((item) => item.code === 'decrease')

  if ((increase?.value || 0) >= 55) {
    return increase || items[0]
  }
  if ((decrease?.value || 0) >= 60) {
    return decrease || items[0]
  }
  return hold || items[0]
}

function AnalysisHeroVisual({
  analysis,
  isLoading,
  sectorName,
  themeName,
}: {
  analysis?: FundAnalysis
  isLoading?: boolean
  sectorName?: string
  themeName?: string
}) {
  if (!analysis) {
    return (
      <section className="glass rounded-3xl p-5 sm:p-6">
        <div className="flex min-h-[18rem] items-center justify-center rounded-3xl border border-dashed border-[var(--card-border)] text-sm text-theme-muted">
          {isLoading ? '量化看板加载中...' : '暂无量化分析数据'}
        </div>
      </section>
    )
  }

  const recommendationItems = buildRecommendationItems(analysis)
  const dominant = dominantRecommendationItem(recommendationItems)
  const primary = analysis.primary_evidence?.[0]
  const counter = analysis.counter_evidence?.[0]

  return (
    <section className="glass overflow-hidden rounded-3xl p-5 sm:p-6">
      <div className="relative h-full">
        <div className="pointer-events-none absolute -right-12 -top-16 h-48 w-48 rounded-full bg-cyan-500/20 blur-3xl" />
        <div className="pointer-events-none absolute -bottom-16 left-16 h-40 w-40 rounded-full bg-fuchsia-500/20 blur-3xl" />

        <div className="relative grid items-stretch gap-5 xl:grid-cols-[17rem_minmax(0,1fr)_19rem]">
          <div className="flex min-h-[17rem] items-center justify-center">
            <AnimatedScoreGauge value={analysis.total_score} label="TOTAL SCORE" variant="hero" />
          </div>

          <div className="flex min-h-[17rem] flex-col justify-center">
            <div className="mb-3 flex flex-wrap gap-2">
              <span className={cn('rounded-full border px-3 py-1 text-xs', dominant.tone)}>{dominant.label}</span>
              <span className="rounded-full border border-[var(--card-border)] bg-[var(--input-bg)]/60 px-3 py-1 text-xs text-theme-secondary">
                {analysis.analysis_basis}
              </span>
              <span className="rounded-full border border-cyan-500/20 bg-cyan-500/10 px-3 py-1 text-xs text-cyan-100">
                {confidenceLevelLabel(analysis.confidence)}
              </span>
            </div>
            <h1 className="text-2xl font-black leading-tight text-theme-primary md:text-3xl">
              {analysis.summary}
            </h1>
            <p className="mt-3 max-w-3xl text-sm leading-7 text-theme-secondary">
              看板优先展示“结论 → 主证据 → 风险限制 → 事件链路”。规则明细仍保留在下方，但不再让用户先面对完整判定列表。
            </p>

            <div className="mt-5 grid items-stretch gap-3 sm:grid-cols-2">
              <HeroEvidencePill
                title="主证据"
                value={primary?.title || '主证据待补充'}
                detail={primary?.summary || '当前证据链有限，建议结合持仓和行情继续观察。'}
                icon={<Target className="h-4 w-4 text-cyan-200" />}
              />
              <HeroEvidencePill
                title="风险限制"
                value={counter?.title || '反方证据待补充'}
                detail={counter?.summary || '暂未识别到明显反方证据，但仍需控制仓位风险。'}
                icon={<ShieldAlert className="h-4 w-4 text-amber-200" />}
                amber
              />
            </div>
          </div>

          <div className="flex min-h-[17rem] flex-col justify-center rounded-[2rem] border border-[var(--card-border)] bg-[var(--card-bg)]/40 p-5">
            <div className="mb-4 flex items-center gap-2 text-sm font-semibold text-theme-primary">
              <Gauge className="h-4 w-4 text-fuchsia-200" />
              结构定位
            </div>
            <div className="space-y-3">
              <MetaRow label="风险等级" value={riskLevelLabel(analysis.risk_level)} />
              <MetaRow label="主行业" value={sectorName || '--'} />
              <MetaRow label="主主题" value={themeName || '--'} />
              <MetaRow label="披露期" value={analysis.latest_holding_period || '--'} />
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}

function HeroEvidencePill({
  icon,
  title,
  value,
  detail,
  amber = false,
}: {
  icon: ReactNode
  title: string
  value: string
  detail: string
  amber?: boolean
}) {
  return (
    <div className={cn('flex h-full min-h-[8.5rem] flex-col justify-center rounded-2xl border p-4', amber ? 'border-amber-500/20 bg-amber-500/10' : 'border-cyan-500/20 bg-cyan-500/10')}>
      <div className="mb-2 flex items-center gap-2 text-xs font-semibold text-theme-primary">
        {icon}
        {title}
      </div>
      <div className="text-sm font-semibold text-theme-primary">{value}</div>
      <div className="mt-2 line-clamp-3 text-xs leading-5 text-theme-secondary">{detail}</div>
    </div>
  )
}

function RecommendationMixPanel({ items }: { items: RecommendationItem[] }) {
  const [panelRef, hasEnteredView] = useLazyReveal<HTMLElement>()
  const total = items.reduce((sum, item) => sum + Math.max(item.value, 0), 0)

  return (
    <section ref={panelRef} className="glass relative flex h-full min-h-[24rem] flex-col overflow-hidden rounded-3xl p-6">
      <div className="pointer-events-none absolute -left-20 top-1/2 h-56 w-56 -translate-y-1/2 rounded-full bg-fuchsia-500/10 blur-3xl" />
      <div className="pointer-events-none absolute -right-16 top-8 h-48 w-48 rounded-full bg-cyan-500/10 blur-3xl" />

      <div className="relative mb-5 flex items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 text-sm font-semibold text-theme-primary">
            <PieChart className="h-4 w-4 text-cyan-200" />
            建议分布
          </div>
          <div className="mt-1 text-xs text-theme-muted">滚动到此处后再绘制环形分布；比例只表示结构偏向，不是交易指令。</div>
        </div>
        <span className="hidden rounded-full border border-cyan-500/20 bg-cyan-500/10 px-3 py-1 text-xs text-cyan-100 sm:inline-flex">
          视口触发动画
        </span>
      </div>

      {total <= 0 ? (
        <div className="relative flex flex-1 items-center">
          <EmptyPanel text="当前没有可绘制的建议分布数据。" />
        </div>
      ) : (
        <div className="relative grid flex-1 items-center gap-6 lg:grid-cols-[18rem_minmax(0,1fr)]">
          <div className="flex min-h-[18rem] items-center justify-center">
            {hasEnteredView ? (
              <AnimatedRecommendationDonut items={items} />
            ) : (
              <div className="relative flex h-64 w-64 items-center justify-center rounded-full border border-dashed border-[var(--card-border)] bg-[var(--card-bg)]/30 text-center">
                <div className="absolute inset-6 rounded-full border border-[var(--card-border)]/70" />
                <div className="px-8 text-xs leading-6 text-theme-muted">
                  到达图表位置后
                  <br />
                  开始绘制建议分布
                </div>
              </div>
            )}
          </div>

          <div className="space-y-3">
            {items.map((item, index) => {
              const normalized = total > 0 ? (Math.max(item.value, 0) / total) * 100 : 0
              return (
                <div
                  key={item.code}
                  className={cn(
                    'rounded-2xl border p-4 transition-all duration-700',
                    item.tone,
                    hasEnteredView ? 'translate-y-0 opacity-100' : 'translate-y-3 opacity-0'
                  )}
                  style={{ transitionDelay: `${index * 120 + 180}ms` }}
                >
                  <div className="mb-3 flex items-center justify-between gap-3">
                    <div className="flex items-center gap-2">
                      <span
                        className="h-2.5 w-2.5 rounded-full shadow-[0_0_18px_currentColor]"
                        style={{ backgroundColor: item.colors.via }}
                      />
                      <span className="text-sm font-semibold text-theme-primary">{item.label}</span>
                    </div>
                    <span className="text-sm font-black text-theme-primary">{formatAnalysisPercent(item.value)}</span>
                  </div>
                  <div className="h-2 overflow-hidden rounded-full bg-[var(--input-bg)]">
                    <div
                      className={cn('h-full rounded-full bg-gradient-to-r transition-all duration-1000 ease-out', item.gradient)}
                      style={{
                        width: hasEnteredView ? `${normalized}%` : '0%',
                        transitionDelay: `${index * 120 + 320}ms`,
                      }}
                    />
                  </div>
                  <div className="mt-2 text-[11px] text-theme-muted">
                    环形占比：{formatAnalysisPercent(normalized)}
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      )}
    </section>
  )
}

function AnimatedRecommendationDonut({ items }: { items: RecommendationItem[] }) {
  const gradientId = useId().replace(/:/g, '')
  const [draw, setDraw] = useState(false)
  const center = 112
  const radius = 80
  const circumference = 2 * Math.PI * radius
  const total = items.reduce((sum, item) => sum + Math.max(item.value, 0), 0)
  const dominant = dominantRecommendationItem(items)
  const segments = items.reduce<Array<{
    item: RecommendationItem
    index: number
    dashLength: number
    dashOffset: number
    gap: number
    nextOffset: number
  }>>((result, item, index) => {
    const previousOffset = result.at(-1)?.nextOffset || 0
    const ratio = total > 0 ? Math.max(item.value, 0) / total : 0
    const dashLength = circumference * ratio

    return [
      ...result,
      {
        item,
        index,
        dashLength,
        dashOffset: -previousOffset,
        gap: ratio > 0.03 ? 3.8 : 0,
        nextOffset: previousOffset + dashLength,
      },
    ]
  }, [])

  useEffect(() => {
    const frame = window.requestAnimationFrame(() => setDraw(true))
    return () => window.cancelAnimationFrame(frame)
  }, [])

  return (
    <div className={cn('relative h-72 w-72 transition-all duration-700', draw ? 'scale-100 opacity-100' : 'scale-95 opacity-0')}>
      <div className="absolute inset-0 rounded-full bg-[conic-gradient(from_140deg,rgba(244,63,94,.22),rgba(34,211,238,.18),rgba(16,185,129,.2),rgba(244,63,94,.22))] blur-2xl" />
      <div className="absolute inset-5 rounded-full border border-white/10 bg-[var(--card-bg)]/20 shadow-[inset_0_0_50px_rgba(255,255,255,0.04)]" />
      <div className="absolute inset-3 rounded-full border border-cyan-300/10 animate-[spin_18s_linear_infinite]" />

      <svg viewBox="0 0 224 224" className="relative h-full w-full -rotate-90 drop-shadow-[0_18px_36px_rgba(34,211,238,0.14)]" role="img" aria-label="量化建议分布环形图">
        <defs>
          {items.map((item) => (
            <linearGradient key={item.code} id={`${gradientId}-${item.code}`} x1="30%" y1="0%" x2="80%" y2="100%">
              <stop offset="0%" stopColor={item.colors.from} />
              <stop offset="55%" stopColor={item.colors.via} />
              <stop offset="100%" stopColor={item.colors.to} />
            </linearGradient>
          ))}
        </defs>
        <circle
          cx={center}
          cy={center}
          r={radius}
          fill="none"
          stroke="rgba(148,163,184,.14)"
          strokeWidth="24"
        />
        {segments.map(({ item, index, dashLength, dashOffset, gap }) => {
          return (
            <circle
              key={item.code}
              cx={center}
              cy={center}
              r={radius}
              fill="none"
              stroke={`url(#${gradientId}-${item.code})`}
              strokeWidth="24"
              strokeLinecap="round"
              strokeDasharray={`${draw ? Math.max(dashLength - gap, 0) : 0} ${circumference}`}
              strokeDashoffset={dashOffset}
              className="transition-[stroke-dasharray] duration-1000 ease-out"
              style={{ transitionDelay: `${index * 160}ms` }}
            />
          )
        })}
      </svg>

      <div className="absolute inset-0 flex items-center justify-center">
        <div className="flex h-36 w-36 flex-col items-center justify-center rounded-full border border-[var(--card-border)] bg-[var(--card-bg)]/85 text-center shadow-[0_18px_44px_rgba(0,0,0,0.18)] backdrop-blur">
          <div className="text-[10px] tracking-[0.24em] text-theme-muted">主方向</div>
          <div className="mt-2 max-w-24 text-lg font-black leading-tight text-theme-primary">{dominant?.label || '--'}</div>
          <div className="mt-1 text-sm font-semibold text-cyan-100">{formatAnalysisPercent(dominant?.value || 0)}</div>
        </div>
      </div>
    </div>
  )
}

function useLazyReveal<T extends HTMLElement>() {
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
        rootMargin: '0px 0px -12% 0px',
        threshold: 0.28,
      }
    )

    observer.observe(element)
    return () => observer.disconnect()
  }, [hasEnteredView])

  return [ref, hasEnteredView] as const
}

function ModuleRadarPanel({ modules }: { modules: FundAnalysisModuleScore[] }) {
  const normalized = modules.slice(0, 6)
  const center = 96
  const radius = 68
  const points = normalized.map((module, index) => {
    const angle = -Math.PI / 2 + (index * Math.PI * 2) / Math.max(normalized.length, 1)
    const valueRadius = radius * (parseAnalysisNumber(module.score) / 100)
    return `${center + Math.cos(angle) * valueRadius},${center + Math.sin(angle) * valueRadius}`
  }).join(' ')
  const gridLevels = [0.33, 0.66, 1]

  return (
    <section className="glass flex h-full min-h-[18rem] flex-col justify-center rounded-3xl p-6">
      <div className="mb-4 flex items-center gap-2 text-sm font-semibold text-theme-primary">
        <Zap className="h-4 w-4 text-fuchsia-200" />
        六维模块雷达
      </div>
      {normalized.length === 0 ? (
        <EmptyPanel text="当前没有可展示的模块分数。" />
      ) : (
      <div className="grid items-center gap-5 md:grid-cols-[13rem_minmax(0,1fr)]">
        <div className="relative mx-auto h-52 w-52">
          <svg viewBox="0 0 192 192" className="h-full w-full">
            {gridLevels.map((level) => (
              <circle key={level} cx={center} cy={center} r={radius * level} fill="none" stroke="rgba(148,163,184,.22)" strokeWidth="1" />
            ))}
            {normalized.map((module, index) => {
              const angle = -Math.PI / 2 + (index * Math.PI * 2) / Math.max(normalized.length, 1)
              return (
                <line
                  key={module.code}
                  x1={center}
                  y1={center}
                  x2={center + Math.cos(angle) * radius}
                  y2={center + Math.sin(angle) * radius}
                  stroke="rgba(148,163,184,.18)"
                  strokeWidth="1"
                />
              )
            })}
            {points && <polygon points={points} fill="rgba(34,211,238,.22)" stroke="rgb(34,211,238)" strokeWidth="2" />}
          </svg>
        </div>

        <div className="space-y-3">
          {normalized.map((module) => {
            const score = parseAnalysisNumber(module.score)
            return (
              <div key={module.code} className="grid grid-cols-[4.5rem_minmax(0,1fr)_3rem] items-center gap-3 text-xs">
                <span className="font-medium text-theme-secondary">{module.name}</span>
                <div className="h-2.5 overflow-hidden rounded-full bg-[var(--input-bg)]">
                  <div
                    className="h-full rounded-full bg-gradient-to-r from-cyan-500 via-sky-400 to-fuchsia-400 transition-all duration-700"
                    style={{ width: `${Math.max(score, 0)}%` }}
                  />
                </div>
                <span className="text-right font-semibold text-theme-primary">{formatAnalysisScore(module.score)}</span>
              </div>
            )
          })}
        </div>
      </div>
      )}
    </section>
  )
}

function EvidenceFocusGrid({ analysis }: { analysis?: FundAnalysis }) {
  const primary = analysis?.primary_evidence || []
  const counter = analysis?.counter_evidence || []
  const ai = analysis?.ai_explanation

  return (
    <section className="grid items-stretch gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_22rem]">
      <EvidenceColumn title="主证据" icon={<Target className="h-4 w-4 text-cyan-200" />} items={primary} />
      <EvidenceColumn title="反方证据 / 限制" icon={<ShieldAlert className="h-4 w-4 text-amber-200" />} items={counter} amber />
      <div className="glass flex h-full min-h-[18rem] flex-col rounded-3xl p-6">
        <div className="mb-4 flex items-center gap-2 text-sm font-semibold text-theme-primary">
          <Sparkles className="h-4 w-4 text-fuchsia-200" />
          AI解释层
        </div>
        {ai ? (
          <div className="flex flex-1 flex-col justify-center space-y-3">
            <div className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-4">
              <div className="text-xs text-theme-muted">状态 / 缓存</div>
              <div className="mt-2 text-sm font-semibold text-theme-primary">
                {ai.status || '--'} · {ai.cache_status || '--'}
              </div>
            </div>
            <div className="rounded-2xl border border-fuchsia-500/20 bg-fuchsia-500/10 p-4 text-sm leading-6 text-theme-secondary">
              {ai.summary}
            </div>
            <div className="text-xs leading-5 text-theme-muted">
              {ai.boundary_notice}
            </div>
          </div>
        ) : (
          <div className="flex flex-1 items-center">
            <EmptyPanel text="当前没有 AI 解释层输出；规则评分与证据仍可独立查看。" />
          </div>
        )}
      </div>
    </section>
  )
}

function EvidenceColumn({
  title,
  icon,
  items,
  amber = false,
}: {
  title: string
  icon: ReactNode
  items: NonNullable<FundAnalysis['primary_evidence']>
  amber?: boolean
}) {
  return (
    <div className={cn('glass flex h-full min-h-[18rem] flex-col rounded-3xl p-6', amber ? 'bg-amber-500/5' : 'bg-cyan-500/5')}>
      <div className="mb-4 flex items-center gap-2 text-sm font-semibold text-theme-primary">
        {icon}
        {title}
      </div>
      <div className={cn('flex flex-1 flex-col gap-3', items.length === 0 && 'justify-center')}>
        {items.length === 0 ? (
          <EmptyPanel text={`${title} 暂无结构化数据。`} />
        ) : items.slice(0, 3).map((item, index) => (
          <div key={`${item.code}-${index}`} className={cn('min-h-[7.5rem] rounded-2xl border p-4 transition-transform duration-300 hover:-translate-y-0.5', amber ? 'border-amber-500/20 bg-amber-500/10' : 'border-cyan-500/20 bg-cyan-500/10')}>
            <div className="text-sm font-semibold text-theme-primary">{item.title}</div>
            <div className="mt-2 line-clamp-3 text-xs leading-5 text-theme-secondary">{item.summary}</div>
            <div className="mt-3 flex flex-wrap gap-2 text-[11px] text-theme-muted">
              {item.source_scope && <span>{item.source_scope}</span>}
              {item.strength && <span>强度：{eventStrengthLabel(item.strength)}</span>}
              {item.horizon && <span>{eventHorizonLabel(item.horizon)}</span>}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

function OverviewCard({
  icon,
  title,
  value,
  description,
}: {
  icon: ReactNode
  title: string
  value: string
  description: string
}) {
  return (
    <section className="glass flex h-full min-h-[8.5rem] flex-col justify-center rounded-2xl p-5">
      <div className="mb-3 flex items-center gap-2 text-theme-primary">
        {icon}
        <span className="text-sm font-medium">{title}</span>
      </div>
      <div className="text-lg font-semibold text-theme-primary">{value}</div>
      <div className="mt-2 text-xs leading-5 text-theme-muted">{description}</div>
    </section>
  )
}

function MetaRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex min-h-[3rem] items-center justify-between gap-4 rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 px-4 py-3">
      <span className="text-theme-muted">{label}</span>
      <span className="text-right font-medium text-theme-primary">{value}</span>
    </div>
  )
}

function isRiskModule(code: string) {
  return code === 'risk' || code === 'event'
}

function EventSignalBoard({ events }: { events: FundAnalysisEventImpact[] }) {
  return (
    <section className="glass flex h-full min-h-[22rem] flex-col rounded-3xl p-6">
      <div className="mb-5 flex items-center gap-3">
        <div className="rounded-2xl bg-cyan-500/15 p-3">
          <CalendarClock className="h-5 w-5 text-cyan-200" />
        </div>
        <div>
          <div className="text-sm font-semibold text-theme-primary">事件信号链</div>
          <div className="text-xs text-theme-muted">优先展示持仓、主线和近期事件，弱化纯规则说明。</div>
        </div>
      </div>

      <div className={cn('relative flex-1 space-y-4', events.length === 0 && 'flex items-center')}>
        {events.length > 0 && (
          <div className="absolute bottom-4 left-3 top-4 w-px bg-gradient-to-b from-cyan-400/0 via-cyan-400/40 to-cyan-400/0" />
        )}
        {events.length === 0 ? (
          <EmptyPanel text="当前还没有可展开的事件信号。" />
        ) : events.slice(0, 6).map((event, index) => (
          <div key={`${event.code}-${index}`} className="relative pl-8">
            <div className="absolute left-0 top-5 flex h-6 w-6 items-center justify-center rounded-full border border-cyan-400/30 bg-[var(--card-bg)] text-[10px] font-bold text-cyan-100">
              {index + 1}
            </div>
            <div className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-4 transition-transform duration-300 hover:-translate-y-0.5">
              <div className="flex flex-wrap items-center gap-2">
                <span className="rounded-full border border-[var(--input-border)] bg-[var(--input-bg)] px-2.5 py-1 text-[11px] text-theme-secondary">
                  {eventHorizonLabel(event.horizon)}
                </span>
                <span className="rounded-full border border-[var(--input-border)] bg-[var(--input-bg)] px-2.5 py-1 text-[11px] text-theme-secondary">
                  {eventStrengthLabel(event.strength)}
                </span>
                <span className={cn('rounded-full border px-2.5 py-1 text-[11px]', eventImpactTone(event.impact))}>
                  {eventImpactLabel(event.impact)}
                </span>
              </div>
              <div className="mt-3 text-sm font-semibold text-theme-primary">{event.title}</div>
              <div className="mt-2 line-clamp-3 text-sm leading-6 text-theme-secondary">{event.summary}</div>
              {event.related_symbols && event.related_symbols.length > 0 && (
                <div className="mt-2 text-xs text-theme-muted">
                  相关标的：{event.related_symbols.join(' / ')}
                </div>
              )}
            </div>
          </div>
        ))}
      </div>
    </section>
  )
}

function QuarterlyDiffCard({ events }: { events: FundAnalysisEventImpact[] }) {
  return (
    <section className="glass flex h-full min-h-[22rem] flex-col rounded-3xl p-6">
      <div className="mb-4 flex items-center gap-3">
        <div className="rounded-2xl bg-fuchsia-500/15 p-3">
          <TimerReset className="h-5 w-5 text-fuchsia-200" />
        </div>
        <div>
          <div className="text-sm font-semibold text-theme-primary">季报变化</div>
          <div className="text-xs text-theme-muted">把上一季与当前季的持仓和主线变化单独拎出来看。</div>
        </div>
      </div>

      <div className={cn('flex flex-1 flex-col gap-3', events.length === 0 && 'justify-center')}>
        {events.length === 0 ? (
          <EmptyPanel text="当前没有显著的季度结构变化事件。" />
        ) : events.map((event, index) => (
          <div key={`${event.code}-${index}`} className="rounded-2xl border border-fuchsia-500/20 bg-fuchsia-500/10 p-4">
            <div className="text-sm font-semibold text-theme-primary">{event.title}</div>
            <div className="mt-2 text-sm leading-6 text-theme-secondary">{event.summary}</div>
          </div>
        ))}
      </div>
    </section>
  )
}

function RiskBreakdownCard({
  analysis,
  riskModules,
}: {
  analysis?: FundAnalysis
  riskModules: FundAnalysisModuleScore[]
}) {
  return (
    <section className="glass flex h-full flex-col rounded-3xl p-6">
      <div className="mb-4 flex items-center gap-3">
        <div className="rounded-2xl bg-amber-500/15 p-3">
          <ShieldAlert className="h-5 w-5 text-amber-200" />
        </div>
        <div>
          <div className="text-sm font-semibold text-theme-primary">风险拆解</div>
          <div className="text-xs text-theme-muted">单独看风险相关模块、warning 与当前总风险等级。</div>
        </div>
      </div>

      <div className="grid flex-1 items-stretch gap-4 lg:grid-cols-[16rem_minmax(0,1fr)]">
        <div className="flex min-h-[11rem] flex-col justify-center rounded-2xl border border-amber-500/20 bg-amber-500/10 p-4">
          <div className="text-xs tracking-[0.18em] text-theme-muted">CURRENT RISK</div>
          <div className="mt-3 text-3xl font-black text-theme-primary">{riskLevelLabel(analysis?.risk_level)}</div>
          <div className="mt-2 text-sm text-theme-secondary">总分 {formatAnalysisScore(analysis?.total_score)}</div>
        </div>

        <div className={cn('flex flex-col gap-3', riskModules.length === 0 && 'justify-center')}>
          {riskModules.length === 0 ? (
            <EmptyPanel text="当前没有单独可拆的风险模块。" />
          ) : riskModules.map((module) => (
            <div key={module.code} className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-4">
              <div className="flex items-center justify-between gap-4">
                <div className="text-sm font-semibold text-theme-primary">{module.name}</div>
                <div className="text-lg font-bold text-theme-primary">{formatAnalysisScore(module.score)}</div>
              </div>
              {module.summary && (
                <div className="mt-2 text-sm leading-6 text-theme-secondary">{module.summary}</div>
              )}
            </div>
          ))}

          {(analysis?.warnings || []).length > 0 && (
            <div className="rounded-2xl border border-amber-500/20 bg-amber-500/10 p-4">
              <div className="mb-2 text-sm font-semibold text-amber-100">重点风险提示</div>
              <div className="space-y-2">
                {analysis?.warnings.map((warning, index) => (
                  <div key={`${warning}-${index}`} className="flex gap-3 text-sm text-theme-secondary">
                    <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-amber-200" />
                    <span>{warning}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </section>
  )
}

function StructureComparisonCard({
  sectorSnapshot,
  themeSnapshot,
  exposureEvents,
  positiveModules,
}: {
  sectorSnapshot?: { primary_sector_name?: string; breakdown?: Array<{ sector_name: string; weight_percent: string }> }
  themeSnapshot?: { primary_theme_name?: string; breakdown?: Array<{ theme_name: string; weight_percent: string }> }
  exposureEvents: FundAnalysisEventImpact[]
  positiveModules: FundAnalysisModuleScore[]
}) {
  const topSectors = (sectorSnapshot?.breakdown || []).slice(0, 3)
  const topThemes = (themeSnapshot?.breakdown || []).slice(0, 3)

  return (
    <section className="glass flex h-full flex-col rounded-3xl p-6">
      <div className="mb-4 flex items-center gap-3">
        <div className="rounded-2xl bg-emerald-500/15 p-3">
          <Layers3 className="h-5 w-5 text-emerald-200" />
        </div>
        <div>
          <div className="text-sm font-semibold text-theme-primary">结构变化对比</div>
          <div className="text-xs text-theme-muted">把当前主行业 / 主主题和暴露变化事件放到一起看。</div>
        </div>
      </div>

      <div className="flex flex-1 flex-col space-y-4">
        <div className="grid gap-3 sm:grid-cols-2">
          <MetaRow label="当前主行业" value={sectorSnapshot?.primary_sector_name || '--'} />
          <MetaRow label="当前主主题" value={themeSnapshot?.primary_theme_name || '--'} />
        </div>

        <div className="grid items-stretch gap-4 lg:grid-cols-2">
          <div className="flex h-full min-h-[10rem] flex-col rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-4">
            <div className="mb-2 text-sm font-semibold text-theme-primary">行业结构 Top3</div>
            <div className="flex flex-1 flex-col justify-center space-y-2">
              {topSectors.length === 0 ? <EmptyInline text="暂无行业结构数据" /> : topSectors.map((item) => (
                <div key={item.sector_name} className="flex items-center justify-between gap-3 text-sm">
                  <span className="text-theme-secondary">{item.sector_name}</span>
                  <span className="font-medium text-theme-primary">{item.weight_percent}</span>
                </div>
              ))}
            </div>
          </div>

          <div className="flex h-full min-h-[10rem] flex-col rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-4">
            <div className="mb-2 text-sm font-semibold text-theme-primary">主题结构 Top3</div>
            <div className="flex flex-1 flex-col justify-center space-y-2">
              {topThemes.length === 0 ? <EmptyInline text="暂无主题结构数据" /> : topThemes.map((item) => (
                <div key={item.theme_name} className="flex items-center justify-between gap-3 text-sm">
                  <span className="text-theme-secondary">{item.theme_name}</span>
                  <span className="font-medium text-theme-primary">{item.weight_percent}</span>
                </div>
              ))}
            </div>
          </div>
        </div>

        <div className="rounded-2xl border border-emerald-500/20 bg-emerald-500/10 p-4">
          <div className="mb-2 text-sm font-semibold text-emerald-100">结构变化观察</div>
          <div className="space-y-2">
            {exposureEvents.length > 0 ? exposureEvents.map((event, index) => (
              <div key={`${event.code}-${index}`} className="text-sm leading-6 text-theme-secondary">
                <span className="font-medium text-theme-primary">{event.title}：</span>
                {event.summary}
              </div>
            )) : (
              <EmptyInline text="当前还没有显著的主线结构变化事件。" />
            )}
          </div>
        </div>

        <div className="rounded-2xl border border-cyan-500/20 bg-cyan-500/10 p-4">
          <div className="mb-2 text-sm font-semibold text-cyan-100">当前较强支撑模块</div>
          <div className="space-y-2">
            {positiveModules.length === 0 ? <EmptyInline text="当前没有明显高分支撑模块。" /> : positiveModules.slice(0, 3).map((module) => (
              <div key={module.code} className="text-sm leading-6 text-theme-secondary">
                <span className="font-medium text-theme-primary">{module.name}（{formatAnalysisScore(module.score)}）：</span>
                {module.summary || '当前暂无附加说明'}
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  )
}

function EmptyPanel({ text }: { text: string }) {
  return (
    <div className="flex min-h-[6rem] w-full items-center justify-center rounded-2xl border border-dashed border-[var(--card-border)] px-4 py-5 text-center text-sm text-theme-muted">
      {text}
    </div>
  )
}

function EmptyInline({ text }: { text: string }) {
  return <div className="text-sm text-theme-muted">{text}</div>
}
