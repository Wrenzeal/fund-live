"use client";

import { useId } from "react";
import { LoaderCircle } from "lucide-react";
import type { Fund } from "@/hooks/use-fund-data";
import type { HoldingAggregateEntry } from "@/hooks/use-user-portfolio";
import { ActionButton } from "@/components/ui/action-button";
import { cn } from "@/lib/utils";
import type { TradeTiming } from "@/lib/holding-display";

interface HoldingRecordComposerProps {
  query: string;
  results: Fund[];
  recentAggregates: HoldingAggregateEntry[];
  resolvedFundID: string;
  resolvedFundName: string;
  amount: string;
  note: string;
  tradeDate: string;
  tradeTiming: TradeTiming;
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

const inputClass = "auth-input min-h-11 w-full rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-base text-theme-primary placeholder:text-theme-muted";

export function HoldingRecordComposer({
  query,
  results,
  recentAggregates,
  resolvedFundID,
  resolvedFundName,
  amount,
  note,
  tradeDate,
  tradeTiming,
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
  const formID = useId();
  const hasFund = Boolean(resolvedFundID);
  const hasAmount = parseAmount(amount) > 0;
  const hasTradeDate = Boolean(tradeDate.trim());
  const canSubmit = hasFund && hasAmount && hasTradeDate && !isAddingHolding;
  const visibleRecentAggregates = recentAggregates.slice(0, 3);
  const remainingHint = !hasFund
    ? "请选择基金后继续。"
    : !hasAmount
      ? "填写本次买入的本金金额。"
      : !hasTradeDate
        ? "选择实际交易日期。"
        : "保存后按确认净值日更新份额和市值。";

  return (
    <section className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)] p-5 sm:p-6" aria-labelledby={`${formID}-title`}>
      <h2 id={`${formID}-title`} className="text-xl font-semibold text-theme-primary">记录买入</h2>
      <p className="mt-2 text-sm leading-6 text-theme-secondary">按实际买入本金和交易日期记录，不把当前市值当作买入金额。</p>

      {visibleRecentAggregates.length > 0 && (
        <div className="mt-5">
          <p className="text-sm font-medium text-theme-secondary">最近记录的基金</p>
          <div className="mt-2 flex flex-wrap gap-2">
            {visibleRecentAggregates.map((aggregate) => (
              <ActionButton
                key={aggregate.fund_id}
                type="button"
                variant="subtle"
                aria-pressed={resolvedFundID === aggregate.fund_id}
                onClick={() => onSelectAggregate(aggregate)}
                className={cn("min-h-11 max-w-full whitespace-normal text-left", resolvedFundID === aggregate.fund_id && "border-[var(--accent-primary)] text-theme-primary")}
              >
                <span className="break-words">{aggregate.fund?.name || aggregate.fund_id}</span>
              </ActionButton>
            ))}
          </div>
        </div>
      )}

      <form
        className="mt-5 space-y-5"
        aria-busy={isAddingHolding}
        onSubmit={(event) => {
          event.preventDefault();
          if (canSubmit) onAddHolding();
        }}
      >
        <div>
          <label htmlFor={`${formID}-fund`} className="mb-2 block text-sm font-medium text-theme-primary">基金</label>
          <input
            id={`${formID}-fund`}
            value={query}
            onChange={(event) => onQueryChange(event.target.value)}
            placeholder="基金代码或名称"
            autoComplete="off"
            required
            aria-describedby={`${formID}-selection`}
            className={inputClass}
          />
          <p id={`${formID}-selection`} className="mt-2 break-words text-sm leading-6 text-theme-secondary" aria-live="polite">
            {hasFund ? `已选择 ${resolvedFundName || resolvedFundID} · ${resolvedFundID}` : "输入代码或名称，再从结果中选择基金。"}
          </p>
          {results.length > 0 && (
            <ul aria-label="基金搜索结果" className="mt-2 grid gap-2 sm:grid-cols-2">
              {results.slice(0, 6).map((fund) => (
                <li key={fund.id} className="min-w-0">
                  <button
                    type="button"
                    onClick={() => onSelectFund(fund)}
                    aria-pressed={resolvedFundID === fund.id}
                    className={cn("flex min-h-11 w-full items-center justify-between gap-3 rounded-xl px-3 py-2.5 text-left text-sm transition-colors hover:bg-[var(--input-bg)]", resolvedFundID === fund.id && "bg-[var(--input-bg)]")}
                  >
                    <span className="min-w-0 break-words font-medium text-theme-primary">{fund.name}</span>
                    <span className="shrink-0 tabular-nums text-theme-secondary">{fund.id}</span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>

        <div className="grid items-start gap-5 sm:grid-cols-2">
          <div>
            <label htmlFor={`${formID}-amount`} className="mb-2 block text-sm font-medium text-theme-primary">买入金额（元）</label>
            <input
              id={`${formID}-amount`}
              value={amount}
              onChange={(event) => onAmountChange(event.target.value)}
              placeholder="例如 1000.00"
              inputMode="decimal"
              required
              className={cn(inputClass, "tabular-nums")}
            />
          </div>
          <div>
            <label htmlFor={`${formID}-date`} className="mb-2 block text-sm font-medium text-theme-primary">交易日期</label>
            <input id={`${formID}-date`} type="date" value={tradeDate} onChange={(event) => onTradeDateChange(event.target.value)} required className={inputClass} />
            <div className="mt-1 flex flex-wrap gap-1">
              {[
                { label: "今天", value: todayTradeDate },
                { label: "上个交易日", value: previousTradeDate },
                { label: "下个交易日", value: nextTradeDate },
              ].map((shortcut) => (
                <ActionButton key={shortcut.label} type="button" variant="ghost" size="sm" disabled={!shortcut.value} onClick={() => onTradeDateChange(shortcut.value)} className="min-h-11">
                  {shortcut.label}
                </ActionButton>
              ))}
            </div>
          </div>
        </div>

        <fieldset>
          <legend className="mb-2 text-sm font-medium text-theme-primary">提交时段</legend>
          <div className="flex flex-wrap gap-2">
            {([
              { id: "before_close", title: "15:00 前" },
              { id: "after_close", title: "15:00 后" },
            ] as const).map((option) => (
              <ActionButton
                key={option.id}
                type="button"
                variant="secondary"
                onClick={() => onTradeTimingChange(option.id)}
                aria-pressed={tradeTiming === option.id}
                className={cn("min-h-11 flex-1 sm:flex-none", tradeTiming === option.id && "border-[var(--accent-primary)] bg-[var(--accent-primary)]/10")}
              >
                {option.title}
              </ActionButton>
            ))}
          </div>
          <p className="mt-3 text-sm leading-6 text-theme-secondary" aria-live="polite">
            {pricingDatePreview && <span className="font-medium text-theme-primary">确认净值日 {pricingDatePreview}。 </span>}
            {pricingRuleLabel}
          </p>
        </fieldset>

        <details className="text-sm text-theme-secondary">
          <summary className="min-h-11 cursor-pointer py-3 font-medium">备注（可选）{note ? " · 已填写" : ""}</summary>
          <label htmlFor={`${formID}-note`} className="sr-only">备注</label>
          <input id={`${formID}-note`} value={note} onChange={(event) => onNoteChange(event.target.value)} placeholder="记录这次买入的说明" className={inputClass} />
        </details>

        <div className="flex flex-col gap-3 border-t border-[var(--card-border)] pt-5 sm:flex-row sm:items-center sm:justify-between">
          <p id={`${formID}-submit-hint`} className="text-sm leading-6 text-theme-secondary">{remainingHint}</p>
          <ActionButton type="submit" variant="primary" disabled={!canSubmit} aria-busy={isAddingHolding} aria-describedby={`${formID}-submit-hint`} className="holding-primary-action min-h-11 w-full sm:w-auto">
            {isAddingHolding && <LoaderCircle className="h-4 w-4 animate-spin motion-reduce:animate-none" aria-hidden="true" />}
            {isAddingHolding ? "正在保存…" : "保存买入记录"}
          </ActionButton>
        </div>
      </form>
    </section>
  );
}
