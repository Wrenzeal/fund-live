'use client'

import Link from 'next/link'
import type { ReactNode } from 'react'
import { Activity, ArrowRight, BarChart3, Clock, LineChart, RefreshCw, Search, ShieldCheck, TrendingUp } from 'lucide-react'

import type { Fund, FundAnalysis, FundEstimate, FundHoldingsDisplayItem, HoldingDetail } from '@/hooks/use-fund-data'
import { formatTimeUntil, useMarketStatus } from '@/hooks/use-market-status'
import { FundSearch } from '@/components/fund-search'
import { Surface } from '@/components/ui/surface'
import { cn, formatCurrency, formatPercent } from '@/lib/utils'

interface HomeVisualShellProps {
  currentFundId: string
  estimate: FundEstimate | undefined
  fund: Fund | undefined
  lastUpdated: Date | null
  analysis: FundAnalysis | undefined
  marketStatus: ReturnType<typeof useMarketStatus>
  marketStatusLabel: string
  isCallAuction: boolean
  isTrading: boolean
  isLoading: boolean
  isWarming: boolean
  isValidating: boolean
  refreshInterval: number
  holdingsDisplayLevel: 'stock_layer' | 'target_layer'
  displayHoldingCoverageCount: number
  displayHoldingRatio: number
  lookthroughAvailable: boolean
  topContributors: HoldingDetail[]
  topDisplayItems: FundHoldingsDisplayItem[]
  onSelect: (id: string) => void
  onRefresh: () => void
}

export function HomeVisualShell(props: HomeVisualShellProps) {
  return (
    <section className="relative isolate overflow-hidden rounded-[2.25rem] border border-[var(--card-border)] bg-[var(--card-bg)]/40 p-4 shadow-[0_32px_90px_rgba(2,8,23,0.34)] md:p-6 lg:p-8">
      <div className="pointer-events-none absolute inset-0 -z-10 bg-[radial-gradient(circle_at_18%_12%,rgba(56,189,248,0.20),transparent_28rem),radial-gradient(circle_at_78%_20%,rgba(244,63,94,0.10),transparent_24rem),linear-gradient(135deg,rgba(15,23,42,0.78),rgba(2,6,23,0.20))]" />
      <div className="pointer-events-none absolute inset-x-8 top-0 -z-10 h-px bg-gradient-to-r from-transparent via-cyan-300/45 to-transparent" />

      <div className="grid gap-6 lg:grid-cols-[minmax(0,0.95fr)_minmax(24rem,1.05fr)] lg:items-stretch">
        <div className="flex min-h-[34rem] flex-col justify-between gap-8 rounded-[1.75rem] border border-white/10 bg-slate-950/35 p-6 md:p-8">
          <div className="space-y-7">
            <div className="inline-flex w-fit items-center gap-2 rounded-full border border-cyan-400/25 bg-cyan-400/10 px-4 py-2 text-sm font-medium text-cyan-100 shadow-[0_12px_30px_rgba(14,165,233,0.10)]">
              <Activity className="h-4 w-4" />
              真实基金估值工作台
            </div>

            <div className="max-w-3xl space-y-5">
              <h1 className="text-balance text-5xl font-black leading-[0.95] tracking-[-0.055em] text-theme-primary sm:text-6xl lg:text-7xl">
                先看清今天涨了多少，再决定下一步。
              </h1>
              <p className="max-w-[42rem] text-pretty text-base leading-7 text-theme-secondary sm:text-lg">
                FundLive 把实时估值、重仓贡献、行业主题和量化证据放在同一个视野里。你不用在多个页面之间来回比对。
              </p>
            </div>

            <div className="max-w-2xl rounded-3xl border border-cyan-300/20 bg-slate-950/45 p-3 shadow-[0_18px_42px_rgba(2,8,23,0.22)] backdrop-blur">
              <div className="mb-3 flex items-center gap-2 px-1 text-sm font-medium text-theme-secondary">
                <Search className="h-4 w-4 text-cyan-300" />
                输入基金代码或名称，立即切换估值视角
              </div>
              <FundSearch onSelect={props.onSelect} currentFundId={props.currentFundId} />
            </div>
          </div>

          <div className="grid gap-3 sm:grid-cols-3">
            <HeroStat icon={<LineChart className="h-4 w-4" />} label="估值口径" value="实时快照" />
            <HeroStat icon={<ShieldCheck className="h-4 w-4" />} label="建议边界" value="量化观察" />
            <HeroStat icon={<Clock className="h-4 w-4" />} label="刷新节奏" value={props.isTrading ? `${props.refreshInterval / 1000}s` : '按交易时段'} />
          </div>
        </div>

        <HeroTerminal {...props} />
      </div>
    </section>
  )
}

export function HomeInsightRail({
  marketStatus,
  marketStatusLabel,
  holdingsDisplayLevel,
  displayHoldingCoverageCount,
  displayHoldingRatio,
  isCallAuction,
  estimate,
  lookthroughAvailable,
  topContributors,
  topDisplayItems,
}: Pick<
  HomeVisualShellProps,
  | 'marketStatus'
  | 'marketStatusLabel'
  | 'holdingsDisplayLevel'
  | 'displayHoldingCoverageCount'
  | 'displayHoldingRatio'
  | 'isCallAuction'
  | 'estimate'
  | 'lookthroughAvailable'
  | 'topContributors'
  | 'topDisplayItems'
>) {
  return (
    <div className="grid gap-4 lg:grid-cols-2 xl:grid-cols-1">
      <Surface padding="md" radius="lg" className="overflow-hidden">
        <div className="mb-5 flex items-center gap-3">
          <div className="rounded-2xl bg-cyan-500/15 p-3">
            <BarChart3 className="h-5 w-5 text-cyan-300" />
          </div>
          <div>
            <h3 className="font-semibold text-theme-primary">市场与数据口径</h3>
            <p className="mt-1 text-xs text-theme-muted">当前基金上下文的刷新状态</p>
          </div>
        </div>
        <div className="space-y-3">
          <RailRow label="交易状态" value={marketStatusLabel} valueClassName={marketStatus.mounted && marketStatus.isTrading ? 'market-open' : 'market-closed'} />
          {marketStatus.mounted && !marketStatus.isTrading && marketStatus.timeUntilNextSession > 0 && (
            <RailRow label="距开盘" value={formatTimeUntil(marketStatus.timeUntilNextSession)} />
          )}
          <RailRow
            label={holdingsDisplayLevel === 'target_layer' ? '追踪目标数' : '重仓股覆盖'}
            value={holdingsDisplayLevel === 'target_layer'
              ? `${displayHoldingCoverageCount} 个`
              : isCallAuction
                ? `${displayHoldingCoverageCount} / 10`
                : `${estimate?.holding_details?.length || 0} / 10`}
          />
          <RailRow
            label={holdingsDisplayLevel === 'target_layer' ? '目标层级' : '持仓占比'}
            value={holdingsDisplayLevel === 'target_layer'
              ? '下一层目标'
              : isCallAuction
                ? `${displayHoldingRatio.toFixed(2)}%`
                : `${parseFloat(estimate?.total_hold_ratio || '0').toFixed(2)}%`}
          />
          <RailRow
            label="数据来源"
            value={holdingsDisplayLevel === 'target_layer'
              ? (lookthroughAvailable ? '追踪目标' : '跟踪标的')
              : isCallAuction
                ? '静态持仓'
                : (estimate?.data_source || 'N/A')}
            valueClassName="text-cyan-300"
          />
        </div>
      </Surface>

      <Surface padding="md" radius="lg" className="overflow-hidden">
        <div className="mb-5 flex items-center gap-3">
          <div className="rounded-2xl bg-[var(--accent-up)]/15 p-3">
            <TrendingUp className="h-5 w-5 text-up" />
          </div>
          <div>
            <h3 className="font-semibold text-theme-primary">
              {holdingsDisplayLevel === 'target_layer'
                ? '追踪目标 TOP'
                : isCallAuction
                  ? '重仓股 TOP3'
                  : '涨幅贡献 TOP3'}
            </h3>
            <p className="mt-1 text-xs text-theme-muted">保留真实持仓与贡献数据</p>
          </div>
        </div>
        <TopList
          holdingsDisplayLevel={holdingsDisplayLevel}
          isCallAuction={isCallAuction}
          topContributors={topContributors}
          topDisplayItems={topDisplayItems}
        />
      </Surface>
    </div>
  )
}

function HeroTerminal({
  estimate,
  fund,
  lastUpdated,
  analysis,
  marketStatus,
  marketStatusLabel,
  isCallAuction,
  isLoading,
  isWarming,
  isValidating,
  holdingsDisplayLevel,
  displayHoldingCoverageCount,
  displayHoldingRatio,
  lookthroughAvailable,
  topContributors,
  topDisplayItems,
  onRefresh,
}: HomeVisualShellProps) {
  const changeInfo = isCallAuction ? { text: '-', isPositive: false } : formatPercent(estimate?.change_percent)
  const changeValue = parseFloat(estimate?.change_percent || '0')
  const isPositive = changeValue >= 0
  const parsedScore = analysis?.total_score ? parseFloat(analysis.total_score) : Number.NaN
  const analysisScore = Number.isFinite(parsedScore) ? parsedScore.toFixed(1) : '等待分析'
  const analysisTone = analysis?.summary || riskLabel(analysis?.risk_level) || '证据生成中'
  const lastUpdatedText = lastUpdated && !isCallAuction
    ? lastUpdated.toLocaleTimeString('zh-CN')
    : isCallAuction
      ? '09:30 后更新'
      : '等待数据'
  const coverageText = holdingsDisplayLevel === 'target_layer'
    ? `${displayHoldingCoverageCount} 个目标`
    : isCallAuction
      ? `${displayHoldingRatio.toFixed(2)}%`
      : `${parseFloat(estimate?.total_hold_ratio || '0').toFixed(2)}%`
  const topRows = buildTopRows(holdingsDisplayLevel, isCallAuction, topContributors, topDisplayItems)

  return (
    <div className="relative overflow-hidden rounded-[1.75rem] border border-white/10 bg-slate-950/72 p-4 shadow-[0_28px_80px_rgba(2,8,23,0.42)] backdrop-blur-xl md:p-5">
      <div className="absolute inset-0 bg-[linear-gradient(135deg,rgba(56,189,248,0.08),transparent_36%),radial-gradient(circle_at_92%_8%,rgba(244,63,94,0.13),transparent_18rem)]" />
      <div className="relative z-10 space-y-4">
        <div className="flex items-start justify-between gap-4 rounded-3xl border border-white/10 bg-white/[0.035] p-4">
          <div className="min-w-0">
            <div className="flex items-center gap-2 text-sm text-theme-muted">
              <span className={cn('h-2 w-2 rounded-full', marketStatus.mounted && marketStatus.isTrading ? 'bg-emerald-400' : 'bg-amber-300')} />
              {marketStatusLabel}
            </div>
            <h2 className="mt-3 truncate text-2xl font-black tracking-[-0.035em] text-theme-primary sm:text-3xl">
              {isCallAuction ? '集合竞价中' : estimate?.fund_name || fund?.name || '选择基金'}
            </h2>
            <p className="mt-1 text-sm text-theme-secondary">
              {isCallAuction ? '等待开盘后恢复动态估值' : estimate?.fund_id || fund?.id || '搜索基金开始查看'}
            </p>
          </div>
          <button
            type="button"
            onClick={onRefresh}
            disabled={isValidating || isWarming}
            className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl border border-cyan-300/25 bg-cyan-300/10 text-cyan-100 transition-all duration-200 hover:-translate-y-0.5 hover:bg-cyan-300/15 active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-60"
            title="手动刷新"
          >
            <RefreshCw className={cn('h-4 w-4', (isValidating || isLoading || isWarming) && 'animate-spin')} />
          </button>
        </div>

        <div className="grid gap-3 sm:grid-cols-[minmax(0,1.1fr)_minmax(12rem,0.9fr)]">
          <div className="rounded-3xl border border-white/10 bg-[var(--background)]/55 p-5">
            <div className="text-sm text-theme-muted">实时预估涨跌幅</div>
            <div className={cn('mt-4 text-6xl font-black tracking-[-0.06em] sm:text-7xl', isPositive ? 'text-up' : 'text-down')}>
              {isLoading && !estimate && !isCallAuction ? '加载中' : changeInfo.text}
            </div>
            <div className="mt-5 grid grid-cols-2 gap-3 text-sm">
              <MetricPill label="预估净值" value={isCallAuction ? '-' : formatCurrency(estimate?.estimate_nav)} />
              <MetricPill label="昨日净值" value={isCallAuction ? '-' : formatCurrency(estimate?.prev_nav)} />
            </div>
          </div>

          <div className="grid gap-3">
            <MetricPill label="持仓覆盖" value={coverageText} large />
            <MetricPill
              label={holdingsDisplayLevel === 'target_layer' ? '展示层级' : '数据来源'}
              value={holdingsDisplayLevel === 'target_layer' ? (lookthroughAvailable ? '追踪目标' : '跟踪标的') : isCallAuction ? '静态持仓' : estimate?.data_source || 'N/A'}
              large
            />
            <MetricPill label="更新时间" value={lastUpdatedText} large />
          </div>
        </div>

        <div className="grid gap-3 lg:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)]">
          <div className="rounded-3xl border border-white/10 bg-white/[0.035] p-4">
            <div className="mb-3 flex items-center justify-between gap-3">
              <div>
                <div className="text-sm font-semibold text-theme-primary">量化观察</div>
                <div className="mt-1 text-xs text-theme-muted">规则评分不等于交易指令</div>
              </div>
              <div className="rounded-2xl border border-cyan-300/20 bg-cyan-300/10 px-3 py-2 text-right">
                <div className="text-xs text-cyan-200">总分</div>
                <div className="text-xl font-black text-theme-primary">{analysisScore}</div>
              </div>
            </div>
            <p className="line-clamp-3 text-sm leading-6 text-theme-secondary">{analysisTone}</p>
          </div>

          <div className="rounded-3xl border border-white/10 bg-white/[0.035] p-4">
            <div className="mb-3 flex items-center justify-between gap-3">
              <div className="text-sm font-semibold text-theme-primary">
                {holdingsDisplayLevel === 'target_layer' ? '追踪目标' : isCallAuction ? '重仓股' : '贡献排行'}
              </div>
              <span className="text-xs text-theme-muted">Top 3</span>
            </div>
            <div className="space-y-2">
              {topRows.length > 0 ? topRows.slice(0, 3).map((row, index) => (
                <div key={row.key} className="flex items-center justify-between gap-3 rounded-2xl bg-[var(--background)]/45 px-3 py-2">
                  <div className="flex min-w-0 items-center gap-3">
                    <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-xl bg-cyan-300/10 text-xs font-semibold text-cyan-200">{index + 1}</span>
                    <span className="truncate text-sm font-medium text-theme-primary">{row.name}</span>
                  </div>
                  <span className={cn('shrink-0 text-sm font-semibold', row.positive ? 'text-up' : 'text-down')}>{row.meta}</span>
                </div>
              )) : (
                <div className="rounded-2xl bg-[var(--background)]/45 px-3 py-6 text-center text-sm text-theme-muted">等待持仓数据</div>
              )}
            </div>
          </div>
        </div>

        <div className="flex flex-wrap items-center justify-between gap-3 rounded-3xl border border-white/10 bg-white/[0.035] px-4 py-3 text-sm text-theme-secondary">
          <span>{isCallAuction ? '集合竞价阶段保留固定持仓信息' : '估值、图表、行业和证据链共享当前基金上下文'}</span>
          <Link href="/analysis/rankings" className="inline-flex items-center gap-1 font-medium text-cyan-200 transition-colors hover:text-cyan-100">
            看量化排行榜
            <ArrowRight className="h-4 w-4" />
          </Link>
        </div>
      </div>
    </div>
  )
}

function HeroStat({ icon, label, value }: { icon: ReactNode; label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-white/10 bg-white/[0.045] px-4 py-3 text-sm backdrop-blur">
      <div className="flex items-center gap-2 text-theme-muted">
        <span className="text-cyan-300">{icon}</span>
        {label}
      </div>
      <div className="mt-2 text-base font-semibold text-theme-primary">{value}</div>
    </div>
  )
}

function MetricPill({ label, value, large = false }: { label: string; value: string; large?: boolean }) {
  return (
    <div className={cn('rounded-2xl border border-white/10 bg-white/[0.04] px-4 py-3', large && 'min-h-[5.1rem]')}>
      <div className="text-xs text-theme-muted">{label}</div>
      <div className={cn('mt-1 truncate font-semibold text-theme-primary', large ? 'text-lg' : 'text-base')}>{value}</div>
    </div>
  )
}

function RailRow({ label, value, valueClassName }: { label: string; value: string; valueClassName?: string }) {
  return (
    <div className="flex items-center justify-between gap-4 rounded-2xl border border-[var(--card-border)] bg-[var(--input-bg)]/45 px-4 py-3 text-sm">
      <span className="text-theme-secondary">{label}</span>
      <span className={cn('text-right font-semibold text-theme-primary', valueClassName)}>{value}</span>
    </div>
  )
}

function TopList({
  holdingsDisplayLevel,
  isCallAuction,
  topContributors,
  topDisplayItems,
}: {
  holdingsDisplayLevel: 'stock_layer' | 'target_layer'
  isCallAuction: boolean
  topContributors: HoldingDetail[]
  topDisplayItems: FundHoldingsDisplayItem[]
}) {
  if (holdingsDisplayLevel !== 'target_layer' && !isCallAuction) {
    if (topContributors.length === 0) {
      return <p className="rounded-2xl bg-[var(--input-bg)]/45 py-6 text-center text-sm text-theme-muted">暂无数据</p>
    }

    return (
      <div className="space-y-2">
        {topContributors.map((holding, index) => {
          const contrib = parseFloat(holding.contribution)
          const isPositive = contrib >= 0
          return (
            <div key={holding.stock_code} className="flex items-center justify-between gap-3 rounded-2xl border border-[var(--card-border)] bg-[var(--input-bg)]/45 px-4 py-3">
              <div className="flex min-w-0 items-center gap-3">
                <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-xl bg-cyan-500/10 text-xs font-semibold text-cyan-300">{index + 1}</span>
                <span className="truncate text-sm font-medium text-theme-primary">{holding.stock_name}</span>
              </div>
              <span className={cn('shrink-0 text-sm font-semibold', isPositive ? 'text-up' : 'text-down')}>
                {isPositive ? '+' : ''}{contrib.toFixed(4)}%
              </span>
            </div>
          )
        })}
      </div>
    )
  }

  if (topDisplayItems.length === 0) {
    return <p className="rounded-2xl bg-[var(--input-bg)]/45 py-6 text-center text-sm text-theme-muted">暂无持仓数据</p>
  }

  return (
    <div className="space-y-2">
      {topDisplayItems.map((holding, index) => (
        <div key={`${holding.item_type}:${holding.code || holding.name}:${index}`} className="flex items-center justify-between gap-3 rounded-2xl border border-[var(--card-border)] bg-[var(--input-bg)]/45 px-4 py-3">
          <div className="flex min-w-0 items-center gap-3">
            <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-xl bg-cyan-500/10 text-xs font-semibold text-cyan-300">{index + 1}</span>
            <span className="truncate text-sm font-medium text-theme-primary">{holding.name}</span>
          </div>
          <span className="shrink-0 text-sm font-semibold text-theme-secondary">
            {holdingsDisplayLevel === 'target_layer'
              ? holding.target_type === 'etf_fund'
                ? 'ETF'
                : holding.target_type === 'index'
                  ? '指数'
                  : '基金'
              : `${parseFloat(holding.holding_ratio || '0').toFixed(2)}%`}
          </span>
        </div>
      ))}
      {isCallAuction && (
        <p className="pt-2 text-xs leading-5 text-theme-muted">
          集合竞价阶段保留固定持仓与占比信息，贡献值等待 09:30 开盘后恢复。
        </p>
      )}
      {holdingsDisplayLevel === 'target_layer' && (
        <p className="pt-2 text-xs leading-5 text-theme-muted">
          默认只展示下一层追踪目标；底层股票用于估值计算，不在这里继续下钻。
        </p>
      )}
    </div>
  )
}

function buildTopRows(
  holdingsDisplayLevel: 'stock_layer' | 'target_layer',
  isCallAuction: boolean,
  topContributors: HoldingDetail[],
  topDisplayItems: FundHoldingsDisplayItem[]
) {
  if (holdingsDisplayLevel === 'target_layer' || isCallAuction) {
    return topDisplayItems.map((holding, index) => ({
      key: `${holding.item_type}:${holding.code || holding.name}:${index}`,
      name: holding.name,
      meta: holdingsDisplayLevel === 'target_layer'
        ? holding.target_type === 'etf_fund'
          ? 'ETF'
          : holding.target_type === 'index'
            ? '指数'
            : '基金'
        : `${parseFloat(holding.holding_ratio || holding.weight_percent || '0').toFixed(2)}%`,
      positive: true,
    }))
  }

  return topContributors.map((holding) => {
    const contribution = parseFloat(holding.contribution)
    return {
      key: holding.stock_code,
      name: holding.stock_name,
      meta: `${contribution >= 0 ? '+' : ''}${contribution.toFixed(4)}%`,
      positive: contribution >= 0,
    }
  })
}

function riskLabel(risk?: FundAnalysis['risk_level']) {
  if (risk === 'high') return '风险暴露偏高'
  if (risk === 'medium') return '风险处于中位'
  if (risk === 'low') return '风险相对较低'
  return ''
}
