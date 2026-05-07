'use client'

import Link from 'next/link'
import { AlertTriangle, ArrowLeft, ArrowRight, Eye, RefreshCw, ShieldAlert, Sparkles } from 'lucide-react'
import { FundAnalysisBadge } from '@/components/fund-analysis-badge'
import { FundAnalysisEventHint } from '@/components/fund-analysis-event-hint'
import { LoadingSpinner } from '@/components/loading-indicator'
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

function RankingsSection({
  title,
  description,
  items,
  accent,
  metricLabel,
  metricValue,
}: {
  title: string
  description: string
  items: FundAnalysisRankingItem[]
  accent: 'rose' | 'slate' | 'amber'
  metricLabel: string
  metricValue: (item: FundAnalysisRankingItem) => string
}) {
  const tone = accent === 'rose'
    ? 'border-rose-500/25 bg-rose-500/10'
    : accent === 'amber'
      ? 'border-amber-500/25 bg-amber-500/10'
      : 'border-cyan-500/25 bg-cyan-500/10'

  return (
    <section className="glass rounded-3xl p-6">
      <div className="mb-4">
        <h2 className="text-lg font-semibold text-theme-primary">{title}</h2>
        <p className="mt-1 text-sm text-theme-secondary">{description}</p>
      </div>

      <div className="space-y-3">
        {items.length === 0 ? (
          <div className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 px-4 py-8 text-center text-sm text-theme-muted">
            暂无可展示结果
          </div>
        ) : items.map((item, index) => (
          <Link
            key={item.fund?.id || `${title}-${index}`}
            href={item.fund?.id ? `/analysis/${item.fund.id}` : '/analysis'}
            className={cn(
              'block rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-4 transition-all duration-200',
              'hover:-translate-y-0.5 hover:border-cyan-400/35 hover:shadow-[0_12px_24px_rgba(34,211,238,0.08)]'
            )}
          >
            <div className="flex items-start justify-between gap-4">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-3">
                  <span className="text-sm font-semibold text-theme-muted">#{index + 1}</span>
                  <div className="min-w-0">
                    <div className="truncate text-base font-semibold text-theme-primary">
                      {item.fund?.name || item.fund?.id || '未知基金'}
                    </div>
                    <div className="mt-1 text-xs text-theme-muted">
                      {item.fund?.id || '--'} · {item.analysis?.analysis_basis || '分析口径待补'}
                    </div>
                  </div>
                </div>

                <div className="mt-3">
                  <FundAnalysisBadge analysis={item.analysis} showScore />
                </div>
                <div className="mt-2">
                  <FundAnalysisEventHint analysis={item.analysis} />
                </div>

                {item.analysis?.summary && (
                  <p className="mt-3 text-sm leading-6 text-theme-secondary">
                    {item.analysis.summary}
                  </p>
                )}
              </div>

              <div className={cn('shrink-0 rounded-2xl border px-4 py-3 text-right', tone)}>
                <div className="text-[11px] tracking-[0.18em] text-theme-muted">{metricLabel}</div>
                <div className="mt-1 text-2xl font-bold text-theme-primary">{metricValue(item)}</div>
                <div className="mt-2 text-xs text-theme-secondary">总分 {scoreValue(item.analysis?.total_score)}</div>
              </div>
            </div>

            <div className="mt-4 flex items-center justify-end gap-2 text-xs text-theme-muted">
              <span>查看完整量化看板</span>
              <ArrowRight className="h-3.5 w-3.5" />
            </div>
          </Link>
        ))}
      </div>
    </section>
  )
}

export default function AnalysisRankingsPage() {
  const { rankings, isLoading, isValidating, isError, error } = useFundAnalysisRankings()

  return (
    <main className="min-h-screen">
      <div className="container mx-auto px-4 py-6 md:py-8">
        <header className="mb-6 rounded-3xl border border-[var(--card-border)] bg-[var(--card-bg)]/40 p-5">
          <Link
            href="/"
            className="mb-3 inline-flex items-center gap-2 text-sm text-theme-secondary transition-colors hover:text-theme-primary"
          >
            <ArrowLeft className="h-4 w-4" />
            返回首页
          </Link>

          <div className="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
            <div>
              <div className="text-2xl font-bold text-theme-primary">量化排行榜</div>
              <p className="mt-2 text-sm text-theme-secondary">
                统一复用独立 analysis 结果，先提供“最值得加仓 / 最值得观察 / 高风险减仓关注”三类榜单。
              </p>
            </div>

            <div className="flex flex-wrap items-center gap-2 text-xs text-theme-muted">
              <span className="rounded-full border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-1">
                {rankings?.generated_at ? `生成时间：${new Date(rankings.generated_at).toLocaleString('zh-CN')}` : '等待生成'}
              </span>
              {isValidating && (
                <span className="inline-flex items-center gap-2 rounded-full border border-cyan-500/20 bg-cyan-500/10 px-3 py-1 text-cyan-100">
                  <RefreshCw className="h-3.5 w-3.5 animate-spin" />
                  刷新中
                </span>
              )}
            </div>
          </div>
        </header>

        {isLoading && !rankings ? (
          <div className="rounded-3xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 px-4 py-16">
            <LoadingSpinner size="lg" text="量化排行榜加载中..." />
          </div>
        ) : isError ? (
          <div className="rounded-3xl border border-amber-500/30 bg-amber-500/10 px-5 py-4 text-sm text-amber-100">
            <div className="flex items-start gap-3">
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
              <span>{error instanceof Error ? error.message : '量化排行榜加载失败'}</span>
            </div>
          </div>
        ) : (
          <div className="grid gap-6 xl:grid-cols-3">
            <RankingsSection
              title="最值得加仓"
              description="优先展示当前建议更偏加仓、且总分相对靠前的基金。"
              items={rankings?.increase_ideas || []}
              accent="rose"
              metricLabel="加仓倾向"
              metricValue={(item) => percentValue(item.analysis?.increase_percent)}
            />
            <RankingsSection
              title="最值得观察"
              description="优先展示当前偏持有、但结构和事件层仍值得持续跟踪的基金。"
              items={rankings?.watch_ideas || []}
              accent="slate"
              metricLabel="观察倾向"
              metricValue={(item) => percentValue(item.analysis?.hold_percent)}
            />
            <RankingsSection
              title="高风险减仓关注"
              description="优先展示当前风险较高、且建议更偏减仓或需谨慎对待的基金。"
              items={rankings?.risk_alerts || []}
              accent="amber"
              metricLabel="减仓倾向"
              metricValue={(item) => percentValue(item.analysis?.decrease_percent)}
            />
          </div>
        )}

        <section className="mt-6 rounded-3xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-5">
          <div className="mb-3 flex items-center gap-2 text-theme-primary">
            <Eye className="h-4 w-4" />
            <span className="text-sm font-semibold">当前口径说明</span>
          </div>
          <div className="grid gap-4 lg:grid-cols-3">
            <p className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-4 text-sm text-theme-secondary">
              <Sparkles className="mb-2 h-4 w-4 text-cyan-200" />
              榜单当前直接复用独立 <code>/api/v1/fund/:id/analysis</code> 结果，不再为排行榜单独派生另一套建议逻辑。
            </p>
            <p className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-4 text-sm text-theme-secondary">
              <ShieldAlert className="mb-2 h-4 w-4 text-amber-200" />
              当前仍属于增强阶段第一版，适合作为快速筛选入口，不应替代基金详情页的完整事件与结构阅读。
            </p>
            <p className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-4 text-sm text-theme-secondary">
              <ArrowRight className="mb-2 h-4 w-4 text-fuchsia-200" />
              下一步会继续把同一套分析结果扩展到持仓页排序、自选页标签和更细的结构变化对比。
            </p>
          </div>
        </section>
      </div>
    </main>
  )
}
