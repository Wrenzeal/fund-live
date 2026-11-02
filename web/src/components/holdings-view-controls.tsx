'use client'

import type { HoldingFilterMode, HoldingSortMode } from '@/lib/holding-display'
import { cn } from '@/lib/utils'

export function HoldingsViewControls({ viewMode, sortMode, filterMode, showIncompleteOnly, incompleteCount, onViewModeChange, onSortModeChange, onFilterModeChange, onToggleIncompleteOnly }: {
  viewMode: 'aggregate' | 'detail'; sortMode: HoldingSortMode; filterMode: HoldingFilterMode
  showIncompleteOnly: boolean; incompleteCount: number
  onViewModeChange: (mode: 'aggregate' | 'detail') => void
  onSortModeChange: (mode: HoldingSortMode) => void; onFilterModeChange: (mode: HoldingFilterMode) => void; onToggleIncompleteOnly: () => void
}) {
  const selectClass = 'min-h-11 rounded-lg border border-[var(--input-border)] bg-[var(--input-bg)] px-3 text-sm text-theme-primary'
  return <div className="space-y-3" aria-label="持仓列表选项">
    <div className="flex flex-wrap items-center justify-between gap-3">
      <div className="inline-flex gap-1 rounded-lg bg-[var(--input-bg)] p-1">{([{ id: 'aggregate', label: '按基金' }, { id: 'detail', label: '分笔明细' }] as const).map(item => <button type="button" key={item.id} aria-pressed={viewMode === item.id} onClick={() => onViewModeChange(item.id)} className={cn('min-h-11 rounded-md px-3 text-sm', viewMode === item.id ? 'bg-[var(--card-bg)] font-semibold text-theme-primary' : 'text-theme-secondary')}>{item.label}</button>)}</div>
      <label className="flex items-center gap-2 text-sm text-theme-secondary">排序<select aria-label="持仓排序" className={selectClass} value={sortMode} onChange={event => onSortModeChange(event.target.value as HoldingSortMode)}>{[
        ['default', '默认顺序'], ['principal_desc', '剩余成本最多'], ['profit_desc', '当日增加最多'], ['profit_asc', '当日减少最多'], ['change_desc', '涨幅最大'], ['change_asc', '跌幅最大'], ['recent_desc', '最近记录'], ['count_desc', '记录笔数最多'],
      ].map(([id, label]) => <option value={id} key={id}>{label}</option>)}</select></label>
    </div>
    <details className="text-sm text-theme-secondary"><summary className="min-h-11 cursor-pointer py-3">筛选{filterMode !== 'all' || showIncompleteOnly ? ' · 已启用' : ''}</summary><div className="flex flex-wrap items-center gap-4 pb-3">
      <label className="flex items-center gap-2">范围<select aria-label="持仓筛选" className={selectClass} value={filterMode} onChange={event => onFilterModeChange(event.target.value as HoldingFilterMode)}>{[['all', '全部'], ['profit', '当日增加'], ['loss', '当日减少'], ['ready', '数据齐全'], ['partial', '待确认或待更新'], ['single', '只有一笔'], ['multiple', '多笔记录']].map(([id, label]) => <option key={id} value={id}>{label}</option>)}</select></label>
      <label className="inline-flex min-h-11 items-center gap-2"><input type="checkbox" checked={showIncompleteOnly} onChange={onToggleIncompleteOnly} />仅看待处理（{incompleteCount}）</label>
    </div></details>
  </div>
}
