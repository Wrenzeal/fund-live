'use client'

import Link from 'next/link'
import { useState } from 'react'
import { Bell, FileUp, LoaderCircle, Megaphone } from 'lucide-react'
import { SiteShell } from '@/components/site-shell'
import { useCurrentUser } from '@/hooks/use-auth'
import {
  createAnnouncement,
  importAnnouncementsFromChangelog,
  useAnnouncements,
} from '@/hooks/use-announcements'
import { cn } from '@/lib/utils'

export default function AnnouncementsPage() {
  const { user } = useCurrentUser()
  const { announcements, isLoading, error, refresh } = useAnnouncements()
  const [title, setTitle] = useState('')
  const [summary, setSummary] = useState('')
  const [content, setContent] = useState('')
  const [feedback, setFeedback] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [isImporting, setIsImporting] = useState(false)

  const handleCreateAnnouncement = async () => {
    if (!user?.is_admin || isSubmitting) {
      return
    }

    setFeedback(null)
    setIsSubmitting(true)
    try {
      await createAnnouncement({ title, summary, content })
      setTitle('')
      setSummary('')
      setContent('')
      setFeedback('公告已发布。')
      await refresh()
    } catch (requestError) {
      setFeedback(requestError instanceof Error ? requestError.message : '发布公告失败。')
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleImportChangelog = async () => {
    if (!user?.is_admin || isImporting) {
      return
    }

    setFeedback(null)
    setIsImporting(true)
    try {
      const result = await importAnnouncementsFromChangelog()
      setFeedback(`CHANGELOG 导入完成，本次处理 ${result?.imported ?? 0} 条公告。`)
      await refresh()
    } catch (requestError) {
      setFeedback(requestError instanceof Error ? requestError.message : '导入 CHANGELOG 失败。')
    } finally {
      setIsImporting(false)
    }
  }

  return (
    <SiteShell
      title="更新公告"
      description="这里记录站点已经发布的功能更新和运营公告。登录用户如有未读内容，会在进入站点后收到弹窗提醒。"
      eyebrowLabel="UPDATE BOARD"
      EyebrowIcon={Bell}
    >
      <div className="space-y-8">
        {user?.is_admin && (
          <section className="grid gap-6 xl:grid-cols-[1.05fr_0.95fr]">
            <div className="rounded-[32px] border border-[var(--card-border)] p-6 glass">
              <div className="text-xs tracking-[0.22em] text-theme-muted">管理员发布</div>
              <div className="mt-2 text-2xl font-bold text-theme-primary">手动新增公告</div>
              <div className="mt-6 space-y-4">
                <label className="block rounded-[22px] border border-[var(--input-border)] bg-[var(--input-bg)]/70 px-4 py-3">
                  <div className="mb-2 text-xs tracking-[0.18em] text-theme-muted">标题</div>
                  <input
                    value={title}
                    onChange={(event) => setTitle(event.target.value)}
                    className="w-full bg-transparent text-sm text-theme-primary outline-none placeholder:text-theme-muted"
                    placeholder="例如：2026.4.6 新增用户反馈系统"
                  />
                </label>

                <label className="block rounded-[22px] border border-[var(--input-border)] bg-[var(--input-bg)]/70 px-4 py-3">
                  <div className="mb-2 text-xs tracking-[0.18em] text-theme-muted">摘要</div>
                  <input
                    value={summary}
                    onChange={(event) => setSummary(event.target.value)}
                    className="w-full bg-transparent text-sm text-theme-primary outline-none placeholder:text-theme-muted"
                    placeholder="可选；不填会自动根据正文提炼"
                  />
                </label>

                <label className="block rounded-[22px] border border-[var(--input-border)] bg-[var(--input-bg)]/70 px-4 py-3">
                  <div className="mb-2 text-xs tracking-[0.18em] text-theme-muted">正文</div>
                  <textarea
                    value={content}
                    onChange={(event) => setContent(event.target.value)}
                    rows={8}
                    className="w-full resize-y bg-transparent text-sm leading-6 text-theme-primary outline-none placeholder:text-theme-muted"
                    placeholder="这里写完整公告内容。"
                  />
                </label>

                {feedback && (
                  <div className={cn(
                    'rounded-[20px] border px-4 py-3 text-sm',
                    feedback.includes('完成') || feedback.includes('已发布')
                      ? 'border-emerald-500/20 bg-emerald-500/10 text-emerald-100'
                      : 'border-rose-500/20 bg-rose-500/10 text-rose-100'
                  )}>
                    {feedback}
                  </div>
                )}

                <div className="flex flex-wrap gap-3">
                  <button
                    type="button"
                    onClick={() => void handleCreateAnnouncement()}
                    disabled={isSubmitting}
                    className="inline-flex items-center gap-2 rounded-2xl bg-gradient-to-r from-cyan-500 via-sky-500 to-blue-600 px-5 py-3 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-70"
                  >
                    <Megaphone className="h-4 w-4" />
                    {isSubmitting ? '发布中...' : '发布公告'}
                  </button>
                  <button
                    type="button"
                    onClick={() => void handleImportChangelog()}
                    disabled={isImporting}
                    className="inline-flex items-center gap-2 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-5 py-3 text-sm font-medium text-theme-primary disabled:cursor-not-allowed disabled:opacity-70"
                  >
                    <FileUp className="h-4 w-4" />
                    {isImporting ? '导入中...' : '从 CHANGELOG 导入'}
                  </button>
                </div>
              </div>
            </div>

            <div className="rounded-[32px] border border-[var(--card-border)] p-6 glass">
              <div className="text-xs tracking-[0.22em] text-theme-muted">管理提示</div>
              <div className="mt-2 text-2xl font-bold text-theme-primary">公告策略</div>
              <div className="mt-5 space-y-4 text-sm leading-7 text-theme-secondary">
                <div className="rounded-[22px] border border-[var(--card-border)] bg-[var(--input-bg)]/60 p-4">
                  手动公告适合发布维护通知、临时说明和运营信息。
                </div>
                <div className="rounded-[22px] border border-[var(--card-border)] bg-[var(--input-bg)]/60 p-4">
                  `CHANGELOG` 导入适合把已经整理好的版本更新同步为站内公告。
                </div>
                <div className="rounded-[22px] border border-[var(--card-border)] bg-[var(--input-bg)]/60 p-4">
                  登录用户会在存在未读公告时收到弹窗提醒，并可手动标记已读。
                </div>
              </div>
            </div>
          </section>
        )}

        <section className="space-y-4">
          {isLoading ? (
            <div className="rounded-[32px] border border-[var(--card-border)] p-10 glass text-center">
              <LoaderCircle className="mx-auto h-8 w-8 animate-spin text-cyan-300" />
              <div className="mt-4 text-sm text-theme-secondary">正在加载公告列表...</div>
            </div>
          ) : error ? (
            <div className="rounded-[32px] border border-rose-500/20 bg-rose-500/10 p-6 text-sm text-rose-100">
              {error instanceof Error ? error.message : '加载公告失败'}
            </div>
          ) : announcements.length === 0 ? (
            <div className="rounded-[32px] border border-dashed border-[var(--card-border)] p-10 text-center glass">
              <Bell className="mx-auto h-8 w-8 text-theme-muted" />
              <div className="mt-4 text-xl font-semibold text-theme-primary">暂时还没有公告</div>
            </div>
          ) : (
            announcements.map((announcement) => (
              <Link
                key={announcement.id}
                href={`/announcements/${announcement.id}`}
                className="block rounded-[32px] border border-[var(--card-border)] p-6 glass transition-colors hover:border-cyan-400/30"
              >
                <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                  <div className="space-y-3">
                    <div className="flex flex-wrap items-center gap-3">
                      <span className="rounded-full border border-cyan-500/25 bg-cyan-500/10 px-3 py-1 text-xs tracking-[0.18em] text-cyan-200">
                        {announcement.source_type === 'changelog' ? 'CHANGELOG 导入' : '手动公告'}
                      </span>
                    </div>

                    <div>
                      <div className="text-xl font-bold text-theme-primary">{announcement.title}</div>
                      <div className="mt-2 text-sm leading-7 text-theme-secondary">{announcement.summary}</div>
                    </div>
                  </div>

                  <div className="text-xs text-theme-muted lg:text-right">
                    <div>
                      {new Date(announcement.published_at).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' })}
                    </div>
                    {announcement.source_ref && (
                      <div className="mt-2">来源标识：{announcement.source_ref}</div>
                    )}
                  </div>
                </div>
              </Link>
            ))
          )}
        </section>
      </div>
    </SiteShell>
  )
}
