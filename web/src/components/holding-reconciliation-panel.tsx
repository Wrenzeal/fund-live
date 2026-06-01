'use client'

import { AlertTriangle, CheckCircle2, Scale, Wrench } from 'lucide-react'
import { useMemo, useState } from 'react'
import type {
  HoldingEntry,
  HoldingTransactionEntry,
} from '@/hooks/use-user-portfolio'
import {
  buildHoldingReconciliationSummary,
  type ReconciliationSeverity,
} from '@/lib/holding-insights'
import { cn } from '@/lib/utils'

interface HoldingReconciliationPanelProps {
  holdings: HoldingEntry[]
  transactions?: HoldingTransactionEntry[]
  compact?: boolean
}

type ReconciliationFilter = 'all' | 'review' | 'manual'

function formatMoney(value: number | null) {
  if (value === null || !Number.isFinite(value)) {
    return '--'
  }
  return new Intl.NumberFormat('zh-CN', {
    style: 'currency',
    currency: 'CNY',
    maximumFractionDigits: 2,
  }).format(value)
}

function formatPercent(value: number | null) {
  if (value === null || !Number.isFinite(value)) {
    return '--'
  }
  return `${value >= 0 ? '+' : ''}${value.toFixed(2)}%`
}

function severityMeta(severity: ReconciliationSeverity) {
  switch (severity) {
    case 'review':
      return {
        label: '需复核',
        className: 'border-rose-400/35 bg-rose-500/12 text-rose-200',
      }
    case 'watch':
      return {
        label: '小差异',
        className: 'border-amber-400/35 bg-amber-400/12 text-amber-100',
      }
    case 'pending':
      return {
        label: '待补齐',
        className: 'border-sky-400/30 bg-sky-400/10 text-sky-100',
      }
    case 'ok':
    default:
      return {
        label: '已对齐',
        className: 'border-emerald-400/30 bg-emerald-400/10 text-emerald-100',
      }
  }
}

export function HoldingReconciliationPanel({
  holdings,
  transactions = [],
  compact = false,
}: HoldingReconciliationPanelProps) {
  const [filter, setFilter] = useState<ReconciliationFilter>('all')
  const summary = buildHoldingReconciliationSummary(holdings)
  const filteredItems = summary.items.filter((item) => {
    if (filter === 'review') {
      return (
        item.severity === 'review' ||
        item.severity === 'watch' ||
        item.severity === 'pending'
      )
    }
    if (filter === 'manual') {
      return item.manual
    }
    return true
  })
  const highlightedItems = filteredItems
    .filter((item) => item.severity !== 'ok' || filter !== 'all')
    .slice(0, 5)
  const allClear = summary.total > 0 && highlightedItems.length === 0
  const correctionHistory = useMemo(() => {
    return transactions
      .filter((transaction) => transaction.type === 'correction')
      .slice(0, 4)
  }, [transactions])

  return (
    <section className="mb-6 overflow-hidden rounded-[30px] border border-[var(--card-border)] bg-[var(--card-bg)]/84 p-5 glass">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div>
          <div className="inline-flex items-center gap-2 rounded-full border border-cyan-400/25 bg-cyan-400/10 px-3 py-1 text-[11px] font-medium tracking-[0.2em] text-cyan-200">
            <Scale className="h-3.5 w-3.5" />
            持仓对账
          </div>
          <h3 className="mt-3 text-2xl font-black text-theme-primary">
            对账与校正
          </h3>
          {!compact && (
            <p className="mt-2 max-w-3xl text-sm leading-6 text-theme-secondary">
              对比本金和“份额 × 净值”，找出需要校正的记录。
            </p>
          )}
        </div>
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-4 lg:w-[520px]">
          {[
            {
              label: '已校正',
              value: summary.manualCount,
              tone: 'text-cyan-200',
            },
            {
              label: '需复核',
              value: summary.reviewCount,
              tone:
                summary.reviewCount > 0
                  ? 'text-rose-200'
                  : 'text-theme-primary',
            },
            {
              label: '小差异',
              value: summary.watchCount,
              tone:
                summary.watchCount > 0
                  ? 'text-amber-100'
                  : 'text-theme-primary',
            },
            {
              label: '待补齐',
              value: summary.pendingCount,
              tone:
                summary.pendingCount > 0
                  ? 'text-sky-100'
                  : 'text-theme-primary',
            },
          ].map((item) => (
            <div
              key={item.label}
              className="rounded-[20px] border border-[var(--card-border)] bg-[var(--input-bg)]/62 px-3 py-3"
            >
              <div className="text-xs text-theme-muted">{item.label}</div>
              <div className={cn('mt-1 text-2xl font-black', item.tone)}>
                {item.value}
              </div>
            </div>
          ))}
        </div>
      </div>

      <div className="mt-5 flex flex-wrap gap-2">
        {[
          { value: 'all' as const, label: '全部对账' },
          { value: 'review' as const, label: '只看需复核' },
          { value: 'manual' as const, label: '只看已校正' },
        ].map((item) => (
          <button
            key={item.value}
            type="button"
            onClick={() => setFilter(item.value)}
            className={cn(
              'rounded-2xl border px-3 py-2 text-xs font-semibold transition',
              filter === item.value
                ? 'border-cyan-300/35 bg-cyan-400/14 text-cyan-100'
                : 'border-[var(--input-border)] bg-[var(--input-bg)] text-theme-secondary hover:border-cyan-300/30 hover:text-theme-primary',
            )}
          >
            {item.label}
          </button>
        ))}
      </div>

      <div
        className={cn(
          'mt-5 grid gap-4',
          compact ? 'xl:grid-cols-1' : 'xl:grid-cols-[0.92fr_1.08fr]',
        )}
      >
        {!compact && (
          <div className="space-y-3">
            <div className="rounded-[24px] border border-cyan-400/18 bg-cyan-400/8 p-4">
              <div className="flex items-start gap-3">
                <Wrench className="mt-1 h-5 w-5 shrink-0 text-cyan-200" />
                <div>
                  <div className="text-sm font-semibold text-theme-primary">
                    什么时候用“校正”？
                  </div>
                  <div className="mt-2 space-y-1 text-xs leading-5 text-theme-secondary">
                    <p>• 支付宝 / 微信显示份额或净值与系统自动确认不同。</p>
                    <p>
                      • 历史迁移、手续费、分红导致本金和份额口径不完全相等。
                    </p>
                    <p>• 这不是买入 / 卖出；只是修正当前这笔持仓的确认口径。</p>
                  </div>
                </div>
              </div>
            </div>
            <div className="rounded-[24px] border border-[var(--card-border)] bg-[var(--input-bg)]/54 p-4">
              <div className="text-sm font-semibold text-theme-primary">
                差异历史第一版
              </div>
              <p className="mt-1 text-xs leading-5 text-theme-muted">
                先从校正流水里还原最近几次“校正前 →
                当前值”，完整影响链可进入流水详情继续查看。
              </p>
              <div className="mt-3 space-y-2">
                {correctionHistory.length === 0 ? (
                  <div className="rounded-2xl border border-dashed border-[var(--card-border)] px-3 py-3 text-xs text-theme-muted">
                    暂无校正流水；后续每次校正都会在这里留下对账痕迹。
                  </div>
                ) : (
                  correctionHistory.map((transaction) => (
                    <div
                      key={transaction.id}
                      className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/56 px-3 py-3 text-xs"
                    >
                      <div className="font-semibold text-theme-primary">
                        {transaction.fund?.name || transaction.fund_id}
                      </div>
                      <div className="mt-1 text-theme-muted">
                        本金 {transaction.metadata?.previous_amount || '--'} →{' '}
                        {formatMoney(
                          Number.parseFloat(transaction.amount || ''),
                        )}
                      </div>
                      <div className="mt-1 text-theme-muted">
                        份额 {transaction.metadata?.previous_shares || '--'} →{' '}
                        {transaction.shares || '--'}；净值日{' '}
                        {transaction.metadata?.previous_confirmed_nav_date ||
                          '--'}{' '}
                        → {transaction.confirmed_nav_date || '--'}
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>
        )}

        <div className="space-y-3">
          {allClear ? (
            <div className="rounded-[24px] border border-emerald-400/22 bg-emerald-400/10 p-4 text-sm text-emerald-100">
              <div className="flex items-center gap-2 font-semibold">
                <CheckCircle2 className="h-4 w-4" />
                当前筛选下没有需要处理的对账项
              </div>
              <p className="mt-2 text-xs leading-5 text-theme-secondary">
                如果外部平台显示不同，可在单条持仓右侧点击“校正”，并保留来源方便以后筛选。
              </p>
            </div>
          ) : (
            highlightedItems.map((item) => {
              const meta = severityMeta(item.severity)
              return (
                <div
                  key={item.id}
                  className="rounded-[22px] border border-[var(--card-border)] bg-[var(--input-bg)]/54 p-4"
                >
                  <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
                    <div className="min-w-0">
                      <div className="truncate text-sm font-semibold text-theme-primary">
                        {item.fundName}
                      </div>
                      <div className="mt-1 text-xs text-theme-muted">
                        {item.fundID}
                      </div>
                    </div>
                    <span
                      className={cn(
                        'inline-flex w-fit items-center rounded-full border px-2.5 py-1 text-[11px] font-medium',
                        meta.className,
                      )}
                    >
                      {meta.label}
                      {item.manual
                        ? ` · ${item.sourceLabel || '外部'}校正`
                        : ''}
                    </span>
                  </div>
                  <div className="mt-3 grid gap-2 text-xs text-theme-secondary sm:grid-cols-3">
                    <div>
                      本金：
                      <span className="text-theme-primary">
                        {formatMoney(item.amount)}
                      </span>
                    </div>
                    <div>
                      份额×净值：
                      <span className="text-theme-primary">
                        {formatMoney(item.impliedPrincipal)}
                      </span>
                    </div>
                    <div>
                      差异：
                      <span
                        className={
                          item.severity === 'review'
                            ? 'text-rose-200'
                            : 'text-theme-primary'
                        }
                      >
                        {formatMoney(item.difference)} /{' '}
                        {formatPercent(item.differencePercent)}
                      </span>
                    </div>
                  </div>
                  <div className="mt-2 flex items-start gap-2 text-xs leading-5 text-theme-muted">
                    <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
                    <span>{item.reason}</span>
                  </div>
                </div>
              )
            })
          )}
        </div>
      </div>
    </section>
  )
}

/*
 * The previous first-round static explanation block has been intentionally
 * folded into the richer two-column panel above. Keep this component local:
 * it owns only presentation/filtering while reconciliation rules stay in
 * `holding-insights.ts`.
 */
