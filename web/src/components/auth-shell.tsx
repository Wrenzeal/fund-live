'use client'

import Link from 'next/link'
import { ArrowLeft, ChartNoAxesCombined, ShieldCheck, WalletCards } from 'lucide-react'
import { BrandMark } from '@/components/brand-mark'
import { HeaderFundSearch } from '@/components/header-fund-search'
import { Disclosure } from '@/components/ui/disclosure'
import { ScrollReveal } from '@/components/scroll-reveal'
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
        <div className="container mx-auto px-3 py-2 sm:px-4 md:py-4">
          <div className="flex min-h-11 items-center gap-2 md:justify-between md:gap-4">
            <div className="flex items-center gap-2 md:gap-4">
              <Link
                href="/"
                className="inline-flex h-11 w-11 items-center justify-center gap-2 rounded-xl border border-[var(--input-border)] bg-[var(--input-bg)] text-sm text-theme-secondary transition-colors hover:text-theme-primary md:w-auto md:px-3"
                aria-label="返回首页"
              >
                <ArrowLeft className="h-4 w-4" />
                <span className="hidden md:inline">返回首页</span>
              </Link>

              <BrandMark compact subtitle="账户中心" href={null} className="md:hidden" />
              <BrandMark subtitle="账户中心" href={null} className="hidden md:flex" />
            </div>

            <div className="hidden shrink-0 md:block">
              <HeaderFundSearch />
            </div>

            <div className="ml-auto flex items-center gap-2 md:ml-0">
              <div className="md:hidden">
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
          </div>
        </div>
      </header>

      <main id="main-content" className="container mx-auto px-4 py-6 sm:py-10">
        <div className="grid gap-8 lg:grid-cols-[1.1fr_0.9fr]">
          <ScrollReveal className="order-2 hidden lg:order-1 lg:block">
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
          </ScrollReveal>

          <ScrollReveal className="order-1 lg:order-2" delay={80}>
            <section className="rounded-3xl border border-[var(--card-border)] glass p-5 sm:rounded-[32px] sm:p-10">
              <div className="mb-6 space-y-2 sm:mb-8">
                <h2 className="text-2xl font-bold text-theme-primary sm:text-3xl">{title}</h2>
                <p className="text-sm text-theme-muted">选择登录方式，继续使用 FundLive。</p>
              </div>

              {children}

              <div className="mt-8 border-t border-[var(--card-border)] pt-6 text-sm text-theme-secondary">
                {footer}
              </div>
            </section>
          </ScrollReveal>

          <ScrollReveal className="order-3 lg:hidden" delay={120}>
            <Disclosure summary="登录后可使用">
              <p className="mb-4 text-sm leading-6 text-theme-secondary">
                统一查看估值、自选与持仓，账户数据会跟随登录状态保存。
              </p>
              <div className="space-y-3">
                {highlights.map(({ title: itemTitle, description: itemDescription, icon: Icon }) => (
                  <div key={itemTitle} className="flex items-start gap-3 rounded-xl bg-[var(--input-bg)]/60 p-3">
                    <div className="rounded-lg bg-[var(--accent-primary)]/12 p-2 text-[var(--accent-primary)]">
                      <Icon className="h-4 w-4" />
                    </div>
                    <div>
                      <div className="text-sm font-semibold text-theme-primary">{itemTitle}</div>
                      <p className="mt-1 text-xs leading-5 text-theme-secondary">{itemDescription}</p>
                    </div>
                  </div>
                ))}
              </div>
            </Disclosure>
          </ScrollReveal>
        </div>
      </main>

      <SiteFooter compact />
    </div>
  )
}
