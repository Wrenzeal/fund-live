import Link, { type LinkProps } from 'next/link'
import type { AnchorHTMLAttributes, ButtonHTMLAttributes, ReactNode } from 'react'

import { cn } from '@/lib/utils'

type ActionButtonVariant = 'primary' | 'secondary' | 'ghost' | 'subtle'
type ActionButtonSize = 'sm' | 'md'

const baseClass =
  'inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-2xl font-medium transition-colors duration-160 active:opacity-80 disabled:pointer-events-none disabled:opacity-60'

const variantClass: Record<ActionButtonVariant, string> = {
  primary:
    'bg-[var(--accent-primary)] text-white shadow-[0_10px_24px_color-mix(in_srgb,var(--accent-primary)_24%,transparent)] hover:brightness-105',
  secondary:
    'border border-[var(--input-border)] bg-[var(--input-bg)] text-theme-primary hover:border-[var(--accent-primary)] hover:bg-[var(--accent-primary)]/10',
  ghost:
    'border border-transparent text-theme-secondary hover:bg-[var(--input-bg)] hover:text-theme-primary',
  subtle:
    'border border-[var(--input-border)] bg-[var(--input-bg)] text-theme-secondary hover:border-[var(--accent-primary)] hover:text-theme-primary',
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
