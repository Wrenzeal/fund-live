import { ArrowLeft, Plus } from 'lucide-react'
import { ActionButton } from '@/components/ui/action-button'
import { cn } from '@/lib/utils'

export type HoldingWorkspaceTab = 'summary' | 'record' | 'list' | 'risk' | 'ledger' | 'tools'

const destinations = [
  { id: 'summary', label: '持仓' }, { id: 'ledger', label: '交易记录' },
  { id: 'risk', label: '复核与提醒' }, { id: 'tools', label: '导入与对账' },
] as const

export function HoldingsWorkspaceNav({ activeTab, onTabChange, holdingCount }: {
  activeTab: HoldingWorkspaceTab
  onTabChange: (tab: HoldingWorkspaceTab) => void
  holdingCount: number
}) {
  return <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
    <nav aria-label="持仓工作区" className="flex min-w-0 gap-1 overflow-x-auto whitespace-nowrap">
      {destinations.map(tab => <button key={tab.id} type="button" aria-current={activeTab === tab.id || activeTab === 'list' && tab.id === 'summary' ? 'page' : undefined} onClick={() => onTabChange(tab.id)} className={cn('min-h-11 rounded-lg px-3 text-sm transition-colors', activeTab === tab.id || activeTab === 'list' && tab.id === 'summary' ? 'bg-[var(--input-bg)] font-semibold text-theme-primary' : 'text-theme-secondary hover:bg-[var(--input-bg)]')}>{tab.label}</button>)}
    </nav>
    {activeTab === 'record' ? <ActionButton className="min-h-11" onClick={() => onTabChange('summary')}><ArrowLeft className="h-4 w-4" />返回持仓</ActionButton> : holdingCount > 0 && <ActionButton className="holding-primary-action min-h-11" variant="primary" onClick={() => onTabChange('record')}><Plus className="h-4 w-4" />记一笔</ActionButton>}
  </div>
}
