import Link, { type LinkProps } from 'next/link'
import type { AnchorHTMLAttributes, ButtonHTMLAttributes, ReactNode } from 'react'

import { cn } from '@/lib/utils'

type ActionButtonVariant = 'primary' | 'secondary' | 'ghost' | 'subtle'
type ActionButtonSize = 'sm' | 'md'

const baseClass =
  'inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-2xl font-medium transition-all duration-200 active:scale-[0.98] disabled:pointer-events-none disabled:opacity-60'

const variantClass: Record<ActionButtonVariant, string> = {
  primary:
    'bg-gradient-to-r from-cyan-500 via-sky-500 to-blue-600 text-white shadow-[0_16px_30px_rgba(14,165,233,0.22)] hover:-translate-y-0.5 hover:shadow-[0_18px_36px_rgba(14,165,233,0.30)]',
  secondary:
    'border border-[var(--input-border)] bg-[var(--input-bg)] text-theme-primary hover:-translate-y-0.5 hover:border-cyan-400/35 hover:bg-cyan-500/10',
  ghost:
    'border border-transparent text-theme-secondary hover:bg-[var(--input-bg)] hover:text-theme-primary',
  subtle:
    'border border-[var(--input-border)] bg-[var(--input-bg)] text-theme-secondary hover:border-cyan-400/35 hover:text-theme-primary',
}

const sizeClass: Record<ActionButtonSize, string> = {
  sm: 'px-3 py-2 text-sm',
  md: 'px-4 py-2.5 text-sm',
}

interface ActionButtonSharedProps {
  variant?: ActionButtonVariant
  size?: ActionButtonSize
  className?: string
  children: ReactNode
}

type ActionButtonLinkProps = ActionButtonSharedProps &
  LinkProps &
  Omit<AnchorHTMLAttributes<HTMLAnchorElement>, keyof LinkProps | keyof ActionButtonSharedProps>

type ActionButtonButtonProps = ActionButtonSharedProps &
  ButtonHTMLAttributes<HTMLButtonElement> & {
    href?: undefined
  }

type ActionButtonProps = ActionButtonLinkProps | ActionButtonButtonProps

export function ActionButton({
  variant = 'secondary',
  size = 'md',
  className,
  children,
  ...props
}: ActionButtonProps) {
  const classes = cn(baseClass, variantClass[variant], sizeClass[size], className)

  if ('href' in props && props.href !== undefined) {
    return (
      <Link className={classes} {...props}>
        {children}
      </Link>
    )
  }

  return (
    <button className={classes} {...props}>
      {children}
    </button>
  )
}
