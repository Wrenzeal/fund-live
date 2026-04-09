'use client'

import Link from 'next/link'
import { useEffect, useState } from 'react'
import { CheckCircle2, Copy, Crown, ExternalLink, LoaderCircle, ShieldAlert } from 'lucide-react'
import { AccountAreaShell } from '@/components/account-area-shell'
import { useCurrentUser } from '@/hooks/use-auth'
import { VIPRequestError, createVIPOrder, useVIPOrder, useVIPPreview } from '@/hooks/use-vip-preview'
import type { VIPBillingCycle } from '@/mocks/vip'
import { cn } from '@/lib/utils'

export default function VIPCheckoutPage() {
  const { user } = useCurrentUser()
  const { membership, plan, activateVIP, resetPreview, refreshMembership, refreshQuota } = useVIPPreview()
  const [selectedCycle, setSelectedCycle] = useState<VIPBillingCycle>('yearly')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [activeOrderID, setActiveOrderID] = useState<string | null>(null)
  const [paymentMessage, setPaymentMessage] = useState<string | null>(null)
  const [isCopied, setIsCopied] = useState(false)

  const selectedOption = plan.billingOptions.find((option) => option.cycle === selectedCycle) ?? plan.billingOptions[0]
  const { order, isLoading: isOrderLoading } = useVIPOrder(activeOrderID)

  useEffect(() => {
    if (order?.status === 'paid') {
      void Promise.all([refreshMembership(), refreshQuota()])
    }
  }, [order?.status, refreshMembership, refreshQuota])

  const handleCreateOrder = async () => {
    if (!user || isSubmitting) {
      return
    }

    setPaymentMessage(null)
    setIsSubmitting(true)
    try {
      const created = await createVIPOrder(selectedCycle)
      if (created) {
        setActiveOrderID(created.id)
        setPaymentMessage('订单已创建。请使用微信扫码或打开支付链接完成支付，页面会自动刷新支付状态。')
      }
    } catch (error) {
      if (error instanceof VIPRequestError && error.code === 'PAYMENT_NOT_CONFIGURED') {
        setPaymentMessage('微信支付尚未完成配置。你可以先补充 YAML 中的支付参数，或临时使用预览开通进行联调。')
      } else {
        setPaymentMessage(error instanceof Error ? error.message : '创建支付订单失败，请稍后重试。')
      }
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleActivatePreview = async () => {
    if (!user || isSubmitting) {
      return
    }

    setIsSubmitting(true)
    setPaymentMessage(null)
    try {
      await activateVIP(selectedCycle)
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleCopyCodeURL = async () => {
    if (!order?.codeURL) {
      return
    }

    await navigator.clipboard.writeText(order.codeURL)
    setIsCopied(true)
    window.setTimeout(() => setIsCopied(false), 1500)
  }

  const currentStatusLabel = (() => {
    switch (order?.status) {
      case 'paid':
        return '已支付'
      case 'closed':
        return '已关闭'
      case 'failed':
        return '下单失败'
      case 'pending_payment':
        return '待支付'
      default:
        return '未创建'
    }
  })()

  return (
    <AccountAreaShell
      title="开通 VIP"
      description="当前版本已支持真实 VIP 订单、微信支付 Native 下单与支付状态查询。若微信配置尚未补齐，页面会明确提示缺失项。"
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

          <div className="vip-urgency-banner mt-6 rounded-[28px] border p-5 text-sm leading-6 text-theme-secondary">
            当前页面已接入微信支付 Native 下单与订单状态查询。若你还没有补齐商户配置，页面会直接提示“支付未配置”，不会再静默回退。
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
              <div className="mt-2 text-4xl font-black text-theme-primary">{selectedOption.priceLabel}</div>
              <div className="mt-2 text-xs text-theme-muted">开通后即可解锁分析入口、任务中心和完整报告阅读权限</div>
            </div>

            {paymentMessage && (
              <div className="mt-6 rounded-[24px] border border-amber-500/20 bg-amber-500/10 p-4 text-sm leading-6 text-amber-50/90">
                {paymentMessage}
              </div>
            )}

            {order && (
              <div className="mt-6 rounded-[24px] border border-[var(--card-border)] bg-[var(--input-bg)]/60 p-5">
                <div className="text-xs tracking-[0.18em] text-theme-muted">当前订单</div>
                <div className="mt-3 space-y-3 text-sm text-theme-secondary">
                  <div>订单号：{order.orderNo}</div>
                  <div>订单状态：{currentStatusLabel}</div>
                  <div>支付方式：微信支付 Native</div>
                  {order.expiresAt && <div>过期时间：{new Date(order.expiresAt).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' })}</div>}
                  {order.paidAt && <div>支付时间：{new Date(order.paidAt).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' })}</div>}
                  {order.wechatTransactionID && <div>微信交易单号：{order.wechatTransactionID}</div>}
                  {order.errorMessage && <div className="text-rose-200">错误信息：{order.errorMessage}</div>}
                </div>

                {order.codeURL && order.status === 'pending_payment' && (
                  <div className="mt-5 space-y-3">
                    <div className="rounded-[20px] border border-[var(--card-border)] bg-[var(--input-bg)]/60 p-4 text-sm leading-6 text-theme-secondary break-all">
                      请将下方 `code_url` 生成二维码后使用微信扫码，或尝试直接打开支付链接。
                      <div className="mt-3 rounded-xl border border-[var(--card-border)] bg-[var(--input-bg)]/70 p-3 font-mono text-xs text-theme-primary">
                        {order.codeURL}
                      </div>
                    </div>
                    <div className="flex flex-wrap gap-3">
                      <button
                        type="button"
                        onClick={() => void handleCopyCodeURL()}
                        className="vip-secondary-cta inline-flex items-center gap-2 rounded-2xl border px-4 py-3 text-sm font-medium"
                      >
                        <Copy className="h-4 w-4" />
                        {isCopied ? '已复制' : '复制 code_url'}
                      </button>
                      <a
                        href={order.codeURL}
                        className="vip-primary-cta inline-flex items-center gap-2 rounded-2xl px-4 py-3 text-sm font-medium text-white"
                      >
                        <ExternalLink className="h-4 w-4" />
                        尝试打开支付链接
                      </a>
                    </div>
                  </div>
                )}
              </div>
            )}

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
                    onClick={() => void resetPreview()}
                    className="vip-secondary-cta rounded-2xl border px-5 py-3 text-sm font-medium"
                  >
                    重置演示状态
                  </button>
                </div>
              </div>
            ) : (
              <div className="mt-6 flex flex-wrap gap-3">
                <button
                  type="button"
                  onClick={() => void handleCreateOrder()}
                  disabled={isSubmitting || isOrderLoading}
                  className="vip-primary-cta inline-flex items-center gap-2 rounded-2xl px-5 py-3 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-80"
                >
                  {isSubmitting || isOrderLoading ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Crown className="h-4 w-4" />}
                  {isSubmitting ? '创建订单中...' : '微信支付下单'}
                </button>

                <button
                  type="button"
                  onClick={() => void handleActivatePreview()}
                  disabled={isSubmitting}
                  className="vip-secondary-cta rounded-2xl border px-5 py-3 text-sm font-medium"
                >
                  开发环境预览开通
                </button>
              </div>
            )}
          </div>

          <div className="rounded-[32px] border border-rose-500/20 bg-rose-500/10 p-6">
          <div className="flex items-center gap-3 text-theme-primary">
            <ShieldAlert className="h-5 w-5" />
            <div className="text-lg font-bold">风险与服务提示</div>
          </div>
          <p className="mt-4 text-sm leading-6 text-theme-secondary">{plan.disclaimer}</p>
        </div>
      </section>
      </div>
    </AccountAreaShell>
  )
}
