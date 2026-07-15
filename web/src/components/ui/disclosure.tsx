import type { ReactNode } from 'react'

import { ChevronDown } from 'lucide-react'

import { cn } from '@/lib/utils'

interface DisclosureProps {
  summary: ReactNode
  children: ReactNode
  className?: string
  defaultOpen?: boolean
}

export function Disclosure({ summary, children, className, defaultOpen = false }: DisclosureProps) {
  return (
    <details className={cn('disclosure-panel', className)} open={defaultOpen || undefined}>
      <summary className="disclosure-summary">
        <span className="min-w-0 text-pretty">{summary}</span>
        <ChevronDown aria-hidden="true" className="h-4 w-4 shrink-0" />
      </summary>
      <div className="disclosure-content">{children}</div>
    </details>
  )
}
