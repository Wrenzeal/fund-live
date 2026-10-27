'use client'

import { useEffect, useRef, useState } from 'react'
import type { ResearchInstrument } from '@/hooks/use-market-research'
import { researchRefreshInterval } from '@/hooks/use-market-research'
import type { WorkspaceConfig, WorkspaceResult } from '@/lib/market-workspace-types'
import { MarketFormulaChart } from './market-formula-chart'

const STORAGE_KEY = 'fund-live.market-workspaces.v1'
const inputClass = 'h-10 min-w-0 w-full rounded-md border border-[var(--input-border)] bg-[var(--input-bg)] px-3 text-sm text-theme-primary'
const buttonClass = 'rounded-md border border-[var(--card-border)] px-3 py-2 text-sm text-theme-secondary hover:text-theme-primary disabled:opacity-40'
const ALIASES = ['A', 'B', 'C']
const OPERATORS = [{ value: 'gt', label: '大于 >' }, { value: 'lt', label: '小于 <' }, { value: 'cross_up', label: '向上穿越' }, { value: 'cross_down', label: '向下穿越' }]
type SavedWorkspace = { id: string; config: WorkspaceConfig }

function preset(instruments: ResearchInstrument[]): WorkspaceConfig {
  const a = instruments.find(item => item.symbol === 'sh512480') || instruments.find(item => item.symbol === 'sh561980') || instruments[0]
  const b = instruments.find(item => item.symbol === 'sh000688') || instruments.find(item => item.symbol === 'sh000001') || instruments.find(item => item.symbol !== a?.symbol)
  return { name: '半导体趋势观察', sources: [{ alias: 'A', symbol: a?.symbol || '', field: 'close' }, ...(b ? [{ alias: 'B', symbol: b.symbol, field: 'close' as const }] : [])], curves: [{ id: 'curve1', name: '趋势强度', expression: 'A/MA(A,20)*100' }, ...(b ? [{ id: 'curve2', name: '相对强弱', expression: 'ROC(A,5)-ROC(B,5)' }] : [])], event: { mode: 'all', conditions: [{ left: 'A', operator: 'cross_up', right: 'MA(A,5)' }, ...(b ? [{ left: 'B', operator: 'gt' as const, right: 'MA(B,5)' }] : [])], direction: 'up' }, target_symbol: a?.symbol || '', holding: 5, fee_bps: 10, limit: 120 }
}

function record(value: unknown): value is Record<string, unknown> { return typeof value === 'object' && value !== null && !Array.isArray(value) }
function text(value: unknown, max: number) { return typeof value === 'string' && value.trim().length > 0 && value.length <= max }

// Validate imported/local expressions without executing code or importing server data modules.
function validExpression(value: string, aliases: Set<string>) {
  const tokens: string[] = []
  const pattern = /\s*(?:(\d+(?:\.\d*)?|\.\d+)|([A-Za-z][A-Za-z0-9_]*)|([+\-*/(),]))/y
  let offset = 0
  while (offset < value.trimEnd().length) {
    pattern.lastIndex = offset
    const match = pattern.exec(value)
    if (!match || tokens.length >= 256) return false
    tokens.push(match[1] || match[2] || match[3]); offset = pattern.lastIndex
  }
  let position = 0
  function operand(depth: number): boolean {
    if (depth > 24) return false
    const token = tokens[position++]
    if (token === '+' || token === '-') return operand(depth + 1)
    if (token === '(') return expression(depth + 1) && tokens[position++] === ')'
    if (token && /^(?:\d|\.)/.test(token)) return Number.isFinite(Number(token))
    if (!token) return false
    if (tokens[position] !== '(') return aliases.has(token)
    if (!['MA', 'EMA', 'RSI', 'ROC', 'STD', 'MIN', 'MAX', 'REF', 'ABS'].includes(token)) return false
    position++
    if (!expression(depth + 1)) return false
    if (token !== 'ABS') {
      if (tokens[position++] !== ',') return false
      const raw = tokens[position++]
      if (!raw || !/^\d+$/.test(raw) || Number(raw) < (token === 'REF' ? 0 : 1) || Number(raw) > 200) return false
    }
    return tokens[position++] === ')'
  }
  function expression(depth: number): boolean {
    if (!operand(depth)) return false
    while (['+', '-', '*', '/'].includes(tokens[position])) { position++; if (!operand(depth)) return false }
    return true
  }
  return expression(0) && position === tokens.length
}

function validateConfig(value: unknown, instruments: ResearchInstrument[]): value is WorkspaceConfig {
  if (!record(value) || !text(value.name, 60) || !Array.isArray(value.sources) || !Array.isArray(value.curves) || !record(value.event)) return false
  const symbols = new Set(instruments.map(item => item.symbol))
  if (value.sources.length < 1 || value.sources.length > 3 || value.curves.length < 1 || value.curves.length > 3) return false
  const aliases = new Set<string>()
  for (const source of value.sources) {
    if (!record(source) || typeof source.alias !== 'string' || !ALIASES.includes(source.alias) || aliases.has(source.alias) || typeof source.symbol !== 'string' || !symbols.has(source.symbol) || !['close', 'open', 'high', 'low', 'volume'].includes(String(source.field))) return false
    aliases.add(source.alias)
  }
  const ids = new Set<string>()
  for (const curve of value.curves) {
    if (!record(curve) || typeof curve.id !== 'string' || !/^curve[1-3]$/.test(curve.id) || ids.has(curve.id) || !text(curve.name, 40) || !text(curve.expression, 256)) return false
    if (!validExpression(curve.expression as string, aliases)) return false
    ids.add(curve.id)
  }
  if (!['all', 'any'].includes(String(value.event.mode)) || !['up', 'down'].includes(String(value.event.direction)) || !Array.isArray(value.event.conditions) || value.event.conditions.length < 1 || value.event.conditions.length > 3) return false
  if (!value.event.conditions.every(condition => record(condition) && text(condition.left, 256) && text(condition.right, 256) && OPERATORS.some(op => op.value === condition.operator) && validExpression(condition.left as string, aliases) && validExpression(condition.right as string, aliases))) return false
  return typeof value.target_symbol === 'string' && value.sources.some(source => source.symbol === value.target_symbol) && symbols.has(value.target_symbol) && Number.isInteger(value.holding) && Number(value.holding) >= 1 && Number(value.holding) <= 60 && typeof value.fee_bps === 'number' && Number.isFinite(value.fee_bps) && value.fee_bps >= 0 && value.fee_bps <= 200 && typeof value.limit === 'number' && [20, 60, 120, 640].includes(value.limit)
}

function loadSaved(instruments: ResearchInstrument[]) {
  const raw = localStorage.getItem(STORAGE_KEY)
  if (!raw) return { saved: [] as SavedWorkspace[], notice: '' }
  try {
    const parsed: unknown = JSON.parse(raw)
    if (!record(parsed) || parsed.version !== 1 || !Array.isArray(parsed.workspaces) || parsed.workspaces.length > 30) throw new Error()
    const ids = new Set<string>()
    const saved = parsed.workspaces.filter((item): item is SavedWorkspace => {
      if (!record(item) || typeof item.id !== 'string' || item.id.length > 100 || !item.id || ids.has(item.id) || !validateConfig(item.config, instruments)) return false
      ids.add(item.id)
      return true
    })
    return { saved, notice: saved.length === parsed.workspaces.length ? '' : '部分本地方案格式无效或标的已不可用，已跳过。' }
  } catch { return { saved: [] as SavedWorkspace[], notice: '本地方案无法读取，已保留原存储；仍可新建并计算。' } }
}

function valueLabel(value: number | null | undefined, percentage = false) { return value === null || value === undefined || !Number.isFinite(value) ? '--' : `${value.toFixed(2)}${percentage ? '%' : ''}` }
function clock(value: string) { const date = new Date(value); return Number.isFinite(date.getTime()) ? date.toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai', hour12: false }) : value }

export function MarketFormulaWorkspace({ instruments, phase }: { instruments: ResearchInstrument[]; phase?: string }) {
  const [draft, setDraft] = useState<WorkspaceConfig>(() => preset(instruments))
  const [submitted, setSubmitted] = useState<WorkspaceConfig | null>(null)
  const [result, setResult] = useState<{ data: WorkspaceResult; config: WorkspaceConfig } | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [normalized, setNormalized] = useState(false)
  const [automatic, setAutomatic] = useState(false)
  const [saved, setSaved] = useState<SavedWorkspace[]>([])
  const [selectedId, setSelectedId] = useState('')
  const [loadedId, setLoadedId] = useState('')
  const [notice, setNotice] = useState('')
  const [importText, setImportText] = useState('')
  const request = useRef<{ controller: AbortController; id: number } | null>(null)
  const sequence = useRef(0)
  const instrumentSignature = instruments.map(item => item.symbol).sort().join(',')
  const submittedRef = useRef<WorkspaceConfig | null>(null)

  useEffect(() => {
    let cancelled = false
    // Run after hydration; server rendering never reads browser storage.
    queueMicrotask(() => {
      if (cancelled) return
      try { const loaded = loadSaved(instruments); setSaved(loaded.saved); setNotice(loaded.notice) }
      catch { setNotice('浏览器不允许读取本地存储；仍可计算与导出方案。') }
    })
    return () => { cancelled = true }
  // Names/quote refreshes do not reload browser state; only the allowed symbol set matters.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [instrumentSignature])

  useEffect(() => () => { request.current?.controller.abort(); request.current = null }, [])

  async function calculate(config: WorkspaceConfig, background = false) {
    if (background && request.current) return
    if (!validateConfig(config, instruments)) { setError('请检查标的、公式、事件与参数；持有期1—60日，费用0—200基点。'); return }
    request.current?.controller.abort()
    const controller = new AbortController()
    const id = ++sequence.current
    request.current = { controller, id }
    setBusy(true)
    setError('')
    if (!background) { submittedRef.current = config; setSubmitted(config) }
    const timeout = window.setTimeout(() => controller.abort(), 25_000)
    try {
      const response = await fetch('/api/v1/market/workspace', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(config), signal: controller.signal })
      const envelope = await response.json() as { success?: boolean; data?: WorkspaceResult; error?: { message?: string } }
      if (!response.ok || !envelope.success || !envelope.data) throw new Error(envelope.error?.message || `计算失败（${response.status}）`)
      if (!Array.isArray(envelope.data.rows) || !envelope.data.study || !Array.isArray(envelope.data.sources)) throw new Error('数据服务返回的公式结果格式异常')
      if (request.current?.id === id) setResult({ data: envelope.data, config })
    } catch (caught) {
      if (request.current?.id === id) setError(controller.signal.aborted ? '计算超时，请重试' : caught instanceof Error ? caught.message : '无法获取计算结果')
    } finally {
      window.clearTimeout(timeout)
      if (request.current?.id === id) { request.current = null; setBusy(false) }
    }
  }

  useEffect(() => {
    if (!automatic || !submitted) return
    const timer = window.setInterval(() => {
      if (document.visibilityState !== 'visible' || !navigator.onLine || request.current) return
      const config = submittedRef.current
      if (config) void calculate(config, true)
    }, researchRefreshInterval(phase))
    return () => window.clearInterval(timer)
    // The timer reads the last explicitly submitted configuration, never an in-progress edit.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [automatic, submitted, phase])

  function persist(next: SavedWorkspace[]) {
    try { localStorage.setItem(STORAGE_KEY, JSON.stringify({ version: 1, workspaces: next })); setSaved(next); return true }
    catch { setNotice('保存失败：本地存储不可用或空间不足。可导出JSON备份。'); return false }
  }
  function save(update: boolean) {
    if (!validateConfig(draft, instruments)) { setNotice('当前配置无效，请补全配置后保存。'); return }
    if (update && (loadedId !== selectedId || !saved.some(item => item.id === selectedId))) { setNotice('未选中有效方案，请先载入或保存为新方案。'); return }
    if (!update && saved.length >= 30) { setNotice('最多保存30个方案，请先删除不需要的方案。'); return }
    const id = update ? selectedId : crypto.randomUUID()
    const item = { id, config: structuredClone(draft) }
    const next = update ? saved.map(entry => entry.id === id ? item : entry) : [...saved, item]
    if (persist(next)) { setSelectedId(id); setLoadedId(id); setNotice(update ? '当前方案已更新，仅保存在此浏览器。' : '新方案已保存，仅保存在此浏览器。') }
  }
  function replaceDraft(config: WorkspaceConfig) {
    request.current?.controller.abort()
    request.current = null
    setBusy(false)
    submittedRef.current = null
    setSubmitted(null)
    setError('')
    setDraft(structuredClone(config))
  }
  function importConfig() {
    try {
      const parsed: unknown = JSON.parse(importText)
      if (!record(parsed) || parsed.version !== 1 || !validateConfig(parsed.config, instruments)) throw new Error()
      replaceDraft(parsed.config)
      setSelectedId('')
      setLoadedId('')
      setNotice('JSON配置已导入到编辑器；尚未计算或保存。')
    } catch { setNotice('导入失败：JSON版本、标的或配置字段无效。') }
  }
  function exportConfig() {
    if (!validateConfig(draft, instruments)) { setNotice('配置无效，无法导出。'); return }
    const json = JSON.stringify({ version: 1, config: draft }, null, 2)
    const url = URL.createObjectURL(new Blob([json], { type: 'application/json' }))
    const anchor = document.createElement('a'); anchor.href = url; anchor.download = 'fund-live-workspace.json'; anchor.click(); window.setTimeout(() => URL.revokeObjectURL(url), 1000)
    setNotice('已导出JSON配置。')
  }
  const dirty = result && JSON.stringify(result.config) !== JSON.stringify(draft)
  const latest = result?.data.rows.at(-1)
  const stale = result?.data.sources.some(source => source.meta.stale) || result?.data.study.history_meta.stale
  const study = result?.data.study

  return <section className="border-t border-[var(--card-border)] py-6" aria-labelledby="market-formula-title">
    <div className="mb-3 flex flex-wrap items-center justify-between gap-3"><h2 id="market-formula-title" className="text-base font-semibold text-theme-primary">自定义公式工作区</h2><span className="text-xs text-theme-muted">日线研究 · 无需登录</span></div>
    <p className="mb-5 text-sm leading-6 text-theme-secondary">选择最多3个数据源，组合公式曲线与事件条件。仅在应用参数后计算；保存的方案只留在当前浏览器。</p>
    <label className="flex flex-col gap-2 text-xs text-theme-secondary">方案名称<input aria-label="方案名称" className={inputClass} value={draft.name} maxLength={60} onChange={event => setDraft({ ...draft, name: event.target.value })} /></label>
    <fieldset className="mt-5 min-w-0"><legend className="mb-3 text-sm font-medium text-theme-primary">数据源</legend>
      <div className="space-y-3">{draft.sources.map((source, index) => <div key={source.alias} className="grid grid-cols-[30px_1fr] gap-2 sm:grid-cols-[30px_1fr_130px_70px]">
        <span className="self-center font-mono text-sm text-[var(--accent-primary)]">{source.alias}</span>
        <label className="min-w-0 text-xs text-theme-muted"><span className="sr-only">数据源{source.alias}标的</span><select aria-label={`数据源${source.alias}标的`} className={inputClass} value={source.symbol} onChange={event => setDraft({ ...draft, target_symbol: draft.target_symbol === source.symbol && !draft.sources.some((item, i) => i !== index && item.symbol === source.symbol) ? event.target.value : draft.target_symbol, sources: draft.sources.map((item, i) => i === index ? { ...item, symbol: event.target.value } : item) })}>{instruments.map(item => <option key={item.symbol} value={item.symbol}>{item.name} · {item.symbol}</option>)}</select></label>
        <label className="col-start-2 min-w-0 sm:col-start-auto"><span className="sr-only">数据源{source.alias}字段</span><select aria-label={`数据源${source.alias}字段`} className={inputClass} value={source.field} onChange={event => setDraft({ ...draft, sources: draft.sources.map((item, i) => i === index ? { ...item, field: event.target.value as typeof source.field } : item) })}><option value="close">收盘价</option><option value="open">开盘价</option><option value="high">最高价</option><option value="low">最低价</option><option value="volume">成交量</option></select></label>
        <button type="button" className={`${buttonClass} col-start-2 sm:col-start-auto`} disabled={draft.sources.length === 1} aria-label={`移除数据源${source.alias}`} onClick={() => setDraft({ ...draft, target_symbol: draft.target_symbol === source.symbol && !draft.sources.some((item, i) => i !== index && item.symbol === source.symbol) ? draft.sources.find((_, i) => i !== index)!.symbol : draft.target_symbol, sources: draft.sources.filter((_, i) => i !== index) })}>移除</button>
      </div>)}</div>
      <button type="button" className={`${buttonClass} mt-3`} disabled={draft.sources.length >= 3} onClick={() => { const alias = ALIASES.find(item => !draft.sources.some(source => source.alias === item))!; setDraft({ ...draft, sources: [...draft.sources, { alias, symbol: instruments[0]?.symbol || '', field: 'close' }] }) }}>添加数据源</button>
      <p className="mt-2 text-xs leading-5 text-theme-muted">移除数据源后，请同步调整引用其别名的公式。价格单位取决于标的；成交量沿用上游单位。</p>
    </fieldset>
    <fieldset className="mt-5 min-w-0"><legend className="mb-3 text-sm font-medium text-theme-primary">公式曲线</legend>
      <div className="space-y-3">{draft.curves.map((curve, index) => <div key={curve.id} className="grid gap-2 sm:grid-cols-[140px_1fr_70px]">
        <input aria-label={`曲线${index + 1}名称`} className={inputClass} maxLength={40} value={curve.name} onChange={event => setDraft({ ...draft, curves: draft.curves.map((item, i) => i === index ? { ...item, name: event.target.value } : item) })} />
        <input aria-label={`曲线${index + 1}公式`} className={`${inputClass} font-mono`} maxLength={256} value={curve.expression} onChange={event => setDraft({ ...draft, curves: draft.curves.map((item, i) => i === index ? { ...item, expression: event.target.value } : item) })} />
        <button type="button" className={buttonClass} disabled={draft.curves.length === 1} aria-label={`移除曲线${index + 1}`} onClick={() => setDraft({ ...draft, curves: draft.curves.filter((_, i) => i !== index) })}>移除</button>
      </div>)}</div>
      <button type="button" className={`${buttonClass} mt-3`} disabled={draft.curves.length >= 3} onClick={() => { const id = ['curve1', 'curve2', 'curve3'].find(item => !draft.curves.some(curve => curve.id === item))!; setDraft({ ...draft, curves: [...draft.curves, { id, name: `曲线${draft.curves.length + 1}`, expression: 'A' }] }) }}>添加曲线</button>
    </fieldset>
    <details className="mt-3 text-xs leading-6 text-theme-secondary"><summary className="cursor-pointer">公式语法与单位</summary><p className="mt-2">使用A/B/C、数字、括号和 + - * /。示例：A/MA(A,20)*100；ROC(A,5)-ROC(B,5)。MA(A,20)为20期均值；ROC(A,5)为5期涨跌百分比；REF(A,1)引用上一期。还支持EMA、RSI、STD、MIN、MAX（表达式,周期）及ABS（表达式）；MIN/MAX为滚动极值，STD为总体标准差，RSI为Wilder平滑。周期1—200，REF允许0；函数名大写。服务端校验语法，不支持脚本。</p><p>原始曲线保留公式单位，不能把价格与成交量直接视为同一量纲。基期100只改变显示，事件条件始终使用原公式值；预热不足、缺失数据和除零会保留空值。</p></details>
    <fieldset className="mt-5 min-w-0"><legend className="mb-3 text-sm font-medium text-theme-primary">事件条件</legend>
      <div className="mb-3 grid grid-cols-2 gap-3"><label className="flex flex-col gap-2 text-xs text-theme-secondary">条件组合<select aria-label="条件组合" className={inputClass} value={draft.event.mode} onChange={event => setDraft({ ...draft, event: { ...draft.event, mode: event.target.value as 'all' | 'any' } })}><option value="all">全部满足</option><option value="any">任一满足</option></select></label><label className="flex flex-col gap-2 text-xs text-theme-secondary">研究方向<select aria-label="研究方向" className={inputClass} value={draft.event.direction} onChange={event => setDraft({ ...draft, event: { ...draft.event, direction: event.target.value as 'up' | 'down' } })}><option value="up">观察上涨</option><option value="down">观察下跌</option></select></label></div>
      <div className="space-y-3">{draft.event.conditions.map((condition, index) => <div key={index} className="grid gap-2 sm:grid-cols-[1fr_145px_1fr_70px]">
        <input aria-label={`条件${index + 1}左公式`} className={`${inputClass} font-mono`} maxLength={256} value={condition.left} onChange={event => setDraft({ ...draft, event: { ...draft.event, conditions: draft.event.conditions.map((item, i) => i === index ? { ...item, left: event.target.value } : item) } })} />
        <select aria-label={`条件${index + 1}比较方式`} className={inputClass} value={condition.operator} onChange={event => setDraft({ ...draft, event: { ...draft.event, conditions: draft.event.conditions.map((item, i) => i === index ? { ...item, operator: event.target.value as typeof condition.operator } : item) } })}>{OPERATORS.map(op => <option key={op.value} value={op.value}>{op.label}</option>)}</select>
        <input aria-label={`条件${index + 1}右公式`} className={`${inputClass} font-mono`} maxLength={256} value={condition.right} onChange={event => setDraft({ ...draft, event: { ...draft.event, conditions: draft.event.conditions.map((item, i) => i === index ? { ...item, right: event.target.value } : item) } })} />
        <button type="button" className={buttonClass} disabled={draft.event.conditions.length === 1} aria-label={`移除条件${index + 1}`} onClick={() => setDraft({ ...draft, event: { ...draft.event, conditions: draft.event.conditions.filter((_, i) => i !== index) } })}>移除</button>
      </div>)}</div>
      <button type="button" className={`${buttonClass} mt-3`} disabled={draft.event.conditions.length >= 3} onClick={() => setDraft({ ...draft, event: { ...draft.event, conditions: [...draft.event.conditions, { left: 'A', operator: 'gt', right: 'MA(A,20)' }] } })}>添加条件</button>
      <p className="mt-2 text-xs leading-5 text-theme-muted">连续阈值条件由不满足转为满足时新增事件，持续满足不重复。组合满足时，有效穿越会记录新事件；穿越需相邻完整交易日，缺值或预热不足不触发。上涨方向统计费用后收益，下跌方向只研究规避下跌，不模拟做空。</p>
    </fieldset>
    <div className="mt-5 grid grid-cols-2 gap-3 lg:grid-cols-4">
      <label className="col-span-2 flex min-w-0 flex-col gap-2 text-xs text-theme-secondary lg:col-span-1">收益研究标的<select aria-label="收益研究标的" className={inputClass} value={draft.target_symbol} onChange={event => setDraft({ ...draft, target_symbol: event.target.value })}>{instruments.filter(item => draft.sources.some(source => source.symbol === item.symbol)).map(item => <option key={item.symbol} value={item.symbol}>{item.name}</option>)}</select></label>
      <label className="flex flex-col gap-2 text-xs text-theme-secondary">持有交易日<input aria-label="公式持有期" className={inputClass} type="number" min={1} max={60} value={draft.holding} onChange={event => setDraft({ ...draft, holding: Number(event.target.value) })} /></label>
      <label className="flex flex-col gap-2 text-xs text-theme-secondary">往返总费用（bp）<input aria-label="公式总交易费用" className={inputClass} type="number" min={0} max={200} value={draft.fee_bps} onChange={event => setDraft({ ...draft, fee_bps: Number(event.target.value) })} /></label>
      <label className="flex flex-col gap-2 text-xs text-theme-secondary">显示交易日<select aria-label="公式显示范围" className={inputClass} value={draft.limit} onChange={event => setDraft({ ...draft, limit: Number(event.target.value) })}>{[20, 60, 120, 640].map(limit => <option key={limit} value={limit}>{limit}日</option>)}</select></label>
    </div>
    <div className="mt-4 flex flex-wrap items-center gap-3"><button type="button" disabled={busy || !instruments.length} className={`${buttonClass} border-[var(--accent-primary)] text-[var(--accent-primary)]`} onClick={() => void calculate(structuredClone(draft))}>{busy ? '正在计算…' : '应用参数并计算'}</button><label className="flex items-center gap-2 text-xs text-theme-secondary"><input type="checkbox" checked={automatic} onChange={event => setAutomatic(event.target.checked)} />自动刷新已应用方案（{researchRefreshInterval(phase) / 1000}秒）</label></div>
    {error && <p role="alert" className="mt-3 break-words border-l-2 border-rose-400 px-3 py-2 text-sm text-rose-400">{error}{result ? '；已保留上次成功结果，请勿当作最新计算。' : ''}</p>}
    {dirty && <p role="status" className="mt-3 text-sm text-amber-500">参数已修改，下方仍为“{result.config.name}”的上次计算结果；请应用参数后更新。</p>}
    {result ? <div className="mt-6 space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3"><h3 className="text-sm font-medium text-theme-primary">公式走势 · {result.config.name}</h3><label className="flex items-center gap-2 text-xs text-theme-secondary">显示方式<select aria-label="公式显示方式" className={inputClass} value={normalized ? 'normalized' : 'raw'} onChange={event => setNormalized(event.target.value === 'normalized')}><option value="raw">原始值</option><option value="normalized">共同日期基期100</option></select></label></div>
      <MarketFormulaChart result={result.data} normalized={normalized} curves={result.config.curves} />
      <div className="text-xs leading-5 text-theme-muted">计算时间 {clock(result.data.generated_at)} · 事件圆点及竖线标出原始公式触发日{stale && <span className="ml-2 text-amber-500">数据缓存已过期</span>}</div>
      {result.data.sources.map(source => <p key={source.alias} className="break-words text-xs text-theme-muted">{source.alias} · {source.symbol} · {source.meta.source} · 读取 {clock(source.meta.fetched_at)}{source.meta.stale ? ' · 已过期缓存' : ''}{source.meta.error ? ` · ${source.meta.error}` : ''}</p>)}
      {(result.data.warnings || []).map((warning, index) => <p role="status" key={index} className="break-words text-xs text-amber-500">{warning}</p>)}
      <div className="overflow-x-auto"><table className="w-full min-w-[420px] text-left text-xs"><caption className="mb-3 text-left text-sm font-medium text-theme-primary">最新事件状态</caption><thead className="text-theme-muted"><tr><th className="py-2">交易日</th><th>条件组合</th><th>条件状态</th><th>新增事件</th>{result.config.curves.map(curve => <th key={curve.id}>{curve.name}</th>)}</tr></thead><tbody><tr className="border-t border-[var(--card-border)] text-theme-secondary"><td className="py-3">{latest?.date || '--'}</td><td>{result.config.event.mode === 'all' ? '全部满足' : '任一满足'}</td><td>{latest?.state === true ? '满足' : latest?.state === false ? '未满足' : '数据不足'}</td><td>{latest ? latest.signal ? '已触发' : '无新增触发' : '--'}</td>{result.config.curves.map(curve => <td key={curve.id}>{valueLabel(latest?.values[curve.id])}</td>)}</tr></tbody></table></div>
      {study && <><h3 className="text-sm font-medium text-theme-primary">自定义事件历史研究</h3><div className="grid grid-cols-2 gap-4 border-y border-[var(--card-border)] py-4 md:grid-cols-3">{[{ label: '完整样本', value: String(study.sample_count) }, { label: '待完成样本', value: String(study.pending_count) }, { label: '方向正确率', value: valueLabel(study.direction_win_rate, true) }, { label: '平均费用后收益', value: study.direction === 'down' ? '不适用' : valueLabel(study.mean_net_return_pct, true) }, { label: '标的平均涨跌', value: valueLabel(study.mean_return_pct, true) }, { label: '最差回撤上界', value: valueLabel(study.worst_drawdown_pct, true) }].map(metric => <div key={metric.label}><span className="text-xs text-theme-muted">{metric.label}</span><div className="mt-1 text-lg font-semibold text-theme-primary">{metric.value}</div></div>)}</div><p className="text-xs leading-5 text-theme-muted">{study.holding}个交易日 · 往返费用{study.fee_bps}bp · 完成日截至{study.last_completed_date || '--'} · 同期窗口基准{valueLabel(study.baseline_mean_return_pct, true)}。历史方向正确率不代表未来获利概率；股票ETF买入当日不能卖出。</p><details className="text-xs leading-6 text-theme-secondary"><summary className="cursor-pointer">公式研究方法与限制</summary><p className="mt-2">{study.method}</p></details><div className="overflow-x-auto"><table className="w-full min-w-[640px] text-left text-xs"><caption className="mb-3 text-left text-theme-primary">自定义事件样本明细</caption><thead className="text-theme-muted"><tr>{['信号日', '进入日', '退出日', '标的涨跌', '方向变化', '费用后收益', '回撤上界'].map(label => <th key={label} className="py-2 font-normal">{label}</th>)}</tr></thead><tbody>{study.events.map((event, index) => <tr key={`${event.signal_date}-${index}`} className="border-t border-[var(--card-border)] text-theme-secondary"><td className="py-3">{event.signal_date}</td><td>{event.entry_date}</td><td>{event.exit_date}</td><td>{valueLabel(event.return_pct, true)}</td><td>{valueLabel(event.direction_pct, true)}</td><td>{valueLabel(event.net_return_pct, true)}</td><td>{valueLabel(event.drawdown_pct, true)}</td></tr>)}</tbody></table>{!study.events.length && <p className="py-3 text-sm text-theme-muted">暂无已完成的历史事件样本</p>}</div></>}
    </div> : <p role="status" className="mt-5 text-sm text-theme-muted">应用参数后显示公式走势、事件状态与历史研究。</p>}
    <div className="mt-6 border-t border-[var(--card-border)] pt-5"><h3 className="mb-3 text-sm font-medium text-theme-primary">本地方案</h3><div className="flex flex-wrap gap-2"><select aria-label="已保存方案" className={`${inputClass} sm:w-72`} value={selectedId} onChange={event => setSelectedId(event.target.value)}><option value="">选择本地方案</option>{saved.map(item => <option key={item.id} value={item.id}>{item.config.name}</option>)}</select><button type="button" className={buttonClass} disabled={!selectedId} onClick={() => { const entry = saved.find(item => item.id === selectedId); if (entry) { replaceDraft(entry.config); setLoadedId(entry.id); setNotice('方案已载入；请应用参数重新计算。') } }}>载入方案</button><button type="button" className={buttonClass} onClick={() => save(false)}>保存为新方案</button><button type="button" className={buttonClass} disabled={!selectedId || loadedId !== selectedId} onClick={() => save(true)}>更新当前方案</button><button type="button" className={buttonClass} disabled={!selectedId} onClick={() => { if (persist(saved.filter(item => item.id !== selectedId))) { setSelectedId(''); setLoadedId(''); setNotice('已删除选中的本地方案。') } }}>删除方案</button><button type="button" className={buttonClass} onClick={exportConfig}>导出JSON</button></div><details className="mt-3 text-xs text-theme-secondary"><summary className="cursor-pointer">导入JSON配置</summary><textarea aria-label="导入JSON配置" className={`${inputClass} mt-3 h-32 py-2 font-mono`} maxLength={20000} value={importText} onChange={event => setImportText(event.target.value)} /><button type="button" className={`${buttonClass} mt-2`} onClick={importConfig}>校验并导入</button></details>{notice && <p role="status" className="mt-3 break-words text-xs text-theme-secondary">{notice}</p>}</div>
  </section>
}
