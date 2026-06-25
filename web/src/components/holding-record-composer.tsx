"use client";

import {
  CalendarDays,
  CheckCircle2,
  Clock3,
  History,
  LoaderCircle,
  Plus,
  Sparkles,
  Tags,
  Wallet,
} from "lucide-react";
import type { Fund } from "@/hooks/use-fund-data";
import type { HoldingAggregateEntry } from "@/hooks/use-user-portfolio";
import { cn } from "@/lib/utils";
import type { TradeTiming } from "@/lib/holding-display";

const quickAmounts = ["5,000", "10,000", "30,000", "50,000"];
const notePresets = ["长期底仓", "回调补仓", "观察仓", "定投", "止盈后留仓"];

interface HoldingRecordComposerProps {
  compact?: boolean;
  holdingsCount: number;
  totalPrincipalText: string;
  query: string;
  results: Fund[];
  recentAggregates: HoldingAggregateEntry[];
  resolvedFundID: string;
  resolvedFundName: string;
  amount: string;
  note: string;
  tradeDate: string;
  tradeTiming: TradeTiming;
  tradeDateLabel: string;
  tradeTimingLabel: string;
  todayTradeDate: string;
  previousTradeDate: string;
  nextTradeDate: string;
  pricingDatePreview: string;
  pricingRuleLabel: string;
  isAddingHolding: boolean;
  onQueryChange: (value: string) => void;
  onSelectFund: (fund: Fund) => void;
  onSelectAggregate: (aggregate: HoldingAggregateEntry) => void;
  onAmountChange: (value: string) => void;
  onNoteChange: (value: string) => void;
  onTradeDateChange: (value: string) => void;
  onTradeTimingChange: (value: TradeTiming) => void;
  onAddHolding: () => void;
}

function parseAmount(value: string) {
  const normalized = value.replace(/,/g, "").trim();
  const parsed = Number.parseFloat(normalized);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 0;
}

function formatInlineMoney(value?: string) {
  const parsed = Number.parseFloat(value || "");
  if (!Number.isFinite(parsed)) {
    return "--";
  }
  return new Intl.NumberFormat("zh-CN", {
    style: "currency",
    currency: "CNY",
    maximumFractionDigits: 0,
  }).format(parsed);
}

export function HoldingRecordComposer({
  compact = false,
  holdingsCount,
  totalPrincipalText,
  query,
  results,
  recentAggregates,
  resolvedFundID,
  resolvedFundName,
  amount,
  note,
  tradeDate,
  tradeTiming,
  tradeDateLabel,
  tradeTimingLabel,
  todayTradeDate,
  previousTradeDate,
  nextTradeDate,
  pricingDatePreview,
  pricingRuleLabel,
  isAddingHolding,
  onQueryChange,
  onSelectFund,
  onSelectAggregate,
  onAmountChange,
  onNoteChange,
  onTradeDateChange,
  onTradeTimingChange,
  onAddHolding,
}: HoldingRecordComposerProps) {
  const amountValue = parseAmount(amount);
  const hasFund = Boolean(resolvedFundID);
  const hasAmount = amountValue > 0;
  const hasTradeDate = Boolean(tradeDate.trim());
  const canSubmit = hasFund && hasAmount && hasTradeDate && !isAddingHolding;
  const visibleRecentAggregates = recentAggregates.slice(0, 5);
  const selectedFundLabel = resolvedFundName || resolvedFundID || "还没选基金";
  const remainingHint = !hasFund
    ? "先选基金"
    : !hasAmount
      ? "补金额"
      : !hasTradeDate
        ? "选交易日"
        : "可以记录";

  const steps = [
    {
      label: "选基金",
      done: hasFund,
      detail: hasFund ? selectedFundLabel : "代码或名称都可以",
    },
    {
      label: "填金额",
      done: hasAmount,
      detail: hasAmount
        ? `¥${amountValue.toLocaleString("zh-CN")}`
        : "可以先填大概本金",
    },
    {
      label: "定日期",
      done: hasTradeDate,
      detail: hasTradeDate
        ? `${tradeDate} · ${tradeTimingLabel}`
        : "影响确认净值日",
    },
  ];

  return (
    <section className="mb-6 overflow-hidden rounded-[34px] border border-cyan-400/18 bg-[var(--card-bg)]/86 p-5 shadow-[0_22px_54px_rgba(2,8,23,0.10)] glass">
      <div className="mb-6 flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div>
          <div className="inline-flex items-center gap-2 rounded-full border border-cyan-400/25 bg-cyan-400/10 px-3 py-1 text-[11px] font-medium tracking-[0.22em] text-cyan-200">
            <Sparkles className="h-3.5 w-3.5" />
            30 秒记录一笔
          </div>
          <h2 className="mt-4 text-3xl font-black text-theme-primary">
            新增一笔持仓
          </h2>
          {!compact && (
            <p className="mt-2 max-w-3xl text-sm leading-6 text-theme-secondary">
              填写基金、金额和交易日，即可开始跟踪净值、份额和盈亏。
            </p>
          )}
        </div>

        {!compact && (
          <div className="grid gap-2 sm:grid-cols-3 lg:w-[460px]">
            {steps.map((step, index) => (
              <div
                key={step.label}
                className={cn(
                  "rounded-[20px] border px-3 py-3",
                  step.done
                    ? "border-emerald-400/25 bg-emerald-400/10"
                    : "border-[var(--card-border)] bg-[var(--input-bg)]/62",
                )}
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="text-xs font-medium text-theme-primary">
                    {index + 1}. {step.label}
                  </span>
                  {step.done ? (
                    <CheckCircle2 className="h-4 w-4 text-emerald-300" />
                  ) : (
                    <span className="h-2 w-2 rounded-full bg-[var(--text-muted)]" />
                  )}
                </div>
                <div className="mt-1 truncate text-[11px] text-theme-muted">
                  {step.detail}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      <div
        className={cn(
          "grid gap-5",
          compact ? "xl:grid-cols-1" : "xl:grid-cols-[1.08fr_0.92fr]",
        )}
      >
        <div className="space-y-4">
          {!compact && visibleRecentAggregates.length > 0 && (
            <div className="rounded-[28px] border border-amber-400/18 bg-amber-400/8 p-4">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <div className="flex items-center gap-2 text-sm font-medium text-theme-primary">
                    <History className="h-4 w-4 text-amber-300" />
                    继续记录已有基金
                  </div>
                  <div className="mt-1 text-xs leading-5 text-theme-muted">
                    补仓或复记时不用重新搜索，点一下就带入基金。
                  </div>
                </div>
                <span className="rounded-full border border-amber-400/25 bg-amber-400/10 px-3 py-1 text-[11px] tracking-[0.18em] text-amber-200">
                  复记
                </span>
              </div>
              <div className="mt-3 flex gap-2 overflow-x-auto pb-1">
                {visibleRecentAggregates.map((aggregate) => {
                  const active = resolvedFundID === aggregate.fund_id;
                  return (
                    <button
                      key={aggregate.fund_id}
                      type="button"
                      onClick={() => onSelectAggregate(aggregate)}
                      className={cn(
                        "min-w-[180px] rounded-[20px] border px-4 py-3 text-left transition-all duration-200",
                        active
                          ? "border-amber-400/55 bg-amber-400/16 shadow-[0_12px_26px_rgba(251,191,36,0.12)]"
                          : "border-[var(--input-border)] bg-[var(--card-bg)]/78 hover:border-amber-400/35",
                      )}
                    >
                      <div className="truncate text-sm font-semibold text-theme-primary">
                        {aggregate.fund?.name || aggregate.fund_id}
                      </div>
                      <div className="mt-1 text-xs text-theme-muted">
                        {aggregate.holding_count} 笔 · 本金{" "}
                        {formatInlineMoney(aggregate.total_principal)}
                      </div>
                    </button>
                  );
                })}
              </div>
            </div>
          )}

          <div className="rounded-[28px] border border-[var(--card-border)] bg-[var(--input-bg)]/58 p-4">
            <div className="flex flex-col gap-4 lg:flex-row lg:items-start">
              <label className="min-w-0 flex-1 space-y-2">
                <span className="text-sm font-medium text-theme-primary">
                  你在记录哪只基金？
                </span>
                <input
                  value={query}
                  onChange={(event) => onQueryChange(event.target.value)}
                  placeholder="输入基金代码或名称，例如 005827"
                  className="auth-input w-full rounded-2xl border border-[var(--input-border)] bg-[var(--card-bg)]/86 px-4 py-3 text-theme-primary outline-none placeholder:text-theme-muted"
                />
                <span className="block truncate text-xs text-theme-muted">
                  {hasFund
                    ? `${selectedFundLabel} · ${resolvedFundID}`
                    : "选中后会自动进入待记录状态"}
                </span>
              </label>

              <div className="grid gap-2 rounded-[22px] border border-cyan-400/18 bg-cyan-400/8 p-3 text-xs text-theme-secondary lg:w-52">
                <div className="font-medium text-theme-primary">
                  录入后你会得到
                </div>
                <div>真实市值与今日盈亏</div>
                <div>量化建议与风险标签</div>
                <div>组合风险与结构回顾</div>
              </div>
            </div>

            <div className="mt-4 grid gap-2 md:grid-cols-2 xl:grid-cols-3">
              {results.slice(0, 6).map((fund) => {
                const active = resolvedFundID === fund.id;
                return (
                  <button
                    key={fund.id}
                    type="button"
                    onClick={() => onSelectFund(fund)}
                    className={cn(
                      "flex min-w-0 items-center justify-between gap-3 rounded-2xl border px-4 py-3 text-left transition-all duration-200",
                      active
                        ? "border-cyan-400/50 bg-cyan-400/14 shadow-[0_12px_26px_rgba(34,211,238,0.12)]"
                        : "border-[var(--input-border)] bg-[var(--card-bg)]/76 hover:border-cyan-500/40",
                    )}
                  >
                    <span className="min-w-0">
                      <span className="block truncate text-sm font-medium text-theme-primary">
                        {fund.name}
                      </span>
                      <span className="mt-1 block text-xs text-theme-muted">
                        {fund.id}
                      </span>
                    </span>
                    {active ? (
                      <CheckCircle2 className="h-4 w-4 shrink-0 text-cyan-300" />
                    ) : (
                      <Plus className="h-4 w-4 shrink-0 text-cyan-300" />
                    )}
                  </button>
                );
              })}

              {results.length === 0 && (
                <div className="rounded-2xl border border-dashed border-[var(--card-border)] px-4 py-8 text-center text-sm text-theme-secondary md:col-span-2 xl:col-span-3">
                  输入至少 2
                  个字符后展示候选基金；如果基金唯一匹配，也会自动识别。
                </div>
              )}
            </div>
          </div>

          <div className="grid gap-4 lg:grid-cols-[0.95fr_1.05fr]">
            <div className="rounded-[28px] border border-[var(--card-border)] bg-[var(--input-bg)]/58 p-4">
              <label className="space-y-2">
                <span className="text-sm font-medium text-theme-primary">
                  这笔大概多少本金？
                </span>
                <input
                  value={amount}
                  onChange={(event) => onAmountChange(event.target.value)}
                  placeholder="例如 30000"
                  inputMode="decimal"
                  className="auth-input w-full rounded-2xl border border-[var(--input-border)] bg-[var(--card-bg)]/86 px-4 py-3 text-theme-primary outline-none placeholder:text-theme-muted"
                />
              </label>
              <div className="mt-3 flex flex-wrap gap-2">
                {quickAmounts.map((option) => (
                  <button
                    key={option}
                    type="button"
                    onClick={() => onAmountChange(option.replace(/,/g, ""))}
                    className="rounded-full border border-[var(--input-border)] bg-[var(--card-bg)]/78 px-3 py-1.5 text-xs text-theme-secondary transition-colors hover:border-cyan-400/35 hover:text-theme-primary"
                  >
                    ¥{option}
                  </button>
                ))}
              </div>
              <div className="mt-3 text-xs leading-5 text-theme-muted">
                已记录 {holdingsCount} 笔，当前总本金 {totalPrincipalText}。
              </div>
            </div>

            <div className="rounded-[28px] border border-[var(--card-border)] bg-[var(--input-bg)]/58 p-4">
              <label>
                <span className="text-sm font-medium text-theme-primary">
                  给未来的自己留一句备注
                </span>
                <input
                  value={note}
                  onChange={(event) => onNoteChange(event.target.value)}
                  placeholder="例如：长期底仓 / 回调补仓 / 观察仓"
                  className="auth-input mt-2 w-full rounded-2xl border border-[var(--input-border)] bg-[var(--card-bg)]/86 px-4 py-3 text-theme-primary outline-none placeholder:text-theme-muted"
                />
              </label>
              <div className="mt-3 flex flex-wrap gap-2">
                {notePresets.map((preset) => (
                  <button
                    key={preset}
                    type="button"
                    onClick={() => onNoteChange(preset)}
                    className={cn(
                      "inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs transition-colors",
                      note === preset
                        ? "border-amber-400/45 bg-amber-400/14 text-amber-100"
                        : "border-[var(--input-border)] bg-[var(--card-bg)]/78 text-theme-secondary hover:border-amber-400/35 hover:text-theme-primary",
                    )}
                  >
                    <Tags className="h-3 w-3" />
                    {preset}
                  </button>
                ))}
              </div>
              <span className="mt-3 block text-xs leading-5 text-theme-muted">
                备注可选，但建议写下这笔钱的策略背景，回看时更有价值。
              </span>
            </div>
          </div>
        </div>

        <div className="rounded-[30px] border border-[var(--card-border)] bg-[var(--card-bg)]/88 p-5">
          <div className="flex items-start justify-between gap-3">
            <div>
              <div className="text-sm font-medium text-theme-primary">
                交易时间与确认规则
              </div>
              <div className="mt-1 text-xs leading-5 text-theme-muted">
                选日期和 15:00 前后即可。
              </div>
            </div>
            <div className="rounded-full border border-cyan-400/30 bg-cyan-400/10 px-3 py-1 text-[11px] font-medium tracking-[0.18em] text-cyan-200">
              T+1
            </div>
          </div>

          <div className="mt-5 grid gap-4">
            <label className="holding-picker-shell relative overflow-hidden rounded-[24px] border border-[var(--input-border)] bg-[var(--input-bg)]/85 px-4 py-4 transition-all duration-200 hover:border-cyan-400/40 hover:bg-[var(--input-bg)] focus-within:border-cyan-400/60 focus-within:bg-[var(--input-bg)] focus-within:shadow-[0_14px_30px_rgba(34,211,238,0.12)]">
              <span className="holding-picker-shine" />
              <span className="relative z-10 flex items-center gap-2 text-xs font-medium text-theme-secondary">
                <CalendarDays className="h-3.5 w-3.5 text-cyan-300" />
                交易日期
              </span>
              <div className="relative z-10 mt-4 rounded-[20px] border border-[var(--input-border)] bg-[var(--card-bg)]/85 px-4 py-3">
                <input
                  type="date"
                  value={tradeDate}
                  onChange={(event) => onTradeDateChange(event.target.value)}
                  className="holding-datetime-input w-full text-sm font-medium text-theme-primary outline-none"
                />
              </div>
              <div className="relative z-10 mt-3 text-xs text-theme-secondary">
                当前选择：
                <span className="font-medium text-theme-primary">
                  {tradeDateLabel}
                </span>
              </div>
            </label>

            <div className="grid gap-3 sm:grid-cols-2">
              {(
                [
                  {
                    id: "before_close",
                    title: "15:00 前",
                    description: "按当日收盘净值确认",
                  },
                  {
                    id: "after_close",
                    title: "15:00 后",
                    description: "顺延至下个交易日确认",
                  },
                ] as const
              ).map((option) => (
                <button
                  key={option.id}
                  type="button"
                  onClick={() => onTradeTimingChange(option.id)}
                  className={cn(
                    "rounded-[20px] border px-4 py-3 text-left transition-all duration-200",
                    tradeTiming === option.id
                      ? "border-cyan-400/55 bg-cyan-400/14 text-cyan-100 shadow-[0_12px_26px_rgba(34,211,238,0.12)]"
                      : "border-[var(--input-border)] bg-[var(--card-bg)]/72 text-theme-secondary hover:border-cyan-400/35 hover:text-theme-primary",
                  )}
                  aria-pressed={tradeTiming === option.id}
                >
                  <div className="flex items-center gap-2 text-sm font-semibold">
                    <Clock3 className="h-3.5 w-3.5" />
                    {option.title}
                  </div>
                  <div className="mt-1 text-xs leading-5 text-theme-muted">
                    {option.description}
                  </div>
                </button>
              ))}
            </div>
          </div>

          <div className="mt-4 flex flex-wrap gap-2">
            {["今天", "上个交易日", "下个交易日"].map((shortcut) => {
              const shortcutDate =
                shortcut === "今天"
                  ? todayTradeDate
                  : shortcut === "上个交易日"
                    ? previousTradeDate
                    : nextTradeDate;
              return (
                <button
                  key={shortcut}
                  type="button"
                  onClick={() => onTradeDateChange(shortcutDate)}
                  className={cn(
                    "rounded-full border px-3 py-1.5 text-xs transition-all duration-200",
                    tradeDate === shortcutDate
                      ? "border-cyan-400/50 bg-cyan-400/15 text-cyan-100 shadow-[0_10px_22px_rgba(34,211,238,0.12)]"
                      : "border-[var(--input-border)] bg-[var(--input-bg)]/70 text-theme-secondary hover:border-cyan-400/35 hover:text-theme-primary",
                  )}
                >
                  {shortcut}
                </button>
              );
            })}
          </div>

          <div className="mt-4 rounded-[24px] border border-cyan-400/20 bg-cyan-400/10 px-4 py-4">
            <div className="text-[11px] font-medium tracking-[0.18em] text-cyan-200">
              净值确认提示
            </div>
            <div className="mt-3 flex items-start justify-between gap-4">
              <div className="min-w-0">
                <div className="text-sm font-medium text-theme-primary">
                  {tradeDate
                    ? `${tradeDate} · ${tradeTimingLabel}`
                    : "请选择交易日期"}
                </div>
                {!compact && (
                  <div className="mt-1 text-xs leading-5 text-theme-secondary">
                    {pricingRuleLabel}
                  </div>
                )}
              </div>
              <div className="shrink-0 text-right">
                <div className="text-xs text-theme-muted">确认净值日</div>
                <div className="mt-1 text-lg font-semibold text-cyan-100">
                  {pricingDatePreview || "--"}
                </div>
              </div>
            </div>
          </div>

          <div className="mt-4 grid gap-3 rounded-[24px] border border-[var(--card-border)] bg-[var(--input-bg)]/60 px-4 py-4 text-sm">
            <div className="text-[11px] font-medium tracking-[0.18em] text-theme-muted">
              本次将记录
            </div>
            <div className="grid gap-3 sm:grid-cols-3">
              <div>
                <div className="text-xs text-theme-muted">基金</div>
                <div className="mt-1 truncate font-semibold text-theme-primary">
                  {selectedFundLabel}
                </div>
              </div>
              <div>
                <div className="text-xs text-theme-muted">本金</div>
                <div className="mt-1 font-semibold text-theme-primary">
                  {hasAmount ? formatInlineMoney(String(amountValue)) : "--"}
                </div>
              </div>
              <div>
                <div className="text-xs text-theme-muted">备注</div>
                <div className="mt-1 truncate font-semibold text-theme-primary">
                  {note || "未填写"}
                </div>
              </div>
            </div>
          </div>

          <button
            type="button"
            onClick={onAddHolding}
            disabled={!canSubmit}
            className={cn(
              "group relative mt-5 inline-flex w-full items-center justify-center gap-2 overflow-hidden rounded-2xl bg-gradient-to-r from-cyan-500 via-sky-500 to-blue-600 px-4 py-3 text-sm font-medium text-white transition-all duration-200",
              "hover:-translate-y-0.5 hover:shadow-[0_18px_35px_rgba(14,165,233,0.28)]",
              "active:scale-[0.985] disabled:cursor-not-allowed disabled:opacity-70",
              isAddingHolding && "holding-action-button",
            )}
            aria-busy={isAddingHolding}
          >
            <span className="holding-action-shine" />
            {isAddingHolding ? (
              <LoaderCircle className="relative z-10 h-4 w-4 animate-spin" />
            ) : (
              <Wallet className="relative z-10 h-4 w-4 transition-transform duration-300 group-hover:-rotate-6 group-hover:scale-110" />
            )}
            <span className="relative z-10">
              {isAddingHolding
                ? "提交中..."
                : canSubmit
                  ? "记录这笔持仓"
                  : remainingHint}
            </span>
          </button>
        </div>
      </div>
    </section>
  );
}
