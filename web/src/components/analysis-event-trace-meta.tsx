import { cn } from '@/lib/utils'

export type AnalysisEventTrace = {
  source_name?: string
  source_url?: string
  source_published_at?: string
  source_confidence?: string
  mapping_basis?: string
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

  if (!sourceName && !sourceURL && !sourcePublishedAt && !sourceConfidence && !mappingBasis) {
    return null
  }

  const pillClass = cn(
    'rounded-full border border-[var(--card-border)] bg-[var(--input-bg)]/55 px-2.5 py-1 text-theme-muted',
    dense ? 'text-[10px] leading-4' : 'text-[11px] leading-5'
  )

  return (
    <div className={cn('flex flex-wrap gap-1.5', dense ? 'mt-2' : 'mt-3', className)}>
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
    </div>
  )
}
