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
      <div className="relative">
        <div className={cn(
          'flex items-center justify-center bg-gradient-to-br from-cyan-500 via-sky-500 to-blue-600 text-white shadow-lg shadow-cyan-500/20',
          compact ? 'h-9 w-9 rounded-xl' : 'h-11 w-11 rounded-2xl'
        )}>
          <Activity className={compact ? 'h-5 w-5' : 'h-6 w-6'} />
        </div>
        <div className={cn(
          'absolute -inset-1 bg-gradient-to-br from-cyan-500 to-blue-600 opacity-30 blur',
          compact ? 'rounded-xl' : 'rounded-2xl'
        )} />
      </div>
      <div>
        <div className={cn('font-bold gradient-text', compact ? 'text-base' : 'text-lg')}>涨了多少</div>
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
