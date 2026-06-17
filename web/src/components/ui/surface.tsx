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
  default: 'glass border border-[var(--card-border)]',
  strong: 'glass-strong border border-[var(--card-border)]',
  subtle: 'border border-[var(--card-border)] bg-[var(--card-bg)]/35',
  dashed: 'glass border border-dashed border-[var(--card-border)]',
}

const paddingClass: Record<SurfacePadding, string> = {
  none: '',
  sm: 'p-4',
  md: 'p-5 md:p-6',
  lg: 'p-8 md:p-10',
}

const radiusClass: Record<SurfaceRadius, string> = {
  md: 'rounded-2xl',
  lg: 'rounded-3xl',
  xl: 'rounded-[2rem]',
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
