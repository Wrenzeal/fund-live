'use client'

import { useEffect, useId, useState, type ReactNode } from 'react'
import Link from 'next/link'
import { AlertTriangle, ArrowLeft, BarChart3, CalendarClock, FileText, Gauge, Layers3, PieChart, ShieldCheck, ShieldAlert, Sparkles, Target, TimerReset, Zap } from 'lucide-react'
import { AnimatedScoreGauge } from '@/components/animated-score-gauge'
import { AnalysisReveal as RevealSection, AnalysisSectionHeading as SectionHeading, useLazyReveal } from '@/components/analysis-layout'
import { EstimateCard } from '@/components/estimate-card'
import { FundSectorCard } from '@/components/fund-sector-card'
import { HoldingsTable } from '@/components/holdings-table'
import { TargetETFHoldingsCard } from '@/components/target-etf-holdings-card'
import { AnalysisEventTraceMeta } from '@/components/analysis-event-trace-meta'
import { useFundAnalysis, useFundDashboard, useFundHoldings, type Fund, type FundAnalysis, type FundAnalysisEventImpact, type FundAnalysisModuleScore, type FundClassificationOverride, type FundEstimate, type FundSectorSnapshot, type FundThemeSnapshot } from '@/hooks/use-fund-data'
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
    classificationOverride,
    displayDate,
    isHistorical,
    officialClose,
    isLoading,
    isValidating,
    mutate: refreshDashboard,
  } = useFundDashboard(fundId)
  const {
    analysis,
    isLoading: isAnalysisLoading,
    mutate: refreshAnalysis,
  } = useFundAnalysis(fundId)

  const {
    holdings: resolvedHoldings,
    displayItems,
    displayLevel,
    lookthroughAvailable,
  } = useFundHoldings(fundId)

  const lastUpdated = estimate?.calculated_at ? new Date(estimate.calculated_at) : null
  const timelineEvents = (analysis?.event_impacts || []).slice().sort((left, right) => eventTimelineRank(left) - eventTimelineRank(right))
  const realtimeRadarEvents = buildRealtimeRadarEvents(analysis?.event_impacts || [])
  const quarterlyEvents = (analysis?.event_impacts || []).filter((item) => item.horizon === 'quarterly')
  const exposureEvents = (analysis?.event_impacts || []).filter((item) => item.target_scope === 'exposure')
  const riskModules = (analysis?.module_scores || []).filter((item) => isRiskModule(item.code))
  const positiveModules = (analysis?.module_scores || []).filter((item) => !isRiskModule(item.code))
  const recommendationItems = buildRecommendationItems(analysis)

  return (
    <main className="min-h-[100dvh]">
      <div className="container mx-auto max-w-7xl px-4 py-6 md:py-8">
        <header className="mb-6 overflow-hidden rounded-[2rem] border border-[var(--card-border)] bg-[var(--card-bg)]/45 p-5 shadow-[0_22px_60px_rgba(0,0,0,0.10)] md:p-6">
          <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
            <div className="min-w-0">
              <Link
                href={fundId ? `/?fund=${fundId}` : '/'}
                className="mb-4 inline-flex items-center gap-2 text-sm text-theme-secondary transition-colors hover:text-theme-primary"
              >
                <ArrowLeft className="h-4 w-4" />
                返回基金详情
              </Link>
              <div className="text-2xl font-black tracking-tight text-theme-primary md:text-3xl">量化看板</div>
              <div className="mt-2 max-w-3xl text-sm leading-6 text-theme-secondary">
                {fund?.name ? `${fund.name}（${fund.id}）` : `基金 ${fundId}`} · 先看结论和证据，再展开事件、风险与底层数据。
              </div>
            </div>

            <div className="grid grid-cols-2 gap-2 text-xs text-theme-secondary sm:grid-cols-4 lg:min-w-[30rem]">
              <QuickStat label="口径" value={analysis?.analysis_basis || '--'} />
              <QuickStat label="风险" value={riskLevelLabel(analysis?.risk_level)} />
              <QuickStat label="覆盖" value={confidenceLevelLabel(analysis?.confidence)} />
              <QuickStat label="披露期" value={analysis?.latest_holding_period || '--'} />
            </div>
          </div>
        </header>

        <div className="space-y-5 md:space-y-6">
          <RevealSection>
            <AnalysisHeroVisual
              analysis={analysis}
              isLoading={isAnalysisLoading}
              sectorName={sectorSnapshot?.primary_sector_name}
              themeName={themeSnapshot?.primary_theme_name}
            />
          </RevealSection>

          <RevealSection delay={80}>
            <InsightStrip
              analysis={analysis}
              sectorName={sectorSnapshot?.primary_sector_name}
              themeName={themeSnapshot?.primary_theme_name}
            />
          </RevealSection>

          <RevealSection delay={120}>
            <div className="grid items-stretch gap-5 xl:grid-cols-[minmax(0,0.95fr)_minmax(0,1.05fr)]">
              <RecommendationMixPanel items={recommendationItems} />
              <ModuleRadarPanel modules={analysis?.module_scores || []} />
            </div>
          </RevealSection>

          <RevealSection delay={120}>
            <EvidenceFocusGrid analysis={analysis} />
          </RevealSection>

          <RevealSection delay={120}>
            <div className="space-y-5">
              <RealtimeEventRadar events={realtimeRadarEvents} />
              <EventSignalBoard events={timelineEvents} />
              <RiskBreakdownCard analysis={analysis} riskModules={riskModules} />
            </div>
          </RevealSection>

          <RevealSection delay={120}>
            <div className="grid items-stretch gap-5 xl:grid-cols-[minmax(0,1fr)_minmax(20rem,0.92fr)]">
              <StructureComparisonCard
                sectorSnapshot={sectorSnapshot}
                themeSnapshot={themeSnapshot}
                exposureEvents={exposureEvents}
                positiveModules={positiveModules}
              />
              <QuarterlyDiffCard events={quarterlyEvents} />
            </div>
          </RevealSection>

          <RevealSection delay={120}>
            <DataContextPanel
              fund={fund}
              estimate={estimate}
              isLoading={isLoading}
              isValidating={isValidating}
              lastUpdated={lastUpdated}
              sectorSnapshot={sectorSnapshot}
              themeSnapshot={themeSnapshot}
              classificationOverride={classificationOverride}
              onClassificationOverrideUpdated={() => {
                void refreshDashboard()
                void refreshAnalysis()
              }}
              displayLevel={displayLevel}
              lookthroughAvailable={lookthroughAvailable}
              displayDate={displayDate}
              isHistorical={isHistorical}
              officialCloseStatus={officialClose?.display_status}
            />
          </RevealSection>

          <RevealSection delay={120}>
            <HoldingsTable
              estimate={estimate}
              displayLevel={displayLevel}
              items={displayItems}
              lookthroughAvailable={lookthroughAvailable}
            />
          </RevealSection>

          {displayLevel === 'target_layer' && resolvedHoldings.length > 0 && (
            <RevealSection delay={120}>
              <TargetETFHoldingsCard
                targetName={displayItems[0]?.name}
                holdings={resolvedHoldings}
              />
            </RevealSection>
          )}

          <RevealSection delay={120}>
            <MethodCompactCard analysis={analysis} />
          </RevealSection>
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
              看板优先展示结论、主证据、风险限制和事件脉络，详细评分放在下方展开查看。
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

function QuickStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 px-3 py-2.5">
      <div className="text-[10px] tracking-[0.18em] text-theme-muted">{label}</div>
      <div className="mt-1 truncate text-sm font-semibold text-theme-primary">{value}</div>
    </div>
  )
}

function InsightStrip({
  analysis,
  sectorName,
  themeName,
}: {
  analysis?: FundAnalysis
  sectorName?: string
  themeName?: string
}) {
  const dominant = dominantRecommendationItem(buildRecommendationItems(analysis))
  const facts = [
    {
      icon: <BarChart3 className="h-4 w-4 text-cyan-200" />,
      label: '当前方向',
      value: analysis ? dominant.label : '--',
      detail: analysis ? `${formatAnalysisPercent(dominant.value)} · 阈值规则口径` : '等待量化结果',
    },
    {
      icon: <ShieldCheck className="h-4 w-4 text-emerald-200" />,
      label: '可信覆盖',
      value: confidenceLevelLabel(analysis?.confidence),
      detail: analysis?.latest_holding_period ? `持仓披露：${analysis.latest_holding_period}` : '持仓披露待补齐',
    },
    {
      icon: <Layers3 className="h-4 w-4 text-fuchsia-200" />,
      label: '主线结构',
      value: themeName || sectorName || '--',
      detail: themeName && sectorName ? `${sectorName} / ${themeName}` : '行业与主题快照合并观察',
    },
    {
      icon: <FileText className="h-4 w-4 text-amber-200" />,
      label: '证据数量',
      value: `${(analysis?.primary_evidence || []).length} / ${(analysis?.counter_evidence || []).length}`,
      detail: '主证据 / 反方限制，避免重复堆叠规则文本',
    },
  ]

  return (
    <section className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
      {facts.map((item) => (
        <div key={item.label} className="rounded-3xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-4 transition-all duration-300 hover:-translate-y-0.5 hover:border-cyan-400/25">
          <div className="mb-3 flex items-center gap-2 text-xs font-semibold text-theme-muted">
            {item.icon}
            {item.label}
          </div>
          <div className="truncate text-base font-bold text-theme-primary">{item.value}</div>
          <div className="mt-2 line-clamp-2 text-xs leading-5 text-theme-muted">{item.detail}</div>
        </div>
      ))}
    </section>
  )
}

function DataContextPanel({
  fund,
  estimate,
  isLoading,
  isValidating,
  lastUpdated,
  sectorSnapshot,
  themeSnapshot,
  classificationOverride,
  onClassificationOverrideUpdated,
  displayLevel,
  lookthroughAvailable,
  displayDate,
  isHistorical,
  officialCloseStatus,
}: {
  fund?: Fund
  estimate?: FundEstimate
  isLoading: boolean
  isValidating: boolean
  lastUpdated: Date | null
  sectorSnapshot?: FundSectorSnapshot
  themeSnapshot?: FundThemeSnapshot
  classificationOverride?: FundClassificationOverride
  onClassificationOverrideUpdated?: () => void
  displayLevel: 'stock_layer' | 'target_layer'
  lookthroughAvailable: boolean
  displayDate: string
  isHistorical: boolean
  officialCloseStatus?: 'hidden' | 'pending' | 'ready'
}) {
  return (
    <section className="space-y-5">
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
        classificationOverride={classificationOverride}
        onClassificationOverrideUpdated={onClassificationOverrideUpdated}
      />

      <section className="glass flex h-full flex-col rounded-3xl p-5">
          <SectionHeading
            icon={<Gauge className="h-4 w-4 text-cyan-200" />}
            title="数据口径"
            description="把展示层级、分时与官方净值状态收在一个位置。"
          />
          <div className="mt-4 flex flex-1 flex-col justify-center space-y-2 text-sm text-theme-secondary">
            <MetaRow label="展示层级" value={displayLevel === 'target_layer' ? '下一层追踪目标' : '股票持仓层'} />
            <MetaRow label="可穿透估值" value={lookthroughAvailable ? '是' : '否'} />
            <MetaRow label="分时日期" value={displayDate || '--'} />
            <MetaRow label="历史数据" value={isHistorical ? '是' : '否'} />
            <MetaRow label="官方收盘" value={officialCloseStatus === 'ready' ? '已就绪' : officialCloseStatus === 'pending' ? '待同步' : '隐藏'} />
          </div>
      </section>
    </section>
  )
}

function MethodCompactCard({ analysis }: { analysis?: FundAnalysis }) {
  const deductions = analysis?.confidence_deductions || []
  const limitations = analysis?.ai_explanation?.limitations || []
  const mergedLimits = [...deductions, ...limitations].slice(0, 4)

  return (
    <section className="glass rounded-3xl p-5 md:p-6">
      <SectionHeading
        icon={<FileText className="h-4 w-4 text-amber-200" />}
        title="方法与限制"
        description="保留必要边界说明，避免和上方证据、事件、风险模块重复。"
      />
      <div className="mt-4 grid gap-4 lg:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)]">
        <div className="rounded-2xl border border-cyan-500/20 bg-cyan-500/10 p-4 text-sm leading-6 text-theme-secondary">
          量化看板会统一估值、持仓、分类与事件快照，再输出观察分布、模块分与证据链；解释说明不会改写评分或风险等级。
        </div>
        <div className="rounded-2xl border border-amber-500/20 bg-amber-500/10 p-4">
          {mergedLimits.length === 0 ? (
            <EmptyInline text="当前没有额外可信度扣分或限制说明。" />
          ) : (
            <div className="space-y-2">
              {mergedLimits.map((item, index) => (
                <div key={`${item}-${index}`} className="flex gap-3 text-sm leading-6 text-theme-secondary">
                  <AlertTriangle className="mt-1 h-3.5 w-3.5 shrink-0 text-amber-200" />
                  <span>{item}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </section>
  )
}

function RecommendationMixPanel({ items }: { items: RecommendationItem[] }) {
  const [panelRef, hasEnteredView] = useLazyReveal<HTMLElement>()
  const total = items.reduce((sum, item) => sum + Math.max(item.value, 0), 0)

  return (
    <section ref={panelRef} className="glass relative flex h-full min-h-[22rem] flex-col overflow-hidden rounded-3xl p-5 md:p-6">
      <div className="pointer-events-none absolute -left-20 top-1/2 h-56 w-56 -translate-y-1/2 rounded-full bg-fuchsia-500/10 blur-3xl" />
      <div className="pointer-events-none absolute -right-16 top-8 h-48 w-48 rounded-full bg-cyan-500/10 blur-3xl" />

      <div className="relative mb-5 flex items-center justify-between gap-4">
        <SectionHeading
          icon={<PieChart className="h-4 w-4 text-cyan-200" />}
          title="建议分布"
          description="比例只表示结构偏向，不是交易指令。"
        />
        <span className="hidden rounded-full border border-cyan-500/20 bg-cyan-500/10 px-3 py-1 text-xs text-cyan-100 sm:inline-flex">
          滚动绘制
        </span>
      </div>

      {total <= 0 ? (
        <div className="relative flex flex-1 items-center">
          <EmptyPanel text="当前没有可绘制的建议分布数据。" />
        </div>
      ) : (
        <div className="relative grid flex-1 items-center gap-5 lg:grid-cols-[16rem_minmax(0,1fr)]">
          <div className="flex min-h-[16rem] items-center justify-center">
            {hasEnteredView ? (
              <AnimatedRecommendationDonut items={items} />
            ) : (
              <div className="relative flex h-56 w-56 items-center justify-center rounded-full border border-dashed border-[var(--card-border)] bg-[var(--card-bg)]/30 text-center">
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
    <div className={cn('relative h-64 w-64 transition-all duration-700 md:h-72 md:w-72', draw ? 'scale-100 opacity-100' : 'scale-95 opacity-0')}>
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
    <section className="glass flex h-full min-h-[22rem] flex-col justify-center rounded-3xl p-5 md:p-6">
      <SectionHeading
        icon={<Zap className="h-4 w-4 text-fuchsia-200" />}
        title="六维模块"
        description="用雷达图压缩展示模块强弱，避免重复展开每条规则。"
      />
      {normalized.length === 0 ? (
        <EmptyPanel text="当前没有可展示的模块分数。" />
      ) : (
      <div className="mt-5 grid items-center gap-5 md:grid-cols-[13rem_minmax(0,1fr)]">
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
    <section className="grid items-stretch gap-5 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_20rem]">
      <EvidenceColumn title="主证据" icon={<Target className="h-4 w-4 text-cyan-200" />} items={primary} />
      <EvidenceColumn title="反方证据 / 限制" icon={<ShieldAlert className="h-4 w-4 text-amber-200" />} items={counter} amber />
      <div className="glass flex h-full min-h-[18rem] flex-col rounded-3xl p-5 md:p-6">
        <SectionHeading
          icon={<Sparkles className="h-4 w-4 text-fuchsia-200" />}
          title="解释说明"
          description="补充结论归因，不改评分。"
        />
        {ai ? (
          <div className="mt-4 flex flex-1 flex-col justify-center space-y-3">
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
            <EmptyPanel text="当前没有补充解释；评分与证据仍可查看。" />
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
    <div className={cn('glass flex h-full min-h-[18rem] flex-col rounded-3xl p-5 md:p-6', amber ? 'bg-amber-500/5' : 'bg-cyan-500/5')}>
      <SectionHeading icon={icon} title={title} />
      <div className={cn('mt-4 flex flex-1 flex-col gap-3', items.length === 0 && 'justify-center')}>
        {items.length === 0 ? (
          <EmptyPanel text={`${title} 暂无结构化数据。`} />
        ) : items.slice(0, 3).map((item, index) => (
          <div key={`${item.code}-${index}`} className={cn('min-h-[7.5rem] rounded-2xl border p-4 transition-transform duration-300 hover:-translate-y-0.5', amber ? 'border-amber-500/20 bg-amber-500/10' : 'border-cyan-500/20 bg-cyan-500/10')}>
            <div className="text-sm font-semibold text-theme-primary">{item.title}</div>
            <div className="mt-2 line-clamp-3 text-xs leading-5 text-theme-secondary">{item.summary}</div>
            <AnalysisEventTraceMeta trace={item} dense />
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

function buildRealtimeRadarEvents(events: FundAnalysisEventImpact[]) {
  return events
    .filter((event) => event.horizon === 'current' || event.horizon === 'intraday')
    .filter((event) => event.target_scope !== 'disclosure' && event.target_scope !== 'methodology')
    .sort((left, right) => realtimeEventPriority(right) - realtimeEventPriority(left))
    .slice(0, 5)
}

function realtimeEventPriority(event: FundAnalysisEventImpact) {
  let score = 0
  if (event.code.startsWith('realtime_')) score += 8
  if (event.target_scope === 'macro') score += 5
  if (event.target_scope === 'holding') score += 4
  if (event.target_scope === 'exposure') score += 3
  if (event.strength === 'high') score += 3
  if (event.strength === 'medium') score += 1
  if (event.weight_hint) score += Math.min(parseAnalysisNumber(event.weight_hint) / 10, 4)
  return score
}

function eventScopeLabel(scope?: FundAnalysisEventImpact['target_scope']) {
  switch (scope) {
    case 'macro':
      return '实时宏观'
    case 'holding':
      return '持仓事件'
    case 'exposure':
      return '主线暴露'
    case 'fund':
      return '基金公告'
    case 'index':
      return '指数事件'
    case 'disclosure':
      return '披露口径'
    case 'methodology':
      return '方法限制'
    default:
      return '事件'
  }
}

function RealtimeEventRadar({ events }: { events: FundAnalysisEventImpact[] }) {
  const lead = events[0]

  return (
    <section className="glass overflow-hidden rounded-3xl p-5 md:p-6">
      <div className="grid gap-5 lg:grid-cols-[minmax(0,0.86fr)_minmax(0,1.14fr)]">
        <div className="relative min-h-[15rem] rounded-[2rem] border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-5">
          <div className="pointer-events-none absolute -right-14 -top-14 h-40 w-40 rounded-full bg-cyan-500/15 blur-3xl" />
          <SectionHeading
            icon={<Zap className="h-4 w-4 text-cyan-200" />}
            title="实时事件雷达"
            description="只展示已映射到持仓、主线或宏观暴露的当前事件。"
          />
          {lead ? (
            <div className="relative mt-5">
              <div className="mb-3 flex flex-wrap gap-2">
                <span className={cn('rounded-full border px-3 py-1 text-xs', eventImpactTone(lead.impact))}>
                  {eventImpactLabel(lead.impact)}
                </span>
                <span className="rounded-full border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-1 text-xs text-theme-secondary">
                  {eventScopeLabel(lead.target_scope)}
                </span>
                {lead.weight_hint && (
                  <span className="rounded-full border border-cyan-500/20 bg-cyan-500/10 px-3 py-1 text-xs text-cyan-100">
                    暴露 {lead.weight_hint}%
                  </span>
                )}
              </div>
              <div className="text-xl font-black leading-tight text-theme-primary md:text-2xl">{lead.title}</div>
              <div className="mt-3 text-sm leading-7 text-theme-secondary">{lead.summary}</div>
              <AnalysisEventTraceMeta trace={lead} />
            </div>
          ) : (
            <div className="mt-5">
              <EmptyPanel text="当前没有可映射的实时事件。若热点未落到基金持仓或主线暴露，不会强行写入结论。" />
            </div>
          )}
        </div>

        <div className={cn('grid gap-3 sm:grid-cols-2', events.length === 0 && 'block')}>
          {events.length === 0 ? (
            <EmptyPanel text="暂无事件雷达卡片。" />
          ) : events.map((event) => (
            <div key={event.code} className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-4 transition-transform duration-300 hover:-translate-y-0.5">
              <div className="mb-3 flex flex-wrap items-center gap-2 text-[11px]">
                <span className="rounded-full border border-[var(--input-border)] bg-[var(--input-bg)] px-2.5 py-1 text-theme-secondary">
                  {eventScopeLabel(event.target_scope)}
                </span>
                <span className={cn('rounded-full border px-2.5 py-1', eventImpactTone(event.impact))}>
                  {eventStrengthLabel(event.strength)}
                </span>
              </div>
              <div className="line-clamp-2 text-sm font-semibold text-theme-primary">{event.title}</div>
              <div className="mt-2 line-clamp-3 text-xs leading-5 text-theme-secondary">{event.summary}</div>
              <AnalysisEventTraceMeta trace={event} dense />
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}

function EventSignalBoard({ events }: { events: FundAnalysisEventImpact[] }) {
  const groups = [
    {
      key: 'macro',
      title: '实时宏观',
      description: '只接收能映射到基金暴露的宏观/地缘事件。',
      events: events.filter((event) => event.target_scope === 'macro'),
    },
    {
      key: 'holding',
      title: '持仓事件',
      description: '重仓股公告、盘中驱动和个股层事件。',
      events: events.filter((event) => event.target_scope === 'holding'),
    },
    {
      key: 'exposure',
      title: '主线暴露',
      description: '行业、主题、季度结构变化和集中度线索。',
      events: events.filter((event) => event.target_scope === 'exposure' || event.target_scope === 'index'),
    },
    {
      key: 'basis',
      title: '口径与限制',
      description: '披露新鲜度、方法口径和其他辅助信号。',
      events: events.filter((event) => !['macro', 'holding', 'exposure', 'index'].includes(event.target_scope || '')),
    },
  ].filter((group) => group.events.length > 0)

  return (
    <section className="glass flex h-full min-h-[22rem] flex-col rounded-3xl p-5 md:p-6">
      <SectionHeading
        icon={<CalendarClock className="h-4 w-4 text-cyan-200" />}
        title="事件信号链"
        description="按实时宏观、持仓、主线和口径限制分组，避免热点与规则说明混在一起。"
      />

      <div className={cn('mt-5 flex-1', groups.length === 0 && 'flex items-center')}>
        {groups.length === 0 ? (
          <EmptyPanel text="当前还没有可展开的事件信号。" />
        ) : (
          <div className="grid gap-4 xl:grid-cols-2">
            {groups.map((group) => (
              <EventGroupCard key={group.key} title={group.title} description={group.description} events={group.events} />
            ))}
          </div>
        )}
      </div>
    </section>
  )
}

function EventGroupCard({
  title,
  description,
  events,
}: {
  title: string
  description: string
  events: FundAnalysisEventImpact[]
}) {
  return (
    <div className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/30 p-4">
      <div className="mb-3 flex items-start justify-between gap-3">
        <div>
          <div className="text-sm font-semibold text-theme-primary">{title}</div>
          <div className="mt-1 text-xs leading-5 text-theme-muted">{description}</div>
        </div>
        <span className="rounded-full border border-[var(--input-border)] bg-[var(--input-bg)] px-2.5 py-1 text-[11px] text-theme-secondary">
          {events.length} 条
        </span>
      </div>

      <div className="space-y-3">
        {events.slice(0, 4).map((event, index) => (
          <div key={`${event.code}-${index}`} className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-4 transition-transform duration-300 hover:-translate-y-0.5">
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
            <AnalysisEventTraceMeta trace={event} dense />
            {event.related_symbols && event.related_symbols.length > 0 && (
              <div className="mt-2 text-xs text-theme-muted">
                相关标的：{event.related_symbols.join(' / ')}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}

function QuarterlyDiffCard({ events }: { events: FundAnalysisEventImpact[] }) {
  return (
    <section className="glass flex h-full min-h-[20rem] flex-col rounded-3xl p-5 md:p-6">
      <SectionHeading
        icon={<TimerReset className="h-4 w-4 text-fuchsia-200" />}
        title="季报变化"
        description="只保留季度结构变化，避免与当前事件链重复。"
      />

      <div className={cn('mt-4 flex flex-1 flex-col gap-3', events.length === 0 && 'justify-center')}>
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
    <section className="glass flex h-full flex-col rounded-3xl p-5 md:p-6">
      <SectionHeading
        icon={<ShieldAlert className="h-4 w-4 text-amber-200" />}
        title="风险拆解"
        description="合并风险模块与 warning，减少重复风险文案。"
      />

      <div className="mt-4 space-y-4">
        <div className="rounded-2xl border border-amber-500/25 bg-amber-500/10 p-5">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <div className="text-[11px] font-semibold tracking-[0.22em] text-amber-100/80">CURRENT RISK</div>
              <div className="mt-2 text-3xl font-black text-theme-primary">{riskLevelLabel(analysis?.risk_level)}</div>
            </div>
            <div className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 px-4 py-3 text-sm text-theme-secondary">
              总分 <span className="ml-2 text-lg font-bold text-theme-primary">{formatAnalysisScore(analysis?.total_score)}</span>
            </div>
          </div>
        </div>

        <div className={cn('grid gap-3 md:grid-cols-2', riskModules.length === 0 && 'block')}>
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

        </div>

        {(analysis?.warnings || []).length > 0 && (
          <div className="rounded-2xl border border-amber-500/25 bg-amber-500/10 p-4">
            <div className="mb-3 flex items-center gap-2 text-sm font-semibold text-theme-primary">
              <AlertTriangle className="h-4 w-4 text-amber-200" />
              重点风险提示
            </div>
            <div className="space-y-2">
              {analysis?.warnings.map((warning, index) => (
                <div key={`${warning}-${index}`} className="flex gap-3 rounded-xl border border-[var(--card-border)] bg-[var(--card-bg)]/30 px-3 py-2 text-sm leading-6 text-theme-secondary">
                  <span className="mt-2 h-1.5 w-1.5 shrink-0 rounded-full bg-amber-300" />
                  <span>{warning}</span>
                </div>
              ))}
            </div>
          </div>
        )}
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
    <section className="glass flex h-full flex-col rounded-3xl p-5 md:p-6">
      <SectionHeading
        icon={<Layers3 className="h-4 w-4 text-emerald-200" />}
        title="结构与暴露"
        description="聚合主行业、主题和暴露变化，不再重复完整分类卡。"
      />

      <div className="mt-4 flex flex-1 flex-col space-y-4">
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
