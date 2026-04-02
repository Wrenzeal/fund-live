'use client'

import Link from 'next/link'
import { useState } from 'react'
import { ChevronDown, Layers3, Loader2, LogOut, ShieldCheck, Wallet } from 'lucide-react'
import { cn } from '@/lib/utils'
import { logout, useCurrentUser } from '@/hooks/use-auth'

function getInitials(name: string) {
  const normalized = name.trim()
  if (!normalized) {
    return 'U'
  }

  const parts = normalized.split(/\s+/).filter(Boolean)
  if (parts.length === 1) {
    return normalized.slice(0, 1).toUpperCase()
  }

  return `${parts[0][0] ?? ''}${parts[parts.length - 1][0] ?? ''}`.toUpperCase()
}

export function UserAccountMenu() {
  const { user, isLoading, mutate } = useCurrentUser()
  const [isOpen, setIsOpen] = useState(false)
  const [isLoggingOut, setIsLoggingOut] = useState(false)

  if (isLoading) {
    return (
      <div className="flex">
        <div className="glass h-11 w-28 animate-pulse rounded-2xl border border-[var(--card-border)] md:w-36" />
      </div>
    )
  }

  if (!user) {
    return (
      <div className="flex items-center gap-2">
        <Link
          href="/auth/login"
          className={cn(
            'group relative overflow-hidden rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2 text-sm text-theme-secondary transition-all duration-200',
            'hover:-translate-y-0.5 hover:border-cyan-400/35 hover:bg-cyan-400/10 hover:text-theme-primary hover:shadow-[0_12px_24px_rgba(34,211,238,0.10)] active:scale-[0.97]'
          )}
        >
          <span className="action-button-shine" />
          <span className="relative z-10">登录</span>
        </Link>
        <Link
          href="/auth/register"
          className={cn(
            'group relative overflow-hidden rounded-xl bg-gradient-to-r from-cyan-500 via-sky-500 to-blue-600 px-3 py-2 text-sm font-medium text-white transition-all duration-200',
            'hover:-translate-y-0.5 hover:shadow-[0_16px_30px_rgba(14,165,233,0.28)] active:scale-[0.97]'
          )}
        >
          <span className="action-button-shine" />
          <span className="relative z-10">注册</span>
        </Link>
      </div>
    )
  }

  const displayName = user.display_name?.trim() || user.email
  const initials = getInitials(displayName)

  const handleLogout = async () => {
    setIsLoggingOut(true)
    try {
      await logout()
      await mutate(null, { revalidate: false })
      setIsOpen(false)
    } finally {
      setIsLoggingOut(false)
    }
  }

  return (
    <div className="relative">
      <button
        type="button"
        onClick={() => setIsOpen((open) => !open)}
        className={cn(
          'flex items-center gap-2 rounded-2xl border border-[var(--input-border)] px-2.5 py-2.5 transition-colors md:gap-3 md:px-3',
          'glass hover:bg-[var(--input-bg)]'
        )}
      >
        {user.avatar_url ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={user.avatar_url}
            alt={`${displayName} avatar`}
            referrerPolicy="no-referrer"
            className="h-9 w-9 rounded-full border border-[var(--card-border)] object-cover"
          />
        ) : (
          <div className="flex h-9 w-9 items-center justify-center rounded-full bg-gradient-to-br from-cyan-500 via-sky-500 to-blue-600 text-sm font-bold text-white shadow-lg shadow-cyan-500/20">
            {initials}
          </div>
        )}

        <div className="hidden min-w-0 text-left lg:block">
          <div className="max-w-36 truncate text-sm font-semibold text-theme-primary">{displayName}</div>
          <div className="max-w-36 truncate text-xs text-theme-muted">{user.email}</div>
        </div>

        <ChevronDown className={cn('h-4 w-4 text-theme-muted transition-transform', isOpen && 'rotate-180')} />
      </button>

      {isOpen && (
        <>
          <div className="fixed inset-0 z-40" onClick={() => setIsOpen(false)} />
          <div className="absolute right-0 top-full z-50 mt-3 w-72">
            <div className="glass switcher-dropdown-panel overflow-hidden rounded-2xl border border-[var(--card-border)] shadow-2xl">
              <div className="border-b border-[var(--card-border)] px-4 py-4">
                <div className="mb-3 flex items-center gap-3">
                  {user.avatar_url ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img
                      src={user.avatar_url}
                      alt={`${displayName} avatar`}
                      referrerPolicy="no-referrer"
                      className="h-11 w-11 rounded-full border border-[var(--card-border)] object-cover"
                    />
                  ) : (
                    <div className="flex h-11 w-11 items-center justify-center rounded-full bg-gradient-to-br from-cyan-500 via-sky-500 to-blue-600 text-sm font-bold text-white shadow-lg shadow-cyan-500/20">
                      {initials}
                    </div>
                  )}

                  <div className="min-w-0">
                    <div className="truncate text-sm font-semibold text-theme-primary">{displayName}</div>
                    <div className="truncate text-xs text-theme-muted">{user.email}</div>
                  </div>
                </div>

                <div className="inline-flex items-center gap-2 rounded-full border border-cyan-500/25 bg-cyan-500/10 px-3 py-1 text-xs text-cyan-300">
                  <ShieldCheck className="h-3.5 w-3.5" />
                  用户 ID: {user.id}
                </div>
              </div>

              <div className="border-b border-[var(--card-border)] p-2">
                <Link
                  href="/watchlist"
                  onClick={() => setIsOpen(false)}
                  className="flex items-center gap-3 rounded-2xl px-3 py-3 text-left transition-colors hover:bg-[var(--input-bg)]"
                >
                  <div className="rounded-xl bg-cyan-500/15 p-2 text-cyan-300">
                    <Layers3 className="h-4 w-4" />
                  </div>
                  <div>
                    <div className="text-sm font-medium text-theme-primary">你的自选</div>
                    <div className="text-xs text-theme-muted">分组查看基金走势与预估涨跌幅</div>
                  </div>
                </Link>

                <Link
                  href="/holdings"
                  onClick={() => setIsOpen(false)}
                  className="flex items-center gap-3 rounded-2xl px-3 py-3 text-left transition-colors hover:bg-[var(--input-bg)]"
                >
                  <div className="rounded-xl bg-amber-500/15 p-2 text-amber-300">
                    <Wallet className="h-4 w-4" />
                  </div>
                  <div>
                    <div className="text-sm font-medium text-theme-primary">持仓明细</div>
                    <div className="text-xs text-theme-muted">查看持仓金额与实时预估涨跌额</div>
                  </div>
                </Link>
              </div>

              <button
                type="button"
                onClick={() => void handleLogout()}
                disabled={isLoggingOut}
                className="flex w-full items-center justify-between px-4 py-3 text-left transition-colors hover:bg-[var(--input-bg)] disabled:cursor-not-allowed disabled:opacity-60"
              >
                <div className="flex items-center gap-3">
                  <div className="rounded-xl bg-rose-500/15 p-2 text-rose-300">
                    {isLoggingOut ? <Loader2 className="h-4 w-4 animate-spin" /> : <LogOut className="h-4 w-4" />}
                  </div>
                  <div>
                    <div className="text-sm font-medium text-theme-primary">退出登录</div>
                    <div className="text-xs text-theme-muted">清除当前浏览器会话</div>
                  </div>
                </div>
              </button>
            </div>
          </div>
        </>
      )}
    </div>
  )
}
