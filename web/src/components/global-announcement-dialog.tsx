'use client'

import Link from 'next/link'
import { useMemo, useState } from 'react'
import { Bell, CheckCircle2, X } from 'lucide-react'
import { useCurrentUser } from '@/hooks/use-auth'
import { markAnnouncementRead, useUnreadAnnouncements } from '@/hooks/use-announcements'

export function GlobalAnnouncementDialog() {
  const { user } = useCurrentUser()
  const { unreadAnnouncements, refresh } = useUnreadAnnouncements(Boolean(user))
  const [dismissedIDs, setDismissedIDs] = useState<string[]>([])
  const [isMarkingRead, setIsMarkingRead] = useState(false)

  const currentAnnouncement = useMemo(() => {
    if (!user) {
      return null
    }
    return unreadAnnouncements.find((item) => !dismissedIDs.includes(item.id)) ?? null
  }, [dismissedIDs, unreadAnnouncements, user])

  if (!user || !currentAnnouncement) {
    return null
  }

  const handleMarkRead = async () => {
    setIsMarkingRead(true)
    try {
      await markAnnouncementRead(currentAnnouncement.id)
      setDismissedIDs((current) => current.includes(currentAnnouncement.id) ? current : [...current, currentAnnouncement.id])
      await refresh()
    } finally {
      setIsMarkingRead(false)
    }
  }

  return (
    <div className="fixed inset-0 z-[70] flex items-center justify-center bg-black/50 px-4 py-8 backdrop-blur-sm">
      <div className="w-full max-w-2xl rounded-[32px] border border-[var(--card-border)] glass-strong p-6 shadow-2xl">
        <div className="flex items-start justify-between gap-4">
          <div className="flex items-start gap-4">
            <div className="rounded-2xl bg-cyan-500/12 p-3 text-cyan-300">
              <Bell className="h-5 w-5" />
            </div>
            <div>
              <div className="text-xs tracking-[0.22em] text-theme-muted">未读公告</div>
              <h2 className="mt-2 text-2xl font-black text-theme-primary">{currentAnnouncement.title}</h2>
              <div className="mt-2 text-sm text-theme-muted">
                {new Date(currentAnnouncement.published_at).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' })}
              </div>
            </div>
          </div>

          <button
            type="button"
            onClick={() => setDismissedIDs((current) => current.includes(currentAnnouncement.id) ? current : [...current, currentAnnouncement.id])}
            className="rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] p-2 text-theme-secondary transition-colors hover:text-theme-primary"
            aria-label="关闭公告"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="mt-6 rounded-[24px] border border-[var(--card-border)] bg-[var(--input-bg)]/60 p-5">
          <div className="text-sm leading-7 text-theme-secondary whitespace-pre-wrap">
            {currentAnnouncement.summary || currentAnnouncement.content}
          </div>
        </div>

        <div className="mt-6 flex flex-wrap justify-end gap-3">
          <Link
            href={`/announcements/${currentAnnouncement.id}`}
            className="inline-flex items-center gap-2 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-sm font-medium text-theme-primary"
          >
            查看详情
          </Link>
          <button
            type="button"
            onClick={() => void handleMarkRead()}
            disabled={isMarkingRead}
            className="inline-flex items-center gap-2 rounded-2xl bg-gradient-to-r from-cyan-500 via-sky-500 to-blue-600 px-4 py-3 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-70"
          >
            <CheckCircle2 className="h-4 w-4" />
            {isMarkingRead ? '处理中...' : '标记已读'}
          </button>
        </div>
      </div>
    </div>
  )
}
