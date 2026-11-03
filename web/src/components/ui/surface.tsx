import type { ElementType, HTMLAttributes, ReactNode } from 'react'

import { cn } from '@/lib/utils'

type SurfaceTone = 'default' | 'strong' | 'subtle' | 'dashed'
type SurfacePadding = 'none' | 'sm' | 'md' | 'lg'
type SurfaceRadius = 'md' | 'lg' | 'xl'

interface SurfaceProps extends HTMLAttributes<HTMLElement> {
  as?: ElementType
  children: ReactNode
  tone?: SurfaceTone
  padding?: SurfacePadding
  radius?: SurfaceRadius
}

const toneClass: Record<SurfaceTone, string> = {
  default: 'border border-[var(--card-border)] bg-[var(--card-bg)]',
  strong: 'border border-[var(--card-border)] bg-[var(--card-bg)]',
  subtle: 'border border-[var(--card-border)] bg-[var(--surface-subtle)]',
  dashed: 'border border-dashed border-[var(--card-border)] bg-[var(--card-bg)]',
}

const paddingClass: Record<SurfacePadding, string> = {
  none: '',
  sm: 'p-4',
  md: 'p-5 md:p-6',
  lg: 'p-8 md:p-10',
}

const radiusClass: Record<SurfaceRadius, string> = {
  md: 'rounded-lg',
  lg: 'rounded-xl',
  xl: 'rounded-xl',
}

export function Surface({
  as: Component = 'div',
  children,
  className,
  tone = 'default',
  padding = 'md',
  radius = 'lg',
  ...props
}: SurfaceProps) {
  return (
    <Component
      className={cn(toneClass[tone], paddingClass[padding], radiusClass[radius], className)}
      {...props}
    >
      {children}
    </Component>
  )
}
