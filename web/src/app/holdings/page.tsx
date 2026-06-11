'use client'

import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useEffect, useMemo, useRef, useState } from 'react'
import {
  AlertTriangle,
  BarChart4,
  BellRing,
  CheckCircle2,
  ClipboardList,
  FileStack,
  HeartPulse,
  History,
  LoaderCircle,
  Wallet,
} from 'lucide-react'
import { AccountAreaShell } from '@/components/account-area-shell'
import {
  HoldingFeedbackBanner,
  type HoldingFeedbackMessage,
} from '@/components/holding-feedback-banner'
import { ScrollRevealStack } from '@/components/scroll-reveal'
import { HoldingActivityTimeline } from '@/components/holding-activity-timeline'
import { HoldingImportPanel } from '@/components/holding-import-panel'
import { HoldingPortfolioHealthPanel } from '@/components/holding-portfolio-health-panel'
import { HoldingReconciliationPanel } from '@/components/holding-reconciliation-panel'
import { HoldingRecordComposer } from '@/components/holding-record-composer'
import { HoldingReminderPanel } from '@/components/holding-reminder-panel'
import { HoldingsViewControls } from '@/components/holdings-view-controls'
import { HoldingsList } from '@/components/holdings-list'
import { HoldingsSummaryMetrics } from '@/components/holdings-summary-metrics'
import { VIPAnalysisEntry } from '@/components/vip-analysis-entry'
import { useCurrentUser } from '@/hooks/use-auth'
import {
  useFundAnalyses,
  useFundExposureSnapshots,
  useFundSearch,
  useFundTopHoldings,
} from '@/hooks/use-fund-data'
import {
  useMarketStatus,
  usePricingDatePreview,
} from '@/hooks/use-market-status'
import {
  useHoldingEstimateMetrics,
  useUserPortfolio,
  type HoldingTransactionStatusFilter,
  type HoldingTransactionType,
} from '@/hooks/use-user-portfolio'
import { useVIPPreview } from '@/hooks/use-vip-preview'
import { VIP_SAMPLE_REPORT_IDS } from '@/mocks/vip'
import type { HoldingSourceFilter } from '@/lib/holding-sources'
import { cn } from '@/lib/utils'
import {
  aggregateChangeValue,
  aggregateLatestUpdatedAt,
  aggregateProfitValue,
  analysisRecommendationWeight,
  analysisRiskWeight,
  buildTradeAtValue,
  compareOptionalNumbers,
  detailChangeValue,
  detailProfitValue,
  formatSummaryMoney,
  formatTradeDateLabel,
  formatTradeTimingLabel,
  isAggregateIncomplete,
  isHoldingIncomplete,
  matchesAggregateFilter,
  matchesHoldingFilter,
  parseOptionalNumber,
  resolveTradeTimingFromServerClock,
  timestampValue,
  type HoldingFilterMode,
  type HoldingMetricScope,
  type HoldingSortMode,
  type TradeTiming,
} from '@/lib/holding-display'

type HoldingViewMode = 'aggregate' | 'detail'
type HoldingWorkspaceTab =
  | 'summary'
  | 'record'
  | 'list'
  | 'risk'
  | 'ledger'
  | 'tools'

const workspaceTabs: Array<{
  id: HoldingWorkspaceTab
  label: string
  description: string
  icon: typeof Wallet
}> = [
  {
    id: 'summary',
    label: '总览',
    description: '价值、收益、对账',
    icon: Wallet,
  },
  {
    id: 'record',
    label: '记录',
    description: '新增/补仓',
    icon: ClipboardList,
  },
  {
    id: 'list',
    label: '持仓',
    description: '排序、筛选、操作',
    icon: BarChart4,
  },
  { id: 'risk', label: '风险', description: '体检和提醒', icon: HeartPulse },
  {
    id: 'ledger',
    label: '流水',
    description: '买卖、校正、分红',
    icon: History,
  },
  { id: 'tools', label: '工具', description: '批量导入/VIP', icon: BellRing },
]

export default function HoldingsPage() {
  const router = useRouter()
  const { user, isLoading } = useCurrentUser()
  const marketStatus = useMarketStatus()
  const [transactionFundFilter, setTransactionFundFilter] = useState('all')
  const [transactionTypeFilter, setTransactionTypeFilter] = useState<
    HoldingTransactionType | 'all'
  >('all')
  const [transactionStatusFilter, setTransactionStatusFilter] =
    useState<HoldingTransactionStatusFilter>('all')
  const [transactionSourceFilter, setTransactionSourceFilter] =
    useState<HoldingSourceFilter>('all')
  const [transactionKeywordFilter, setTransactionKeywordFilter] = useState('')
  const [transactionStartDateFilter, setTransactionStartDateFilter] =
    useState('')
  const [transactionEndDateFilter, setTransactionEndDateFilter] = useState('')
  const [transactionPage, setTransactionPage] = useState(1)
  const transactionPageSize = 10
  const transactionVisibleLimit = transactionPage * transactionPageSize
  const {
    holdings,
    holdingAggregates,
    holdingTransactions,
    holdingSummary,
    seedDemoData,
    addHolding,
    updateHolding,
    sellHolding,
    recordHoldingDividend,
    adjustHoldingShares,
    removeHolding,
    voidHoldingTransaction,
    previewHoldingTransactionRollback,
    applyHoldingTransactionRollback,
    addHoldingsBatch,
  } = useUserPortfolio(user?.id ?? null, {
    fundID: transactionFundFilter,
    type: transactionTypeFilter,
    status: transactionStatusFilter,
    sourcePlatform: transactionSourceFilter,
    keyword: transactionKeywordFilter,
    startDate: transactionStartDateFilter,
    endDate: transactionEndDateFilter,
    offset: 0,
    limit: transactionVisibleLimit,
  })
  const [query, setQuery] = useState('')
  const [selectedFundID, setSelectedFundID] = useState('')
  const [selectedFundName, setSelectedFundName] = useState('')
  const [amount, setAmount] = useState('')
  const [tradeDate, setTradeDate] = useState('')
  const [tradeTiming, setTradeTiming] = useState<TradeTiming>('before_close')
  const [viewMode, setViewMode] = useState<HoldingViewMode>('aggregate')
  const [workspaceTab, setWorkspaceTab] =
    useState<HoldingWorkspaceTab>('summary')
  const [metricScope, setMetricScope] = useState<HoldingMetricScope>('official')
  const [sortMode, setSortMode] = useState<HoldingSortMode>('default')
  const [filterMode, setFilterMode] = useState<HoldingFilterMode>('all')
  const [showIncompleteOnly, setShowIncompleteOnly] = useState(false)
  const [note, setNote] = useState('')
  const [feedback, setFeedback] = useState<HoldingFeedbackMessage | null>(null)
  const [vipFeedback, setVipFeedback] = useState<HoldingFeedbackMessage | null>(
    null,
  )
  const [isSeedingDemo, setIsSeedingDemo] = useState(false)
  const [isAddingHolding, setIsAddingHolding] = useState(false)
  const defaultsInitializedRef = useRef(false)
  const { results } = useFundSearch(query)
  const fundIDsForAnalysis = useMemo(
    () =>
      Array.from(
        new Set(holdings.map((holding) => holding.fund_id).filter(Boolean)),
      ),
    [holdings],
  )
  const { analysesByFundID } = useFundAnalyses(fundIDsForAnalysis)
  const { exposureSnapshotsByFundID } =
    useFundExposureSnapshots(fundIDsForAnalysis)
  const { topHoldingsByFundID } = useFundTopHoldings(fundIDsForAnalysis)
  const {
    isAdmin: canAccessVIP,
    membership,
    remainingQuota,
    createTask,
    latestCompletedTask,
  } = useVIPPreview()
  const normalizedQuery = query.trim()

  const autoMatchedFund = useMemo(() => {
    if (!normalizedQuery) {
      return null
    }

    const exactMatch = results.find(
      (fund) => fund.id === normalizedQuery || fund.name === normalizedQuery,
    )
    if (exactMatch) {
      return exactMatch
    }

    if (results.length === 1) {
      return results[0]
    }

    return null
  }, [normalizedQuery, results])

  useEffect(() => {
    if (
      defaultsInitializedRef.current ||
      !marketStatus.currentDate ||
      marketStatus.currentTime.getTime() === 0
    ) {
      return
    }

    setTradeDate(marketStatus.currentDate)
    setTradeTiming(resolveTradeTimingFromServerClock(marketStatus.currentTime))
    defaultsInitializedRef.current = true
  }, [marketStatus.currentDate, marketStatus.currentTime])

  useEffect(() => {
    setTransactionPage(1)
  }, [
    transactionFundFilter,
    transactionTypeFilter,
    transactionStatusFilter,
    transactionSourceFilter,
    transactionKeywordFilter,
    transactionStartDateFilter,
    transactionEndDateFilter,
  ])

  const resolvedFundID = selectedFundID || autoMatchedFund?.id || ''
  const resolvedFundName = selectedFundName || autoMatchedFund?.name || ''
  const tradeAtPayload = buildTradeAtValue(tradeDate, tradeTiming)
  const {
    preview,
    isLoading: isPricingPreviewLoading,
    error: pricingPreviewError,
  } = usePricingDatePreview(tradeAtPayload || null)
  const pricingDatePreview = preview?.pricingDate || ''
  const tradeDateLabel = formatTradeDateLabel(tradeDate)
  const tradeTimingLabel = formatTradeTimingLabel(tradeTiming)
  const todayTradeDate = marketStatus.currentDate || tradeDate
  const previousTradeDate = marketStatus.previousTradingDay || ''
  const nextTradeDate = marketStatus.nextTradingDay || ''
  const pricingRuleLabel = !tradeDate
    ? '选择交易日期和提交时段后，会自动显示确认净值日。'
    : pricingPreviewError
      ? pricingPreviewError.message
      : isPricingPreviewLoading
        ? '正在按交易日历校验确认净值日...'
        : preview?.message || '正在按交易日历校验确认净值日...'
  const hasOfficialSummaryMetrics = holdingSummary.real_metrics_ready_count > 0
  const officialSummaryCoverage = hasOfficialSummaryMetrics
    ? `${holdingSummary.real_metrics_ready_count}/${holdingSummary.total_holdings} 条`
    : ''
  const officialReadyPrincipalText = hasOfficialSummaryMetrics
    ? formatSummaryMoney(holdingSummary.ready_principal)
    : '--'
  const totalPrincipalText = formatSummaryMoney(holdingSummary.total_principal)
  const {
    estimatesByFundID,
    aggregateMetrics,
    summary: previewSummary,
  } = useHoldingEstimateMetrics(holdingAggregates)
  const hasPreviewSummaryMetrics = previewSummary.ready_count > 0
  const previewReadyPrincipalText = hasPreviewSummaryMetrics
    ? formatSummaryMoney(previewSummary.ready_principal)
    : '--'
  const holdingsByFundID = useMemo(() => {
    return holdings.reduce<Record<string, typeof holdings>>(
      (groups, holding) => {
        if (!groups[holding.fund_id]) {
          groups[holding.fund_id] = []
        }
        groups[holding.fund_id].push(holding)
        return groups
      },
      {},
    )
  }, [holdings])
  const transactionFundOptions = useMemo(() => {
    const options = holdingAggregates.map((aggregate) => ({
      fund_id: aggregate.fund_id,
      name: aggregate.fund?.name || aggregate.fund_id,
    }))
    return options.sort((left, right) =>
      left.name.localeCompare(right.name, 'zh-Hans-CN'),
    )
  }, [holdingAggregates])
  const recentHoldingAggregates = useMemo(() => {
    return holdingAggregates
      .slice()
      .sort((left, right) =>
        compareOptionalNumbers(
          aggregateLatestUpdatedAt(holdingsByFundID[left.fund_id] ?? []),
          aggregateLatestUpdatedAt(holdingsByFundID[right.fund_id] ?? []),
          'desc',
        ),
      )
  }, [holdingAggregates, holdingsByFundID])
  const incompleteHoldingCount = useMemo(
    () => holdings.filter(isHoldingIncomplete).length,
    [holdings],
  )
  const incompleteAggregateCount = useMemo(
    () => holdingAggregates.filter(isAggregateIncomplete).length,
    [holdingAggregates],
  )
  const filteredHoldingAggregates = useMemo(
    () =>
      holdingAggregates.filter((aggregate) => {
        if (showIncompleteOnly && !isAggregateIncomplete(aggregate)) {
          return false
        }
        return matchesAggregateFilter(
          aggregate,
          aggregateMetrics[aggregate.fund_id],
          metricScope,
          filterMode,
        )
      }),
    [
      aggregateMetrics,
      filterMode,
      holdingAggregates,
      metricScope,
      showIncompleteOnly,
    ],
  )
  const filteredHoldings = useMemo(
    () =>
      holdings.filter((holding) => {
        if (showIncompleteOnly && !isHoldingIncomplete(holding)) {
          return false
        }
        return matchesHoldingFilter(
          holding,
          estimatesByFundID[holding.fund_id],
          metricScope,
          filterMode,
          holdingsByFundID[holding.fund_id]?.length ?? 1,
        )
      }),
    [
      estimatesByFundID,
      filterMode,
      holdings,
      holdingsByFundID,
      metricScope,
      showIncompleteOnly,
    ],
  )
  const sortedHoldingAggregates = useMemo(() => {
    if (sortMode === 'default') {
      return filteredHoldingAggregates
    }

    return filteredHoldingAggregates.slice().sort((left, right) => {
      const leftAnalysis = analysesByFundID[left.fund_id]
      const rightAnalysis = analysesByFundID[right.fund_id]

      if (sortMode === 'analysis_recommendation') {
        return (
          analysisRecommendationWeight(rightAnalysis) -
          analysisRecommendationWeight(leftAnalysis)
        )
      }

      if (sortMode === 'analysis_risk') {
        return (
          analysisRiskWeight(rightAnalysis) - analysisRiskWeight(leftAnalysis)
        )
      }

      if (sortMode === 'principal_desc') {
        return compareOptionalNumbers(
          parseOptionalNumber(left.total_principal),
          parseOptionalNumber(right.total_principal),
          'desc',
        )
      }

      if (sortMode === 'count_desc') {
        return compareOptionalNumbers(
          left.holding_count ?? null,
          right.holding_count ?? null,
          'desc',
        )
      }

      if (sortMode === 'recent_desc') {
        return compareOptionalNumbers(
          aggregateLatestUpdatedAt(holdingsByFundID[left.fund_id] ?? []),
          aggregateLatestUpdatedAt(holdingsByFundID[right.fund_id] ?? []),
          'desc',
        )
      }

      if (sortMode === 'change_asc' || sortMode === 'change_desc') {
        const leftChange = aggregateChangeValue(
          left,
          aggregateMetrics[left.fund_id],
          metricScope,
        )
        const rightChange = aggregateChangeValue(
          right,
          aggregateMetrics[right.fund_id],
          metricScope,
        )
        return compareOptionalNumbers(
          leftChange,
          rightChange,
          sortMode === 'change_asc' ? 'asc' : 'desc',
        )
      }

      const leftProfit = aggregateProfitValue(
        left,
        aggregateMetrics[left.fund_id],
        metricScope,
      )
      const rightProfit = aggregateProfitValue(
        right,
        aggregateMetrics[right.fund_id],
        metricScope,
      )
      return compareOptionalNumbers(
        leftProfit,
        rightProfit,
        sortMode === 'profit_asc' ? 'asc' : 'desc',
      )
    })
  }, [
    aggregateMetrics,
    analysesByFundID,
    filteredHoldingAggregates,
    holdingsByFundID,
    metricScope,
    sortMode,
  ])
  const sortedHoldings = useMemo(() => {
    if (sortMode === 'default') {
      return filteredHoldings
    }

    return filteredHoldings.slice().sort((left, right) => {
      const leftAnalysis = analysesByFundID[left.fund_id]
      const rightAnalysis = analysesByFundID[right.fund_id]

      if (sortMode === 'analysis_recommendation') {
        return (
          analysisRecommendationWeight(rightAnalysis) -
          analysisRecommendationWeight(leftAnalysis)
        )
      }

      if (sortMode === 'analysis_risk') {
        return (
          analysisRiskWeight(rightAnalysis) - analysisRiskWeight(leftAnalysis)
        )
      }

      if (sortMode === 'principal_desc') {
        return compareOptionalNumbers(
          parseOptionalNumber(left.amount),
          parseOptionalNumber(right.amount),
          'desc',
        )
      }

      if (sortMode === 'count_desc') {
        return compareOptionalNumbers(
          holdingsByFundID[left.fund_id]?.length ?? null,
          holdingsByFundID[right.fund_id]?.length ?? null,
          'desc',
        )
      }

      if (sortMode === 'recent_desc') {
        return compareOptionalNumbers(
          timestampValue(left.updated_at || left.created_at),
          timestampValue(right.updated_at || right.created_at),
          'desc',
        )
      }

      if (sortMode === 'change_asc' || sortMode === 'change_desc') {
        const leftChange = detailChangeValue(
          left,
          estimatesByFundID[left.fund_id],
          metricScope,
        )
        const rightChange = detailChangeValue(
          right,
          estimatesByFundID[right.fund_id],
          metricScope,
        )
        return compareOptionalNumbers(
          leftChange,
          rightChange,
          sortMode === 'change_asc' ? 'asc' : 'desc',
        )
      }

      const leftProfit = detailProfitValue(
        left,
        estimatesByFundID[left.fund_id],
        metricScope,
      )
      const rightProfit = detailProfitValue(
        right,
        estimatesByFundID[right.fund_id],
        metricScope,
      )
      return compareOptionalNumbers(
        leftProfit,
        rightProfit,
        sortMode === 'profit_asc' ? 'asc' : 'desc',
      )
    })
  }, [
    analysesByFundID,
    estimatesByFundID,
    filteredHoldings,
    holdingsByFundID,
    metricScope,
    sortMode,
  ])
  const shouldUseOfficialSummary =
    metricScope === 'official' && hasOfficialSummaryMetrics
  const activeDisplayCount =
    viewMode === 'aggregate'
      ? sortedHoldingAggregates.length
      : sortedHoldings.length
  const canLoadMoreTransactions =
    holdingTransactions.length >= transactionVisibleLimit &&
    transactionVisibleLimit < 50
  const hasActiveFilter = showIncompleteOnly || filterMode !== 'all'
  const activeFilterDescription =
    viewMode === 'aggregate'
      ? `当前筛选后显示 ${activeDisplayCount}/${holdingAggregates.length} 只基金。`
      : `当前筛选后显示 ${activeDisplayCount}/${holdings.length} 条记录。`
  const incompleteFilterDescription =
    viewMode === 'aggregate'
      ? `当前仅显示部分就绪或完全未就绪的基金，共 ${activeDisplayCount}/${holdingAggregates.length} 只。`
      : `当前仅显示待确认份额、待同步官方净值或真实口径未就绪的记录，共 ${activeDisplayCount}/${holdings.length} 条。`
  const activeWorkspaceTab = holdings.length === 0 ? 'record' : workspaceTab
  const quickActions = [
    { label: '记一笔', tab: 'record' as const, primary: true },
    { label: '看持仓', tab: 'list' as const },
    { label: '查风险', tab: 'risk' as const },
    { label: '看流水', tab: 'ledger' as const },
  ]

  const handleSeedDemo = async () => {
    setFeedback(null)
    setIsSeedingDemo(true)

    try {
      await seedDemoData()
      setFeedback({
        type: 'success',
        message:
          '已为你填入一组参考持仓；如果还没有自选分组，也会一并创建默认分组。',
      })
    } catch (error) {
      setFeedback({
        type: 'error',
        message:
          error instanceof Error
            ? error.message
            : '初始化参考持仓失败，请稍后重试。',
      })
    } finally {
      setIsSeedingDemo(false)
    }
  }

  const handleAddHolding = async () => {
    setFeedback(null)

    if (!resolvedFundID) {
      setFeedback({
        type: 'error',
        message: '请先从搜索结果中选择基金，或输入能唯一匹配的基金代码/名称。',
      })
      return
    }

    const normalizedAmount = amount.replace(/,/g, '').trim()
    const parsedAmount = Number.parseFloat(normalizedAmount)
    if (
      !normalizedAmount ||
      !Number.isFinite(parsedAmount) ||
      parsedAmount <= 0
    ) {
      setFeedback({
        type: 'error',
        message: '请输入有效的持仓金额。',
      })
      return
    }

    if (!tradeDate.trim()) {
      setFeedback({
        type: 'error',
        message: '请选择交易日期。',
      })
      return
    }

    setIsAddingHolding(true)

    try {
      await addHolding(resolvedFundID, normalizedAmount, tradeAtPayload, note)
      setSelectedFundID('')
      setSelectedFundName('')
      setQuery('')
      setAmount('')
      setTradeDate(marketStatus.currentDate || tradeDate)
      setTradeTiming(
        marketStatus.currentTime.getTime() === 0
          ? 'before_close'
          : resolveTradeTimingFromServerClock(marketStatus.currentTime),
      )
      setNote('')
      setFeedback({
        type: 'success',
        message: pricingDatePreview
          ? `已加入 ${resolvedFundName || resolvedFundID} 的持仓记录，将按 ${pricingDatePreview} 收盘净值确认。`
          : `已加入 ${resolvedFundName || resolvedFundID} 的持仓记录，确认净值日已按交易日历计算。`,
      })
    } catch (error) {
      setFeedback({
        type: 'error',
        message:
          error instanceof Error ? error.message : '加入持仓失败，请稍后重试。',
      })
    } finally {
      setIsAddingHolding(false)
    }
  }

  const handleCreatePortfolioVIPTask = async () => {
    setVipFeedback(null)

    if (!membership.isVip) {
      router.push('/vip')
      return
    }

    if (holdings.length === 0) {
      setVipFeedback({
        type: 'error',
        message: '当前还没有持仓记录，先录入持仓后再发起组合分析。',
      })
      return
    }

    let created
    try {
      created = await createTask({
        type: 'portfolio_analysis',
        targetType: 'holdings_all',
        targetId: 'all-holdings',
        targetName: '全部持仓组合',
      })
    } catch (error) {
      setVipFeedback({
        type: 'error',
        message:
          error instanceof Error
            ? error.message
            : '创建 VIP 任务失败，请稍后重试。',
      })
      return
    }

    if (!created.ok) {
      setVipFeedback({
        type: 'error',
        message:
          created.reason === 'quota_exhausted'
            ? '今日组合分析额度已用完，请明天再试。'
            : '当前账号尚未开通 VIP。',
      })
      if (created.reason === 'not_vip') {
        router.push('/vip')
      }
      return
    }

    setVipFeedback({
      type: 'success',
      message: '组合分析任务已创建，正在跳转到任务中心。',
    })
    router.push(`/vip/tasks?focus=${created.taskId}`)
  }

  if (isLoading) {
    return (
      <AccountAreaShell
        title="持仓明细"
        description="记录持仓本金，并在官方净值同步后查看真实市值与今日盈亏。"
      >
        <div className="glass h-64 rounded-[32px] border border-[var(--card-border)]" />
      </AccountAreaShell>
    )
  }

  if (!user) {
    return (
      <AccountAreaShell
        title="持仓明细"
        description="记录持仓本金，并在官方净值同步后查看真实市值与今日盈亏。"
      >
        <div className="glass rounded-[32px] border border-[var(--card-border)] p-8 text-center">
          <div className="mb-3 text-2xl font-bold text-theme-primary">
            登录后可查看持仓明细
          </div>
          <p className="mx-auto max-w-xl text-sm leading-6 text-theme-secondary">
            登录后可同步查看和管理你的基金持仓记录。
          </p>
          <div className="mt-6 flex justify-center gap-3">
            <Link
              href="/auth/login"
              className="rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-sm text-theme-secondary"
            >
              去登录
            </Link>
            <Link
              href="/auth/register"
              className="rounded-2xl bg-gradient-to-r from-cyan-500 via-sky-500 to-blue-600 px-4 py-3 text-sm font-medium text-white"
            >
              去注册
            </Link>
          </div>
        </div>
      </AccountAreaShell>
    )
  }

  return (
    <AccountAreaShell
      title="我的持仓账本"
      description="快速记录仓位，查看净值确认、真实盈亏、风险提醒和组合分析。"
    >
      <ScrollRevealStack className="space-y-8">
        {feedback && <HoldingFeedbackBanner feedback={feedback} />}
        {vipFeedback && (
          <HoldingFeedbackBanner feedback={vipFeedback} showIcon={false} />
        )}

        <section className="rounded-[32px] border border-[var(--card-border)] p-5 glass md:p-6">
          <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <div className="text-sm text-theme-muted">持仓工作台</div>
              <div className="mt-1 text-3xl font-black text-theme-primary">
                {holdings.length} 条持仓
              </div>
              <div className="mt-2 text-xs text-theme-muted">
                {holdings.length === 0
                  ? '先记录一笔持仓，再查看净值、份额和盈亏。'
                  : `${metricScope === 'official' ? '官方口径' : '盘中预估'} · ${viewMode === 'aggregate' ? '按基金' : '分笔明细'}`}
              </div>
            </div>

            <div className="flex flex-wrap gap-2">
              {quickActions.map((action) => (
                <button
                  key={action.tab}
                  type="button"
                  onClick={() => setWorkspaceTab(action.tab)}
                  className={cn(
                    'rounded-2xl border px-4 py-2 text-sm font-medium transition-all duration-200',
                    activeWorkspaceTab === action.tab
                      ? 'border-cyan-300/45 bg-cyan-400/14 text-cyan-100 shadow-[0_12px_26px_rgba(34,211,238,0.12)]'
                      : action.primary
                        ? 'border-cyan-300/30 bg-cyan-400/10 text-cyan-100 hover:border-cyan-200/45'
                        : 'border-[var(--input-border)] bg-[var(--input-bg)] text-theme-secondary hover:border-cyan-300/35 hover:text-theme-primary',
                  )}
                >
                  {action.label}
                </button>
              ))}
              {holdings.length === 0 && (
                <button
                  type="button"
                  onClick={() => void handleSeedDemo()}
                  disabled={isSeedingDemo}
                  className={cn(
                    'group relative inline-flex items-center gap-2 overflow-hidden rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-2 text-sm text-theme-secondary transition-all duration-200',
                    'hover:-translate-y-0.5 hover:border-cyan-400/45 hover:text-theme-primary',
                    isSeedingDemo && 'holding-action-button',
                  )}
                >
                  <span className="holding-action-shine" />
                  {isSeedingDemo ? (
                    <LoaderCircle className="relative z-10 h-4 w-4 animate-spin" />
                  ) : (
                    <BarChart4 className="relative z-10 h-4 w-4" />
                  )}
                  <span className="relative z-10">
                    {isSeedingDemo ? '准备中...' : '快速开始'}
                  </span>
                </button>
              )}
            </div>
          </div>

          {holdings.length > 0 && (
            <div className="mt-5 grid gap-3 md:grid-cols-3 xl:grid-cols-6">
              {workspaceTabs.map((tab) => {
                const Icon = tab.icon
                return (
                  <button
                    key={tab.id}
                    type="button"
                    onClick={() => setWorkspaceTab(tab.id)}
                    className={cn(
                      'rounded-[20px] border p-3 text-left transition-all duration-200',
                      activeWorkspaceTab === tab.id
                        ? 'border-cyan-300/45 bg-cyan-400/14 text-cyan-100 shadow-[0_14px_30px_rgba(34,211,238,0.12)]'
                        : 'border-[var(--card-border)] bg-[var(--card-bg)]/56 text-theme-secondary hover:border-cyan-300/30 hover:text-theme-primary',
                    )}
                  >
                    <div className="flex items-center gap-2 text-sm font-semibold">
                      <Icon className="h-4 w-4" />
                      {tab.label}
                    </div>
                    <div className="mt-1 text-[11px] text-theme-muted">
                      {tab.description}
                    </div>
                  </button>
                )
              })}
            </div>
          )}
        </section>

        {activeWorkspaceTab === 'record' && (
          <HoldingRecordComposer
            compact={holdings.length > 0}
            holdingsCount={holdings.length}
            totalPrincipalText={totalPrincipalText}
            query={query}
            results={results}
            recentAggregates={recentHoldingAggregates}
            resolvedFundID={resolvedFundID}
            resolvedFundName={resolvedFundName}
            amount={amount}
            note={note}
            tradeDate={tradeDate}
            tradeTiming={tradeTiming}
            tradeDateLabel={tradeDateLabel}
            tradeTimingLabel={tradeTimingLabel}
            todayTradeDate={todayTradeDate}
            previousTradeDate={previousTradeDate}
            nextTradeDate={nextTradeDate}
            pricingDatePreview={pricingDatePreview}
            pricingRuleLabel={pricingRuleLabel}
            isAddingHolding={isAddingHolding}
            onQueryChange={(value) => {
              setQuery(value)
              setSelectedFundID('')
              setSelectedFundName('')
            }}
            onSelectFund={(fund) => {
              setSelectedFundID(fund.id)
              setSelectedFundName(fund.name)
              setQuery(fund.name)
              setFeedback(null)
            }}
            onSelectAggregate={(aggregate) => {
              setSelectedFundID(aggregate.fund_id)
              setSelectedFundName(aggregate.fund?.name || '')
              setQuery(aggregate.fund?.name || aggregate.fund_id)
              setFeedback(null)
            }}
            onAmountChange={setAmount}
            onNoteChange={setNote}
            onTradeDateChange={setTradeDate}
            onTradeTimingChange={setTradeTiming}
            onAddHolding={() => void handleAddHolding()}
          />
        )}

        {holdings.length === 0 && activeWorkspaceTab === 'record' && (
          <div className="rounded-[32px] border border-dashed border-[var(--card-border)] p-8 text-center glass">
            <Wallet className="mx-auto h-10 w-10 text-theme-muted" />
            <div className="mt-4 text-xl font-semibold text-theme-primary">
              记录第一笔后，这里会变成你的持仓驾驶舱
            </div>
            <div className="mt-6 grid gap-3 md:grid-cols-3">
              {[
                { title: '自动补齐', description: '净值、份额、市值' },
                { title: '看见风险', description: '量化建议、事件标签' },
                { title: '形成组合', description: '组合分析入口' },
              ].map((item) => (
                <div
                  key={item.title}
                  className="rounded-[22px] border border-[var(--card-border)] bg-[var(--input-bg)]/66 px-4 py-4 text-left"
                >
                  <div className="text-sm font-semibold text-theme-primary">
                    {item.title}
                  </div>
                  <div className="mt-2 text-xs leading-5 text-theme-secondary">
                    {item.description}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {holdings.length > 0 && activeWorkspaceTab === 'summary' && (
          <>
            <HoldingsSummaryMetrics
              holdingSummary={holdingSummary}
              previewSummary={previewSummary}
              metricScope={metricScope}
              shouldUseOfficialSummary={shouldUseOfficialSummary}
              hasOfficialSummaryMetrics={hasOfficialSummaryMetrics}
              hasPreviewSummaryMetrics={hasPreviewSummaryMetrics}
              officialReadyPrincipalText={officialReadyPrincipalText}
              previewReadyPrincipalText={previewReadyPrincipalText}
              totalPrincipalText={totalPrincipalText}
              officialSummaryCoverage={officialSummaryCoverage}
            />
            <HoldingReconciliationPanel
              compact
              holdings={holdings}
              transactions={holdingTransactions}
            />
          </>
        )}

        {holdings.length > 0 && activeWorkspaceTab === 'list' && (
          <>
            <HoldingsViewControls
              viewMode={viewMode}
              metricScope={metricScope}
              sortMode={sortMode}
              filterMode={filterMode}
              showIncompleteOnly={showIncompleteOnly}
              incompleteCount={
                viewMode === 'aggregate'
                  ? incompleteAggregateCount
                  : incompleteHoldingCount
              }
              onViewModeChange={setViewMode}
              onMetricScopeChange={setMetricScope}
              onSortModeChange={setSortMode}
              onFilterModeChange={setFilterMode}
              onToggleIncompleteOnly={() =>
                setShowIncompleteOnly((value) => !value)
              }
            />

            {hasActiveFilter && (
              <div className="mb-6 flex flex-col gap-2 rounded-[22px] border border-amber-400/20 bg-amber-400/10 px-4 py-3 text-sm text-amber-100 sm:flex-row sm:items-center sm:justify-between">
                <div className="flex items-start gap-2">
                  <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
                  <span>
                    {showIncompleteOnly
                      ? incompleteFilterDescription
                      : activeFilterDescription}
                  </span>
                </div>
                <button
                  type="button"
                  onClick={() => {
                    setShowIncompleteOnly(false)
                    setFilterMode('all')
                  }}
                  className="text-left text-xs font-medium text-amber-50 underline-offset-4 hover:underline sm:text-right"
                >
                  取消筛选
                </button>
              </div>
            )}

            {hasActiveFilter && activeDisplayCount === 0 ? (
              <div className="rounded-[32px] border border-dashed border-[var(--card-border)] p-10 text-center glass">
                <CheckCircle2 className="mx-auto h-10 w-10 text-emerald-300" />
                <div className="mt-4 text-xl font-semibold text-theme-primary">
                  当前没有匹配记录
                </div>
                <p className="mt-2 text-sm leading-6 text-theme-secondary">
                  {showIncompleteOnly
                    ? '当前口径下没有待补齐记录。'
                    : '取消筛选后可查看全部持仓。'}
                </p>
                <button
                  type="button"
                  onClick={() => {
                    setShowIncompleteOnly(false)
                    setFilterMode('all')
                  }}
                  className="mt-5 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-2 text-sm text-theme-secondary transition-colors hover:border-cyan-400/35 hover:text-theme-primary"
                >
                  查看全部持仓
                </button>
              </div>
            ) : (
              <HoldingsList
                viewMode={viewMode}
                metricScope={metricScope}
                sortedHoldingAggregates={sortedHoldingAggregates}
                sortedHoldings={sortedHoldings}
                holdingsByFundID={holdingsByFundID}
                aggregateMetrics={aggregateMetrics}
                estimatesByFundID={estimatesByFundID}
                analysesByFundID={analysesByFundID}
                showIncompleteOnly={showIncompleteOnly}
                onRemoveHolding={removeHolding}
                onUpdateHolding={updateHolding}
                onSellHolding={sellHolding}
                onRecordHoldingDividend={recordHoldingDividend}
                onAdjustHoldingShares={adjustHoldingShares}
              />
            )}
          </>
        )}

        {holdings.length > 0 && activeWorkspaceTab === 'risk' && (
          <>
            <HoldingPortfolioHealthPanel
              compact
              holdings={holdings}
              aggregates={holdingAggregates}
              analysesByFundID={analysesByFundID}
              aggregateMetrics={aggregateMetrics}
              metricScope={metricScope}
              exposureSnapshots={exposureSnapshotsByFundID}
              topHoldingsByFundID={topHoldingsByFundID}
            />
            <HoldingReminderPanel
              compact
              holdings={holdings}
              aggregates={holdingAggregates}
              analysesByFundID={analysesByFundID}
              aggregateMetrics={aggregateMetrics}
              metricScope={metricScope}
              exposureSnapshots={exposureSnapshotsByFundID}
            />
          </>
        )}

        {holdings.length > 0 && activeWorkspaceTab === 'ledger' && (
          <HoldingActivityTimeline
            compact
            transactions={holdingTransactions}
            onVoidTransaction={voidHoldingTransaction}
            fundOptions={transactionFundOptions}
            fundFilter={transactionFundFilter}
            typeFilter={transactionTypeFilter}
            statusFilter={transactionStatusFilter}
            sourceFilter={transactionSourceFilter}
            keywordFilter={transactionKeywordFilter}
            startDateFilter={transactionStartDateFilter}
            endDateFilter={transactionEndDateFilter}
            visibleLimit={transactionVisibleLimit}
            canLoadMore={canLoadMoreTransactions}
            onFundFilterChange={setTransactionFundFilter}
            onTypeFilterChange={setTransactionTypeFilter}
            onStatusFilterChange={setTransactionStatusFilter}
            onSourceFilterChange={setTransactionSourceFilter}
            onKeywordFilterChange={setTransactionKeywordFilter}
            onStartDateFilterChange={setTransactionStartDateFilter}
            onEndDateFilterChange={setTransactionEndDateFilter}
            onPreviewRollback={previewHoldingTransactionRollback}
            onApplyRollback={applyHoldingTransactionRollback}
            onLoadMore={() => setTransactionPage((page) => page + 1)}
            onClearFilters={() => {
              setTransactionFundFilter('all')
              setTransactionTypeFilter('all')
              setTransactionStatusFilter('all')
              setTransactionSourceFilter('all')
              setTransactionKeywordFilter('')
              setTransactionStartDateFilter('')
              setTransactionEndDateFilter('')
              setTransactionPage(1)
            }}
          />
        )}

        {holdings.length > 0 && activeWorkspaceTab === 'tools' && (
          <>
            <HoldingImportPanel
              compact
              recentAggregates={recentHoldingAggregates}
              onImportBatch={async (items) => {
                setFeedback(null)
                try {
                  const result = await addHoldingsBatch(items)
                  setFeedback({
                    type: 'success',
                    message: result
                      ? `已导入 ${result.created_count}/${result.total} 行${result.failed_count ? `，${result.failed_count} 行需修正。` : '。'}`
                      : '导入完成。',
                  })
                  return result
                } catch (error) {
                  setFeedback({
                    type: 'error',
                    message:
                      error instanceof Error
                        ? error.message
                        : '批量导入失败，请检查数据后重试。',
                  })
                  throw error
                }
              }}
              onSelectDraft={(draft) => {
                const matchedAggregate = holdingAggregates.find(
                  (aggregate) => aggregate.fund_id === draft.fundID,
                )
                setSelectedFundID(draft.fundID)
                setSelectedFundName(matchedAggregate?.fund?.name || '')
                setQuery(matchedAggregate?.fund?.name || draft.fundID)
                setAmount(draft.amount)
                setNote(draft.note)
                setWorkspaceTab('record')
                setFeedback({
                  type: 'success',
                  message:
                    '已把导入预览行带入记录入口，请核对交易日期和金额后再提交。',
                })
              }}
            />
            {canAccessVIP && (
              <VIPAnalysisEntry
                title="VIP 持仓组合分析"
                description="围绕全部持仓输出组合报告。"
                accent="amber"
                badgeLabel={membership.isVip ? '已开通 VIP' : 'VIP'}
                quotaLabel={
                  membership.isVip
                    ? `今日剩余：组合 ${remainingQuota.portfolioAnalysis} 次 · 板块 ${remainingQuota.sectorAnalysis} 次`
                    : '开通后可用：2 次板块分析 · 2 次组合分析'
                }
                note="默认分析对象：全部持仓组合"
                actions={
                  membership.isVip
                    ? [
                        {
                          label: '发起组合分析',
                          onClick: () => void handleCreatePortfolioVIPTask(),
                          variant: 'primary',
                        },
                        {
                          label: latestCompletedTask?.reportId
                            ? '查看最近报告'
                            : '查看报告样例',
                          href: latestCompletedTask?.reportId
                            ? `/vip/reports/${latestCompletedTask.reportId}`
                            : `/vip/reports/${VIP_SAMPLE_REPORT_IDS.defaultPortfolio}`,
                          variant: 'secondary',
                          icon: <FileStack className="h-4 w-4" />,
                        },
                      ]
                    : [
                        {
                          label: '开通 VIP',
                          href: '/vip',
                          variant: 'primary',
                        },
                        {
                          label: '查看报告样例',
                          href: `/vip/reports/${VIP_SAMPLE_REPORT_IDS.defaultPortfolio}`,
                          variant: 'secondary',
                          icon: <FileStack className="h-4 w-4" />,
                        },
                      ]
                }
              />
            )}
          </>
        )}
      </ScrollRevealStack>
    </AccountAreaShell>
  )
}
