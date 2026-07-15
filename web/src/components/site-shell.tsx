'use client'

import { ArrowUp, Bug, type LucideIcon } from 'lucide-react'
import { AppTopBar } from '@/components/app-top-bar'
import { SiteFooter } from '@/components/site-footer'
import { useMobileTopSection } from '@/hooks/use-mobile-top-section'
import { cn } from '@/lib/utils'

interface SiteShellProps {
  title: string
  description: string
  eyebrowLabel?: string
  EyebrowIcon?: LucideIcon
  children: React.ReactNode
}

export function SiteShell({
  title,
  description,
  eyebrowLabel = '社区动态',
  EyebrowIcon = Bug,
  children,
}: SiteShellProps) {
  const { isAtTop, showBackToTop, scrollToTop } = useMobileTopSection()

  return (
    <div className="min-h-[100dvh]">
      <AppTopBar />

      <section
        className={cn(
          'container mx-auto overflow-hidden px-4 transition-all duration-300 md:overflow-visible md:transition-none',
          isAtTop
            ? 'pt-5 max-h-[28rem] opacity-100'
            : 'pt-0 max-h-0 opacity-0 pointer-events-none md:pointer-events-auto',
          'md:pt-5 md:max-h-none md:opacity-100'
        )}
      >
        <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
          <div className="max-w-3xl space-y-3">
            <div className="inline-flex items-center gap-2 text-sm font-semibold text-[var(--accent-primary)]">
              <EyebrowIcon className="h-3.5 w-3.5" />
              {eyebrowLabel}
            </div>
            <div>
              <h1 className="text-3xl font-black text-theme-primary sm:text-4xl">{title}</h1>
              <p className="mt-2 max-w-2xl text-sm leading-6 text-theme-secondary sm:text-base">
                {description}
              </p>
            </div>
          </div>

        </div>
      </section>

      <main id="main-content" className="container mx-auto px-4 py-8">
        {children}
      </main>

      <SiteFooter compact />

      {showBackToTop && (
        <button
          type="button"
          onClick={scrollToTop}
          className="fixed bottom-5 right-4 z-50 inline-flex items-center gap-2 rounded-full border border-cyan-400/30 bg-[var(--card-bg)]/95 px-4 py-3 text-sm font-medium text-theme-primary shadow-[0_18px_36px_rgba(2,8,23,0.28)] backdrop-blur md:hidden"
          aria-label="回到顶部"
        >
          <ArrowUp className="h-4 w-4" />
          顶部
        </button>
      )}
    </div>
  )
}
