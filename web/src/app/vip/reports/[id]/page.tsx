'use client'

import Link from 'next/link'
import { useParams } from 'next/navigation'
import { AlertTriangle, BookOpenText, CalendarDays, FileStack, LoaderCircle, ShieldAlert, TrendingUp } from 'lucide-react'
import { AccountAreaShell } from '@/components/account-area-shell'
import { useVIPReport } from '@/hooks/use-vip-preview'
import { VIP_SAMPLE_REPORT_IDS } from '@/mocks/vip'
import { cn } from '@/lib/utils'

function riskMeta(level: 'low' | 'medium' | 'high') {
  switch (level) {
    case 'low':
      return {
        label: '低风险',
        className: 'border-emerald-500/25 bg-emerald-500/10 text-emerald-200',
      }
    case 'medium':
      return {
        label: '中风险',
        className: 'border-amber-500/25 bg-amber-500/10 text-amber-200',
      }
    default:
      return {
        label: '高风险',
        className: 'border-rose-500/25 bg-rose-500/10 text-rose-200',
      }
  }
}

export default function VIPReportDetailPage() {
  const params = useParams<{ id: string }>()
  const reportID = typeof params?.id === 'string' ? params.id : ''
  const { report, isLoading, error } = useVIPReport(reportID)

  if (isLoading) {
    return (
      <AccountAreaShell
        title="VIP 报告详情"
        description="查看结构化研究报告，包括摘要结论、操作建议、风险提示以及引用来源。"
      >
        <div className="rounded-[36px] border border-[var(--card-border)] p-10 glass text-center">
          <LoaderCircle className="mx-auto h-8 w-8 animate-spin text-cyan-300" />
          <div className="mt-4 text-sm text-theme-secondary">正在读取报告...</div>
        </div>
      </AccountAreaShell>
    )
  }

  if (!report) {
    return (
      <AccountAreaShell
        title="VIP 报告详情"
        description="查看结构化研究报告，包括摘要结论、操作建议、风险提示以及引用来源。"
      >
        <section className="rounded-[32px] border border-rose-500/20 bg-rose-500/10 p-6">
          <div className="flex items-start gap-3 text-rose-100">
            <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0" />
            <div>
              <div className="text-lg font-bold">报告不存在或当前账号无权查看</div>
              <p className="mt-2 text-sm leading-6 text-rose-50/90">
                {error instanceof Error ? error.message : '请返回任务中心，或查看系统内置示例报告。'}
              </p>
              <div className="mt-5 flex flex-wrap gap-3">
                <Link
                  href="/vip/tasks"
                  className="inline-flex items-center gap-2 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-sm font-medium text-theme-primary"
                >
                  <FileStack className="h-4 w-4" />
                  返回任务中心
                </Link>
                <Link
                  href={`/vip/reports/${VIP_SAMPLE_REPORT_IDS.defaultSector}`}
                  className="inline-flex items-center gap-2 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-sm font-medium text-theme-primary"
                >
                  查看示例报告
                </Link>
              </div>
            </div>
          </div>
        </section>
      </AccountAreaShell>
    )
  }

  const risk = riskMeta(report.riskLevel)

  return (
    <AccountAreaShell
      title="VIP 报告详情"
      description="查看结构化研究报告，包括摘要结论、操作建议、风险提示以及引用来源。"
    >
      <div className="space-y-8">
        <section className="rounded-[36px] border border-[var(--card-border)] p-8 glass">
          <div className="flex flex-col gap-5 lg:flex-row lg:items-start lg:justify-between">
            <div className="space-y-4">
              <div className="flex flex-wrap items-center gap-3">
                <span className={cn('rounded-full border px-3 py-1 text-xs tracking-[0.2em]', risk.className)}>
                  {risk.label}
                </span>
                <span className="rounded-full border border-[var(--card-border)] px-3 py-1 text-xs tracking-[0.2em] text-theme-muted">
                  {report.type === 'sector_analysis' ? '板块分析' : '组合分析'}
                </span>
              </div>

              <div>
                <h2 className="text-4xl font-black leading-tight text-theme-primary">{report.title}</h2>
                <p className="mt-3 max-w-3xl text-base leading-7 text-theme-secondary">
                  {report.summary.headline}
                </p>
              </div>
            </div>

            <div className="rounded-[28px] border border-[var(--card-border)] bg-[var(--input-bg)]/70 p-5 text-sm text-theme-secondary">
              <div className="space-y-3">
                <div className="flex items-center gap-2">
                  <CalendarDays className="h-4 w-4 text-cyan-300" />
                  生成时间：{new Date(report.generatedAt).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' })}
                </div>
                <div>分析对象：{report.targetName}</div>
                <div>覆盖范围：{report.coverageWindow}</div>
              </div>
            </div>
          </div>
        </section>

        <section className="grid gap-6 xl:grid-cols-[1.15fr_0.85fr]">
          <div className="space-y-6">
            <article className="rounded-[32px] border border-[var(--card-border)] p-6 glass">
              <div className="mb-4 flex items-center gap-3">
                <div className="rounded-2xl bg-cyan-500/12 p-3 text-cyan-300">
                  <TrendingUp className="h-5 w-5" />
                </div>
                <div className="text-2xl font-bold text-theme-primary">摘要结论</div>
              </div>
              <div className="space-y-3">
                {report.summary.bullets.map((bullet) => (
                  <div key={bullet} className="rounded-[22px] border border-[var(--card-border)] bg-[var(--input-bg)]/60 px-4 py-4 text-sm leading-6 text-theme-secondary">
                    {bullet}
                  </div>
                ))}
              </div>
            </article>

            {[
              report.macro,
              report.policy,
              {
                title: report.earnings.title,
                content: '以下公司为本报告重点观察对象，用于展示财报与基本面分析模块的呈现方式。',
                bullets: report.earnings.companies.map((company) => `${company.name}：${company.note}`),
              },
              report.market,
            ].map((section) => (
              <article key={section.title} className="rounded-[32px] border border-[var(--card-border)] p-6 glass">
                <div className="text-2xl font-bold text-theme-primary">{section.title}</div>
                <p className="mt-3 text-sm leading-7 text-theme-secondary">{section.content}</p>
                <div className="mt-5 space-y-3">
                  {section.bullets.map((item) => (
                    <div key={item} className="rounded-[22px] border border-[var(--card-border)] bg-[var(--input-bg)]/60 px-4 py-4 text-sm leading-6 text-theme-secondary">
                      {item}
                    </div>
                  ))}
                </div>
              </article>
            ))}
          </div>

          <div className="space-y-6">
            <article className="rounded-[32px] border border-cyan-500/25 bg-cyan-500/10 p-6">
              <div className="text-xs tracking-[0.22em] text-cyan-300">操作建议</div>
              <div className="mt-3 text-3xl font-black text-cyan-50">{report.advice.action}</div>
              <div className="mt-2 text-sm text-cyan-50/90">建议仓位区间：{report.advice.positionRange}</div>
              <div className="mt-5 space-y-3">
                {report.advice.conditions.map((condition) => (
                  <div key={condition} className="rounded-[22px] border border-cyan-400/20 bg-black/10 px-4 py-4 text-sm leading-6 text-cyan-50/90">
                    {condition}
                  </div>
                ))}
              </div>
            </article>

            <article className="rounded-[32px] border border-rose-500/20 bg-rose-500/10 p-6">
              <div className="mb-4 flex items-center gap-3 text-rose-100">
                <AlertTriangle className="h-5 w-5" />
                <div className="text-xl font-bold">风险提示</div>
              </div>
              <div className="space-y-3">
                {report.risks.map((riskItem) => (
                  <div key={riskItem} className="rounded-[22px] border border-rose-400/20 bg-black/10 px-4 py-4 text-sm leading-6 text-rose-50/90">
                    {riskItem}
                  </div>
                ))}
              </div>
            </article>

            <article className="rounded-[32px] border border-[var(--card-border)] p-6 glass">
              <div className="mb-4 flex items-center gap-3">
                <div className="rounded-2xl bg-[var(--input-bg)] p-3 text-theme-primary">
                  <BookOpenText className="h-5 w-5" />
                </div>
                <div className="text-xl font-bold text-theme-primary">引用来源</div>
              </div>
              <div className="space-y-4">
                {report.sources.map((source) => (
                  <a
                    key={source.id}
                    href={source.url}
                    target="_blank"
                    rel="noreferrer"
                    className="block rounded-[22px] border border-[var(--card-border)] bg-[var(--input-bg)]/60 px-4 py-4 transition-colors hover:border-cyan-400/35"
                  >
                    <div className="flex items-center justify-between gap-3">
                      <div className="text-sm font-semibold text-theme-primary">{source.title}</div>
                      <span className="text-[11px] tracking-[0.16em] text-theme-muted">{source.type}</span>
                    </div>
                    <div className="mt-2 text-xs text-theme-muted">
                      {source.publisher} · {new Date(source.publishedAt).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' })}
                    </div>
                    <div className="mt-3 text-sm leading-6 text-theme-secondary">{source.snippet}</div>
                  </a>
                ))}
              </div>
            </article>
          </div>
        </section>

        <section className="rounded-[32px] border border-amber-500/20 bg-amber-500/10 p-6 text-sm leading-7 text-amber-50/90">
          <div className="mb-3 flex items-center gap-3">
            <ShieldAlert className="h-5 w-5" />
            <div className="text-lg font-bold">免责声明</div>
          </div>
          {report.footerDisclaimer}
        </section>

        <section className="flex flex-wrap gap-3">
          <Link
            href="/vip/tasks"
            className="inline-flex items-center gap-2 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-sm font-medium text-theme-primary"
          >
            <FileStack className="h-4 w-4" />
            返回任务中心
          </Link>
          <Link
            href={`/vip/reports/${VIP_SAMPLE_REPORT_IDS.defaultSector}`}
            className="inline-flex items-center gap-2 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-sm font-medium text-theme-primary"
          >
            查看另一份示例报告
          </Link>
        </section>
      </div>
    </AccountAreaShell>
  )
}
