'use client'

import Link from 'next/link'
import type { ReactNode } from 'react'
import { Crown, LockKeyhole, Sparkles } from 'lucide-react'
import { cn } from '@/lib/utils'

interface VIPAnalysisAction {
  label: string
  href?: string
  onClick?: () => void
  variant?: 'primary' | 'secondary' | 'ghost'
  disabled?: boolean
  icon?: ReactNode
}

interface VIPAnalysisEntryProps {
  title: string
  description: string
  accent?: 'cyan' | 'amber'
  badgeLabel?: string
  quotaLabel?: string
  note?: string
  actions?: VIPAnalysisAction[]
}

export function VIPAnalysisEntry({
  title,
  description,
  accent = 'cyan',
  badgeLabel = 'VIP',
  quotaLabel,
  note,
  actions = [],
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
            {badgeLabel}
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

      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex flex-wrap gap-3">
          {actions.length > 0 ? actions.map((action, index) => {
            const content = (
              <>
                {action.icon}
                <span>{action.label}</span>
              </>
            )
            const className = cn(
              'inline-flex items-center gap-2 rounded-2xl border px-4 py-3 text-sm font-medium transition-all duration-200',
              'group relative overflow-hidden',
              (action.variant === 'primary' || action.variant === 'secondary') && 'vip-entry-action',
              action.variant === 'primary' && (
                accent === 'amber'
                  ? 'vip-entry-action-amber border-amber-300/45 bg-gradient-to-r from-amber-400/22 via-orange-400/14 to-yellow-300/18 text-amber-50 shadow-[0_16px_34px_rgba(251,191,36,0.12)] hover:shadow-[0_22px_46px_rgba(251,191,36,0.18)]'
                  : 'vip-entry-action-cyan border-cyan-300/45 bg-gradient-to-r from-cyan-400/22 via-sky-400/14 to-blue-300/18 text-cyan-50 shadow-[0_16px_34px_rgba(34,211,238,0.12)] hover:shadow-[0_22px_46px_rgba(34,211,238,0.18)]'
              ),
              action.variant === 'secondary' && 'vip-entry-action-secondary border-[var(--input-border)] bg-[var(--input-bg)] text-theme-primary hover:border-cyan-400/30',
              action.variant === 'ghost' && 'border-transparent bg-transparent text-theme-secondary hover:bg-[var(--input-bg)]',
              action.disabled && 'cursor-not-allowed opacity-60 hover:bg-[var(--input-bg)]'
            )

            if (action.href) {
              return (
                <Link key={`${action.label}-${index}`} href={action.href} className={className}>
                  {(action.variant === 'primary' || action.variant === 'secondary') && <span className="vip-entry-action-shine" />}
                  {content}
                </Link>
              )
            }

            return (
              <button
                key={`${action.label}-${index}`}
                type="button"
                disabled={action.disabled}
                onClick={action.onClick}
                className={className}
              >
                {(action.variant === 'primary' || action.variant === 'secondary') && <span className="vip-entry-action-shine" />}
                {content}
              </button>
            )
          }) : (
            <button
              type="button"
              disabled
              className="inline-flex items-center gap-2 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-sm font-medium text-theme-secondary opacity-80"
            >
              <LockKeyhole className="h-4 w-4" />
              VIP 功能预留中
            </button>
          )}
        </div>

        <div className="space-y-1 text-right">
          {quotaLabel && (
            <div className="text-xs tracking-[0.14em] text-theme-muted">{quotaLabel}</div>
          )}
          {note && (
            <div className="text-xs leading-5 text-theme-secondary">{note}</div>
          )}
        </div>
      </div>
    </div>
  )
}
