import Link from 'next/link'
import { ArrowLeft, SearchX } from 'lucide-react'

import { BrandMark } from '@/components/brand-mark'
import { SiteFooter } from '@/components/site-footer'

export default function NotFound() {
  return (
    <div className="flex min-h-dvh flex-col">
      <header className="border-b border-[var(--card-border)] glass-strong">
        <div className="container mx-auto flex items-center justify-between gap-4 px-4 py-4">
          <BrandMark subtitle="页面未找到" />
          <Link
            href="/"
            className="inline-flex items-center gap-2 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-2.5 text-sm font-medium text-theme-primary transition-colors hover:border-cyan-400/35 hover:bg-cyan-500/10"
          >
            <ArrowLeft className="h-4 w-4" />
            回到首页
          </Link>
        </div>
      </header>

      <main id="main-content" className="container mx-auto flex flex-1 items-center px-4 py-16">
        <section className="mx-auto grid max-w-5xl gap-8 rounded-[36px] border border-[var(--card-border)] p-8 glass-strong md:grid-cols-[0.8fr_1.2fr] md:p-10">
          <div className="flex min-h-48 items-center justify-center rounded-[28px] border border-[var(--card-border)] bg-[var(--input-bg)]/70">
            <SearchX className="h-20 w-20 text-cyan-200/80" />
          </div>
          <div className="flex flex-col justify-center">
            <p className="text-sm font-medium tracking-[0.22em] text-theme-muted">404 / PAGE NOT FOUND</p>
            <h1 className="mt-4 text-4xl font-bold tracking-tight text-theme-primary sm:text-5xl">这个页面暂时不存在</h1>
            <p className="mt-4 max-w-2xl text-base leading-7 text-theme-secondary">
              链接可能已经移动，或对应基金、报告、公告还没有生成。你可以返回首页搜索基金，或查看最近的更新公告。
            </p>
            <div className="mt-8 flex flex-wrap gap-3">
              <Link href="/" className="rounded-2xl bg-cyan-500 px-5 py-3 text-sm font-semibold text-slate-950 transition-transform hover:-translate-y-0.5 active:scale-[0.98]">
                搜索基金
              </Link>
              <Link href="/announcements" className="rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-5 py-3 text-sm font-medium text-theme-primary transition-colors hover:border-cyan-400/35 hover:bg-cyan-500/10">
                查看更新公告
              </Link>
            </div>
          </div>
        </section>
      </main>

      <SiteFooter compact />
    </div>
  )
}
