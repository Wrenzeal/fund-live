import Link from 'next/link'
import { SearchX } from 'lucide-react'

import { StaticInfoHero, StaticInfoShell } from '@/components/static-info-page'

export default function NotFound() {
  return (
    <StaticInfoShell subtitle="页面未找到" backLabel="回到首页">
      <section className="mx-auto grid max-w-5xl gap-8 rounded-[36px] border border-[var(--card-border)] p-8 glass-strong md:grid-cols-[0.8fr_1.2fr] md:p-10">
        <div className="flex min-h-48 items-center justify-center rounded-[28px] border border-[var(--card-border)] bg-[var(--input-bg)]/70">
          <SearchX className="h-20 w-20 text-cyan-200/80" />
        </div>
        <div className="flex flex-col justify-center">
          <StaticInfoHero
            eyebrow="404 / 页面未找到"
            title="这个页面暂时不存在"
            description="链接可能已经移动，或对应基金、报告、公告还没有生成。你可以返回首页搜索基金，或查看最近的更新公告。"
            className="border-0 bg-transparent p-0 shadow-none backdrop-blur-0"
          />
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
    </StaticInfoShell>
  )
}
