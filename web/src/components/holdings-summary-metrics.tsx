'use client'

import type { HoldingEstimateSummary, HoldingSummary } from '@/hooks/use-user-portfolio'
import { cn } from '@/lib/utils'
import { formatSummaryMoney, formatSummaryPercent, type HoldingMetricScope } from '@/lib/holding-display'

interface HoldingsSummaryMetricsProps {
  holdingSummary: HoldingSummary
  previewSummary: HoldingEstimateSummary
  metricScope: HoldingMetricScope
  shouldUseOfficialSummary: boolean
  hasOfficialSummaryMetrics: boolean
  hasPreviewSummaryMetrics: boolean
  officialReadyPrincipalText: string
  previewReadyPrincipalText: string
  totalPrincipalText: string
  officialSummaryCoverage: string
}

function metricTone(isReady: boolean, value?: string) {
  if (!isReady) {
    return 'text-theme-primary'
  }
  return Number.parseFloat(value || '0') >= 0 ? 'text-up' : 'text-down'
}

function metricToneFromNumber(isReady: boolean, value: number | null) {
  if (!isReady || value === null) {
    return 'text-theme-primary'
  }
  return value >= 0 ? 'text-up' : 'text-down'
}

function parseSummaryNumber(value?: string) {
  if (!value) {
    return null
  }

  const parsed = Number.parseFloat(value)
  return Number.isNaN(parsed) ? null : parsed
}

function formatSignedSummaryMoney(value: number | null) {
  if (value === null) {
    return '--'
  }

  const formatted = formatSummaryMoney(value.toFixed(2))
  if (formatted === '--' || value < 0) {
    return formatted
  }
  return `+${formatted}`
}

function formatReturnPercent(returnValue: number | null, principal?: string) {
  const principalValue = parseSummaryNumber(principal)
  if (returnValue === null || principalValue === null || principalValue <= 0) {
    return '--'
  }
  return formatSummaryPercent(((returnValue / principalValue) * 100).toFixed(4))
}

export function HoldingsSummaryMetrics({
  holdingSummary,
  previewSummary,
  metricScope,
  shouldUseOfficialSummary,
  hasOfficialSummaryMetrics,
  hasPreviewSummaryMetrics,
  officialReadyPrincipalText,
  previewReadyPrincipalText,
  totalPrincipalText,
  officialSummaryCoverage,
}: HoldingsSummaryMetricsProps) {
  const activeReady = shouldUseOfficialSummary ? hasOfficialSummaryMetrics : hasPreviewSummaryMetrics
  const activeTodayProfit = shouldUseOfficialSummary ? holdingSummary.total_today_profit : previewSummary.total_today_profit
  const activeChange = shouldUseOfficialSummary ? holdingSummary.total_today_change_percent : previewSummary.total_today_change_percent
  const activeValue = shouldUseOfficialSummary ? holdingSummary.total_current_market_value : previewSummary.total_current_market_value
  const activePrincipal = shouldUseOfficialSummary ? holdingSummary.ready_principal : previewSummary.ready_principal
  const activeValueNumber = parseSummaryNumber(activeValue)
  const activePrincipalNumber = parseSummaryNumber(activePrincipal)
  const totalReturn = activeReady && activeValueNumber !== null && activePrincipalNumber !== null
    ? activeValueNumber - activePrincipalNumber
    : null
  const totalReturnPercent = formatReturnPercent(totalReturn, activePrincipal)
  const metrics = [
    {
      label: '总本金',
      value: totalPrincipalText,
      tone: 'text-theme-primary',
      note: '按录入持仓本金汇总',
    },
    {
      label: shouldUseOfficialSummary ? '总价值（官方）' : '总价值（盘中）',
      value: shouldUseOfficialSummary
        ? (hasOfficialSummaryMetrics ? formatSummaryMoney(holdingSummary.total_current_market_value) : '--')
        : (hasPreviewSummaryMetrics ? formatSummaryMoney(previewSummary.total_current_market_value) : '--'),
      tone: 'text-theme-primary',
      note: shouldUseOfficialSummary
        ? (hasOfficialSummaryMetrics
          ? `已就绪本金 ${officialReadyPrincipalText} / ${totalPrincipalText}`
          : '待真实净值与份额齐备')
        : (hasPreviewSummaryMetrics
          ? `已确认本金 ${previewReadyPrincipalText} / ${totalPrincipalText}`
          : '待确认份额齐备'),
    },
    {
      label: '总收益',
      value: activeReady ? formatSignedSummaryMoney(totalReturn) : '--',
      tone: metricToneFromNumber(activeReady, totalReturn),
      note: activeReady
        ? (shouldUseOfficialSummary
          ? `总价值 - 已就绪本金，收益率 ${totalReturnPercent}`
          : `盘中总价值 - 已确认本金，收益率 ${totalReturnPercent}`)
        : (shouldUseOfficialSummary ? '待真实净值与份额齐备' : '待确认份额齐备'),
    },
    {
      label: shouldUseOfficialSummary ? '总官方盈亏' : '总实时盈亏预估',
      value: shouldUseOfficialSummary
        ? (hasOfficialSummaryMetrics ? formatSummaryMoney(holdingSummary.total_today_profit) : '--')
        : (hasPreviewSummaryMetrics ? formatSummaryMoney(previewSummary.total_today_profit) : '--'),
      tone: metricTone(activeReady, activeTodayProfit),
      note: shouldUseOfficialSummary
        ? (hasOfficialSummaryMetrics
          ? `官方口径：已就绪 ${officialSummaryCoverage}`
          : '暂不混入盘中预估')
        : (hasPreviewSummaryMetrics
          ? '根据基金预估涨跌幅估算，夜间真实值会自动覆盖'
          : '仅在确认份额后纳入预估'),
    },
    {
      label: shouldUseOfficialSummary ? '总官方涨跌幅' : '总实时涨跌预估',
      value: shouldUseOfficialSummary
        ? (hasOfficialSummaryMetrics ? formatSummaryPercent(holdingSummary.total_today_change_percent) : '--')
        : (hasPreviewSummaryMetrics ? formatSummaryPercent(previewSummary.total_today_change_percent) : '--'),
      tone: metricTone(activeReady, activeChange),
      note: shouldUseOfficialSummary
        ? (hasOfficialSummaryMetrics
          ? (holdingSummary.metrics_scope === 'partial'
            ? `按已就绪部分加权（剩余 ${holdingSummary.incomplete_holdings_count ?? 0} 条待补齐）`
            : '按总仓位加权计算')
          : `已就绪 ${holdingSummary.real_metrics_ready_count}/${holdingSummary.total_holdings} 条`)
        : (hasPreviewSummaryMetrics
          ? (previewSummary.metrics_scope === 'partial'
            ? `按确认份额部分加权（剩余 ${previewSummary.total_count - previewSummary.ready_count} 只基金待补齐）`
            : '夜间真实涨跌幅同步后会自动覆盖')
          : `已就绪 ${previewSummary.ready_count}/${previewSummary.total_count} 只基金`),
    },
  ]

  return (
    <div className="mb-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-5" data-metric-scope={metricScope}>
      {metrics.map((metric) => (
        <div key={metric.label} className="rounded-[24px] border border-[var(--card-border)] bg-[var(--card-bg)]/76 p-4">
          <div className="text-xs text-theme-muted">{metric.label}</div>
          <div className={cn('mt-3 text-2xl font-black', metric.tone)}>{metric.value}</div>
          <div className="mt-2 text-xs text-theme-secondary">{metric.note}</div>
        </div>
      ))}
    </div>
  )
}
