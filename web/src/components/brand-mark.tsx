import Link from 'next/link'
import { Activity } from 'lucide-react'

import { cn } from '@/lib/utils'

interface BrandMarkProps {
  subtitle: string
  href?: string | null
  className?: string
  compact?: boolean
}

export function BrandMark({ subtitle, href = '/', className, compact = false }: BrandMarkProps) {
  const content = (
    <>
      <div>
        <div className={cn(
          'flex items-center justify-center border border-[var(--card-border)] bg-[var(--accent-soft)] text-[var(--accent-primary)]',
          compact ? 'h-9 w-9 rounded-lg' : 'h-11 w-11 rounded-lg'
        )}>
          <Activity className={compact ? 'h-5 w-5' : 'h-6 w-6'} />
        </div>
      </div>
      <div>
        <div className={cn('font-semibold tracking-tight text-theme-primary', compact ? 'text-base' : 'text-lg')}>涨了多少</div>
        {!compact && <div className="text-xs text-theme-muted">{subtitle}</div>}
      </div>
    </>
  )

  if (href === null) {
    return <div className={cn('flex items-center', compact ? 'gap-2' : 'gap-3', className)}>{content}</div>
  }

  return (
    <Link href={href} className={cn('flex min-h-11 items-center', compact ? 'gap-2' : 'gap-3', className)}>
      {content}
    </Link>
  )
}
