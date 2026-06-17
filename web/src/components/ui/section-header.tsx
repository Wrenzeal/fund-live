import type { ReactNode } from 'react'

import { cn } from '@/lib/utils'

interface SectionHeaderProps {
  title: string
  description?: ReactNode
  eyebrow?: ReactNode
  action?: ReactNode
  className?: string
}

export function SectionHeader({ title, description, eyebrow, action, className }: SectionHeaderProps) {
  return (
    <div className={cn('flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between', className)}>
      <div className="min-w-0">
        {eyebrow && <div className="mb-3 text-sm font-medium text-theme-muted">{eyebrow}</div>}
        <h2 className="text-2xl font-semibold tracking-tight text-theme-primary text-balance sm:text-3xl">
          {title}
        </h2>
        {description && (
          <div className="mt-3 max-w-3xl text-sm leading-6 text-theme-secondary text-pretty">
            {description}
          </div>
        )}
      </div>
      {action && <div className="shrink-0">{action}</div>}
    </div>
  )
}
