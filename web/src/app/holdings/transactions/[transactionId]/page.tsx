'use client'

import Link from 'next/link'
import { useParams } from 'next/navigation'
import { AlertTriangle, ArrowLeft, Ban, CheckCircle2, Clock3, FileText, History, LoaderCircle, RotateCcw, WalletCards } from 'lucide-react'
import { AccountAreaShell } from '@/components/account-area-shell'
import { ScrollRevealStack } from '@/components/scroll-reveal'
import { useCurrentUser } from '@/hooks/use-auth'
import {
  useHoldingTransactionDetail,
  type HoldingTransactionEntry,
  type HoldingTransactionRollbackField,
  type HoldingTransactionType,
} from '@/hooks/use-user-portfolio'
import { formatSummaryMoney } from '@/lib/holding-display'
import { resolveHoldingSourceLabel } from '@/lib/holding-sources'
import { cn } from '@/lib/utils'

function transactionTypeLabel(type: HoldingTransactionType) {
  switch (type) {
    case 'buy':
      return '买入/补仓'
    case 'sell':
      return '卖出/清仓'
    case 'correction':
      return '校正持仓'
    case 'delete':
      return '删除记录'
    case 'dividend':
      return '分红'
    case 'adjustment':
    default:
      return '份额调整'
  }
}

function transactionTone(type: HoldingTransactionType) {
  switch (type) {
    case 'buy':
      return 'border-emerald-400/28 bg-emerald-400/10 text-emerald-100'
    case 'sell':
      return 'border-amber-400/28 bg-amber-400/10 text-amber-100'
    case 'correction':
      return 'border-cyan-400/28 bg-cyan-400/10 text-cyan-100'
    case 'delete':
      return 'border-rose-400/28 bg-rose-500/10 text-rose-100'
    case 'dividend':
      return 'border-violet-400/28 bg-violet-400/10 text-violet-100'
    case 'adjustment':
    default:
      return 'border-sky-400/28 bg-sky-400/10 text-sky-100'
  }
}

function formatDateTime(value?: string) {
  if (!value) {
    return '--'
  }
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return value
  }
  return new Intl.DateTimeFormat('zh-CN', {
    timeZone: 'Asia/Shanghai',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(parsed)
}

function emptyText(value?: string) {
  const trimmed = (value || '').trim()
  return trimmed || '--'
}

function DetailMetric({ label, value }: { label: string; value?: string }) {
  return (
    <div className="rounded-[20px] border border-[var(--card-border)] bg-[var(--input-bg)]/58 p-4">
      <div className="text-xs text-theme-muted">{label}</div>
      <div className="mt-2 break-all text-sm font-semibold text-theme-primary">{emptyText(value)}</div>
    </div>
  )
}

function RollbackFieldCard({ field }: { field: HoldingTransactionRollbackField }) {
  return (
    <div className="rounded-[20px] border border-white/10 bg-[var(--input-bg)]/56 p-4">
      <div className="text-xs text-theme-muted">{field.label}</div>
      <div className="mt-3 grid grid-cols-2 gap-2 text-xs">
        <div>
          <div className="text-theme-muted">当前值</div>
          <div className="mt-1 break-all font-semibold text-theme-primary">{emptyText(field.current_value)}</div>
        </div>
        <div>
          <div className="text-theme-muted">建议回到</div>
          <div className="mt-1 break-all font-semibold text-cyan-100">{emptyText(field.rollback_value)}</div>
        </div>
      </div>
      {field.delta && (
        <div className="mt-3 rounded-2xl border border-amber-300/18 bg-amber-400/10 px-3 py-2 text-xs text-amber-100">
          {field.direction || '变化'}：{field.delta}
        </div>
      )}
    </div>
  )
}

function RelatedTransactionRow({ transaction }: { transaction: HoldingTransactionEntry }) {
  return (
    <Link
      href={`/holdings/transactions/${transaction.id}`}
      className="block rounded-[20px] border border-[var(--card-border)] bg-[var(--input-bg)]/54 p-4 transition hover:-translate-y-0.5 hover:border-cyan-300/30"
    >
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <span className={cn('rounded-full border px-2 py-0.5 text-[11px] font-semibold', transactionTone(transaction.type))}>
              {transactionTypeLabel(transaction.type)}
            </span>
            {transaction.voided && <span className="rounded-full border border-slate-400/25 bg-slate-400/10 px-2 py-0.5 text-[11px] text-slate-200">已作废</span>}
          </div>
          <div className="mt-2 truncate text-sm font-semibold text-theme-primary">{transaction.note || transaction.fund?.name || transaction.fund_id}</div>
          <div className="mt-1 text-xs text-theme-muted">{formatDateTime(transaction.created_at)}</div>
        </div>
        <div className="text-sm font-black text-theme-primary">{formatSummaryMoney(transaction.amount)}</div>
      </div>
    </Link>
  )
}

export default function HoldingTransactionDetailPage() {
  const params = useParams<{ transactionId: string }>()
  const transactionID = typeof params?.transactionId === 'string' ? params.transactionId : ''
  const { user, isLoading: isUserLoading } = useCurrentUser()
  const { detail, error, isLoading } = useHoldingTransactionDetail(user?.id ?? null, transactionID || null)

  if (isUserLoading || isLoading) {
    return (
      <AccountAreaShell title="流水详情" description="查看单条持仓流水的完整字段、影响链和回滚预览。">
        <div className="rounded-[36px] border border-[var(--card-border)] p-10 text-center glass">
          <LoaderCircle className="mx-auto h-8 w-8 animate-spin text-cyan-300" />
          <div className="mt-4 text-sm text-theme-secondary">正在读取流水详情...</div>
        </div>
      </AccountAreaShell>
    )
  }

  if (!user) {
    return (
      <AccountAreaShell title="流水详情" description="登录后查看你的持仓流水详情。">
        <div className="rounded-[32px] border border-[var(--card-border)] p-8 text-center glass">
          <div className="text-xl font-bold text-theme-primary">登录后可查看流水详情</div>
          <Link href="/auth/login" className="mt-5 inline-flex rounded-2xl bg-gradient-to-r from-cyan-500 to-blue-600 px-4 py-3 text-sm font-semibold text-white">
            去登录
          </Link>
        </div>
      </AccountAreaShell>
    )
  }

  if (!detail) {
    return (
      <AccountAreaShell title="流水详情" description="查看单条持仓流水的完整字段、影响链和回滚预览。">
        <div className="rounded-[32px] border border-rose-500/20 bg-rose-500/10 p-6 text-rose-100">
          <div className="flex items-start gap-3">
            <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0" />
            <div>
              <div className="text-lg font-bold">流水不存在或当前账号无权查看</div>
              <p className="mt-2 text-sm text-theme-secondary">{error instanceof Error ? error.message : '请返回持仓页重新选择流水。'}</p>
              <Link href="/holdings" className="mt-4 inline-flex items-center gap-2 text-sm font-semibold text-rose-50 underline-offset-4 hover:underline">
                <ArrowLeft className="h-4 w-4" /> 返回持仓页
              </Link>
            </div>
          </div>
        </div>
      </AccountAreaShell>
    )
  }

  const transaction = detail.transaction
  const rollbackPreview = detail.rollback_preview
  const sourceLabel = resolveHoldingSourceLabel(transaction.source_platform, transaction.source_label)
  const metadataEntries = Object.entries(transaction.metadata ?? {})

  return (
    <AccountAreaShell title="流水详情" description="单条流水完整字段、metadata、当前快照和后续影响链。">
      <ScrollRevealStack className="space-y-6">
        <Link href="/holdings" className="inline-flex items-center gap-2 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-2 text-sm text-theme-secondary transition hover:border-cyan-300/35 hover:text-theme-primary">
          <ArrowLeft className="h-4 w-4" /> 返回持仓页
        </Link>

        <section className="overflow-hidden rounded-[34px] border border-[var(--card-border)] bg-[var(--card-bg)]/86 p-6 glass">
          <div className="flex flex-col gap-5 lg:flex-row lg:items-start lg:justify-between">
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                <span className={cn('rounded-full border px-3 py-1 text-xs font-semibold', transactionTone(transaction.type))}>
                  {transactionTypeLabel(transaction.type)}
                </span>
                {transaction.voided && (
                  <span className="inline-flex items-center gap-1 rounded-full border border-slate-400/25 bg-slate-400/10 px-3 py-1 text-xs font-semibold text-slate-200">
                    <Ban className="h-3.5 w-3.5" /> 已作废
                  </span>
                )}
                {sourceLabel && <span className="rounded-full border border-cyan-300/20 bg-cyan-400/10 px-3 py-1 text-xs font-semibold text-cyan-100">来源：{sourceLabel}</span>}
              </div>
              <h1 className="mt-4 truncate text-3xl font-black text-theme-primary">{transaction.fund?.name || transaction.fund_id}</h1>
              <p className="mt-2 max-w-3xl text-sm leading-6 text-theme-secondary">
                {transaction.note || '这条流水没有备注。详情页只做追溯、对账和回滚影响预览，不会自动修改当前持仓快照。'}
              </p>
            </div>
            <div className="rounded-[26px] border border-cyan-300/18 bg-cyan-400/10 px-5 py-4 text-right">
              <div className="text-xs text-theme-muted">流水金额</div>
              <div className="mt-2 text-3xl font-black text-theme-primary">{formatSummaryMoney(transaction.amount)}</div>
              <div className="mt-1 text-xs text-cyan-100">{formatDateTime(transaction.created_at)}</div>
            </div>
          </div>

          <div className="mt-6 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
            <DetailMetric label="基金代码" value={transaction.fund_id} />
            <DetailMetric label="关联持仓 ID" value={transaction.holding_id} />
            <DetailMetric label="确认份额" value={transaction.shares} />
            <DetailMetric label="确认净值 / 日期" value={`${emptyText(transaction.confirmed_nav)} / ${emptyText(transaction.confirmed_nav_date || transaction.as_of_date)}`} />
          </div>
        </section>

        <section className="grid gap-6 xl:grid-cols-[0.9fr_1.1fr]">
          <div className="rounded-[30px] border border-[var(--card-border)] bg-[var(--card-bg)]/84 p-5 glass">
            <div className="inline-flex items-center gap-2 rounded-full border border-amber-400/25 bg-amber-400/10 px-3 py-1 text-[11px] font-medium tracking-[0.2em] text-amber-100">
              <Clock3 className="h-3.5 w-3.5" />
              影响链
            </div>
            <div className="mt-4 space-y-3">
              {(detail.impact_chain ?? []).map((item, index) => (
                <div key={`${index}-${item}`} className="flex gap-3 rounded-[20px] border border-[var(--card-border)] bg-[var(--input-bg)]/54 p-3 text-sm leading-6 text-theme-secondary">
                  <span className="mt-1 flex h-6 w-6 shrink-0 items-center justify-center rounded-full border border-amber-300/24 bg-amber-400/10 text-[11px] font-bold text-amber-100">{index + 1}</span>
                  <span>{item}</span>
                </div>
              ))}
            </div>
          </div>

          <div className="rounded-[30px] border border-[var(--card-border)] bg-[var(--card-bg)]/84 p-5 glass">
            <div className="inline-flex items-center gap-2 rounded-full border border-cyan-400/25 bg-cyan-400/10 px-3 py-1 text-[11px] font-medium tracking-[0.2em] text-cyan-100">
              <RotateCcw className="h-3.5 w-3.5" />
              回滚预览
            </div>
            <h2 className="mt-3 text-xl font-black text-theme-primary">{rollbackPreview?.title || '只读预览'}</h2>
            <p className="mt-2 text-sm leading-6 text-theme-secondary">{rollbackPreview?.summary || '当前暂无回滚预览。'}</p>
            {rollbackPreview?.suggested_action && <p className="mt-2 text-xs leading-5 text-cyan-100">{rollbackPreview.suggested_action}</p>}
            <div className="mt-4 grid gap-3 md:grid-cols-2">
              {(rollbackPreview?.affected_fields ?? []).map((field) => <RollbackFieldCard key={`${field.field}-${field.label}`} field={field} />)}
            </div>
            {(rollbackPreview?.warnings ?? []).length > 0 && (
              <div className="mt-4 space-y-2">
                {rollbackPreview?.warnings?.map((warning) => (
                  <div key={warning} className="rounded-2xl border border-amber-300/18 bg-amber-400/10 px-3 py-2 text-xs leading-5 text-amber-100">{warning}</div>
                ))}
              </div>
            )}
          </div>
        </section>

        <section className="grid gap-6 xl:grid-cols-2">
          <div className="rounded-[30px] border border-[var(--card-border)] bg-[var(--card-bg)]/84 p-5 glass">
            <div className="inline-flex items-center gap-2 rounded-full border border-sky-400/25 bg-sky-400/10 px-3 py-1 text-[11px] font-medium tracking-[0.2em] text-sky-100">
              <WalletCards className="h-3.5 w-3.5" />
              当前快照
            </div>
            {detail.current_holding ? (
              <div className="mt-4 grid gap-3 sm:grid-cols-2">
                <DetailMetric label="当前本金" value={detail.current_holding.amount} />
                <DetailMetric label="当前份额" value={detail.current_holding.shares} />
                <DetailMetric label="确认净值" value={detail.current_holding.confirmed_nav} />
                <DetailMetric label="确认净值日" value={detail.current_holding.confirmed_nav_date} />
              </div>
            ) : (
              <div className="mt-4 rounded-[22px] border border-dashed border-[var(--card-border)] px-4 py-8 text-center text-sm text-theme-muted">
                当前持仓快照已不存在，通常表示已清仓或删除。
              </div>
            )}
          </div>

          <div className="rounded-[30px] border border-[var(--card-border)] bg-[var(--card-bg)]/84 p-5 glass">
            <div className="inline-flex items-center gap-2 rounded-full border border-violet-400/25 bg-violet-400/10 px-3 py-1 text-[11px] font-medium tracking-[0.2em] text-violet-100">
              <FileText className="h-3.5 w-3.5" />
              Metadata
            </div>
            <div className="mt-4 space-y-2">
              {metadataEntries.length === 0 ? (
                <div className="rounded-[22px] border border-dashed border-[var(--card-border)] px-4 py-8 text-center text-sm text-theme-muted">暂无扩展字段。</div>
              ) : metadataEntries.map(([key, value]) => (
                <div key={key} className="grid gap-2 rounded-[18px] border border-[var(--card-border)] bg-[var(--input-bg)]/54 px-3 py-2 text-xs sm:grid-cols-[0.34fr_0.66fr]">
                  <div className="break-all text-theme-muted">{key}</div>
                  <div className="break-all font-semibold text-theme-primary">{value || '--'}</div>
                </div>
              ))}
            </div>
          </div>
        </section>

        <section className="grid gap-6 xl:grid-cols-2">
          <div className="rounded-[30px] border border-[var(--card-border)] bg-[var(--card-bg)]/84 p-5 glass">
            <div className="inline-flex items-center gap-2 rounded-full border border-cyan-400/25 bg-cyan-400/10 px-3 py-1 text-[11px] font-medium tracking-[0.2em] text-cyan-100">
              <History className="h-3.5 w-3.5" />
              后续流水
            </div>
            <div className="mt-4 space-y-3">
              {(detail.subsequent_transactions ?? []).length === 0 ? (
                <div className="rounded-[22px] border border-dashed border-[var(--card-border)] px-4 py-8 text-center text-sm text-theme-muted">这条流水之后暂无同基金流水。</div>
              ) : detail.subsequent_transactions?.map((item) => <RelatedTransactionRow key={item.id} transaction={item} />)}
            </div>
          </div>

          <div className="rounded-[30px] border border-[var(--card-border)] bg-[var(--card-bg)]/84 p-5 glass">
            <div className="inline-flex items-center gap-2 rounded-full border border-emerald-400/25 bg-emerald-400/10 px-3 py-1 text-[11px] font-medium tracking-[0.2em] text-emerald-100">
              <CheckCircle2 className="h-3.5 w-3.5" />
              同基金相关流水
            </div>
            <div className="mt-4 space-y-3">
              {(detail.related_transactions ?? []).length === 0 ? (
                <div className="rounded-[22px] border border-dashed border-[var(--card-border)] px-4 py-8 text-center text-sm text-theme-muted">暂无更多相关流水。</div>
              ) : detail.related_transactions?.map((item) => <RelatedTransactionRow key={item.id} transaction={item} />)}
            </div>
          </div>
        </section>
      </ScrollRevealStack>
    </AccountAreaShell>
  )
}
