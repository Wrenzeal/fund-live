'use client'

import { useMemo, useState } from 'react'
import Link from 'next/link'
import { Check, ChevronDown, FolderPlus, Layers3, LoaderCircle, Plus, Sparkles, Trash2 } from 'lucide-react'
import { AccountAreaShell } from '@/components/account-area-shell'
import { VIPAnalysisEntry } from '@/components/vip-analysis-entry'
import { WatchlistFundCard } from '@/components/watchlist-fund-card'
import { useCurrentUser } from '@/hooks/use-auth'
import { useFundSearch } from '@/hooks/use-fund-data'
import { useUserPortfolio } from '@/hooks/use-user-portfolio'
import { cn } from '@/lib/utils'

export default function WatchlistPage() {
  const { user, isLoading } = useCurrentUser()
  const {
    watchlistGroups,
    totalWatchlistFunds,
    seedDemoData,
    createGroup,
    deleteGroup,
    addFundToGroup,
    removeFundFromGroup,
  } = useUserPortfolio(user?.id ?? null)

  const [groupName, setGroupName] = useState('')
  const [groupDescription, setGroupDescription] = useState('')
  const [selectedGroupID, setSelectedGroupID] = useState<string>('')
  const [isGroupMenuOpen, setIsGroupMenuOpen] = useState(false)
  const [fundQuery, setFundQuery] = useState('')
  const [isCreatingGroup, setIsCreatingGroup] = useState(false)
  const [deletingGroupID, setDeletingGroupID] = useState<string | null>(null)
  const { results } = useFundSearch(fundQuery)

  const selectedGroup = useMemo(
    () => watchlistGroups.find((group) => group.id === selectedGroupID) ?? null,
    [selectedGroupID, watchlistGroups]
  )
  const selectedGroupLabel = selectedGroup?.name || '选择一个分组'

  const handleCreateGroup = async () => {
    if (isCreatingGroup) {
      return
    }

    setIsCreatingGroup(true)

    try {
      await createGroup(groupName, groupDescription)
      setGroupName('')
      setGroupDescription('')
    } finally {
      setIsCreatingGroup(false)
    }
  }

  const handleDeleteGroup = async (groupID: string) => {
    if (deletingGroupID) {
      return
    }

    setDeletingGroupID(groupID)

    try {
      await new Promise((resolve) => window.setTimeout(resolve, 180))
      await deleteGroup(groupID)
    } finally {
      setDeletingGroupID(null)
    }
  }

  if (isLoading) {
    return (
      <AccountAreaShell title="你的自选" description="按分组管理你的重点观察基金，当前版本已改为服务端持久化。">
        <div className="glass h-64 rounded-[32px] border border-[var(--card-border)]" />
      </AccountAreaShell>
    )
  }

  if (!user) {
    return (
      <AccountAreaShell title="你的自选" description="按分组管理你的重点观察基金，当前版本已改为服务端持久化。">
        <div className="glass rounded-[32px] border border-[var(--card-border)] p-8 text-center">
          <div className="mb-3 text-2xl font-bold text-theme-primary">登录后可查看你的自选</div>
          <p className="mx-auto max-w-xl text-sm leading-6 text-theme-secondary">
            你的自选分组和基金清单现在已经绑定到账户，登录后可直接读取服务端数据。
          </p>
          <div className="mt-6 flex justify-center gap-3">
            <Link href="/auth/login" className="rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-sm text-theme-secondary">
              去登录
            </Link>
            <Link href="/auth/register" className="rounded-2xl bg-gradient-to-r from-cyan-500 via-sky-500 to-blue-600 px-4 py-3 text-sm font-medium text-white">
              去注册
            </Link>
          </div>
        </div>
      </AccountAreaShell>
    )
  }

  return (
    <AccountAreaShell title="你的自选" description="按分组整理你的观察基金，并快速看到每只基金的实时预估涨跌幅与迷你走势。数据已改为服务端存储。">
      <div className="space-y-8">
        <div className="grid gap-6 xl:grid-cols-[1.4fr_0.9fr]">
          <section className="rounded-[32px] border border-[var(--card-border)] p-6 glass">
            <div className="mb-6 flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
              <div>
                <div className="text-sm text-theme-muted">分组总览</div>
                <div className="mt-1 text-3xl font-black text-theme-primary">
                  {watchlistGroups.length} 个分组 / {totalWatchlistFunds} 只基金
                </div>
              </div>

              {watchlistGroups.length === 0 && (
                <button
                  type="button"
                  onClick={() => void seedDemoData()}
                  className="inline-flex items-center gap-2 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-sm text-theme-secondary"
                >
                  <Sparkles className="h-4 w-4" />
                  载入演示自选
                </button>
              )}
            </div>

            <div className="grid gap-4 lg:grid-cols-2">
              <label className="space-y-2">
                <span className="text-sm text-theme-secondary">新增分组</span>
                <input
                  value={groupName}
                  onChange={(event) => setGroupName(event.target.value)}
                  placeholder="例如：核心观察"
                  className="auth-input w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-theme-primary outline-none placeholder:text-theme-muted"
                />
              </label>

              <label className="space-y-2">
                <span className="text-sm text-theme-secondary">分组说明</span>
                <input
                  value={groupDescription}
                  onChange={(event) => setGroupDescription(event.target.value)}
                  placeholder="例如：长期定投、行业轮动等"
                  className="auth-input w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-theme-primary outline-none placeholder:text-theme-muted"
                />
              </label>
            </div>

            <button
              type="button"
              onClick={() => void handleCreateGroup()}
              disabled={isCreatingGroup}
              className={cn(
                'group relative mt-4 inline-flex items-center gap-2 overflow-hidden rounded-2xl bg-gradient-to-r from-cyan-500 via-sky-500 to-blue-600 px-4 py-3 text-sm font-medium text-white transition-all duration-200',
                'hover:-translate-y-0.5 hover:shadow-[0_18px_35px_rgba(14,165,233,0.28)] active:scale-[0.985]',
                'disabled:cursor-not-allowed disabled:opacity-85',
                isCreatingGroup && 'action-button-pop'
              )}
            >
              <span className="action-button-shine" />
              {isCreatingGroup ? (
                <LoaderCircle className="relative z-10 h-4 w-4 animate-spin" />
              ) : (
                <FolderPlus className="relative z-10 h-4 w-4 transition-transform duration-300 group-hover:-rotate-6 group-hover:scale-110" />
              )}
              <span className="relative z-10">{isCreatingGroup ? '创建中...' : '创建分组'}</span>
            </button>
          </section>

          <section className="rounded-[32px] border border-[var(--card-border)] p-6 glass">
            <div className="mb-5">
              <div className="text-sm text-theme-muted">把基金放进分组</div>
              <div className="mt-1 text-2xl font-bold text-theme-primary">服务端分组管理</div>
            </div>

            <div className="space-y-4">
              <div className="relative">
                <button
                  type="button"
                  onClick={() => {
                    if (watchlistGroups.length === 0) return
                    setIsGroupMenuOpen((open) => !open)
                  }}
                  disabled={watchlistGroups.length === 0}
                  className={cn(
                    'watchlist-select-shell group relative block w-full overflow-hidden rounded-[24px] border border-[var(--input-border)] bg-[var(--input-bg)]/90 px-4 py-3 text-left transition-all duration-200',
                    'hover:border-cyan-400/35 hover:bg-[var(--input-bg)] focus:outline-none focus-visible:border-cyan-400/55 focus-visible:bg-[var(--input-bg)] focus-visible:shadow-[0_14px_30px_rgba(34,211,238,0.12)]',
                    'disabled:cursor-not-allowed disabled:opacity-70',
                    isGroupMenuOpen && 'border-cyan-400/55 bg-[var(--input-bg)] shadow-[0_14px_30px_rgba(34,211,238,0.12)]'
                  )}
                  aria-haspopup="listbox"
                  aria-expanded={isGroupMenuOpen}
                >
                  <span className="holding-picker-shine" />
                  <span className="relative z-10 block text-xs font-medium tracking-[0.18em] text-theme-muted">目标分组</span>
                  <div className="relative z-10 mt-3 flex items-center justify-between gap-3">
                    <div className="min-w-0">
                      <div className={cn('truncate text-sm font-medium', selectedGroup ? 'text-theme-primary' : 'text-theme-secondary')}>
                        {watchlistGroups.length === 0 ? '暂无可用分组' : selectedGroupLabel}
                      </div>
                      <div className="mt-1 text-xs text-theme-muted">
                        {watchlistGroups.length === 0
                          ? '先在左侧创建分组，再把基金加入进去'
                          : selectedGroup
                            ? `当前分组共 ${selectedGroup.funds.length} 只基金`
                            : '先选择分组，再把搜索结果加入进去'}
                      </div>
                    </div>
                    <ChevronDown
                      className={cn(
                        'h-4 w-4 shrink-0 text-cyan-300 transition-all duration-300',
                        isGroupMenuOpen ? 'rotate-180' : 'group-hover:translate-y-0.5'
                      )}
                    />
                  </div>
                </button>

                {isGroupMenuOpen && (
                  <>
                    <div className="fixed inset-0 z-40" onClick={() => setIsGroupMenuOpen(false)} />
                    <div className="absolute left-0 right-0 top-full z-50 mt-3 overflow-hidden rounded-[24px] border border-cyan-400/22 bg-[var(--card-bg)]/98 p-2 shadow-[0_24px_60px_rgba(2,8,23,0.42)] backdrop-blur-xl">
                      <div className="max-h-72 space-y-1 overflow-y-auto">
                        <button
                          type="button"
                          onClick={() => {
                            setSelectedGroupID('')
                            setIsGroupMenuOpen(false)
                          }}
                          className={cn(
                            'flex w-full items-start justify-between gap-3 rounded-[18px] px-4 py-3 text-left transition-colors',
                            !selectedGroup
                              ? 'bg-cyan-500/14 text-cyan-100'
                              : 'text-theme-secondary hover:bg-[var(--input-bg)] hover:text-theme-primary'
                          )}
                          role="option"
                          aria-selected={!selectedGroup}
                        >
                          <div>
                            <div className="text-sm font-medium">暂不选择分组</div>
                            <div className="mt-1 text-xs text-theme-muted">保留当前搜索结果，不立即加入任何分组</div>
                          </div>
                          {!selectedGroup && <Check className="mt-0.5 h-4 w-4 shrink-0 text-cyan-300" />}
                        </button>

                        {watchlistGroups.map((group) => {
                          const active = group.id === selectedGroupID
                          return (
                            <button
                              key={group.id}
                              type="button"
                              onClick={() => {
                                setSelectedGroupID(group.id)
                                setIsGroupMenuOpen(false)
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
                                <div className="truncate text-sm font-medium">{group.name}</div>
                                <div className="mt-1 text-xs text-theme-muted">
                                  {group.description || '未填写分组说明'} · {group.funds.length} 只基金
                                </div>
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

              <input
                value={fundQuery}
                onChange={(event) => setFundQuery(event.target.value)}
                placeholder="搜索基金代码或名称"
                className="auth-input w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-theme-primary outline-none placeholder:text-theme-muted"
              />

              <div className="space-y-2">
                {results.slice(0, 5).map((fund) => (
                  <button
                    key={fund.id}
                    type="button"
                    disabled={!selectedGroup}
                    onClick={() => {
                      if (!selectedGroup) return
                      void addFundToGroup(selectedGroup.id, fund.id)
                      setFundQuery('')
                    }}
                    className="flex w-full items-center justify-between rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-left transition-colors hover:border-cyan-500/40 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    <div>
                      <div className="text-sm font-medium text-theme-primary">{fund.name}</div>
                      <div className="mt-1 text-xs text-theme-muted">{fund.id}</div>
                    </div>
                    <Plus className="h-4 w-4 text-cyan-300" />
                  </button>
                ))}
              </div>
            </div>
          </section>
        </div>

        <div className="space-y-6">
          {watchlistGroups.length === 0 ? (
            <div className="rounded-[32px] border border-dashed border-[var(--card-border)] p-10 text-center glass">
              <Layers3 className="mx-auto h-10 w-10 text-theme-muted" />
              <div className="mt-4 text-xl font-semibold text-theme-primary">还没有自选分组</div>
              <p className="mt-2 text-sm leading-6 text-theme-secondary">
                你可以先创建分组，再把基金加入对应的观察篮子。当前版本数据会保存到服务端。
              </p>
            </div>
          ) : (
            watchlistGroups.map((group) => (
              <section key={group.id} className="rounded-[32px] border border-[var(--card-border)] p-6 glass">
                <div className={`mb-6 rounded-[28px] bg-gradient-to-r ${watchlistAccentToClass(group.accent)} p-5`}>
                  <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                    <div>
                      <div className="text-2xl font-black text-theme-primary">{group.name}</div>
                      <p className="mt-2 max-w-2xl text-sm leading-6 text-theme-secondary">
                        {group.description || '未填写分组说明'}
                      </p>
                      <div className="mt-3 text-xs text-theme-muted">共 {group.funds.length} 只基金</div>
                    </div>

                    <button
                      type="button"
                      onClick={() => void handleDeleteGroup(group.id)}
                      disabled={deletingGroupID !== null}
                      className={cn(
                        'group relative inline-flex items-center gap-2 overflow-hidden rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-2 text-sm text-theme-secondary transition-all duration-200',
                        'hover:-translate-y-0.5 hover:border-rose-400/40 hover:bg-rose-500/12 hover:text-rose-200 active:scale-[0.985]',
                        'disabled:cursor-not-allowed disabled:opacity-80',
                        deletingGroupID === group.id && 'danger-button-pop border-rose-400/45 bg-rose-500/14 text-rose-100'
                      )}
                    >
                      <span className="action-button-shine" />
                      {deletingGroupID === group.id ? (
                        <LoaderCircle className="relative z-10 h-4 w-4 animate-spin" />
                      ) : (
                        <Trash2 className="relative z-10 h-4 w-4 transition-transform duration-300 group-hover:-rotate-12 group-hover:scale-110" />
                      )}
                      <span className="relative z-10">{deletingGroupID === group.id ? '删除中...' : '删除分组'}</span>
                    </button>
                  </div>
                </div>

                {group.funds.length === 0 ? (
                  <div className="rounded-2xl border border-dashed border-[var(--card-border)] px-5 py-10 text-center text-sm text-theme-secondary">
                    当前分组还没有基金，从上面的搜索结果里把基金加入这里。
                  </div>
                ) : (
                  <div className="grid gap-5 md:grid-cols-2 xl:grid-cols-3">
                    {group.funds.map((item) => (
                      <WatchlistFundCard
                        key={`${group.id}:${item.fund_id}`}
                        fundId={item.fund_id}
                        onRemove={() => void removeFundFromGroup(group.id, item.fund_id)}
                      />
                    ))}
                  </div>
                )}
              </section>
            ))
          )}
        </div>

        <VIPAnalysisEntry
          title="AI 深度研判入口"
          description="这里预留给后续的 Deep Research 场景，重点分析当日大盘走势、全球经济变化、宏观风险与接下来可能的演化路径。当前版本只保留 VIP 入口，不接实际分析能力。"
        />
      </div>
    </AccountAreaShell>
  )
}

function watchlistAccentToClass(accent: string) {
  switch (accent) {
    case 'emerald':
      return 'from-emerald-500/30 via-teal-500/15 to-transparent'
    case 'amber':
      return 'from-amber-500/25 via-orange-500/15 to-transparent'
    case 'fuchsia':
      return 'from-fuchsia-500/25 via-violet-500/15 to-transparent'
    default:
      return 'from-cyan-500/30 via-sky-500/20 to-transparent'
  }
}
