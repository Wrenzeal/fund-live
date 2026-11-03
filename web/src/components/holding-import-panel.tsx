"use client";

import { useMemo, useState } from 'react'
import { ActionButton } from '@/components/ui/action-button'
import type { CreateHoldingBatchItem, HoldingAggregateEntry, HoldingBatchCreateResult } from '@/hooks/use-user-portfolio'
import { HOLDING_SOURCE_OPTIONS, type HoldingSourceFilter } from '@/lib/holding-sources'
import { parseHoldingImport, type HoldingImportKind, type HoldingImportBasis } from '@/lib/holding-import'
import type { TradeTiming } from '@/lib/holding-display'

interface Props {
  recentAggregates: HoldingAggregateEntry[]; compact?: boolean
  onSelectDraft?: (draft: { fundID: string; amount: string; note: string; tradeAt: string }) => void
  onImportBatch?: (items: CreateHoldingBatchItem[]) => Promise<HoldingBatchCreateResult | null> | HoldingBatchCreateResult | null | void
}
const templates = {
  shares: '基金代码,持有份额,剩余成本,登记日期,备注\n014662,500,900,2026-11-02,已有持仓迁移',
  value: '基金代码,持仓市值,持有盈亏,净值日期,备注\n014662,1000,100,2026-11-02,同一净值日的平台数据',
  buy: '基金代码,买入金额,交易日期,提交时段,备注\n014662,1000,2026-11-02,15:00前,历史买入',
}
export function HoldingImportPanel({ onImportBatch, onSelectDraft }: Props) {
  const [kind, setKind] = useState<HoldingImportKind>('opening')
  const [basis, setBasis] = useState<HoldingImportBasis>('shares')
  const [source, setSource] = useState<HoldingSourceFilter>('alipay')
  const [defaultTiming, setDefaultTiming] = useState<TradeTiming>('before_close')
  const [raw, setRaw] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [result, setResult] = useState<HoldingBatchCreateResult | null>(null)
  const [imported, setImported] = useState<Set<number>>(new Set())
  const parsed = useMemo(() => parseHoldingImport(raw, kind, basis, source, defaultTiming), [raw, kind, basis, source, defaultTiming])
  const ready = parsed.rows.filter(row => row.errors.length === 0 && !imported.has(row.index))
  const reset = () => { setImported(new Set()); setResult(null); setError('') }
  async function submit() {
    if (!onImportBatch || busy || !ready.length || parsed.errors.length) return
    setBusy(true); setError('')
    try {
      const next = await onImportBatch(ready.map(row => row.payload))
      if (next) {
        setResult(next)
        const failed = new Set(next.failed?.map(item => item.index) || [])
        if (next.created_count + next.failed_count === ready.length) setImported(previous => new Set([...previous, ...ready.filter((_, index) => !failed.has(index)).map(row => row.index)]))
      }
    } catch (cause) { setError(cause instanceof Error ? cause.message : '导入失败，请重试。') } finally { setBusy(false) }
  }
  const control = 'min-h-11 rounded-lg border border-[var(--input-border)] bg-[var(--input-bg)] px-3 text-sm text-theme-primary'
  return <section aria-label="持仓批量导入" className="rounded-xl border border-[var(--card-border)] bg-[var(--card-bg)] p-5 sm:p-6">
    <h2 className="text-xl font-semibold text-theme-primary">批量导入</h2>
    <p className="mt-2 text-sm leading-7 text-theme-secondary">每行新增一笔持仓，不覆盖已有持仓。已有持仓与买入记录分开导入，最多 20 行；已成功的行不会在本次预览中重复提交。</p>
    <div className="mt-4 flex flex-wrap gap-2"><ActionButton type="button" className={kind === 'opening' ? 'border-[var(--accent-primary)] bg-accent-soft' : ''} aria-pressed={kind === 'opening'} disabled={busy} onClick={() => { setKind('opening'); setRaw(''); reset() }}>导入已有持仓</ActionButton><ActionButton type="button" className={kind === 'buy' ? 'border-[var(--accent-primary)] bg-accent-soft' : ''} aria-pressed={kind === 'buy'} disabled={busy} onClick={() => { setKind('buy'); setRaw(''); reset() }}>导入买入记录</ActionButton></div>
    <div className="mt-4 flex flex-wrap items-end gap-4">
      {kind === 'opening' && <label className="grid gap-2 text-sm text-theme-secondary">已有持仓录入方式<select className={control} value={basis} disabled={busy} onChange={event => { setBasis(event.target.value as HoldingImportBasis); setRaw(''); reset() }}><option value="shares">份额与成本</option><option value="value">市值与持有盈亏</option></select></label>}
      <label className="grid gap-2 text-sm text-theme-secondary">来源平台<select className={control} value={source} disabled={busy} onChange={event => { setSource(event.target.value as HoldingSourceFilter); reset() }}>{HOLDING_SOURCE_OPTIONS.map(option => <option key={option.value} value={option.value}>{option.label}</option>)}</select></label>
      {kind === 'buy' && <label className="grid gap-2 text-sm text-theme-secondary">未填写时段的日期按<select className={control} value={defaultTiming} disabled={busy} onChange={event => { setDefaultTiming(event.target.value as TradeTiming); reset() }}><option value="before_close">15:00 前</option><option value="after_close">15:00 后</option></select></label>}
      <ActionButton type="button" disabled={busy} onClick={() => { setRaw(templates[kind === 'buy' ? 'buy' : basis]); reset() }}>使用示例模板</ActionButton>
    </div>
    <label className="mt-5 block space-y-2 text-sm text-theme-primary"><span>CSV 数据</span><textarea value={raw} disabled={busy} onChange={event => { setRaw(event.target.value); reset() }} rows={6} placeholder={templates[kind === 'buy' ? 'buy' : basis]} className="w-full rounded-lg border border-[var(--input-border)] bg-[var(--input-bg)] p-3 font-mono text-sm" /></label>
    {kind === 'opening' && basis === 'value' && <p className="mt-3 text-sm leading-7 text-theme-secondary">市值与持有盈亏必须对应同一官方净值日。保存时按该日净值换算份额，盈亏不含已卖出收益与现金分红。</p>}
    {parsed.errors.map(message => <p role="alert" key={message} className="mt-3 text-sm text-danger">{message}</p>)}
    <div className="mt-5 space-y-3">{parsed.rows.map(row => <div key={row.index} className="rounded-lg border border-[var(--card-border)] p-3 text-sm"><div className="flex flex-wrap justify-between gap-3"><div><p className="font-medium text-theme-primary">第 {row.index} 行 · {row.payload.fund_id || '基金待补'}</p><p className="mt-1 break-words text-theme-secondary">{row.payload.entry_type === 'opening' ? `已有持仓 · ${row.payload.shares ? '份额 ' + row.payload.shares + ' · 成本 ' + row.payload.amount : '市值 ' + row.payload.market_value + ' · 持有盈亏 ' + row.payload.holding_profit} · ${row.payload.snapshot_date}` : `买入金额 ${row.payload.amount} · ${row.payload.trade_at}`}</p></div><span className={row.errors.length ? 'text-danger' : 'text-theme-secondary'}>{imported.has(row.index) ? '已导入' : row.errors.length ? row.errors.join('；') : '可导入'}</span></div>{kind === 'buy' && !row.errors.length && onSelectDraft && <ActionButton type="button" className="mt-2" onClick={() => onSelectDraft({ fundID: row.payload.fund_id, amount: row.payload.amount || '', note: row.payload.note || '', tradeAt: row.payload.trade_at || '' })}>带入买入表单</ActionButton>}</div>)}</div>
    {error && <p role="alert" className="mt-4 text-sm text-danger">{error}</p>}
    {result && <div role="status" className="mt-4 text-sm leading-7 text-theme-secondary"><p>已导入 {result.created_count} 行，失败 {result.failed_count} 行。</p>{result.failed?.map(item => <p key={item.index}>{item.fund_id}：{item.message}</p>)}</div>}
    <div className="mt-5 flex flex-wrap items-center justify-between gap-3 border-t border-[var(--card-border)] pt-4"><p className="text-sm text-theme-secondary">本次可提交 {ready.length} 行</p><ActionButton variant="primary" type="button" disabled={busy || !ready.length || !!parsed.errors.length} onClick={() => void submit()}>{busy ? '正在导入…' : '确认导入'}</ActionButton></div>
  </section>
}
