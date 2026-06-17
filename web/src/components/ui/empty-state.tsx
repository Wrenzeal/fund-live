import type { ReactNode } from 'react'

import { Surface } from '@/components/ui/surface'
import { cn } from '@/lib/utils'

interface EmptyStateFeature {
  title: string
  description: string
}

interface EmptyStateProps {
  icon?: ReactNode
  title: string
  description?: ReactNode
  features?: EmptyStateFeature[]
  action?: ReactNode
  className?: string
}

export function EmptyState({ icon, title, description, features, action, className }: EmptyStateProps) {
  return (
    <Surface tone="dashed" padding="lg" radius="xl" className={cn('text-center', className)}>
      {icon && <div className="mx-auto flex justify-center text-theme-muted">{icon}</div>}
      <div className="mt-4 text-xl font-semibold text-theme-primary text-balance">{title}</div>
      {description && (
        <div className="mx-auto mt-2 max-w-2xl text-sm leading-6 text-theme-secondary text-pretty">
          {description}
        </div>
      )}
      {features && features.length > 0 && (
        <div className="mt-6 grid gap-3 md:grid-cols-3">
          {features.map((item) => (
            <div
              key={item.title}
              className="rounded-2xl border border-[var(--card-border)] bg-[var(--input-bg)]/66 px-4 py-4 text-left"
            >
              <div className="text-sm font-semibold text-theme-primary">{item.title}</div>
              <div className="mt-2 text-xs leading-5 text-theme-secondary">{item.description}</div>
            </div>
          ))}
        </div>
      )}
      {action && <div className="mt-5">{action}</div>}
    </Surface>
  )
}
