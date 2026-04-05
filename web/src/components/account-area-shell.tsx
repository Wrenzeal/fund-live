'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { Activity, ChevronRight, Crown, Layers3, Sparkles, Wallet } from 'lucide-react'
import { HeaderFundSearch } from '@/components/header-fund-search'
import { ThemeSwitcher } from '@/components/theme-switcher'
import { UserAccountMenu } from '@/components/user-account-menu'
import { useUIPreferences } from '@/hooks/use-ui-preferences'
import { cn } from '@/lib/utils'

interface AccountAreaShellProps {
  title: string
  description: string
  children: React.ReactNode
}

const tabs = [
  { href: '/watchlist', label: '你的自选', icon: Layers3 },
  { href: '/holdings', label: '持仓明细', icon: Wallet },
  { href: '/vip', label: 'VIP 分析', icon: Crown },
]

export function AccountAreaShell({ title, description, children }: AccountAreaShellProps) {
  const pathname = usePathname()
  const { themeType, setThemeType, viewMode, setViewMode } = useUIPreferences()

  return (
    <div className="min-h-screen">
      <header className="sticky top-0 z-50 border-b border-[var(--card-border)] glass-strong">
        <div className="container mx-auto px-4 py-4">
          <div className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-4">
              <Link href="/" className="flex items-center gap-3">
                <div className="relative">
                  <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-gradient-to-br from-cyan-500 via-sky-500 to-blue-600 text-white shadow-lg shadow-cyan-500/20">
                    <Activity className="h-6 w-6" />
                  </div>
                  <div className="absolute -inset-1 rounded-2xl bg-gradient-to-br from-cyan-500 to-blue-600 opacity-30 blur" />
                </div>
                <div>
                  <div className="text-lg font-bold gradient-text">FundLive</div>
                  <div className="text-xs text-theme-muted">用户模块</div>
                </div>
              </Link>

              <div className="hidden items-center gap-2 text-sm text-theme-muted lg:flex">
                <ChevronRight className="h-4 w-4" />
                <span>{title}</span>
              </div>
            </div>

            <div className="hidden max-w-md flex-1 md:block">
              <HeaderFundSearch />
            </div>

            <div className="flex items-center gap-3">
              <UserAccountMenu />
              <ThemeSwitcher
                themeType={themeType}
                setThemeType={setThemeType}
                viewMode={viewMode}
                setViewMode={setViewMode}
                hideViewMode
              />
            </div>
          </div>

          <div className="mt-4 md:hidden">
            <HeaderFundSearch />
          </div>

          <div className="mt-5 flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
            <div className="max-w-3xl space-y-3">
              <div className="inline-flex items-center gap-2 rounded-full border border-cyan-500/25 bg-cyan-500/10 px-4 py-2 text-xs tracking-[0.3em] text-cyan-300">
                <Sparkles className="h-3.5 w-3.5" />
                USER SPACE
              </div>
              <div>
                <h1 className="text-3xl font-black text-theme-primary sm:text-4xl">{title}</h1>
                <p className="mt-2 max-w-2xl text-sm leading-6 text-theme-secondary sm:text-base">
                  {description}
                </p>
              </div>
            </div>

            <nav className="flex flex-wrap gap-3">
              {tabs.map((tab) => {
                const Icon = tab.icon
                const isVIPTab = tab.href === '/vip'
                const active = tab.href === '/vip'
                  ? pathname.startsWith('/vip')
                  : pathname === tab.href
                return (
                  <Link
                    key={tab.href}
                    href={tab.href}
                    className={cn(
                      'group relative inline-flex items-center gap-2 overflow-hidden rounded-2xl border px-4 py-2.5 text-sm transition-all duration-200',
                      'hover:-translate-y-0.5 active:scale-[0.985]',
                      isVIPTab && 'vip-tab-shell',
                      isVIPTab
                        ? active
                          ? 'vip-tab-shell-active'
                          : 'vip-tab-shell-idle'
                        : active
                          ? 'account-standard-tab account-standard-tab-active border-cyan-500/40 bg-cyan-500/15 text-cyan-300 shadow-[0_14px_28px_rgba(34,211,238,0.14)]'
                          : 'account-standard-tab account-standard-tab-idle border-[var(--input-border)] bg-[var(--input-bg)] text-theme-secondary hover:border-cyan-400/35 hover:bg-cyan-400/10 hover:text-theme-primary hover:shadow-[0_12px_24px_rgba(34,211,238,0.10)]'
                    )}
                  >
                    <span className={cn(isVIPTab ? 'vip-tab-shine' : 'account-tab-shine')} />
                    {isVIPTab && <span className="vip-tab-glow" />}
                    <span
                      className={cn(
                        'relative z-10 flex items-center gap-2',
                        active && (isVIPTab ? 'vip-tab-active' : 'account-tab-active')
                      )}
                    >
                      <Icon
                        className={cn(
                          'h-4 w-4 transition-transform duration-300',
                          isVIPTab
                            ? active
                              ? 'scale-110'
                              : 'group-hover:rotate-6 group-hover:scale-115'
                            : active
                              ? 'scale-105'
                              : 'group-hover:-rotate-6 group-hover:scale-110'
                        )}
                      />
                      {tab.label}
                    </span>
                  </Link>
                )
              })}
            </nav>
          </div>
        </div>
      </header>

      <main className="container mx-auto px-4 py-8">
        {children}
      </main>
    </div>
  )
}
