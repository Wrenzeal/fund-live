'use client'

import { useId, useRef, useState } from 'react'
import { Trash2 } from 'lucide-react'
import { ActionButton } from '@/components/ui/action-button'
import type { HoldingTransactionEntry } from '@/hooks/use-user-portfolio'
import { formatSummaryMoney } from '@/lib/holding-display'

export function HoldingTransactionDeleteButton({ transaction, onDelete, onDeleted }: {
  transaction: HoldingTransactionEntry
  onDelete: (transactionID: string) => Promise<void> | void
  onDeleted?: () => void
}) {
  const dialog = useRef<HTMLDialogElement>(null)
  const titleID = useId()
  const descriptionID = useId()
  const [isDeleting, setIsDeleting] = useState(false)
  const [error, setError] = useState('')
  const recordedAt = new Date(transaction.created_at)
  const recordedDate = Number.isNaN(recordedAt.getTime()) ? '--' : new Intl.DateTimeFormat('zh-CN', { timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit' }).format(recordedAt)

  async function confirmDelete() {
    if (isDeleting) return
    setIsDeleting(true)
    setError('')
    try {
      await onDelete(transaction.id)
      dialog.current?.close()
      onDeleted?.()
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : '删除失败，请稍后重试。')
    } finally {
      setIsDeleting(false)
    }
  }

  return <>
    <ActionButton type="button" aria-label={`删除 ${transaction.fund?.name || transaction.fund_id} 流水`} disabled={isDeleting} className="text-danger hover:border-danger" onClick={() => { setError(''); dialog.current?.showModal() }}>
      <Trash2 className="h-4 w-4" />删除流水
    </ActionButton>
    <dialog ref={dialog} aria-labelledby={titleID} aria-describedby={descriptionID} aria-busy={isDeleting} onCancel={event => { if (isDeleting) event.preventDefault() }} className="fixed inset-0 m-auto max-h-[calc(100dvh-2rem)] w-[calc(100%-2rem)] max-w-md overflow-y-auto rounded-xl border border-[var(--card-border)] bg-[var(--card-bg)] p-6 text-theme-primary backdrop:bg-overlay/60">
      <h2 id={titleID} className="text-xl font-semibold">删除这条流水？</h2>
      <dl className="my-5 space-y-3 rounded-lg bg-[var(--surface-subtle)] p-4 text-sm">
        <div><dt className="text-xs text-theme-secondary">基金</dt><dd className="mt-1 break-words">{transaction.fund?.name || transaction.fund_id} · {transaction.fund_id}</dd></div>
        <div className="flex justify-between gap-4"><div><dt className="text-xs text-theme-secondary">流水金额</dt><dd className="mt-1 tabular-nums">{formatSummaryMoney(transaction.amount)}</dd></div><div><dt className="text-xs text-theme-secondary">记录日期</dt><dd className="mt-1">{recordedDate}</dd></div></div>
      </dl>
      <p id={descriptionID} className="text-sm leading-7 text-theme-secondary">仅删除这条历史流水，当前持仓金额和份额不会改变。删除后无法恢复；关联旧流水的自动冲正可能不再可用。</p>
      {error && <p role="alert" className="mt-4 text-sm leading-6 text-danger">{error}</p>}
      <div className="mt-6 flex justify-end gap-3">
        <ActionButton type="button" autoFocus disabled={isDeleting} onClick={() => dialog.current?.close()}>取消</ActionButton>
        <ActionButton type="button" disabled={isDeleting} onClick={() => void confirmDelete()} className="border-transparent bg-danger text-[var(--on-accent)] hover:bg-danger/90">{isDeleting ? '正在删除…' : '确认删除'}</ActionButton>
      </div>
    </dialog>
  </>
}
