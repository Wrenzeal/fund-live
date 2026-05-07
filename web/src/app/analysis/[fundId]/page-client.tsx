'use client'

import type { ReactNode } from 'react'
import Link from 'next/link'
import { AlertTriangle, ArrowLeft, BarChart3, CalendarClock, FileText, Layers3, ShieldCheck, ShieldAlert, Sparkles, TimerReset } from 'lucide-react'
import { EstimateCard } from '@/components/estimate-card'
import { FundAnalysisCard } from '@/components/fund-analysis-card'
import { FundSectorCard } from '@/components/fund-sector-card'
import { HoldingsTable } from '@/components/holdings-table'
import { TargetETFHoldingsCard } from '@/components/target-etf-holdings-card'
import { useFundAnalysis, useFundDashboard, useFundHoldings, type FundAnalysis, type FundAnalysisEventImpact, type FundAnalysisModuleScore } from '@/hooks/use-fund-data'
import { cn } from '@/lib/utils'

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
  const timelineEvents = (analysis?.event_impacts || []).slice().sort((left, right) => timelineRank(left) - timelineRank(right))
  const quarterlyEvents = (analysis?.event_impacts || []).filter((item) => item.horizon === 'quarterly')
  const exposureEvents = (analysis?.event_impacts || []).filter((item) => item.target_scope === 'exposure')
  const riskModules = (analysis?.module_scores || []).filter((item) => isRiskModule(item.code))
  const positiveModules = (analysis?.module_scores || []).filter((item) => !isRiskModule(item.code))

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

        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <OverviewCard
            icon={<BarChart3 className="h-4 w-4 text-cyan-200" />}
            title="当前分析口径"
            value={analysis?.analysis_basis || '--'}
            description={analysis?.analysis_type === 'tracked_etf' ? '联接基金优先复用目标 ETF 口径' : '当前量化分析以单快照规则打分为主'}
          />
          <OverviewCard
            icon={<ShieldCheck className="h-4 w-4 text-emerald-200" />}
            title="风险与覆盖"
            value={`${riskLabel(analysis?.risk_level)} / ${confidenceLabel(analysis?.confidence)}`}
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
          <div className="grid gap-6 xl:grid-cols-[minmax(0,1.05fr)_minmax(24rem,0.95fr)]">
            <EstimateCard
              estimate={estimate}
              fund={fund}
              isLoading={isLoading}
              isValidating={isValidating}
              lastUpdated={lastUpdated}
            />

            <section className="glass rounded-3xl p-6">
              <div className="mb-4 flex items-center gap-3">
                <div className="rounded-2xl bg-cyan-500/15 p-3">
                  <Sparkles className="h-5 w-5 text-cyan-200" />
                </div>
                <div>
                  <div className="text-sm font-semibold text-theme-primary">看板说明</div>
                  <div className="text-xs text-theme-muted">先把整体信息架构搭出来，后续再逐项细化数据源与评分权重。</div>
                </div>
              </div>

              <div className="space-y-4 text-sm text-theme-secondary">
                <div className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-4">
                  <div className="font-medium text-theme-primary">当前已覆盖</div>
                  <ul className="mt-3 space-y-2">
                    <Bullet>综合评分与加 / 平 / 减分布</Bullet>
                    <Bullet>六模块评分卡与理由 / 风险提示</Bullet>
                    <Bullet>事件影响列表（披露新鲜度、分析口径、主暴露集中度、识别覆盖）</Bullet>
                    <Bullet>行业板块 / 主题分类 / 持仓明细联动展示</Bullet>
                  </ul>
                </div>

                <div className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-4">
                  <div className="font-medium text-theme-primary">下一步细化方向</div>
                  <ul className="mt-3 space-y-2">
                    <Bullet>季报变化：前十大持仓、行业权重、主题暴露与风格漂移</Bullet>
                    <Bullet>事件源扩展：板块时事、重仓股公司动态、基金自身公告、ETF/指数层事件</Bullet>
                    <Bullet>校准样本池：按 ETF / 联接 / 主动权益 / QDII 逐只调权重</Bullet>
                    <Bullet>后续继续把同一套分析结果扩展到持仓页排序、自选页标签与排行榜联动</Bullet>
                  </ul>
                  <Link
                    href="/analysis/rankings"
                    className="mt-4 inline-flex items-center gap-2 text-sm text-cyan-200 transition-colors hover:text-cyan-100"
                  >
                    查看量化排行榜
                    <ArrowLeft className="h-4 w-4 rotate-180" />
                  </Link>
                </div>
              </div>
            </section>
          </div>

          <FundAnalysisCard analysis={analysis} isLoading={isAnalysisLoading} />

          <div className="grid gap-6 xl:grid-cols-[minmax(0,1.1fr)_minmax(22rem,0.9fr)]">
            <AnalysisTimelineCard events={timelineEvents} />
            <QuarterlyDiffCard events={quarterlyEvents} />
          </div>

          <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(20rem,0.95fr)]">
            <RiskBreakdownCard analysis={analysis} riskModules={riskModules} />
            <StructureComparisonCard
              sectorSnapshot={sectorSnapshot}
              themeSnapshot={themeSnapshot}
              exposureEvents={exposureEvents}
              positiveModules={positiveModules}
            />
          </div>

          <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(20rem,0.9fr)]">
            <FundSectorCard
              fund={fund}
              sectorSnapshot={sectorSnapshot}
              themeSnapshot={themeSnapshot}
            />

            <section className="glass rounded-3xl p-6">
              <div className="mb-4 text-sm font-semibold text-theme-primary">结构与数据视角</div>
              <div className="space-y-3 text-sm text-theme-secondary">
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
    <section className="glass rounded-2xl p-5">
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
    <div className="flex items-center justify-between gap-4 rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 px-4 py-3">
      <span className="text-theme-muted">{label}</span>
      <span className="text-right font-medium text-theme-primary">{value}</span>
    </div>
  )
}

function Bullet({ children }: { children: ReactNode }) {
  return (
    <li className="flex gap-3">
      <span className={cn('mt-2 inline-block h-1.5 w-1.5 shrink-0 rounded-full bg-cyan-300')} />
      <span>{children}</span>
    </li>
  )
}

function riskLabel(level?: string) {
  switch (level) {
    case 'low':
      return '低风险'
    case 'medium':
      return '中风险'
    case 'high':
      return '高风险'
    default:
      return '待判断'
  }
}

function confidenceLabel(level?: string) {
  switch (level) {
    case 'high':
      return '覆盖较高'
    case 'medium':
      return '覆盖一般'
    case 'low':
      return '覆盖有限'
    default:
      return '覆盖未知'
  }
}

function timelineRank(event: FundAnalysisEventImpact) {
  const horizonRank = event.horizon === 'intraday'
    ? 0
    : event.horizon === 'current'
      ? 1
      : event.horizon === 'quarterly'
        ? 2
        : 3
  const strengthRank = event.strength === 'high'
    ? 0
    : event.strength === 'medium'
      ? 1
      : 2
  return horizonRank * 10 + strengthRank
}

function horizonLabel(horizon?: string) {
  switch (horizon) {
    case 'intraday':
      return '盘中'
    case 'current':
      return '当前'
    case 'quarterly':
      return '季报变化'
    case 'medium_term':
      return '中期'
    default:
      return '事件'
  }
}

function strengthLabel(strength?: string) {
  switch (strength) {
    case 'high':
      return '高强度'
    case 'medium':
      return '中强度'
    case 'low':
      return '低强度'
    default:
      return '普通'
  }
}

function impactLabel(impact?: string) {
  switch (impact) {
    case 'positive':
      return '偏正向'
    case 'negative':
      return '偏负向'
    default:
      return '中性'
  }
}

function impactTone(impact?: string) {
  switch (impact) {
    case 'positive':
      return 'border-emerald-500/25 bg-emerald-500/10 text-emerald-100'
    case 'negative':
      return 'border-rose-500/25 bg-rose-500/10 text-rose-100'
    default:
      return 'border-cyan-500/25 bg-cyan-500/10 text-cyan-100'
  }
}

function formatScore(value?: string) {
  const parsed = Number.parseFloat(value || '')
  if (Number.isNaN(parsed)) {
    return '--'
  }
  return parsed.toFixed(1)
}

function isRiskModule(code: string) {
  return code === 'risk' || code === 'event'
}

function AnalysisTimelineCard({ events }: { events: FundAnalysisEventImpact[] }) {
  return (
    <section className="glass rounded-3xl p-6">
      <div className="mb-4 flex items-center gap-3">
        <div className="rounded-2xl bg-cyan-500/15 p-3">
          <CalendarClock className="h-5 w-5 text-cyan-200" />
        </div>
        <div>
          <div className="text-sm font-semibold text-theme-primary">事件时间线</div>
          <div className="text-xs text-theme-muted">按盘中 / 当前 / 季报变化 / 中期观察的顺序梳理影响链路。</div>
        </div>
      </div>

      <div className="space-y-3">
        {events.length === 0 ? (
          <EmptyPanel text="当前还没有可展开的事件时间线。" />
        ) : events.map((event, index) => (
          <div key={`${event.code}-${index}`} className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-4">
            <div className="flex flex-wrap items-center gap-2">
              <span className="rounded-full border border-[var(--input-border)] bg-[var(--input-bg)] px-2.5 py-1 text-[11px] text-theme-secondary">
                {horizonLabel(event.horizon)}
              </span>
              <span className="rounded-full border border-[var(--input-border)] bg-[var(--input-bg)] px-2.5 py-1 text-[11px] text-theme-secondary">
                {strengthLabel(event.strength)}
              </span>
              <span className={cn('rounded-full border px-2.5 py-1 text-[11px]', impactTone(event.impact))}>
                {impactLabel(event.impact)}
              </span>
            </div>
            <div className="mt-3 text-sm font-semibold text-theme-primary">{event.title}</div>
            <div className="mt-2 text-sm leading-6 text-theme-secondary">{event.summary}</div>
            {event.related_symbols && event.related_symbols.length > 0 && (
              <div className="mt-2 text-xs text-theme-muted">
                相关标的：{event.related_symbols.join(' / ')}
              </div>
            )}
          </div>
        ))}
      </div>
    </section>
  )
}

function QuarterlyDiffCard({ events }: { events: FundAnalysisEventImpact[] }) {
  return (
    <section className="glass rounded-3xl p-6">
      <div className="mb-4 flex items-center gap-3">
        <div className="rounded-2xl bg-fuchsia-500/15 p-3">
          <TimerReset className="h-5 w-5 text-fuchsia-200" />
        </div>
        <div>
          <div className="text-sm font-semibold text-theme-primary">季报变化</div>
          <div className="text-xs text-theme-muted">把上一季与当前季的持仓和主线变化单独拎出来看。</div>
        </div>
      </div>

      <div className="space-y-3">
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
    <section className="glass rounded-3xl p-6">
      <div className="mb-4 flex items-center gap-3">
        <div className="rounded-2xl bg-amber-500/15 p-3">
          <ShieldAlert className="h-5 w-5 text-amber-200" />
        </div>
        <div>
          <div className="text-sm font-semibold text-theme-primary">风险拆解</div>
          <div className="text-xs text-theme-muted">单独看风险相关模块、warning 与当前总风险等级。</div>
        </div>
      </div>

      <div className="grid gap-4 lg:grid-cols-[16rem_minmax(0,1fr)]">
        <div className="rounded-2xl border border-amber-500/20 bg-amber-500/10 p-4">
          <div className="text-xs tracking-[0.18em] text-theme-muted">CURRENT RISK</div>
          <div className="mt-3 text-3xl font-black text-theme-primary">{riskLabel(analysis?.risk_level)}</div>
          <div className="mt-2 text-sm text-theme-secondary">总分 {formatScore(analysis?.total_score)}</div>
        </div>

        <div className="space-y-3">
          {riskModules.length === 0 ? (
            <EmptyPanel text="当前没有单独可拆的风险模块。" />
          ) : riskModules.map((module) => (
            <div key={module.code} className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-4">
              <div className="flex items-center justify-between gap-4">
                <div className="text-sm font-semibold text-theme-primary">{module.name}</div>
                <div className="text-lg font-bold text-theme-primary">{formatScore(module.score)}</div>
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
    <section className="glass rounded-3xl p-6">
      <div className="mb-4 flex items-center gap-3">
        <div className="rounded-2xl bg-emerald-500/15 p-3">
          <Layers3 className="h-5 w-5 text-emerald-200" />
        </div>
        <div>
          <div className="text-sm font-semibold text-theme-primary">结构变化对比</div>
          <div className="text-xs text-theme-muted">把当前主行业 / 主主题和暴露变化事件放到一起看。</div>
        </div>
      </div>

      <div className="space-y-4">
        <div className="grid gap-3 sm:grid-cols-2">
          <MetaRow label="当前主行业" value={sectorSnapshot?.primary_sector_name || '--'} />
          <MetaRow label="当前主主题" value={themeSnapshot?.primary_theme_name || '--'} />
        </div>

        <div className="grid gap-4 lg:grid-cols-2">
          <div className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-4">
            <div className="mb-2 text-sm font-semibold text-theme-primary">行业结构 Top3</div>
            <div className="space-y-2">
              {topSectors.length === 0 ? <EmptyInline text="暂无行业结构数据" /> : topSectors.map((item) => (
                <div key={item.sector_name} className="flex items-center justify-between gap-3 text-sm">
                  <span className="text-theme-secondary">{item.sector_name}</span>
                  <span className="font-medium text-theme-primary">{item.weight_percent}</span>
                </div>
              ))}
            </div>
          </div>

          <div className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-4">
            <div className="mb-2 text-sm font-semibold text-theme-primary">主题结构 Top3</div>
            <div className="space-y-2">
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
            {positiveModules.slice(0, 3).map((module) => (
              <div key={module.code} className="text-sm leading-6 text-theme-secondary">
                <span className="font-medium text-theme-primary">{module.name}（{formatScore(module.score)}）：</span>
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
    <div className="rounded-2xl border border-dashed border-[var(--card-border)] px-4 py-8 text-center text-sm text-theme-muted">
      {text}
    </div>
  )
}

function EmptyInline({ text }: { text: string }) {
  return <div className="text-sm text-theme-muted">{text}</div>
}
