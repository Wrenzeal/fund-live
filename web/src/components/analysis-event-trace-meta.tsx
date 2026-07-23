import { cn } from '@/lib/utils'

export type AnalysisEventTrace = {
  source_name?: string
  source_url?: string
  source_published_at?: string
  source_confidence?: string
  source_tier?: string
  mapping_basis?: string
  event_status?: string
  expected_at?: string
  announced_at?: string
  effective_at?: string
  known_at?: string
}

function eventStatusLabel(status?: string) {
  switch (status) {
    case 'expected': return '即将发生'
    case 'disclosed': return '已披露'
    case 'active': return '当前生效'
    case 'expired': return '已失效'
    case 'cancelled': return '已取消'
    default: return ''
  }
}

function sourceTierLabel(tier?: string) {
  switch (tier) {
    case 'official': return '官方来源'
    case 'official_aggregator': return '公告聚合'
    case 'secondary': return '补充来源'
    case 'heuristic': return '规则推演'
    default: return ''
  }
}

function formatTraceTime(value?: string) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

interface AnalysisEventTraceMetaProps {
  trace: AnalysisEventTrace
  dense?: boolean
  className?: string
}

function compactTraceText(value: string, maxLength = 84) {
  if (value.length <= maxLength) {
    return value
  }
  return `${value.slice(0, maxLength)}…`
}

function sourceConfidenceLabel(level?: string) {
  switch (level) {
    case 'high':
      return '高可信'
    case 'medium':
      return '中可信'
    case 'low':
      return '低可信'
    default:
      return ''
  }
}

export function AnalysisEventTraceMeta({ trace, dense = false, className }: AnalysisEventTraceMetaProps) {
  const sourceName = trace.source_name
  const sourceURL = trace.source_url
  const sourcePublishedAt = trace.source_published_at
  const sourceConfidence = trace.source_confidence
  const mappingBasis = trace.mapping_basis
  const eventStatus = eventStatusLabel(trace.event_status)
  const sourceTier = sourceTierLabel(trace.source_tier)
  const expectedAt = formatTraceTime(trace.expected_at)
  const effectiveAt = formatTraceTime(trace.effective_at)
  const knownAt = formatTraceTime(trace.known_at)

  if (!sourceName && !sourceURL && !sourcePublishedAt && !sourceConfidence && !mappingBasis && !eventStatus && !knownAt) {
    return null
  }

  const pillClass = cn(
    'rounded-full border border-[var(--card-border)] bg-[var(--input-bg)]/55 px-2.5 py-1 text-theme-muted',
    dense ? 'text-[10px] leading-4' : 'text-[11px] leading-5'
  )

  return (
    <div className={cn('flex flex-wrap gap-1.5', dense ? 'mt-2' : 'mt-3', className)}>
      {eventStatus && <span className={pillClass}>{eventStatus}</span>}
      {expectedAt && <span className={pillClass}>预计：{expectedAt}</span>}
      {effectiveAt && <span className={pillClass}>生效：{effectiveAt}</span>}
      {knownAt && <span className={pillClass}>可知：{knownAt}</span>}
      {mappingBasis && (
        <span className={pillClass}>映射：{compactTraceText(mappingBasis, dense ? 38 : 74)}</span>
      )}
      {sourceURL ? (
        <a
          href={sourceURL}
          target="_blank"
          rel="noreferrer"
          className={cn(pillClass, 'border-cyan-500/20 bg-cyan-500/10 text-cyan-100 transition-colors hover:border-cyan-300/40 hover:text-cyan-50')}
        >
          来源：{sourceName || '事件源'}
        </a>
      ) : sourceName ? (
        <span className={pillClass}>来源：{sourceName}</span>
      ) : null}
      {sourcePublishedAt && (
        <span className={pillClass}>发布：{sourcePublishedAt}</span>
      )}
      {sourceConfidence && (
        <span className={pillClass}>{sourceConfidenceLabel(sourceConfidence) || sourceConfidence}</span>
      )}
      {sourceTier && <span className={pillClass}>{sourceTier}</span>}
    </div>
  )
}
