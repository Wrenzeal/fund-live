'use client'

import Link from 'next/link'
import { usePathname, useRouter } from 'next/navigation'
import type { Ref } from 'react'
import { Clock, RefreshCw } from 'lucide-react'
import { BrandMark } from '@/components/brand-mark'
import { FundSearch } from '@/components/fund-search'
import { MarketStatusIndicator } from '@/components/market-status-indicator'
import { ThemeSwitcher } from '@/components/theme-switcher'
import { UserAccountMenu } from '@/components/user-account-menu'
import { useUIPreferences } from '@/hooks/use-ui-preferences'
import type { MarketStatus } from '@/hooks/use-market-status'
import { cn } from '@/lib/utils'

interface AppTopBarProps {
  currentFundId?: string
  onFundSelect?: (fundId: string) => void
  marketStatus?: MarketStatus & { mounted?: boolean }
  isTrading?: boolean
  refreshInterval?: number
  isRefreshing?: boolean
  onRefresh?: () => void
  headerRef?: Ref<HTMLElement>
  topBarRef?: Ref<HTMLDivElement>
}

const navItems = [
  { href: '/', label: '首页' },
  { href: '/watchlist', label: '自选' },
  { href: '/holdings', label: '持仓' },
  { href: '/analysis/rankings', label: '量化' },
]

function isNavItemActive(pathname: string, href: string) {
  return href === '/' ? pathname === '/' : pathname.startsWith(href)
}

function PrimaryNavigation({ pathname, placement }: { pathname: string; placement: 'inline' | 'row' }) {
  return (
    <nav
      aria-label="主要页面"
      data-primary-navigation={placement}
      className={cn(
        'bg-[var(--input-bg)]/45 p-1',
        placement === 'inline'
          ? 'hidden shrink-0 items-center rounded-xl xl:flex'
          : 'mt-3 grid grid-cols-4 rounded-xl xl:hidden',
      )}
    >
      {navItems.map((item) => {
        const active = isNavItemActive(pathname, item.href)

        return (
          <Link
            key={item.href}
            href={item.href}
            aria-current={active ? 'page' : undefined}
            className={cn(
              'group relative flex min-h-10 items-center justify-center rounded-lg px-3 text-sm font-medium text-theme-muted outline-none transition-colors duration-160',
              'hover:bg-[var(--card-bg)]/55 hover:text-theme-primary focus-visible:bg-[var(--card-bg)]/55 focus-visible:text-theme-primary focus-visible:ring-2 focus-visible:ring-[var(--input-focus)]/55',
              active && 'font-semibold text-theme-primary',
            )}
          >
            <span>{item.label}</span>
            <span
              aria-hidden="true"
              className={cn(
                'absolute bottom-1 h-0.5 w-4 rounded-full bg-[var(--accent-primary)] transition-[opacity,transform] duration-160',
                active ? 'scale-x-100 opacity-100' : 'scale-x-50 opacity-0 group-hover:opacity-35',
              )}
            />
          </Link>
        )
      })}
    </nav>
  )
}

export function AppTopBar({
  currentFundId,
  onFundSelect,
  marketStatus,
  isTrading,
  refreshInterval,
  isRefreshing,
  onRefresh,
  headerRef,
  topBarRef,
}: AppTopBarProps) {
  const router = useRouter()
  const pathname = usePathname()
  const { themeType, setThemeType, viewMode, setViewMode } = useUIPreferences()
  const showTradingInterval = Boolean(isTrading && refreshInterval)

  const handleFundSelect = (fundId: string) => {
    if (onFundSelect) {
      onFundSelect(fundId)
      return
    }

    router.push(`/?fund=${encodeURIComponent(fundId)}`)
  }

  return (
    <header ref={headerRef} className="sticky top-0 z-50 glass-strong border-b border-[var(--card-border)]">
      <div className="container mx-auto px-4 py-3">
        <div ref={topBarRef} className="flex items-center justify-between gap-4">
          <BrandMark subtitle="FundLive - 实时基金估值" />

          <div className="hidden shrink-0 md:block">
            <FundSearch onSelect={handleFundSelect} currentFundId={currentFundId} />
          </div>

          <PrimaryNavigation pathname={pathname} placement="inline" />

          <div className="flex items-center gap-4">
            <div className="hidden lg:flex items-center gap-4">
              <MarketStatusIndicator showDetails status={marketStatus} />

              {showTradingInterval && (
                <div className="flex items-center gap-2 text-xs text-theme-muted">
                  <Clock className="w-3 h-3" />
                  <span>{(refreshInterval ?? 0) / 1000}s 刷新</span>
                </div>
              )}

              {onRefresh && (
                <button
                  onClick={onRefresh}
                  disabled={isRefreshing}
                  className={cn(
                    'p-2 rounded-lg transition-all glass',
                    'hover:bg-[var(--input-bg)]',
                    'disabled:opacity-50 disabled:cursor-not-allowed'
                  )}
                  title="手动刷新"
                >
                  <RefreshCw className={cn('w-4 h-4 text-theme-secondary', isRefreshing && 'animate-spin')} />
                </button>
              )}
            </div>

            <UserAccountMenu />

            <ThemeSwitcher
              themeType={themeType}
              setThemeType={setThemeType}
              viewMode={viewMode}
              setViewMode={setViewMode}
            />
          </div>
        </div>

        <div className="mt-4 flex justify-end md:hidden">
          <FundSearch onSelect={handleFundSelect} currentFundId={currentFundId} />
        </div>
        <PrimaryNavigation pathname={pathname} placement="row" />
      </div>
    </header>
  )
}
