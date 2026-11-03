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
  info: 'border-accent/30 bg-accent/10 text-accent',
  warning: 'border-warning/30 bg-warning/10 text-warning',
  success: 'border-success/25 bg-success/10 text-success',
  danger: 'border-danger/30 bg-danger/10 text-danger',
}

const iconClass: Record<StatusBannerTone, string> = {
  info: 'text-accent',
  warning: 'text-warning',
  success: 'text-success',
  danger: 'text-danger',
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
        'rounded-xl border px-4 py-3 text-sm shadow-none',
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
