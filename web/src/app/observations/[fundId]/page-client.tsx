'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { ArrowLeft, ExternalLink, FileText, Loader2, RefreshCw } from 'lucide-react'
import { AppTopBar } from '@/components/app-top-bar'
import { SiteFooter } from '@/components/site-footer'
import { ActionButton } from '@/components/ui/action-button'
import { Surface } from '@/components/ui/surface'
import { useFundObservation, type FundNotice, type ObservationSource } from '@/hooks/use-fund-observation'
import { cn } from '@/lib/utils'

type NoticeFilter = 'all' | 'fund' | 'holding'

function formatDate(value: string, withTime = false) {
  if (!value) return '未提供'
  const parsed = new Date(value)
  if (!Number.isFinite(parsed.getTime())) return '未提供'
  const date = parsed.toLocaleDateString('sv-SE', { timeZone: 'Asia/Shanghai' })
  return withTime ? `${date} ${parsed.toLocaleTimeString('zh-CN', { timeZone: 'Asia/Shanghai', hour: '2-digit', minute: '2-digit', hour12: false })}` : date
}

function formatReportPeriod(value: string) {
  const quarter = value.match(/^(\d{4})-?Q([1-4])$/i)
  if (quarter) return `${quarter[1]} 年第 ${quarter[2]} 季度`
  const date = formatDate(value)
  return date === '未提供' ? value || date : date
}

function holdingsSourceLabel(value: string) {
  if (value === 'direct_holdings') return '基金公开持仓'
  if (value === 'target_etf_fallback') return '目标 ETF 公开持仓'
  return value || '未提供'
}

function noticeUrl(value: string) {
  try {
    const url = new URL(value)
    return ['https:', 'http:'].includes(url.protocol) ? url.href : undefined
  } catch {
    return undefined
  }
}

function SourceStatus({ source }: { source: ObservationSource }) {
  const labels = { ok: '读取正常', partial: '部分可用', unavailable: '暂时不可用', not_applicable: '不适用' }
  const warning = source.status === 'partial' || source.status === 'unavailable'
  return (
    <li className="py-3 text-sm">
      <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-1">
        <span className="font-medium text-theme-primary">{source.code === 'fund_notices' ? '基金公告来源' : '持仓公告来源'}</span>
        <span className={warning ? 'text-[var(--accent-secondary)]' : 'text-theme-secondary'}>{labels[source.status]}</span>
      </div>
      <p className="mt-1 text-xs leading-6 text-theme-muted">读取时间 {formatDate(source.checked_at, true)}{source.message && ` · ${source.message}`}</p>
    </li>
  )
}

function NoticeItem({ notice }: { notice: FundNotice }) {
  const href = noticeUrl(notice.source_url)
  const scopeLabel = notice.scope === 'holding' ? '持仓公告' : notice.scope === 'target_etf' ? '目标 ETF 公告' : '基金公告'
  return (
    <li className="py-5 first:pt-0 last:pb-0">
      <div className="mb-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs leading-5 text-theme-muted">
        <time dateTime={notice.published_at}>{formatDate(notice.published_at)}</time>
        <span>{scopeLabel}</span>
        {(notice.related_name || notice.related_symbol) && <span>{[notice.related_name, notice.related_symbol].filter(Boolean).join(' · ')}</span>}
      </div>
      {href ? <a href={href} target="_blank" rel="noopener noreferrer" className="inline text-base font-medium leading-7 text-theme-primary underline-offset-4 hover:text-[var(--accent-primary)] hover:underline [overflow-wrap:anywhere]">{notice.title}<ExternalLink aria-hidden="true" className="ml-2 inline h-3.5 w-3.5 align-baseline text-theme-muted" /></a> : <p className="break-words text-base font-medium leading-7 text-theme-primary">{notice.title}</p>}
      <p className="mt-2 text-xs text-theme-muted">来源：{notice.source_name || '未提供'}{!href && ' · 来源链接暂缺'}</p>
    </li>
  )
}

export function FundObservationPageClient({ fundId }: { fundId: string }) {
  const router = useRouter()
  const [filter, setFilter] = useState<NoticeFilter>('all')
  const { data, error, isLoading, isValidating, mutate } = useFundObservation(fundId)
  const observation = data?.observation
  const notices = (observation?.notices ?? []).filter((notice) => filter === 'all' || (filter === 'fund' ? notice.scope !== 'holding' : notice.scope === 'holding'))
  const sources = observation?.sources ?? []
  const relevantSources = sources.filter((source) => filter === 'all' || source.code === (filter === 'holding' ? 'holding_notices' : 'fund_notices'))
  const incomplete = relevantSources.length < (filter === 'all' ? 2 : 1) || relevantSources.some((source) => source.status === 'unavailable' || source.status === 'partial')
  const notApplicable = relevantSources.length > 0 && relevantSources.every((source) => source.status === 'not_applicable')
  const refresh = () => { void mutate() }

  return (
    <div className="min-h-[100dvh]">
      <AppTopBar currentFundId={fundId} onFundSelect={(id) => router.push(`/observations/${encodeURIComponent(id)}`)} />
      <main id="main-content" className="container mx-auto max-w-5xl space-y-6 px-4 py-6 sm:py-8">
        <div className="flex flex-wrap items-center justify-between gap-3 text-sm">
          <Link href="/observations" className="inline-flex min-h-11 items-center gap-2 text-theme-secondary hover:text-theme-primary"><ArrowLeft className="h-4 w-4" />基金观察</Link>
          <Link href={`/?fund=${encodeURIComponent(fundId)}`} className="inline-flex min-h-11 items-center text-[var(--accent-primary)] hover:underline">返回基金详情</Link>
        </div>

        {isLoading && !data && <div role="status" className="flex min-h-48 items-center justify-center gap-3 text-sm text-theme-secondary"><Loader2 className="h-5 w-5 animate-spin motion-reduce:animate-none" />正在读取 {fundId} 的公告与披露资料</div>}

        {error && <div role="alert" className="flex flex-wrap items-center justify-between gap-4 rounded-2xl border border-[var(--card-border)] bg-[var(--input-bg)] px-5 py-4"><div className="min-w-0"><p className="text-sm font-semibold text-theme-primary">{data ? '本次刷新未完成，以下保留上次读取结果' : '观察资料读取失败'}</p><p className="mt-1 break-words text-sm leading-6 text-theme-secondary">{error.message}</p></div><ActionButton onClick={refresh} disabled={isValidating}>重试</ActionButton></div>}

        {data && observation && <>
          <header className="flex flex-wrap items-start justify-between gap-4">
            <div className="min-w-0">
              <p className="mb-2 flex items-center gap-2 text-sm text-[var(--accent-primary)]"><FileText className="h-4 w-4" />公告与披露资料</p>
              <h1 className="break-words text-2xl font-bold leading-tight text-theme-primary sm:text-3xl">{data.fund.name}</h1>
              <p className="mt-3 text-xs leading-6 text-theme-muted">{[fundId, data.fund.type, data.fund.company].filter(Boolean).join(' · ')}</p>
              <p className="mt-1 text-xs leading-6 text-theme-muted">最近读取 {formatDate(observation.checked_at, true)}（北京时间）</p>
            </div>
            <ActionButton onClick={refresh} disabled={isValidating} className="min-h-11"><RefreshCw className={cn('h-4 w-4', isValidating && 'animate-spin motion-reduce:animate-none')} />刷新公告</ActionButton>
          </header>

          <Surface as="section" aria-labelledby="disclosure-heading" padding="sm">
            <h2 id="disclosure-heading" className="text-base font-semibold text-theme-primary">持仓披露范围</h2>
            {observation.holdings_status === 'available' ? <>
              <dl className="mt-4 grid grid-cols-2 gap-x-5 gap-y-5 sm:grid-cols-3">
                <div><dt className="text-xs text-theme-muted">持仓报告期</dt><dd className="mt-1 text-lg font-semibold tabular-nums text-theme-primary">{formatReportPeriod(observation.latest_holding_period)}</dd>{observation.holding_age_days !== null && <p className="mt-1 text-xs text-theme-muted">距今 {observation.holding_age_days} 天</p>}</div>
                <div><dt className="text-xs text-theme-muted">已披露资产占比</dt><dd className="mt-1 text-lg font-semibold tabular-nums text-theme-primary">{observation.disclosed_weight_percent}%</dd></div>
                <div><dt className="text-xs text-theme-muted">已披露持仓</dt><dd className="mt-1 text-lg font-semibold tabular-nums text-theme-primary">{observation.holdings_count} 项</dd></div>
              </dl>
              <p className="mt-4 text-xs leading-6 text-theme-muted">资料来源：<span>{holdingsSourceLabel(observation.holdings_source)}</span></p>
            </> : <p className="mt-3 text-sm leading-6 text-theme-secondary">{observation.holdings_status === 'unavailable' ? '持仓披露资料暂时无法读取，仍可查看可用的基金公告。' : '尚无公开持仓披露，仍可查看基金公告。'}</p>}
          </Surface>

          <section aria-labelledby="notices-heading" className="space-y-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div><h2 id="notices-heading" className="text-lg font-semibold text-theme-primary">近期公告</h2><p className="mt-1 text-xs leading-6 text-theme-muted">近 {observation.window_days} 天 · 按发布日期排列 · 点击标题查看来源页面</p></div>
              <div aria-label="公告范围" className="inline-flex rounded-xl border border-[var(--card-border)] p-1">
                {([['all', '全部'], ['fund', '基金公告'], ['holding', '持仓公告']] as const).map(([value, label]) => <button type="button" key={value} aria-pressed={filter === value} onClick={() => setFilter(value)} className={cn('min-h-11 rounded-lg px-3 text-sm transition-colors', filter === value ? 'bg-[var(--input-bg)] font-medium text-theme-primary' : 'text-theme-muted hover:text-theme-primary')}>{label}</button>)}
              </div>
            </div>
            {incomplete && <div role="status" className="flex flex-wrap items-center justify-between gap-3 border-l-2 border-[var(--accent-secondary)] bg-[var(--input-bg)]/50 px-4 py-3"><div className="text-sm leading-6 text-theme-secondary"><p className="font-medium text-theme-primary">公告暂时无法完整读取</p><p>仅展示已读取的资料，来源恢复后可重试。</p></div><ActionButton size="sm" className="min-h-11" onClick={refresh} disabled={isValidating}>重试</ActionButton></div>}
            {notices.length > 0 ? <ul className="divide-y divide-[var(--card-border)] rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/35 px-4 py-5 sm:px-5">{notices.map((notice) => <NoticeItem key={notice.id} notice={notice} />)}</ul> : !incomplete && <div className="rounded-2xl border border-dashed border-[var(--card-border)] px-5 py-8 text-center"><p className="text-sm font-medium text-theme-primary">{notApplicable ? '当前披露范围没有可关联的公告来源' : `近 ${observation.window_days} 天暂无${filter === 'holding' ? '持仓' : filter === 'fund' ? '基金' : ''}公告`}</p><p className="mt-2 text-xs leading-6 text-theme-muted">{notApplicable ? '待有可关联的公开披露资料后再检查。' : '当前可用来源未返回符合时间范围的公告。'}</p></div>}
          </section>

          <section aria-labelledby="sources-heading" className="border-t border-[var(--card-border)] pt-5">
            <h2 id="sources-heading" className="text-base font-semibold text-theme-primary">来源与资料说明</h2>
            <ul className="mt-2 divide-y divide-[var(--card-border)]">{sources.map((source) => <SourceStatus key={source.code} source={source} />)}</ul>
            {(observation.notes ?? []).length > 0 && <ul className="mt-3 list-disc space-y-1 pl-4 text-xs leading-6 text-theme-muted">{observation.notes.map((note) => <li key={note}>{note}</li>)}</ul>}
          </section>
        </>}
      </main>
      <SiteFooter compact />
    </div>
  )
}
