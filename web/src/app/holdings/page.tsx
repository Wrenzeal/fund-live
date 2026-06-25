"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { Wallet } from "lucide-react";
import { AccountAreaShell } from "@/components/account-area-shell";
import { ActionButton } from "@/components/ui/action-button";
import {
  HoldingFeedbackBanner,
  type HoldingFeedbackMessage,
} from "@/components/holding-feedback-banner";
import { ScrollRevealStack } from "@/components/scroll-reveal";
import { HoldingActivityTimeline } from "@/components/holding-activity-timeline";
import { HoldingImportPanel } from "@/components/holding-import-panel";
import { HoldingPortfolioHealthPanel } from "@/components/holding-portfolio-health-panel";
import { HoldingReconciliationPanel } from "@/components/holding-reconciliation-panel";
import { HoldingRecordComposer } from "@/components/holding-record-composer";
import { HoldingReminderPanel } from "@/components/holding-reminder-panel";
import {
  HoldingsWorkspaceNav,
  type HoldingWorkspaceTab,
} from "@/components/holdings-workspace-nav";
import { HoldingsViewControls } from "@/components/holdings-view-controls";
import { HoldingsList } from "@/components/holdings-list";
import { HoldingsSummaryMetrics } from "@/components/holdings-summary-metrics";
import { EmptyState } from "@/components/ui/empty-state";
import { StatusBanner } from "@/components/ui/status-banner";
import { useCurrentUser } from "@/hooks/use-auth";
import {
  useFundAnalyses,
  useFundExposureSnapshots,
  useFundSearch,
  useFundTopHoldings,
} from "@/hooks/use-fund-data";
import {
  useMarketStatus,
  usePricingDatePreview,
} from "@/hooks/use-market-status";
import {
  useHoldingEstimateMetrics,
  useUserPortfolio,
  type HoldingTransactionStatusFilter,
  type HoldingTransactionType,
} from "@/hooks/use-user-portfolio";
import type { HoldingSourceFilter } from "@/lib/holding-sources";
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
} from "@/lib/holding-display";

type HoldingViewMode = "aggregate" | "detail";
export default function HoldingsPage() {
  const { user, isLoading } = useCurrentUser();
  const marketStatus = useMarketStatus();
  const [transactionFundFilter, setTransactionFundFilter] = useState("all");
  const [transactionTypeFilter, setTransactionTypeFilter] = useState<
    HoldingTransactionType | "all"
  >("all");
  const [transactionStatusFilter, setTransactionStatusFilter] =
    useState<HoldingTransactionStatusFilter>("all");
  const [transactionSourceFilter, setTransactionSourceFilter] =
    useState<HoldingSourceFilter>("all");
  const [transactionKeywordFilter, setTransactionKeywordFilter] = useState("");
  const [transactionStartDateFilter, setTransactionStartDateFilter] =
    useState("");
  const [transactionEndDateFilter, setTransactionEndDateFilter] = useState("");
  const [transactionPage, setTransactionPage] = useState(1);
  const transactionPageSize = 10;
  const transactionVisibleLimit = transactionPage * transactionPageSize;
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
  });
  const [query, setQuery] = useState("");
  const [selectedFundID, setSelectedFundID] = useState("");
  const [selectedFundName, setSelectedFundName] = useState("");
  const [amount, setAmount] = useState("");
  const [tradeDate, setTradeDate] = useState("");
  const [tradeTiming, setTradeTiming] = useState<TradeTiming>("before_close");
  const [viewMode, setViewMode] = useState<HoldingViewMode>("aggregate");
  const [workspaceTab, setWorkspaceTab] =
    useState<HoldingWorkspaceTab>("summary");
  const [metricScope, setMetricScope] =
    useState<HoldingMetricScope>("official");
  const [sortMode, setSortMode] = useState<HoldingSortMode>("default");
  const [filterMode, setFilterMode] = useState<HoldingFilterMode>("all");
  const [showIncompleteOnly, setShowIncompleteOnly] = useState(false);
  const [note, setNote] = useState("");
  const [feedback, setFeedback] = useState<HoldingFeedbackMessage | null>(null);
  const [isSeedingDemo, setIsSeedingDemo] = useState(false);
  const [isAddingHolding, setIsAddingHolding] = useState(false);
  const defaultsInitializedRef = useRef(false);
  const { results } = useFundSearch(query);
  const fundIDsForAnalysis = useMemo(
    () =>
      Array.from(
        new Set(holdings.map((holding) => holding.fund_id).filter(Boolean)),
      ),
    [holdings],
  );
  const { analysesByFundID } = useFundAnalyses(fundIDsForAnalysis);
  const { exposureSnapshotsByFundID } =
    useFundExposureSnapshots(fundIDsForAnalysis);
  const { topHoldingsByFundID } = useFundTopHoldings(fundIDsForAnalysis);
  const normalizedQuery = query.trim();

  const autoMatchedFund = useMemo(() => {
    if (!normalizedQuery) {
      return null;
    }

    const exactMatch = results.find(
      (fund) => fund.id === normalizedQuery || fund.name === normalizedQuery,
    );
    if (exactMatch) {
      return exactMatch;
    }

    if (results.length === 1) {
      return results[0];
    }

    return null;
  }, [normalizedQuery, results]);

  useEffect(() => {
    if (
      defaultsInitializedRef.current ||
      !marketStatus.currentDate ||
      marketStatus.currentTime.getTime() === 0
    ) {
      return;
    }

    setTradeDate(marketStatus.currentDate);
    setTradeTiming(resolveTradeTimingFromServerClock(marketStatus.currentTime));
    defaultsInitializedRef.current = true;
  }, [marketStatus.currentDate, marketStatus.currentTime]);

  useEffect(() => {
    setTransactionPage(1);
  }, [
    transactionFundFilter,
    transactionTypeFilter,
    transactionStatusFilter,
    transactionSourceFilter,
    transactionKeywordFilter,
    transactionStartDateFilter,
    transactionEndDateFilter,
  ]);

  const resolvedFundID = selectedFundID || autoMatchedFund?.id || "";
  const resolvedFundName = selectedFundName || autoMatchedFund?.name || "";
  const tradeAtPayload = buildTradeAtValue(tradeDate, tradeTiming);
  const {
    preview,
    isLoading: isPricingPreviewLoading,
    error: pricingPreviewError,
  } = usePricingDatePreview(tradeAtPayload || null);
  const pricingDatePreview = preview?.pricingDate || "";
  const tradeDateLabel = formatTradeDateLabel(tradeDate);
  const tradeTimingLabel = formatTradeTimingLabel(tradeTiming);
  const todayTradeDate = marketStatus.currentDate || tradeDate;
  const previousTradeDate = marketStatus.previousTradingDay || "";
  const nextTradeDate = marketStatus.nextTradingDay || "";
  const pricingRuleLabel = !tradeDate
    ? "选择交易日期和提交时段后，会自动显示确认净值日。"
    : pricingPreviewError
      ? pricingPreviewError.message
      : isPricingPreviewLoading
        ? "正在按交易日历校验确认净值日..."
        : preview?.message || "正在按交易日历校验确认净值日...";
  const hasOfficialSummaryMetrics = holdingSummary.real_metrics_ready_count > 0;
  const officialSummaryCoverage = hasOfficialSummaryMetrics
    ? `${holdingSummary.real_metrics_ready_count}/${holdingSummary.total_holdings} 条`
    : "";
  const officialReadyPrincipalText = hasOfficialSummaryMetrics
    ? formatSummaryMoney(holdingSummary.ready_principal)
    : "--";
  const totalPrincipalText = formatSummaryMoney(holdingSummary.total_principal);
  const {
    estimatesByFundID,
    aggregateMetrics,
    summary: previewSummary,
  } = useHoldingEstimateMetrics(holdingAggregates);
  const hasPreviewSummaryMetrics = previewSummary.ready_count > 0;
  const previewReadyPrincipalText = hasPreviewSummaryMetrics
    ? formatSummaryMoney(previewSummary.ready_principal)
    : "--";
  const holdingsByFundID = useMemo(() => {
    return holdings.reduce<Record<string, typeof holdings>>(
      (groups, holding) => {
        if (!groups[holding.fund_id]) {
          groups[holding.fund_id] = [];
        }
        groups[holding.fund_id].push(holding);
        return groups;
      },
      {},
    );
  }, [holdings]);
  const transactionFundOptions = useMemo(() => {
    const options = holdingAggregates.map((aggregate) => ({
      fund_id: aggregate.fund_id,
      name: aggregate.fund?.name || aggregate.fund_id,
    }));
    return options.sort((left, right) =>
      left.name.localeCompare(right.name, "zh-Hans-CN"),
    );
  }, [holdingAggregates]);
  const recentHoldingAggregates = useMemo(() => {
    return holdingAggregates
      .slice()
      .sort((left, right) =>
        compareOptionalNumbers(
          aggregateLatestUpdatedAt(holdingsByFundID[left.fund_id] ?? []),
          aggregateLatestUpdatedAt(holdingsByFundID[right.fund_id] ?? []),
          "desc",
        ),
      );
  }, [holdingAggregates, holdingsByFundID]);
  const incompleteHoldingCount = useMemo(
    () => holdings.filter(isHoldingIncomplete).length,
    [holdings],
  );
  const incompleteAggregateCount = useMemo(
    () => holdingAggregates.filter(isAggregateIncomplete).length,
    [holdingAggregates],
  );
  const filteredHoldingAggregates = useMemo(
    () =>
      holdingAggregates.filter((aggregate) => {
        if (showIncompleteOnly && !isAggregateIncomplete(aggregate)) {
          return false;
        }
        return matchesAggregateFilter(
          aggregate,
          aggregateMetrics[aggregate.fund_id],
          metricScope,
          filterMode,
        );
      }),
    [
      aggregateMetrics,
      filterMode,
      holdingAggregates,
      metricScope,
      showIncompleteOnly,
    ],
  );
  const filteredHoldings = useMemo(
    () =>
      holdings.filter((holding) => {
        if (showIncompleteOnly && !isHoldingIncomplete(holding)) {
          return false;
        }
        return matchesHoldingFilter(
          holding,
          estimatesByFundID[holding.fund_id],
          metricScope,
          filterMode,
          holdingsByFundID[holding.fund_id]?.length ?? 1,
        );
      }),
    [
      estimatesByFundID,
      filterMode,
      holdings,
      holdingsByFundID,
      metricScope,
      showIncompleteOnly,
    ],
  );
  const sortedHoldingAggregates = useMemo(() => {
    if (sortMode === "default") {
      return filteredHoldingAggregates;
    }

    return filteredHoldingAggregates.slice().sort((left, right) => {
      const leftAnalysis = analysesByFundID[left.fund_id];
      const rightAnalysis = analysesByFundID[right.fund_id];

      if (sortMode === "analysis_recommendation") {
        return (
          analysisRecommendationWeight(rightAnalysis) -
          analysisRecommendationWeight(leftAnalysis)
        );
      }

      if (sortMode === "analysis_risk") {
        return (
          analysisRiskWeight(rightAnalysis) - analysisRiskWeight(leftAnalysis)
        );
      }

      if (sortMode === "principal_desc") {
        return compareOptionalNumbers(
          parseOptionalNumber(left.total_principal),
          parseOptionalNumber(right.total_principal),
          "desc",
        );
      }

      if (sortMode === "count_desc") {
        return compareOptionalNumbers(
          left.holding_count ?? null,
          right.holding_count ?? null,
          "desc",
        );
      }

      if (sortMode === "recent_desc") {
        return compareOptionalNumbers(
          aggregateLatestUpdatedAt(holdingsByFundID[left.fund_id] ?? []),
          aggregateLatestUpdatedAt(holdingsByFundID[right.fund_id] ?? []),
          "desc",
        );
      }

      if (sortMode === "change_asc" || sortMode === "change_desc") {
        const leftChange = aggregateChangeValue(
          left,
          aggregateMetrics[left.fund_id],
          metricScope,
        );
        const rightChange = aggregateChangeValue(
          right,
          aggregateMetrics[right.fund_id],
          metricScope,
        );
        return compareOptionalNumbers(
          leftChange,
          rightChange,
          sortMode === "change_asc" ? "asc" : "desc",
        );
      }

      const leftProfit = aggregateProfitValue(
        left,
        aggregateMetrics[left.fund_id],
        metricScope,
      );
      const rightProfit = aggregateProfitValue(
        right,
        aggregateMetrics[right.fund_id],
        metricScope,
      );
      return compareOptionalNumbers(
        leftProfit,
        rightProfit,
        sortMode === "profit_asc" ? "asc" : "desc",
      );
    });
  }, [
    aggregateMetrics,
    analysesByFundID,
    filteredHoldingAggregates,
    holdingsByFundID,
    metricScope,
    sortMode,
  ]);
  const sortedHoldings = useMemo(() => {
    if (sortMode === "default") {
      return filteredHoldings;
    }

    return filteredHoldings.slice().sort((left, right) => {
      const leftAnalysis = analysesByFundID[left.fund_id];
      const rightAnalysis = analysesByFundID[right.fund_id];

      if (sortMode === "analysis_recommendation") {
        return (
          analysisRecommendationWeight(rightAnalysis) -
          analysisRecommendationWeight(leftAnalysis)
        );
      }

      if (sortMode === "analysis_risk") {
        return (
          analysisRiskWeight(rightAnalysis) - analysisRiskWeight(leftAnalysis)
        );
      }

      if (sortMode === "principal_desc") {
        return compareOptionalNumbers(
          parseOptionalNumber(left.amount),
          parseOptionalNumber(right.amount),
          "desc",
        );
      }

      if (sortMode === "count_desc") {
        return compareOptionalNumbers(
          holdingsByFundID[left.fund_id]?.length ?? null,
          holdingsByFundID[right.fund_id]?.length ?? null,
          "desc",
        );
      }

      if (sortMode === "recent_desc") {
        return compareOptionalNumbers(
          timestampValue(left.updated_at || left.created_at),
          timestampValue(right.updated_at || right.created_at),
          "desc",
        );
      }

      if (sortMode === "change_asc" || sortMode === "change_desc") {
        const leftChange = detailChangeValue(
          left,
          estimatesByFundID[left.fund_id],
          metricScope,
        );
        const rightChange = detailChangeValue(
          right,
          estimatesByFundID[right.fund_id],
          metricScope,
        );
        return compareOptionalNumbers(
          leftChange,
          rightChange,
          sortMode === "change_asc" ? "asc" : "desc",
        );
      }

      const leftProfit = detailProfitValue(
        left,
        estimatesByFundID[left.fund_id],
        metricScope,
      );
      const rightProfit = detailProfitValue(
        right,
        estimatesByFundID[right.fund_id],
        metricScope,
      );
      return compareOptionalNumbers(
        leftProfit,
        rightProfit,
        sortMode === "profit_asc" ? "asc" : "desc",
      );
    });
  }, [
    analysesByFundID,
    estimatesByFundID,
    filteredHoldings,
    holdingsByFundID,
    metricScope,
    sortMode,
  ]);
  const shouldUseOfficialSummary =
    metricScope === "official" && hasOfficialSummaryMetrics;
  const activeDisplayCount =
    viewMode === "aggregate"
      ? sortedHoldingAggregates.length
      : sortedHoldings.length;
  const canLoadMoreTransactions =
    holdingTransactions.length >= transactionVisibleLimit &&
    transactionVisibleLimit < 50;
  const hasActiveFilter = showIncompleteOnly || filterMode !== "all";
  const activeFilterDescription =
    viewMode === "aggregate"
      ? `当前筛选后显示 ${activeDisplayCount}/${holdingAggregates.length} 只基金。`
      : `当前筛选后显示 ${activeDisplayCount}/${holdings.length} 条记录。`;
  const incompleteFilterDescription =
    viewMode === "aggregate"
      ? `当前仅显示部分就绪或完全未就绪的基金，共 ${activeDisplayCount}/${holdingAggregates.length} 只。`
      : `当前仅显示待确认份额、待同步官方净值或真实口径未就绪的记录，共 ${activeDisplayCount}/${holdings.length} 条。`;
  const activeWorkspaceTab = holdings.length === 0 ? "record" : workspaceTab;
  const handleSeedDemo = async () => {
    setFeedback(null);
    setIsSeedingDemo(true);

    try {
      await seedDemoData();
      setFeedback({
        type: "success",
        message:
          "已为你填入一组参考持仓；如果还没有自选分组，也会一并创建默认分组。",
      });
    } catch (error) {
      setFeedback({
        type: "error",
        message:
          error instanceof Error
            ? error.message
            : "初始化参考持仓失败，请稍后重试。",
      });
    } finally {
      setIsSeedingDemo(false);
    }
  };

  const handleAddHolding = async () => {
    setFeedback(null);

    if (!resolvedFundID) {
      setFeedback({
        type: "error",
        message: "请先从搜索结果中选择基金，或输入能唯一匹配的基金代码/名称。",
      });
      return;
    }

    const normalizedAmount = amount.replace(/,/g, "").trim();
    const parsedAmount = Number.parseFloat(normalizedAmount);
    if (
      !normalizedAmount ||
      !Number.isFinite(parsedAmount) ||
      parsedAmount <= 0
    ) {
      setFeedback({
        type: "error",
        message: "请输入有效的持仓金额。",
      });
      return;
    }

    if (!tradeDate.trim()) {
      setFeedback({
        type: "error",
        message: "请选择交易日期。",
      });
      return;
    }

    setIsAddingHolding(true);

    try {
      await addHolding(resolvedFundID, normalizedAmount, tradeAtPayload, note);
      setSelectedFundID("");
      setSelectedFundName("");
      setQuery("");
      setAmount("");
      setTradeDate(marketStatus.currentDate || tradeDate);
      setTradeTiming(
        marketStatus.currentTime.getTime() === 0
          ? "before_close"
          : resolveTradeTimingFromServerClock(marketStatus.currentTime),
      );
      setNote("");
      setFeedback({
        type: "success",
        message: pricingDatePreview
          ? `已加入 ${resolvedFundName || resolvedFundID} 的持仓记录，将按 ${pricingDatePreview} 收盘净值确认。`
          : `已加入 ${resolvedFundName || resolvedFundID} 的持仓记录，确认净值日已按交易日历计算。`,
      });
    } catch (error) {
      setFeedback({
        type: "error",
        message:
          error instanceof Error ? error.message : "加入持仓失败，请稍后重试。",
      });
    } finally {
      setIsAddingHolding(false);
    }
  };

  if (isLoading) {
    return (
      <AccountAreaShell
        title="持仓明细"
        description="记录持仓本金，并在官方净值同步后查看真实市值与今日盈亏。"
      >
        <div className="glass h-64 animate-pulse rounded-[32px] border border-[var(--card-border)]" />
      </AccountAreaShell>
    );
  }

  if (!user) {
    return (
      <AccountAreaShell
        title="持仓明细"
        description="记录持仓本金，并在官方净值同步后查看真实市值与今日盈亏。"
      >
        <EmptyState
          icon={<Wallet className="h-10 w-10" />}
          title="登录后可查看持仓明细"
          description="登录后可同步查看和管理你的基金持仓记录。"
          action={
            <div className="flex justify-center gap-3">
              <ActionButton href="/auth/login" variant="subtle">
                去登录
              </ActionButton>
              <ActionButton href="/auth/register" variant="primary">
                去注册
              </ActionButton>
            </div>
          }
        />
      </AccountAreaShell>
    );
  }

  return (
    <AccountAreaShell
      title="我的持仓账本"
      description="快速记录仓位，查看净值确认、真实盈亏、风险提醒和组合分析。"
    >
      <ScrollRevealStack className="space-y-8">
        {feedback && <HoldingFeedbackBanner feedback={feedback} />}

        <HoldingsWorkspaceNav
          holdingCount={holdings.length}
          activeTab={activeWorkspaceTab}
          detailText={
            holdings.length === 0
              ? "先记录一笔持仓，再查看净值、份额和盈亏。"
              : `${metricScope === "official" ? "官方口径" : "盘中预估"} · ${viewMode === "aggregate" ? "按基金" : "分笔明细"}`
          }
          isSeedingDemo={isSeedingDemo}
          onTabChange={setWorkspaceTab}
          onSeedDemo={() => void handleSeedDemo()}
        />

        {activeWorkspaceTab === "record" && (
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
              setQuery(value);
              setSelectedFundID("");
              setSelectedFundName("");
            }}
            onSelectFund={(fund) => {
              setSelectedFundID(fund.id);
              setSelectedFundName(fund.name);
              setQuery(fund.name);
              setFeedback(null);
            }}
            onSelectAggregate={(aggregate) => {
              setSelectedFundID(aggregate.fund_id);
              setSelectedFundName(aggregate.fund?.name || "");
              setQuery(aggregate.fund?.name || aggregate.fund_id);
              setFeedback(null);
            }}
            onAmountChange={setAmount}
            onNoteChange={setNote}
            onTradeDateChange={setTradeDate}
            onTradeTimingChange={setTradeTiming}
            onAddHolding={() => void handleAddHolding()}
          />
        )}

        {holdings.length === 0 && activeWorkspaceTab === "record" && (
          <EmptyState
            icon={<Wallet className="h-10 w-10" />}
            title="记录第一笔后，这里会变成你的持仓驾驶舱"
            features={[
              { title: "自动补齐", description: "净值、份额、市值" },
              { title: "看见风险", description: "量化建议、事件标签" },
              { title: "形成组合", description: "组合分析入口" },
            ]}
          />
        )}

        {holdings.length > 0 && activeWorkspaceTab === "summary" && (
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

        {holdings.length > 0 && activeWorkspaceTab === "list" && (
          <>
            <HoldingsViewControls
              viewMode={viewMode}
              metricScope={metricScope}
              sortMode={sortMode}
              filterMode={filterMode}
              showIncompleteOnly={showIncompleteOnly}
              incompleteCount={
                viewMode === "aggregate"
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
              <StatusBanner
                className="mb-6"
                tone="warning"
                action={
                  <button
                    type="button"
                    onClick={() => {
                      setShowIncompleteOnly(false);
                      setFilterMode("all");
                    }}
                    className="shrink-0 text-left text-xs font-medium text-amber-50 underline-offset-4 hover:underline sm:text-right"
                  >
                    取消筛选
                  </button>
                }
              >
                {showIncompleteOnly
                  ? incompleteFilterDescription
                  : activeFilterDescription}
              </StatusBanner>
            )}

            {hasActiveFilter && activeDisplayCount === 0 ? (
              <EmptyState
                title="当前没有匹配记录"
                description={
                  showIncompleteOnly
                    ? "当前口径下没有待补齐记录。"
                    : "取消筛选后可查看全部持仓。"
                }
                action={
                  <ActionButton
                    type="button"
                    variant="subtle"
                    onClick={() => {
                      setShowIncompleteOnly(false);
                      setFilterMode("all");
                    }}
                  >
                    查看全部持仓
                  </ActionButton>
                }
              />
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

        {holdings.length > 0 && activeWorkspaceTab === "risk" && (
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

        {holdings.length > 0 && activeWorkspaceTab === "ledger" && (
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
              setTransactionFundFilter("all");
              setTransactionTypeFilter("all");
              setTransactionStatusFilter("all");
              setTransactionSourceFilter("all");
              setTransactionKeywordFilter("");
              setTransactionStartDateFilter("");
              setTransactionEndDateFilter("");
              setTransactionPage(1);
            }}
          />
        )}

        {holdings.length > 0 && activeWorkspaceTab === "tools" && (
          <>
            <HoldingImportPanel
              compact
              recentAggregates={recentHoldingAggregates}
              onImportBatch={async (items) => {
                setFeedback(null);
                try {
                  const result = await addHoldingsBatch(items);
                  setFeedback({
                    type: "success",
                    message: result
                      ? `已导入 ${result.created_count}/${result.total} 行${result.failed_count ? `，${result.failed_count} 行需修正。` : "。"}`
                      : "导入完成。",
                  });
                  return result;
                } catch (error) {
                  setFeedback({
                    type: "error",
                    message:
                      error instanceof Error
                        ? error.message
                        : "批量导入失败，请检查数据后重试。",
                  });
                  throw error;
                }
              }}
              onSelectDraft={(draft) => {
                const matchedAggregate = holdingAggregates.find(
                  (aggregate) => aggregate.fund_id === draft.fundID,
                );
                setSelectedFundID(draft.fundID);
                setSelectedFundName(matchedAggregate?.fund?.name || "");
                setQuery(matchedAggregate?.fund?.name || draft.fundID);
                setAmount(draft.amount);
                setNote(draft.note);
                setWorkspaceTab("record");
                setFeedback({
                  type: "success",
                  message:
                    "已把导入预览行带入记录入口，请核对交易日期和金额后再提交。",
                });
              }}
            />
          </>
        )}
      </ScrollRevealStack>
    </AccountAreaShell>
  );
}
