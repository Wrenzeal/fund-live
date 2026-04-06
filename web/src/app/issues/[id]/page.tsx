'use client'

import Link from 'next/link'
import { useParams } from 'next/navigation'
import { useState } from 'react'
import { AlertTriangle, ArrowLeft, LoaderCircle, WandSparkles } from 'lucide-react'
import { SiteShell } from '@/components/site-shell'
import { useCurrentUser } from '@/hooks/use-auth'
import { type IssueStatus, useIssue, updateIssueStatus } from '@/hooks/use-issues'
import { cn } from '@/lib/utils'

function issueTypeMeta(type: 'bug' | 'feature' | 'improvement') {
  switch (type) {
    case 'bug':
      return { label: 'Bug', className: 'border-rose-500/25 bg-rose-500/10 text-rose-200' }
    case 'feature':
      return { label: '功能诉求', className: 'border-cyan-500/25 bg-cyan-500/10 text-cyan-200' }
    default:
      return { label: '改进建议', className: 'border-amber-500/25 bg-amber-500/10 text-amber-200' }
  }
}

function issueStatusMeta(status: IssueStatus) {
  switch (status) {
    case 'accepted':
      return { label: '处理中', className: 'border-cyan-500/25 bg-cyan-500/10 text-cyan-200' }
    case 'completed':
      return { label: '已完成', className: 'border-emerald-500/25 bg-emerald-500/10 text-emerald-200' }
    default:
      return { label: '待接收', className: 'border-amber-500/25 bg-amber-500/10 text-amber-200' }
  }
}

const statuses: IssueStatus[] = ['pending', 'accepted', 'completed']

export default function IssueDetailPage() {
  const params = useParams<{ id: string }>()
  const issueID = typeof params?.id === 'string' ? params.id : ''
  const { issue, error, isLoading, refresh } = useIssue(issueID)
  const { user } = useCurrentUser()
  const [isUpdating, setIsUpdating] = useState(false)
  const [feedback, setFeedback] = useState<string | null>(null)

  const handleStatusUpdate = async (status: IssueStatus) => {
    if (!issue || !user?.is_admin || isUpdating) {
      return
    }

    setFeedback(null)
    setIsUpdating(true)
    try {
      await updateIssueStatus(issue.id, status)
      await refresh()
      setFeedback('Issue 状态已更新。')
    } catch (requestError) {
      setFeedback(requestError instanceof Error ? requestError.message : '更新状态失败。')
    } finally {
      setIsUpdating(false)
    }
  }

  if (isLoading) {
    return (
      <SiteShell
        title="想法详情"
        description="查看这条公开想法的完整内容和当前处理状态。"
        eyebrowLabel="IDEA DETAIL"
        EyebrowIcon={WandSparkles}
      >
        <div className="rounded-[32px] border border-[var(--card-border)] p-10 glass text-center">
          <LoaderCircle className="mx-auto h-8 w-8 animate-spin text-cyan-300" />
          <div className="mt-4 text-sm text-theme-secondary">正在加载想法详情...</div>
        </div>
      </SiteShell>
    )
  }

  if (!issue) {
    return (
      <SiteShell
        title="想法详情"
        description="查看这条公开想法的完整内容和当前处理状态。"
        eyebrowLabel="IDEA DETAIL"
        EyebrowIcon={WandSparkles}
      >
        <div className="rounded-[32px] border border-rose-500/20 bg-rose-500/10 p-6 text-sm text-rose-100">
          {error instanceof Error ? error.message : '这条想法不存在。'}
        </div>
      </SiteShell>
    )
  }

  const typeMeta = issueTypeMeta(issue.type)
  const statusMeta = issueStatusMeta(issue.status)

  return (
    <SiteShell
      title="想法详情"
      description="查看这条公开想法的完整内容和当前处理状态。"
      eyebrowLabel="IDEA DETAIL"
      EyebrowIcon={WandSparkles}
    >
      <div className="space-y-6">
        <div>
          <Link
            href="/issues"
            className="inline-flex items-center gap-2 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-sm font-medium text-theme-primary"
          >
            <ArrowLeft className="h-4 w-4" />
            返回我有想法！
          </Link>
        </div>

        <section className="rounded-[36px] border border-[var(--card-border)] p-8 glass">
          <div className="flex flex-col gap-5 lg:flex-row lg:items-start lg:justify-between">
            <div className="space-y-4">
              <div className="flex flex-wrap items-center gap-3">
                <span className={cn('rounded-full border px-3 py-1 text-xs tracking-[0.18em]', typeMeta.className)}>
                  {typeMeta.label}
                </span>
                <span className={cn('rounded-full border px-3 py-1 text-xs tracking-[0.18em]', statusMeta.className)}>
                  {statusMeta.label}
                </span>
              </div>

              <div>
                <h2 className="text-4xl font-black leading-tight text-theme-primary">{issue.title}</h2>
                <div className="mt-3 text-sm text-theme-muted">
                  提交人：{issue.created_by_display_name} · {new Date(issue.created_at).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' })}
                </div>
              </div>
            </div>

            {user?.is_admin && (
              <div className="w-full rounded-[28px] border border-[var(--card-border)] bg-[var(--input-bg)]/70 p-5 lg:max-w-sm">
                <div className="text-xs tracking-[0.18em] text-theme-muted">管理员操作</div>
                <div className="mt-4 flex flex-wrap gap-3">
                  {statuses.map((status) => (
                    <button
                      key={status}
                      type="button"
                      onClick={() => void handleStatusUpdate(status)}
                      disabled={isUpdating || issue.status === status}
                      className={cn(
                        'rounded-2xl border px-4 py-3 text-sm font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-60',
                        issue.status === status
                          ? 'border-cyan-500/30 bg-cyan-500/15 text-cyan-200'
                          : 'border-[var(--input-border)] bg-[var(--input-bg)] text-theme-primary hover:border-cyan-400/30'
                      )}
                    >
                      {issueStatusMeta(status).label}
                    </button>
                  ))}
                </div>

                {feedback && (
                  <div className={cn(
                    'mt-4 rounded-[20px] border px-4 py-3 text-sm',
                    feedback.includes('已更新')
                      ? 'border-emerald-500/20 bg-emerald-500/10 text-emerald-100'
                      : 'border-rose-500/20 bg-rose-500/10 text-rose-100'
                  )}>
                    {feedback}
                  </div>
                )}
              </div>
            )}
          </div>
        </section>

        <section className="rounded-[32px] border border-[var(--card-border)] p-6 glass">
          <div className="mb-4 text-xl font-bold text-theme-primary">详情描述</div>
          <div className="whitespace-pre-wrap text-sm leading-8 text-theme-secondary">
            {issue.body}
          </div>
        </section>

        {!user?.is_admin && (
          <section className="rounded-[32px] border border-amber-500/20 bg-amber-500/10 p-6">
            <div className="flex items-start gap-3 text-amber-100">
              <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0" />
              <div className="text-sm leading-6">
                当前页面对所有用户公开展示，只有管理员账号可以修改处理状态。
              </div>
            </div>
          </section>
        )}
      </div>
    </SiteShell>
  )
}
