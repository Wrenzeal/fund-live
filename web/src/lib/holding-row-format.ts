import type { HoldingEntry } from '@/hooks/use-user-portfolio'
import { isHoldingSourcePlatform, type HoldingSourcePlatform } from '@/lib/holding-sources'

export function formatAmount(amount: string) {
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

export function formatMetricCurrency(amount?: string) {
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

export function formatNumberCurrency(value?: number) {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return '--'
  }

  return new Intl.NumberFormat('zh-CN', {
    style: 'currency',
    currency: 'CNY',
    maximumFractionDigits: 2,
  }).format(value)
}

export function formatPercentValue(value?: string) {
  if (!value) {
    return '--'
  }

  const parsed = Number.parseFloat(value)
  if (Number.isNaN(parsed)) {
    return '--'
  }

  return `${parsed >= 0 ? '+' : ''}${parsed.toFixed(2)}%`
}

export function formatNumberPercent(value?: number) {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return '--'
  }

  return `${value >= 0 ? '+' : ''}${value.toFixed(2)}%`
}

export function parseMetricNumber(value?: string) {
  if (!value) {
    return null
  }

  const parsed = Number.parseFloat(value)
  return Number.isNaN(parsed) ? null : parsed
}

export function formatEstimatedDelta(amount: string, changePercent?: string) {
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

export function formatTradeAt(tradeAt?: string) {
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

export function normalizeDecimalInput(value: string) {
  return value.replace(/,/g, '').trim()
}

export function formatDateForInput(raw?: string) {
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

export function buildCorrectionTradeAt(date: string, originalTradeAt?: string) {
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

export interface CorrectionFormState {
  amount: string
  shares: string
  confirmedNav: string
  confirmedNavDate: string
  tradeDate: string
  note: string
  sourcePlatform: HoldingSourcePlatform | ''
}

export interface SellFormState {
  amount: string
  shares: string
  tradeDate: string
  note: string
  sellAll: boolean
}

export interface DividendFormState {
  amount: string
  shares: string
  tradeDate: string
  note: string
  reinvest: boolean
  sourcePlatform: HoldingSourcePlatform | ''
}

export interface AdjustmentFormState {
  targetShares: string
  sharesDelta: string
  confirmedNav: string
  confirmedNavDate: string
  tradeDate: string
  note: string
  sourcePlatform: HoldingSourcePlatform | ''
}

export function buildCorrectionFormState(holding: HoldingEntry): CorrectionFormState {
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

export function buildSellFormState(holding: HoldingEntry): SellFormState {
  return {
    amount: '',
    shares: '',
    tradeDate: formatDateForInput(new Date().toISOString()) || formatDateForInput(holding.trade_at || holding.as_of_date),
    note: '',
    sellAll: false,
  }
}

export function buildDividendFormState(holding: HoldingEntry): DividendFormState {
  return {
    amount: '',
    shares: '',
    tradeDate: formatDateForInput(new Date().toISOString()) || formatDateForInput(holding.trade_at || holding.as_of_date),
    note: '',
    reinvest: false,
    sourcePlatform: isHoldingSourcePlatform(holding.source_platform) ? holding.source_platform : '',
  }
}

export function buildAdjustmentFormState(holding: HoldingEntry): AdjustmentFormState {
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

export function validatePositiveDecimal(value: string) {
  const parsed = Number.parseFloat(normalizeDecimalInput(value))
  return Number.isFinite(parsed) && parsed > 0
}

export function validateNonZeroDecimal(value: string) {
  const parsed = Number.parseFloat(normalizeDecimalInput(value))
  return Number.isFinite(parsed) && parsed !== 0
}
