import type { ResearchSourceMeta, ResearchStudy } from '../hooks/use-market-research'

export type WorkspaceSource = { alias: string; symbol: string; field: 'close' | 'open' | 'high' | 'low' | 'volume' }
export type WorkspaceCurve = { id: string; name: string; expression: string }
export type WorkspaceCondition = { left: string; operator: 'gt' | 'lt' | 'cross_up' | 'cross_down'; right: string }
export type WorkspaceConfig = {
  name: string
  sources: WorkspaceSource[]
  curves: WorkspaceCurve[]
  event: { mode: 'all' | 'any'; conditions: WorkspaceCondition[]; direction: 'up' | 'down' }
  target_symbol: string
  holding: number
  fee_bps: number
  limit: number
}
// The existing study fields and accounting semantics are preserved; formula studies have their own rule identity.
export type WorkspaceStudy = Omit<ResearchStudy, 'rule'> & { rule: 'workspace' }
export type WorkspaceResult = {
  generated_at: string
  rows: { date: string; values: Record<string, number | null>; signal: boolean; state: boolean | null }[]
  sources: { alias: string; symbol: string; meta: ResearchSourceMeta }[]
  study: WorkspaceStudy
  warnings: string[]
}
