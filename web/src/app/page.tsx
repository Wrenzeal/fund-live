'use client'

import { Suspense, useEffect, useMemo, useRef, useState, useTransition } from 'react'
import { useSearchParams } from 'next/navigation'
import { useFundAnalysis, useFundDashboard, useFundHoldings, useTimeSeries } from '@/hooks/use-fund-data'
import { useMarketStatus, getSessionLabel } from '@/hooks/use-market-status'
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
import { StatusBanner } from '@/components/ui/status-banner'
import { HomeInsightRail, HomeVisualShell } from '@/components/home-visual-shell'
import { Clock, RefreshCw } from 'lucide-react'
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
            <div className="flex-1 max-w-md hidden md:block">
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
          <div className="mt-4 md:hidden">
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
          <div className="space-y-8">
            <ScrollReveal>
              <HomeVisualShell
                currentFundId={currentFundId}
                estimate={activeEstimate}
                fund={activeFund}
                lastUpdated={activeLastUpdated}
                analysis={activeAnalysis}
                marketStatus={marketStatus}
                marketStatusLabel={marketStatusLabel}
                isCallAuction={isCallAuction}
                isTrading={isTrading}
                isLoading={isDashboardLoading}
                isWarming={isDashboardWarming}
                isValidating={isValidating}
                refreshInterval={refreshInterval}
                holdingsDisplayLevel={holdingsDisplayLevel}
                displayHoldingCoverageCount={displayHoldingCoverageCount}
                displayHoldingRatio={displayHoldingRatio}
                lookthroughAvailable={lookthroughAvailable}
                topContributors={topContributors}
                topDisplayItems={topDisplayItems}
                onSelect={handleFundSelect}
                onRefresh={handleRefresh}
              />
            </ScrollReveal>

            <ScrollRevealStack className="space-y-6">
              <div className="grid grid-cols-1 gap-6 xl:grid-cols-[minmax(0,1.12fr)_minmax(20rem,0.88fr)]">
                <EstimateCard
                  estimate={activeEstimate}
                  fund={activeFund}
                  isLoading={isDashboardLoading}
                  isCallAuction={isCallAuction}
                  isValidating={isValidating}
                  lastUpdated={activeLastUpdated}
                  className="min-h-full"
                />

                <HomeInsightRail
                  marketStatus={marketStatus}
                  marketStatusLabel={marketStatusLabel}
                  holdingsDisplayLevel={holdingsDisplayLevel}
                  displayHoldingCoverageCount={displayHoldingCoverageCount}
                  displayHoldingRatio={displayHoldingRatio}
                  isCallAuction={isCallAuction}
                  estimate={estimate}
                  lookthroughAvailable={lookthroughAvailable}
                  topContributors={topContributors}
                  topDisplayItems={topDisplayItems}
                />
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
          </div>
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
