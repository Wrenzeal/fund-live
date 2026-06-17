'use client'

import Link from 'next/link'
import { ArrowLeft, ChartNoAxesCombined, ShieldCheck, Sparkles, WalletCards } from 'lucide-react'
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
    description: '查看实时估值、分时曲线和常用基金动态。',
    icon: ChartNoAxesCombined,
  },
  {
    title: '自选清单',
    description: '把常看的基金保存到账号，换设备也能继续查看。',
    icon: WalletCards,
  },
  {
    title: '用户持仓',
    description: '保存持仓记录和校正信息，方便长期跟踪。',
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
    <div className="min-h-screen">
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

            <div className="hidden max-w-md flex-1 md:block">
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

          <div className="mt-4 md:hidden">
            <HeaderFundSearch />
          </div>
        </div>
      </header>

      <main id="main-content" className="container mx-auto px-4 py-10">
        <ScrollRevealStack className="grid gap-8 lg:grid-cols-[1.1fr_0.9fr]">
          <section className="relative overflow-hidden rounded-[32px] border border-[var(--card-border)] glass-strong p-8 sm:p-10">
            <div className="absolute inset-x-0 top-0 h-48 bg-gradient-to-br from-cyan-500/20 via-sky-500/10 to-transparent" />
            <div className="relative space-y-8">
              <div className="inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-500/10 px-4 py-2 text-sm text-cyan-300">
                <Sparkles className="h-4 w-4" />
                {eyebrow}
              </div>

              <div className="max-w-2xl space-y-4">
                <h1 className="text-4xl font-black tracking-tight text-theme-primary sm:text-5xl">
                  让估值系统开始记住
                  <span className="gradient-text">你的偏好</span>
                </h1>
                <p className="max-w-xl text-base leading-7 text-theme-secondary sm:text-lg">
                  {description}
                </p>
              </div>

              <div className="grid gap-4 sm:grid-cols-3">
                {highlights.map(({ title: itemTitle, description: itemDescription, icon: Icon }) => (
                  <ScrollReveal key={itemTitle} delay={80} className="h-full">
                    <div className="h-full rounded-2xl border border-[var(--card-border)] bg-[var(--input-bg)]/70 p-5 backdrop-blur">
                      <div className="mb-3 inline-flex rounded-xl bg-cyan-500/15 p-2 text-cyan-300">
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
            <div className="mb-8 space-y-3">
              <p className="text-sm uppercase tracking-[0.3em] text-theme-muted">{title}</p>
              <h2 className="text-3xl font-bold text-theme-primary">{title}</h2>
              <p className="text-sm leading-6 text-theme-secondary">{description}</p>
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
