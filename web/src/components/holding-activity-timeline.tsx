'use client'

import { Ban, CalendarDays, CheckCircle2, History, Pencil, Plus, RotateCcw, Search, Trash2, WalletCards } from 'lucide-react'
import Link from 'next/link'
import { useState } from 'react'
import type {
  HoldingTransactionEntry,
  HoldingTransactionRollbackApplyResult,
  HoldingTransactionRollbackPreview,
  HoldingTransactionStatusFilter,
  HoldingTransactionType,
} from '@/hooks/use-user-portfolio'
import { formatSummaryMoney } from '@/lib/holding-display'
import {
  HOLDING_SOURCE_OPTIONS,
  resolveHoldingSourceLabel,
  type HoldingSourceFilter,
} from '@/lib/holding-sources'
import { cn } from '@/lib/utils'

interface HoldingActivityTimelineProps {
  transactions: HoldingTransactionEntry[]
  onVoidTransaction?: (transactionID: string, reason: string) => Promise<void> | void
  onPreviewRollback?: (transactionID: string) => Promise<HoldingTransactionRollbackPreview | null> | HoldingTransactionRollbackPreview | null | void
  onApplyRollback?: (transactionID: string, reason: string) => Promise<HoldingTransactionRollbackApplyResult | null> | HoldingTransactionRollbackApplyResult | null | void
  fundOptions?: Array<{ fund_id: string; name: string }>
  fundFilter?: string
  typeFilter?: HoldingTransactionType | 'all'
  statusFilter?: HoldingTransactionStatusFilter
  sourceFilter?: HoldingSourceFilter
  keywordFilter?: string
  startDateFilter?: string
  endDateFilter?: string
  visibleLimit?: number
  canLoadMore?: boolean
  onFundFilterChange?: (value: string) => void
  onTypeFilterChange?: (value: HoldingTransactionType | 'all') => void
  onStatusFilterChange?: (value: HoldingTransactionStatusFilter) => void
  onSourceFilterChange?: (value: HoldingSourceFilter) => void
  onKeywordFilterChange?: (value: string) => void
  onStartDateFilterChange?: (value: string) => void
  onEndDateFilterChange?: (value: string) => void
  onLoadMore?: () => void
  onClearFilters?: () => void
}

function transactionMeta(type: HoldingTransactionType) {
  switch (type) {
    case 'buy':
      return {
        label: '买入/补仓',
        description: '新增持仓本金，等待或已按确认净值补齐份额。',
        className: 'border-emerald-400/28 bg-emerald-400/10 text-emerald-100',
        dotClassName: 'bg-emerald-300 shadow-[0_0_18px_rgba(110,231,183,0.42)]',
        icon: Plus,
      }
    case 'correction':
      return {
        label: '校正持仓',
        description: '按外部平台或真实确认口径修正金额、份额或净值。',
        className: 'border-cyan-400/28 bg-cyan-400/10 text-cyan-100',
        dotClassName: 'bg-cyan-300 shadow-[0_0_18px_rgba(103,232,249,0.42)]',
        icon: Pencil,
      }
    case 'delete':
      return {
        label: '删除记录',
        description: '从当前持仓中移除，但保留账本活动痕迹。',
        className: 'border-rose-400/28 bg-rose-500/10 text-rose-100',
        dotClassName: 'bg-rose-300 shadow-[0_0_18px_rgba(253,164,175,0.42)]',
        icon: Trash2,
      }
    case 'sell':
      return {
        label: '卖出',
        description: '赎回或降低当前仓位。',
        className: 'border-amber-400/28 bg-amber-400/10 text-amber-100',
        dotClassName: 'bg-amber-300 shadow-[0_0_18px_rgba(252,211,77,0.42)]',
        icon: WalletCards,
      }
    case 'dividend':
      return {
        label: '分红',
        description: '现金分红或红利再投导致账面口径变化。',
        className: 'border-violet-400/28 bg-violet-400/10 text-violet-100',
        dotClassName: 'bg-violet-300 shadow-[0_0_18px_rgba(196,181,253,0.42)]',
        icon: WalletCards,
      }
    case 'adjustment':
    default:
      return {
        label: '调整',
        description: '平台迁移、手续费或其他持仓口径调整。',
        className: 'border-sky-400/28 bg-sky-400/10 text-sky-100',
        dotClassName: 'bg-sky-300 shadow-[0_0_18px_rgba(125,211,252,0.42)]',
        icon: CheckCircle2,
      }
  }
}

function formatDateTime(value?: string) {
  if (!value) {
    return '--'
  }

  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return value
  }

  return new Intl.DateTimeFormat('zh-CN', {
    timeZone: 'Asia/Shanghai',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(parsed)
}

function formatCompactDate(value?: string) {
  if (!value) {
    return '待确认'
  }

  const parsed = new Date(`${value}T12:00:00+08:00`)
  if (Number.isNaN(parsed.getTime())) {
    return value
  }

  return new Intl.DateTimeFormat('zh-CN', {
    timeZone: 'Asia/Shanghai',
    month: '2-digit',
    day: '2-digit',
  }).format(parsed)
}

function formatRollbackValue(value?: string) {
  if (!value) {
    return '--'
  }
  return value
}

const activityCoverageTypes: HoldingTransactionType[] = ['buy', 'sell', 'correction', 'dividend', 'adjustment', 'delete']
const transactionTypeOptions: Array<{ value: HoldingTransactionType | 'all'; label: string }> = [
  { value: 'all', label: '全部类型' },
  { value: 'buy', label: '买入/补仓' },
  { value: 'sell', label: '卖出/清仓' },
  { value: 'correction', label: '校正' },
  { value: 'delete', label: '删除' },
  { value: 'dividend', label: '分红' },
  { value: 'adjustment', label: '调整' },
]
const transactionStatusOptions: Array<{ value: HoldingTransactionStatusFilter; label: string }> = [
  { value: 'all', label: '全部状态' },
  { value: 'active', label: '仅有效' },
  { value: 'voided', label: '仅作废' },
]

export function HoldingActivityTimeline({
  transactions,
  onVoidTransaction,
  onPreviewRollback,
  onApplyRollback,
  fundOptions = [],
  fundFilter = 'all',
  typeFilter = 'all',
  statusFilter = 'all',
  sourceFilter = 'all',
  keywordFilter = '',
  startDateFilter = '',
  endDateFilter = '',
  visibleLimit = 6,
  canLoadMore = false,
  onFundFilterChange,
  onTypeFilterChange,
  onStatusFilterChange,
  onSourceFilterChange,
  onKeywordFilterChange,
  onStartDateFilterChange,
  onEndDateFilterChange,
  onLoadMore,
  onClearFilters,
}: HoldingActivityTimelineProps) {
  const [pendingVoidID, setPendingVoidID] = useState<string | null>(null)
  const [pendingPreviewID, setPendingPreviewID] = useState<string | null>(null)
  const [pendingApplyID, setPendingApplyID] = useState<string | null>(null)
  const [rollbackPreview, setRollbackPreview] = useState<HoldingTransactionRollbackPreview | null>(null)
  const [voidFeedback, setVoidFeedback] = useState<string | null>(null)
  const [previewFeedback, setPreviewFeedback] = useState<string | null>(null)

  const latest = transactions[0] ?? null
  const displayTransactions = transactions.slice(0, visibleLimit)
  const activeCount = transactions.filter((transaction) => !transaction.voided).length
  const voidedCount = transactions.length - activeCount
  const hasActiveFilter = fundFilter !== 'all' ||
    typeFilter !== 'all' ||
    statusFilter !== 'all' ||
    sourceFilter !== 'all' ||
    keywordFilter.trim() !== '' ||
    startDateFilter !== '' ||
    endDateFilter !== ''
  const filterControlClass = 'rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2 text-xs font-medium text-theme-secondary outline-none transition focus:border-cyan-300/50'

  const handleClearFilters = () => {
    setRollbackPreview(null)
    setPreviewFeedback(null)
    onClearFilters?.()
  }

  const handleVoidTransaction = async (transaction: HoldingTransactionEntry) => {
    if (!onVoidTransaction || transaction.voided || pendingVoidID) {
      return
    }

    const reason = window.prompt(
      '请输入作废原因。作废只会标记这条历史流水无效，不会自动回滚或修改当前持仓快照。',
      transaction.note ? `原备注：${transaction.note}` : '录入错误，保留流水痕迹'
    )
    if (reason === null) {
      return
    }

    setVoidFeedback(null)
    setPendingVoidID(transaction.id)
    try {
      await onVoidTransaction(transaction.id, reason.trim() || '用户标记该流水无效')
      setVoidFeedback('已标记为作废；当前持仓快照不会自动回滚，如金额/份额也需要调整，请使用持仓校正。')
    } catch (error) {
      setVoidFeedback(error instanceof Error ? error.message : '作废流水失败，请稍后重试。')
    } finally {
      setPendingVoidID(null)
    }
  }

  const handlePreviewRollback = async (transaction: HoldingTransactionEntry) => {
    if (!onPreviewRollback || pendingPreviewID) {
      return
    }

    setPreviewFeedback(null)
    setPendingPreviewID(transaction.id)
    try {
      const preview = await onPreviewRollback(transaction.id)
      setRollbackPreview(preview ?? null)
      if (!preview) {
        setPreviewFeedback('未获取到回滚预览，请稍后重试。')
      }
    } catch (error) {
      setPreviewFeedback(error instanceof Error ? error.message : '获取回滚预览失败，请稍后重试。')
    } finally {
      setPendingPreviewID(null)
    }
  }

  const handleApplyRollback = async (preview: HoldingTransactionRollbackPreview) => {
    if (!onApplyRollback || !preview.can_apply_automatically || pendingApplyID) {
      return
    }

    const reason = window.prompt(
      '确认自动冲正？系统会先作废原流水，再按安全规则更新当前持仓快照。请输入原因：',
      '确认该流水录错，自动冲正'
    )
    if (reason === null) {
      return
    }

    setPreviewFeedback(null)
    setPendingApplyID(preview.transaction.id)
    try {
      const result = await onApplyRollback(preview.transaction.id, reason.trim() || '确认自动冲正')
      setPreviewFeedback(result?.message || '自动冲正已完成。')
      setRollbackPreview(result?.preview ?? null)
    } catch (error) {
      setPreviewFeedback(error instanceof Error ? error.message : '自动冲正失败，请稍后重试。')
    } finally {
      setPendingApplyID(null)
    }
  }

  return (
    <section className="mb-6 overflow-hidden rounded-[30px] border border-[var(--card-border)] bg-[var(--card-bg)]/84 p-5 glass">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <div className="inline-flex items-center gap-2 rounded-full border border-amber-400/25 bg-amber-400/10 px-3 py-1 text-[11px] font-medium tracking-[0.2em] text-amber-100">
            <History className="h-3.5 w-3.5" />
            持仓流水
          </div>
          <h3 className="mt-3 text-2xl font-black text-theme-primary">最近账本活动</h3>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-theme-secondary">
            买入、卖出、分红、份额调整、校正、删除都会保留可追溯流水；低风险录错流水可在预览后自动冲正，高风险流水仍建议人工校正。
          </p>
        </div>

        <div className="rounded-[24px] border border-amber-400/20 bg-amber-400/10 px-4 py-3 lg:min-w-[260px]">
          <div className="text-xs text-theme-muted">{latest ? '最近一次' : '当前筛选'}</div>
          <div className="mt-1 truncate text-lg font-black text-theme-primary">
            {latest ? (latest.fund?.name || latest.fund_id) : '暂无匹配流水'}
          </div>
          <div className="mt-1 text-xs text-amber-100">
            {latest
              ? `${transactionMeta(latest.type).label} · ${latest.voided ? '已作废 · ' : ''}${formatDateTime(latest.created_at)}`
              : hasActiveFilter
                ? '可清空筛选查看全部最近活动'
                : '记录买入、校正或卖出后会出现在这里'}
          </div>
        </div>
      </div>

      <div className="mt-5 space-y-3 rounded-[24px] border border-[var(--card-border)] bg-[var(--input-bg)]/42 p-3">
        <div className="grid gap-3 lg:grid-cols-[1fr_1fr_1fr_1fr_auto] lg:items-center">
          <label className="grid gap-1 text-xs text-theme-muted">
            基金
            <select
              value={fundFilter}
              onChange={(event) => onFundFilterChange?.(event.target.value)}
              className={filterControlClass}
            >
              <option value="all">全部基金</option>
              {fundOptions.map((fund) => (
                <option key={fund.fund_id} value={fund.fund_id}>
                  {fund.name || fund.fund_id}
                </option>
              ))}
            </select>
          </label>
          <label className="grid gap-1 text-xs text-theme-muted">
            类型
            <select
              value={typeFilter}
              onChange={(event) => onTypeFilterChange?.(event.target.value as HoldingTransactionType | 'all')}
              className={filterControlClass}
            >
              {transactionTypeOptions.map((option) => (
                <option key={option.value} value={option.value}>{option.label}</option>
              ))}
            </select>
          </label>
          <label className="grid gap-1 text-xs text-theme-muted">
            状态
            <select
              value={statusFilter}
              onChange={(event) => onStatusFilterChange?.(event.target.value as HoldingTransactionStatusFilter)}
              className={filterControlClass}
            >
              {transactionStatusOptions.map((option) => (
                <option key={option.value} value={option.value}>{option.label}</option>
              ))}
            </select>
          </label>
          <label className="grid gap-1 text-xs text-theme-muted">
            来源
            <select
              value={sourceFilter}
              onChange={(event) => onSourceFilterChange?.(event.target.value as HoldingSourceFilter)}
              className={filterControlClass}
            >
              <option value="all">全部来源</option>
              {HOLDING_SOURCE_OPTIONS.map((option) => (
                <option key={option.value} value={option.value}>{option.label}</option>
              ))}
            </select>
          </label>
          <button
            type="button"
            onClick={handleClearFilters}
            disabled={!hasActiveFilter}
            className="self-end rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-2 text-xs font-medium text-theme-secondary transition hover:border-cyan-300/45 hover:text-theme-primary disabled:cursor-not-allowed disabled:opacity-45 md:self-auto"
          >
            清空筛选
          </button>
        </div>

        <div className="grid gap-3 md:grid-cols-[minmax(0,1.4fr)_1fr_1fr]">
          <label className="grid gap-1 text-xs text-theme-muted">
            关键词
            <span className="relative">
              <Search className="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-theme-muted" />
              <input
                value={keywordFilter}
                onChange={(event) => onKeywordFilterChange?.(event.target.value)}
                placeholder="搜索基金代码、备注、来源或作废原因"
                className={cn(filterControlClass, 'w-full pl-8')}
              />
            </span>
          </label>
          <label className="grid gap-1 text-xs text-theme-muted">
            开始日期
            <span className="relative">
              <CalendarDays className="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-theme-muted" />
              <input
                type="date"
                value={startDateFilter}
                onChange={(event) => onStartDateFilterChange?.(event.target.value)}
                className={cn(filterControlClass, 'w-full pl-8')}
              />
            </span>
          </label>
          <label className="grid gap-1 text-xs text-theme-muted">
            结束日期
            <span className="relative">
              <CalendarDays className="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-theme-muted" />
              <input
                type="date"
                value={endDateFilter}
                onChange={(event) => onEndDateFilterChange?.(event.target.value)}
                className={cn(filterControlClass, 'w-full pl-8')}
              />
            </span>
          </label>
        </div>
      </div>

      {(voidFeedback || previewFeedback) && (
        <div className="mt-4 rounded-[20px] border border-slate-400/20 bg-slate-400/10 px-4 py-3 text-xs leading-5 text-theme-secondary">
          {voidFeedback || previewFeedback}
        </div>
      )}

      {rollbackPreview && (
        <div className="mt-4 rounded-[26px] border border-cyan-300/22 bg-cyan-400/10 p-4 shadow-[0_18px_60px_rgba(34,211,238,0.08)]">
          <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
            <div>
              <div className="inline-flex items-center gap-2 rounded-full border border-cyan-300/25 bg-cyan-400/12 px-3 py-1 text-[11px] font-semibold text-cyan-100">
                <RotateCcw className="h-3.5 w-3.5" />
                回滚影响预览
              </div>
              <h4 className="mt-3 text-lg font-black text-theme-primary">{rollbackPreview.title}</h4>
              <p className="mt-2 text-sm leading-6 text-theme-secondary">{rollbackPreview.summary}</p>
              <p className="mt-2 text-xs leading-5 text-cyan-100">{rollbackPreview.suggested_action}</p>
            </div>
            <button
              type="button"
              onClick={() => setRollbackPreview(null)}
              className="rounded-2xl border border-cyan-300/20 bg-cyan-400/10 px-3 py-2 text-xs font-medium text-cyan-100 transition hover:border-cyan-200/45 hover:bg-cyan-400/16"
            >
              收起预览
            </button>
          </div>

          <div className="mt-4 flex flex-col gap-3 rounded-[20px] border border-white/10 bg-[var(--input-bg)]/44 px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <div className="text-sm font-semibold text-theme-primary">
                {rollbackPreview.can_apply_automatically ? '这条流水可安全自动冲正' : '这条流水仅支持人工校正建议'}
              </div>
              <div className="mt-1 text-xs leading-5 text-theme-secondary">
                自动冲正只在没有后续有效流水、且当前快照可安全计算时开放；系统会保留原流水作废痕迹和冲正记录。
              </div>
            </div>
            {rollbackPreview.can_apply_automatically && onApplyRollback && (
              <button
                type="button"
                onClick={() => void handleApplyRollback(rollbackPreview)}
                disabled={pendingApplyID !== null}
                className="rounded-2xl border border-emerald-300/28 bg-emerald-400/12 px-4 py-2 text-xs font-semibold text-emerald-100 transition hover:border-emerald-200/55 hover:bg-emerald-400/18 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {pendingApplyID === rollbackPreview.transaction.id ? '冲正中...' : '应用自动冲正'}
              </button>
            )}
          </div>

          {rollbackPreview.affected_fields.length > 0 && (
            <div className="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-3">
              {rollbackPreview.affected_fields.map((field) => (
                <div key={`${field.field}-${field.label}`} className="rounded-[20px] border border-white/10 bg-[var(--input-bg)]/56 p-3">
                  <div className="text-xs text-theme-muted">{field.label}</div>
                  <div className="mt-2 grid grid-cols-2 gap-2 text-xs">
                    <div>
                      <div className="text-theme-muted">当前值</div>
                      <div className="mt-1 break-all font-semibold text-theme-primary">{formatRollbackValue(field.current_value)}</div>
                    </div>
                    <div>
                      <div className="text-theme-muted">建议回到</div>
                      <div className="mt-1 break-all font-semibold text-cyan-100">{formatRollbackValue(field.rollback_value)}</div>
                    </div>
                  </div>
                  {field.delta && (
                    <div className="mt-2 rounded-2xl border border-amber-300/18 bg-amber-400/10 px-3 py-2 text-xs text-amber-100">
                      {field.direction || '变化'}：{field.delta}
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}

          {rollbackPreview.warnings && rollbackPreview.warnings.length > 0 && (
            <div className="mt-4 space-y-2">
              {rollbackPreview.warnings.map((warning) => (
                <div key={warning} className="rounded-2xl border border-amber-300/18 bg-amber-400/10 px-3 py-2 text-xs leading-5 text-amber-100">
                  {warning}
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      <div className="mt-5 grid gap-4 xl:grid-cols-[0.85fr_1.15fr]">
        <div className="rounded-[24px] border border-[var(--card-border)] bg-[var(--input-bg)]/54 p-4">
          <div className="flex items-center justify-between gap-3">
            <div>
              <div className="text-sm font-semibold text-theme-primary">活动覆盖</div>
              <div className="mt-1 text-xs text-theme-muted">当前筛选 {transactions.length} 条记录，{activeCount} 条有效</div>
            </div>
            <div className="text-3xl font-black text-theme-primary">{transactions.length}</div>
          </div>
          <div className="mt-4 grid grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-cols-2">
            {activityCoverageTypes.map((type) => {
              const meta = transactionMeta(type)
              const count = transactions.filter((transaction) => transaction.type === type && !transaction.voided).length
              return (
                <div key={type} className={cn('rounded-[18px] border px-3 py-3 text-center', meta.className)}>
                  <div className="text-2xl font-black text-theme-primary">{count}</div>
                  <div className="mt-1 text-[11px]">{meta.label}</div>
                </div>
              )
            })}
          </div>
          <div className="mt-3 rounded-[18px] border border-slate-400/15 bg-slate-400/8 px-3 py-3 text-xs text-theme-muted">
            已作废 {voidedCount} 条；低风险流水可在预览后自动冲正，高风险流水仍只给人工校正建议。
          </div>
        </div>

        {transactions.length === 0 ? (
          <div className="flex min-h-[260px] flex-col items-center justify-center rounded-[24px] border border-dashed border-[var(--card-border)] bg-[var(--input-bg)]/40 px-6 py-10 text-center">
            <History className="h-10 w-10 text-theme-muted" />
            <div className="mt-3 text-lg font-black text-theme-primary">当前筛选没有流水</div>
            <p className="mt-2 max-w-md text-sm leading-6 text-theme-secondary">
              可以切换基金、类型、日期或关键词继续查找；作废流水不会修改当前持仓快照，只用于降低历史流水可信度。
            </p>
            {hasActiveFilter && (
              <button
                type="button"
                onClick={handleClearFilters}
                className="mt-4 rounded-2xl border border-cyan-300/25 bg-cyan-400/10 px-4 py-2 text-xs font-semibold text-cyan-100 transition hover:border-cyan-200/45 hover:bg-cyan-400/16"
              >
                查看全部最近活动
              </button>
            )}
          </div>
        ) : (
          <div className="space-y-4">
            <div className="relative space-y-3 before:absolute before:left-[18px] before:top-3 before:h-[calc(100%-24px)] before:w-px before:bg-gradient-to-b before:from-cyan-300/35 before:via-white/12 before:to-transparent">
              {displayTransactions.map((transaction) => {
                const meta = transactionMeta(transaction.type)
                const Icon = meta.icon
                const isVoided = Boolean(transaction.voided)
                const sourceLabel = resolveHoldingSourceLabel(transaction.source_platform, transaction.source_label)
                return (
                  <div key={transaction.id} className="relative pl-11">
                    <span className={cn(
                      'absolute left-[10px] top-5 h-4 w-4 rounded-full ring-4 ring-[var(--card-bg)]',
                      isVoided ? 'bg-slate-400 shadow-none' : meta.dotClassName
                    )} />
                    <div className={cn(
                      'rounded-[24px] border border-[var(--card-border)] bg-[var(--input-bg)]/58 p-4 transition-transform duration-200 hover:-translate-y-0.5',
                      isVoided && 'border-slate-400/20 bg-slate-500/8 opacity-70'
                    )}>
                      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                        <div className="min-w-0">
                          <div className="flex items-center gap-2">
                            <span className={cn(
                              'inline-flex h-8 w-8 items-center justify-center rounded-2xl border',
                              isVoided ? 'border-slate-400/30 bg-slate-400/10 text-slate-200' : meta.className
                            )}>
                              {isVoided ? <Ban className="h-4 w-4" /> : <Icon className="h-4 w-4" />}
                            </span>
                            <div className="min-w-0">
                              <div className="flex flex-wrap items-center gap-2">
                                <div className="truncate text-sm font-semibold text-theme-primary">{transaction.fund?.name || transaction.fund_id}</div>
                                <span className={cn('rounded-full border px-2 py-0.5 text-[10px] font-semibold', meta.className)}>
                                  {meta.label}
                                </span>
                                {isVoided && (
                                  <span className="rounded-full border border-slate-400/25 bg-slate-400/10 px-2 py-0.5 text-[10px] font-semibold text-slate-200">
                                    已作废
                                  </span>
                                )}
                                {sourceLabel && (
                                  <span className="rounded-full border border-cyan-300/20 bg-cyan-400/10 px-2 py-0.5 text-[10px] font-semibold text-cyan-100">
                                    来源：{sourceLabel}
                                  </span>
                                )}
                              </div>
                              <div className="mt-0.5 text-xs text-theme-muted">
                                {transaction.fund_id} · {formatDateTime(transaction.created_at)}
                                {isVoided && transaction.voided_at ? ` · 作废于 ${formatDateTime(transaction.voided_at)}` : ''}
                              </div>
                            </div>
                          </div>
                          <p className="mt-3 text-xs leading-5 text-theme-secondary">{transaction.note || meta.description}</p>
                          {isVoided && transaction.void_reason && (
                            <p className="mt-2 rounded-2xl border border-slate-400/15 bg-slate-400/8 px-3 py-2 text-xs leading-5 text-theme-muted">
                              作废原因：{transaction.void_reason}
                            </p>
                          )}
                        </div>

                        <div className="min-w-[210px] space-y-3">
                          <div className="grid grid-cols-2 gap-2 text-xs sm:text-right">
                            <div>
                              <div className="text-theme-muted">金额</div>
                              <div className="mt-0.5 font-semibold text-theme-primary">{formatSummaryMoney(transaction.amount)}</div>
                            </div>
                            <div>
                              <div className="text-theme-muted">确认日</div>
                              <div className="mt-0.5 font-semibold text-theme-primary">{formatCompactDate(transaction.confirmed_nav_date || transaction.as_of_date)}</div>
                            </div>
                          </div>
                          <div className="grid gap-2 sm:grid-cols-3">
                            <Link
                              href={`/holdings/transactions/${transaction.id}`}
                              className="rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2 text-center text-xs font-medium text-theme-secondary transition hover:border-cyan-300/35 hover:text-theme-primary"
                            >
                              查看详情
                            </Link>
                            {onPreviewRollback && (
                              <button
                                type="button"
                                onClick={() => void handlePreviewRollback(transaction)}
                                disabled={pendingPreviewID !== null}
                                className="rounded-2xl border border-cyan-300/20 bg-cyan-400/10 px-3 py-2 text-xs font-medium text-cyan-100 transition hover:border-cyan-200/45 hover:bg-cyan-400/16 disabled:cursor-not-allowed disabled:opacity-60"
                              >
                                {pendingPreviewID === transaction.id ? '预览中...' : '回滚预览'}
                              </button>
                            )}
                            {onVoidTransaction && !isVoided && (
                              <button
                                type="button"
                                onClick={() => void handleVoidTransaction(transaction)}
                                disabled={pendingVoidID !== null}
                                className="rounded-2xl border border-slate-400/20 bg-slate-400/8 px-3 py-2 text-xs font-medium text-theme-secondary transition hover:border-rose-300/35 hover:bg-rose-400/10 hover:text-rose-100 disabled:cursor-not-allowed disabled:opacity-60"
                              >
                                {pendingVoidID === transaction.id ? '标记中...' : '作废流水'}
                              </button>
                            )}
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                )
              })}
            </div>

            {canLoadMore && (
              <button
                type="button"
                onClick={onLoadMore}
                className="ml-11 w-[calc(100%-2.75rem)] rounded-2xl border border-cyan-300/20 bg-cyan-400/10 px-4 py-3 text-xs font-semibold text-cyan-100 transition hover:border-cyan-200/45 hover:bg-cyan-400/16"
              >
                加载更多流水
              </button>
            )}
            {!canLoadMore && transactions.length > visibleLimit && (
              <div className="pl-11 text-xs text-theme-muted">已展示当前筛选下前 {displayTransactions.length} 条流水。</div>
            )}
          </div>
        )}
      </div>
    </section>
  )
}
