'use client'

import Link from 'next/link'
import { useParams } from 'next/navigation'
import { ArrowLeft, BarChart3, CircleAlert, RefreshCw } from 'lucide-react'
import { CartesianGrid, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import { AppTopBar } from '@/components/app-top-bar'
import { LoadingSpinner } from '@/components/loading-indicator'
import { SiteFooter } from '@/components/site-footer'
import { useQuantBacktest } from '@/hooks/use-fund-data'

type EquityPoint = { time: number; value: number }

export default function QuantBacktestPage() {
  const params = useParams<{ jobId: string }>()
  const jobId = typeof params?.jobId === 'string' ? params.jobId : ''
  const { job, error, isLoading, isValidating } = useQuantBacktest(jobId)
  const equity = extractEquityPoints(job?.equity_curve)
  const metrics = flattenMetrics(job?.metrics)

  return (
    <div className="min-h-[100dvh]">
      <AppTopBar />
      <main id="main-content" className="container mx-auto max-w-7xl px-4 py-4 md:py-8">
        <header className="rounded-3xl border border-[var(--card-border)] bg-[var(--card-bg)]/45 p-5 md:p-6">
          <Link href="/analysis/rankings" className="inline-flex items-center gap-2 text-sm text-theme-secondary hover:text-theme-primary">
            <ArrowLeft className="h-4 w-4" />返回量化验证
          </Link>
          <div className="mt-5 flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
            <div>
              <div className="flex items-center gap-2 text-2xl font-black text-theme-primary">
                <BarChart3 className="h-6 w-6 text-cyan-200" />Lean 回测结果
              </div>
              <p className="mt-2 text-sm text-theme-secondary">任务 {jobId.slice(0, 12)} · 周度 Top 5 等权组合</p>
            </div>
            <div className="flex items-center gap-2 rounded-full border border-[var(--card-border)] bg-[var(--card-bg)]/35 px-3 py-2 text-xs text-theme-secondary">
              {isValidating && <RefreshCw className="h-3.5 w-3.5 animate-spin" />}
              {job ? statusLabel(job.status) : '读取中'}
            </div>
          </div>
        </header>

        <div className="mt-5 space-y-5">
          {isLoading && !job ? (
            <div className="rounded-3xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 py-20"><LoadingSpinner text="正在读取回测结果…" /></div>
          ) : error ? (
            <div className="flex items-start gap-3 rounded-3xl border border-amber-500/25 bg-amber-500/10 p-5 text-sm text-amber-100">
              <CircleAlert className="mt-0.5 h-4 w-4" />回测结果暂时不可用。
            </div>
          ) : (
            <>
              <section className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
                <Metric label="引擎" value={job?.engine_version ? `LEAN ${job.engine_version.slice(0, 8)}` : 'LEAN'} />
                <Metric label="信号模式" value={job?.signal_mode || '--'} />
                <Metric label="试点池" value={job?.universe_version || '--'} />
                <Metric label="完成时间" value={job?.completed_at ? new Date(job.completed_at).toLocaleString('zh-CN') : '--'} />
              </section>

              <section className="rounded-3xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-5 md:p-6">
                <h2 className="text-lg font-black text-theme-primary">组合与基准曲线</h2>
                <p className="mt-1 text-xs text-theme-muted">成交费用与滑点已计入组合结果；基准包含沪深300、试点池等权和现金。</p>
                <div className="mt-5 h-80">
                  {equity.length > 1 ? (
                    <ResponsiveContainer width="100%" height="100%">
                      <LineChart data={equity}>
                        <CartesianGrid strokeDasharray="3 3" stroke="var(--card-border)" />
                        <XAxis dataKey="time" type="number" domain={['dataMin', 'dataMax']} tickFormatter={(value) => new Date(value * 1000).toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' })} stroke="var(--text-muted)" />
                        <YAxis stroke="var(--text-muted)" width={56} />
                        <Tooltip labelFormatter={(value) => new Date(Number(value) * 1000).toLocaleString('zh-CN')} />
                        <Line type="monotone" dataKey="value" stroke="#67e8f9" strokeWidth={2} dot={false} />
                      </LineChart>
                    </ResponsiveContainer>
                  ) : <div className="flex h-full items-center justify-center text-sm text-theme-muted">任务完成后显示净值曲线</div>}
                </div>
              </section>

              <section className="rounded-3xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-5 md:p-6">
                <h2 className="text-lg font-black text-theme-primary">统计摘要</h2>
                <div className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
                  {metrics.length ? metrics.slice(0, 16).map(([label, value]) => <Metric key={label} label={label} value={value} />) : <div className="text-sm text-theme-muted">暂无统计结果</div>}
                </div>
                {job?.error_message && <div className="mt-4 rounded-2xl border border-rose-500/25 bg-rose-500/10 p-4 text-sm text-rose-100">{job.error_message}</div>}
              </section>
            </>
          )}
        </div>
      </main>
      <SiteFooter compact />
    </div>
  )
}

function Metric({ label, value }: { label: string; value: string }) {
  return <div className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 p-4"><div className="text-[11px] text-theme-muted">{label}</div><div className="mt-1 break-words text-sm font-bold text-theme-primary">{value}</div></div>
}

function statusLabel(status: string) {
  return ({ queued: '排队中', running: '运行中', completed: '已完成', failed: '运行失败', queue_failed: '队列不可用' } as Record<string, string>)[status] || status
}

function flattenMetrics(metrics?: Record<string, unknown>) {
  if (!metrics) return [] as [string, string][]
  return Object.entries(metrics).filter(([, value]) => ['string', 'number', 'boolean'].includes(typeof value)).map(([key, value]) => [key, String(value)] as [string, string])
}

function extractEquityPoints(charts?: Record<string, unknown>): EquityPoint[] {
  if (!charts) return []
  const candidates: EquityPoint[][] = []
  const visit = (value: unknown) => {
    if (Array.isArray(value)) {
      const points = value.map((item) => {
        if (!item || typeof item !== 'object') return null
        const record = item as Record<string, unknown>
        const time = Number(record.x ?? record.time)
        const pointValue = Number(record.y ?? record.value)
        return Number.isFinite(time) && Number.isFinite(pointValue) ? { time, value: pointValue } : null
      }).filter((item): item is EquityPoint => item !== null)
      if (points.length > 1) candidates.push(points)
      value.forEach(visit)
      return
    }
    if (value && typeof value === 'object') Object.values(value as Record<string, unknown>).forEach(visit)
  }
  visit(charts)
  return candidates.sort((left, right) => right.length - left.length)[0] || []
}
