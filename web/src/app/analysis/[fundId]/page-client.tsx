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
import { FundAnalysisDecisionList } from '@/components/fund-analysis-decision-points'
import { useFundAnalysis, useFundDashboard, useFundHoldings, type Fund, type FundAnalysis, type FundAnalysisEventImpact, type FundAnalysisModuleScore, type FundClassificationOverride, type FundEstimate, type FundSectorSnapshot, type FundThemeSnapshot } from '@/hooks/use-fund-data'
import { cn } from '@/lib/utils'
import { buildFundAnalysisDecision, type FundAnalysisDecisionView } from '@/lib/fund-analysis-decision'
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
    fund: dashboardFund,
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
    fund: analysisFund,
    analysis,
    isLoading: isAnalysisLoading,
    isValidating: isAnalysisValidating,
    mutate: refreshAnalysis,
  } = useFundAnalysis(fundId)

  const {
    fund: holdingsFund,
    holdings: resolvedHoldings,
    displayItems,
    displayLevel,
    lookthroughAvailable,
    isLoading: isHoldingsLoading,
  } = useFundHoldings(fundId)

  const resolvedFund = dashboardFund || analysisFund || holdingsFund
  const lastUpdated = estimate?.calculated_at ? new Date(estimate.calculated_at) : null
  const isAnalysisPending = (isAnalysisLoading || isAnalysisValidating) && !analysis
  const isHoldingsPending = isHoldingsLoading && displayItems.length === 0
  const holdingsReady = displayItems.length > 0
  const timelineEvents = (analysis?.event_impacts || []).slice().sort((left, right) => eventTimelineRank(left) - eventTimelineRank(right))
  const realtimeRadarEvents = buildRealtimeRadarEvents(analysis?.event_impacts || [])
  const quarterlyEvents = (analysis?.event_impacts || []).filter((item) => item.horizon === 'quarterly')
  const exposureEvents = (analysis?.event_impacts || []).filter((item) => item.target_scope === 'exposure')
  const riskModules = (analysis?.module_scores || []).filter((item) => isRiskModule(item.code))
  const positiveModules = (analysis?.module_scores || []).filter((item) => !isRiskModule(item.code))
  const recommendationItems = buildRecommendationItems(analysis)
  const decision = buildFundAnalysisDecision(analysis)

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
                {resolvedFund?.name ? `${resolvedFund.name}（${resolvedFund.id || fundId}）` : isLoading ? '基金信息加载中' : `基金 ${fundId}`} · 先看结论和证据，再展开事件、风险与底层数据。
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
              decision={decision}
              isLoading={isAnalysisLoading}
              sectorName={sectorSnapshot?.primary_sector_name}
              themeName={themeSnapshot?.primary_theme_name}
            />
          </RevealSection>

          <RevealSection delay={80}>
            <InsightStrip
              analysis={analysis}
              decision={decision}
              sectorName={sectorSnapshot?.primary_sector_name}
              themeName={themeSnapshot?.primary_theme_name}
            />
          </RevealSection>

          <RevealSection delay={120}>
            <div className="grid items-stretch gap-5 xl:grid-cols-[minmax(0,0.95fr)_minmax(0,1.05fr)]">
              <RecommendationMixPanel items={recommendationItems} isLoading={isAnalysisPending} />
              <ModuleRadarPanel modules={analysis?.module_scores || []} isLoading={isAnalysisPending} />
            </div>
          </RevealSection>

          <RevealSection delay={120}>
            <EvidenceFocusGrid analysis={analysis} decision={decision} isLoading={isAnalysisPending} />
          </RevealSection>

          <RevealSection delay={120}>
            <div className="space-y-5">
              <RealtimeEventRadar events={realtimeRadarEvents} isLoading={isAnalysisPending} />
              <EventSignalBoard events={timelineEvents} isLoading={isAnalysisPending} />
              <RiskBreakdownCard analysis={analysis} decision={decision} riskModules={riskModules} isLoading={isAnalysisPending} />
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
              fund={resolvedFund}
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
              holdingsReady={holdingsReady}
              isHoldingsLoading={isHoldingsLoading}
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
              isLoading={isHoldingsPending}
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
            <MethodCompactCard analysis={analysis} decision={decision} />
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
  decision,
  isLoading,
  sectorName,
  themeName,
}: {
  analysis?: FundAnalysis
  decision?: FundAnalysisDecisionView | null
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

  const view = decision || buildFundAnalysisDecision(analysis)
  const recommendationItems = buildRecommendationItems(analysis)
  const dominant = dominantRecommendationItem(recommendationItems)

  return (
    <section className="glass overflow-hidden rounded-3xl p-5 sm:p-6">
      <div className="relative h-full">
        <div className="pointer-events-none absolute -right-12 -top-16 h-48 w-48 rounded-full bg-cyan-500/16 blur-3xl" />
        <div className="pointer-events-none absolute -bottom-16 left-16 h-40 w-40 rounded-full bg-sky-500/12 blur-3xl" />

        <div className="relative grid items-stretch gap-5 xl:grid-cols-[17rem_minmax(0,1fr)_22rem]">
          <div className="flex min-h-[17rem] items-center justify-center">
            <AnimatedScoreGauge value={analysis.total_score} label="TOTAL SCORE" variant="hero" />
          </div>

          <div className="flex min-h-[17rem] flex-col justify-center">
            <div className="mb-3 flex flex-wrap gap-2">
              <span className={cn('rounded-full border px-3 py-1 text-xs', dominant.tone)}>{view?.result.label || dominant.label}</span>
              <span className="rounded-full border border-[var(--card-border)] bg-[var(--input-bg)]/60 px-3 py-1 text-xs text-theme-secondary">
                {view?.basisLabel || analysis.analysis_basis}
              </span>
              <span className="rounded-full border border-cyan-500/20 bg-cyan-500/10 px-3 py-1 text-xs text-cyan-100">
                {view?.confidenceLabel || confidenceLevelLabel(analysis.confidence)}
              </span>
            </div>
            <h1 className="text-2xl font-black leading-tight text-theme-primary md:text-3xl">
              {view?.result.summary || analysis.summary}
            </h1>
            <p className="mt-3 max-w-3xl text-sm leading-7 text-theme-secondary">
              当前看板先回答结果，再把原因拆成主证据、风险限制和可追溯来源。
            </p>

            <div className="mt-5">
              <FundAnalysisDecisionList
                points={view?.mainReasons || []}
                emptyText="主原因仍在生成，先按弱观察处理。"
                compact
              />
            </div>
          </div>

          <div className="flex min-h-[17rem] flex-col justify-center rounded-[2rem] border border-[var(--card-border)] bg-[var(--card-bg)]/40 p-5">
            <div className="mb-4 flex items-center gap-2 text-sm font-semibold text-theme-primary">
              <ShieldAlert className="h-4 w-4 text-amber-200" />
              风险与口径
            </div>
            <div className="space-y-3 text-sm">
              <MetaRow label="风险等级" value={view?.riskLabel || riskLevelLabel(analysis.risk_level)} />
              <MetaRow label="主行业" value={sectorName || '--'} />
              <MetaRow label="主主题" value={themeName || '--'} />
              <MetaRow label="披露期" value={view?.holdingPeriodLabel || analysis.latest_holding_period || '--'} />
            </div>
            <div className="mt-4">
              <FundAnalysisDecisionList
                points={(view?.riskReasons || []).slice(0, 2)}
                emptyText="当前没有明显反方证据。"
                compact
              />
            </div>
          </div>
        </div>
      </div>
    </section>
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
  decision,
  sectorName,
  themeName,
}: {
  analysis?: FundAnalysis
  decision?: FundAnalysisDecisionView | null
  sectorName?: string
  themeName?: string
}) {
  const dominant = dominantRecommendationItem(buildRecommendationItems(analysis))
  const view = decision || buildFundAnalysisDecision(analysis)
  const facts = [
    {
      icon: <BarChart3 className="h-4 w-4 text-cyan-200" />,
      label: '当前方向',
      value: view?.result.label || (analysis ? dominant.label : '--'),
      detail: view ? `${view.result.percentLabel}，规则阈值口径` : '等待量化结果',
    },
    {
      icon: <ShieldCheck className="h-4 w-4 text-emerald-200" />,
      label: '可信覆盖',
      value: view?.confidenceLabel || confidenceLevelLabel(analysis?.confidence),
      detail: view?.holdingPeriodLabel ? `持仓披露：${view.holdingPeriodLabel}` : '持仓披露待补齐',
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
      value: view?.evidenceCountLabel || `${(analysis?.primary_evidence || []).length} / ${(analysis?.counter_evidence || []).length}`,
      detail: view?.topSignal?.title || '主证据 / 反方限制，避免重复堆叠规则文本',
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
  holdingsReady,
  isHoldingsLoading,
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
  holdingsReady: boolean
  isHoldingsLoading: boolean
  displayDate: string
  isHistorical: boolean
  officialCloseStatus?: 'hidden' | 'pending' | 'ready'
}) {
  const displayLevelText = isHoldingsLoading && !holdingsReady
    ? '拉取中'
    : holdingsReady
      ? displayLevel === 'target_layer' ? '下一层追踪目标' : '股票持仓层'
      : '待确认'
  const lookthroughText = isHoldingsLoading && !holdingsReady
    ? '拉取中'
    : holdingsReady
      ? lookthroughAvailable ? '是' : '否'
      : '--'

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
            <MetaRow label="展示层级" value={displayLevelText} />
            <MetaRow label="可穿透估值" value={lookthroughText} />
            <MetaRow label="分时日期" value={displayDate || '--'} />
            <MetaRow label="历史数据" value={isHistorical ? '是' : '否'} />
            <MetaRow label="官方收盘" value={officialCloseStatus === 'ready' ? '已就绪' : officialCloseStatus === 'pending' ? '待同步' : '隐藏'} />
          </div>
      </section>
    </section>
  )
}

function MethodCompactCard({
  analysis,
  decision,
}: {
  analysis?: FundAnalysis
  decision?: FundAnalysisDecisionView | null
}) {
  const view = decision || buildFundAnalysisDecision(analysis)
  const notes = view?.methodNotes || []

  return (
    <section className="glass rounded-3xl p-5 md:p-6">
      <SectionHeading
        icon={<FileText className="h-4 w-4 text-amber-200" />}
        title="方法与限制"
        description="只保留口径边界，避免重复上方风险与事件。"
      />
      <div className="mt-4 grid gap-4 lg:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)]">
        <div className="rounded-2xl border border-cyan-500/20 bg-cyan-500/10 p-4 text-sm leading-6 text-theme-secondary">
          量化看板统一估值、持仓、分类与事件快照；解释层只增强可读性，不改评分或风险等级。
        </div>
        <div className="rounded-2xl border border-amber-500/20 bg-amber-500/10 p-4">
          {notes.length === 0 ? (
            <EmptyInline text="当前没有额外可信度扣分或限制说明。" />
          ) : (
            <div className="space-y-2">
              {notes.map((item, index) => (
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

function RecommendationMixPanel({ items, isLoading = false }: { items: RecommendationItem[]; isLoading?: boolean }) {
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
          <EmptyPanel
            code={isLoading ? 'ANALYSIS_PIPELINE_ACTIVE' : 'PENDING_QUANT_SIGNAL'}
            text={isLoading ? '正在生成建议分布' : '建议分布未返回'}
            scanning={isLoading}
          />
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

function ModuleRadarPanel({ modules, isLoading = false }: { modules: FundAnalysisModuleScore[]; isLoading?: boolean }) {
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
        <RadarBasePlaceholder
          code={isLoading ? 'MODULE_VECTOR_SYNC' : 'MODULE_VECTOR_EMPTY'}
          text={isLoading ? '正在计算六维模块' : '模块分数未返回'}
          scanning={isLoading}
        />
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

function EvidenceFocusGrid({
  analysis,
  decision,
  isLoading = false,
}: {
  analysis?: FundAnalysis
  decision?: FundAnalysisDecisionView | null
  isLoading?: boolean
}) {
  const view = decision || buildFundAnalysisDecision(analysis)
  const ai = analysis?.ai_explanation

  return (
    <section className="grid items-stretch gap-5 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_20rem]">
      <DecisionEvidenceColumn
        title="为什么是这个结果"
        icon={<Target className="h-4 w-4 text-cyan-200" />}
        points={view?.mainReasons || []}
        isLoading={isLoading}
      />
      <DecisionEvidenceColumn
        title="反方证据与限制"
        icon={<ShieldAlert className="h-4 w-4 text-amber-200" />}
        points={view?.riskReasons || []}
        isLoading={isLoading}
        amber
      />
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
            <EmptyPanel
              code={isLoading ? 'AI_EXPLANATION_PENDING' : 'AI_EXPLANATION_EMPTY'}
              text={isLoading ? '正在同步解释层' : '解释层未返回'}
              scanning={isLoading}
            />
          </div>
        )}
      </div>
    </section>
  )
}

function DecisionEvidenceColumn({
  title,
  icon,
  points,
  isLoading = false,
  amber = false,
}: {
  title: string
  icon: ReactNode
  points: FundAnalysisDecisionView['mainReasons']
  isLoading?: boolean
  amber?: boolean
}) {
  return (
    <div className={cn('glass flex h-full min-h-[18rem] flex-col rounded-3xl p-5 md:p-6', amber ? 'bg-amber-500/5' : 'bg-cyan-500/5')}>
      <SectionHeading icon={icon} title={title} />
      <div className={cn('mt-4 flex flex-1 flex-col gap-3', points.length === 0 && 'justify-center')}>
        {points.length === 0 && isLoading ? (
          <EvidenceSkeletonList amber={amber} />
        ) : points.length === 0 ? (
          <EmptyPanel code={amber ? 'COUNTER_EVIDENCE_EMPTY' : 'PRIMARY_EVIDENCE_EMPTY'} text={`${title}未返回`} />
        ) : (
          <FundAnalysisDecisionList
            points={points}
            emptyText={`${title}未返回`}
            compact
          />
        )}
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

function RealtimeEventRadar({ events, isLoading = false }: { events: FundAnalysisEventImpact[]; isLoading?: boolean }) {
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
              <EmptyPanel
                code={isLoading ? 'EVENT_RADAR_SYNC' : 'EVENT_RADAR_IDLE'}
                text={isLoading ? '正在映射实时事件' : '暂无映射事件'}
                scanning={isLoading}
              />
            </div>
          )}
        </div>

        <div className={cn('grid gap-3 sm:grid-cols-2', events.length === 0 && 'block')}>
          {events.length === 0 && isLoading ? (
            <EventSkeletonGrid />
          ) : events.length === 0 ? (
            <EmptyPanel code="EVENT_CARD_EMPTY" text="事件卡片未返回" />
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

function EventSignalBoard({ events, isLoading = false }: { events: FundAnalysisEventImpact[]; isLoading?: boolean }) {
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
          <EmptyPanel
            code={isLoading ? 'SIGNAL_CHAIN_SYNC' : 'SIGNAL_CHAIN_IDLE'}
            text={isLoading ? '正在生成事件信号链' : '暂无事件信号'}
            scanning={isLoading}
          />
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
  decision,
  riskModules,
  isLoading = false,
}: {
  analysis?: FundAnalysis
  decision?: FundAnalysisDecisionView | null
  riskModules: FundAnalysisModuleScore[]
  isLoading?: boolean
}) {
  const view = decision || buildFundAnalysisDecision(analysis)

  return (
    <section className="glass flex h-full flex-col rounded-3xl p-5 md:p-6">
      <SectionHeading
        icon={<ShieldAlert className="h-4 w-4 text-amber-200" />}
        title="风险拆解"
        description="先看统一风险原因，再展开风险模块分。"
      />

      <div className="mt-4 space-y-4">
        <div className="rounded-2xl border border-amber-500/25 bg-amber-500/10 p-5">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <div className="text-[11px] font-semibold tracking-[0.22em] text-amber-100/80">RISK LEVEL</div>
              <div className="mt-2 text-3xl font-black text-theme-primary">{view?.riskLabel || riskLevelLabel(analysis?.risk_level)}</div>
            </div>
            <div className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 px-4 py-3 text-sm text-theme-secondary">
              总分 <span className="ml-2 text-lg font-bold text-theme-primary">{view?.scoreLabel || formatAnalysisScore(analysis?.total_score)}</span>
            </div>
          </div>
        </div>

        <div className="rounded-2xl border border-amber-500/20 bg-amber-500/8 p-4">
          <div className="mb-3 flex items-center gap-2 text-sm font-semibold text-theme-primary">
            <AlertTriangle className="h-4 w-4 text-amber-200" />
            风险原因链
          </div>
          <FundAnalysisDecisionList
            points={view?.riskReasons || []}
            emptyText={isLoading ? '风险原因正在生成。' : '当前没有明显反方证据。'}
            compact
          />
        </div>

        <div className={cn('grid gap-3 md:grid-cols-2', riskModules.length === 0 && 'block')}>
          {riskModules.length === 0 ? (
            <EmptyPanel
              code={isLoading ? 'RISK_MODULE_SYNC' : 'RISK_MODULE_EMPTY'}
              text={isLoading ? '正在拆解风险模块' : '风险模块未返回'}
              scanning={isLoading}
            />
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

function EmptyPanel({
  text,
  code = 'DATA_LINK_OFFLINE',
  scanning = false,
}: {
  text: string
  code?: string
  scanning?: boolean
}) {
  return (
    <div className="relative flex min-h-[6rem] w-full overflow-hidden rounded-2xl border border-dashed border-[var(--card-border)] bg-[var(--card-bg)]/25 px-4 py-5 text-center text-sm text-theme-muted">
      {scanning && <div className="pointer-events-none absolute inset-y-0 left-[-40%] w-1/2 animate-[pulse_1.8s_ease-in-out_infinite] bg-gradient-to-r from-transparent via-cyan-300/10 to-transparent" />}
      <div className="relative z-10 m-auto">
        <div className="font-mono text-[11px] tracking-[0.22em] text-cyan-100/55">[ {code} ]</div>
        <div className="mt-2 text-theme-muted">{text}</div>
      </div>
    </div>
  )
}

function EmptyInline({ text }: { text: string }) {
  return <div className="text-sm text-theme-muted">{text}</div>
}

function RadarBasePlaceholder({
  code,
  text,
  scanning = false,
}: {
  code: string
  text: string
  scanning?: boolean
}) {
  const center = 96
  const radius = 68
  const gridLevels = [0.33, 0.66, 1]
  const axes = Array.from({ length: 6 }, (_, index) => index)

  return (
    <div className="mt-5 grid items-center gap-5 md:grid-cols-[13rem_minmax(0,1fr)]">
      <div className="relative mx-auto h-52 w-52 overflow-hidden rounded-full border border-cyan-300/10 bg-[radial-gradient(circle_at_center,rgba(34,211,238,0.12),rgba(15,23,42,0)_64%)]">
        <svg viewBox="0 0 192 192" className="h-full w-full">
          {gridLevels.map((level) => (
            <circle key={level} cx={center} cy={center} r={radius * level} fill="none" stroke="rgba(148,163,184,.22)" strokeWidth="1" />
          ))}
          {axes.map((axis) => {
            const angle = -Math.PI / 2 + (axis * Math.PI * 2) / axes.length
            return (
              <line
                key={axis}
                x1={center}
                y1={center}
                x2={center + Math.cos(angle) * radius}
                y2={center + Math.sin(angle) * radius}
                stroke="rgba(148,163,184,.18)"
                strokeWidth="1"
              />
            )
          })}
        </svg>
        <div className={cn('absolute inset-3 rounded-full border border-cyan-200/10', scanning && 'animate-[spin_6s_linear_infinite] border-t-cyan-200/45')} />
        <div className="absolute left-1/2 top-1/2 h-2 w-2 -translate-x-1/2 -translate-y-1/2 rounded-full bg-cyan-200/55 shadow-[0_0_24px_rgba(34,211,238,0.45)]" />
      </div>
      <EmptyPanel code={code} text={text} scanning={scanning} />
    </div>
  )
}

function EvidenceSkeletonList({ amber = false }: { amber?: boolean }) {
  return (
    <div className="space-y-3">
      {[0, 1].map((item) => (
        <div key={item} className={cn('min-h-[7.5rem] rounded-2xl border p-4', amber ? 'border-amber-500/15 bg-amber-500/10' : 'border-cyan-500/15 bg-cyan-500/10')}>
          <div className="h-3 w-24 rounded-full bg-slate-400/15" />
          <div className="mt-4 h-2.5 w-full rounded-full bg-slate-400/10" />
          <div className="mt-2 h-2.5 w-4/5 rounded-full bg-slate-400/10" />
          <div className="mt-4 font-mono text-[10px] tracking-[0.2em] text-theme-muted opacity-70">[ EVIDENCE_STREAM_PENDING ]</div>
        </div>
      ))}
    </div>
  )
}

function EventSkeletonGrid() {
  return (
    <div className="grid gap-3 sm:grid-cols-2">
      {[0, 1, 2, 3].map((item) => (
        <div key={item} className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-4">
          <div className="flex gap-2">
            <div className="h-5 w-16 rounded-full bg-slate-400/10" />
            <div className="h-5 w-14 rounded-full bg-slate-400/10" />
          </div>
          <div className="mt-4 h-3 w-3/4 rounded-full bg-slate-400/15" />
          <div className="mt-3 h-2.5 w-full rounded-full bg-slate-400/10" />
          <div className="mt-2 h-2.5 w-2/3 rounded-full bg-slate-400/10" />
        </div>
      ))}
    </div>
  )
}
