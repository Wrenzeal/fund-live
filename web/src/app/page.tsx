'use client'

import Link from 'next/link'
import { Suspense, useEffect, useRef, useState, useTransition } from 'react'
import { useSearchParams } from 'next/navigation'
import { isFundDataWarmingError, useFundEstimate, useFund, useTimeSeries } from '@/hooks/use-fund-data'
import { useMarketStatus, getSessionLabel, formatTimeUntil } from '@/hooks/use-market-status'
import { useUIPreferences } from '@/hooks/use-ui-preferences'
import { FundSearch } from '@/components/fund-search'
import { EstimateCard } from '@/components/estimate-card'
import { IntradayChart } from '@/components/intraday-chart'
import { HoldingsTable } from '@/components/holdings-table'
import { ThemeSwitcher } from '@/components/theme-switcher'
import { MarketStatusIndicator } from '@/components/market-status-indicator'
import { FundLoadingIndicator } from '@/components/loading-indicator'
import { UserAccountMenu } from '@/components/user-account-menu'
import { Activity, AlertTriangle, BarChart3, TrendingUp, Clock, RefreshCw, X } from 'lucide-react'
import { cn } from '@/lib/utils'

// 默认基金 ID
const DEFAULT_FUND_ID = '005827'

export default function Home() {
  return (
    <Suspense fallback={<HomeContent initialFundId={DEFAULT_FUND_ID} />}>
      <HomeWithSearchParams />
    </Suspense>
  )
}

function HomeWithSearchParams() {
  const searchParams = useSearchParams()
  const requestedFundId = searchParams.get('fund')?.trim() || DEFAULT_FUND_ID

  return <HomeContent key={requestedFundId} initialFundId={requestedFundId} />
}

function HomeContent({ initialFundId }: { initialFundId: string }) {
  // 当前选中的基金 ID
  const [currentFundId, setCurrentFundId] = useState<string>(initialFundId)

  const { themeType, setThemeType, viewMode, setViewMode } = useUIPreferences()

  // React 18 useTransition 用于非阻塞更新
  const [isPending, startTransition] = useTransition()

  // 基金切换加载状态
  const [switchingFundId, setSwitchingFundId] = useState<string | null>(null)
  const [selectionError, setSelectionError] = useState<string | null>(null)
  const lastStableFundIdRef = useRef<string>(DEFAULT_FUND_ID)
  const switchingFundIdRef = useRef<string | null>(null)

  // 市场状态 hook
  const marketStatus = useMarketStatus()

  const handleEstimateSuccess = (data: { fund_id?: string }) => {
    if (!data?.fund_id) return

    setSelectionError(null)
    lastStableFundIdRef.current = data.fund_id
    if (switchingFundIdRef.current === data.fund_id) {
      setSwitchingFundId(null)
    }
  }

  const handleEstimateError = (err: unknown) => {
    const failedFundId = switchingFundIdRef.current
    if (!failedFundId) return

    if (isFundDataWarmingError(err)) {
      return
    }

    const message = err instanceof Error ? err.message : '加载失败'
    setSelectionError(`基金 ${failedFundId} 加载失败：${message}`)
    setSwitchingFundId(null)

    startTransition(() => {
      setCurrentFundId(lastStableFundIdRef.current)
    })
  }

  // SWR 数据获取 hooks - 根据市场状态智能轮询
  const {
    estimate,
    isLoading: isEstimateLoading,
    isValidating,
    mutate: refreshEstimate,
    isTrading,
    refreshInterval,
    isWarming: isEstimateWarming,
    warmingMessage: estimateWarmingMessage,
    retryAfterSeconds,
  } = useFundEstimate(currentFundId, {
    onSuccess: handleEstimateSuccess,
    onError: handleEstimateError,
  })

  const { fund, cacheStatus: fundCacheStatus } = useFund(currentFundId)
  const {
    timeSeries,
    displayDate,
    isHistorical,
    isLoading: isTimeSeriesLoading,
    isWarming: isTimeSeriesWarming,
    warmingMessage: timeSeriesWarmingMessage,
  } = useTimeSeries(currentFundId)

  // 切换基金时使用 transition 避免阻塞
  const handleFundSelect = (fundId: string) => {
    if (fundId === currentFundId && !selectionError) return

    setSelectionError(null)
    setSwitchingFundId(fundId)

    startTransition(() => {
      setCurrentFundId(fundId)
    })
  }

  const isFundSwitching = Boolean(
    switchingFundId &&
    (isEstimateLoading || isEstimateWarming || estimate?.fund_id !== switchingFundId)
  )

  // 超时自动关闭加载指示器（防止无限加载）
  useEffect(() => {
    switchingFundIdRef.current = switchingFundId
  }, [switchingFundId])

  useEffect(() => {
    if (isFundSwitching) {
      const timeout = window.setTimeout(() => {
        setSwitchingFundId(null)
      }, isEstimateWarming ? 30000 : 15000) // 预热时放宽等待时间
      return () => window.clearTimeout(timeout)
    }
  }, [isEstimateWarming, isFundSwitching, switchingFundId])

  // 手动刷新
  const handleRefresh = () => {
    setSelectionError(null)
    refreshEstimate()
  }

  const lastUpdated = estimate?.calculated_at ? new Date(estimate.calculated_at) : null

  const warmupNotice = isEstimateWarming
    ? estimateWarmingMessage || `基金 ${currentFundId} 数据预热中，正在自动重试。`
    : isTimeSeriesWarming
      ? timeSeriesWarmingMessage || '分时数据预热中，正在自动重试。'
      : fundCacheStatus === 'warming'
        ? `基金 ${currentFundId} 的基础资料正在后台补全，页面会自动刷新。`
        : ''
  const warmupDetailText = isEstimateWarming
    ? `数据预热中，约 ${Math.max(retryAfterSeconds || 5, 1)} 秒后自动重试`
    : warmupNotice

  // 计算 Top 贡献者
  const topContributors = (estimate?.holding_details ?? [])
    .slice()
    .sort((a, b) => parseFloat(b.contribution) - parseFloat(a.contribution))
    .slice(0, 3)
  const marketStatusLabel = !marketStatus.mounted
    ? '加载中...'
    : marketStatus.isTrading
      ? '交易中'
      : getSessionLabel(marketStatus.session)

  return (
    <div className="min-h-screen">
      {/* 基金切换全屏加载指示器 */}
      <FundLoadingIndicator
        isVisible={isFundSwitching}
        fundName={fund?.name}
        detailText={warmupDetailText}
      />
      {/* Header */}
      <header className="sticky top-0 z-50 glass-strong border-b border-[var(--card-border)]">
        <div className="container mx-auto px-4 py-4">
          <div className="flex items-center justify-between gap-4">
            {/* Logo */}
            <Link href="/" className="flex items-center gap-3">
              <div className="relative">
                <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-gradient-to-br from-cyan-500 via-sky-500 to-blue-600 text-white shadow-lg shadow-cyan-500/20">
                  <Activity className="h-6 w-6" />
                </div>
                <div className="absolute -inset-1 rounded-2xl bg-gradient-to-br from-cyan-500 to-blue-600 opacity-30 blur" />
              </div>
              <div>
                <div className="text-lg font-bold gradient-text">FundLive</div>
                <div className="text-xs text-theme-muted">实时基金估值</div>
              </div>
            </Link>

            {/* Search */}
            <div className="flex-1 max-w-md hidden md:block">
              <FundSearchWrapper onSelect={handleFundSelect} currentFundId={currentFundId} />
            </div>

            <nav className="hidden items-center gap-2 xl:flex">
              {[
                { href: '/issues', label: '我有想法！' },
                { href: '/announcements', label: '更新公告' },
              ].map((item) => (
                <Link
                  key={item.href}
                  href={item.href}
                  className={cn(
                    'group relative overflow-hidden rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2 text-sm text-theme-secondary transition-all duration-200',
                    'hover:-translate-y-0.5 hover:border-cyan-400/35 hover:bg-cyan-400/10 hover:text-theme-primary hover:shadow-[0_12px_24px_rgba(34,211,238,0.10)] active:scale-[0.97]'
                  )}
                >
                  <span className="action-button-shine" />
                  <span className="relative z-10">{item.label}</span>
                </Link>
              ))}
            </nav>

            {/* Controls */}
            <div className="flex items-center gap-4">
              {/* Market status & refresh controls */}
              <div className="hidden lg:flex items-center gap-4">
                <MarketStatusIndicator showDetails status={marketStatus} />

                {/* 仅交易时段显示刷新间隔 */}
                {isTrading && (
                  <div className="flex items-center gap-2 text-xs text-theme-muted">
                    <Clock className="w-3 h-3" />
                    <span>{refreshInterval / 1000}s 刷新</span>
                  </div>
                )}

                {/* 手动刷新按钮 */}
                <button
                  onClick={handleRefresh}
                  disabled={isValidating}
                  className={cn(
                    'p-2 rounded-lg transition-all glass',
                    'hover:bg-[var(--input-bg)]',
                    'disabled:opacity-50 disabled:cursor-not-allowed'
                  )}
                  title="手动刷新"
                >
                  <RefreshCw className={cn('w-4 h-4 text-theme-secondary', isValidating && 'animate-spin')} />
                </button>
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

          {/* Mobile search */}
          <div className="mt-4 md:hidden">
            <FundSearchWrapper onSelect={handleFundSelect} currentFundId={currentFundId} />
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="container mx-auto px-4 py-8">
        {selectionError && (
          <div className="mb-6 flex items-start justify-between gap-4 rounded-2xl border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-100">
            <div className="flex items-start gap-3">
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
              <span>{selectionError}</span>
            </div>
            <button
              type="button"
              onClick={() => setSelectionError(null)}
              className="text-amber-100/70 transition-colors hover:text-amber-100"
              aria-label="关闭错误提示"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        )}
        {warmupNotice && (
          <div className="mb-6 rounded-2xl border border-cyan-500/30 bg-cyan-500/10 px-4 py-3 text-sm text-cyan-50">
            <div className="flex items-start gap-3">
              <RefreshCw className={cn('mt-0.5 h-4 w-4 shrink-0', isEstimateWarming || isTimeSeriesWarming ? 'animate-spin' : '')} />
              <span>{warmupNotice}</span>
            </div>
          </div>
        )}
        {/* 加载过渡状态指示 */}
        {isPending && (
          <div className="fixed top-20 left-1/2 -translate-x-1/2 z-50 glass rounded-full px-4 py-2 text-sm text-theme-secondary flex items-center gap-2">
            <RefreshCw className="w-4 h-4 animate-spin" />
            切换中...
          </div>
        )}

        {viewMode === 'minimal' ? (
          /* ===== Minimal Mode ===== */
          <div className="flex items-center justify-center min-h-[60vh]">
            <EstimateCard
              estimate={estimate}
              fund={fund}
              isLoading={isEstimateLoading}
              isValidating={isValidating}
              lastUpdated={lastUpdated}
              className="w-full max-w-2xl"
            />
          </div>
        ) : (
          /* ===== Professional Mode ===== */
          <div className="space-y-6">
            {/* Top Section: Estimate Card + Stats */}
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
              <EstimateCard
                estimate={estimate}
                fund={fund}
                isLoading={isEstimateLoading}
                isValidating={isValidating}
                lastUpdated={lastUpdated}
                className="lg:col-span-2"
              />

              {/* Quick Stats */}
              <div className="space-y-4">
                {/* Trading Status Card */}
                <div className="glass rounded-2xl p-6">
                  <div className="flex items-center gap-3 mb-4">
                    <div className="p-2 rounded-lg bg-cyan-500/20">
                      <BarChart3 className="w-5 h-5 text-cyan-400" />
                    </div>
                    <h3 className="font-semibold text-theme-primary">市场状态</h3>
                  </div>
                  <div className="space-y-3">
                    <div className="flex justify-between text-sm">
                      <span className="text-theme-secondary">交易状态</span>
                      <span className={cn(
                        'font-medium',
                        marketStatus.mounted && marketStatus.isTrading ? 'market-open' : 'market-closed'
                      )}>
                        {marketStatusLabel}
                      </span>
                    </div>
                    {marketStatus.mounted && !marketStatus.isTrading && marketStatus.timeUntilNextSession > 0 && (
                      <div className="flex justify-between text-sm">
                        <span className="text-theme-secondary">距开盘</span>
                        <span className="text-theme-primary font-medium">
                          {formatTimeUntil(marketStatus.timeUntilNextSession)}
                        </span>
                      </div>
                    )}
                    <div className="flex justify-between text-sm">
                      <span className="text-theme-secondary">重仓股覆盖</span>
                      <span className="text-theme-primary font-medium">
                        {estimate?.holding_details?.length || 0} / 10
                      </span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-theme-secondary">持仓占比</span>
                      <span className="text-theme-primary font-medium">
                        {parseFloat(estimate?.total_hold_ratio || '0').toFixed(2)}%
                      </span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-theme-secondary">数据来源</span>
                      <span className="text-cyan-400 font-medium">
                        {estimate?.data_source || 'N/A'}
                      </span>
                    </div>
                  </div>
                </div>

                {/* Top Contributors */}
                <div className="glass rounded-2xl p-6">
                  <div className="flex items-center gap-3 mb-4">
                    <div className="p-2 rounded-lg bg-[var(--accent-up)]/20">
                      <TrendingUp className="w-5 h-5 text-up" />
                    </div>
                    <h3 className="font-semibold text-theme-primary">涨幅贡献 TOP3</h3>
                  </div>
                  <div className="space-y-2">
                    {topContributors.map((holding, index) => {
                      const contrib = parseFloat(holding.contribution)
                      const isPositive = contrib >= 0
                      return (
                        <div
                          key={holding.stock_code}
                          className="flex items-center justify-between py-2 border-b border-[var(--card-border)] last:border-0"
                        >
                          <div className="flex items-center gap-2">
                            <span className="text-xs text-theme-muted w-4">{index + 1}</span>
                            <span className="text-sm text-theme-primary">{holding.stock_name}</span>
                          </div>
                          <span className={cn('text-sm font-medium', isPositive ? 'text-up' : 'text-down')}>
                            {isPositive ? '+' : ''}{contrib.toFixed(4)}%
                          </span>
                        </div>
                      )
                    })}
                    {topContributors.length === 0 && (
                      <p className="text-sm text-theme-muted text-center py-4">暂无数据</p>
                    )}
                  </div>
                </div>
              </div>
            </div>

            {/* Chart Section */}
            <IntradayChart
              timeSeries={timeSeries}
              estimate={estimate}
              isLoading={isTimeSeriesLoading}
              displayDate={displayDate}
              isHistorical={isHistorical}
            />

            {/* Holdings Table */}
            <HoldingsTable estimate={estimate} />
          </div>
        )}
      </main>

      {/* Footer */}
      <footer className="border-t border-[var(--card-border)] mt-12">
        <div className="container mx-auto px-4 py-6">
          <div className="flex flex-col md:flex-row items-center justify-between gap-4 text-sm text-theme-muted">
            <div>
              <span className="gradient-text font-semibold">FundLive</span>
              {' '}© 2024 - 2026. 实时基金估值系统
            </div>
            <div className="flex items-center gap-4">
              <Link href="/auth/login" className="transition-colors hover:text-theme-primary">账户登录</Link>
              <span>⚠️ 本数据仅供参考，不构成投资建议</span>
            </div>
          </div>
        </div>
      </footer>
    </div>
  )
}

// 搜索组件包装器
function FundSearchWrapper({
  onSelect,
  currentFundId
}: {
  onSelect: (id: string) => void
  currentFundId: string
}) {
  return <FundSearch onSelect={onSelect} currentFundId={currentFundId} />
}
