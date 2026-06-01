'use client'

import { AlertTriangle, Layers3, ListTree } from 'lucide-react'
import { cn } from '@/lib/utils'
import type {
  HoldingFilterMode,
  HoldingMetricScope,
  HoldingSortMode,
} from '@/lib/holding-display'

type HoldingViewMode = 'aggregate' | 'detail'

interface HoldingsViewControlsProps {
  viewMode: HoldingViewMode
  metricScope: HoldingMetricScope
  sortMode: HoldingSortMode
  filterMode: HoldingFilterMode
  showIncompleteOnly: boolean
  incompleteCount: number
  onViewModeChange: (mode: HoldingViewMode) => void
  onMetricScopeChange: (scope: HoldingMetricScope) => void
  onSortModeChange: (mode: HoldingSortMode) => void
  onFilterModeChange: (mode: HoldingFilterMode) => void
  onToggleIncompleteOnly: () => void
}

const viewModeOptions = [
  { id: 'aggregate', label: '按基金', icon: Layers3 },
  { id: 'detail', label: '分笔明细', icon: ListTree },
] as const

const metricScopeOptions = [
  { id: 'official', label: '官方口径' },
  { id: 'estimate', label: '盘中预估' },
] as const

const filterOptions = [
  { id: 'all', label: '全部' },
  { id: 'profit', label: '只看盈利' },
  { id: 'loss', label: '只看亏损' },
  { id: 'ready', label: '已完整就绪' },
  { id: 'partial', label: '部分/未就绪' },
  { id: 'single', label: '单笔基金' },
  { id: 'multiple', label: '多笔基金' },
] as const

function sortOptions(viewMode: HoldingViewMode) {
  return [
    { id: 'default', label: '默认顺序' },
    {
      id: 'principal_desc',
      label: viewMode === 'aggregate' ? '仓位最大优先' : '金额最大优先',
    },
    { id: 'profit_asc', label: '亏损最多优先' },
    { id: 'profit_desc', label: '盈利最多优先' },
    { id: 'change_asc', label: '跌幅最大优先' },
    { id: 'change_desc', label: '涨幅最大优先' },
    { id: 'count_desc', label: '分笔多优先' },
    { id: 'recent_desc', label: '最近录入优先' },
    { id: 'analysis_recommendation', label: '建议倾向优先' },
    { id: 'analysis_risk', label: '风险等级优先' },
  ] as const
}

export function HoldingsViewControls({
  viewMode,
  metricScope,
  sortMode,
  filterMode,
  showIncompleteOnly,
  incompleteCount,
  onViewModeChange,
  onMetricScopeChange,
  onSortModeChange,
  onFilterModeChange,
  onToggleIncompleteOnly,
}: HoldingsViewControlsProps) {
  return (
    <>
      <div className="mb-6 flex flex-col gap-3 rounded-[24px] border border-[var(--card-border)] bg-[var(--card-bg)]/66 p-4 lg:flex-row lg:items-center lg:justify-between">
        <div className="flex flex-wrap gap-2">
          {viewModeOptions.map((option) => (
            <button
              key={option.id}
              type="button"
              onClick={() => onViewModeChange(option.id)}
              className={cn(
                'inline-flex items-center gap-2 rounded-full border px-3 py-2 text-xs transition-all duration-200',
                viewMode === option.id
                  ? 'border-cyan-400/50 bg-cyan-400/14 text-cyan-100 shadow-[0_10px_22px_rgba(34,211,238,0.12)]'
                  : 'border-[var(--input-border)] bg-[var(--input-bg)] text-theme-secondary hover:border-cyan-400/35 hover:text-theme-primary',
              )}
            >
              <option.icon className="h-3.5 w-3.5" />
              <span>{option.label}</span>
            </button>
          ))}
        </div>

        <div className="flex flex-wrap gap-2">
          {metricScopeOptions.map((option) => (
            <button
              key={option.id}
              type="button"
              onClick={() => onMetricScopeChange(option.id)}
              className={cn(
                'rounded-full border px-3 py-2 text-xs transition-all duration-200',
                metricScope === option.id
                  ? 'border-amber-400/45 bg-amber-400/14 text-amber-100 shadow-[0_10px_22px_rgba(251,191,36,0.12)]'
                  : 'border-[var(--input-border)] bg-[var(--input-bg)] text-theme-secondary hover:border-amber-300/35 hover:text-theme-primary',
              )}
            >
              {option.label}
            </button>
          ))}
        </div>
      </div>

      <div className="mb-6 flex flex-col gap-3 rounded-[24px] border border-[var(--card-border)] bg-[var(--card-bg)]/50 p-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <div className="text-sm font-medium text-theme-primary">
            列表怎么排？
          </div>
          <div className="mt-1 text-xs text-theme-muted">
            只影响下方持仓列表，不改变任何数据。
          </div>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <div className="flex flex-wrap gap-2">
            {sortOptions(viewMode).map((option) => (
              <button
                key={option.id}
                type="button"
                onClick={() => onSortModeChange(option.id)}
                className={cn(
                  'rounded-full border px-3 py-2 text-xs transition-all duration-200',
                  sortMode === option.id
                    ? 'border-fuchsia-400/45 bg-fuchsia-400/14 text-fuchsia-100 shadow-[0_10px_22px_rgba(217,70,239,0.12)]'
                    : 'border-[var(--input-border)] bg-[var(--input-bg)] text-theme-secondary hover:border-fuchsia-300/35 hover:text-theme-primary',
                )}
              >
                {option.label}
              </button>
            ))}
          </div>
          <div className="flex flex-wrap gap-2 border-t border-[var(--card-border)] pt-2 lg:border-l lg:border-t-0 lg:pl-3 lg:pt-0">
            {filterOptions.map((option) => (
              <button
                key={option.id}
                type="button"
                onClick={() => onFilterModeChange(option.id)}
                className={cn(
                  'rounded-full border px-3 py-2 text-xs transition-all duration-200',
                  filterMode === option.id
                    ? 'border-cyan-400/45 bg-cyan-400/14 text-cyan-100 shadow-[0_10px_22px_rgba(34,211,238,0.12)]'
                    : 'border-[var(--input-border)] bg-[var(--input-bg)] text-theme-secondary hover:border-cyan-300/35 hover:text-theme-primary',
                )}
              >
                {option.label}
              </button>
            ))}
          </div>
          <button
            type="button"
            onClick={onToggleIncompleteOnly}
            className={cn(
              'inline-flex items-center gap-2 rounded-full border px-3 py-2 text-xs transition-all duration-200',
              showIncompleteOnly
                ? 'border-amber-400/50 bg-amber-400/14 text-amber-100 shadow-[0_10px_22px_rgba(251,191,36,0.12)]'
                : 'border-[var(--input-border)] bg-[var(--input-bg)] text-theme-secondary hover:border-amber-300/35 hover:text-theme-primary',
            )}
            aria-pressed={showIncompleteOnly}
          >
            <AlertTriangle className="h-3.5 w-3.5" />
            只看待补齐
            <span className="rounded-full border border-current/20 px-1.5 py-0.5 text-[10px]">
              {incompleteCount}
            </span>
          </button>
        </div>
      </div>
    </>
  )
}
