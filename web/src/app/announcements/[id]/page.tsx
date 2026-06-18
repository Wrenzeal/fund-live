'use client'

import { useParams } from 'next/navigation'
import { useState } from 'react'
import { ArrowLeft, Bell, CheckCircle2, LoaderCircle } from 'lucide-react'
import { ScrollRevealStack } from '@/components/scroll-reveal'
import { SiteShell } from '@/components/site-shell'
import { ActionButton } from '@/components/ui/action-button'
import { StatusBanner } from '@/components/ui/status-banner'
import { Surface } from '@/components/ui/surface'
import { useCurrentUser } from '@/hooks/use-auth'
import { markAnnouncementRead, useAnnouncement } from '@/hooks/use-announcements'

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
        <Surface radius="xl" padding="lg" className="text-center">
          <LoaderCircle className="mx-auto h-8 w-8 animate-spin text-cyan-300" />
          <div className="mt-4 text-sm text-theme-secondary">正在加载公告详情...</div>
        </Surface>
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
        <StatusBanner tone="danger">
          {error instanceof Error ? error.message : '公告不存在。'}
        </StatusBanner>
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
      <ScrollRevealStack className="space-y-6">
        <div className="flex flex-wrap gap-3">
          <ActionButton
            href="/announcements"
            variant="secondary"
          >
            <ArrowLeft className="h-4 w-4" />
            返回公告列表
          </ActionButton>
          {user && (
            <ActionButton
              type="button"
              variant="primary"
              onClick={() => void handleMarkRead()}
              disabled={isMarkingRead}
            >
              <CheckCircle2 className="h-4 w-4" />
              {isMarkingRead ? '处理中...' : '标记已读'}
            </ActionButton>
          )}
        </div>

        {feedback && (
          <StatusBanner tone={feedback.includes('已标记') ? 'success' : 'danger'}>
            {feedback}
          </StatusBanner>
        )}

        <Surface as="section" radius="xl" padding="lg">
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
        </Surface>

        <Surface as="section" radius="xl" padding="md">
          <div className="whitespace-pre-wrap text-sm leading-8 text-theme-secondary">
            {announcement.content}
          </div>
        </Surface>
      </ScrollRevealStack>
    </SiteShell>
  )
}
