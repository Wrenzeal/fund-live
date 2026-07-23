'use client'

import { type ReactNode } from 'react'
import Link from 'next/link'
import { AlertTriangle, ArrowRight, BarChart3, Eye, Flame, RefreshCw, ShieldAlert, Sparkles } from 'lucide-react'
import { AnalysisReveal as RevealBlock, AnalysisSectionHeading as SectionHeading } from '@/components/analysis-layout'
import { AppTopBar } from '@/components/app-top-bar'
import { FundAnalysisBadge } from '@/components/fund-analysis-badge'
import { FundAnalysisEventHint } from '@/components/fund-analysis-event-hint'
import { LoadingSpinner } from '@/components/loading-indicator'
import { SiteFooter } from '@/components/site-footer'
import { useFundAnalysisRankings, type FundAnalysisRankingItem } from '@/hooks/use-fund-data'
import { cn } from '@/lib/utils'

function percentValue(value?: string) {
  const parsed = Number.parseFloat(value || '')
  if (Number.isNaN(parsed)) {
    return '--'
  }
  return `${parsed.toFixed(1)}%`
}

function scoreValue(value?: string) {
  const parsed = Number.parseFloat(value || '')
  if (Number.isNaN(parsed)) {
    return '--'
  }
  return parsed.toFixed(1)
}

type Accent = 'rose' | 'slate' | 'amber'

type RankingSectionConfig = {
  title: string
  shortTitle: string
  description: string
  items: FundAnalysisRankingItem[]
  accent: Accent
  metricLabel: string
  metricValue: (item: FundAnalysisRankingItem) => string
  icon: ReactNode
}

function accentTone(accent: Accent) {
  if (accent === 'rose') {
    return {
      panel: 'border-rose-500/25 bg-rose-500/10',
      soft: 'border-rose-500/20 bg-rose-500/10 text-rose-100',
      glow: 'from-rose-500/25 via-fuchsia-500/10 to-transparent',
    }
  }
  if (accent === 'amber') {
    return {
      panel: 'border-amber-500/25 bg-amber-500/10',
      soft: 'border-amber-500/20 bg-amber-500/10 text-amber-100',
      glow: 'from-amber-500/25 via-orange-500/10 to-transparent',
    }
  }
  return {
    panel: 'border-cyan-500/25 bg-cyan-500/10',
    soft: 'border-cyan-500/20 bg-cyan-500/10 text-cyan-100',
    glow: 'from-cyan-500/25 via-sky-500/10 to-transparent',
  }
}

function RankingsHero({
  generatedAt,
  isValidating,
  totalCount,
}: {
  generatedAt?: string
  isValidating: boolean
  totalCount: number
}) {
  return (
    <header className="overflow-hidden rounded-3xl border border-[var(--card-border)] bg-[var(--card-bg)]/45 p-4 shadow-[0_22px_60px_rgba(0,0,0,0.10)] md:rounded-[2rem] md:p-6">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between lg:gap-5">
        <div className="min-w-0">
          <div className="text-2xl font-black tracking-tight text-theme-primary md:text-3xl">量化排行榜</div>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-theme-secondary">
            按近期分析快照列出三类观察池：结构偏积极、适合继续观察、风险暴露偏高。点击基金查看完整看板。
          </p>
        </div>

        <div className="grid grid-cols-3 gap-2 text-xs text-theme-secondary lg:min-w-[26rem]">
          <HeroStat label="覆盖基金" value={`${totalCount}`} />
          <HeroStat label="生成时间" value={generatedAt ? new Date(generatedAt).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }) : '等待生成'} />
          <HeroStat label="状态" value={isValidating ? '刷新中' : '已就绪'} active={isValidating} />
        </div>
      </div>
    </header>
  )
}

function HeroStat({ label, value, active = false }: { label: string; value: string; active?: boolean }) {
  return (
    <div className={cn('rounded-xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 px-2 py-2.5 sm:rounded-2xl sm:px-3', active && 'border-cyan-500/25 bg-cyan-500/10 text-cyan-100')}>
      <div className="flex items-center gap-1 text-[10px] tracking-[0.08em] text-theme-muted sm:gap-1.5 sm:tracking-[0.18em]">
        {active && <RefreshCw className="h-3 w-3 animate-spin" />}
        {label}
      </div>
      <div className="mt-1 truncate text-xs font-semibold text-theme-primary sm:text-sm">{value}</div>
    </div>
  )
}

function RankingsOverview({ sections }: { sections: RankingSectionConfig[] }) {
  return (
    <section className="grid gap-3 md:grid-cols-3">
      {sections.map((section) => {
        const tone = accentTone(section.accent)
        const top = section.items[0]

        return (
          <Link
            key={section.title}
            href={top?.fund?.id ? `/analysis/${top.fund.id}` : '#rankings-list'}
            className={cn(
              'group relative overflow-hidden rounded-3xl border p-4 transition-all duration-300 hover:-translate-y-0.5',
              tone.panel
            )}
          >
            <div className={cn('pointer-events-none absolute inset-x-0 top-0 h-24 bg-gradient-to-b opacity-80', tone.glow)} />
            <div className="relative flex items-start justify-between gap-4">
              <div>
                <div className="flex items-center gap-2 text-xs font-semibold text-theme-primary">
                  {section.icon}
                  {section.shortTitle}
                </div>
                <div className="mt-3 text-3xl font-black text-theme-primary">{section.items.length}</div>
                <div className="mt-1 text-xs text-theme-muted">入榜数量</div>
              </div>
              <div className={cn('rounded-2xl border px-3 py-2 text-right', tone.soft)}>
                <div className="text-[10px] text-theme-muted">首位</div>
                <div className="mt-1 max-w-28 truncate text-sm font-semibold text-theme-primary">
                  {top?.fund?.name || '暂无'}
                </div>
              </div>
            </div>
            <div className="relative mt-4 line-clamp-2 text-xs leading-5 text-theme-secondary">{section.description}</div>
          </Link>
        )
      })}
    </section>
  )
}

function FeaturedRankingCard({
  item,
  config,
}: {
  item?: FundAnalysisRankingItem
  config: RankingSectionConfig
}) {
  const tone = accentTone(config.accent)

  return (
    <section className={cn('glass relative overflow-hidden rounded-3xl p-5 md:p-6', tone.panel)}>
      <div className={cn('pointer-events-none absolute -right-20 -top-20 h-52 w-52 rounded-full bg-gradient-to-br blur-3xl', tone.glow)} />
      <div className="relative flex flex-col gap-5 lg:flex-row lg:items-center lg:justify-between">
        <SectionHeading
          icon={config.icon}
          title={`${config.title} · 首位`}
          description="把最值得先点开的样本放大展示，剩余条目在下方分榜浏览。"
        />

        {item ? (
          <Link
            href={item.fund?.id ? `/analysis/${item.fund.id}` : '/analysis/rankings'}
            className="group grid gap-4 rounded-[1.75rem] border border-[var(--card-border)] bg-[var(--card-bg)]/50 p-4 transition-all duration-300 hover:-translate-y-0.5 hover:border-cyan-400/30 lg:min-w-[32rem] lg:grid-cols-[minmax(0,1fr)_9rem]"
          >
            <div className="min-w-0">
              <div className="truncate text-xl font-black text-theme-primary">{item.fund?.name || item.fund?.id || '未知基金'}</div>
              <div className="mt-1 text-xs text-theme-muted">{item.fund?.id || '--'} · {item.analysis?.analysis_basis || '口径待补'}</div>
              <div className="mt-3 flex flex-wrap gap-2">
                <FundAnalysisBadge analysis={item.analysis} showScore />
              </div>
              {item.analysis?.summary && (
                <p className="mt-3 line-clamp-2 text-sm leading-6 text-theme-secondary">{item.analysis.summary}</p>
              )}
            </div>
            <div className={cn('flex min-h-[8rem] flex-col justify-center rounded-2xl border px-4 py-3 text-right', tone.soft)}>
              <div className="text-[11px] tracking-[0.18em] text-theme-muted">{config.metricLabel}</div>
              <div className="mt-1 text-3xl font-black text-theme-primary">{config.metricValue(item)}</div>
              <div className="mt-2 text-xs text-theme-secondary">总分 {scoreValue(item.analysis?.total_score)}</div>
              <div className="mt-3 inline-flex items-center justify-end gap-1 text-xs text-theme-muted group-hover:text-theme-primary">
                看完整看板
                <ArrowRight className="h-3.5 w-3.5" />
              </div>
            </div>
          </Link>
        ) : (
          <div className="rounded-2xl border border-dashed border-[var(--card-border)] px-4 py-8 text-center text-sm text-theme-muted lg:min-w-[28rem]">
            暂无首位样本
          </div>
        )}
      </div>
    </section>
  )
}

function RankingItemCard({
  item,
  index,
  config,
}: {
  item: FundAnalysisRankingItem
  index: number
  config: RankingSectionConfig
}) {
  const tone = accentTone(config.accent)

  return (
    <Link
      href={item.fund?.id ? `/analysis/${item.fund.id}` : '/analysis/rankings'}
      className={cn(
        'group block rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-4 transition-all duration-300',
        'hover:-translate-y-0.5 hover:border-cyan-400/35 hover:shadow-[0_12px_24px_rgba(34,211,238,0.08)]'
      )}
    >
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0 flex-1">
          <div className="flex items-start gap-3">
            <span className={cn('mt-0.5 rounded-full border px-2.5 py-1 text-xs font-semibold', tone.soft)}>#{index + 1}</span>
            <div className="min-w-0">
              <div className="truncate text-base font-semibold text-theme-primary">
                {item.fund?.name || item.fund?.id || '未知基金'}
              </div>
              <div className="mt-1 text-xs text-theme-muted">
                {item.fund?.id || '--'} · {item.analysis?.analysis_basis || '口径待补'}
              </div>
            </div>
          </div>

          <div className="mt-3 flex flex-wrap items-center gap-2">
            <FundAnalysisBadge analysis={item.analysis} showScore />
          </div>
          <div className="mt-2">
            <FundAnalysisEventHint analysis={item.analysis} />
          </div>

          {item.analysis?.summary && (
            <p className="mt-3 line-clamp-2 text-sm leading-6 text-theme-secondary">
              {item.analysis.summary}
            </p>
          )}
        </div>

        <div className={cn('shrink-0 rounded-2xl border px-4 py-3 text-right sm:min-w-[8rem]', tone.soft)}>
          <div className="text-[11px] tracking-[0.18em] text-theme-muted">{config.metricLabel}</div>
          <div className="mt-1 text-2xl font-bold text-theme-primary">{config.metricValue(item)}</div>
          <div className="mt-2 text-xs text-theme-secondary">总分 {scoreValue(item.analysis?.total_score)}</div>
        </div>
      </div>

      <div className="mt-4 flex items-center justify-end gap-2 text-xs text-theme-muted group-hover:text-theme-primary">
        <span>查看完整量化看板</span>
        <ArrowRight className="h-3.5 w-3.5" />
      </div>
    </Link>
  )
}

function RankingsSection({ config }: { config: RankingSectionConfig }) {
  return (
    <section className="glass rounded-3xl p-5 md:p-6">
      <SectionHeading
        icon={config.icon}
        title={config.title}
        description={config.description}
      />

      <div className="mt-4 space-y-3">
        {config.items.length === 0 ? (
          <div className="rounded-2xl border border-dashed border-[var(--card-border)] bg-[var(--card-bg)]/35 px-4 py-8 text-center text-sm text-theme-muted">
            暂无可展示结果
          </div>
        ) : config.items.map((item, index) => (
          <RankingItemCard key={item.fund?.id || `${config.title}-${index}`} item={item} index={index} config={config} />
        ))}
      </div>
    </section>
  )
}

function MethodNote() {
  return (
    <section className="rounded-3xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-5">
      <SectionHeading
        icon={<Eye className="h-4 w-4 text-cyan-200" />}
        title="榜单说明"
        description="排行榜只做入口和筛选，不派生第二套建议逻辑。"
      />
      <div className="mt-4 grid gap-4 lg:grid-cols-3">
        <p className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-4 text-sm leading-6 text-theme-secondary">
          <Sparkles className="mb-2 h-4 w-4 text-cyan-200" />
          榜单汇总近期分析快照；证据、事件和限制请查看详情页。
        </p>
        <p className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-4 text-sm leading-6 text-theme-secondary">
          <ShieldAlert className="mb-2 h-4 w-4 text-amber-200" />
          “结构偏积极 / 风险偏高”是量化观察，不是交易指令；低可信样本需要结合持仓覆盖继续复核。
        </p>
        <p className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-4 text-sm leading-6 text-theme-secondary">
          <ArrowRight className="mb-2 h-4 w-4 text-fuchsia-200" />
          对榜单结果有疑问时，进入详情查看主要依据、风险和数据缺口。
        </p>
      </div>
    </section>
  )
}

export default function AnalysisRankingsPage() {
  const { rankings, isLoading, isValidating, isError, error } = useFundAnalysisRankings()
  const sections: RankingSectionConfig[] = [
    {
      title: '结构偏积极',
      shortTitle: '积极池',
      description: '优先展示当前持仓结构、行业主题和近期事件相对更积极的基金。',
      items: rankings?.increase_ideas || [],
      accent: 'rose',
      metricLabel: '积极倾向',
      metricValue: (item) => percentValue(item.analysis?.increase_percent),
      icon: <Flame className="h-4 w-4 text-rose-200" />,
    },
    {
      title: '最值得观察',
      shortTitle: '观察池',
      description: '优先展示当前更适合观察、但结构和事件层仍值得持续跟踪的基金。',
      items: rankings?.watch_ideas || [],
      accent: 'slate',
      metricLabel: '观察倾向',
      metricValue: (item) => percentValue(item.analysis?.hold_percent),
      icon: <BarChart3 className="h-4 w-4 text-cyan-200" />,
    },
    {
      title: '高风险关注',
      shortTitle: '风险池',
      description: '优先展示当前风险暴露较高、或需谨慎对待的基金。',
      items: rankings?.risk_alerts || [],
      accent: 'amber',
      metricLabel: '风险倾向',
      metricValue: (item) => percentValue(item.analysis?.decrease_percent),
      icon: <ShieldAlert className="h-4 w-4 text-amber-200" />,
    },
  ]
  const totalCount = sections.reduce((sum, section) => sum + section.items.length, 0)
  const featuredConfig = sections.find((section) => section.items.length > 0) || sections[0]
  const featuredItem = featuredConfig.items[0]

  return (
    <div className="min-h-[100dvh]">
      <AppTopBar />
      <main id="main-content" className="container mx-auto max-w-7xl px-4 py-4 md:py-8">
        <div className="space-y-5 md:space-y-6">
          <RevealBlock>
            <RankingsHero generatedAt={rankings?.generated_at} isValidating={isValidating} totalCount={totalCount} />
          </RevealBlock>

          {isLoading && !rankings ? (
            <div className="rounded-3xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 px-4 py-16">
              <LoadingSpinner size="lg" text="正在读取量化榜单…" />
            </div>
          ) : isError ? (
            <div className="rounded-3xl border border-amber-500/30 bg-amber-500/10 px-5 py-4 text-sm text-amber-100">
              <div className="flex items-start gap-3">
                <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
                <span>{error instanceof Error ? error.message : '量化排行榜加载失败'}</span>
              </div>
            </div>
          ) : (
            <>
              <RevealBlock delay={80}>
                <RankingsOverview sections={sections} />
              </RevealBlock>

              <RevealBlock delay={120}>
                <FeaturedRankingCard item={featuredItem} config={featuredConfig} />
              </RevealBlock>

              <div id="rankings-list" className="grid gap-5">
                {sections.map((section, index) => (
                  <RevealBlock key={section.title} delay={index * 80}>
                    <RankingsSection config={section} />
                  </RevealBlock>
                ))}
              </div>
            </>
          )}

          <RevealBlock delay={120}>
            <MethodNote />
          </RevealBlock>
        </div>
      </main>
      <SiteFooter compact />
    </div>
  )
}
