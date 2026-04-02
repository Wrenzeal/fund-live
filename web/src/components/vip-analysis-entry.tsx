'use client'

import { Crown, LockKeyhole, Sparkles } from 'lucide-react'
import { cn } from '@/lib/utils'

interface VIPAnalysisEntryProps {
  title: string
  description: string
  accent?: 'cyan' | 'amber'
}

export function VIPAnalysisEntry({
  title,
  description,
  accent = 'cyan',
}: VIPAnalysisEntryProps) {
  return (
    <div className={cn(
      'rounded-[28px] border p-6 glass',
      accent === 'amber' ? 'border-amber-500/20' : 'border-cyan-500/20'
    )}>
      <div className="mb-5 flex items-start justify-between gap-4">
        <div className="space-y-3">
          <div className={cn(
            'inline-flex items-center gap-2 rounded-full px-3 py-1 text-xs tracking-[0.25em]',
            accent === 'amber'
              ? 'border border-amber-500/30 bg-amber-500/10 text-amber-300'
              : 'border border-cyan-500/30 bg-cyan-500/10 text-cyan-300'
          )}>
            <Crown className="h-3.5 w-3.5" />
            VIP
          </div>
          <div>
            <h3 className="text-xl font-bold text-theme-primary">{title}</h3>
            <p className="mt-2 max-w-2xl text-sm leading-6 text-theme-secondary">{description}</p>
          </div>
        </div>

        <div className="hidden rounded-2xl bg-[var(--input-bg)] p-3 text-theme-secondary md:block">
          <Sparkles className="h-5 w-5" />
        </div>
      </div>

      <button
        type="button"
        disabled
        className="inline-flex items-center gap-2 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-sm font-medium text-theme-secondary opacity-80"
      >
        <LockKeyhole className="h-4 w-4" />
        VIP 功能预留中
      </button>
    </div>
  )
}
