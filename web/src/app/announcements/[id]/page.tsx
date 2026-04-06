'use client'

import Link from 'next/link'
import { useParams } from 'next/navigation'
import { useState } from 'react'
import { ArrowLeft, Bell, CheckCircle2, LoaderCircle } from 'lucide-react'
import { SiteShell } from '@/components/site-shell'
import { useCurrentUser } from '@/hooks/use-auth'
import { markAnnouncementRead, useAnnouncement } from '@/hooks/use-announcements'
import { cn } from '@/lib/utils'

export default function AnnouncementDetailPage() {
  const params = useParams<{ id: string }>()
  const announcementID = typeof params?.id === 'string' ? params.id : ''
  const { announcement, error, isLoading } = useAnnouncement(announcementID)
  const { user } = useCurrentUser()
  const [feedback, setFeedback] = useState<string | null>(null)
  const [isMarkingRead, setIsMarkingRead] = useState(false)

  const handleMarkRead = async () => {
    if (!user || !announcement || isMarkingRead) {
      return
    }

    setFeedback(null)
    setIsMarkingRead(true)
    try {
      await markAnnouncementRead(announcement.id)
      setFeedback('已标记为已读。')
    } catch (requestError) {
      setFeedback(requestError instanceof Error ? requestError.message : '标记已读失败。')
    } finally {
      setIsMarkingRead(false)
    }
  }

  if (isLoading) {
    return (
      <SiteShell
        title="公告详情"
        description="查看站点已经发布的公告和历史更新记录。"
        eyebrowLabel="UPDATE DETAIL"
        EyebrowIcon={Bell}
      >
        <div className="rounded-[32px] border border-[var(--card-border)] p-10 glass text-center">
          <LoaderCircle className="mx-auto h-8 w-8 animate-spin text-cyan-300" />
          <div className="mt-4 text-sm text-theme-secondary">正在加载公告详情...</div>
        </div>
      </SiteShell>
    )
  }

  if (!announcement) {
    return (
      <SiteShell
        title="公告详情"
        description="查看站点已经发布的公告和历史更新记录。"
        eyebrowLabel="UPDATE DETAIL"
        EyebrowIcon={Bell}
      >
        <div className="rounded-[32px] border border-rose-500/20 bg-rose-500/10 p-6 text-sm text-rose-100">
          {error instanceof Error ? error.message : '公告不存在。'}
        </div>
      </SiteShell>
    )
  }

  return (
    <SiteShell
      title="公告详情"
      description="查看站点已经发布的公告和历史更新记录。"
      eyebrowLabel="UPDATE DETAIL"
      EyebrowIcon={Bell}
    >
      <div className="space-y-6">
        <div className="flex flex-wrap gap-3">
          <Link
            href="/announcements"
            className="inline-flex items-center gap-2 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-sm font-medium text-theme-primary"
          >
            <ArrowLeft className="h-4 w-4" />
            返回公告列表
          </Link>
          {user && (
            <button
              type="button"
              onClick={() => void handleMarkRead()}
              disabled={isMarkingRead}
              className="inline-flex items-center gap-2 rounded-2xl bg-gradient-to-r from-cyan-500 via-sky-500 to-blue-600 px-4 py-3 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-70"
            >
              <CheckCircle2 className="h-4 w-4" />
              {isMarkingRead ? '处理中...' : '标记已读'}
            </button>
          )}
        </div>

        {feedback && (
          <div className={cn(
            'rounded-[24px] border px-4 py-3 text-sm',
            feedback.includes('已标记')
              ? 'border-emerald-500/20 bg-emerald-500/10 text-emerald-100'
              : 'border-rose-500/20 bg-rose-500/10 text-rose-100'
          )}>
            {feedback}
          </div>
        )}

        <section className="rounded-[36px] border border-[var(--card-border)] p-8 glass">
          <div className="flex flex-col gap-5 lg:flex-row lg:items-start lg:justify-between">
            <div>
              <div className="inline-flex items-center gap-2 rounded-full border border-cyan-500/25 bg-cyan-500/10 px-3 py-1 text-xs tracking-[0.18em] text-cyan-200">
                <Bell className="h-3.5 w-3.5" />
                {announcement.source_type === 'changelog' ? 'CHANGELOG 导入' : '手动公告'}
              </div>
              <h2 className="mt-4 text-4xl font-black leading-tight text-theme-primary">{announcement.title}</h2>
              <p className="mt-4 max-w-3xl text-sm leading-7 text-theme-secondary">{announcement.summary}</p>
            </div>

            <div className="rounded-[28px] border border-[var(--card-border)] bg-[var(--input-bg)]/70 p-5 text-sm text-theme-secondary">
              <div>发布时间：{new Date(announcement.published_at).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' })}</div>
              {announcement.source_ref && <div className="mt-2">来源标识：{announcement.source_ref}</div>}
            </div>
          </div>
        </section>

        <section className="rounded-[32px] border border-[var(--card-border)] p-6 glass">
          <div className="whitespace-pre-wrap text-sm leading-8 text-theme-secondary">
            {announcement.content}
          </div>
        </section>
      </div>
    </SiteShell>
  )
}
