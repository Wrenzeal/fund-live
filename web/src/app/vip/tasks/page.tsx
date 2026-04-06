'use client'

import Link from 'next/link'
import { useMemo } from 'react'
import { AlertTriangle, Clock3, FileStack, Layers3, LoaderCircle, Wallet } from 'lucide-react'
import { AccountAreaShell } from '@/components/account-area-shell'
import { useVIPPreview } from '@/hooks/use-vip-preview'
import { cn } from '@/lib/utils'

function statusMeta(status: 'queued' | 'running' | 'completed' | 'failed') {
  switch (status) {
    case 'queued':
      return {
        label: '排队中',
        className: 'border-amber-500/25 bg-amber-500/10 text-amber-200',
      }
    case 'running':
      return {
        label: '生成中',
        className: 'border-cyan-500/25 bg-cyan-500/10 text-cyan-200',
      }
    case 'completed':
      return {
        label: '已完成',
        className: 'border-emerald-500/25 bg-emerald-500/10 text-emerald-200',
      }
    default:
      return {
        label: '失败',
        className: 'border-rose-500/25 bg-rose-500/10 text-rose-200',
      }
  }
}

export default function VIPTasksPage() {
  const { membership, tasks, remainingQuota } = useVIPPreview()

  const focusTask = useMemo(
    () => tasks[0] ?? null,
    [tasks]
  )

  return (
    <AccountAreaShell
      title="分析任务中心"
      description="查看 VIP 板块分析和组合分析的任务状态。当前任务、额度与报告入口来自后端，任务进度仍按模板化异步流程推进。"
    >
      <div className="space-y-8">
        {!membership.isVip && (
          <section className="rounded-[32px] border border-amber-500/25 bg-amber-500/10 p-6">
            <div className="flex items-start gap-3">
              <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0 text-amber-200" />
              <div>
                <div className="text-lg font-bold text-amber-50">当前账号尚未开通 VIP</div>
                <p className="mt-2 text-sm leading-6 text-amber-50/85">
                  先前往会员介绍页和开通页体验 VIP 页面与流程，开通后即可在这里看到分析任务。
                </p>
                <Link
                  href="/vip"
                  className="mt-4 inline-flex items-center gap-2 rounded-2xl border border-amber-400/30 bg-black/10 px-4 py-3 text-sm font-medium text-amber-50"
                >
                  去查看 VIP 介绍
                </Link>
              </div>
            </div>
          </section>
        )}

        <section className="grid gap-5 lg:grid-cols-3">
          <article className="rounded-[28px] border border-[var(--card-border)] p-6 glass">
            <div className="text-xs tracking-[0.22em] text-theme-muted">板块分析额度</div>
            <div className="mt-3 text-4xl font-black text-cyan-50">{remainingQuota.sectorAnalysis}</div>
            <div className="mt-2 text-sm text-theme-secondary">今日剩余可用次数</div>
          </article>
          <article className="rounded-[28px] border border-[var(--card-border)] p-6 glass">
            <div className="text-xs tracking-[0.22em] text-theme-muted">组合分析额度</div>
            <div className="mt-3 text-4xl font-black text-cyan-50">{remainingQuota.portfolioAnalysis}</div>
            <div className="mt-2 text-sm text-theme-secondary">今日剩余可用次数</div>
          </article>
          <article className="rounded-[28px] border border-[var(--card-border)] p-6 glass">
            <div className="text-xs tracking-[0.22em] text-theme-muted">最近状态</div>
            <div className="mt-3 text-2xl font-black text-theme-primary">
              {tasks[0] ? statusMeta(tasks[0].status).label : '暂无任务'}
            </div>
            <div className="mt-2 text-sm text-theme-secondary">
              {tasks[0]?.progressText || '从自选页或持仓页发起一次分析任务'}
            </div>
          </article>
        </section>

        {focusTask && (
          <section className="rounded-[32px] border border-cyan-500/25 bg-cyan-500/10 p-6">
            <div className="text-xs tracking-[0.22em] text-cyan-300">当前聚焦任务</div>
            <div className="mt-2 text-2xl font-bold text-cyan-50">{focusTask.targetName}</div>
            <div className="mt-3 text-sm leading-6 text-cyan-50/90">{focusTask.progressText}</div>
          </section>
        )}

        {tasks.length === 0 ? (
          <section className="rounded-[32px] border border-dashed border-[var(--card-border)] p-10 text-center glass">
            <Clock3 className="mx-auto h-10 w-10 text-theme-muted" />
            <div className="mt-4 text-xl font-semibold text-theme-primary">还没有分析任务</div>
            <p className="mt-2 text-sm leading-6 text-theme-secondary">
              你可以回到自选页发起板块分析，或者到持仓页发起组合分析。
            </p>
            <div className="mt-6 flex flex-wrap justify-center gap-3">
              <Link
                href="/watchlist"
                className="inline-flex items-center gap-2 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-sm font-medium text-theme-primary"
              >
                <Layers3 className="h-4 w-4" />
                去自选页
              </Link>
              <Link
                href="/holdings"
                className="inline-flex items-center gap-2 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-sm font-medium text-theme-primary"
              >
                <Wallet className="h-4 w-4" />
                去持仓页
              </Link>
            </div>
          </section>
        ) : (
          <section className="space-y-4">
            {tasks.map((task) => {
              const meta = statusMeta(task.status)
              return (
                <article
                  key={task.id}
                  className="rounded-[32px] border border-[var(--card-border)] p-6 glass"
                >
                  <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                    <div className="space-y-3">
                      <div className="flex flex-wrap items-center gap-3">
                        <span className={cn('rounded-full border px-3 py-1 text-xs tracking-[0.2em]', meta.className)}>
                          {meta.label}
                        </span>
                        <span className="text-xs tracking-[0.18em] text-theme-muted">
                          {task.type === 'sector_analysis' ? '板块分析' : '组合分析'}
                        </span>
                      </div>

                      <div>
                        <div className="text-xl font-bold text-theme-primary">{task.targetName}</div>
                        <div className="mt-2 text-sm leading-6 text-theme-secondary">{task.progressText}</div>
                      </div>

                      <div className="flex flex-wrap gap-4 text-xs text-theme-muted">
                        <span>创建时间：{new Date(task.createdAt).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' })}</span>
                        {task.completedAt && (
                          <span>完成时间：{new Date(task.completedAt).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' })}</span>
                        )}
                      </div>
                    </div>

                    <div className="flex flex-wrap gap-3">
                      {task.status === 'completed' && task.reportId ? (
                        <Link
                          href={`/vip/reports/${task.reportId}`}
                          className="inline-flex items-center gap-2 rounded-2xl bg-gradient-to-r from-cyan-500 via-sky-500 to-blue-600 px-4 py-3 text-sm font-medium text-white"
                        >
                          <FileStack className="h-4 w-4" />
                          查看报告
                        </Link>
                      ) : (
                        <div className="inline-flex items-center gap-2 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-sm text-theme-secondary">
                          <LoaderCircle className={cn('h-4 w-4', task.status !== 'failed' && 'animate-spin')} />
                          等待生成完成
                        </div>
                      )}
                    </div>
                  </div>
                </article>
              )
            })}
          </section>
        )}
      </div>
    </AccountAreaShell>
  )
}
