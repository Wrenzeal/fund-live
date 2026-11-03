'use client'

import { useEffect, useId, useState } from 'react'
import { ActionButton } from '@/components/ui/action-button'
import { useFundHistory, type Fund } from '@/hooks/use-fund-data'
import type { CreateHoldingBatchItem, HoldingEntry } from '@/hooks/use-user-portfolio'
import { holdingMoney } from '@/components/holdings-summary-metrics'

const inputClass = 'min-h-11 w-full rounded-lg border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2 text-base text-theme-primary'

export function HoldingOpeningComposer({ fundID, fundName, query, results, today, previousTradingDay, onQueryChange, onSelectFund, onSave }: {
  fundID: string; fundName: string; query: string; results: Fund[]; today: string; previousTradingDay: string
  onQueryChange: (value: string) => void; onSelectFund: (fund: Fund) => void
  onSave: (payload: CreateHoldingBatchItem) => Promise<HoldingEntry | null>
}) {
  const id = useId()
  const [showResults, setShowResults] = useState(false)
  const [basis, setBasis] = useState<'shares' | 'value'>('shares')
  const [shares, setShares] = useState('')
  const [cost, setCost] = useState('')
  const [marketValue, setMarketValue] = useState('')
  const [profit, setProfit] = useState('')
  const [holdingDate, setHoldingDate] = useState('')
  const [navDate, setNavDate] = useState('')
  const [note, setNote] = useState('')
  const [error, setError] = useState('')
  const [isSaving, setIsSaving] = useState(false)
  const { points } = useFundHistory(fundID || null, 1)
  const reference = points.at(-1)
  const date = basis === 'shares' ? holdingDate || today : navDate || reference?.date || previousTradingDay
  const derivedCost = marketValue.trim() && profit.trim() && Number.isFinite(Number(marketValue.replace(/,/g, ''))) && Number.isFinite(Number(profit.replace(/,/g, ''))) ? Number(marketValue.replace(/,/g, '')) - Number(profit.replace(/,/g, '')) : null

  useEffect(() => { setShares(''); setCost(''); setMarketValue(''); setProfit(''); setNavDate(''); setError('') }, [fundID])

  async function save() {
    if (isSaving) return
    setError('')
    if (!fundID || !date) { setError('请选择基金和登记日期。'); return }
    const clean = (value: string) => value.replace(/,/g, '').trim()
    const payload: CreateHoldingBatchItem = { fund_id: fundID, entry_type: 'opening', snapshot_date: date, note: note.trim(), source_platform: 'manual' }
    if (basis === 'shares') {
      if (!(Number(clean(shares)) > 0) || !(Number(clean(cost)) > 0)) { setError('请输入大于 0 的持有份额和剩余成本。'); return }
      payload.shares = clean(shares); payload.amount = clean(cost)
    } else {
      if (!(Number(clean(marketValue)) > 0) || !profit.trim() || derivedCost === null || derivedCost <= 0) { setError('请填写有效市值和持有盈亏，推算的剩余成本必须大于 0。'); return }
      payload.market_value = clean(marketValue); payload.holding_profit = clean(profit)
    }
    setIsSaving(true)
    try { await onSave(payload) } catch (cause) { setError(cause instanceof Error ? cause.message : '保存已有持仓失败，请重试。') } finally { setIsSaving(false) }
  }

  return <section aria-labelledby={id + '-title'} className="rounded-xl border border-[var(--card-border)] bg-[var(--card-bg)] p-5 sm:p-6">
    <h2 id={id + '-title'} className="text-xl font-semibold text-theme-primary">录入已有持仓</h2>
    <p className="mt-2 text-sm leading-7 text-theme-secondary">登记你已经持有的份额和成本，无需补造一笔买入。当前市值将按已公布官方净值计算。</p>
    <form className="mt-5 space-y-5" onSubmit={event => { event.preventDefault(); void save() }}>
      <label className="block space-y-2 text-sm text-theme-primary"><span>基金</span><input className={inputClass} value={query} onChange={event => { setShowResults(true); onQueryChange(event.target.value) }} placeholder="输入基金代码或名称" required /></label>
      {showResults && results.length > 0 && <ul aria-label="已有持仓基金搜索结果" className="grid gap-2 sm:grid-cols-2">{results.slice(0, 6).map(fund => <li key={fund.id}><ActionButton type="button" className="w-full justify-between whitespace-normal" onClick={() => { onSelectFund(fund); setShowResults(false) }}><span>{fund.name}</span><span>{fund.id}</span></ActionButton></li>)}</ul>}
      {fundID && <p className="text-sm text-theme-secondary">已选择 {fundName || fundID} · {fundID}</p>}
      <fieldset><legend className="mb-2 text-sm font-medium text-theme-primary">录入方式</legend><div className="flex flex-wrap gap-2"><ActionButton type="button" className={basis === 'shares' ? 'border-[var(--accent-primary)] bg-accent-soft' : ''} aria-pressed={basis === 'shares'} onClick={() => { setBasis('shares'); setError('') }}>份额与成本</ActionButton><ActionButton type="button" className={basis === 'value' ? 'border-[var(--accent-primary)] bg-accent-soft' : ''} aria-pressed={basis === 'value'} onClick={() => { setBasis('value'); setError('') }}>市值与持有盈亏</ActionButton></div></fieldset>
      <div className="grid gap-4 sm:grid-cols-2">
        {basis === 'shares' ? <><label className="space-y-2 text-sm text-theme-primary"><span>持有份额</span><input className={inputClass} inputMode="decimal" value={shares} onChange={event => setShares(event.target.value)} required placeholder="例如 500.000000" /></label><label className="space-y-2 text-sm text-theme-primary"><span>剩余成本（元）</span><input className={inputClass} inputMode="decimal" value={cost} onChange={event => setCost(event.target.value)} required placeholder="例如 900.00" /></label></> : <><label className="space-y-2 text-sm text-theme-primary"><span>持仓市值（元）</span><input className={inputClass} inputMode="decimal" value={marketValue} onChange={event => setMarketValue(event.target.value)} required placeholder="平台上对应净值日的持仓金额" /></label><label className="space-y-2 text-sm text-theme-primary"><span>持有盈亏（元）</span><input className={inputClass} inputMode="decimal" value={profit} onChange={event => setProfit(event.target.value)} required placeholder="盈利填正数，亏损填负数" /></label></>}
      </div>
      {basis === 'value' && <p className="text-sm leading-7 text-theme-secondary">按同一官方净值日的市值与盈亏换算份额、成本；持有盈亏不含已卖出收益或现金分红。{derivedCost !== null && <>推算剩余成本 {holdingMoney(derivedCost)}。</>}</p>}
      <label className="block space-y-2 text-sm text-theme-primary"><span>{basis === 'shares' ? '持仓登记日期' : '净值对应日期'}</span><input type="date" className={inputClass} value={date} max={today || undefined} onChange={event => basis === 'shares' ? setHoldingDate(event.target.value) : setNavDate(event.target.value)} required /></label>
      {basis === 'value' && <p className="text-sm leading-7 text-theme-secondary">{reference ? `最新已收录净值日 ${reference.date}。` : '保存时会校验所选日期的官方净值。'}金额需对应所选净值日；净值未公布时可先用持有份额录入。</p>}
      <details className="text-sm text-theme-secondary"><summary className="min-h-11 cursor-pointer py-3">备注（可选）</summary><label className="block space-y-2"><span className="sr-only">已有持仓备注</span><input className={inputClass} value={note} onChange={event => setNote(event.target.value)} /></label></details>
      {error && <p role="alert" className="text-sm leading-6 text-danger">{error}</p>}
      <div className="flex flex-wrap items-center justify-between gap-4 border-t border-[var(--card-border)] pt-5"><p className="text-sm text-theme-secondary">登记成功后可直接核对市值；不会重复记作新买入。</p><ActionButton type="submit" variant="primary" disabled={isSaving || !fundID}>{isSaving ? '正在保存…' : '保存已有持仓'}</ActionButton></div>
    </form>
  </section>
}
