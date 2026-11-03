"use client";

import { useEffect, useId, useRef, useState } from 'react'
import { Pencil, Save, X } from 'lucide-react'
import { ActionButton } from '@/components/ui/action-button'
import {
  fetchFundClassificationOptions, updateFundClassificationOverride,
  type Fund, type FundClassificationOption, type FundClassificationOverride,
  type FundSectorSnapshot, type FundThemeSnapshot,
} from '@/hooks/use-fund-data'
import { useCurrentUser } from '@/hooks/use-auth'
import { coverageLevelLabel } from '@/lib/fund-exposure-display'
import { cn } from '@/lib/utils'

interface FundSectorCardProps {
  fund?: Fund
  sectorSnapshot?: FundSectorSnapshot
  themeSnapshot?: FundThemeSnapshot
  classificationOverride?: FundClassificationOverride
  onClassificationOverrideUpdated?: () => void
  isLoading?: boolean
  error?: boolean
  isPaused?: boolean
  onRetry?: () => void
}
interface ClassificationFormState { category_code: string; primary_sector_code: string; primary_theme_code: string; manual_tags_text: string; note: string }
function buildFormState(override?: FundClassificationOverride): ClassificationFormState {
  return { category_code: override?.category_code || '', primary_sector_code: override?.primary_sector_code || '', primary_theme_code: override?.primary_theme_code || '', manual_tags_text: override?.manual_tags?.join('，') || '', note: override?.note || '' }
}
function splitTags(value: string) { return value.split(/[，,;；\n]/).map(item => item.trim()).filter(Boolean) }
function hasOverride(override?: FundClassificationOverride) { return Boolean(override?.category_code || override?.primary_sector_code || override?.primary_theme_code || override?.manual_tags?.length || override?.note) }
function sourceLabel(source?: string) {
  switch (source) {
    case 'target_etf_fallback': return '目标 ETF 披露口径'
    case 'qdii_holdings': return 'QDII 披露持仓口径'
    case 'direct_holdings': case 'holdings': return '基金披露持仓口径'
    default: return '来源口径待确认'
  }
}
function weight(value: string) { const n = Number(value); return value.trim() && Number.isFinite(n) && n >= 0 ? n : null }
function formatWeight(value: string) { const n = weight(value); return n === null ? '--' : n.toFixed(1) + '%' }
const inputClass = 'min-h-11 w-full rounded-lg border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2 text-sm text-theme-primary'
function ClassificationSelect({ label, value, options, onChange }: { label: string; value: string; options: FundClassificationOption[]; onChange: (value: string) => void }) {
  return <label className="space-y-2 text-sm text-theme-secondary"><span>{label}</span><select className={inputClass} value={value} onChange={event => onChange(event.target.value)}><option value="">不覆盖</option>{options.map(option => <option key={option.code} value={option.code}>{option.name}</option>)}</select></label>
}
const tabs = [{ id: 'industry', label: '行业分布' }, { id: 'theme', label: '相关主题' }, { id: 'manual', label: '人工标注' }] as const
type View = typeof tabs[number]['id']

export function FundSectorCard({ fund, sectorSnapshot, themeSnapshot, classificationOverride, onClassificationOverrideUpdated, isLoading = false, error = false, isPaused = false, onRetry }: FundSectorCardProps) {
  const { user } = useCurrentUser()
  const id = useId()
  const [view, setView] = useState<View>('industry')
  const tabRefs = useRef<Array<HTMLButtonElement | null>>([])
  const [isEditing, setIsEditing] = useState(false)
  const [options, setOptions] = useState<{ categories: FundClassificationOption[]; sectors: FundClassificationOption[]; themes: FundClassificationOption[] }>({ categories: [], sectors: [], themes: [] })
  const [formState, setFormState] = useState(() => buildFormState(classificationOverride))
  const [isSaving, setIsSaving] = useState(false)
  const [formError, setFormError] = useState('')
  const [optionsLoading, setOptionsLoading] = useState(false)
  useEffect(() => { if (!isEditing) setFormState(buildFormState(classificationOverride)) }, [classificationOverride, isEditing])
  useEffect(() => {
    if (!isEditing || !user?.is_admin) return
    let cancelled = false
    setOptionsLoading(true)
    fetchFundClassificationOptions().then(data => { if (!cancelled) setOptions({ categories: data.categories || [], sectors: data.sectors || [], themes: data.themes || [] }) }).catch(cause => { if (!cancelled) setFormError(cause instanceof Error ? cause.message : '分类字典加载失败') }).finally(() => { if (!cancelled) setOptionsLoading(false) })
    return () => { cancelled = true }
  }, [isEditing, user?.is_admin])
  const industries = sectorSnapshot?.breakdown || []
  const themes = themeSnapshot?.breakdown || []
  const validIndustries = industries.filter(item => weight(item.weight_percent) !== null)
  const shownWeight = validIndustries.length === industries.length && industries.length > 0 ? validIndustries.reduce((sum, item) => sum + weight(item.weight_percent)!, 0) : null
  // Snapshot primary labels can be overwritten by administrators; the actual breakdown remains automatic.
  const automaticSector = industries.find(item => item.sector_code !== 'other_equity')?.sector_name || industries[0]?.sector_name
  const manual = hasOverride(classificationOverride)
  const handleSave = async () => {
    if (!fund?.id || isSaving) return
    setIsSaving(true)
    setFormError('')
    try {
      await updateFundClassificationOverride(fund.id, {
        category_code: formState.category_code,
        primary_sector_code: formState.primary_sector_code,
        primary_theme_code: formState.primary_theme_code,
        manual_tags: splitTags(formState.manual_tags_text),
        note: formState.note.trim(),
      })
      setIsEditing(false)
      onClassificationOverrideUpdated?.()
    } catch (error) {
      setFormError(error instanceof Error ? error.message : '保存失败')
    } finally {
      setIsSaving(false)
    }
  }

  return <section className="fund-structure rounded-xl border border-[var(--card-border)] bg-[var(--card-bg)] p-5 sm:p-6" aria-labelledby={id + '-title'}>
    <header className="flex flex-wrap items-start justify-between gap-3">
      <div><h2 id={id + '-title'} className="text-lg font-semibold text-theme-primary">基金持仓结构</h2><p className="mt-1 text-sm text-theme-secondary">依据已披露持仓，了解基金主要投向哪里。</p></div>
      <details className="max-w-2xl text-sm text-theme-secondary"><summary className="flex min-h-11 cursor-pointer items-center text-accent">查看分类依据</summary><p className="py-3 leading-7">行业权重来自已披露持仓的分类映射，仅显示返回的条目。主题按映射规则分摊，不是官方主题仓位。人工标注独立于自动权重；不同数据源的比例不能直接相加。</p></details>
    </header>
    <div role="tablist" aria-label="持仓结构视角" className="mt-4 flex gap-5 border-b border-[var(--card-border)]">
      {tabs.map((tab, index) => <button key={tab.id} ref={node => { tabRefs.current[index] = node }} type="button" role="tab" id={id + '-' + tab.id} aria-selected={view === tab.id} aria-controls={id + '-panel-' + tab.id} tabIndex={view === tab.id ? 0 : -1} onClick={() => setView(tab.id)} onKeyDown={event => {
        let next = index
        if (event.key === 'ArrowRight') next = (index + 1) % tabs.length
        else if (event.key === 'ArrowLeft') next = (index + tabs.length - 1) % tabs.length
        else if (event.key === 'Home') next = 0
        else if (event.key === 'End') next = tabs.length - 1
        else return
        event.preventDefault(); setView(tabs[next].id); tabRefs.current[next]?.focus()
      }} className={cn('min-h-11 border-b-2 px-0 pb-3 pt-2 text-sm', view === tab.id ? 'border-[var(--accent-primary)] font-semibold text-theme-primary' : 'border-transparent text-theme-secondary')}>{tab.label}</button>)}
    </div>
    {error ? <div role="alert" className="py-7 text-sm text-theme-secondary">持仓结构暂时无法读取{onRetry && <ActionButton className="ml-3" onClick={onRetry}>重新加载</ActionButton>}</div> : isLoading ? <div role="status" className="py-7 text-sm text-theme-secondary">正在读取持仓结构…</div> : <>
      {view === 'industry' && <div role="tabpanel" id={id + '-panel-industry'} aria-labelledby={id + '-industry'} tabIndex={0} className="mt-5 grid gap-6 lg:grid-cols-[minmax(0,1fr)_220px]">
        <div>{industries.length ? <><div className="mb-5 flex justify-between gap-3 text-xs text-theme-secondary"><span>按披露数据自动归类 · 仅列返回条目</span><span>原始披露权重</span></div>{industries.map((item, index) => <div key={item.sector_code} className="mb-4 grid grid-cols-[minmax(72px,110px)_minmax(0,1fr)_52px] items-center gap-3 text-sm"><span className="break-words text-theme-primary">{item.sector_name}</span><span className="h-2 rounded-sm bg-[var(--surface-subtle)]"><span data-testid="industry-bar" className="block h-full rounded-sm bg-[var(--accent-primary)]" style={{ width: Math.max(0, Math.min(100, weight(item.weight_percent) ?? 0)) + '%', opacity: Math.max(.35, 1 - index * .22) }} /></span><span className="text-right font-medium tabular-nums text-theme-primary">{formatWeight(item.weight_percent)}</span></div>)}<p className="mt-5 text-xs leading-6 text-theme-secondary">条形按 0–100% 刻度展示，不将前三项放大为全仓比例。</p></> : <p className="py-4 text-sm text-theme-secondary">{isPaused ? '集合竞价期间暂无已缓存的持仓结构' : '暂无已归类的行业持仓'}</p>}</div>
        <dl className="grid grid-cols-2 gap-4 border-t border-[var(--card-border)] pt-4 text-sm lg:grid-cols-1 lg:border-l lg:border-t-0 lg:pl-5 lg:pt-0"><div><dt className="text-xs text-theme-secondary">行业披露期</dt><dd className="mt-1 text-theme-primary">{sectorSnapshot?.as_of_date || '待确认'}</dd></div><div><dt className="text-xs text-theme-secondary">来源与权重口径</dt><dd className="mt-1 text-theme-primary">{sourceLabel(sectorSnapshot?.source)}</dd></div><div><dt className="text-xs text-theme-secondary">已展示权重合计</dt><dd className="mt-1 tabular-nums text-theme-primary">{shownWeight === null ? '--' : shownWeight.toFixed(1) + '%'}</dd></div><div><dt className="text-xs text-theme-secondary">归类覆盖情况</dt><dd className="mt-1 text-theme-primary">{coverageLevelLabel(sectorSnapshot?.confidence)}</dd></div></dl>
      </div>}
      {view === 'theme' && <div role="tabpanel" id={id + '-panel-theme'} aria-labelledby={id + '-theme'} tabIndex={0} className="py-5">
        <p className="mb-4 text-sm text-theme-secondary">主题快照 {themeSnapshot?.as_of_date || '待确认'} · {sourceLabel(themeSnapshot?.source)}</p>
        {themes.length ? <><div className="flex flex-wrap gap-2">{themes.map(item => <span key={item.theme_code} className="rounded-md bg-[var(--accent-soft)] px-3 py-1.5 text-sm text-theme-primary">{item.theme_name}</span>)}</div><details className="mt-4 text-sm text-theme-secondary"><summary className="min-h-11 cursor-pointer py-3">查看主题权重与规则</summary><p className="mb-3 leading-7">同一股票关联多个主题时，权重会按现有映射规则分摊；这是参考归因，不是基金公司披露的主题仓位，不能与行业比例相加。</p>{themes.map(item => <div key={item.theme_code} className="flex justify-between border-t border-[var(--card-border)] py-3"><span>{item.theme_name}</span><span className="tabular-nums">{formatWeight(item.weight_percent)}</span></div>)}</details></> : <p className="text-sm text-theme-secondary">暂无已归类的相关主题</p>}
      </div>}
      {view === 'manual' && <div role="tabpanel" id={id + '-panel-manual'} aria-labelledby={id + '-manual'} tabIndex={0} className="space-y-5 py-5">
        <dl className="grid gap-5 sm:grid-cols-2"><div><dt className="text-xs text-theme-secondary">按分布识别的主要行业</dt><dd className="mt-2 text-sm text-theme-primary">{automaticSector || '暂无自动归类'}</dd></div><div><dt className="text-xs text-theme-secondary">人工行业标注</dt><dd className="mt-2 text-sm text-theme-primary">{classificationOverride?.primary_sector_name || '未设置'}</dd></div>{classificationOverride?.category_name && <div><dt className="text-xs text-theme-secondary">人工基金分类</dt><dd className="mt-2 text-sm text-theme-primary">{classificationOverride.category_name}</dd></div>}{classificationOverride?.primary_theme_name && <div><dt className="text-xs text-theme-secondary">人工主题标注</dt><dd className="mt-2 text-sm text-theme-primary">{classificationOverride.primary_theme_name}</dd></div>}</dl>
        {!!classificationOverride?.manual_tags?.length && <div className="flex flex-wrap gap-2">{classificationOverride.manual_tags.map(tag => <span key={tag} className="rounded-md border border-[var(--card-border)] px-2 py-1 text-sm text-theme-secondary">{tag}</span>)}</div>}
        {classificationOverride?.note && <p className="break-words text-sm leading-7 text-theme-secondary">{classificationOverride.note}</p>}
        <p className="text-xs leading-6 text-theme-secondary">{manual ? '人工标注仅补充说明，不改变行业分布权重。' : '暂无人工标注。'}{classificationOverride?.updated_at ? ' 更新于 ' + classificationOverride.updated_at.slice(0, 10) : ''}</p>
        {user?.is_admin && fund?.id && <ActionButton onClick={() => { setIsEditing(value => !value); setFormError('') }}><Pencil className="h-4 w-4" />{isEditing ? '收起编辑' : '编辑人工标注'}</ActionButton>}
        {isEditing && user?.is_admin && <div className="space-y-4 border-t border-[var(--card-border)] pt-5">
          {optionsLoading && <p role="status" className="text-sm text-theme-secondary">正在读取分类字典…</p>}
          <div className="grid gap-4 md:grid-cols-3"><ClassificationSelect label="人工主分类" value={formState.category_code} options={options.categories} onChange={value => setFormState(current => ({ ...current, category_code: value }))} /><ClassificationSelect label="人工主板块" value={formState.primary_sector_code} options={options.sectors} onChange={value => setFormState(current => ({ ...current, primary_sector_code: value }))} /><ClassificationSelect label="人工主主题" value={formState.primary_theme_code} options={options.themes} onChange={value => setFormState(current => ({ ...current, primary_theme_code: value }))} /></div>
          <label className="block space-y-2 text-sm text-theme-secondary"><span>人工标签（逗号分隔）</span><input className={inputClass} value={formState.manual_tags_text} onChange={event => setFormState(current => ({ ...current, manual_tags_text: event.target.value }))} /></label>
          <label className="block space-y-2 text-sm text-theme-secondary"><span>备注</span><textarea className={inputClass} rows={2} value={formState.note} onChange={event => setFormState(current => ({ ...current, note: event.target.value }))} /></label>
          {formError && <p role="alert" className="text-sm text-danger">{formError}</p>}
          <div className="flex gap-3"><ActionButton variant="primary" disabled={isSaving || optionsLoading} onClick={() => void handleSave()}><Save className="h-4 w-4" />{isSaving ? '保存中…' : '保存人工标注'}</ActionButton><ActionButton disabled={isSaving} onClick={() => setIsEditing(false)}><X className="h-4 w-4" />取消</ActionButton></div>
        </div>}
      </div>}
    </>}
    <footer className="mt-5 border-t border-[var(--card-border)] pt-4 text-xs leading-6 text-theme-secondary">仅代表已披露并归类的部分，不是实时全量仓位。目标 ETF 来源沿用目标 ETF 披露权重，不能直接视为联接基金全部资产比例。</footer>
  </section>
}
