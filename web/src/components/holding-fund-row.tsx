'use client'

import { useEffect, useState } from 'react'
import { Gift, MinusCircle, Pencil, Save, SlidersHorizontal, Trash2, X } from 'lucide-react'
import { FundAnalysisBadge } from '@/components/fund-analysis-badge'
import { FundAnalysisEventHint } from '@/components/fund-analysis-event-hint'
import type { FundAnalysis } from '@/hooks/use-fund-data'
import { useMarketTradingState } from '@/hooks/use-market-status'
import { useFund, type FundEstimate } from '@/hooks/use-fund-data'
import {
  HOLDING_SOURCE_OPTIONS,
  isHoldingSourcePlatform,
  resolveHoldingSourceLabel,
  type HoldingSourcePlatform,
} from '@/lib/holding-sources'
import { cn } from '@/lib/utils'
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
  analysis?: FundAnalysis | null
  onRemove: () => Promise<void> | void
  onUpdate: (payload: UpdateHoldingPayload) => Promise<void> | void
  onSell: (payload: SellHoldingPayload) => Promise<void> | void
  onRecordDividend: (payload: DividendHoldingPayload) => Promise<void> | void
  onAdjustShares: (payload: AdjustHoldingSharesPayload) => Promise<void> | void
}

function formatAmount(amount: string) {
  const value = Number.parseFloat(amount)
  if (Number.isNaN(value)) {
    return '¥0.00'
  }
  return new Intl.NumberFormat('zh-CN', {
    style: 'currency',
    currency: 'CNY',
    maximumFractionDigits: 2,
  }).format(value)
}

function formatMetricCurrency(amount?: string) {
  if (!amount) {
    return '--'
  }

  const value = Number.parseFloat(amount)
  if (Number.isNaN(value)) {
    return '--'
  }

  return new Intl.NumberFormat('zh-CN', {
    style: 'currency',
    currency: 'CNY',
    maximumFractionDigits: 2,
  }).format(value)
}

function formatNumberCurrency(value?: number) {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return '--'
  }

  return new Intl.NumberFormat('zh-CN', {
    style: 'currency',
    currency: 'CNY',
    maximumFractionDigits: 2,
  }).format(value)
}

function formatPercentValue(value?: string) {
  if (!value) {
    return '--'
  }

  const parsed = Number.parseFloat(value)
  if (Number.isNaN(parsed)) {
    return '--'
  }

  return `${parsed >= 0 ? '+' : ''}${parsed.toFixed(2)}%`
}

function formatNumberPercent(value?: number) {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return '--'
  }

  return `${value >= 0 ? '+' : ''}${value.toFixed(2)}%`
}

function parseMetricNumber(value?: string) {
  if (!value) {
    return null
  }

  const parsed = Number.parseFloat(value)
  return Number.isNaN(parsed) ? null : parsed
}

function formatEstimatedDelta(amount: string, changePercent?: string) {
  const amountNumber = Number.parseFloat(amount)
  const percentNumber = Number.parseFloat(changePercent || '0')
  if (Number.isNaN(amountNumber) || Number.isNaN(percentNumber)) {
    return { text: '¥0.00', isPositive: false }
  }

  const delta = amountNumber * percentNumber / 100
  const isPositive = delta >= 0
  return {
    text: `${isPositive ? '+' : ''}${new Intl.NumberFormat('zh-CN', {
      style: 'currency',
      currency: 'CNY',
      maximumFractionDigits: 2,
    }).format(delta)}`,
    isPositive,
  }
}

function formatTradeAt(tradeAt?: string) {
  if (!tradeAt) {
    return ''
  }

  const parsed = new Date(tradeAt)
  if (Number.isNaN(parsed.getTime())) {
    return tradeAt
  }

  const formatter = new Intl.DateTimeFormat('zh-CN', {
    timeZone: 'Asia/Shanghai',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  })

  const parts = formatter.formatToParts(parsed)
  const values = Object.fromEntries(parts.map((part) => [part.type, part.value]))
  const dateLabel = `${values.month}/${values.day}`
  const timeLabel = `${values.hour}:${values.minute}`

  if (timeLabel === '14:59') {
    return `${dateLabel} 15:00前`
  }

  if (timeLabel === '15:01') {
    return `${dateLabel} 15:00后`
  }

  return formatter.format(parsed)
}

function normalizeDecimalInput(value: string) {
  return value.replace(/,/g, '').trim()
}

function formatDateForInput(raw?: string) {
  if (!raw) {
    return ''
  }
  if (/^\d{4}-\d{2}-\d{2}$/.test(raw)) {
    return raw
  }
  const parsed = new Date(raw)
  if (Number.isNaN(parsed.getTime())) {
    return ''
  }
  const formatter = new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Shanghai',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  })
  return formatter.format(parsed)
}

function buildCorrectionTradeAt(date: string, originalTradeAt?: string) {
  if (!date) {
    return ''
  }

  if (originalTradeAt && formatDateForInput(originalTradeAt) === date) {
    const parsed = new Date(originalTradeAt)
    if (!Number.isNaN(parsed.getTime())) {
      const formatter = new Intl.DateTimeFormat('en-GB', {
        timeZone: 'Asia/Shanghai',
        hour: '2-digit',
        minute: '2-digit',
        hour12: false,
      })
      const values = Object.fromEntries(formatter.formatToParts(parsed).map((part) => [part.type, part.value]))
      if (values.hour && values.minute) {
        return `${date}T${values.hour}:${values.minute}:00+08:00`
      }
    }
  }

  return `${date}T14:59:00+08:00`
}

interface CorrectionFormState {
  amount: string
  shares: string
  confirmedNav: string
  confirmedNavDate: string
  tradeDate: string
  note: string
  sourcePlatform: HoldingSourcePlatform | ''
}

interface SellFormState {
  amount: string
  shares: string
  tradeDate: string
  note: string
  sellAll: boolean
}

interface DividendFormState {
  amount: string
  shares: string
  tradeDate: string
  note: string
  reinvest: boolean
  sourcePlatform: HoldingSourcePlatform | ''
}

interface AdjustmentFormState {
  targetShares: string
  sharesDelta: string
  confirmedNav: string
  confirmedNavDate: string
  tradeDate: string
  note: string
  sourcePlatform: HoldingSourcePlatform | ''
}

function buildCorrectionFormState(holding: HoldingEntry): CorrectionFormState {
  return {
    amount: holding.amount || '',
    shares: holding.shares || '',
    confirmedNav: holding.confirmed_nav || '',
    confirmedNavDate: formatDateForInput(holding.confirmed_nav_date),
    tradeDate: formatDateForInput(holding.trade_at || holding.as_of_date),
    note: holding.note || '',
    sourcePlatform: isHoldingSourcePlatform(holding.source_platform) ? holding.source_platform : '',
  }
}

function buildSellFormState(holding: HoldingEntry): SellFormState {
  return {
    amount: '',
    shares: '',
    tradeDate: formatDateForInput(new Date().toISOString()) || formatDateForInput(holding.trade_at || holding.as_of_date),
    note: '',
    sellAll: false,
  }
}

function buildDividendFormState(holding: HoldingEntry): DividendFormState {
  return {
    amount: '',
    shares: '',
    tradeDate: formatDateForInput(new Date().toISOString()) || formatDateForInput(holding.trade_at || holding.as_of_date),
    note: '',
    reinvest: false,
    sourcePlatform: isHoldingSourcePlatform(holding.source_platform) ? holding.source_platform : '',
  }
}

function buildAdjustmentFormState(holding: HoldingEntry): AdjustmentFormState {
  return {
    targetShares: '',
    sharesDelta: '',
    confirmedNav: holding.confirmed_nav || '',
    confirmedNavDate: formatDateForInput(holding.confirmed_nav_date),
    tradeDate: formatDateForInput(new Date().toISOString()) || formatDateForInput(holding.trade_at || holding.as_of_date),
    note: '',
    sourcePlatform: isHoldingSourcePlatform(holding.source_platform) ? holding.source_platform : '',
  }
}

function validatePositiveDecimal(value: string) {
  const parsed = Number.parseFloat(normalizeDecimalInput(value))
  return Number.isFinite(parsed) && parsed > 0
}

function validateNonZeroDecimal(value: string) {
  const parsed = Number.parseFloat(normalizeDecimalInput(value))
  return Number.isFinite(parsed) && parsed !== 0
}

export function HoldingFundRow({
  holding,
  metricScope = 'official',
  estimate: providedEstimate,
  analysis,
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
  const { session } = useMarketTradingState()
  const isCallAuction = session === 'call_auction'
  const { fund } = useFund(holding.fund_id)
  const estimate = providedEstimate ?? null
  const fundName = holding.fund?.name || fund?.name || estimate?.fund_name || holding.fund_id
  const tradeAtLabel = formatTradeAt(holding.trade_at)
  const sourceLabel = resolveHoldingSourceLabel(holding.source_platform, holding.source_label)
  const estimateDelta = isCallAuction ? { text: '-', isPositive: false } : formatEstimatedDelta(holding.amount, estimate?.change_percent)
  const confirmedDateLabel = holding.confirmed_nav_date || holding.as_of_date
  const sharesNumber = parseMetricNumber(holding.shares)
  const estimateNavNumber = parseMetricNumber(estimate?.estimate_nav)
  const prevNavNumber = parseMetricNumber(estimate?.prev_nav)
  const estimateChangePercentNumber = parseMetricNumber(estimate?.change_percent)
  const hasEstimatedHoldingMetrics = !isCallAuction &&
    typeof sharesNumber === 'number' &&
    sharesNumber > 0 &&
    typeof estimateNavNumber === 'number' &&
    typeof prevNavNumber === 'number' &&
    prevNavNumber > 0
  const estimatedCurrentMarketValue = hasEstimatedHoldingMetrics ? sharesNumber * estimateNavNumber : null
  const estimatedTodayProfit = hasEstimatedHoldingMetrics ? sharesNumber * (estimateNavNumber - prevNavNumber) : null
  const isOfficialScope = metricScope === 'official'
  const shouldUseOfficialValues = isOfficialScope && holding.real_metrics_ready
  const currentMarketValueLabel = shouldUseOfficialValues ? '最新官方市值' : '盘中预估市值'
  const profitLabel = shouldUseOfficialValues ? '今日官方盈亏' : '实时盈亏预估'
  const changeLabel = shouldUseOfficialValues ? '今日官方涨跌幅' : '实时涨跌预估'
  const currentMarketValueText = isOfficialScope
    ? (holding.real_metrics_ready ? formatMetricCurrency(holding.current_market_value) : formatNumberCurrency(estimatedCurrentMarketValue ?? undefined))
    : formatNumberCurrency(estimatedCurrentMarketValue ?? undefined)
  const todayProfitText = isOfficialScope
    ? (holding.real_metrics_ready ? formatMetricCurrency(holding.today_profit) : hasEstimatedHoldingMetrics
      ? formatNumberCurrency(estimatedTodayProfit ?? undefined)
      : (!isCallAuction && estimate?.change_percent ? estimateDelta.text : '--'))
    : hasEstimatedHoldingMetrics
      ? formatNumberCurrency(estimatedTodayProfit ?? undefined)
      : (!isCallAuction && estimate?.change_percent ? estimateDelta.text : '--')
  const todayChangePercentText = isOfficialScope
    ? (holding.real_metrics_ready ? formatPercentValue(holding.today_change_percent) : formatNumberPercent(estimateChangePercentNumber ?? undefined))
    : formatNumberPercent(estimateChangePercentNumber ?? undefined)
  const realMetricTone = (() => {
    if (shouldUseOfficialValues) {
      const profit = Number.parseFloat(holding.today_profit || '0')
      return profit >= 0 ? 'text-up' : 'text-down'
    }
    if (hasEstimatedHoldingMetrics && typeof estimatedTodayProfit === 'number') {
      return estimatedTodayProfit >= 0 ? 'text-up' : 'text-down'
    }
    if (!isCallAuction && estimate?.change_percent) {
      return estimateDelta.isPositive ? 'text-up' : 'text-down'
    }
    return 'text-theme-primary'
  })()
  const changeTone = (() => {
    if (shouldUseOfficialValues) {
      const change = Number.parseFloat(holding.today_change_percent || '0')
      return change >= 0 ? 'text-up' : 'text-down'
    }
    if (typeof estimateChangePercentNumber === 'number') {
      return estimateChangePercentNumber >= 0 ? 'text-up' : 'text-down'
    }
    return 'text-theme-primary'
  })()
  const marketValueNote = shouldUseOfficialValues
    ? holding.real_metrics_ready
      ? `${holding.actual_date || '最新'} 官方净值口径`
      : ''
    : hasEstimatedHoldingMetrics
      ? `按 ${holding.shares || '--'} 份与盘中预估净值估算`
      : '待确认净值补齐后展示'
  const profitNote = shouldUseOfficialValues
    ? `${holding.actual_date || '最新'} 官方净值口径`
    : hasEstimatedHoldingMetrics
      ? '盘中预估，夜间官方净值同步后会自动覆盖'
      : !isCallAuction && estimate?.change_percent
        ? `按预估涨跌幅折算，夜间官方净值同步后会自动覆盖`
        : '待确认份额补齐后展示'
  const officialProfitNote = holding.real_metrics_ready
    ? `已按 ${holding.actual_date || '最新'} 官方净值结算`
    : holding.real_metrics_message || '待官方净值同步后展示'
  const displayedProfitNote = shouldUseOfficialValues ? officialProfitNote : profitNote
  const changeNote = shouldUseOfficialValues
    ? '按最新官方涨跌幅展示'
    : estimate?.change_percent
      ? '根据基金预估涨跌幅计算，夜间真实涨跌幅会自动覆盖'
      : '待确认份额补齐后展示'
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
    const hasAnyConfirmation = Boolean(normalizedShares || normalizedNav || form.confirmedNavDate)
    const hasFullConfirmation = Boolean(normalizedShares && normalizedNav && form.confirmedNavDate)

    if (!validatePositiveDecimal(normalizedAmount)) {
      setFormError('请输入有效的持仓本金。')
      return
    }

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
        trade_at: buildCorrectionTradeAt(form.tradeDate, holding.trade_at),
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
      setSellError('请输入本次卖出金额，或在已确认份额后输入卖出份额；如果是全部赎回，请开启“全部清仓”。')
      return
    }

    if (normalizedAmount && !validatePositiveDecimal(normalizedAmount)) {
      setSellError('请输入有效的卖出金额。')
      return
    }

    if (normalizedShares && !validatePositiveDecimal(normalizedShares)) {
      setSellError('请输入有效的卖出份额。')
      return
    }

    if (!sellForm.sellAll && amountValue !== null && Number.isFinite(currentAmount) && amountValue >= currentAmount) {
      setSellError('部分减仓金额需要小于当前持仓；如果是全部赎回，请开启“全部清仓”。')
      return
    }

    if (sharesValue !== null) {
      if (!Number.isFinite(currentShares) || currentShares <= 0) {
        setSellError('当前持仓还没有确认份额，暂时只能按金额减仓。')
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
    <div className="space-y-3">
      <div className="grid gap-4 rounded-[28px] border border-[var(--card-border)] p-5 glass lg:grid-cols-[minmax(0,1.25fr)_0.8fr_0.85fr_0.85fr_0.7fr_0.7fr_auto] lg:items-center">
        <div className="min-w-0">
          <div className="truncate text-base font-semibold text-theme-primary">{fundName}</div>
          <div className="mt-1 text-xs text-theme-muted">{holding.fund_id}</div>
          <div className="mt-2">
            <FundAnalysisBadge analysis={analysis} compact />
          </div>
          <div className="mt-2">
            <FundAnalysisEventHint analysis={analysis} compact />
          </div>
          {holding.note && <div className="mt-2 text-xs text-theme-secondary">{holding.note}</div>}
          {holding.manual_confirmation && (
            <div className="mt-2 inline-flex items-center rounded-full border border-cyan-400/25 bg-cyan-500/10 px-2.5 py-1 text-[11px] font-medium text-cyan-200">
              已按{sourceLabel || '外部平台'}校正
            </div>
          )}
          {!holding.manual_confirmation && sourceLabel && (
            <div className="mt-2 inline-flex items-center rounded-full border border-slate-400/20 bg-slate-400/8 px-2.5 py-1 text-[11px] font-medium text-theme-secondary">
              来源：{sourceLabel}
            </div>
          )}
          {!holding.real_metrics_ready && (
            <div className="mt-2 text-xs text-theme-muted">
              {holding.real_metrics_message || '待真实净值同步后展示真实市值与盈亏。'}
            </div>
          )}
        </div>

        <div>
          <div className="text-xs text-theme-muted">持仓本金</div>
          <div className="mt-1 text-lg font-semibold text-theme-primary">{formatAmount(holding.amount)}</div>
        </div>

        <div>
          <div className="text-xs text-theme-muted">{currentMarketValueLabel}</div>
          <div className="mt-1 text-lg font-semibold text-theme-primary">
            {currentMarketValueText}
          </div>
          <div className="mt-1 text-xs text-theme-muted">{marketValueNote}</div>
        </div>

        <div>
          <div className="text-xs text-theme-muted">{profitLabel}</div>
          <div className={cn('mt-1 text-lg font-semibold', realMetricTone)}>
            {todayProfitText}
          </div>
          <div className={cn('mt-1 text-xs', shouldUseOfficialValues || !estimate?.change_percent ? 'text-theme-muted' : estimateDelta.isPositive ? 'text-up' : 'text-down')}>
            {displayedProfitNote}
          </div>
        </div>

        <div>
          <div className="text-xs text-theme-muted">{changeLabel}</div>
          <div className={cn('mt-1 text-lg font-semibold', changeTone)}>
            {todayChangePercentText}
          </div>
          <div className="mt-1 text-xs text-theme-muted">{changeNote}</div>
        </div>

        <div>
          <div className="text-xs text-theme-muted">确认净值日</div>
          <div className="mt-1 text-sm font-medium text-theme-primary">{confirmedDateLabel}</div>
          {tradeAtLabel && <div className="mt-1 text-xs text-theme-muted">提交于 {tradeAtLabel}</div>}
        </div>

        <div className="flex items-center gap-2 lg:flex-col">
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
              'group relative inline-flex items-center justify-center overflow-hidden rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] p-2 text-theme-muted transition-all duration-200',
              'hover:-translate-y-0.5 hover:border-cyan-400/50 hover:bg-cyan-500/12 hover:text-cyan-200',
              'active:scale-95 disabled:cursor-not-allowed',
              isEditing && 'border-cyan-400/45 bg-cyan-500/14 text-cyan-200'
            )}
            aria-label={`校正 ${fundName} 持仓`}
          >
            {isEditing ? <X className="h-4 w-4" /> : <Pencil className="h-4 w-4" />}
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
              'group relative inline-flex items-center justify-center overflow-hidden rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] p-2 text-theme-muted transition-all duration-200',
              'hover:-translate-y-0.5 hover:border-amber-400/50 hover:bg-amber-500/12 hover:text-amber-200',
              'active:scale-95 disabled:cursor-not-allowed',
              isSelling && 'border-amber-400/45 bg-amber-500/14 text-amber-200'
            )}
            aria-label={`卖出/减仓 ${fundName} 持仓`}
          >
            {isSelling ? <X className="h-4 w-4" /> : <MinusCircle className="h-4 w-4" />}
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
              'group relative inline-flex items-center justify-center overflow-hidden rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] p-2 text-theme-muted transition-all duration-200',
              'hover:-translate-y-0.5 hover:border-violet-400/50 hover:bg-violet-500/12 hover:text-violet-200',
              'active:scale-95 disabled:cursor-not-allowed',
              isDividendOpen && 'border-violet-400/45 bg-violet-500/14 text-violet-200'
            )}
            aria-label={`记录 ${fundName} 分红`}
          >
            {isDividendOpen ? <X className="h-4 w-4" /> : <Gift className="h-4 w-4" />}
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
              'group relative inline-flex items-center justify-center overflow-hidden rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] p-2 text-theme-muted transition-all duration-200',
              'hover:-translate-y-0.5 hover:border-sky-400/50 hover:bg-sky-500/12 hover:text-sky-200',
              'active:scale-95 disabled:cursor-not-allowed',
              isAdjustmentOpen && 'border-sky-400/45 bg-sky-500/14 text-sky-200'
            )}
            aria-label={`调整 ${fundName} 份额`}
          >
            {isAdjustmentOpen ? <X className="h-4 w-4" /> : <SlidersHorizontal className="h-4 w-4" />}
          </button>
          <button
            type="button"
            onClick={() => void handleRemove()}
            disabled={actionDisabled}
            className={cn(
              'group relative inline-flex items-center justify-center overflow-hidden rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] p-2 text-theme-muted transition-all duration-200',
              'hover:-translate-y-0.5 hover:border-rose-400/50 hover:bg-rose-500/12 hover:text-rose-300',
              'active:scale-95 disabled:cursor-not-allowed',
              isRemoving && 'holding-delete-button border-rose-400/50 bg-rose-500/16 text-rose-200'
            )}
            aria-label={`移除 ${fundName} 持仓`}
            aria-busy={isRemoving}
          >
            <span
              className={cn(
                'pointer-events-none absolute inset-0 rounded-xl bg-rose-400/0 opacity-0 transition-opacity duration-200',
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
        <div className="rounded-[28px] border border-cyan-400/20 bg-cyan-500/8 p-5 shadow-[0_20px_50px_rgba(8,145,178,0.12)] backdrop-blur-xl">
          <div className="flex flex-col gap-2 md:flex-row md:items-start md:justify-between">
            <div>
              <div className="text-sm font-semibold text-theme-primary">校正持仓口径</div>
              <p className="mt-1 max-w-2xl text-xs leading-5 text-theme-muted">
                用于对齐支付宝、微信等外部平台显示差异；这不是买入/卖出记录，不会改变基金代码，只修正当前这笔持仓的金额、确认份额、确认净值和备注。
              </p>
            </div>
            <div className="rounded-2xl border border-cyan-400/20 bg-cyan-500/10 px-3 py-2 text-xs text-cyan-100">
              留空份额/净值/日期会回到系统自动确认口径
            </div>
          </div>

          <div className="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-6">
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>持仓本金</span>
              <input
                value={form.amount}
                onChange={(event) => updateForm({ amount: event.target.value })}
                className="w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-cyan-400/60"
                inputMode="decimal"
                placeholder="例如 50000"
              />
            </label>
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>确认份额</span>
              <input
                value={form.shares}
                onChange={(event) => updateForm({ shares: event.target.value })}
                className="w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-cyan-400/60"
                inputMode="decimal"
                placeholder="外部平台显示份额"
              />
            </label>
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>确认净值</span>
              <input
                value={form.confirmedNav}
                onChange={(event) => updateForm({ confirmedNav: event.target.value })}
                className="w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-cyan-400/60"
                inputMode="decimal"
                placeholder="例如 1.2345"
              />
            </label>
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>确认净值日</span>
              <input
                type="date"
                value={form.confirmedNavDate}
                onChange={(event) => updateForm({ confirmedNavDate: event.target.value })}
                className="w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-cyan-400/60"
              />
            </label>
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>申购日期</span>
              <input
                type="date"
                value={form.tradeDate}
                onChange={(event) => updateForm({ tradeDate: event.target.value })}
                className="w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-cyan-400/60"
              />
            </label>
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>校正来源</span>
              <select
                value={form.sourcePlatform}
                onChange={(event) => updateForm({ sourcePlatform: event.target.value as HoldingSourcePlatform | '' })}
                className="w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-cyan-400/60"
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
            <div className="mt-3 rounded-[20px] border border-cyan-400/15 bg-cyan-400/8 px-4 py-3 text-xs leading-5 text-theme-muted">
              {HOLDING_SOURCE_OPTIONS.find((option) => option.value === form.sourcePlatform)?.description}
              ；来源会写入持仓和校正流水，便于筛选与对账。
            </div>
          )}

          <label className="mt-3 block space-y-2 text-xs text-theme-secondary">
            <span>备注</span>
            <input
              value={form.note}
              onChange={(event) => updateForm({ note: event.target.value })}
              className="w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-cyan-400/60"
              placeholder="例如：按支付宝份额校正 / 微信显示净值差异"
            />
          </label>

          {formError && <div className="mt-3 text-xs text-rose-300">{formError}</div>}

          <div className="mt-4 flex flex-wrap items-center gap-3">
            <button
              type="button"
              onClick={() => void handleSaveCorrection()}
              disabled={isSaving}
              className="inline-flex items-center gap-2 rounded-2xl bg-gradient-to-r from-cyan-500 via-sky-500 to-blue-600 px-4 py-2.5 text-sm font-semibold text-white shadow-[0_14px_35px_rgba(14,165,233,0.22)] transition-transform hover:-translate-y-0.5 active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-70"
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
              className="rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-2.5 text-sm text-theme-secondary transition-colors hover:border-cyan-400/35 hover:text-theme-primary"
            >
              取消
            </button>
            <button
              type="button"
              onClick={() => updateForm({ shares: '', confirmedNav: '', confirmedNavDate: '', sourcePlatform: '' })}
              className="rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-2.5 text-sm text-theme-muted transition-colors hover:border-amber-400/35 hover:text-amber-200"
            >
              回到自动确认
            </button>
          </div>
        </div>
      )}
      {isDividendOpen && (
        <div className="rounded-[28px] border border-violet-400/20 bg-violet-400/8 p-5 shadow-[0_20px_50px_rgba(139,92,246,0.12)] backdrop-blur-xl">
          <div className="flex flex-col gap-2 md:flex-row md:items-start md:justify-between">
            <div>
              <div className="text-sm font-semibold text-theme-primary">记录分红 / 红利再投</div>
              <p className="mt-1 max-w-2xl text-xs leading-5 text-theme-muted">
                现金分红只写入流水，不改变当前持仓份额；红利再投会把新增份额合并到当前持仓，并保留分红流水痕迹。
              </p>
            </div>
            <div className="rounded-2xl border border-violet-400/20 bg-violet-400/10 px-3 py-2 text-xs text-violet-100">
              当前份额：{holding.shares || '待确认'}
            </div>
          </div>

          <div className="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-5">
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>分红金额</span>
              <input
                value={dividendForm.amount}
                onChange={(event) => updateDividendForm({ amount: event.target.value })}
                className="w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-violet-400/60"
                inputMode="decimal"
                placeholder="例如 125.00"
              />
            </label>
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>新增份额</span>
              <input
                value={dividendForm.shares}
                onChange={(event) => updateDividendForm({ shares: event.target.value })}
                className="w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-violet-400/60"
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
                className="w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-violet-400/60"
              />
            </label>
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>来源</span>
              <select
                value={dividendForm.sourcePlatform}
                onChange={(event) => updateDividendForm({ sourcePlatform: event.target.value as HoldingSourcePlatform | '' })}
                className="w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-violet-400/60"
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
                className="w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-violet-400/60"
                placeholder="例如：现金分红 / 红利再投"
              />
            </label>
          </div>

          <label className="mt-4 flex cursor-pointer items-start gap-3 rounded-[22px] border border-violet-400/20 bg-violet-400/8 p-4 text-sm text-theme-secondary transition-colors hover:border-violet-400/35">
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

          {dividendError && <div className="mt-3 text-xs text-rose-300">{dividendError}</div>}

          <div className="mt-4 flex flex-wrap items-center gap-3">
            <button
              type="button"
              onClick={() => void handleSaveDividend()}
              disabled={isDividendSaving}
              className="inline-flex items-center gap-2 rounded-2xl bg-gradient-to-r from-violet-500 via-fuchsia-500 to-purple-600 px-4 py-2.5 text-sm font-semibold text-white shadow-[0_14px_35px_rgba(139,92,246,0.22)] transition-transform hover:-translate-y-0.5 active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-70"
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
              className="rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-2.5 text-sm text-theme-secondary transition-colors hover:border-violet-400/35 hover:text-theme-primary"
            >
              取消
            </button>
          </div>
        </div>
      )}
      {isAdjustmentOpen && (
        <div className="rounded-[28px] border border-sky-400/20 bg-sky-400/8 p-5 shadow-[0_20px_50px_rgba(56,189,248,0.12)] backdrop-blur-xl">
          <div className="flex flex-col gap-2 md:flex-row md:items-start md:justify-between">
            <div>
              <div className="text-sm font-semibold text-theme-primary">记录份额调整</div>
              <p className="mt-1 max-w-2xl text-xs leading-5 text-theme-muted">
                适合平台迁移、拆分合并、手续费或历史口径修正；不会作为买卖交易处理，但会更新当前持仓快照并写入调整流水。
              </p>
            </div>
            <div className="rounded-2xl border border-sky-400/20 bg-sky-400/10 px-3 py-2 text-xs text-sky-100">
              当前份额：{holding.shares || '待确认'}
            </div>
          </div>

          <div className="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-6">
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>目标份额</span>
              <input
                value={adjustmentForm.targetShares}
                onChange={(event) => updateAdjustmentForm({ targetShares: event.target.value, sharesDelta: '' })}
                className="w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-sky-400/60"
                inputMode="decimal"
                placeholder="调整后的总份额"
              />
            </label>
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>份额差额</span>
              <input
                value={adjustmentForm.sharesDelta}
                onChange={(event) => updateAdjustmentForm({ sharesDelta: event.target.value, targetShares: '' })}
                className="w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-sky-400/60"
                inputMode="decimal"
                placeholder="例如 100 或 -50"
              />
            </label>
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>确认净值</span>
              <input
                value={adjustmentForm.confirmedNav}
                onChange={(event) => updateAdjustmentForm({ confirmedNav: event.target.value })}
                className="w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-sky-400/60"
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
                className="w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-sky-400/60"
              />
            </label>
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>调整日期</span>
              <input
                type="date"
                value={adjustmentForm.tradeDate}
                onChange={(event) => updateAdjustmentForm({ tradeDate: event.target.value })}
                className="w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-sky-400/60"
              />
            </label>
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>来源</span>
              <select
                value={adjustmentForm.sourcePlatform}
                onChange={(event) => updateAdjustmentForm({ sourcePlatform: event.target.value as HoldingSourcePlatform | '' })}
                className="w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-sky-400/60"
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
              className="w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-sky-400/60"
              placeholder="例如：平台迁移 / 手续费份额修正"
            />
          </label>

          {adjustmentError && <div className="mt-3 text-xs text-rose-300">{adjustmentError}</div>}

          <div className="mt-4 flex flex-wrap items-center gap-3">
            <button
              type="button"
              onClick={() => void handleSaveAdjustment()}
              disabled={isAdjustmentSaving}
              className="inline-flex items-center gap-2 rounded-2xl bg-gradient-to-r from-sky-500 via-cyan-500 to-blue-600 px-4 py-2.5 text-sm font-semibold text-white shadow-[0_14px_35px_rgba(56,189,248,0.22)] transition-transform hover:-translate-y-0.5 active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-70"
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
              className="rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-2.5 text-sm text-theme-secondary transition-colors hover:border-sky-400/35 hover:text-theme-primary"
            >
              取消
            </button>
          </div>
        </div>
      )}
      {isSelling && (
        <div className="rounded-[28px] border border-amber-400/20 bg-amber-400/8 p-5 shadow-[0_20px_50px_rgba(245,158,11,0.12)] backdrop-blur-xl">
          <div className="flex flex-col gap-2 md:flex-row md:items-start md:justify-between">
            <div>
              <div className="text-sm font-semibold text-theme-primary">记录卖出 / 减仓</div>
              <p className="mt-1 max-w-2xl text-xs leading-5 text-theme-muted">
                用于记录部分赎回或全部清仓，并同步调整当前这笔持仓快照。清仓会写入卖出流水并从当前持仓中移除，不再和“删除错误记录”混用。
              </p>
            </div>
            <div className="rounded-2xl border border-amber-400/20 bg-amber-400/10 px-3 py-2 text-xs text-amber-100">
              当前持仓：{formatAmount(holding.amount)}{holding.shares ? ` · ${holding.shares} 份` : ''}
            </div>
          </div>

          <div className="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>卖出金额</span>
              <input
                value={sellForm.amount}
                onChange={(event) => updateSellForm({ amount: event.target.value })}
                className="w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-amber-400/60"
                inputMode="decimal"
                placeholder="例如 10000"
              />
            </label>
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>卖出份额（可选）</span>
              <input
                value={sellForm.shares}
                onChange={(event) => updateSellForm({ shares: event.target.value })}
                className="w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-amber-400/60"
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
                className="w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-amber-400/60"
              />
            </label>
            <label className="space-y-2 text-xs text-theme-secondary">
              <span>备注</span>
              <input
                value={sellForm.note}
                onChange={(event) => updateSellForm({ note: event.target.value })}
                className="w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2.5 text-sm text-theme-primary outline-none transition-colors focus:border-amber-400/60"
                placeholder="例如：止盈减仓 / 调整仓位"
              />
            </label>
          </div>

          <label className="mt-4 flex cursor-pointer items-start gap-3 rounded-[22px] border border-amber-400/20 bg-amber-400/8 p-4 text-sm text-theme-secondary transition-colors hover:border-amber-400/35">
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

          {sellError && <div className="mt-3 text-xs text-rose-300">{sellError}</div>}

          <div className="mt-4 flex flex-wrap items-center gap-3">
            <button
              type="button"
              onClick={() => void handleSaveSell()}
              disabled={isSellSaving}
              className="inline-flex items-center gap-2 rounded-2xl bg-gradient-to-r from-amber-500 via-orange-500 to-rose-500 px-4 py-2.5 text-sm font-semibold text-white shadow-[0_14px_35px_rgba(245,158,11,0.22)] transition-transform hover:-translate-y-0.5 active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-70"
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
              className="rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-2.5 text-sm text-theme-secondary transition-colors hover:border-amber-400/35 hover:text-theme-primary"
            >
              取消
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
