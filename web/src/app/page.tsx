'use client'

import { Suspense, useEffect, useMemo, useRef, useState, useTransition } from 'react'
import { useSearchParams } from 'next/navigation'
import { useFundAnalysis, useFundDashboard, useFundHoldings, useTimeSeries } from '@/hooks/use-fund-data'
import { useMarketStatus, getSessionLabel, formatTimeUntil } from '@/hooks/use-market-status'
import { useUIPreferences } from '@/hooks/use-ui-preferences'
import { FundSearch } from '@/components/fund-search'
import { EstimateCard } from '@/components/estimate-card'
import { FundAnalysisCard } from '@/components/fund-analysis-card'
import { FundSectorCard } from '@/components/fund-sector-card'
import { IntradayChart } from '@/components/intraday-chart'
import { HoldingsTable } from '@/components/holdings-table'
import { TargetETFHoldingsCard } from '@/components/target-etf-holdings-card'
import { ThemeSwitcher } from '@/components/theme-switcher'
import { BrandMark } from '@/components/brand-mark'
import { MarketStatusIndicator } from '@/components/market-status-indicator'
import { FundLoadingIndicator } from '@/components/loading-indicator'
import { UserAccountMenu } from '@/components/user-account-menu'
import { ScrollReveal, ScrollRevealStack } from '@/components/scroll-reveal'
import { SiteFooter } from '@/components/site-footer'
import { ActionButton } from '@/components/ui/action-button'
import { Surface } from '@/components/ui/surface'
import { StatusBanner } from '@/components/ui/status-banner'
import { BarChart3, TrendingUp, Clock, RefreshCw } from 'lucide-react'
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
  const isCallAuction = marketStatus.session === 'call_auction'

  const syncFundInUrl = (fundId: string) => {
    if (typeof window === 'undefined') {
      return
    }

    const url = new URL(window.location.href)
    if (fundId === DEFAULT_FUND_ID) {
      url.searchParams.delete('fund')
    } else {
      url.searchParams.set('fund', fundId)
    }
    window.history.replaceState({}, '', `${url.pathname}${url.search}${url.hash}`)
  }

  const handleDashboardSuccess = (data: { estimate?: { fund_id?: string } }) => {
    const resolvedFundId = data?.estimate?.fund_id
    if (!resolvedFundId) return

    setSelectionError(null)
    lastStableFundIdRef.current = resolvedFundId
    if (switchingFundIdRef.current === resolvedFundId) {
      setSwitchingFundId(null)
    }
  }

  const handleDashboardError = (err: unknown) => {
    const failedFundId = switchingFundIdRef.current
    if (!failedFundId) return

    const message = err instanceof Error ? err.message : '加载失败'
    setSelectionError(`基金 ${failedFundId} 加载失败：${message}`)
    setSwitchingFundId(null)

    startTransition(() => {
      setCurrentFundId(lastStableFundIdRef.current)
      syncFundInUrl(lastStableFundIdRef.current)
    })
  }

  // SWR 数据获取 hooks - 首页统一用 dashboard 快照，避免卡片与图表分叉
  const {
    fund,
    estimate,
    sectorSnapshot,
    themeSnapshot,
    classificationOverride,
    cacheStatus,
    isLoading: isDashboardLoading,
    isValidating,
    mutate: refreshDashboard,
    isTrading,
    refreshInterval,
    isWarming: isDashboardWarming,
  } = useFundDashboard(isCallAuction ? null : currentFundId, {
    onSuccess: handleDashboardSuccess,
    onError: handleDashboardError,
  })
  const {
    analysis,
    isLoading: isAnalysisLoading,
    mutate: refreshAnalysis,
  } = useFundAnalysis(isCallAuction ? null : currentFundId)
  const {
    timeSeries,
    displayDate,
    isHistorical,
    officialClose,
    isLoading: isTimeSeriesLoading,
    mutate: refreshTimeSeries,
  } = useTimeSeries(isCallAuction ? null : currentFundId)
  const {
    holdings: resolvedHoldings,
    displayItems: holdingsDisplayItems,
    displayLevel: holdingsDisplayLevel,
    lookthroughAvailable,
  } = useFundHoldings(currentFundId)

  // 切换基金时使用 transition 避免阻塞
  const handleFundSelect = (fundId: string) => {
    if (fundId === currentFundId && !selectionError) return

    setSelectionError(null)
    setSwitchingFundId(fundId)
    syncFundInUrl(fundId)

    startTransition(() => {
      setCurrentFundId(fundId)
    })
  }

  const isFundSwitching = Boolean(
    !isCallAuction &&
    switchingFundId &&
    (isDashboardLoading || isDashboardWarming || estimate?.fund_id !== switchingFundId)
  )

  // 超时自动关闭加载指示器（防止无限加载）
  useEffect(() => {
    switchingFundIdRef.current = switchingFundId
  }, [switchingFundId])

  useEffect(() => {
    if (isFundSwitching && !isDashboardWarming) {
      const timeout = window.setTimeout(() => {
        setSwitchingFundId(null)
      }, 15000)
      return () => window.clearTimeout(timeout)
    }
  }, [isDashboardWarming, isFundSwitching, switchingFundId])

  // 手动刷新
  const handleRefresh = () => {
    setSelectionError(null)
    refreshDashboard()
    refreshAnalysis()
    refreshTimeSeries()
  }

  const lastUpdated = estimate?.calculated_at ? new Date(estimate.calculated_at) : null

  const warmupNotice = isDashboardWarming
    ? `基金 ${currentFundId} 数据预热中，正在自动重试。`
    : cacheStatus === 'warming'
      ? `基金 ${currentFundId} 的基础资料正在后台补全，页面会自动刷新。`
      : ''
  const activeEstimate = isCallAuction ? undefined : estimate
  const activeFund = isCallAuction ? undefined : fund
  const activeTimeSeries = isCallAuction ? [] : timeSeries
  const activeLastUpdated = isCallAuction ? null : lastUpdated
  const activeAnalysis = isCallAuction ? undefined : analysis
  const warmupDetailText = isDashboardWarming
    ? '数据预热中，约 5 秒后自动重试'
    : isCallAuction
      ? '集合竞价中，等待 09:30 开盘后更新基金数据'
      : warmupNotice

  const displayHoldingCoverageCount = holdingsDisplayItems.length
  const displayHoldingRatio = useMemo(
    () => holdingsDisplayItems.reduce((sum, holding) => {
      const rawValue = holdingsDisplayLevel === 'target_layer' ? holding.weight_percent : holding.holding_ratio
      return sum + parseFloat(rawValue || '0')
    }, 0),
    [holdingsDisplayItems, holdingsDisplayLevel]
  )
  // 计算 Top 贡献者 / 集合竞价下的重仓股 TOP3
  const topContributors = (activeEstimate?.holding_details ?? [])
    .slice()
    .sort((a, b) => parseFloat(b.contribution) - parseFloat(a.contribution))
    .slice(0, 3)
  const topDisplayItems = useMemo(
    () => holdingsDisplayItems
      .slice()
      .sort((a, b) => {
        const left = holdingsDisplayLevel === 'target_layer' ? parseFloat(a.weight_percent || '0') : parseFloat(a.holding_ratio || '0')
        const right = holdingsDisplayLevel === 'target_layer' ? parseFloat(b.weight_percent || '0') : parseFloat(b.holding_ratio || '0')
        return right - left
      })
      .slice(0, 3),
    [holdingsDisplayItems, holdingsDisplayLevel]
  )
  const marketStatusLabel = !marketStatus.mounted
    ? '加载中...'
    : marketStatus.isTrading
      ? '交易中'
      : getSessionLabel(marketStatus.session)

  return (
    <div className="min-h-[100dvh]">
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
            <BrandMark subtitle="FundLive - 实时基金估值" />

            {/* Search */}
            <div className="hidden shrink-0 md:block">
              <FundSearchWrapper onSelect={handleFundSelect} currentFundId={currentFundId} />
            </div>

            <nav className="hidden items-center gap-2 xl:flex">
              {[
                { href: '/issues', label: '反馈与想法' },
                { href: '/announcements', label: '更新公告' },
                { href: '/analysis/rankings', label: '量化排行榜' },
              ].map((item) => (
                <ActionButton key={item.href} href={item.href} variant="subtle" size="sm">
                  {item.label}
                </ActionButton>
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
          <div className="mt-4 flex justify-end md:hidden">
            <FundSearchWrapper onSelect={handleFundSelect} currentFundId={currentFundId} />
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main id="main-content" className="container mx-auto px-4 py-8">
        {selectionError && (
          <StatusBanner className="mb-6" tone="warning" onDismiss={() => setSelectionError(null)}>
            {selectionError}
          </StatusBanner>
        )}
        {warmupNotice && !isCallAuction && (
          <StatusBanner
            className="mb-6"
            tone="info"
            icon={<RefreshCw className={cn('h-4 w-4', isDashboardWarming ? 'animate-spin' : '')} />}
          >
            {warmupNotice}
          </StatusBanner>
        )}
        {isCallAuction && (
          <StatusBanner className="mb-6" tone="warning" icon={<Clock className="h-4 w-4" />}>
            集合竞价中，等待 09:30 开盘后更新基金数据。
          </StatusBanner>
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
          <ScrollReveal className="flex min-h-[60vh] items-center justify-center" variant="scale-in">
            <EstimateCard
              estimate={activeEstimate}
              fund={activeFund}
              isLoading={isDashboardLoading}
              isCallAuction={isCallAuction}
              isValidating={isValidating}
              lastUpdated={activeLastUpdated}
              className="w-full max-w-2xl"
            />
          </ScrollReveal>
        ) : (
          /* ===== Professional Mode ===== */
          <ScrollRevealStack className="space-y-6">
            {/* Top Section: Estimate Card + Stats */}
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
              <EstimateCard
                estimate={activeEstimate}
                fund={activeFund}
                isLoading={isDashboardLoading}
                isCallAuction={isCallAuction}
                isValidating={isValidating}
                lastUpdated={activeLastUpdated}
                className="lg:col-span-2"
              />

              {/* Quick Stats */}
              <div className="space-y-4">
                {/* Trading Status Card */}
                <Surface padding="md" radius="md">
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
                      <span className="text-theme-secondary">
                        {holdingsDisplayLevel === 'target_layer' ? '追踪目标数' : '重仓股覆盖'}
                      </span>
                      <span className="text-theme-primary font-medium">
                        {holdingsDisplayLevel === 'target_layer'
                          ? `${displayHoldingCoverageCount} 个`
                          : isCallAuction
                            ? `${displayHoldingCoverageCount} / 10`
                            : `${estimate?.holding_details?.length || 0} / 10`}
                      </span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-theme-secondary">
                        {holdingsDisplayLevel === 'target_layer' ? '目标层级' : '持仓占比'}
                      </span>
                      <span className="text-theme-primary font-medium">
                        {holdingsDisplayLevel === 'target_layer'
                          ? '下一层目标'
                          : isCallAuction
                            ? `${displayHoldingRatio.toFixed(2)}%`
                            : `${parseFloat(estimate?.total_hold_ratio || '0').toFixed(2)}%`}
                      </span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-theme-secondary">数据来源</span>
                      <span className="text-cyan-400 font-medium">
                        {holdingsDisplayLevel === 'target_layer'
                          ? (lookthroughAvailable ? '追踪目标' : '跟踪标的')
                          : isCallAuction
                            ? '静态持仓'
                            : (estimate?.data_source || 'N/A')}
                      </span>
                    </div>
                  </div>
                </Surface>

                {/* Top Contributors */}
                <Surface padding="md" radius="md">
                  <div className="flex items-center gap-3 mb-4">
                    <div className="p-2 rounded-lg bg-[var(--accent-up)]/20">
                      <TrendingUp className="w-5 h-5 text-up" />
                    </div>
                    <h3 className="font-semibold text-theme-primary">
                      {holdingsDisplayLevel === 'target_layer'
                        ? '追踪目标 TOP'
                        : isCallAuction
                          ? '重仓股 TOP3'
                          : '涨幅贡献 TOP3'}
                    </h3>
                  </div>
                  <div className="space-y-2">
                    {holdingsDisplayLevel !== 'target_layer' && !isCallAuction && topContributors.map((holding, index) => {
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
                    {holdingsDisplayLevel === 'target_layer' ? (
                      topDisplayItems.length > 0 ? topDisplayItems.map((holding, index) => (
                        <div
                          key={`${holding.item_type}:${holding.code || holding.name}:${index}`}
                          className="flex items-center justify-between py-2 border-b border-[var(--card-border)] last:border-0"
                        >
                          <div className="flex items-center gap-2">
                            <span className="text-xs text-theme-muted w-4">{index + 1}</span>
                            <span className="text-sm text-theme-primary">{holding.name}</span>
                          </div>
                          <span className="text-sm font-medium text-theme-secondary">
                            {holding.target_type === 'etf_fund'
                              ? 'ETF'
                              : holding.target_type === 'index'
                                ? '指数'
                                : '基金'}
                          </span>
                        </div>
                      )) : (
                        <p className="text-sm text-theme-muted text-center py-4">暂无追踪目标</p>
                      )
                    ) : isCallAuction ? (
                      topDisplayItems.length > 0 ? topDisplayItems.map((holding, index) => (
                        <div
                          key={`${holding.item_type}:${holding.code || holding.name}:${index}`}
                          className="flex items-center justify-between py-2 border-b border-[var(--card-border)] last:border-0"
                        >
                          <div className="flex items-center gap-2">
                            <span className="text-xs text-theme-muted w-4">{index + 1}</span>
                            <span className="text-sm text-theme-primary">{holding.name}</span>
                          </div>
                          <span className="text-sm font-medium text-theme-secondary">
                            {parseFloat(holding.holding_ratio || '0').toFixed(2)}%
                          </span>
                        </div>
                      )) : (
                        <p className="text-sm text-theme-muted text-center py-4">暂无持仓数据</p>
                      )
                    ) : topContributors.length === 0 && (
                      <p className="text-sm text-theme-muted text-center py-4">暂无数据</p>
                    )}
                    {isCallAuction && (
                      <p className="pt-2 text-xs text-theme-muted">
                        集合竞价阶段保留固定持仓与占比信息，贡献值等待 09:30 开盘后恢复。
                      </p>
                    )}
                    {holdingsDisplayLevel === 'target_layer' && (
                      <p className="pt-2 text-xs text-theme-muted">
                        默认只展示下一层追踪目标；底层股票仅用于估值计算，不在这里继续下钻。
                      </p>
                    )}
                  </div>
                </Surface>
              </div>
            </div>

            {/* Chart Section */}
            <IntradayChart
              timeSeries={activeTimeSeries}
              estimate={activeEstimate}
              isLoading={isTimeSeriesLoading}
              isCallAuction={isCallAuction}
              displayDate={displayDate}
              isHistorical={isHistorical}
              officialClose={officialClose}
            />

            <FundSectorCard
              fund={activeFund}
              sectorSnapshot={isCallAuction ? undefined : sectorSnapshot}
              themeSnapshot={isCallAuction ? undefined : themeSnapshot}
              classificationOverride={isCallAuction ? undefined : classificationOverride}
              onClassificationOverrideUpdated={() => {
                void refreshDashboard()
                void refreshAnalysis()
              }}
            />

            {/* Holdings Table */}
            <HoldingsTable
              estimate={activeEstimate}
              displayLevel={holdingsDisplayLevel}
              items={holdingsDisplayItems}
              lookthroughAvailable={lookthroughAvailable}
              isCallAuction={isCallAuction}
            />

            {holdingsDisplayLevel === 'target_layer' && resolvedHoldings.length > 0 && (
              <TargetETFHoldingsCard
                targetName={holdingsDisplayItems[0]?.name}
                holdings={resolvedHoldings}
              />
            )}

            <FundAnalysisCard
              analysis={activeAnalysis}
              fundId={activeFund?.id || currentFundId}
              isLoading={!isCallAuction && isAnalysisLoading}
            />
          </ScrollRevealStack>
        )}
      </main>

      <SiteFooter className="mt-12" compact />
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
