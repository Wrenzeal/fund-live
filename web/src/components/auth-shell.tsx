'use client'

import Link from 'next/link'
import { ArrowLeft, ChartNoAxesCombined, ShieldCheck, WalletCards } from 'lucide-react'
import { BrandMark } from '@/components/brand-mark'
import { HeaderFundSearch } from '@/components/header-fund-search'
import { ScrollReveal, ScrollRevealStack } from '@/components/scroll-reveal'
import { SiteFooter } from '@/components/site-footer'
import { ThemeSwitcher } from '@/components/theme-switcher'
import { useUIPreferences } from '@/hooks/use-ui-preferences'

interface AuthShellProps {
  title: string
  description: string
  eyebrow: string
  children: React.ReactNode
  footer: React.ReactNode
}

const highlights = [
  {
    title: '盘中估值',
    description: '实时估值与收盘净值分开查看。',
    icon: ChartNoAxesCombined,
  },
  {
    title: '自选清单',
    description: '按分组保存常看的基金。',
    icon: WalletCards,
  },
  {
    title: '用户持仓',
    description: '记录金额、份额和确认口径。',
    icon: ShieldCheck,
  },
]

export function AuthShell({
  title,
  description,
  eyebrow,
  children,
  footer,
}: AuthShellProps) {
  const { themeType, setThemeType, viewMode, setViewMode } = useUIPreferences()

  return (
    <div className="min-h-[100dvh]">
      <header className="sticky top-0 z-40 border-b border-[var(--card-border)] glass-strong">
        <div className="container mx-auto px-4 py-4">
          <div className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-4">
              <Link
                href="/"
                className="inline-flex items-center gap-2 rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2 text-sm text-theme-secondary transition-colors hover:text-theme-primary"
              >
                <ArrowLeft className="h-4 w-4" />
                返回首页
              </Link>

              <BrandMark subtitle="账户中心" href={null} />
            </div>

            <div className="hidden shrink-0 md:block">
              <HeaderFundSearch />
            </div>

            <ThemeSwitcher
              themeType={themeType}
              setThemeType={setThemeType}
              viewMode={viewMode}
              setViewMode={setViewMode}
              hideViewMode
            />
          </div>

          <div className="mt-4 flex justify-end md:hidden">
            <HeaderFundSearch />
          </div>
        </div>
      </header>

      <main id="main-content" className="container mx-auto px-4 py-10">
        <ScrollRevealStack className="grid gap-8 lg:grid-cols-[1.1fr_0.9fr]">
          <section className="overflow-hidden rounded-3xl border border-[var(--card-border)] glass-strong p-7 sm:p-9">
            <div className="space-y-7">
              <div className="text-sm font-semibold text-[var(--accent-primary)]">{eyebrow}</div>

              <div className="max-w-2xl space-y-4">
                <h1 className="text-3xl font-bold tracking-tight text-theme-primary sm:text-4xl">
                  统一查看估值、自选与持仓
                </h1>
                <p className="max-w-xl text-base leading-7 text-theme-secondary">
                  {description}
                </p>
              </div>

              <div className="grid gap-3 sm:grid-cols-3">
                {highlights.map(({ title: itemTitle, description: itemDescription, icon: Icon }) => (
                  <ScrollReveal key={itemTitle} delay={80} className="h-full">
                    <div className="h-full rounded-2xl border border-[var(--card-border)] bg-[var(--input-bg)]/60 p-4">
                      <div className="mb-3 inline-flex rounded-lg bg-[var(--accent-primary)]/12 p-2 text-[var(--accent-primary)]">
                        <Icon className="h-5 w-5" />
                      </div>
                      <div className="mb-2 text-base font-semibold text-theme-primary">{itemTitle}</div>
                      <p className="text-sm leading-6 text-theme-secondary">{itemDescription}</p>
                    </div>
                  </ScrollReveal>
                ))}
              </div>
            </div>
          </section>

          <section className="rounded-[32px] border border-[var(--card-border)] glass p-8 sm:p-10">
            <div className="mb-8 space-y-2">
              <h2 className="text-2xl font-bold text-theme-primary sm:text-3xl">{title}</h2>
              <p className="text-sm text-theme-muted">选择登录方式，继续使用 FundLive。</p>
            </div>

            {children}

            <div className="mt-8 border-t border-[var(--card-border)] pt-6 text-sm text-theme-secondary">
              {footer}
            </div>
          </section>
        </ScrollRevealStack>
      </main>

      <SiteFooter compact />
    </div>
  )
}
