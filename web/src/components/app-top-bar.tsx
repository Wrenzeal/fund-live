'use client'

import { usePathname, useRouter } from 'next/navigation'
import type { Ref } from 'react'
import { Clock, RefreshCw } from 'lucide-react'
import { BrandMark } from '@/components/brand-mark'
import { FundSearch } from '@/components/fund-search'
import { MarketStatusIndicator } from '@/components/market-status-indicator'
import { ThemeSwitcher } from '@/components/theme-switcher'
import { UserAccountMenu } from '@/components/user-account-menu'
import { ActionButton } from '@/components/ui/action-button'
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
      <div className="container mx-auto px-4 py-4">
        <div ref={topBarRef} className="flex items-center justify-between gap-4">
          <BrandMark subtitle="FundLive - 实时基金估值" />

          <div className="hidden shrink-0 md:block">
            <FundSearch onSelect={handleFundSelect} currentFundId={currentFundId} />
          </div>

          <nav className="hidden items-center gap-1 xl:flex" aria-label="主要页面">
            {navItems.map((item) => (
              <ActionButton
                key={item.href}
                href={item.href}
                variant="subtle"
                size="sm"
                aria-current={
                  item.href === '/' ? pathname === '/' || undefined : pathname.startsWith(item.href) ? 'page' : undefined
                }
                className={cn(
                  pathname === item.href || (item.href !== '/' && pathname.startsWith(item.href))
                    ? 'border-[var(--accent-primary)] bg-[var(--accent-primary)]/10 text-theme-primary'
                    : undefined,
                )}
              >
                {item.label}
              </ActionButton>
            ))}
          </nav>

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
        <nav className="mt-3 flex gap-2 overflow-x-auto pb-1 md:hidden" aria-label="主要页面">
          {navItems.map((item) => (
            <ActionButton
              key={item.href}
              href={item.href}
              variant="subtle"
              size="sm"
              aria-current={
                item.href === '/' ? pathname === '/' || undefined : pathname.startsWith(item.href) ? 'page' : undefined
              }
              className={cn(
                'shrink-0',
                pathname === item.href || (item.href !== '/' && pathname.startsWith(item.href))
                  ? 'border-[var(--accent-primary)] bg-[var(--accent-primary)]/10 text-theme-primary'
                  : undefined,
              )}
            >
              {item.label}
            </ActionButton>
          ))}
        </nav>
      </div>
    </header>
  )
}
