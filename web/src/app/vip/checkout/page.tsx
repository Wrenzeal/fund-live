'use client'

import Link from 'next/link'
import { useState } from 'react'
import { CheckCircle2, Crown, LoaderCircle, ShieldAlert } from 'lucide-react'
import { AccountAreaShell } from '@/components/account-area-shell'
import { useCurrentUser } from '@/hooks/use-auth'
import { useVIPPreview } from '@/hooks/use-vip-preview'
import type { VIPBillingCycle } from '@/mocks/vip'
import { cn } from '@/lib/utils'

export default function VIPCheckoutPage() {
  const { user } = useCurrentUser()
  const { membership, plan, activateVIP, resetPreview } = useVIPPreview()
  const [selectedCycle, setSelectedCycle] = useState<VIPBillingCycle>('yearly')
  const [isActivating, setIsActivating] = useState(false)

  const selectedOption = plan.billingOptions.find((option) => option.cycle === selectedCycle) ?? plan.billingOptions[0]

  const handleActivate = async () => {
    if (!user || isActivating) {
      return
    }

    setIsActivating(true)
    try {
      await new Promise((resolve) => window.setTimeout(resolve, 900))
      activateVIP(selectedCycle)
    } finally {
      setIsActivating(false)
    }
  }

  return (
    <AccountAreaShell
      title="开通 VIP"
      description="当前版本先完成前端展示与交互闭环。页面会保留真实支付所需的结构，但按钮暂用于模拟开通流程。"
    >
      <div className="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]">
        <section className="vip-checkout-shell rounded-[36px] border p-6">
          <div className="mb-6">
            <div className="vip-proof-chip inline-flex items-center gap-2 rounded-full px-4 py-2 text-xs tracking-[0.26em]">
              <Crown className="h-3.5 w-3.5" />
              VIP CHECKOUT
            </div>
            <div className="mt-4 text-4xl font-black leading-tight text-theme-primary">把普通查看，升级成付费级研究入口</div>
            <p className="mt-3 max-w-2xl text-sm leading-6 text-theme-secondary">
              如果你已经开始关注自选分组、跟踪持仓节奏，就不该每天靠零散信息做判断。VIP 页面会把你真正关心的风险、机会和操作方向，整理成更容易消费的结构化结果。
            </p>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            {plan.billingOptions.map((option) => {
              const active = selectedCycle === option.cycle
              return (
                <button
                  key={option.cycle}
                  type="button"
                  onClick={() => setSelectedCycle(option.cycle)}
                  className={cn(
                    'vip-checkout-option rounded-[28px] border p-5 text-left transition-all duration-200',
                    active
                      ? 'vip-checkout-option-active'
                      : 'vip-checkout-option-idle'
                  )}
                >
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <div className="text-lg font-bold">{option.label}</div>
                      <div className="mt-2 text-sm text-theme-secondary">{option.dailyCostLabel}</div>
                    </div>
                    {option.badge && (
                      <span className="rounded-full border border-cyan-400/30 bg-cyan-500/10 px-3 py-1 text-[11px] tracking-[0.18em] text-cyan-300">
                        {option.badge}
                      </span>
                    )}
                  </div>
                  <div className="mt-6 text-3xl font-black text-theme-primary">{option.priceLabel}</div>
                  <div className="mt-3 text-xs tracking-[0.14em] text-theme-muted">
                    {option.cycle === 'yearly' ? '适合持续使用和形成日报习惯' : '适合先体验完整工作流'}
                  </div>
                </button>
              )
            })}
          </div>

          <div className="vip-urgency-banner mt-6 rounded-[28px] border p-5 text-sm leading-6 text-amber-50/92">
            当前是前端展示版，但页面结构已经按真实开通路径设计。正式支付接入后，这里会直接切成真实支付表单、支付状态和回调结果。
          </div>
        </section>

        <section className="space-y-6">
          <div className="vip-order-summary rounded-[32px] border p-6">
            <div className="text-xs tracking-[0.22em] text-theme-muted">订单摘要</div>
            <div className="mt-3 text-2xl font-bold text-theme-primary">{selectedOption.label}</div>
            <div className="mt-2 text-sm text-theme-secondary">{plan.subtitle}</div>

            <div className="mt-6 space-y-4">
              {plan.highlights.map((item) => (
                <div key={item} className="flex items-start gap-3">
                  <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0 text-cyan-300" />
                  <span className="text-sm leading-6 text-theme-secondary">{item}</span>
                </div>
              ))}
            </div>

            <div className="vip-total-box mt-6 rounded-[24px] border p-5">
              <div className="text-sm text-theme-secondary">应付金额</div>
              <div className="mt-2 text-4xl font-black text-cyan-50">{selectedOption.priceLabel}</div>
              <div className="mt-2 text-xs text-theme-muted">开通后即可解锁分析入口、任务中心和完整报告阅读权限</div>
            </div>

            {!user ? (
              <div className="mt-6 space-y-3">
                <div className="rounded-[24px] border border-[var(--card-border)] p-4 text-sm leading-6 text-theme-secondary">
                  会员状态与额度与账号绑定。请先登录，再进行开通操作。
                </div>
                <Link
                  href="/auth/login"
                  className="vip-primary-cta inline-flex items-center gap-2 rounded-2xl px-5 py-3 text-sm font-medium text-white"
                >
                  去登录后开通
                </Link>
              </div>
            ) : membership.isVip ? (
              <div className="mt-6 space-y-3">
                <div className="rounded-[24px] border border-emerald-500/20 bg-emerald-500/10 p-4 text-sm leading-6 text-emerald-50">
                  当前账号已处于 VIP 状态。你可以直接前往任务中心，或者重置演示状态重新体验开通过程。
                </div>
                <div className="flex flex-wrap gap-3">
                  <Link
                    href="/vip/tasks"
                    className="vip-primary-cta inline-flex items-center gap-2 rounded-2xl px-5 py-3 text-sm font-medium text-white"
                  >
                    查看任务中心
                  </Link>
                  <button
                    type="button"
                    onClick={resetPreview}
                    className="vip-secondary-cta rounded-2xl border px-5 py-3 text-sm font-medium"
                  >
                    重置演示状态
                  </button>
                </div>
              </div>
            ) : (
              <button
                type="button"
                onClick={() => void handleActivate()}
                disabled={isActivating}
                className="vip-primary-cta mt-6 inline-flex items-center gap-2 rounded-2xl px-5 py-3 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-80"
              >
                {isActivating ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Crown className="h-4 w-4" />}
                {isActivating ? '开通中...' : '模拟开通 VIP'}
              </button>
            )}
          </div>

          <div className="rounded-[32px] border border-rose-500/20 bg-rose-500/10 p-6">
            <div className="flex items-center gap-3 text-rose-100">
              <ShieldAlert className="h-5 w-5" />
              <div className="text-lg font-bold">风险与服务提示</div>
            </div>
            <p className="mt-4 text-sm leading-6 text-rose-50/90">{plan.disclaimer}</p>
          </div>
        </section>
      </div>
    </AccountAreaShell>
  )
}
