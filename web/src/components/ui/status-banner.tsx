import type { ReactNode } from 'react'
import { AlertTriangle, CheckCircle2, Info, X } from 'lucide-react'

import { cn } from '@/lib/utils'

type StatusBannerTone = 'info' | 'warning' | 'success' | 'danger'

interface StatusBannerProps {
  tone?: StatusBannerTone
  children: ReactNode
  className?: string
  icon?: ReactNode
  action?: ReactNode
  onDismiss?: () => void
  dismissLabel?: string
}

const toneClass: Record<StatusBannerTone, string> = {
  info: 'border-cyan-500/30 bg-cyan-500/10 text-cyan-50',
  warning: 'border-amber-500/30 bg-amber-500/10 text-amber-100',
  success: 'border-emerald-500/25 bg-emerald-500/10 text-emerald-50',
  danger: 'border-rose-500/30 bg-rose-500/10 text-rose-50',
}

const iconClass: Record<StatusBannerTone, string> = {
  info: 'text-cyan-200',
  warning: 'text-amber-100',
  success: 'text-emerald-200',
  danger: 'text-rose-100',
}

const defaultIcon: Record<StatusBannerTone, ReactNode> = {
  info: <Info className="h-4 w-4" />,
  warning: <AlertTriangle className="h-4 w-4" />,
  success: <CheckCircle2 className="h-4 w-4" />,
  danger: <AlertTriangle className="h-4 w-4" />,
}

export function StatusBanner({
  tone = 'info',
  children,
  className,
  icon,
  action,
  onDismiss,
  dismissLabel = '关闭提示',
}: StatusBannerProps) {
  return (
    <div
      className={cn(
        'rounded-2xl border px-4 py-3 text-sm shadow-[0_14px_34px_rgba(2,8,23,0.10)]',
        toneClass[tone],
        className,
      )}
    >
      <div className="flex items-start justify-between gap-4">
        <div className="flex min-w-0 items-start gap-3">
          <span className={cn('mt-0.5 shrink-0', iconClass[tone])}>{icon ?? defaultIcon[tone]}</span>
          <div className="min-w-0 leading-6 text-pretty">{children}</div>
        </div>
        {action}
        {onDismiss && (
          <button
            type="button"
            onClick={onDismiss}
            className="shrink-0 opacity-70 transition hover:opacity-100 active:scale-[0.97]"
            aria-label={dismissLabel}
          >
            <X className="h-4 w-4" />
          </button>
        )}
      </div>
    </div>
  )
}
