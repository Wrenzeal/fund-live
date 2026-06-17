import Link from 'next/link'
import { ArrowLeft, FileText, Scale, TriangleAlert } from 'lucide-react'

import { BrandMark } from '@/components/brand-mark'
import { SiteFooter } from '@/components/site-footer'

const sections = [
  {
    title: '仅作数据观察',
    icon: TriangleAlert,
    body: '估值、量化分数、报告摘要和风险提示都只是基于公开数据与本地规则整理的观察结果，不构成投资建议。',
  },
  {
    title: '数据可能延迟或缺失',
    icon: FileText,
    body: '基金净值、持仓、公告、行情和海外数据可能存在延迟、缓存、缺失或来源不可用，最终结果应以官方披露为准。',
  },
  {
    title: '合理使用账号',
    icon: Scale,
    body: '请勿批量抓取、绕过限制、攻击接口、提交违法内容或把本服务生成的内容包装成确定性收益承诺。',
  },
]

export default function TermsPage() {
  return (
    <div className="flex min-h-dvh flex-col">
      <header className="border-b border-[var(--card-border)] glass-strong">
        <div className="container mx-auto flex items-center justify-between gap-4 px-4 py-4">
          <BrandMark subtitle="服务条款" />
          <Link href="/" className="inline-flex items-center gap-2 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-2.5 text-sm text-theme-primary">
            <ArrowLeft className="h-4 w-4" />
            返回首页
          </Link>
        </div>
      </header>

      <main id="main-content" className="container mx-auto flex-1 px-4 py-10">
        <section className="rounded-[36px] border border-[var(--card-border)] p-8 glass-strong md:p-10">
          <p className="text-sm font-medium tracking-[0.22em] text-theme-muted">TERMS</p>
          <h1 className="mt-4 text-4xl font-bold tracking-tight text-theme-primary">服务条款</h1>
          <p className="mt-4 max-w-3xl text-base leading-7 text-theme-secondary">
            使用 FundLive 即表示你理解本产品是数据整理与个人观察工具。你仍需要独立判断风险，并以官方披露和自身情况为准。
          </p>
        </section>

        <section className="mt-8 grid gap-5 md:grid-cols-3">
          {sections.map(({ title, body, icon: Icon }) => (
            <article key={title} className="rounded-[28px] border border-[var(--card-border)] p-6 glass">
              <div className="inline-flex rounded-2xl bg-cyan-500/12 p-3 text-cyan-200">
                <Icon className="h-5 w-5" />
              </div>
              <h2 className="mt-5 text-xl font-semibold text-theme-primary">{title}</h2>
              <p className="mt-3 text-sm leading-6 text-theme-secondary">{body}</p>
            </article>
          ))}
        </section>
      </main>

      <SiteFooter compact />
    </div>
  )
}
