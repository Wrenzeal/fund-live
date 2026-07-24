'use client'

import { useMemo } from 'react'
import Link from 'next/link'
import { useParams } from 'next/navigation'
import { ArrowLeft, CheckCircle2, CircleAlert, FlaskConical, RefreshCw, ShieldCheck } from 'lucide-react'
import { CartesianGrid, Legend, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import { AppTopBar } from '@/components/app-top-bar'
import { LoadingSpinner } from '@/components/loading-indicator'
import { SiteFooter } from '@/components/site-footer'
import { useQuantBacktestExperiment, type QuantBacktestExperiment, type QuantNormalizedResult } from '@/hooks/use-fund-data'
import { cn } from '@/lib/utils'

const variantColors = { A: '#94a3b8', B: '#38bdf8', C: '#a78bfa', D: '#34d399' } as const

export default function QuantExperimentPage() {
  const params = useParams<{ experimentId: string }>()
  const experimentId = typeof params?.experimentId === 'string' ? params.experimentId : ''
  const { experiment, error, isLoading, isValidating } = useQuantBacktestExperiment(experimentId)
  const chartData = useMemo(() => buildComparisonSeries(experiment), [experiment])
  const annualRows = useMemo(() => buildAnnualRows(experiment), [experiment])

  return (
    <div className="min-h-[100dvh]">
      <AppTopBar />
      <main id="main-content" className="container mx-auto max-w-7xl px-4 py-4 md:py-8">
        <header className="overflow-hidden rounded-3xl border border-[var(--card-border)] bg-[var(--card-bg)]/45 p-5 md:rounded-[2rem] md:p-7">
          <Link href="/analysis/rankings" className="inline-flex items-center gap-2 text-sm text-theme-secondary transition-colors hover:text-theme-primary">
            <ArrowLeft className="h-4 w-4" />返回量化排行榜
          </Link>
          <div className="mt-5 flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
            <div className="max-w-3xl">
              <div className="flex items-center gap-2 text-xs font-semibold text-cyan-100">
                <FlaskConical className="h-4 w-4" />RISK‑V1 · 研究验证
              </div>
              <h1 className="mt-3 text-2xl font-black tracking-tight text-theme-primary md:text-3xl">A–D 风险收益对照</h1>
              <p className="mt-2 text-sm leading-6 text-theme-secondary">
                在同一信号、区间、费用和成交规则下，依次比较排名缓冲、低换手与风险控制。结果只用于验证策略，不改变FundLive正式评分。
              </p>
            </div>
            <div className="grid grid-cols-2 gap-2 text-xs sm:grid-cols-4 lg:min-w-[32rem]">
              <HeaderStat label="实验状态" value={statusLabel(experiment?.status)} active={isValidating} />
              <HeaderStat label="样本区间" value={experiment ? `${experiment.start_date.slice(0, 7)} 至 ${experiment.end_date.slice(0, 7)}` : '--'} />
              <HeaderStat label="标的池" value={experiment?.universe_version || '--'} />
              <HeaderStat label="信号" value={experiment?.signal_mode === 'historical_proxy' ? '历史代理' : experiment?.signal_mode || '--'} />
            </div>
          </div>
        </header>

        <div className="mt-5 space-y-5">
          {isLoading && !experiment ? (
            <div className="rounded-3xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 py-20"><LoadingSpinner text="正在读取策略实验…" /></div>
          ) : error ? (
            <div className="flex items-start gap-3 rounded-3xl border border-amber-500/25 bg-amber-500/10 p-5 text-sm text-amber-100">
              <CircleAlert className="mt-0.5 h-4 w-4" />实验结果暂时不可用。
            </div>
          ) : experiment ? (
            <>
              <section className="grid gap-3 lg:grid-cols-2 xl:grid-cols-4">
                {experiment.variants.map((variant) => <VariantCard key={variant.key} variant={variant} />)}
              </section>

              <section className="rounded-3xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-5 md:p-6">
                <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
                  <div>
                    <h2 className="text-lg font-black text-theme-primary">组合与沪深300</h2>
                    <p className="mt-1 text-xs leading-5 text-theme-muted">所有曲线以实验开始日归一为100，费用和滑点已计入策略净值。</p>
                  </div>
                  <span className="text-xs text-theme-muted">下一交易日开盘成交</span>
                </div>
                <div className="mt-5 h-[24rem]">
                  {chartData.length > 1 ? (
                    <ResponsiveContainer width="100%" height="100%">
                      <LineChart data={chartData} margin={{ top: 8, right: 10, left: 0, bottom: 0 }}>
                        <CartesianGrid strokeDasharray="3 3" stroke="var(--card-border)" />
                        <XAxis dataKey="time" type="number" domain={['dataMin', 'dataMax']} tickFormatter={shortDate} stroke="var(--text-muted)" minTickGap={42} />
                        <YAxis stroke="var(--text-muted)" width={48} />
                        <Tooltip labelFormatter={(value) => fullDate(Number(value))} formatter={(value, name) => [Number(value).toFixed(2), chartLabel(String(name))]} />
                        <Legend formatter={(value) => chartLabel(String(value))} />
                        <Line type="monotone" dataKey="benchmark" stroke="#f59e0b" strokeWidth={1.7} strokeDasharray="5 4" dot={false} connectNulls />
                        {experiment.variants.map((variant) => (
                          <Line key={variant.key} type="monotone" dataKey={variant.key} stroke={variantColors[variant.key]} strokeWidth={variant.key === 'A' ? 1.7 : 2.2} dot={false} connectNulls />
                        ))}
                      </LineChart>
                    </ResponsiveContainer>
                  ) : <div className="flex h-full items-center justify-center text-sm text-theme-muted">四组任务完成后显示对照曲线</div>}
                </div>
              </section>

              <section className="overflow-hidden rounded-3xl border border-[var(--card-border)] bg-[var(--card-bg)]/35">
                <div className="p-5 md:p-6">
                  <h2 className="text-lg font-black text-theme-primary">核心指标</h2>
                  <p className="mt-1 text-xs text-theme-muted">风险收益改善需同时通过Sharpe、回撤、CAGR和年度稳定性门槛。</p>
                </div>
                <MetricComparison experiment={experiment} />
              </section>

              <section className="overflow-hidden rounded-3xl border border-[var(--card-border)] bg-[var(--card-bg)]/35">
                <div className="p-5 md:p-6">
                  <h2 className="text-lg font-black text-theme-primary">年度表现</h2>
                  <p className="mt-1 text-xs text-theme-muted">按连续组合净值切分自然年；不足120个交易日的片段不参与稳定性门槛。</p>
                </div>
                <AnnualComparison rows={annualRows} />
              </section>

              <section className="rounded-3xl border border-amber-500/20 bg-amber-500/[0.07] p-5 text-xs leading-6 text-theme-secondary md:p-6">
                <div className="flex items-center gap-2 font-semibold text-amber-100"><ShieldCheck className="h-4 w-4" />验证边界</div>
                <p className="mt-2">历史代理信号不等于线上V4评分，公开行情也仍需复核复权、分红、停牌和数据许可。本页不构成投资建议，任何候选策略都不会自动进入正式推荐。</p>
              </section>
            </>
          ) : null}
        </div>
      </main>
      <SiteFooter compact />
    </div>
  )
}

function HeaderStat({ label, value, active = false }: { label: string; value: string; active?: boolean }) {
  return <div className={cn('rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-3', active && 'border-cyan-500/25 bg-cyan-500/10')}><div className="flex items-center gap-1.5 text-[10px] text-theme-muted">{active && <RefreshCw className="h-3 w-3 animate-spin" />}{label}</div><div className="mt-1 break-words font-semibold leading-4 text-theme-primary">{value}</div></div>
}

function VariantCard({ variant }: { variant: QuantBacktestExperiment['variants'][number] }) {
  const summary = variant.job.normalized_result?.summary
  const qualified = variant.risk_gate?.qualified
  return (
    <article className={cn('rounded-3xl border p-4', qualified ? 'border-emerald-500/30 bg-emerald-500/10' : 'border-[var(--card-border)] bg-[var(--card-bg)]/35')}>
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-center gap-2"><span className="flex h-8 w-8 items-center justify-center rounded-xl text-sm font-black text-slate-950" style={{ backgroundColor: variantColors[variant.key] }}>{variant.key}</span><div><div className="font-bold text-theme-primary">{variant.name}</div><div className="text-[10px] text-theme-muted">{jobStatusLabel(variant.job.status)}</div></div></div>
        {qualified && <span className="inline-flex items-center gap-1 rounded-full border border-emerald-500/25 bg-emerald-500/10 px-2 py-1 text-[10px] font-semibold text-emerald-100"><CheckCircle2 className="h-3 w-3" />风险收益改善</span>}
      </div>
      <p className="mt-3 min-h-10 text-xs leading-5 text-theme-secondary">{variant.description}</p>
      <div className="mt-4 grid grid-cols-3 gap-2">
        <MiniMetric label="收益" value={summary ? percent(summary.total_return_pct) : '--'} />
        <MiniMetric label="回撤" value={summary ? drawdownPercent(summary.max_drawdown_pct) : '--'} />
        <MiniMetric label="Sharpe" value={summary ? summary.sharpe.toFixed(2) : '--'} />
      </div>
    </article>
  )
}

function MiniMetric({ label, value }: { label: string; value: string }) {
  return <div className="rounded-xl border border-[var(--card-border)] bg-black/5 p-2"><div className="text-[9px] text-theme-muted">{label}</div><div className="mt-0.5 text-xs font-bold text-theme-primary">{value}</div></div>
}

const metricRows: Array<{ label: string; value: (summary: QuantNormalizedResult['summary']) => string }> = [
  { label: '累计收益', value: (s) => percent(s.total_return_pct) },
  { label: '年化收益', value: (s) => percent(s.cagr_pct) },
  { label: '最大回撤', value: (s) => drawdownPercent(s.max_drawdown_pct) },
  { label: 'Sharpe', value: (s) => s.sharpe.toFixed(3) },
  { label: 'Sortino', value: (s) => s.sortino.toFixed(3) },
  { label: '沪深300超额', value: (s) => percent(s.excess_return_pct) },
  { label: '组合换手率', value: (s) => percent(s.portfolio_turnover_pct) },
  { label: '总费用', value: (s) => `¥${s.total_fees_cny.toLocaleString('zh-CN', { maximumFractionDigits: 2 })}` },
  { label: '订单数', value: (s) => `${s.total_orders}` },
]

function MetricComparison({ experiment }: { experiment: QuantBacktestExperiment }) {
  return <div className="overflow-x-auto"><table className="min-w-[46rem] w-full text-left text-xs"><thead className="border-y border-[var(--card-border)] bg-[var(--card-bg)]/45 text-theme-muted"><tr><th className="px-5 py-3 font-medium">指标</th>{experiment.variants.map((variant) => <th key={variant.key} className="px-4 py-3 font-semibold text-theme-primary">{variant.key} · {variant.name}</th>)}</tr></thead><tbody>{metricRows.map((row) => <tr key={row.label} className="border-b border-[var(--card-border)]/70 last:border-0"><td className="px-5 py-3 text-theme-secondary">{row.label}</td>{experiment.variants.map((variant) => <td key={variant.key} className="px-4 py-3 font-semibold text-theme-primary">{variant.job.normalized_result ? row.value(variant.job.normalized_result.summary) : '--'}</td>)}</tr>)}</tbody></table></div>
}

type AnnualRow = { year: number; tradingDays: number; values: Record<string, number> }

function AnnualComparison({ rows }: { rows: AnnualRow[] }) {
  if (!rows.length) return <div className="px-5 pb-6 text-sm text-theme-muted">任务完成后显示年度结果</div>
  return <div className="overflow-x-auto"><table className="min-w-[42rem] w-full text-left text-xs"><thead className="border-y border-[var(--card-border)] bg-[var(--card-bg)]/45 text-theme-muted"><tr><th className="px-5 py-3">年度</th><th className="px-4 py-3">交易日</th>{['A', 'B', 'C', 'D'].map((key) => <th key={key} className="px-4 py-3">策略 {key}</th>)}<th className="px-4 py-3">沪深300</th></tr></thead><tbody>{rows.map((row) => <tr key={row.year} className="border-b border-[var(--card-border)]/70 last:border-0"><td className="px-5 py-3 font-semibold text-theme-primary">{row.year}</td><td className="px-4 py-3 text-theme-muted">{row.tradingDays}</td>{['A', 'B', 'C', 'D'].map((key) => <td key={key} className={cn('px-4 py-3 font-semibold', (row.values[key] || 0) >= 0 ? 'text-emerald-200' : 'text-rose-200')}>{Number.isFinite(row.values[key]) ? percent(row.values[key]) : '--'}</td>)}<td className="px-4 py-3 text-theme-secondary">{Number.isFinite(row.values.benchmark) ? percent(row.values.benchmark) : '--'}</td></tr>)}</tbody></table></div>
}

function buildComparisonSeries(experiment?: QuantBacktestExperiment) {
  const byTime = new Map<number, Record<string, number>>()
  const add = (key: string, points?: Array<{ time: number; value: number }>) => points?.forEach((point) => byTime.set(point.time, { ...(byTime.get(point.time) || {}), time: point.time, [key]: point.value }))
  experiment?.variants.forEach((variant) => add(variant.key, variant.job.normalized_result?.series.strategy))
  add('benchmark', experiment?.variants[0]?.job.normalized_result?.series.csi300)
  return Array.from(byTime.values()).sort((left, right) => left.time - right.time)
}

function buildAnnualRows(experiment?: QuantBacktestExperiment): AnnualRow[] {
  const rows = new Map<number, AnnualRow>()
  experiment?.variants.forEach((variant) => variant.job.normalized_result?.annual_returns.forEach((annual) => {
    const row = rows.get(annual.year) || { year: annual.year, tradingDays: annual.trading_days, values: {} }
    row.tradingDays = Math.max(row.tradingDays, annual.trading_days)
    row.values[variant.key] = annual.strategy_return_pct
    if (variant.key === 'A') row.values.benchmark = annual.csi300_return_pct
    rows.set(annual.year, row)
  }))
  return Array.from(rows.values()).sort((left, right) => left.year - right.year)
}

function statusLabel(status?: string) { return ({ pending: '等待运行', running: '运行中', completed: '已完成', failed: '有任务失败', empty: '暂无任务' } as Record<string, string>)[status || ''] || '读取中' }
function jobStatusLabel(status: string) { return ({ queued: '排队中', running: '运行中', completed: '已完成', failed: '运行失败', queue_failed: '队列不可用' } as Record<string, string>)[status] || status }
function percent(value: number) { return `${value >= 0 ? '+' : ''}${value.toFixed(2)}%` }
function drawdownPercent(value: number) { return `-${Math.abs(value).toFixed(2)}%` }
function shortDate(value: number) { return new Date(value * 1000).toLocaleDateString('zh-CN', { year: '2-digit', month: '2-digit' }) }
function fullDate(value: number) { return new Date(value * 1000).toLocaleDateString('zh-CN') }
function chartLabel(value: string) { return ({ A: 'A 周度基线', B: 'B 排名缓冲', C: 'C 低换手', D: 'D 风险控制', benchmark: '沪深300' } as Record<string, string>)[value] || value }
