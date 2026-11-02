"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { Wallet } from "lucide-react";
import { AccountAreaShell } from "@/components/account-area-shell";
import { ActionButton } from "@/components/ui/action-button";
import {
  HoldingFeedbackBanner,
  type HoldingFeedbackMessage,
} from "@/components/holding-feedback-banner";
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
  useFundHistoryBatch,
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
  buildTradeAtValue,
  compareOptionalNumbers,
  detailChangeValue,
  detailProfitValue,
  matchesAggregateFilter,
  matchesHoldingFilter,
  parseOptionalNumber,
  resolveAggregateMetrics,
  resolveHoldingMetrics,
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
    isHoldingsLoading,
    holdingsError,
    refreshHoldings,
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
  const [isAddingHolding, setIsAddingHolding] = useState(false);
  const defaultsInitializedRef = useRef(false);
  const { results } = useFundSearch(query);
  const heldFundIDs = useMemo(
    () =>
      Array.from(
        new Set(holdings.map((holding) => holding.fund_id).filter(Boolean)),
      ),
    [holdings],
  );
  const { historiesByFundID, isLoading: isHistoryLoading } = useFundHistoryBatch(heldFundIDs, 15);
  const { exposureSnapshotsByFundID } =
    useFundExposureSnapshots(heldFundIDs);
  const { topHoldingsByFundID } = useFundTopHoldings(heldFundIDs);
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
  const {
    estimatesByFundID,
    aggregateMetrics,
  } = useHoldingEstimateMetrics(holdingAggregates);
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
    () => holdings.filter(holding => !resolveHoldingMetrics(holding, estimatesByFundID[holding.fund_id], metricScope).ready).length,
    [holdings, estimatesByFundID, metricScope],
  );
  const incompleteAggregateCount = useMemo(
    () => holdingAggregates.filter(aggregate => { const metrics = resolveAggregateMetrics(aggregate, aggregateMetrics[aggregate.fund_id], metricScope); return !metrics.ready || metrics.partial; }).length,
    [holdingAggregates, aggregateMetrics, metricScope],
  );
  const filteredHoldingAggregates = useMemo(
    () =>
      holdingAggregates.filter((aggregate) => {
        const metrics = resolveAggregateMetrics(aggregate, aggregateMetrics[aggregate.fund_id], metricScope);
        if (showIncompleteOnly && metrics.ready && !metrics.partial) {
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
        if (showIncompleteOnly && resolveHoldingMetrics(holding, estimatesByFundID[holding.fund_id], metricScope).ready) {
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
    estimatesByFundID,
    filteredHoldings,
    holdingsByFundID,
    metricScope,
    sortMode,
  ]);
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
  const activeWorkspaceTab = workspaceTab;

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
      setWorkspaceTab("summary");
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
        description="记录本金、确认份额和盈亏。"
      >
        <div className="glass h-64 animate-pulse rounded-[32px] border border-[var(--card-border)]" />
      </AccountAreaShell>
    );
  }

  if (!user) {
    return (
      <AccountAreaShell
        title="持仓明细"
        description="登录后记录本金、确认份额和盈亏。"
      >
        <EmptyState
          icon={<Wallet className="h-10 w-10" />}
          title="登录后查看持仓"
          description="登录后即可保存和管理基金持仓。"
          action={
            <div className="flex justify-center gap-3">
              <ActionButton href="/auth/login?returnTo=%2Fholdings" variant="subtle">
                去登录
              </ActionButton>
              <ActionButton href="/auth/register?returnTo=%2Fholdings" variant="primary">
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
      title="我的持仓"
      description="看清还持有多少钱，记下每一次买入与调整。"
    >
      <div className="holdings-workspace space-y-4">
        {feedback && <HoldingFeedbackBanner feedback={feedback} />}

        <HoldingsWorkspaceNav
          holdingCount={holdings.length}
          activeTab={activeWorkspaceTab}
          onTabChange={setWorkspaceTab}
        />

        {isHoldingsLoading ? (
          <div role="status" className="rounded-2xl border border-[var(--card-border)] p-8 text-theme-secondary">正在读取持仓…</div>
        ) : holdingsError ? (
          <div role="alert" className="rounded-2xl border border-[var(--card-border)] p-6 text-theme-primary">
            <p>持仓暂时无法加载</p>
            <p className="mt-2 text-sm text-theme-secondary">{holdingsError.message}</p>
            <ActionButton className="mt-4" onClick={() => void refreshHoldings()}>重新加载</ActionButton>
          </div>
        ) : <>

        {holdings.length > 0 && activeWorkspaceTab !== "record" && activeWorkspaceTab !== "ledger" && activeWorkspaceTab !== "tools" && (
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="inline-flex rounded-xl border border-[var(--card-border)] p-1" aria-label="净值口径">
              {([['official', '官方净值'], ['estimate', '盘中预估']] as const).map(([scope, label]) => (
                <button key={scope} type="button" aria-pressed={metricScope === scope} onClick={() => setMetricScope(scope)} className={`min-h-11 rounded-lg px-4 text-sm transition-colors ${metricScope === scope ? 'bg-[var(--input-bg)] font-semibold text-theme-primary' : 'text-theme-secondary hover:text-theme-primary'}`}>{label}</button>
              ))}
            </div>
            <span className="text-xs text-theme-secondary">预估仅供参考，实际以基金公告净值为准</span>
          </div>
        )}

        {activeWorkspaceTab === "record" && (
          <HoldingRecordComposer
            query={query}
            results={results}
            recentAggregates={recentHoldingAggregates}
            resolvedFundID={resolvedFundID}
            resolvedFundName={resolvedFundName}
            amount={amount}
            note={note}
            tradeDate={tradeDate}
            tradeTiming={tradeTiming}
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

        {holdings.length === 0 && activeWorkspaceTab === "summary" && (
          <EmptyState
            icon={<Wallet className="h-10 w-10" />}
            title="从第一笔记录开始"
            description="记录买入金额与日期，确认份额后就能查看当前市值和持有盈亏。已有持仓请导入原始买入记录，再核对份额。"
            action={<div className="flex flex-wrap justify-center gap-3"><ActionButton className="holding-primary-action min-h-11" variant="primary" onClick={() => setWorkspaceTab('record')}>记录第一笔</ActionButton><ActionButton variant="subtle" onClick={() => setWorkspaceTab('tools')}>导入历史记录</ActionButton></div>}
          />
        )}

        {holdings.length > 0 && (activeWorkspaceTab === "summary" || activeWorkspaceTab === "list") && (
          <>
            <HoldingsSummaryMetrics
              holdings={holdings}
              estimatesByFundID={estimatesByFundID}
              metricScope={metricScope}
            />
            <HoldingsViewControls
              viewMode={viewMode}
              sortMode={sortMode}
              filterMode={filterMode}
              showIncompleteOnly={showIncompleteOnly}
              incompleteCount={
                viewMode === "aggregate"
                  ? incompleteAggregateCount
                  : incompleteHoldingCount
              }
              onViewModeChange={setViewMode}
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
                historiesByFundID={historiesByFundID}
                isHistoryLoading={isHistoryLoading}
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
              aggregateMetrics={aggregateMetrics}
              metricScope={metricScope}
              exposureSnapshots={exposureSnapshotsByFundID}
              topHoldingsByFundID={topHoldingsByFundID}
            />
            <HoldingReminderPanel
              compact
              holdings={holdings}
              aggregates={holdingAggregates}
              aggregateMetrics={aggregateMetrics}
              metricScope={metricScope}
            />
          </>
        )}

        {activeWorkspaceTab === "ledger" && (
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

        {activeWorkspaceTab === "tools" && (
          <>
            <p className="text-sm leading-6 text-theme-secondary">导入金额是买入成本，不是当前市值。请使用真实交易日期，导入后在基金明细中校正确认份额；当前接口不记录赎回到账金额。</p>
            {holdings.length > 0 && <HoldingReconciliationPanel compact holdings={holdings} transactions={holdingTransactions} />}
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
        </>}
      </div>
    </AccountAreaShell>
  );
}
