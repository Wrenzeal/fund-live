import Link from 'next/link'
import { ArrowLeft, Database, LockKeyhole, ShieldCheck } from 'lucide-react'

import { BrandMark } from '@/components/brand-mark'
import { SiteFooter } from '@/components/site-footer'

const sections = [
  {
    title: '我们保存哪些信息',
    icon: Database,
    body: '账户邮箱、显示名称、自选分组、持仓记录、偏好设置和必要的登录会话会用于提供同步、估值观察和个人工作台能力。',
  },
  {
    title: '我们如何使用信息',
    icon: ShieldCheck,
    body: '数据只用于基金搜索、持仓估算、量化观察、报告生成和账号安全校验；页面展示的数据不用于承诺收益或替代个人投资判断。',
  },
  {
    title: '账号与安全',
    icon: LockKeyhole,
    body: '登录会话使用 HttpOnly Cookie。请不要在备注、导入内容或反馈中填写银行卡号、身份证号、真实交易密码等敏感信息。',
  },
]

export default function PrivacyPage() {
  return (
    <div className="flex min-h-dvh flex-col">
      <header className="border-b border-[var(--card-border)] glass-strong">
        <div className="container mx-auto flex items-center justify-between gap-4 px-4 py-4">
          <BrandMark subtitle="隐私政策" />
          <Link href="/" className="inline-flex items-center gap-2 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-2.5 text-sm text-theme-primary">
            <ArrowLeft className="h-4 w-4" />
            返回首页
          </Link>
        </div>
      </header>

      <main id="main-content" className="container mx-auto flex-1 px-4 py-10">
        <section className="rounded-[36px] border border-[var(--card-border)] p-8 glass-strong md:p-10">
          <p className="text-sm font-medium tracking-[0.22em] text-theme-muted">PRIVACY</p>
          <h1 className="mt-4 text-4xl font-bold tracking-tight text-theme-primary">隐私政策</h1>
          <p className="mt-4 max-w-3xl text-base leading-7 text-theme-secondary">
            FundLive 是面向个人基金观察的工具。我们尽量只收集提供功能所必需的数据，并把敏感配置和登录状态与前端展示隔离。
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
