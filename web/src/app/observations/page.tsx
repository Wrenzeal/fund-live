'use client'

import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { ArrowRight, FileText, Search } from 'lucide-react'
import { AppTopBar } from '@/components/app-top-bar'
import { FundSearch } from '@/components/fund-search'
import { SiteFooter } from '@/components/site-footer'
import { Surface } from '@/components/ui/surface'
import { useSearchPreferences } from '@/hooks/use-search-preferences'

export default function ObservationsPage() {
  const router = useRouter()
  const { recentFunds } = useSearchPreferences()
  const selectFund = (fundId: string) => router.push(`/observations/${encodeURIComponent(fundId)}`)

  return (
    <div className="min-h-[100dvh]">
      <AppTopBar onFundSelect={selectFund} />
      <main id="main-content" className="container mx-auto max-w-5xl space-y-8 px-4 py-7 sm:py-10">
        <header className="max-w-2xl">
          <p className="mb-3 flex items-center gap-2 text-sm font-medium text-[var(--accent-primary)]"><FileText className="h-4 w-4" />公告与披露资料</p>
          <h1 className="text-3xl font-bold text-theme-primary sm:text-4xl">基金观察</h1>
          <p className="mt-3 text-sm leading-7 text-theme-secondary sm:text-base">查看基金和已披露持仓的近期公告，核对报告期、持仓覆盖及数据来源。</p>
        </header>

        <Surface as="section" className="flex flex-wrap items-center justify-between gap-5">
          <div className="min-w-0">
            <h2 className="text-lg font-semibold text-theme-primary">选择要观察的基金</h2>
            <p className="mt-2 text-sm leading-6 text-theme-secondary">按代码或名称搜索，进入该基金的公告时间线。</p>
          </div>
          <FundSearch onSelect={selectFund} />
        </Surface>

        {recentFunds.length > 0 && (
          <section aria-labelledby="recent-observations-title">
            <h2 id="recent-observations-title" className="mb-3 text-base font-semibold text-theme-primary">最近查看</h2>
            <div className="divide-y divide-[var(--card-border)] border-y border-[var(--card-border)]">
              {recentFunds.map((fund) => (
                <Link key={fund.id} href={`/observations/${encodeURIComponent(fund.id)}`} className="flex min-h-16 items-center justify-between gap-4 py-4 text-theme-primary transition-colors hover:text-[var(--accent-primary)]">
                  <span className="min-w-0"><span className="block break-words text-sm font-medium">{fund.name}</span><span className="mt-1 block font-mono text-xs text-theme-muted">{fund.id}</span></span>
                  <ArrowRight className="h-4 w-4 shrink-0" />
                </Link>
              ))}
            </div>
          </section>
        )}

        <div className="flex items-start gap-3 text-sm leading-7 text-theme-secondary">
          <Search className="mt-1.5 h-4 w-4 shrink-0 text-theme-muted" />
          <p>估值与持仓贡献可在<Link href="/" className="mx-1 text-[var(--accent-primary)] underline underline-offset-4">首页</Link>查看；市场走势与公式工具在<Link href="/research" className="mx-1 text-[var(--accent-primary)] underline underline-offset-4">研究</Link>。</p>
        </div>
      </main>
      <SiteFooter compact />
    </div>
  )
}
