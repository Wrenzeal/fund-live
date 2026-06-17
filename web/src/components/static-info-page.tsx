import Link from 'next/link'
import { ArrowLeft, type LucideIcon } from 'lucide-react'

import { BrandMark } from '@/components/brand-mark'
import { SiteFooter } from '@/components/site-footer'
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
          <Link
            href={backHref}
            className="inline-flex items-center gap-2 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-2.5 text-sm font-medium text-theme-primary transition-all duration-200 hover:-translate-y-0.5 hover:border-cyan-400/35 hover:bg-cyan-500/10 active:scale-[0.985]"
          >
            <ArrowLeft className="h-4 w-4" />
            {backLabel}
          </Link>
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
    <section className={cn('rounded-[36px] border border-[var(--card-border)] p-8 glass-strong md:p-10', className)}>
      <p className="text-sm font-medium tracking-[0.22em] text-theme-muted">{eyebrow}</p>
      <h1 className="mt-4 text-4xl font-bold tracking-tight text-theme-primary text-balance sm:text-5xl">
        {title}
      </h1>
      <p className="mt-4 max-w-3xl text-base leading-7 text-theme-secondary text-pretty">
        {description}
      </p>
    </section>
  )
}

export function StaticInfoCardGrid({ cards, accentClassName }: StaticInfoCardGridProps) {
  return (
    <section className="mt-8 grid gap-5 md:grid-cols-3">
      {cards.map(({ title, body, icon: Icon }) => (
        <article key={title} className="rounded-[28px] border border-[var(--card-border)] p-6 glass">
          <div className={cn('inline-flex rounded-2xl bg-cyan-500/12 p-3 text-cyan-200', accentClassName)}>
            <Icon className="h-5 w-5" />
          </div>
          <h2 className="mt-5 text-xl font-semibold text-theme-primary">{title}</h2>
          <p className="mt-3 text-sm leading-6 text-theme-secondary text-pretty">{body}</p>
        </article>
      ))}
    </section>
  )
}
