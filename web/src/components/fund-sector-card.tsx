'use client'

import { Layers3 } from 'lucide-react'
import type { Fund, FundSectorSnapshot } from '@/hooks/use-fund-data'

interface FundSectorCardProps {
  fund?: Fund
  sectorSnapshot?: FundSectorSnapshot
}

function formatWeight(value?: string) {
  const parsed = Number.parseFloat(value || '')
  if (Number.isNaN(parsed)) {
    return '--'
  }
  return `${parsed.toFixed(1)}%`
}

export function FundSectorCard({ fund, sectorSnapshot }: FundSectorCardProps) {
  if (!sectorSnapshot || sectorSnapshot.breakdown.length === 0) {
    return null
  }

  return (
    <div className="glass rounded-2xl p-6">
      <div className="mb-4 flex items-center gap-3">
        <div className="rounded-lg bg-cyan-500/20 p-2">
          <Layers3 className="h-5 w-5 text-cyan-300" />
        </div>
        <div>
          <div className="text-sm font-semibold text-theme-primary">持仓分类</div>
          <div className="text-xs text-theme-muted">
            快照日期：{sectorSnapshot.as_of_date}
          </div>
        </div>
      </div>

      {fund?.category_name && (
        <div className="mb-4 inline-flex items-center gap-2 rounded-full border border-[var(--card-border)] bg-[var(--input-bg)]/60 px-3 py-1 text-xs text-theme-secondary">
          主分类：<span className="font-medium text-theme-primary">{fund.category_name}</span>
        </div>
      )}

      <div className="mb-4 rounded-2xl border border-cyan-500/20 bg-cyan-500/10 px-4 py-3">
        <div className="text-xs tracking-[0.18em] text-theme-muted">主板块</div>
        <div className="mt-2 text-xl font-bold text-theme-primary">{sectorSnapshot.primary_sector_name}</div>
      </div>

      <div className="space-y-3">
        {sectorSnapshot.breakdown.map((item) => (
          <div
            key={item.sector_code}
            className="flex items-center justify-between rounded-xl border border-[var(--card-border)] bg-[var(--input-bg)]/50 px-4 py-3"
          >
            <div className="min-w-0">
              <div className="text-sm font-medium text-theme-primary">{item.sector_name}</div>
              <div className="mt-1 text-xs text-theme-muted">Top {item.rank}</div>
            </div>
            <div className="text-sm font-semibold text-cyan-200">{formatWeight(item.weight_percent)}</div>
          </div>
        ))}
      </div>
    </div>
  )
}
