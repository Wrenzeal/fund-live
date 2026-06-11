import Link from 'next/link'
import { Activity } from 'lucide-react'

import { cn } from '@/lib/utils'

interface BrandMarkProps {
  subtitle: string
  href?: string | null
  className?: string
}

export function BrandMark({ subtitle, href = '/', className }: BrandMarkProps) {
  const content = (
    <>
      <div className="relative">
        <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-gradient-to-br from-cyan-500 via-sky-500 to-blue-600 text-white shadow-lg shadow-cyan-500/20">
          <Activity className="h-6 w-6" />
        </div>
        <div className="absolute -inset-1 rounded-2xl bg-gradient-to-br from-cyan-500 to-blue-600 opacity-30 blur" />
      </div>
      <div>
        <div className="text-lg font-bold gradient-text">涨了多少</div>
        <div className="text-xs text-theme-muted">{subtitle}</div>
      </div>
    </>
  )

  if (href === null) {
    return <div className={cn('flex items-center gap-3', className)}>{content}</div>
  }

  return (
    <Link href={href} className={cn('flex items-center gap-3', className)}>
      {content}
    </Link>
  )
}
