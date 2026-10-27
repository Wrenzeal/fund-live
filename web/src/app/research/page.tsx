'use client'

import Link from 'next/link'
import { useMemo, useState, type FormEvent, type ReactNode } from 'react'
import { Activity, AlertTriangle, ArrowUpRight, BarChart3, BookOpen, ChevronDown, ExternalLink, RefreshCw } from 'lucide-react'
import { AppTopBar } from '@/components/app-top-bar'
import { SiteFooter } from '@/components/site-footer'
import { MarketFormulaWorkspace } from '@/components/market-formula-workspace'
import { MarketSectorFlows } from '@/components/market-sector-flows'
import { MarketComparisonChart, MarketDayChart, MarketMinuteChart } from '@/components/market-research-charts'
import {
  researchRefreshInterval,
  useMarketResearchInstrument,
  useMarketResearchOverview,
  useMarketResearchStudy,
  type ResearchInstrument,
  type ResearchInstrumentDetail,
  type ResearchQuote,
  type ResearchRule,
  type ResearchSourceMeta,
} from '@/hooks/use-market-research'
import { cn } from '@/lib/utils'

const RULES: { value: ResearchRule; label: string; direction: string }[] = [
  { value: 'ma20_up', label: '收盘价上穿 MA20', direction: '观察上涨' },
  { value: 'ma20_down', label: '收盘价下穿 MA20', direction: '观察下跌' },
  { value: 'rsi_recovery', label: 'RSI14 上穿 30', direction: '观察上涨' },
  { value: 'previous_low_break', label: '收盘价跌破前日低点', direction: '观察下跌' },
]
const PHASES: Record<string, string> = {
  morning: '早盘交易', afternoon: '午盘交易', call_auction: '集合竞价', lunch_break: '午间休市',
  pre_market: '盘前', after_hours: '已收盘', weekend: '周末休市', holiday: '节假日休市',
}
const inputClass = 'h-10 min-w-0 rounded-md border border-[var(--input-border)] bg-[var(--input-bg)] px-3 text-sm text-theme-primary'
const sectionClass = 'border-t border-[var(--card-border)] py-6'

function number(value?: number | null, precision = 2) {
  return value === undefined || value === null || !Number.isFinite(value) ? '--' : value.toLocaleString('zh-CN', { minimumFractionDigits: precision, maximumFractionDigits: precision })
}

function percent(value?: number | null, signed = true) {
  if (value === undefined || value === null || !Number.isFinite(value)) return '--'
  return `${signed && value > 0 ? '+' : ''}${value.toFixed(2)}%`
}

function price(value?: number | null) {
  return number(value, value !== undefined && value !== null && value < 10 ? 3 : 2)
}

function timestamp(value?: string, dateOnly = false) {
  if (!value) return '--'
  const parsed = new Date(value)
  if (!Number.isFinite(parsed.getTime())) return value
  return parsed.toLocaleString('zh-CN', {
    timeZone: 'Asia/Shanghai', month: '2-digit', day: '2-digit',
    ...(dateOnly ? {} : { hour: '2-digit', minute: '2-digit', second: '2-digit' }),
    hour12: false,
  })
}

function beijingDate(value?: string) {
  if (!value) return ''
  const parsed = new Date(value)
  if (!Number.isFinite(parsed.getTime())) return ''
  return parsed.toLocaleDateString('sv-SE', { timeZone: 'Asia/Shanghai' })
}

function changeClass(value?: number | null) {
  return value === undefined || value === null || value === 0 ? 'text-theme-secondary' : value > 0 ? 'text-up' : 'text-down'
}

function Header({ title, icon, action }: { title: string; icon?: ReactNode; action?: ReactNode }) {
  return <div className="mb-4 flex flex-wrap items-center justify-between gap-3"><h2 className="flex items-center gap-2 text-base font-semibold text-theme-primary">{icon}{title}</h2>{action}</div>
}

function Notice({ children, error = false }: { children: ReactNode; error?: boolean }) {
  return <div role="status" className={cn('flex items-start gap-2 border-l-2 px-3 py-2 text-sm leading-6', error ? 'border-rose-400 bg-rose-500/5 text-rose-400' : 'border-amber-400 bg-amber-500/5 text-amber-500')}><AlertTriangle className="mt-1 h-4 w-4 shrink-0" /><div className="min-w-0 break-words">{children}</div></div>
}

function Source({ meta, label }: { meta?: ResearchSourceMeta; label: string }) {
  if (!meta) return null
  return (
    <div className="space-y-1 text-xs leading-5 text-theme-muted">
      <div className="flex flex-wrap gap-x-3"><span>{label} · {meta.source || '来源未返回'}</span><span>读取 {timestamp(meta.fetched_at)}</span>{meta.stale && <span className="font-medium text-amber-500">缓存已过期</span>}</div>
      {meta.error && <div className="break-words text-amber-500">{meta.error}</div>}
    </div>
  )
}

function Metric({ label, value, detail, className }: { label: string; value: string; detail?: string; className?: string }) {
  return <div className="min-w-0"><div className="text-xs text-theme-muted">{label}</div><div className={cn('mt-1 break-words text-xl font-semibold tabular-nums text-theme-primary', className)}>{value}</div>{detail && <div className="mt-1 text-xs text-theme-muted">{detail}</div>}</div>
}

function QuoteRow({ instrument, quote, selected, onSelect }: { instrument: ResearchInstrument; quote?: ResearchQuote; selected: boolean; onSelect: () => void }) {
  return (
    <tr className={cn('border-t border-[var(--card-border)] transition-colors hover:bg-[var(--input-bg)]/50', selected && 'bg-[var(--input-bg)]/80')}>
      <td className="py-3 pr-3"><button onClick={onSelect} aria-pressed={selected} className="text-left"><span className={cn('block text-sm font-medium', selected ? 'text-[var(--accent-primary)]' : 'text-theme-primary')}>{instrument.name}</span><span className="mt-0.5 block text-xs text-theme-muted">{instrument.symbol} · {instrument.group}</span><span className="mt-0.5 block text-[10px] text-theme-muted sm:hidden">{quote ? timestamp(quote.timestamp) : '行情暂缺'}{quote?.delayed && <span className="ml-2 text-amber-500">延迟</span>}</span></button></td>
      <td className="whitespace-nowrap py-3 text-right text-sm tabular-nums text-theme-primary">{price(quote?.price)}</td>
      <td className={cn('whitespace-nowrap py-3 pl-4 text-right text-sm font-semibold tabular-nums', changeClass(quote?.change_pct))}>{percent(quote?.change_pct)}</td>
      <td className="hidden whitespace-nowrap py-3 pl-4 text-right text-xs text-theme-muted sm:table-cell">{quote ? timestamp(quote.timestamp) : '--'}{quote?.delayed && <span className="ml-2 text-amber-500">延迟</span>}</td>
    </tr>
  )
}

function Comparison({ instruments, phase }: { instruments: ResearchInstrument[]; phase?: string }) {
  const defaults = ['sz159994', 'sh561980', 'sz159363'].filter((symbol) => instruments.some((instrument) => instrument.symbol === symbol))
  const fallback = instruments.slice(0, 3).map((instrument) => instrument.symbol)
  const [selection, setSelection] = useState<string[] | null>(null)
  const [limit, setLimit] = useState(60)
  const symbols = Array.from(new Set(selection || (defaults.length >= 2 ? defaults : fallback))).filter((symbol) => instruments.some((instrument) => instrument.symbol === symbol)).slice(0, 3)
  const first = useMarketResearchInstrument(symbols[0] || null, phase)
  const second = useMarketResearchInstrument(symbols[1] || null, phase)
  const third = useMarketResearchInstrument(symbols[2] || null, phase)
  const results = [first, second, third].slice(0, symbols.length)
  const details = Array.from(new Map(
    results
      .flatMap((result, index) => result.data?.instrument.symbol === symbols[index] ? [result.data] : [])
      .map((detail) => [detail.instrument.symbol, detail] as const),
  ).values())
  const loading = results.some((result) => result.isLoading)
  const errors = results.flatMap((result) => result.error ? [result.error.message] : [])
  const toggle = (symbol: string) => {
    setSelection(symbols.includes(symbol) ? symbols.filter((item) => item !== symbol) : symbols.length < 3 ? [...symbols, symbol] : symbols)
  }
  return (
    <section className={sectionClass}>
      <Header title="相对走势" icon={<BarChart3 className="h-4 w-4 text-[var(--accent-primary)]" />} action={<label className="flex items-center gap-2 text-xs text-theme-muted">交易日<select aria-label="对比区间" className={inputClass} value={limit} onChange={(event) => setLimit(Number(event.target.value))}>{[20, 60, 120].map((value) => <option key={value} value={value}>{value} 日</option>)}</select></label>} />
      <fieldset className="mb-4 flex flex-wrap gap-x-5 gap-y-3"><legend className="mb-3 text-xs text-theme-muted">对比标的 · 选择 2–3 个</legend>{instruments.map((instrument) => <label key={instrument.symbol} className={cn('flex items-center gap-2 text-xs text-theme-secondary', !symbols.includes(instrument.symbol) && symbols.length >= 3 && 'opacity-40')}><input type="checkbox" checked={symbols.includes(instrument.symbol)} disabled={!symbols.includes(instrument.symbol) && symbols.length >= 3} onChange={() => toggle(instrument.symbol)} className="h-4 w-4 accent-[var(--accent-primary)]" />{instrument.name}</label>)}</fieldset>
      {errors.length > 0 && <div className="mb-3"><Notice error>部分对比数据读取失败：{[...new Set(errors)].join('；')}</Notice></div>}
      <div className="rounded-lg border border-[var(--card-border)] p-3 sm:p-4">
        {symbols.length < 2 ? <div className="flex h-72 items-center justify-center text-sm text-theme-muted">选择至少两个标的</div> : details.length !== symbols.length ? <div role="status" className="flex h-72 items-center justify-center text-sm text-theme-muted">{loading ? '正在读取对比数据…' : '部分标的暂无数据'}</div> : <MarketComparisonChart details={details} limit={limit} />}
      </div>
      <div className="mt-3 space-y-1">{details.filter((detail) => detail.history_meta.stale || detail.history_meta.error).map((detail) => <Source key={detail.instrument.symbol} meta={detail.history_meta} label={detail.instrument.name} />)}</div>
      <details className="mt-3 text-xs text-theme-muted"><summary className="cursor-pointer">对比口径</summary><p className="mt-2 leading-5">仅保留所有选中标的都有收盘价的共同交易日，以区间首日为 100。ETF 日线采用前复权；区间起点不同会影响相对表现。</p></details>
    </section>
  )
}

function EventStudy({ symbol, name }: { symbol: string; name: string }) {
  const [rule, setRule] = useState<ResearchRule>('ma20_up')
  const [holding, setHolding] = useState('10')
  const [fee, setFee] = useState('10')
  const [formError, setFormError] = useState('')
  const [parameters, setParameters] = useState<{ symbol: string; rule: ResearchRule; holding: number; fee_bps: number } | null>(null)
  const activeParameters = parameters || { symbol, rule: 'ma20_up' as ResearchRule, holding: 10, fee_bps: 10 }
  const study = useMarketResearchStudy(activeParameters)
  const ruleLabel = RULES.find((item) => item.value === activeParameters.rule)
  const downRule = ['ma20_down', 'previous_low_break'].includes(activeParameters.rule)
  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const holdingNumber = Number(holding)
    const feeNumber = Number(fee)
    if (!Number.isInteger(holdingNumber) || holdingNumber < 1 || holdingNumber > 60 || !Number.isFinite(feeNumber) || feeNumber < 0 || feeNumber > 200) {
      setFormError('持有期为 1–60 个交易日；费用为 0–200 个基点。')
      return
    }
    setFormError('')
    setParameters({ symbol, rule, holding: holdingNumber, fee_bps: feeNumber })
    if (activeParameters.rule === rule && activeParameters.holding === holdingNumber && activeParameters.fee_bps === feeNumber) void study.mutate()
  }
  return (
    <section className={sectionClass}>
      <Header title="规则事件研究" icon={<BookOpen className="h-4 w-4 text-[var(--accent-primary)]" />} action={<span className="text-xs text-theme-muted">{name} · 已完成日线</span>} />
      <form onSubmit={submit} className="grid items-end gap-3 sm:grid-cols-2 lg:grid-cols-[minmax(15rem,2fr)_1fr_1fr_auto]">
        <label className="grid gap-2 text-xs text-theme-secondary">事件规则<select aria-label="事件规则" value={rule} onChange={(event) => setRule(event.target.value as ResearchRule)} className={inputClass}>{RULES.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}</select></label>
        <label className="grid gap-2 text-xs text-theme-secondary">持有期（交易日）<input aria-label="持有期" type="number" min={1} max={60} step={1} required value={holding} onChange={(event) => setHolding(event.target.value)} className={inputClass} /></label>
        <label className="grid gap-2 text-xs text-theme-secondary">总交易费用（基点）<input aria-label="总交易费用" type="number" min={0} max={200} step={1} required value={fee} onChange={(event) => setFee(event.target.value)} className={inputClass} /></label>
        <button type="submit" disabled={study.isValidating} className="flex h-10 items-center justify-center gap-2 rounded-md border border-[var(--input-border)] bg-[var(--input-bg)] px-4 text-sm font-medium text-theme-primary disabled:opacity-50"><RefreshCw className={cn('h-4 w-4', study.isValidating && 'animate-spin')} />{study.isValidating ? '计算中' : '计算'}</button>
      </form>
      {formError && <p role="alert" className="mt-3 text-sm text-rose-400">{formError}</p>}
      {study.error && <div className="mt-4"><Notice error>{study.error.message}{study.data && '；以下保留上次成功结果。'}</Notice></div>}
      {!study.data && !study.error && <div role="status" className="py-10 text-center text-sm text-theme-muted">正在读取历史事件…</div>}
      {study.data && <>
        <div className="mt-5 flex flex-wrap gap-x-4 gap-y-1 text-xs text-theme-secondary"><span>{ruleLabel?.label}</span><span>{ruleLabel?.direction}</span><span>持有 {study.data.holding} 日</span><span>往返 {study.data.fee_bps} bp</span><span>截止 {study.data.last_completed_date || '--'}</span></div>
        <div className="mt-5 grid grid-cols-2 gap-x-5 gap-y-5 border-y border-[var(--card-border)] py-4 md:grid-cols-3 xl:grid-cols-6">
          <Metric label="完整样本" value={String(study.data.sample_count)} detail={`${study.data.pending_count} 个未完成样本`} />
          <Metric label="方向正确率" value={percent(study.data.direction_win_rate, false)} />
          <Metric label="标的平均涨跌" value={percent(study.data.mean_return_pct)} className={changeClass(study.data.mean_return_pct)} />
          <Metric label="平均费用后收益" value={downRule ? '不适用' : percent(study.data.mean_net_return_pct)} className={downRule ? undefined : changeClass(study.data.mean_net_return_pct)} />
          <Metric label="最差回撤上界" value={percent(study.data.worst_drawdown_pct, false)} className="text-down" />
          <Metric label="同期窗口基准" value={percent(study.data.baseline_mean_return_pct)} className={changeClass(study.data.baseline_mean_return_pct)} />
        </div>
        <details className="mt-4 text-xs text-theme-secondary"><summary className="flex cursor-pointer items-center gap-2 font-medium"><ChevronDown className="h-3.5 w-3.5" />研究方法与限制</summary><div className="mt-3 space-y-2 leading-6"><p>{study.data.method}</p><p>方向正确率是规则之后价格方向的历史比例，不代表未来获利概率。下跌规则只衡量方向，不计算做空收益。基点费用为往返总成本，100 bp = 1%；场外基金还需考虑净值确认、申赎截止时间及实际费率。历史价格结果不等于基金可实现收益。</p><p>基准采用同一历史区间全部可完成持有窗口的标的平均涨跌。回撤以正数表示损失幅度；日线无法确定同一天高低价发生顺序，按先高后低推算回撤上界，可能高估实际回撤，也不代表连续策略组合回撤。</p></div></details>
        <div className="mt-5 overflow-x-auto"><table className="w-full min-w-[650px] text-left text-xs"><caption className="pb-3 text-left text-sm font-medium text-theme-primary">完整事件明细 · {study.data.events.length} 条</caption><thead className="border-b border-[var(--card-border)] text-theme-muted"><tr>{['信号日', '进入参考日', '结束日', '标的涨跌', '方向结果', '费用后收益', '回撤上界'].map((label, index) => <th key={label} scope="col" className={cn('px-2 py-2 font-normal', index >= 3 && 'text-right')}>{label}</th>)}</tr></thead><tbody>{study.data.events.map((event) => <tr key={`${event.signal_date}-${event.entry_date}`} className="border-b border-[var(--card-border)]/50 text-theme-secondary"><td className="px-2 py-3">{event.signal_date}</td><td className="px-2 py-3">{event.entry_date}</td><td className="px-2 py-3">{event.exit_date}</td><td className={cn('px-2 py-3 text-right tabular-nums', changeClass(event.return_pct))}>{percent(event.return_pct)}</td><td className="px-2 py-3 text-right">{event.direction_pct > 0 ? '符合' : event.direction_pct < 0 ? '相反' : '持平'}</td><td className={cn('px-2 py-3 text-right tabular-nums', changeClass(event.net_return_pct))}>{percent(event.net_return_pct)}</td><td className={cn('px-2 py-3 text-right tabular-nums', 'text-down')}>{percent(event.drawdown_pct, false)}</td></tr>)}</tbody></table>{study.data.events.length === 0 && <div className="py-8 text-center text-sm text-theme-muted">当前区间没有可完成的规则样本</div>}</div>
        <div className="mt-3"><Source meta={study.data.history_meta} label="研究日线" /></div>
      </>}
    </section>
  )
}

function InstrumentPanel({ detail, loading, error, refresh }: { detail?: ResearchInstrumentDetail; loading: boolean; error?: Error; refresh: () => void }) {
  const [mode, setMode] = useState<'day' | 'minute'>('day')
  const [limit, setLimit] = useState(60)
  const quote = detail?.quote
  const technical = detail?.technical
  const quoteDate = beijingDate(quote?.timestamp)
  const minuteMatchesQuote = Boolean(detail?.minute?.date && detail.minute.date === quoteDate)
  const technicalDate = detail?.bars.reduce((latest, bar) => bar.date > latest ? bar.date : latest, '')
  return (
    <section className={sectionClass}>
      <Header title={detail?.instrument.name || '标的走势'} icon={<Activity className="h-4 w-4 text-[var(--accent-primary)]" />} action={<button type="button" onClick={refresh} title="刷新标的" aria-label="刷新标的" className="rounded-md border border-[var(--input-border)] p-2 text-theme-secondary"><RefreshCw className={cn('h-4 w-4', loading && 'animate-spin')} /></button>} />
      {error && <div className="mb-4"><Notice error>{error.message}{detail && '；保留上次成功数据。'}</Notice></div>}
      {detail && <>
        <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between"><div><div className="flex items-baseline gap-4"><span className="text-2xl font-semibold tabular-nums text-theme-primary">{price(quote?.price)}</span><span className={cn('text-lg font-semibold tabular-nums', changeClass(quote?.change_pct))}>{percent(quote?.change_pct)}</span></div><div className="mt-2 flex flex-wrap gap-x-3 text-xs text-theme-muted"><span>{detail.instrument.symbol}</span><span>行情 {quote ? timestamp(quote.timestamp) : '暂缺'}</span>{quote?.delayed && <span className="text-amber-500">上游行情延迟</span>}</div></div><div className="flex flex-wrap gap-x-5 gap-y-2 text-xs text-theme-secondary"><span>开 {price(quote?.open)}</span><span>高 {price(quote?.high)}</span><span>低 {price(quote?.low)}</span><span>昨收 {price(quote?.prev_close)}</span></div></div>
        <Source meta={detail.quote_meta} label="行情" />
      </>}
      <div className="mt-4 rounded-lg border border-[var(--card-border)] p-3 sm:p-4">
        <div className="mb-3 flex flex-wrap items-center justify-between gap-3"><div role="tablist" aria-label="走势周期" className="flex border-b border-[var(--card-border)]">{([{ value: 'day', label: '日线' }, { value: 'minute', label: '分时' }] as const).map((tab) => <button key={tab.value} type="button" role="tab" aria-selected={mode === tab.value} onClick={() => setMode(tab.value)} className={cn('min-h-10 border-b-2 px-4 text-sm', mode === tab.value ? 'border-[var(--accent-primary)] text-theme-primary' : 'border-transparent text-theme-muted')}>{tab.label}</button>)}</div>{mode === 'day' ? <label className="flex items-center gap-2 text-xs text-theme-muted">区间<select aria-label="日线区间" value={limit} onChange={(event) => setLimit(Number(event.target.value))} className={inputClass}>{[20, 60, 120].map((value) => <option key={value} value={value}>{value} 日</option>)}</select></label> : <span className="text-xs text-theme-muted">{detail?.minute?.date || '--'}{minuteMatchesQuote && ' · 昨收参考虚线'}</span>}</div>
        {mode === 'day' ? <><div className="mb-2 flex flex-wrap gap-x-4 text-xs text-theme-muted"><span className="text-[var(--accent-primary)]">收盘价</span><span className="text-[#e6a23c]">MA5</span><span className="text-[#c97db8]">MA20</span><span className="text-[#72bfa3]">MA60</span></div><MarketDayChart bars={detail?.bars || []} limit={limit} loading={loading} /></> : <><MarketMinuteChart minute={detail?.minute || null} previousClose={minuteMatchesQuote ? quote?.prev_close : undefined} loading={loading} />{detail?.minute && !minuteMatchesQuote && <p className="mt-2 text-xs text-amber-500">分时日期与报价日期不一致，昨收参考线暂不显示</p>}</>}
      </div>
      <div className="mt-3"><Source meta={mode === 'day' ? detail?.history_meta : detail?.minute_meta} label={mode === 'day' ? '日线' : '分时'} /></div>
      {technical ? <>
        <div className="mt-5 flex flex-wrap items-center gap-x-3 gap-y-1 border-l-2 border-[var(--accent-primary)] pl-3 text-sm"><h3 className="font-semibold text-theme-primary">技术指标 · 日线截至 {technicalDate || '日期未提供'}</h3><span className={cn('text-xs', technical.provisional || technicalDate && quoteDate && technicalDate < quoteDate ? 'text-amber-500' : 'text-theme-muted')}>{technical.provisional ? '当日未完成' : technicalDate && quoteDate && technicalDate < quoteDate ? '早于最新报价日期' : '已完成日线'}</span></div>
        <div className="mt-5 grid grid-cols-2 gap-x-4 gap-y-4 border-y border-[var(--card-border)] py-4 sm:grid-cols-5"><Metric label="MA5" value={price(technical.ma5)} /><Metric label="MA20" value={price(technical.ma20)} /><Metric label="MA60" value={price(technical.ma60)} /><Metric label="RSI14" value={number(technical.rsi14)} /><Metric label="前日低点" value={price(technical.previous_low)} /></div>
        {technical.provisional && <p className="mt-3 text-xs text-amber-500">包含未完成交易日，盘中技术值会变化</p>}
        <ul className="mt-4 divide-y divide-[var(--card-border)]">{technical.conditions.map((condition, index) => <li key={`${condition.label}-${index}`} className="flex flex-col gap-1 py-3 text-sm sm:flex-row sm:gap-5"><span className={cn('shrink-0 font-medium sm:w-40', ['positive', 'up', 'bullish'].includes(condition.tone) ? 'text-up' : ['negative', 'down', 'bearish', 'risk'].includes(condition.tone) ? 'text-down' : 'text-theme-primary')}>{condition.label}</span><span className="text-theme-secondary">{condition.detail}</span></li>)}</ul>
      </> : !loading && <p className="mt-4 text-sm text-theme-muted">日线不足，暂无法计算技术条件</p>}
    </section>
  )
}

export default function ResearchPage() {
  const overview = useMarketResearchOverview()
  const [requestedSymbol, setRequestedSymbol] = useState('sz159994')
  const [group, setGroup] = useState('all')
  const instruments = useMemo(() => Array.from(new Map((overview.data?.instruments || []).map((instrument) => [instrument.symbol, instrument])).values()), [overview.data?.instruments])
  const selected = instruments.find((instrument) => instrument.symbol === requestedSymbol) || instruments[0]
  const detail = useMarketResearchInstrument(selected?.symbol || null, overview.data?.market_phase)
  const groups = [...new Set(instruments.map((instrument) => instrument.group))]
  const filtered = group === 'all' ? instruments : instruments.filter((instrument) => instrument.group === group)
  const quotes = new Map((overview.data?.quotes || []).map((quote) => [quote.symbol, quote]))
  const indexSymbols = ['sh000001', 'sz399001', 'sz399006', 'sh000688', 'sh000300']
  const indices = indexSymbols.flatMap((symbol) => quotes.has(symbol) ? [quotes.get(symbol)!] : [])
  const breadth = overview.data?.breadth
  const breadthTotal = breadth ? breadth.advancing + breadth.declining + breadth.unchanged : 0
  const refreshing = overview.isValidating || detail.isValidating
  const refresh = () => { void overview.mutate(); void detail.mutate() }
  return (
    <div className="min-h-[100dvh]">
      <AppTopBar standaloneResearch />
      <main id="main-content" className="container mx-auto px-3 pb-10 sm:px-4">
        <header className="flex flex-col gap-4 py-6 sm:flex-row sm:items-center sm:justify-between"><div><h1 className="text-2xl font-semibold text-theme-primary">市场研究</h1><div className="mt-2 flex flex-wrap gap-x-3 gap-y-1 text-xs text-theme-muted"><span>{PHASES[overview.data?.market_phase || ''] || '等待市场状态'}</span><span>北京时间</span><span>读取 {timestamp(overview.data?.generated_at)}</span><span>自动刷新 {researchRefreshInterval(overview.data?.market_phase) / 1000} 秒</span></div></div><div className="flex items-center gap-3"><Link href="/holdings" className="flex items-center gap-1 text-sm text-theme-secondary hover:text-theme-primary">持仓<ArrowUpRight className="h-4 w-4" /></Link><button type="button" onClick={refresh} disabled={refreshing} title="刷新研究数据" aria-label="刷新研究数据" className="flex h-10 w-10 items-center justify-center rounded-md border border-[var(--input-border)] bg-[var(--input-bg)] text-theme-secondary disabled:opacity-50"><RefreshCw className={cn('h-4 w-4', refreshing && 'animate-spin')} /></button></div></header>
        {overview.error && <Notice error>{overview.error.message}{overview.data ? '；当前保留上次成功数据。' : '，请确认本机数据服务已经启动。'}</Notice>}
        {overview.isLoading && <div role="status" className="border-t border-[var(--card-border)] py-14 text-center text-sm text-theme-muted">正在读取行情、市场宽度与消息…</div>}
        {overview.data && <>
          {(overview.data.quote_meta.stale || overview.data.breadth_meta.stale || overview.data.news_meta.stale) && <div className="mb-3"><Notice>部分上游数据已过期，时间与错误详情见各数据来源。当前显示缓存数据。</Notice></div>}
          <section className={sectionClass}>
            <Header title="大盘概览" icon={<Activity className="h-4 w-4 text-[var(--accent-primary)]" />} />
            <div className="grid grid-cols-2 gap-x-6 gap-y-5 sm:grid-cols-3 lg:grid-cols-5">{indices.map((quote) => <button key={quote.symbol} type="button" onClick={() => setRequestedSymbol(quote.symbol)} className="min-w-0 text-left"><Metric label={quote.name} value={number(quote.price)} detail={timestamp(quote.timestamp)} /><div className={cn('mt-1 text-sm font-semibold tabular-nums', changeClass(quote.change_pct))}>{percent(quote.change_pct)}{quote.delayed && <span className="ml-2 text-xs font-normal text-amber-500">延迟</span>}</div></button>)}</div>
            {indices.length === 0 && <p className="text-sm text-theme-muted">暂无指数行情</p>}
            <div className="mt-4"><Source meta={overview.data.quote_meta} label="指数行情" /></div>
            <div className="mt-5 border-t border-[var(--card-border)] pt-5">{breadth ? <><div className="grid grid-cols-2 gap-x-6 gap-y-4 sm:grid-cols-4"><Metric label="沪深上涨" value={number(breadth.advancing, 0)} className="text-up" /><Metric label="沪深下跌" value={number(breadth.declining, 0)} className="text-down" /><Metric label="平盘" value={number(breadth.unchanged, 0)} /><Metric label="沪深成交额" value={`${number(breadth.turnover_yuan / 100_000_000, 1)} 亿`} /></div>{breadthTotal > 0 && <div className="mt-4 flex h-1.5 w-full overflow-hidden rounded-sm" aria-label={`上涨 ${breadth.advancing} 家，下跌 ${breadth.declining} 家，平盘 ${breadth.unchanged} 家`}><span style={{ width: `${breadth.advancing / breadthTotal * 100}%`, background: 'var(--accent-up)' }} /><span style={{ width: `${breadth.declining / breadthTotal * 100}%`, background: 'var(--accent-down)' }} /><span className="bg-[var(--text-muted)]" style={{ width: `${breadth.unchanged / breadthTotal * 100}%` }} /></div>}</> : <p className="text-sm text-theme-muted">市场宽度暂不可用</p>}<div className="mt-3"><Source meta={overview.data.breadth_meta} label="沪深市场宽度" /></div></div>
          </section>
          <section className={sectionClass}>
            <Header title="标的行情" action={<label className="flex items-center gap-2 text-xs text-theme-muted">分组<select aria-label="标的分组" value={group} onChange={(event) => setGroup(event.target.value)} className={inputClass}><option value="all">全部</option>{groups.map((item) => <option key={item} value={item}>{item}</option>)}</select></label>} />
            <div className="overflow-x-auto"><table className="w-full text-left"><thead className="text-xs text-theme-muted"><tr><th className="pb-2 text-left font-normal">标的</th><th className="pb-2 text-right font-normal">最新</th><th className="pb-2 pl-4 text-right font-normal">涨跌</th><th className="hidden pb-2 pl-4 text-right font-normal sm:table-cell">行情时间</th></tr></thead><tbody>{filtered.map((instrument) => <QuoteRow key={instrument.symbol} instrument={instrument} quote={quotes.get(instrument.symbol)} selected={selected?.symbol === instrument.symbol} onSelect={() => setRequestedSymbol(instrument.symbol)} />)}</tbody></table></div>
            {!filtered.length && <p className="py-6 text-center text-sm text-theme-muted">当前分组暂无标的</p>}
            <label className="mt-4 flex flex-col gap-2 text-xs text-theme-secondary sm:flex-row sm:items-center">当前研究<select aria-label="当前研究标的" className={cn(inputClass, 'sm:min-w-72')} value={selected?.symbol || ''} onChange={(event) => setRequestedSymbol(event.target.value)}>{instruments.map((instrument) => <option key={instrument.symbol} value={instrument.symbol}>{instrument.name} · {instrument.symbol}</option>)}</select></label>
          </section>
          {selected && <><InstrumentPanel key={`detail-${selected.symbol}`} detail={detail.data?.instrument.symbol === selected.symbol ? detail.data : undefined} loading={detail.isLoading} error={detail.error} refresh={() => void detail.mutate()} /><Comparison instruments={instruments} phase={overview.data.market_phase} /><MarketFormulaWorkspace instruments={instruments} phase={overview.data.market_phase} /><EventStudy key={`study-${selected.symbol}`} symbol={selected.symbol} name={selected.name} /></>}
          <MarketSectorFlows phase={overview.data.market_phase} />
          <section className={sectionClass}>
            <Header title="最新消息" icon={<ExternalLink className="h-4 w-4 text-[var(--accent-primary)]" />} action={<span className="text-xs text-theme-muted">原文报道</span>} />
            <ul className="divide-y divide-[var(--card-border)]">{overview.data.news.map((item, index) => <li key={`${item.url}-${index}`}><a href={item.url} target="_blank" rel="noopener noreferrer" className="flex items-start justify-between gap-4 py-3 text-sm text-theme-secondary hover:text-theme-primary"><span className="min-w-0"><span className="block break-words leading-6">{item.title}</span><span className="mt-1 block text-xs text-theme-muted">{item.date || '日期未提供'}{item.related && <span className="ml-3 text-[var(--accent-primary)]">科技相关</span>}</span></span><ExternalLink className="mt-1 h-3.5 w-3.5 shrink-0 text-theme-muted" /></a></li>)}</ul>
            {overview.data.news.length === 0 && <p className="py-6 text-sm text-theme-muted">暂无可读取的消息原文</p>}
            <div className="mt-3"><Source meta={overview.data.news_meta} label="消息" /></div>
          </section>
          <details className="border-t border-[var(--card-border)] py-5 text-xs text-theme-muted"><summary className="cursor-pointer font-medium text-theme-secondary">数据口径</summary><div className="mt-3 space-y-2 leading-6"><p>公开行情可能延迟，行情时间取自上游；读取时间只表示本机获取数据的时间。接口异常时可能保留已过期缓存，不补造行情。</p><p>技术条件是历史价格的计算结果，盘中值可能变化；ETF 市场价格并非场外基金最终净值。新闻标题是原文报道入口，未经独立核实，不能视作事实或交易指令。</p></div></details>
        </>}
      </main>
      <SiteFooter compact />
    </div>
  )
}
