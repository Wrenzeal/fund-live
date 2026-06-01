'use client'

import { type ReactNode } from 'react'

export {
  ScrollReveal,
  ScrollRevealStack,
  ScrollReveal as AnalysisReveal,
  useLazyReveal,
} from '@/components/scroll-reveal'

export function AnalysisSectionHeading({
  icon,
  title,
  description,
}: {
  icon: ReactNode
  title: string
  description?: string
}) {
  return (
    <div className="flex items-start gap-3">
      <div className="rounded-2xl bg-[var(--input-bg)] p-2.5">
        {icon}
      </div>
      <div className="min-w-0">
        <div className="text-sm font-semibold text-theme-primary">{title}</div>
        {description && <div className="mt-1 text-xs leading-5 text-theme-muted">{description}</div>}
      </div>
    </div>
  )
}
