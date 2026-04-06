'use client'

import Link from 'next/link'
import { BarChart3, BookOpenText, Crown, FileStack, ShieldAlert, Sparkles, Wallet } from 'lucide-react'
import { AccountAreaShell } from '@/components/account-area-shell'
import { useVIPPreview } from '@/hooks/use-vip-preview'
import { VIP_SAMPLE_REPORT_IDS } from '@/mocks/vip'

const valueCards = [
  {
    title: '板块分析',
    description: '基于自选分组识别主导板块，从宏观、政策、财报和市场走势四个维度生成分析。',
    icon: BarChart3,
  },
  {
    title: '组合分析',
    description: '面向全部持仓组合输出结构化报告、风险等级和操作建议，帮助你避免只盯单一基金。',
    icon: Wallet,
  },
  {
    title: '真实引用',
    description: '报告保留来源列表与引用摘要，为后续真实版接入资讯源和审计链路预留展示结构。',
    icon: BookOpenText,
  },
]

const workflowSteps = [
  '选择自选分组或持仓组合作为分析对象',
  '提交异步分析任务，系统开始整理宏观、政策、财报与市场信息',
  '在任务中心查看进度，完成后进入报告详情页阅读结论与建议',
]

export default function VIPPage() {
  const { membership, plan, remainingQuota, latestCompletedTask } = useVIPPreview()

  return (
    <AccountAreaShell
      title="VIP 智能投研"
      description="围绕你的自选分组与持仓组合生成结构化研究报告，帮助你从板块、政策、财报和市场走势四个维度理解当下环境。"
    >
      <div className="space-y-8">
        <section className="vip-hero-panel overflow-hidden rounded-[40px] border p-8 lg:p-10">
          <div className="vip-hero-orb vip-hero-orb-primary" />
          <div className="vip-hero-orb vip-hero-orb-secondary" />
          <div className="vip-hero-grid grid gap-8 lg:grid-cols-[1.2fr_0.8fr]">
            <div className="space-y-6">
              <div className="vip-proof-chip inline-flex items-center gap-2 rounded-full px-4 py-2 text-xs tracking-[0.28em]">
                <Crown className="h-3.5 w-3.5" />
                VIP INTELLIGENCE
              </div>

              <div className="space-y-4">
                <h2 className="max-w-3xl text-4xl font-black leading-[1.04] text-theme-primary sm:text-6xl">
                  把“看行情”
                  <span className="block vip-premium-gradient">升级成看结论</span>
                </h2>
                <p className="max-w-2xl text-base leading-7 text-theme-secondary">
                  FundLive VIP 会把自选分组和持仓组合转换成结构化研究报告。你不用再手动拼接新闻、政策、财报和板块走势，重点只看真正影响你当下决策的内容。
                </p>
              </div>

              <div className="grid gap-3 sm:grid-cols-3">
                {[
                  { label: '每日板块分析', value: '2 次' },
                  { label: '每日组合分析', value: '2 次' },
                  { label: '报告核心价值', value: '看结论' },
                ].map((item) => (
                  <div key={item.label} className="vip-stat-card rounded-[24px] border px-4 py-4">
                    <div className="text-xs tracking-[0.18em] text-theme-muted">{item.label}</div>
                    <div className="mt-2 text-2xl font-black text-theme-primary">{item.value}</div>
                  </div>
                ))}
              </div>

              <div className="flex flex-wrap gap-3">
                <Link
                  href={membership.isVip ? '/vip/tasks' : '/vip/checkout'}
                  className="vip-primary-cta inline-flex items-center gap-2 rounded-2xl px-5 py-3 text-sm font-medium text-white"
                >
                  <Sparkles className="h-4 w-4" />
                  {membership.isVip ? '查看分析任务' : '立即开通 VIP'}
                </Link>
                <Link
                  href={`/vip/reports/${VIP_SAMPLE_REPORT_IDS.defaultPortfolio}`}
                  className="vip-secondary-cta inline-flex items-center gap-2 rounded-2xl border px-5 py-3 text-sm font-medium"
                >
                  <FileStack className="h-4 w-4" />
                  查看示例报告
                </Link>
              </div>
            </div>

            <div className="grid gap-4 self-start">
              <div className="vip-price-spotlight rounded-[30px] border p-6">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="text-xs tracking-[0.2em] text-amber-200/85">当前开放档位</div>
                    <div className="mt-3 text-3xl font-black text-white">{plan.name}</div>
                    <div className="mt-2 text-sm leading-6 text-white/72">{plan.subtitle}</div>
                  </div>
                  <div className="vip-limited-chip rounded-full px-3 py-1 text-[11px] tracking-[0.2em]">
                    PAYMENT PREVIEW
                  </div>
                </div>
                <div className="mt-5 grid gap-3">
                  {plan.billingOptions.map((option) => (
                    <div key={option.cycle} className="vip-price-option rounded-2xl border px-4 py-3">
                      <div className="flex items-center justify-between gap-3">
                        <div>
                          <div className="text-sm font-semibold text-white">{option.label}</div>
                          <div className="mt-1 text-xs text-white/60">{option.dailyCostLabel}</div>
                        </div>
                        <div className="text-right">
                          {option.badge && (
                            <div className="mb-1 text-[10px] tracking-[0.2em] text-amber-200">{option.badge}</div>
                          )}
                          <div className="text-sm font-semibold text-cyan-100">{option.priceLabel}</div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
                <div className="mt-5 rounded-2xl border border-white/10 bg-black/15 px-4 py-4 text-sm leading-6 text-white/78">
                  从“每天看很多信息”，变成“每天只看一份结果清晰、结构完整的报告”。
                </div>
              </div>

              <div className="vip-benefit-panel rounded-[28px] border p-6">
                <div className="text-xs tracking-[0.2em] text-amber-200/90">权益摘要</div>
                <div className="mt-3 space-y-3 text-sm text-amber-50">
                  <div>每交易日 2 次板块分析</div>
                  <div>每交易日 2 次组合分析</div>
                  <div>异步生成完整报告</div>
                  <div>带风险提示和引用来源</div>
                </div>
                {membership.isVip && (
                  <div className="mt-5 rounded-2xl border border-amber-300/18 bg-black/10 px-4 py-3 text-xs leading-6 text-amber-100/90">
                    当前剩余：板块分析 {remainingQuota.sectorAnalysis} 次，组合分析 {remainingQuota.portfolioAnalysis} 次
                  </div>
                )}
              </div>
            </div>
          </div>
        </section>

        <section className="grid gap-5 lg:grid-cols-3">
          {valueCards.map((card) => {
            const Icon = card.icon
            return (
              <article key={card.title} className="vip-value-card rounded-[28px] border p-6">
                <div className="inline-flex rounded-2xl bg-cyan-500/12 p-3 text-cyan-200">
                  <Icon className="h-5 w-5" />
                </div>
                <div className="mt-5 text-xl font-bold text-theme-primary">{card.title}</div>
                <p className="mt-3 text-sm leading-6 text-theme-secondary">{card.description}</p>
              </article>
            )
          })}
        </section>

        <section className="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]">
          <div className="vip-preview-shell rounded-[32px] border p-6">
            <div className="mb-5">
              <div className="text-xs tracking-[0.22em] text-theme-muted">报告示例</div>
              <div className="mt-2 text-2xl font-bold text-theme-primary">你将看到怎样的报告</div>
            </div>

            <div className="space-y-4">
              <div className="vip-preview-highlight rounded-[24px] border p-5">
                <div className="text-sm font-semibold text-cyan-50">摘要结论</div>
                <div className="mt-3 text-sm leading-6 text-cyan-50/90">
                  组合当前处于“成长驱动 + 医药修复”并存阶段，适合维持平衡偏积极而非极端押注。
                </div>
              </div>

              <div className="grid gap-4 md:grid-cols-2">
                <div className="vip-value-card rounded-[24px] border p-5">
                  <div className="text-sm font-semibold text-theme-primary">操作建议</div>
                  <div className="mt-3 text-3xl font-black text-up">低吸</div>
                  <div className="mt-2 text-sm text-theme-secondary">建议仓位区间：5%-10%</div>
                </div>
                <div className="vip-value-card rounded-[24px] border p-5">
                  <div className="text-sm font-semibold text-theme-primary">风险等级</div>
                  <div className="mt-3 text-3xl font-black text-amber-300">中等</div>
                  <div className="mt-2 text-sm text-theme-secondary">接受波动，但不适合追高</div>
                </div>
              </div>

              <div className="vip-value-card rounded-[24px] border p-5">
                <div className="text-sm font-semibold text-theme-primary">引用来源</div>
                <div className="mt-3 space-y-3 text-sm text-theme-secondary">
                  <div>市场日报：成交额、风格强弱、板块活跃度</div>
                  <div>政策汇编：产业政策与监管口径变化</div>
                  <div>公司公告整理：核心持仓公司财报与基本面摘要</div>
                </div>
              </div>
            </div>
          </div>

          <div className="space-y-6">
            <section className="vip-value-card rounded-[32px] border p-6">
              <div className="mb-5 text-2xl font-bold text-theme-primary">使用流程</div>
              <div className="space-y-4">
                {workflowSteps.map((step, index) => (
                  <div key={step} className="flex items-start gap-4">
                    <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-2xl bg-cyan-500/15 text-sm font-semibold text-cyan-300">
                      {index + 1}
                    </div>
                    <div className="pt-1 text-sm leading-6 text-theme-secondary">{step}</div>
                  </div>
                ))}
              </div>
            </section>

            <section className="rounded-[32px] border border-rose-500/20 bg-rose-500/10 p-6">
              <div className="flex items-center gap-3 text-rose-100">
                <ShieldAlert className="h-5 w-5" />
                <div className="text-lg font-bold">风险与免责声明</div>
              </div>
              <p className="mt-4 text-sm leading-6 text-rose-50/90">{plan.disclaimer}</p>
            </section>

            {membership.isVip && latestCompletedTask && latestCompletedTask.reportId && (
              <section className="vip-value-card rounded-[32px] border p-6">
                <div className="text-xs tracking-[0.2em] text-theme-muted">最近完成的报告</div>
                <div className="mt-2 text-xl font-bold text-theme-primary">{latestCompletedTask.targetName}</div>
                <div className="mt-2 text-sm text-theme-secondary">{latestCompletedTask.progressText}</div>
                <Link
                  href={`/vip/reports/${latestCompletedTask.reportId}`}
                  className="mt-5 inline-flex items-center gap-2 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-sm font-medium text-theme-primary transition-colors hover:border-cyan-400/35 hover:bg-cyan-500/10"
                >
                  <FileStack className="h-4 w-4" />
                  继续阅读报告
                </Link>
              </section>
            )}
          </div>
        </section>

        <section className="vip-value-card rounded-[32px] border p-6">
          <div className="text-xs tracking-[0.22em] text-theme-muted">适合谁</div>
          <div className="mt-3 grid gap-4 md:grid-cols-3">
            {[
              '有自选分组，想优先看主导板块节奏的用户',
              '有持仓组合，想获得结构化结论和操作提示的用户',
              '不想每天自己拼新闻、财报和市场数据的用户',
            ].map((item) => (
              <div key={item} className="rounded-[24px] border border-[var(--card-border)] bg-[var(--input-bg)]/70 px-4 py-4 text-sm leading-6 text-theme-secondary">
                {item}
              </div>
            ))}
          </div>
        </section>
      </div>
    </AccountAreaShell>
  )
}
