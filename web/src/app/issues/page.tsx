'use client'

import Link from 'next/link'
import { useMemo, useState } from 'react'
import { AlertTriangle, Bug, Check, CheckCircle2, ChevronDown, LoaderCircle, Search, Sparkles, WandSparkles } from 'lucide-react'
import { SiteShell } from '@/components/site-shell'
import { useCurrentUser } from '@/hooks/use-auth'
import { createIssue, type IssueStatus, type IssueType, useIssues } from '@/hooks/use-issues'
import { useDebounce } from '@/hooks/use-debounce'
import { cn } from '@/lib/utils'

const issueTypes: { id: '' | IssueType; label: string }[] = [
  { id: '', label: '全部类型' },
  { id: 'bug', label: 'Bug' },
  { id: 'feature', label: '功能诉求' },
  { id: 'improvement', label: '改进建议' },
]

const issueStatuses: { id: '' | IssueStatus; label: string }[] = [
  { id: '', label: '全部状态' },
  { id: 'pending', label: '待接收' },
  { id: 'accepted', label: '处理中' },
  { id: 'completed', label: '已完成' },
]

interface IdeaSelectOption<T extends string> {
  id: T
  label: string
  hint: string
}

function IdeaSelect<T extends string>({
  label,
  value,
  options,
  isOpen,
  onToggle,
  onSelect,
}: {
  label: string
  value: T
  options: IdeaSelectOption<T>[]
  isOpen: boolean
  onToggle: () => void
  onSelect: (value: T) => void
}) {
  const selected = options.find((option) => option.id === value) ?? options[0]

  return (
    <div className="relative">
      <button
        type="button"
        onClick={onToggle}
        className={cn(
          'watchlist-select-shell group relative block w-full overflow-hidden rounded-[22px] border border-[var(--input-border)] bg-[var(--input-bg)]/70 px-4 py-3 text-left transition-all duration-200',
          'hover:border-cyan-400/35 hover:bg-[var(--input-bg)] focus:outline-none focus-visible:border-cyan-400/55 focus-visible:bg-[var(--input-bg)] focus-visible:shadow-[0_14px_30px_rgba(34,211,238,0.12)]',
          isOpen && 'border-cyan-400/55 bg-[var(--input-bg)] shadow-[0_14px_30px_rgba(34,211,238,0.12)]'
        )}
        aria-haspopup="listbox"
        aria-expanded={isOpen}
      >
        <span className="holding-picker-shine" />
        <span className="relative z-10 block text-xs font-medium tracking-[0.18em] text-theme-muted">{label}</span>
        <div className="relative z-10 mt-3 flex items-center justify-between gap-3">
          <div className="min-w-0">
            <div className="truncate text-sm font-medium text-theme-primary">{selected?.label}</div>
            <div className="mt-1 text-xs text-theme-muted">{selected?.hint}</div>
          </div>
          <ChevronDown
            className={cn(
              'h-4 w-4 shrink-0 text-cyan-300 transition-all duration-300',
              isOpen ? 'rotate-180' : 'group-hover:translate-y-0.5'
            )}
          />
        </div>
      </button>

      {isOpen && (
        <>
          <div className="fixed inset-0 z-40" onClick={onToggle} />
          <div className="absolute left-0 right-0 top-full z-50 mt-3 overflow-hidden rounded-[24px] border border-cyan-400/22 bg-[var(--card-bg)]/98 p-2 shadow-[0_24px_60px_rgba(2,8,23,0.42)] backdrop-blur-xl">
            <div className="max-h-72 space-y-1 overflow-y-auto">
              {options.map((option) => {
                const active = option.id === value
                return (
                  <button
                    key={`${label}-${option.id}`}
                    type="button"
                    onClick={() => {
                      onSelect(option.id)
                      onToggle()
                    }}
                    className={cn(
                      'flex w-full items-start justify-between gap-3 rounded-[18px] px-4 py-3 text-left transition-colors',
                      active
                        ? 'bg-cyan-500/14 text-cyan-100'
                        : 'text-theme-secondary hover:bg-[var(--input-bg)] hover:text-theme-primary'
                    )}
                    role="option"
                    aria-selected={active}
                  >
                    <div className="min-w-0">
                      <div className="truncate text-sm font-medium">{option.label}</div>
                      <div className="mt-1 text-xs text-theme-muted">{option.hint}</div>
                    </div>
                    {active && <Check className="mt-0.5 h-4 w-4 shrink-0 text-cyan-300" />}
                  </button>
                )
              })}
            </div>
          </div>
        </>
      )}
    </div>
  )
}

function issueTypeMeta(type: IssueType) {
  switch (type) {
    case 'bug':
      return {
        label: 'Bug',
        className: 'border-rose-500/25 bg-rose-500/10 text-rose-200',
        icon: Bug,
      }
    case 'feature':
      return {
        label: '功能',
        className: 'border-cyan-500/25 bg-cyan-500/10 text-cyan-200',
        icon: Sparkles,
      }
    default:
      return {
        label: '改进',
        className: 'border-amber-500/25 bg-amber-500/10 text-amber-200',
        icon: WandSparkles,
      }
  }
}

function issueStatusMeta(status: IssueStatus) {
  switch (status) {
    case 'accepted':
      return {
        label: '处理中',
        className: 'border-cyan-500/25 bg-cyan-500/10 text-cyan-200',
      }
    case 'completed':
      return {
        label: '已完成',
        className: 'border-emerald-500/25 bg-emerald-500/10 text-emerald-200',
      }
    default:
      return {
        label: '待接收',
        className: 'border-amber-500/25 bg-amber-500/10 text-amber-200',
      }
  }
}

export default function IssuesPage() {
  const { user } = useCurrentUser()
  const [query, setQuery] = useState('')
  const [type, setType] = useState<'' | IssueType>('')
  const [status, setStatus] = useState<'' | IssueStatus>('')
  const [openSelect, setOpenSelect] = useState<'filter-type' | 'filter-status' | 'form-type' | null>(null)
  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')
  const [formType, setFormType] = useState<IssueType>('bug')
  const [feedback, setFeedback] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)

  const debouncedQuery = useDebounce(query, 300)
  const { issues, isLoading, error, refresh } = useIssues({
    query: debouncedQuery,
    type,
    status,
  })

  const counts = useMemo(() => ({
    total: issues.length,
    pending: issues.filter((issue) => issue.status === 'pending').length,
    accepted: issues.filter((issue) => issue.status === 'accepted').length,
    completed: issues.filter((issue) => issue.status === 'completed').length,
  }), [issues])

  const handleSubmit = async () => {
    if (!user || isSubmitting) {
      return
    }

    setFeedback(null)
    setIsSubmitting(true)
    try {
      await createIssue({
        title,
        body,
        type: formType,
      })
      setTitle('')
      setBody('')
      setFormType('bug')
      setFeedback('想法已经送达。管理员会先接收，再更新处理状态。')
      await refresh()
    } catch (requestError) {
      setFeedback(requestError instanceof Error ? requestError.message : '想法发送失败，请稍后重试。')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <SiteShell
      title="我有想法！"
      description="这里是面向所有用户公开的想法区。你可以查看别人已经提出的问题、功能诉求和改进建议；登录后还可以把自己的想法发出来。"
      eyebrowLabel="IDEA BOARD"
      EyebrowIcon={WandSparkles}
    >
      <div className="space-y-8">
        <section className="grid gap-5 lg:grid-cols-4">
          {[
            { label: '当前想法', value: counts.total, accent: 'text-cyan-300' },
            { label: '待接收', value: counts.pending, accent: 'text-amber-300' },
            { label: '处理中', value: counts.accepted, accent: 'text-cyan-300' },
            { label: '已完成', value: counts.completed, accent: 'text-emerald-300' },
          ].map((item) => (
            <article key={item.label} className="rounded-[28px] border border-[var(--card-border)] p-6 glass">
              <div className="text-xs tracking-[0.22em] text-theme-muted">{item.label}</div>
              <div className={cn('mt-3 text-4xl font-black', item.accent)}>{item.value}</div>
            </article>
          ))}
        </section>

        <section className="rounded-[32px] border border-[var(--card-border)] p-6 glass">
          <div className="grid gap-4 lg:grid-cols-[1.4fr_0.7fr_0.7fr]">
            <label className="rounded-[22px] border border-[var(--input-border)] bg-[var(--input-bg)]/70 px-4 py-3">
              <div className="mb-2 text-xs tracking-[0.18em] text-theme-muted">关键字搜索</div>
              <div className="flex items-center gap-3">
                <Search className="h-4 w-4 text-theme-muted" />
                <input
                  value={query}
                  onChange={(event) => setQuery(event.target.value)}
                  placeholder="搜索标题或详情"
                  className="w-full bg-transparent text-sm text-theme-primary outline-none placeholder:text-theme-muted"
                />
              </div>
            </label>

            <IdeaSelect
              label="按类型筛选"
              value={type}
              options={issueTypes.map((option) => ({
                id: option.id,
                label: option.label,
                hint: option.id ? `只看 ${option.label} 相关想法` : '查看所有类型的想法',
              }))}
              isOpen={openSelect === 'filter-type'}
              onToggle={() => setOpenSelect((current) => current === 'filter-type' ? null : 'filter-type')}
              onSelect={(value) => setType(value as '' | IssueType)}
            />

            <IdeaSelect
              label="按状态筛选"
              value={status}
              options={issueStatuses.map((option) => ({
                id: option.id,
                label: option.label,
                hint: option.id ? `只看 ${option.label} 的处理状态` : '查看所有处理状态',
              }))}
              isOpen={openSelect === 'filter-status'}
              onToggle={() => setOpenSelect((current) => current === 'filter-status' ? null : 'filter-status')}
              onSelect={(value) => setStatus(value as '' | IssueStatus)}
            />
          </div>
        </section>

        <section className="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]">
          <div className="space-y-4">
            {isLoading ? (
              <div className="rounded-[32px] border border-[var(--card-border)] p-10 glass text-center">
                <LoaderCircle className="mx-auto h-8 w-8 animate-spin text-cyan-300" />
                <div className="mt-4 text-sm text-theme-secondary">正在加载 Issue 列表...</div>
              </div>
            ) : error ? (
              <div className="rounded-[32px] border border-rose-500/20 bg-rose-500/10 p-6 text-sm text-rose-100">
                {error instanceof Error ? error.message : '加载 Issue 失败'}
              </div>
            ) : issues.length === 0 ? (
              <div className="rounded-[32px] border border-dashed border-[var(--card-border)] p-10 text-center glass">
                <AlertTriangle className="mx-auto h-8 w-8 text-theme-muted" />
                <div className="mt-4 text-xl font-semibold text-theme-primary">暂时没有匹配的 Issue</div>
                <p className="mt-2 text-sm leading-6 text-theme-secondary">
                  你可以调整筛选条件，或者登录后提交一个新的反馈。
                </p>
              </div>
            ) : (
              issues.map((issue) => {
                const typeMeta = issueTypeMeta(issue.type)
                const statusMeta = issueStatusMeta(issue.status)
                const TypeIcon = typeMeta.icon

                return (
                  <Link
                    key={issue.id}
                    href={`/issues/${issue.id}`}
                    className="block rounded-[32px] border border-[var(--card-border)] p-6 glass transition-colors hover:border-cyan-400/30"
                  >
                    <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                      <div className="space-y-3">
                        <div className="flex flex-wrap items-center gap-3">
                          <span className={cn('inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs tracking-[0.18em]', typeMeta.className)}>
                            <TypeIcon className="h-3.5 w-3.5" />
                            {typeMeta.label}
                          </span>
                          <span className={cn('rounded-full border px-3 py-1 text-xs tracking-[0.18em]', statusMeta.className)}>
                            {statusMeta.label}
                          </span>
                        </div>

                        <div>
                          <div className="text-xl font-bold text-theme-primary">{issue.title}</div>
                          <div className="mt-2 line-clamp-3 text-sm leading-7 text-theme-secondary">{issue.body}</div>
                        </div>
                      </div>

                      <div className="text-xs text-theme-muted lg:text-right">
                        <div>提交人：{issue.created_by_display_name}</div>
                        <div className="mt-2">
                          {new Date(issue.created_at).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' })}
                        </div>
                      </div>
                    </div>
                  </Link>
                )
              })
            )}
          </div>

          <div className="space-y-6">
            <section className="rounded-[32px] border border-[var(--card-border)] p-6 glass">
              <div className="text-xs tracking-[0.22em] text-theme-muted">想法投递</div>
              <div className="mt-2 text-2xl font-bold text-theme-primary">发出一个新的想法</div>
              <p className="mt-3 text-sm leading-6 text-theme-secondary">
                登录后可以提交你碰到的问题、想要的功能，或者觉得值得优化的地方。管理员会统一接收并处理。
              </p>

              {!user ? (
                <div className="mt-6 rounded-[24px] border border-amber-500/20 bg-amber-500/10 p-5">
                  <div className="text-sm leading-6 text-amber-50/90">
                    你可以先公开浏览所有 Issue。若要提交新的反馈，请先登录账号。
                  </div>
                  <div className="mt-4 flex flex-wrap gap-3">
                    <Link
                      href="/auth/login"
                      className="inline-flex items-center gap-2 rounded-2xl bg-gradient-to-r from-cyan-500 via-sky-500 to-blue-600 px-4 py-3 text-sm font-medium text-white"
                    >
                      去登录
                    </Link>
                    <Link
                      href="/auth/register"
                      className="inline-flex items-center gap-2 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-sm font-medium text-theme-primary"
                    >
                      注册账号
                    </Link>
                  </div>
                </div>
              ) : (
                <div className="mt-6 space-y-4">
                  <IdeaSelect
                    label="想法类型"
                    value={formType}
                    options={issueTypes.filter((item) => item.id).map((option) => ({
                      id: option.id as IssueType,
                      label: option.label,
                      hint: option.id === 'bug'
                        ? '记录你实际遇到的问题'
                        : option.id === 'feature'
                          ? '提出你希望增加的新功能'
                          : '描述你想优化的地方',
                    }))}
                    isOpen={openSelect === 'form-type'}
                    onToggle={() => setOpenSelect((current) => current === 'form-type' ? null : 'form-type')}
                    onSelect={(value) => setFormType(value as IssueType)}
                  />

                  <label className="block rounded-[22px] border border-[var(--input-border)] bg-[var(--input-bg)]/70 px-4 py-3">
                    <div className="mb-2 text-xs tracking-[0.18em] text-theme-muted">标题</div>
                    <input
                      value={title}
                      onChange={(event) => setTitle(event.target.value)}
                      placeholder="一句话概括你碰到的问题或建议"
                      className="w-full bg-transparent text-sm text-theme-primary outline-none placeholder:text-theme-muted"
                    />
                  </label>

                  <label className="block rounded-[22px] border border-[var(--input-border)] bg-[var(--input-bg)]/70 px-4 py-3">
                    <div className="mb-2 text-xs tracking-[0.18em] text-theme-muted">详情</div>
                    <textarea
                      value={body}
                      onChange={(event) => setBody(event.target.value)}
                      rows={7}
                      placeholder="尽量写清楚现象、期望行为、复现步骤，或者你希望新增的功能。"
                      className="w-full resize-y bg-transparent text-sm leading-6 text-theme-primary outline-none placeholder:text-theme-muted"
                    />
                  </label>

                  {feedback && (
                    <div className={cn(
                      'rounded-[22px] border px-4 py-3 text-sm leading-6',
                      feedback.includes('已提交')
                        ? 'border-emerald-500/20 bg-emerald-500/10 text-emerald-100'
                        : 'border-rose-500/20 bg-rose-500/10 text-rose-100'
                    )}>
                      {feedback}
                    </div>
                  )}

                  <button
                    type="button"
                    onClick={() => void handleSubmit()}
                    disabled={isSubmitting}
                    className={cn(
                      'group relative inline-flex items-center gap-2 overflow-hidden rounded-2xl bg-gradient-to-r from-cyan-500 via-sky-500 to-blue-600 px-5 py-3 text-sm font-medium text-white transition-all duration-200',
                      'hover:-translate-y-0.5 hover:shadow-[0_18px_35px_rgba(14,165,233,0.28)] active:scale-[0.985]',
                      'disabled:cursor-not-allowed disabled:opacity-70',
                      isSubmitting && 'action-button-pop'
                    )}
                  >
                    <span className="action-button-shine" />
                    <CheckCircle2 className="relative z-10 h-4 w-4 transition-transform duration-300 group-hover:-rotate-6 group-hover:scale-110" />
                    <span className="relative z-10">{isSubmitting ? '发送中...' : '想法发送'}</span>
                  </button>
                </div>
              )}
            </section>
          </div>
        </section>
      </div>
    </SiteShell>
  )
}
