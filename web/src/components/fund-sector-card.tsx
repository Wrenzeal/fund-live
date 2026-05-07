'use client'

import { Layers3 } from 'lucide-react'
import type { Fund, FundSectorSnapshot, FundThemeSnapshot } from '@/hooks/use-fund-data'

interface FundSectorCardProps {
  fund?: Fund
  sectorSnapshot?: FundSectorSnapshot
  themeSnapshot?: FundThemeSnapshot
}

interface ClassificationModuleProps {
  title: string
  badge: string
  primaryLabel: string
  primaryValue: string
  tone: 'cyan' | 'fuchsia'
  items: Array<{
    code: string
    name: string
    weight: string
    rank: number
  }>
}

function formatWeight(value?: string) {
  const parsed = Number.parseFloat(value || '')
  if (Number.isNaN(parsed)) {
    return '--'
  }
  return `${parsed.toFixed(1)}%`
}

function confidenceLabel(confidence?: string) {
  switch (confidence) {
    case 'high':
      return '识别覆盖较高'
    case 'medium':
      return '识别覆盖一般'
    case 'low':
      return '识别覆盖有限'
    default:
      return ''
  }
}

function ClassificationModule({
  title,
  badge,
  primaryLabel,
  primaryValue,
  tone,
  items,
}: ClassificationModuleProps) {
  const toneStyles = tone === 'cyan'
    ? {
        shell: 'border-cyan-500/20 bg-cyan-500/10',
        value: 'text-cyan-200',
        iconText: 'text-cyan-300',
      }
    : {
        shell: 'border-fuchsia-500/20 bg-fuchsia-500/10',
        value: 'text-fuchsia-200',
        iconText: 'text-fuchsia-300',
      }

  return (
    <section className="rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/45 p-4">
      <div className="mb-4 flex items-center justify-between gap-3">
        <div>
          <div className="text-sm font-semibold text-theme-primary">{title}</div>
          <div className="mt-1 text-xs text-theme-muted">{badge}</div>
        </div>
      </div>

      <div className={`mb-4 rounded-2xl border px-4 py-3 ${toneStyles.shell}`}>
        <div className="text-xs tracking-[0.18em] text-theme-muted">{primaryLabel}</div>
        <div className="mt-2 text-xl font-bold text-theme-primary">{primaryValue}</div>
      </div>

      <div className="space-y-3">
        {items.map((item) => (
          <div
            key={item.code}
            className="flex items-center justify-between rounded-xl border border-[var(--card-border)] bg-[var(--input-bg)]/50 px-4 py-3"
          >
            <div className="min-w-0">
              <div className="text-sm font-medium text-theme-primary">{item.name}</div>
              <div className="mt-1 text-xs text-theme-muted">Top {item.rank}</div>
            </div>
            <div className={`text-sm font-semibold ${toneStyles.value}`}>{item.weight}</div>
          </div>
        ))}
      </div>
    </section>
  )
}

export function FundSectorCard({ fund, sectorSnapshot, themeSnapshot }: FundSectorCardProps) {
  if ((!sectorSnapshot || sectorSnapshot.breakdown.length === 0) && (!themeSnapshot || themeSnapshot.breakdown.length === 0)) {
    return null
  }

  const snapshotDate = sectorSnapshot?.as_of_date || themeSnapshot?.as_of_date || '--'
  const sectorConfidence = confidenceLabel(sectorSnapshot?.confidence)
  const themeConfidence = confidenceLabel(themeSnapshot?.confidence)
  const showCoverageHint = sectorSnapshot?.confidence === 'low' || themeSnapshot?.confidence === 'low'
  const modules = [
    sectorSnapshot && sectorSnapshot.breakdown.length > 0
      ? (
        <ClassificationModule
          key="sector"
          title="行业板块"
          badge="按持仓聚合的行业暴露"
          primaryLabel="主板块"
          primaryValue={sectorSnapshot.primary_sector_name}
          tone="cyan"
          items={sectorSnapshot.breakdown.map((item) => ({
            code: item.sector_code,
            name: item.sector_name,
            weight: formatWeight(item.weight_percent),
            rank: item.rank,
          }))}
        />
      )
      : null,
    themeSnapshot && themeSnapshot.breakdown.length > 0
      ? (
        <ClassificationModule
          key="theme"
          title="主题分类"
          badge="按持仓聚合的主题暴露"
          primaryLabel="主主题"
          primaryValue={themeSnapshot.primary_theme_name}
          tone="fuchsia"
          items={themeSnapshot.breakdown.map((item) => ({
            code: item.theme_code,
            name: item.theme_name,
            weight: formatWeight(item.weight_percent),
            rank: item.rank,
          }))}
        />
      )
      : null,
  ].filter(Boolean)

  return (
    <div className="glass rounded-2xl p-6">
      <div className="mb-4 flex items-center gap-3">
        <div className="rounded-lg bg-cyan-500/20 p-2">
          <Layers3 className="h-5 w-5 text-cyan-300" />
        </div>
        <div>
          <div className="text-sm font-semibold text-theme-primary">持仓分类</div>
          <div className="text-xs text-theme-muted">
            快照日期：{snapshotDate}
          </div>
        </div>
      </div>

      {fund?.category_name && (
        <div className="mb-4 inline-flex items-center gap-2 rounded-full border border-[var(--card-border)] bg-[var(--input-bg)]/60 px-3 py-1 text-xs text-theme-secondary">
          主分类：<span className="font-medium text-theme-primary">{fund.category_name}</span>
        </div>
      )}

      {showCoverageHint && (
        <div className="mb-4 rounded-2xl border border-amber-500/20 bg-amber-500/10 px-4 py-3 text-sm text-amber-100">
          当前分类只覆盖了部分持仓，未归类部分仍然较高；结果更适合作为参考，不宜当作绝对结论。
        </div>
      )}

      <div className={`grid gap-4 ${modules.length > 1 ? 'xl:grid-cols-2' : ''}`}>
        {modules}
      </div>

      {(sectorConfidence || themeConfidence) && (
        <div className="mt-4 flex flex-wrap gap-2 text-xs text-theme-muted">
          {sectorConfidence && <span>行业板块：{sectorConfidence}</span>}
          {themeConfidence && <span>主题分类：{themeConfidence}</span>}
        </div>
      )}
    </div>
  )
}
