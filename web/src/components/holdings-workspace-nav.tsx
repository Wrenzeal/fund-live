import { BarChart4, BellRing, ClipboardList, HeartPulse, History, LoaderCircle, Wallet, type LucideIcon } from 'lucide-react'

import { Surface } from '@/components/ui/surface'
import { cn } from '@/lib/utils'

export type HoldingWorkspaceTab = 'summary' | 'record' | 'list' | 'risk' | 'ledger' | 'tools'

interface WorkspaceTabConfig {
  id: HoldingWorkspaceTab
  label: string
  description: string
  icon: LucideIcon
}

interface QuickActionConfig {
  label: string
  tab: HoldingWorkspaceTab
  primary?: boolean
}

interface HoldingsWorkspaceNavProps {
  holdingCount: number
  activeTab: HoldingWorkspaceTab
  detailText: string
  isSeedingDemo: boolean
  onTabChange: (tab: HoldingWorkspaceTab) => void
  onSeedDemo: () => void
}

const workspaceTabs: WorkspaceTabConfig[] = [
  {
    id: 'summary',
    label: '总览',
    description: '价值、收益、对账',
    icon: Wallet,
  },
  {
    id: 'record',
    label: '记录',
    description: '新增/补仓',
    icon: ClipboardList,
  },
  {
    id: 'list',
    label: '持仓',
    description: '排序、筛选、操作',
    icon: BarChart4,
  },
  { id: 'risk', label: '风险', description: '体检和提醒', icon: HeartPulse },
  {
    id: 'ledger',
    label: '流水',
    description: '买卖、校正、分红',
    icon: History,
  },
  { id: 'tools', label: '工具', description: '批量导入/VIP', icon: BellRing },
]

const quickActions: QuickActionConfig[] = [
  { label: '记一笔', tab: 'record', primary: true },
  { label: '看持仓', tab: 'list' },
  { label: '查风险', tab: 'risk' },
  { label: '看流水', tab: 'ledger' },
]

export function HoldingsWorkspaceNav({
  holdingCount,
  activeTab,
  detailText,
  isSeedingDemo,
  onTabChange,
  onSeedDemo,
}: HoldingsWorkspaceNavProps) {
  return (
    <Surface as="section" radius="xl" padding="md">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <div className="text-sm text-theme-muted">持仓工作台</div>
          <div className="mt-1 text-3xl font-black text-theme-primary">
            {holdingCount} 条持仓
          </div>
          <div className="mt-2 text-xs text-theme-muted">{detailText}</div>
        </div>

        <div className="flex flex-wrap gap-2">
          {quickActions.map((action) => (
            <button
              key={action.tab}
              type="button"
              onClick={() => onTabChange(action.tab)}
              className={cn(
                'rounded-2xl border px-4 py-2 text-sm font-medium transition-all duration-200 active:scale-[0.98]',
                activeTab === action.tab
                  ? 'border-cyan-300/45 bg-cyan-400/14 text-cyan-100 shadow-[0_12px_26px_rgba(34,211,238,0.12)]'
                  : action.primary
                    ? 'border-cyan-300/30 bg-cyan-400/10 text-cyan-100 hover:border-cyan-200/45'
                    : 'border-[var(--input-border)] bg-[var(--input-bg)] text-theme-secondary hover:border-cyan-300/35 hover:text-theme-primary',
              )}
            >
              {action.label}
            </button>
          ))}
          {holdingCount === 0 && (
            <button
              type="button"
              onClick={onSeedDemo}
              disabled={isSeedingDemo}
              className={cn(
                'group relative inline-flex items-center gap-2 overflow-hidden rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-2 text-sm text-theme-secondary transition-all duration-200 active:scale-[0.98] disabled:pointer-events-none disabled:opacity-60',
                'hover:-translate-y-0.5 hover:border-cyan-400/45 hover:text-theme-primary',
                isSeedingDemo && 'holding-action-button',
              )}
            >
              <span className="holding-action-shine" />
              {isSeedingDemo ? (
                <LoaderCircle className="relative z-10 h-4 w-4 animate-spin" />
              ) : (
                <BarChart4 className="relative z-10 h-4 w-4" />
              )}
              <span className="relative z-10">
                {isSeedingDemo ? '准备中...' : '快速开始'}
              </span>
            </button>
          )}
        </div>
      </div>

      {holdingCount > 0 && (
        <div className="mt-5 grid gap-3 md:grid-cols-3 xl:grid-cols-6">
          {workspaceTabs.map((tab) => {
            const Icon = tab.icon
            return (
              <button
                key={tab.id}
                type="button"
                onClick={() => onTabChange(tab.id)}
                className={cn(
                  'rounded-[20px] border p-3 text-left transition-all duration-200 active:scale-[0.99]',
                  activeTab === tab.id
                    ? 'border-cyan-300/45 bg-cyan-400/14 text-cyan-100 shadow-[0_14px_30px_rgba(34,211,238,0.12)]'
                    : 'border-[var(--card-border)] bg-[var(--card-bg)]/56 text-theme-secondary hover:border-cyan-300/30 hover:text-theme-primary',
                )}
              >
                <div className="flex items-center gap-2 text-sm font-semibold">
                  <Icon className="h-4 w-4" />
                  {tab.label}
                </div>
                <div className="mt-1 text-[11px] text-theme-muted">
                  {tab.description}
                </div>
              </button>
            )
          })}
        </div>
      )}
    </Surface>
  )
}
