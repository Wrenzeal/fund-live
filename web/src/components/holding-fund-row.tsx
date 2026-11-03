'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { Gift, MinusCircle, Pencil, Save, SlidersHorizontal, Trash2, X } from 'lucide-react'
import { useFund, type FundEstimate } from '@/hooks/use-fund-data'
import {
  HOLDING_SOURCE_OPTIONS,
  resolveHoldingSourceLabel,
  type HoldingSourcePlatform,
} from '@/lib/holding-sources'
import { cn } from '@/lib/utils'
import { resolveHoldingMetrics } from '@/lib/holding-display'
import { holdingMoney, holdingTone } from '@/components/holdings-summary-metrics'

import {
  buildAdjustmentFormState,
  buildCorrectionFormState,
  buildCorrectionTradeAt,
  buildDividendFormState,
  buildSellFormState,
  formatAmount,
  formatTradeAt,
  normalizeDecimalInput,
  validateNonZeroDecimal,
  validatePositiveDecimal,
  type AdjustmentFormState,
  type CorrectionFormState,
  type DividendFormState,
  type SellFormState,
} from '@/lib/holding-row-format'
import type {
  AdjustHoldingSharesPayload,
  DividendHoldingPayload,
  HoldingEntry,
  SellHoldingPayload,
  UpdateHoldingPayload,
} from '@/hooks/use-user-portfolio'

interface HoldingFundRowProps {
  holding: HoldingEntry
  metricScope?: 'official' | 'estimate'
  estimate?: FundEstimate | null
  onRemove: () => Promise<void> | void
  onUpdate: (payload: UpdateHoldingPayload) => Promise<void> | void
  onSell: (payload: SellHoldingPayload) => Promise<void> | void
  onRecordDividend: (payload: DividendHoldingPayload) => Promise<void> | void
  onAdjustShares: (payload: AdjustHoldingSharesPayload) => Promise<void> | void
}

export function HoldingFundRow({
  holding,
  metricScope = 'official',
  estimate: providedEstimate,
  onRemove,
  onUpdate,
  onSell,
  onRecordDividend,
  onAdjustShares,
}: HoldingFundRowProps) {
  const [isRemoving, setIsRemoving] = useState(false)
  const [isEditing, setIsEditing] = useState(false)
  const [isSelling, setIsSelling] = useState(false)
  const [isDividendOpen, setIsDividendOpen] = useState(false)
  const [isAdjustmentOpen, setIsAdjustmentOpen] = useState(false)
  const [isSaving, setIsSaving] = useState(false)
  const [isSellSaving, setIsSellSaving] = useState(false)
  const [isDividendSaving, setIsDividendSaving] = useState(false)
  const [isAdjustmentSaving, setIsAdjustmentSaving] = useState(false)
  const [form, setForm] = useState<CorrectionFormState>(() => buildCorrectionFormState(holding))
  const [sellForm, setSellForm] = useState<SellFormState>(() => buildSellFormState(holding))
  const [dividendForm, setDividendForm] = useState<DividendFormState>(() => buildDividendFormState(holding))
  const [adjustmentForm, setAdjustmentForm] = useState<AdjustmentFormState>(() => buildAdjustmentFormState(holding))
  const [formError, setFormError] = useState('')
  const [sellError, setSellError] = useState('')
  const [dividendError, setDividendError] = useState('')
  const [adjustmentError, setAdjustmentError] = useState('')
  const { fund } = useFund(holding.fund_id)
  const estimate = providedEstimate ?? null
  const fundName = holding.fund?.name || fund?.name || estimate?.fund_name || holding.fund_id
  const isOpening = holding.entry_type === 'opening'
  const tradeAtLabel = formatTradeAt(holding.trade_at)
  const sourceLabel = resolveHoldingSourceLabel(holding.source_platform, holding.source_label)
  const metrics = resolveHoldingMetrics(holding, estimate, metricScope)
  const confirmedDateLabel = holding.confirmed_nav_date || holding.as_of_date || '待确认'
  const metricNote = metrics.ready
    ? metricScope === 'official' ? '官方净值日 ' + (holding.actual_date || '待更新') + (holding.real_metrics_stale ? ' · 新净值待更新' : '') : '按确认份额与盘中预估净值计算'
    : metricScope === 'official' ? holding.real_metrics_message || '等待净值更新' : '确认份额或预估净值不足'
  const actionDisabled = isRemoving || isSaving || isSellSaving || isDividendSaving || isAdjustmentSaving

  useEffect(() => {
    if (!isEditing) {
      setForm(buildCorrectionFormState(holding))
      setFormError('')
    }
  }, [holding, isEditing])

  useEffect(() => {
    if (!isSelling) {
      setSellForm(buildSellFormState(holding))
      setSellError('')
    }
  }, [holding, isSelling])

  useEffect(() => {
    if (!isDividendOpen) {
      setDividendForm(buildDividendFormState(holding))
      setDividendError('')
    }
  }, [holding, isDividendOpen])

  useEffect(() => {
    if (!isAdjustmentOpen) {
      setAdjustmentForm(buildAdjustmentFormState(holding))
      setAdjustmentError('')
    }
  }, [holding, isAdjustmentOpen])

  const updateForm = (patch: Partial<CorrectionFormState>) => {
    setForm((current) => ({ ...current, ...patch }))
    setFormError('')
  }

  const updateSellForm = (patch: Partial<SellFormState>) => {
    setSellForm((current) => ({ ...current, ...patch }))
    setSellError('')
  }

  const updateDividendForm = (patch: Partial<DividendFormState>) => {
    setDividendForm((current) => ({ ...current, ...patch }))
    setDividendError('')
  }

  const updateAdjustmentForm = (patch: Partial<AdjustmentFormState>) => {
    setAdjustmentForm((current) => ({ ...current, ...patch }))
    setAdjustmentError('')
  }

  const handleRemove = async () => {
    if (actionDisabled) {
      return
    }

    setIsRemoving(true)

    try {
      await new Promise((resolve) => window.setTimeout(resolve, 180))
      await Promise.resolve(onRemove())
    } catch (error) {
      console.error('Failed to remove holding', error)
      setIsRemoving(false)
    }
  }

  const handleSaveCorrection = async () => {
    if (isSaving) {
      return
    }

    const normalizedAmount = normalizeDecimalInput(form.amount)
    const normalizedShares = normalizeDecimalInput(form.shares)
    const normalizedNav = normalizeDecimalInput(form.confirmedNav)
    const hasAnyConfirmation = !isOpening && Boolean(normalizedShares || normalizedNav || form.confirmedNavDate)
    const hasFullConfirmation = Boolean(normalizedShares && normalizedNav && form.confirmedNavDate)

    if (!validatePositiveDecimal(normalizedAmount)) {
      setFormError('请输入有效的持仓本金。')
      return
    }

    if (isOpening && !validatePositiveDecimal(normalizedShares)) { setFormError('请输入有效的持有份额。'); return }
    if (hasAnyConfirmation && !hasFullConfirmation) {
      setFormError('如果要按支付宝/微信校正确认口径，需要同时填写份额、确认净值和净值日。')
      return
    }

    if (normalizedShares && !validatePositiveDecimal(normalizedShares)) {
      setFormError('请输入有效的确认份额。')
      return
    }

    if (normalizedNav && !validatePositiveDecimal(normalizedNav)) {
      setFormError('请输入有效的确认净值。')
      return
    }

    setIsSaving(true)
    try {
      await Promise.resolve(onUpdate({
        amount: normalizedAmount,
        shares: normalizedShares,
        confirmed_nav: normalizedNav,
        confirmed_nav_date: form.confirmedNavDate,
        snapshot_date: isOpening ? form.tradeDate : undefined,
        trade_at: isOpening ? undefined : buildCorrectionTradeAt(form.tradeDate, holding.trade_at),
        note: form.note.trim(),
        source_platform: form.sourcePlatform || undefined,
      }))
      setIsEditing(false)
    } catch (error) {
      setFormError(error instanceof Error ? error.message : '校正失败，请稍后重试。')
    } finally {
      setIsSaving(false)
    }
  }

  const handleSaveSell = async () => {
    if (isSellSaving) {
      return
    }

    const normalizedAmount = normalizeDecimalInput(sellForm.amount)
    const normalizedShares = normalizeDecimalInput(sellForm.shares)
    const amountValue = normalizedAmount ? Number.parseFloat(normalizedAmount) : null
    const sharesValue = normalizedShares ? Number.parseFloat(normalizedShares) : null
    const currentAmount = Number.parseFloat(holding.amount)
    const currentShares = Number.parseFloat(holding.shares || '')

    if (!sellForm.sellAll && !normalizedAmount && !normalizedShares) {
      setSellError('请输入卖出份额或减少的成本本金；如果是全部赎回，请开启“全部清仓”。')
      return
    }

    if (normalizedAmount && !validatePositiveDecimal(normalizedAmount)) {
      setSellError('请输入有效的减少成本本金。')
      return
    }

    if (normalizedShares && !validatePositiveDecimal(normalizedShares)) {
      setSellError('请输入有效的卖出份额。')
      return
    }

    if (!sellForm.sellAll && amountValue !== null && Number.isFinite(currentAmount) && amountValue >= currentAmount) {
      setSellError('减少的成本本金需要小于当前剩余成本；如果是全部赎回，请开启“全部清仓”。')
      return
    }

    if (sharesValue !== null) {
      if (!Number.isFinite(currentShares) || currentShares <= 0) {
        setSellError('当前持仓还没有确认份额，暂时只能按减少成本本金记录。')
        return
      }
      if (!sellForm.sellAll && sharesValue >= currentShares) {
        setSellError('部分减仓份额需要小于当前确认份额；如果是全部赎回，请开启“全部清仓”。')
        return
      }
    }

    setIsSellSaving(true)
    try {
      await Promise.resolve(onSell({
        amount: sellForm.sellAll ? undefined : normalizedAmount,
        shares: sellForm.sellAll ? undefined : normalizedShares,
        trade_at: buildCorrectionTradeAt(sellForm.tradeDate, undefined),
        note: sellForm.note.trim(),
        sell_all: sellForm.sellAll,
      }))
      setIsSelling(false)
    } catch (error) {
      setSellError(error instanceof Error ? error.message : '减仓失败，请稍后重试。')
    } finally {
      setIsSellSaving(false)
    }
  }

  const handleSaveDividend = async () => {
    if (isDividendSaving) {
      return
    }

    const normalizedAmount = normalizeDecimalInput(dividendForm.amount)
    const normalizedShares = normalizeDecimalInput(dividendForm.shares)

    if (!validatePositiveDecimal(normalizedAmount)) {
      setDividendError('请输入有效的分红金额。')
      return
    }

    if (normalizedShares && !validatePositiveDecimal(normalizedShares)) {
      setDividendError('请输入有效的红利再投份额。')
      return
    }

    if (dividendForm.reinvest && !normalizedShares) {
      setDividendError('红利再投需要填写新增份额；现金分红可以关闭红利再投。')
      return
    }

    setIsDividendSaving(true)
    try {
      await Promise.resolve(onRecordDividend({
        amount: normalizedAmount,
        shares: normalizedShares || undefined,
        trade_at: buildCorrectionTradeAt(dividendForm.tradeDate, undefined),
        note: dividendForm.note.trim(),
        reinvest: dividendForm.reinvest,
        source_platform: dividendForm.sourcePlatform || undefined,
      }))
      setIsDividendOpen(false)
    } catch (error) {
      setDividendError(error instanceof Error ? error.message : '记录分红失败，请稍后重试。')
    } finally {
      setIsDividendSaving(false)
    }
  }

  const handleSaveAdjustment = async () => {
    if (isAdjustmentSaving) {
      return
    }

    const normalizedTargetShares = normalizeDecimalInput(adjustmentForm.targetShares)
    const normalizedSharesDelta = normalizeDecimalInput(adjustmentForm.sharesDelta)
    const normalizedNav = normalizeDecimalInput(adjustmentForm.confirmedNav)
    const hasTarget = Boolean(normalizedTargetShares)
    const hasDelta = Boolean(normalizedSharesDelta)
    const hasAnyNav = Boolean(normalizedNav || adjustmentForm.confirmedNavDate)
    const hasFullNav = Boolean(normalizedNav && adjustmentForm.confirmedNavDate)

    if (hasTarget === hasDelta) {
      setAdjustmentError('请只填写“目标份额”或“份额差额”其中一种。')
      return
    }

    if (hasTarget && !validatePositiveDecimal(normalizedTargetShares)) {
      setAdjustmentError('请输入有效的目标份额。')
      return
    }

    if (hasDelta && !validateNonZeroDecimal(normalizedSharesDelta)) {
      setAdjustmentError('请输入有效且不为 0 的份额差额。')
      return
    }

    if (hasAnyNav && !hasFullNav) {
      setAdjustmentError('如果要同步确认净值，需要同时填写确认净值和确认净值日。')
      return
    }

    if (normalizedNav && !validatePositiveDecimal(normalizedNav)) {
      setAdjustmentError('请输入有效的确认净值。')
      return
    }

    setIsAdjustmentSaving(true)
    try {
      await Promise.resolve(onAdjustShares({
        target_shares: hasTarget ? normalizedTargetShares : undefined,
        shares_delta: hasDelta ? normalizedSharesDelta : undefined,
        confirmed_nav: normalizedNav || undefined,
        confirmed_nav_date: adjustmentForm.confirmedNavDate || undefined,
        trade_at: buildCorrectionTradeAt(adjustmentForm.tradeDate, undefined),
        note: adjustmentForm.note.trim(),
        source_platform: adjustmentForm.sourcePlatform || undefined,
      }))
      setIsAdjustmentOpen(false)
    } catch (error) {
      setAdjustmentError(error instanceof Error ? error.message : '份额调整失败，请稍后重试。')
    } finally {
      setIsAdjustmentSaving(false)
    }
  }

  return (
    <div data-testid="holding-detail-row" className="space-y-3">
      <div className="space-y-4 rounded-xl border border-[var(--card-border)] p-4 sm:p-5">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="min-w-0">
            <div className="break-words text-base font-semibold text-theme-primary">{fundName}</div>
            <div className="mt-1 text-xs text-theme-secondary">{holding.fund_id} · {isOpening ? '已有持仓 · 登记于 ' + holding.as_of_date : '买入于 ' + (tradeAtLabel || '日期待补充')}</div>
          </div>
          <Link href={'/observations/' + holding.fund_id} className="inline-flex min-h-11 items-center text-sm text-theme-secondary hover:text-theme-primary">公告与资料</Link>
        </div>
        <dl className="grid grid-cols-3 gap-3">
          <div><dt className="text-xs text-theme-secondary">持仓市值{metricScope === 'estimate' ? '（预估）' : ''}</dt><dd className="mt-2 break-words text-base font-semibold tabular-nums text-theme-primary sm:text-xl">{holdingMoney(metrics.marketValue)}</dd></div>
          <div><dt className="text-xs text-theme-secondary">持有盈亏</dt><dd className={cn('mt-2 break-words text-base font-semibold tabular-nums sm:text-xl', holdingTone(metrics.holdingProfit))}>{holdingMoney(metrics.holdingProfit, true)}</dd></div>
          <div><dt className="text-xs text-theme-secondary">当日变动{metricScope === 'estimate' ? '预估' : ''}</dt><dd className={cn('mt-2 break-words text-base font-semibold tabular-nums sm:text-xl', holdingTone(metrics.todayProfit))}>{holdingMoney(metrics.todayProfit, true)}</dd></div>
        </dl>
        <p className="text-xs leading-6 text-theme-secondary">{metricNote} · 剩余成本 {holdingMoney(metrics.principal)} · 确认份额 {holding.shares || '待确认'}{!isOpening && <> · 确认净值日 {confirmedDateLabel}</>}{sourceLabel ? ' · ' + sourceLabel : ''}{holding.manual_confirmation && !isOpening ? ' · 手动校正' : ''}</p>
        {holding.note && <p className="break-words text-sm text-theme-secondary">{holding.note}</p>}
        <div className="flex flex-wrap items-center gap-2 border-t border-[var(--card-border)] pt-3">

          <button
            type="button"
            onClick={() => {
              setIsEditing((current) => !current)
              if (!isEditing) {
                setIsSelling(false)
                setIsDividendOpen(false)
                setIsAdjustmentOpen(false)
              }
            }}
            disabled={actionDisabled}
            className={cn(
              'group relative inline-flex items-center justify-center overflow-hidden rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] min-h-11 min-w-11 gap-2 px-3 text-theme-secondary focus-visible:outline-2 focus-visible:outline-offset-2 transition-all duration-200',
              'hover:-translate-y-0.5 hover:border-accent/50 hover:bg-accent/12 hover:text-theme-primary',
              'active:scale-95 disabled:cursor-not-allowed',
              isEditing && 'border-accent/45 bg-accent/14 text-theme-primary'
            )}
            aria-label={`校正 ${fundName} 持仓`}
          >
            {isEditing ? <X className="h-4 w-4" /> : <Pencil className="h-4 w-4" />}<span className="text-xs">校正</span>
          </button>
          <button
            type="button"
            onClick={() => {
              setIsSelling((current) => !current)
              if (!isSelling) {
                setIsEditing(false)
                setIsDividendOpen(false)
                setIsAdjustmentOpen(false)
              }
            }}
            disabled={actionDisabled}
            className={cn(
              'group relative inline-flex items-center justify-center overflow-hidden rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] min-h-11 min-w-11 gap-2 px-3 text-theme-secondary focus-visible:outline-2 focus-visible:outline-offset-2 transition-all duration-200',
              'hover:-translate-y-0.5 hover:border-warning/50 hover:bg-warning/12 hover:text-theme-primary',
              'active:scale-95 disabled:cursor-not-allowed',
              isSelling && 'border-warning/45 bg-warning/14 text-theme-primary'
            )}
            aria-label={`卖出/减仓 ${fundName} 持仓`}
          >
            {isSelling ? <X className="h-4 w-4" /> : <MinusCircle className="h-4 w-4" />}<span className="text-xs">减仓</span>
          </button>
          <button
            type="button"
            onClick={() => {
              setIsDividendOpen((current) => !current)
              if (!isDividendOpen) {
                setIsEditing(false)
                setIsSelling(false)
                setIsAdjustmentOpen(false)
              }
            }}
            disabled={actionDisabled}
            className={cn(
              'group relative inline-flex items-center justify-center overflow-hidden rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] min-h-11 min-w-11 gap-2 px-3 text-theme-secondary focus-visible:outline-2 focus-visible:outline-offset-2 transition-all duration-200',
              'hover:-translate-y-0.5 hover:border-accent/50 hover:bg-accent/12 hover:text-theme-primary',
              'active:scale-95 disabled:cursor-not-allowed',
              isDividendOpen && 'border-accent/45 bg-accent/14 text-theme-primary'
            )}
            aria-label={`记录 ${fundName} 分红`}
          >
            {isDividendOpen ? <X className="h-4 w-4" /> : <Gift className="h-4 w-4" />}<span className="text-xs">分红</span>
          </button>
          <button
            type="button"
            onClick={() => {
              setIsAdjustmentOpen((current) => !current)
              if (!isAdjustmentOpen) {
                setIsEditing(false)
                setIsSelling(false)
                setIsDividendOpen(false)
              }
            }}
            disabled={actionDisabled}
            className={cn(
              'group relative inline-flex items-center justify-center overflow-hidden rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] min-h-11 min-w-11 gap-2 px-3 text-theme-secondary focus-visible:outline-2 focus-visible:outline-offset-2 transition-all duration-200',
              'hover:-translate-y-0.5 hover:border-accent/50 hover:bg-accent/12 hover:text-theme-primary',
              'active:scale-95 disabled:cursor-not-allowed',
              isAdjustmentOpen && 'border-accent/45 bg-accent/14 text-theme-primary'
            )}
            aria-label={`调整 ${fundName} 份额`}
          >
            {isAdjustmentOpen ? <X className="h-4 w-4" /> : <SlidersHorizontal className="h-4 w-4" />}<span className="text-xs">份额调整</span>
          </button>
          <button
            type="button"
            onClick={() => void handleRemove()}
            disabled={actionDisabled}
            className={cn(
              'group relative inline-flex items-center justify-center overflow-hidden rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] min-h-11 min-w-11 gap-2 px-3 text-theme-secondary focus-visible:outline-2 focus-visible:outline-offset-2 transition-all duration-200',
              'hover:-translate-y-0.5 hover:border-danger/50 hover:bg-danger/12 hover:text-danger',
              'active:scale-95 disabled:cursor-not-allowed',
              isRemoving && 'holding-delete-button border-danger/50 bg-danger/16 text-danger'
            )}
            aria-label={`移除 ${fundName} 持仓`}
            aria-busy={isRemoving}
          >
            <span
              className={cn(
                'pointer-events-none absolute inset-0 rounded-xl bg-danger/0 opacity-0 transition-opacity duration-200',
                'group-hover:opacity-100',
                isRemoving && 'opacity-100'
              )}
            />
            <Trash2
              className={cn(
                'relative z-10 h-4 w-4 transition-transform duration-300',
                'group-hover:-rotate-12 group-hover:scale-110',
                isRemoving && 'holding-delete-icon'
              )}
            />
          </button>
        </div>
      </div>

      {isEditing && (
        <div className="rounded-xl border border-accent/20 bg-accent/8 p-5 shadow-none backdrop-blur-xl">
          <div className="flex flex-col gap-2 md:flex-row md:items-start md:justify-between">
            <div>
              <div className="text-sm font-semibold text-theme-primary">校正持仓口径</div>
              <p className="mt-1 max-w-2xl text-xs leading-5 text-theme-muted">
                用于对齐支付宝、微信等外部平台显示差异；这不是买入/卖出记录，不会改变基金代码，只修正当前这笔持仓的金额、确认份额、确认净值和备注。
              </p>
            </div>
            <div className="rounded-xl border border-accent/20 bg-accent/10 px-3 py-2 text-xs text-theme-primary">
              留空份额/净值/日期会回到系统自动确认口径
            </div>
          </div>

          <div className="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-6">
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>持仓本金</span>
              <input
                value={form.amount}
                onChange={(event) => updateForm({ amount: event.target.value })}
                className="min-h-11 w-full rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-accent/60"
                inputMode="decimal"
                placeholder="例如 50000"
              />
            </label>
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>{isOpening ? '持有份额' : '确认份额'}</span>
              <input
                value={form.shares}
                onChange={(event) => updateForm({ shares: event.target.value })}
                className="min-h-11 w-full rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-accent/60"
                inputMode="decimal"
                placeholder="外部平台显示份额"
              />
            </label>
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>{isOpening ? '参考净值（无需填写）' : '确认净值'}</span>
              <input
                value={form.confirmedNav}
                disabled={isOpening}
                onChange={(event) => updateForm({ confirmedNav: event.target.value })}
                className="min-h-11 w-full rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-accent/60"
                inputMode="decimal"
                placeholder="例如 1.2345"
              />
            </label>
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>确认净值日</span>
              <input
                type="date"
                value={form.confirmedNavDate}
                disabled={isOpening}
                onChange={(event) => updateForm({ confirmedNavDate: event.target.value })}
                className="min-h-11 w-full rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-accent/60"
              />
            </label>
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>{isOpening ? '持仓登记日期' : '申购日期'}</span>
              <input
                type="date"
                value={form.tradeDate}
                onChange={(event) => updateForm({ tradeDate: event.target.value })}
                className="min-h-11 w-full rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-accent/60"
              />
            </label>
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>校正来源</span>
              <select
                value={form.sourcePlatform}
                onChange={(event) => updateForm({ sourcePlatform: event.target.value as HoldingSourcePlatform | '' })}
                className="min-h-11 w-full rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-accent/60"
              >
                <option value="">未记录来源</option>
                {HOLDING_SOURCE_OPTIONS.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
            </label>
          </div>

          {form.sourcePlatform && (
            <div className="mt-3 rounded-[20px] border border-accent/15 bg-accent/8 px-4 py-3 text-xs leading-5 text-theme-muted">
              {HOLDING_SOURCE_OPTIONS.find((option) => option.value === form.sourcePlatform)?.description}
              ；来源会写入持仓和校正流水，便于筛选与对账。
            </div>
          )}

          <label className="mt-3 block space-y-2 text-xs text-theme-secondary">
            <span>备注</span>
            <input
              value={form.note}
              onChange={(event) => updateForm({ note: event.target.value })}
              className="min-h-11 w-full rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-accent/60"
              placeholder="例如：按支付宝份额校正 / 微信显示净值差异"
            />
          </label>

          {formError && <div className="mt-3 text-xs text-danger">{formError}</div>}

          <div className="mt-4 flex flex-wrap items-center gap-3">
            <button
              type="button"
              onClick={() => void handleSaveCorrection()}
              disabled={isSaving}
              className="inline-flex items-center gap-2 rounded-xl bg-[var(--input-bg)] border border-[var(--input-border)] px-4 py-2.5 text-sm font-semibold text-theme-primary shadow-none transition-transform hover:-translate-y-0.5 active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-70"
            >
              <Save className={cn('h-4 w-4', isSaving && 'animate-pulse')} />
              {isSaving ? '保存中...' : '保存校正'}
            </button>
            <button
              type="button"
              onClick={() => {
                setIsEditing(false)
                setForm(buildCorrectionFormState(holding))
                setFormError('')
              }}
              className="rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-2.5 text-sm text-theme-secondary transition-colors hover:border-accent/35 hover:text-theme-primary"
            >
              取消
            </button>
            {!isOpening && (
            <button
              type="button"
              onClick={() => updateForm({ shares: '', confirmedNav: '', confirmedNavDate: '', sourcePlatform: '' })}
              className="rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-2.5 text-sm text-theme-muted transition-colors hover:border-warning/35 hover:text-theme-primary"
            >
              回到自动确认
            </button>)}
          </div>
        </div>
      )}
      {isDividendOpen && (
        <div className="rounded-xl border border-accent/20 bg-accent/8 p-5 shadow-none backdrop-blur-xl">
          <div className="flex flex-col gap-2 md:flex-row md:items-start md:justify-between">
            <div>
              <div className="text-sm font-semibold text-theme-primary">记录分红 / 红利再投</div>
              <p className="mt-1 max-w-2xl text-xs leading-5 text-theme-muted">
                现金分红只写入流水，不改变当前持仓份额；红利再投会把新增份额合并到当前持仓，并保留分红流水痕迹。
              </p>
            </div>
            <div className="rounded-xl border border-accent/20 bg-accent/10 px-3 py-2 text-xs text-theme-primary">
              当前份额：{holding.shares || '待确认'}
            </div>
          </div>

          <div className="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-5">
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>分红金额</span>
              <input
                value={dividendForm.amount}
                onChange={(event) => updateDividendForm({ amount: event.target.value })}
                className="min-h-11 w-full rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-accent/60"
                inputMode="decimal"
                placeholder="例如 125.00"
              />
            </label>
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>新增份额</span>
              <input
                value={dividendForm.shares}
                onChange={(event) => updateDividendForm({ shares: event.target.value })}
                className="min-h-11 w-full rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-accent/60"
                inputMode="decimal"
                placeholder="红利再投时填写"
              />
            </label>
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>分红日期</span>
              <input
                type="date"
                value={dividendForm.tradeDate}
                onChange={(event) => updateDividendForm({ tradeDate: event.target.value })}
                className="min-h-11 w-full rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-accent/60"
              />
            </label>
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>来源</span>
              <select
                value={dividendForm.sourcePlatform}
                onChange={(event) => updateDividendForm({ sourcePlatform: event.target.value as HoldingSourcePlatform | '' })}
                className="min-h-11 w-full rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-accent/60"
              >
                <option value="">未记录来源</option>
                {HOLDING_SOURCE_OPTIONS.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
            </label>
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>备注</span>
              <input
                value={dividendForm.note}
                onChange={(event) => updateDividendForm({ note: event.target.value })}
                className="min-h-11 w-full rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-accent/60"
                placeholder="例如：现金分红 / 红利再投"
              />
            </label>
          </div>

          <label className="mt-4 flex cursor-pointer items-start gap-3 rounded-[22px] border border-accent/20 bg-accent/8 p-4 text-sm text-theme-secondary transition-colors hover:border-accent/35">
            <input
              type="checkbox"
              checked={dividendForm.reinvest}
              onChange={(event) => updateDividendForm({ reinvest: event.target.checked })}
              className="mt-1 h-4 w-4 accent-violet-400"
            />
            <span>
              <span className="block font-semibold text-theme-primary">红利再投</span>
              <span className="mt-1 block text-xs leading-5 text-theme-muted">开启后必须填写新增份额；保存后会增加持仓份额。关闭时只记录现金分红流水。</span>
            </span>
          </label>

          {dividendError && <div className="mt-3 text-xs text-danger">{dividendError}</div>}

          <div className="mt-4 flex flex-wrap items-center gap-3">
            <button
              type="button"
              onClick={() => void handleSaveDividend()}
              disabled={isDividendSaving}
              className="inline-flex items-center gap-2 rounded-xl bg-[var(--input-bg)] border border-[var(--input-border)] px-4 py-2.5 text-sm font-semibold text-theme-primary shadow-none transition-transform hover:-translate-y-0.5 active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-70"
            >
              <Gift className={cn('h-4 w-4', isDividendSaving && 'animate-pulse')} />
              {isDividendSaving ? '记录中...' : dividendForm.reinvest ? '确认红利再投' : '确认现金分红'}
            </button>
            <button
              type="button"
              onClick={() => {
                setIsDividendOpen(false)
                setDividendForm(buildDividendFormState(holding))
                setDividendError('')
              }}
              className="rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-2.5 text-sm text-theme-secondary transition-colors hover:border-accent/35 hover:text-theme-primary"
            >
              取消
            </button>
          </div>
        </div>
      )}
      {isAdjustmentOpen && (
        <div className="rounded-xl border border-accent/20 bg-accent/8 p-5 shadow-none backdrop-blur-xl">
          <div className="flex flex-col gap-2 md:flex-row md:items-start md:justify-between">
            <div>
              <div className="text-sm font-semibold text-theme-primary">记录份额调整</div>
              <p className="mt-1 max-w-2xl text-xs leading-5 text-theme-muted">
                适合平台迁移、拆分合并、手续费或历史口径修正；不会作为买卖交易处理，但会更新当前持仓快照并写入调整流水。
              </p>
            </div>
            <div className="rounded-xl border border-accent/20 bg-accent/10 px-3 py-2 text-xs text-theme-primary">
              当前份额：{holding.shares || '待确认'}
            </div>
          </div>

          <div className="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-6">
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>目标份额</span>
              <input
                value={adjustmentForm.targetShares}
                onChange={(event) => updateAdjustmentForm({ targetShares: event.target.value, sharesDelta: '' })}
                className="min-h-11 w-full rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-accent/60"
                inputMode="decimal"
                placeholder="调整后的总份额"
              />
            </label>
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>份额差额</span>
              <input
                value={adjustmentForm.sharesDelta}
                onChange={(event) => updateAdjustmentForm({ sharesDelta: event.target.value, targetShares: '' })}
                className="min-h-11 w-full rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-accent/60"
                inputMode="decimal"
                placeholder="例如 100 或 -50"
              />
            </label>
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>{isOpening ? '参考净值（无需填写）' : '确认净值'}</span>
              <input
                value={adjustmentForm.confirmedNav}
                onChange={(event) => updateAdjustmentForm({ confirmedNav: event.target.value })}
                className="min-h-11 w-full rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-accent/60"
                inputMode="decimal"
                placeholder="可选"
              />
            </label>
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>净值日</span>
              <input
                type="date"
                value={adjustmentForm.confirmedNavDate}
                onChange={(event) => updateAdjustmentForm({ confirmedNavDate: event.target.value })}
                className="min-h-11 w-full rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-accent/60"
              />
            </label>
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>调整日期</span>
              <input
                type="date"
                value={adjustmentForm.tradeDate}
                onChange={(event) => updateAdjustmentForm({ tradeDate: event.target.value })}
                className="min-h-11 w-full rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-accent/60"
              />
            </label>
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>来源</span>
              <select
                value={adjustmentForm.sourcePlatform}
                onChange={(event) => updateAdjustmentForm({ sourcePlatform: event.target.value as HoldingSourcePlatform | '' })}
                className="min-h-11 w-full rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-accent/60"
              >
                <option value="">未记录来源</option>
                {HOLDING_SOURCE_OPTIONS.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
            </label>
          </div>

          <label className="mt-3 block space-y-2 text-xs text-theme-secondary">
            <span>备注</span>
            <input
              value={adjustmentForm.note}
              onChange={(event) => updateAdjustmentForm({ note: event.target.value })}
              className="min-h-11 w-full rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-accent/60"
              placeholder="例如：平台迁移 / 手续费份额修正"
            />
          </label>

          {adjustmentError && <div className="mt-3 text-xs text-danger">{adjustmentError}</div>}

          <div className="mt-4 flex flex-wrap items-center gap-3">
            <button
              type="button"
              onClick={() => void handleSaveAdjustment()}
              disabled={isAdjustmentSaving}
              className="inline-flex items-center gap-2 rounded-xl bg-[var(--input-bg)] border border-[var(--input-border)] px-4 py-2.5 text-sm font-semibold text-theme-primary shadow-none transition-transform hover:-translate-y-0.5 active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-70"
            >
              <SlidersHorizontal className={cn('h-4 w-4', isAdjustmentSaving && 'animate-pulse')} />
              {isAdjustmentSaving ? '保存中...' : '保存份额调整'}
            </button>
            <button
              type="button"
              onClick={() => {
                setIsAdjustmentOpen(false)
                setAdjustmentForm(buildAdjustmentFormState(holding))
                setAdjustmentError('')
              }}
              className="rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-2.5 text-sm text-theme-secondary transition-colors hover:border-accent/35 hover:text-theme-primary"
            >
              取消
            </button>
          </div>
        </div>
      )}
      {isSelling && (
        <div className="rounded-xl border border-warning/20 bg-warning/8 p-5 shadow-none backdrop-blur-xl">
          <div className="flex flex-col gap-2 md:flex-row md:items-start md:justify-between">
            <div>
              <div className="text-sm font-semibold text-theme-primary">记录卖出 / 减仓</div>
              <p className="mt-1 max-w-2xl text-xs leading-5 text-theme-muted">
                优先按卖出份额记录，系统按比例减少成本。若按金额记录，请填写减少的成本本金，不是赎回到账金额；当前不计算实际卖出盈亏。
              </p>
            </div>
            <div className="rounded-xl border border-warning/20 bg-warning/10 px-3 py-2 text-xs text-theme-primary">
              当前持仓：{formatAmount(holding.amount)}{holding.shares ? ` · ${holding.shares} 份` : ''}
            </div>
          </div>

          <div className="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>减少成本本金（非到账金额）</span>
              <input
                value={sellForm.amount}
                onChange={(event) => updateSellForm({ amount: event.target.value, shares: '' })}
                className="min-h-11 w-full rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-warning/60"
                inputMode="decimal"
                placeholder="例如 10000"
              />
            </label>
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>卖出份额（优先填写）</span>
              <input
                value={sellForm.shares}
                onChange={(event) => updateSellForm({ shares: event.target.value, amount: '' })}
                className="min-h-11 w-full rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-warning/60"
                inputMode="decimal"
                placeholder="已确认份额后可填"
              />
            </label>
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>卖出日期</span>
              <input
                type="date"
                value={sellForm.tradeDate}
                onChange={(event) => updateSellForm({ tradeDate: event.target.value })}
                className="min-h-11 w-full rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-warning/60"
              />
            </label>
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>备注</span>
              <input
                value={sellForm.note}
                onChange={(event) => updateSellForm({ note: event.target.value })}
                className="min-h-11 w-full rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-warning/60"
                placeholder="例如：止盈减仓 / 调整仓位"
              />
            </label>
          </div>

          <label className="mt-4 flex cursor-pointer items-start gap-3 rounded-[22px] border border-warning/20 bg-warning/8 p-4 text-sm text-theme-secondary transition-colors hover:border-warning/35">
            <input
              type="checkbox"
              checked={sellForm.sellAll}
              onChange={(event) => updateSellForm({ sellAll: event.target.checked })}
              className="mt-1 h-4 w-4 accent-amber-400"
            />
            <span>
              <span className="block font-semibold text-theme-primary">全部清仓</span>
              <span className="mt-1 block text-xs leading-5 text-theme-muted">开启后会按当前持仓本金和确认份额记录一笔卖出流水，并移除这条当前持仓快照；如果只是录错基金或录错记录，仍应使用删除。</span>
            </span>
          </label>

          {sellError && <div className="mt-3 text-xs text-danger">{sellError}</div>}

          <div className="mt-4 flex flex-wrap items-center gap-3">
            <button
              type="button"
              onClick={() => void handleSaveSell()}
              disabled={isSellSaving}
              className="inline-flex items-center gap-2 rounded-xl bg-[var(--input-bg)] border border-[var(--input-border)] px-4 py-2.5 text-sm font-semibold text-theme-primary shadow-none transition-transform hover:-translate-y-0.5 active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-70"
            >
              <MinusCircle className={cn('h-4 w-4', isSellSaving && 'animate-pulse')} />
              {isSellSaving ? '记录中...' : sellForm.sellAll ? '确认清仓' : '确认减仓'}
            </button>
            <button
              type="button"
              onClick={() => {
                setIsSelling(false)
                setSellForm(buildSellFormState(holding))
                setSellError('')
              }}
              className="rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-2.5 text-sm text-theme-secondary transition-colors hover:border-warning/35 hover:text-theme-primary"
            >
              取消
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
