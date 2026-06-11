'use client'

import { useEffect, useState } from 'react'
import { Layers3, Pencil, Save, X } from 'lucide-react'
import {
  fetchFundClassificationOptions,
  updateFundClassificationOverride,
  type Fund,
  type FundClassificationOption,
  type FundClassificationOverride,
  type FundSectorSnapshot,
  type FundThemeSnapshot,
} from '@/hooks/use-fund-data'
import { useCurrentUser } from '@/hooks/use-auth'
import { confidenceLevelLabel } from '@/lib/fund-analysis-display'

interface FundSectorCardProps {
  fund?: Fund
  sectorSnapshot?: FundSectorSnapshot
  themeSnapshot?: FundThemeSnapshot
  classificationOverride?: FundClassificationOverride
  onClassificationOverrideUpdated?: () => void
}

interface ClassificationModuleProps {
  title: string
  badge: string
  primaryLabel: string
  primaryValue: string
  tone: 'cyan' | 'fuchsia'
  isManual?: boolean
  items: Array<{
    code: string
    name: string
    weight: string
    rank: number
  }>
}

interface ClassificationFormState {
  category_code: string
  primary_sector_code: string
  primary_theme_code: string
  manual_tags_text: string
  note: string
}

function formatWeight(value?: string) {
  const parsed = Number.parseFloat(value || '')
  if (Number.isNaN(parsed)) {
    return '--'
  }
  return `${parsed.toFixed(1)}%`
}

function hasOverride(override?: FundClassificationOverride) {
  return Boolean(
    override?.category_code
    || override?.primary_sector_code
    || override?.primary_theme_code
    || (override?.manual_tags?.length ?? 0) > 0
    || override?.note
  )
}

function splitTags(value: string) {
  return value
    .split(/[，,;；\n]/)
    .map((item) => item.trim())
    .filter(Boolean)
}

function buildFormState(override?: FundClassificationOverride): ClassificationFormState {
  return {
    category_code: override?.category_code || '',
    primary_sector_code: override?.primary_sector_code || '',
    primary_theme_code: override?.primary_theme_code || '',
    manual_tags_text: override?.manual_tags?.join('，') || '',
    note: override?.note || '',
  }
}

function optionLabel(options: FundClassificationOption[], code?: string) {
  if (!code) return ''
  const option = options.find((item) => item.code === code)
  return option?.name || code
}

function ClassificationModule({
  title,
  badge,
  primaryLabel,
  primaryValue,
  tone,
  isManual,
  items,
}: ClassificationModuleProps) {
  const toneStyles = tone === 'cyan'
    ? {
        shell: 'border-cyan-500/20 bg-cyan-500/10',
        value: 'text-cyan-200',
      }
    : {
        shell: 'border-fuchsia-500/20 bg-fuchsia-500/10',
        value: 'text-fuchsia-200',
      }

  return (
    <section className="flex h-full flex-col rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/45 p-4">
      <div className="mb-4 flex items-center justify-between gap-3">
        <div>
          <div className="text-sm font-semibold text-theme-primary">{title}</div>
          <div className="mt-1 text-xs text-theme-muted">{badge}</div>
        </div>
      </div>

      <div className={`mb-4 rounded-2xl border px-4 py-3 ${toneStyles.shell}`}>
        <div className="flex items-center gap-2 text-xs tracking-[0.18em] text-theme-muted">
          {primaryLabel}
          {isManual && (
            <span className="rounded-full border border-emerald-400/25 bg-emerald-400/10 px-2 py-0.5 text-[10px] font-semibold tracking-normal text-emerald-100">
              人工校正
            </span>
          )}
        </div>
        <div className="mt-2 text-xl font-bold text-theme-primary">{primaryValue || '--'}</div>
      </div>

      <div className="flex flex-1 flex-col justify-center space-y-3">
        {items.map((item) => (
          <div
            key={item.code}
            className="flex items-center justify-between rounded-xl border border-[var(--card-border)] bg-[var(--input-bg)]/50 px-4 py-3"
          >
            <div className="min-w-0">
              <div className="text-sm font-medium text-theme-primary">{item.name}</div>
              <div className="mt-1 text-xs text-theme-muted">Top {item.rank}</div>
            </div>
            <div className={`text-sm font-semibold ${toneStyles.value}`}>{item.weight}</div>
          </div>
        ))}
      </div>
    </section>
  )
}

function ClassificationSelect({
  label,
  value,
  options,
  onChange,
}: {
  label: string
  value: string
  options: FundClassificationOption[]
  onChange: (value: string) => void
}) {
  return (
    <label className="space-y-1 text-xs text-theme-muted">
      <span>{label}</span>
      <select
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="w-full rounded-xl border border-[var(--card-border)] bg-[var(--input-bg)] px-3 py-2 text-sm text-theme-primary outline-none transition focus:border-cyan-400/60"
      >
        <option value="">不覆盖</option>
        {options.map((option) => (
          <option key={option.code} value={option.code}>
            {option.name} · {option.code}
          </option>
        ))}
      </select>
    </label>
  )
}

export function FundSectorCard({
  fund,
  sectorSnapshot,
  themeSnapshot,
  classificationOverride,
  onClassificationOverrideUpdated,
}: FundSectorCardProps) {
  const { user } = useCurrentUser()
  const [isEditing, setIsEditing] = useState(false)
  const [options, setOptions] = useState<{ categories: FundClassificationOption[]; sectors: FundClassificationOption[]; themes: FundClassificationOption[] }>({
    categories: [],
    sectors: [],
    themes: [],
  })
  const [formState, setFormState] = useState<ClassificationFormState>(() => buildFormState(classificationOverride))
  const [isSaving, setIsSaving] = useState(false)
  const [formError, setFormError] = useState('')

  useEffect(() => {
    setFormState(buildFormState(classificationOverride))
  }, [classificationOverride])

  useEffect(() => {
    if (!isEditing || !user?.is_admin) {
      return
    }
    let cancelled = false
    fetchFundClassificationOptions()
      .then((data) => {
        if (!cancelled) {
          setOptions({
            categories: data.categories || [],
            sectors: data.sectors || [],
            themes: data.themes || [],
          })
        }
      })
      .catch((error) => {
        if (!cancelled) {
          setFormError(error instanceof Error ? error.message : '分类字典加载失败')
        }
      })
    return () => {
      cancelled = true
    }
  }, [isEditing, user?.is_admin])

  const hasManualOverride = hasOverride(classificationOverride)
  const effectiveCategoryName = classificationOverride?.category_name || fund?.category_name || ''
  const effectiveSectorName = classificationOverride?.primary_sector_name || sectorSnapshot?.primary_sector_name || ''
  const effectiveThemeName = classificationOverride?.primary_theme_name || themeSnapshot?.primary_theme_name || ''
  const manualTags = classificationOverride?.manual_tags || []

  const canRenderClassification = Boolean(
    (sectorSnapshot && sectorSnapshot.breakdown.length > 0)
    || (themeSnapshot && themeSnapshot.breakdown.length > 0)
    || hasManualOverride
  )
  if (!canRenderClassification) {
    return null
  }

  const snapshotDate = sectorSnapshot?.as_of_date || themeSnapshot?.as_of_date || '--'
  const confidenceLabels = { high: '识别覆盖较高', medium: '识别覆盖一般', low: '识别覆盖有限', unknown: '' }
  const sectorConfidence = confidenceLevelLabel(sectorSnapshot?.confidence, confidenceLabels)
  const themeConfidence = confidenceLevelLabel(themeSnapshot?.confidence, confidenceLabels)
  const showCoverageHint = sectorSnapshot?.confidence === 'low' || themeSnapshot?.confidence === 'low'

  const modules = [
    sectorSnapshot && sectorSnapshot.breakdown.length > 0
      ? (
        <ClassificationModule
          key="sector"
          title="行业板块"
          badge="下方权重按持仓自动聚合"
          primaryLabel="主板块"
          primaryValue={effectiveSectorName}
          tone="cyan"
          isManual={Boolean(classificationOverride?.primary_sector_code)}
          items={sectorSnapshot.breakdown.map((item) => ({
            code: item.sector_code,
            name: item.sector_name,
            weight: formatWeight(item.weight_percent),
            rank: item.rank,
          }))}
        />
      )
      : null,
    themeSnapshot && themeSnapshot.breakdown.length > 0
      ? (
        <ClassificationModule
          key="theme"
          title="主题分类"
          badge="下方权重按持仓自动聚合"
          primaryLabel="主主题"
          primaryValue={effectiveThemeName}
          tone="fuchsia"
          isManual={Boolean(classificationOverride?.primary_theme_code)}
          items={themeSnapshot.breakdown.map((item) => ({
            code: item.theme_code,
            name: item.theme_name,
            weight: formatWeight(item.weight_percent),
            rank: item.rank,
          }))}
        />
      )
      : null,
  ].filter(Boolean)

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

  const categoryPreview = formState.category_code
    ? optionLabel(options.categories, formState.category_code)
    : effectiveCategoryName

  return (
    <div className="glass flex h-full flex-col rounded-2xl p-6">
      <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="flex items-center gap-3">
          <div className="rounded-lg bg-cyan-500/20 p-2">
            <Layers3 className="h-5 w-5 text-cyan-300" />
          </div>
          <div>
            <div className="flex items-center gap-2 text-sm font-semibold text-theme-primary">
              持仓分类
              {hasManualOverride && (
                <span className="rounded-full border border-emerald-400/25 bg-emerald-400/10 px-2 py-0.5 text-[10px] text-emerald-100">
                  人工校正
                </span>
              )}
            </div>
            <div className="text-xs text-theme-muted">
              快照日期：{snapshotDate}
            </div>
          </div>
        </div>

        {user?.is_admin && fund?.id && (
          <button
            type="button"
            onClick={() => {
              setIsEditing((value) => !value)
              setFormError('')
            }}
            className="inline-flex items-center justify-center gap-2 rounded-full border border-[var(--card-border)] bg-[var(--input-bg)]/60 px-3 py-1.5 text-xs font-medium text-theme-secondary transition hover:border-cyan-400/40 hover:text-theme-primary"
          >
            {isEditing ? <X className="h-3.5 w-3.5" /> : <Pencil className="h-3.5 w-3.5" />}
            {isEditing ? '收起编辑' : '编辑人工标签'}
          </button>
        )}
      </div>

      {(effectiveCategoryName || manualTags.length > 0) && (
        <div className="mb-4 flex flex-wrap items-center gap-2 text-xs text-theme-secondary">
          {effectiveCategoryName && (
            <span className="inline-flex items-center gap-2 rounded-full border border-[var(--card-border)] bg-[var(--input-bg)]/60 px-3 py-1">
              主分类：<span className="font-medium text-theme-primary">{effectiveCategoryName}</span>
            </span>
          )}
          {manualTags.map((tag) => (
            <span key={tag} className="rounded-full border border-cyan-400/20 bg-cyan-400/10 px-3 py-1 text-cyan-100">
              {tag}
            </span>
          ))}
        </div>
      )}

      {hasManualOverride && (
        <div className="mb-4 rounded-2xl border border-emerald-400/20 bg-emerald-400/10 px-4 py-3 text-sm text-emerald-50">
          主分类 / 主板块 / 主主题可来自人工校正；下方 Top 权重仍保留自动持仓聚合结果，避免把人工标签误读为真实持仓占比。
          {classificationOverride?.note && (
            <div className="mt-2 text-xs text-emerald-100/80">备注：{classificationOverride.note}</div>
          )}
        </div>
      )}

      {isEditing && user?.is_admin && (
        <div className="mb-4 space-y-4 rounded-2xl border border-cyan-400/20 bg-cyan-400/10 p-4">
          <div className="grid gap-3 md:grid-cols-3">
            <ClassificationSelect
              label="人工主分类"
              value={formState.category_code}
              options={options.categories}
              onChange={(value) => setFormState((current) => ({ ...current, category_code: value }))}
            />
            <ClassificationSelect
              label="人工主板块"
              value={formState.primary_sector_code}
              options={options.sectors}
              onChange={(value) => setFormState((current) => ({ ...current, primary_sector_code: value }))}
            />
            <ClassificationSelect
              label="人工主主题"
              value={formState.primary_theme_code}
              options={options.themes}
              onChange={(value) => setFormState((current) => ({ ...current, primary_theme_code: value }))}
            />
          </div>

          <label className="block space-y-1 text-xs text-theme-muted">
            <span>人工标签（逗号分隔）</span>
            <input
              value={formState.manual_tags_text}
              onChange={(event) => setFormState((current) => ({ ...current, manual_tags_text: event.target.value }))}
              placeholder="例如：AI算力，港股互联网，红利防御"
              className="w-full rounded-xl border border-[var(--card-border)] bg-[var(--input-bg)] px-3 py-2 text-sm text-theme-primary outline-none transition focus:border-cyan-400/60"
            />
          </label>

          <label className="block space-y-1 text-xs text-theme-muted">
            <span>备注</span>
            <textarea
              value={formState.note}
              onChange={(event) => setFormState((current) => ({ ...current, note: event.target.value }))}
              rows={2}
              placeholder="记录人工校正原因，便于日后复核。"
              className="w-full resize-none rounded-xl border border-[var(--card-border)] bg-[var(--input-bg)] px-3 py-2 text-sm text-theme-primary outline-none transition focus:border-cyan-400/60"
            />
          </label>

          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="text-xs text-theme-muted">
              当前预览：{categoryPreview || '未覆盖'} · {optionLabel(options.sectors, formState.primary_sector_code) || effectiveSectorName || '未覆盖主板块'} · {optionLabel(options.themes, formState.primary_theme_code) || effectiveThemeName || '未覆盖主主题'}
            </div>
            <button
              type="button"
              onClick={handleSave}
              disabled={isSaving}
              className="inline-flex items-center justify-center gap-2 rounded-full bg-cyan-400 px-4 py-2 text-sm font-bold text-slate-950 transition hover:bg-cyan-300 disabled:cursor-not-allowed disabled:opacity-60"
            >
              <Save className="h-4 w-4" />
              {isSaving ? '保存中...' : '保存人工分类'}
            </button>
          </div>
          {formError && (
            <div className="rounded-xl border border-red-400/20 bg-red-400/10 px-3 py-2 text-xs text-red-100">
              {formError}
            </div>
          )}
        </div>
      )}

      {showCoverageHint && (
        <div className="mb-4 rounded-2xl border border-amber-500/20 bg-amber-500/10 px-4 py-3 text-sm text-amber-100">
          当前分类只覆盖了部分持仓，未归类部分仍然较高；结果更适合作为参考，不宜当作绝对结论。
        </div>
      )}

      {modules.length > 0 && (
        <div className={`grid flex-1 items-stretch gap-4 ${modules.length > 1 ? 'xl:grid-cols-2' : ''}`}>
          {modules}
        </div>
      )}

      {(sectorConfidence || themeConfidence) && (
        <div className="mt-4 flex flex-wrap gap-2 text-xs text-theme-muted">
          {sectorConfidence && <span>行业板块：{sectorConfidence}</span>}
          {themeConfidence && <span>主题分类：{themeConfidence}</span>}
        </div>
      )}
    </div>
  )
}
