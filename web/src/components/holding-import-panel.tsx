'use client'

import { useMemo, useState } from 'react'
import { AlertTriangle, ClipboardList, FileSpreadsheet, Upload } from 'lucide-react'
import type {
  CreateHoldingBatchItem,
  HoldingAggregateEntry,
  HoldingBatchCreateResult,
} from '@/hooks/use-user-portfolio'
import { HOLDING_SOURCE_OPTIONS, type HoldingSourceFilter } from '@/lib/holding-sources'
import { cn } from '@/lib/utils'

interface HoldingImportPanelProps {
  recentAggregates: HoldingAggregateEntry[]
  onSelectDraft?: (draft: { fundID: string; amount: string; note: string }) => void
  onImportBatch?: (items: CreateHoldingBatchItem[]) => Promise<HoldingBatchCreateResult | null> | HoldingBatchCreateResult | null | void
}

interface ImportPreviewRow {
  index: number
  fundID: string
  amount: string
  tradeDate: string
  note: string
  source: string
  valid: boolean
  errors: string[]
}

const sampleTemplate = `基金代码,金额,交易日期,备注\n005827,50000,2026-03-30,支付宝迁移\n003095,12000,2026-04-01,定投补仓`

function parseImportText(raw: string, source: HoldingSourceFilter): ImportPreviewRow[] {
  const lines = raw
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
  const dataLines = lines[0]?.includes('基金代码') || lines[0]?.toLowerCase().includes('fund') ? lines.slice(1) : lines

  return dataLines.slice(0, 20).map((line, index) => {
    const parts = line.split(/[,	，]/).map((item) => item.trim())
    const [fundID = '', amount = '', tradeDate = '', note = ''] = parts
    const normalizedAmount = amount.replace(/,/g, '')
    const errors: string[] = []
    if (!/^\d{6}$/.test(fundID)) {
      errors.push('基金代码需要 6 位数字')
    }
    const amountValue = Number.parseFloat(normalizedAmount)
    if (!Number.isFinite(amountValue) || amountValue <= 0) {
      errors.push('金额需要大于 0')
    }
    if (tradeDate && !/^\d{4}-\d{2}-\d{2}$/.test(tradeDate)) {
      errors.push('交易日期建议使用 YYYY-MM-DD')
    }
    if (!tradeDate) {
      errors.push('直接导入需要交易日期')
    }

    return {
      index: index + 1,
      fundID,
      amount: normalizedAmount,
      tradeDate,
      note,
      source,
      valid: errors.length === 0,
      errors,
    }
  })
}

export function HoldingImportPanel({ recentAggregates, onSelectDraft, onImportBatch }: HoldingImportPanelProps) {
  const [rawText, setRawText] = useState('')
  const [source, setSource] = useState<HoldingSourceFilter>('alipay')
  const [showTemplate, setShowTemplate] = useState(false)
  const [isImporting, setIsImporting] = useState(false)
  const [importResult, setImportResult] = useState<HoldingBatchCreateResult | null>(null)
  const previewRows = useMemo(() => parseImportText(rawText, source), [rawText, source])
  const validRows = previewRows.filter((row) => row.valid)
  const invalidRows = previewRows.length - validRows.length
  const commonFunds = recentAggregates.slice(0, 3)
  const canImport = Boolean(onImportBatch && validRows.length > 0 && !isImporting)

  const handleImportBatch = async () => {
    if (!onImportBatch || validRows.length === 0 || isImporting) {
      return
    }

    setIsImporting(true)
    setImportResult(null)
    try {
      const result = await onImportBatch(validRows.map<CreateHoldingBatchItem>((row) => ({
        fund_id: row.fundID,
        amount: row.amount,
        trade_at: row.tradeDate,
        note: row.note,
        source_platform: row.source === 'all' ? undefined : row.source,
      })))
      setImportResult(result ?? null)
    } finally {
      setIsImporting(false)
    }
  }

  return (
    <section className="mb-6 overflow-hidden rounded-[30px] border border-[var(--card-border)] bg-[var(--card-bg)]/84 p-5 glass">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div>
          <div className="inline-flex items-center gap-2 rounded-full border border-violet-400/25 bg-violet-400/10 px-3 py-1 text-[11px] font-medium tracking-[0.2em] text-violet-100">
            <Upload className="h-3.5 w-3.5" />
            批量导入
          </div>
          <h3 className="mt-3 text-2xl font-black text-theme-primary">先预览，再安全导入</h3>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-theme-secondary">
            不抓取支付宝 / 微信；支持 CSV / 粘贴模板预览、错误行提示，可将有效行批量写入，每行仍走和单笔记录相同的校验与流水链路。
          </p>
        </div>
        <button
          type="button"
          onClick={() => {
            setRawText(sampleTemplate)
            setShowTemplate(true)
          }}
          className="rounded-2xl border border-violet-300/24 bg-violet-400/10 px-4 py-3 text-sm font-semibold text-violet-100 transition hover:border-violet-200/45 hover:bg-violet-400/16"
        >
          使用示例模板
        </button>
      </div>

      <div className="mt-5 grid gap-4 xl:grid-cols-[0.9fr_1.1fr]">
        <div className="space-y-3 rounded-[24px] border border-[var(--card-border)] bg-[var(--input-bg)]/48 p-4">
          <label className="grid gap-2 text-xs font-medium text-theme-muted">
            平台模板
            <select
              value={source}
              onChange={(event) => setSource(event.target.value as HoldingSourceFilter)}
              className="rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-3 py-2 text-sm text-theme-primary outline-none focus:border-violet-300/45"
            >
              {HOLDING_SOURCE_OPTIONS.map((option) => (
                <option key={option.value} value={option.value}>{option.label}</option>
              ))}
            </select>
          </label>
          <label className="grid gap-2 text-xs font-medium text-theme-muted">
            CSV / 粘贴内容
            <textarea
              value={rawText}
              onChange={(event) => setRawText(event.target.value)}
              rows={8}
              placeholder="基金代码,金额,交易日期,备注"
              className="resize-none rounded-[20px] border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-sm leading-6 text-theme-primary outline-none transition focus:border-violet-300/45"
            />
          </label>
          <div className="rounded-[20px] border border-violet-300/16 bg-violet-400/8 px-3 py-3 text-xs leading-5 text-theme-secondary">
            <FileSpreadsheet className="mr-2 inline h-3.5 w-3.5 text-violet-100" />
            支持逗号、中文逗号或 Tab 分隔；当前最多预览并导入 20 行，单行失败不会影响其他有效行。
          </div>
        </div>

        <div className="rounded-[24px] border border-[var(--card-border)] bg-[var(--input-bg)]/48 p-4">
          <div className="flex items-start justify-between gap-3">
            <div>
              <div className="text-sm font-semibold text-theme-primary">导入前预览</div>
              <div className="mt-1 text-xs text-theme-muted">
                有效 {validRows.length} 行；错误 {invalidRows} 行
              </div>
            </div>
            <button
              type="button"
              onClick={() => void handleImportBatch()}
              disabled={!canImport}
              className={cn(
                'rounded-2xl border px-3 py-2 text-xs font-semibold transition',
                canImport
                  ? 'border-emerald-300/28 bg-emerald-400/12 text-emerald-100 hover:border-emerald-200/55 hover:bg-emerald-400/18'
                  : invalidRows > 0
                    ? 'cursor-not-allowed border-amber-300/25 bg-amber-400/10 text-amber-100 opacity-70'
                    : 'cursor-not-allowed border-[var(--input-border)] bg-[var(--input-bg)] text-theme-muted opacity-70'
              )}
            >
              {isImporting
                ? '导入中...'
                : validRows.length > 0
                  ? `直接导入 ${validRows.length} 行`
                  : previewRows.length > 0
                    ? '需修正'
                    : '待粘贴'}
            </button>
          </div>

          {importResult && (
            <div className="mt-4 rounded-[20px] border border-emerald-300/18 bg-emerald-400/10 px-3 py-3 text-xs leading-5 text-theme-secondary">
              <div className="font-semibold text-theme-primary">
                已写入 {importResult.created_count}/{importResult.total} 行
                {importResult.failed_count > 0 ? `，${importResult.failed_count} 行未导入` : ''}
              </div>
              {importResult.failed && importResult.failed.length > 0 && (
                <div className="mt-2 space-y-1">
                  {importResult.failed.slice(0, 5).map((failure) => (
                    <div key={`${failure.index}-${failure.fund_id}-${failure.code}`} className="text-amber-100">
                      第 {failure.index + 1} 行 {failure.fund_id || ''}：{failure.message}
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          <div className="mt-4 space-y-2">
            {previewRows.length === 0 ? (
              <div className="rounded-[22px] border border-dashed border-[var(--card-border)] px-4 py-8 text-center text-sm text-theme-muted">
                粘贴平台导出的基金代码、金额和日期后，会在这里预览并提示错误行。
              </div>
            ) : previewRows.map((row) => (
              <div key={`${row.index}-${row.fundID}-${row.amount}`} className="rounded-[20px] border border-[var(--card-border)] bg-[var(--card-bg)]/58 p-3">
                <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                  <div className="min-w-0">
                    <div className="text-sm font-semibold text-theme-primary">第 {row.index} 行 · {row.fundID || '未识别基金'}</div>
                    <div className="mt-1 text-xs text-theme-muted">金额 {row.amount || '--'} · 日期 {row.tradeDate || '待补'} · {row.note || '无备注'}</div>
                  </div>
                  {row.valid ? (
                    <div className="flex flex-wrap gap-2">
                      <span className="rounded-2xl border border-emerald-300/24 bg-emerald-400/10 px-3 py-2 text-xs font-semibold text-emerald-100">
                        可批量导入
                      </span>
                      <button
                        type="button"
                        onClick={() => onSelectDraft?.({ fundID: row.fundID, amount: row.amount, note: row.note })}
                        className="rounded-2xl border border-cyan-300/24 bg-cyan-400/10 px-3 py-2 text-xs font-semibold text-cyan-100 transition hover:border-cyan-200/45"
                      >
                        带入记录入口
                      </button>
                    </div>
                  ) : (
                    <span className="inline-flex items-center gap-1 rounded-2xl border border-amber-300/22 bg-amber-400/10 px-3 py-2 text-xs text-amber-100">
                      <AlertTriangle className="h-3.5 w-3.5" />
                      {row.errors.join(' / ')}
                    </span>
                  )}
                </div>
              </div>
            ))}
          </div>

          {showTemplate && commonFunds.length > 0 && (
            <div className="mt-4 rounded-[20px] border border-cyan-300/18 bg-cyan-400/8 px-3 py-3 text-xs text-theme-secondary">
              <ClipboardList className="mr-2 inline h-3.5 w-3.5 text-cyan-100" />
              常用基金：{commonFunds.map((item) => item.fund?.name || item.fund_id).join(' / ')}。批量写入前请确认交易日期；不确定的行仍可带入上方逐笔补录。
            </div>
          )}
        </div>
      </div>
    </section>
  )
}
