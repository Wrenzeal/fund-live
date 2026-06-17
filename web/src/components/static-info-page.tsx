import { ArrowLeft, type LucideIcon } from 'lucide-react'

import { BrandMark } from '@/components/brand-mark'
import { SiteFooter } from '@/components/site-footer'
import { ActionButton } from '@/components/ui/action-button'
import { Surface } from '@/components/ui/surface'
import { cn } from '@/lib/utils'

interface StaticInfoShellProps {
  subtitle: string
  backHref?: string
  backLabel?: string
  children: React.ReactNode
}

interface StaticInfoHeroProps {
  eyebrow: string
  title: string
  description: string
  className?: string
}

interface StaticInfoCard {
  title: string
  body: string
  icon: LucideIcon
}

interface StaticInfoCardGridProps {
  cards: StaticInfoCard[]
  accentClassName?: string
}

export function StaticInfoShell({
  subtitle,
  backHref = '/',
  backLabel = '返回首页',
  children,
}: StaticInfoShellProps) {
  return (
    <div className="flex min-h-dvh flex-col">
      <header className="border-b border-[var(--card-border)] glass-strong">
        <div className="container mx-auto flex items-center justify-between gap-4 px-4 py-4">
          <BrandMark subtitle={subtitle} />
          <ActionButton href={backHref} variant="secondary">
            <ArrowLeft className="h-4 w-4" />
            {backLabel}
          </ActionButton>
        </div>
      </header>

      <main id="main-content" className="container mx-auto flex-1 px-4 py-10">
        {children}
      </main>

      <SiteFooter compact />
    </div>
  )
}

export function StaticInfoHero({ eyebrow, title, description, className }: StaticInfoHeroProps) {
  return (
    <Surface as="section" tone="strong" padding="lg" radius="xl" className={className}>
      <p className="text-sm font-medium text-theme-muted">{eyebrow}</p>
      <h1 className="mt-4 text-4xl font-bold tracking-tight text-theme-primary text-balance sm:text-5xl">
        {title}
      </h1>
      <p className="mt-4 max-w-3xl text-base leading-7 text-theme-secondary text-pretty">
        {description}
      </p>
    </Surface>
  )
}

export function StaticInfoCardGrid({ cards, accentClassName }: StaticInfoCardGridProps) {
  return (
    <section className="mt-8 grid gap-5 md:grid-cols-3">
      {cards.map(({ title, body, icon: Icon }) => (
        <Surface as="article" key={title} padding="md" radius="lg">
          <div className={cn('inline-flex rounded-2xl bg-cyan-500/12 p-3 text-cyan-200', accentClassName)}>
            <Icon className="h-5 w-5" />
          </div>
          <h2 className="mt-5 text-xl font-semibold text-theme-primary">{title}</h2>
          <p className="mt-3 text-sm leading-6 text-theme-secondary text-pretty">{body}</p>
        </Surface>
      ))}
    </section>
  )
}
