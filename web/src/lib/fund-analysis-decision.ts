import type {
  FundAnalysis,
  FundAnalysisEventImpact,
  FundAnalysisEvidenceItem,
  FundAnalysisModuleScore,
} from '@/hooks/use-fund-data'
import {
  confidenceLevelLabel,
  dominantAnalysisRecommendation,
  eventHorizonLabel,
  eventImpactLabel,
  eventStrengthLabel,
  formatAnalysisPercent,
  formatAnalysisScore,
  parseAnalysisNumber,
  riskLevelLabel,
  type FundAnalysisRecommendationCode,
} from '@/lib/fund-analysis-display'

export type FundAnalysisDecisionTone = 'positive' | 'neutral' | 'negative' | 'warning'
export type FundAnalysisDecisionPointRole = 'primary' | 'support' | 'risk' | 'data'

export interface FundAnalysisTraceSource {
  source_name?: string
  source_url?: string
  source_published_at?: string
  source_confidence?: string
  mapping_basis?: string
}

export interface FundAnalysisDecisionPoint {
  id: string
  role: FundAnalysisDecisionPointRole
  tone: FundAnalysisDecisionTone
  title: string
  summary: string
  sourceLabel: string
  metaLabel?: string
  trace?: FundAnalysisTraceSource
}

export interface FundAnalysisDecisionView {
  result: {
    code: FundAnalysisRecommendationCode
    label: string
    summary: string
    percent: number
    percentLabel: string
    tone: FundAnalysisDecisionTone
  }
  scoreLabel: string
  riskLabel: string
  confidenceLabel: string
  basisLabel: string
  holdingPeriodLabel: string
  evidenceCountLabel: string
  sourceBadges: string[]
  mainReasons: FundAnalysisDecisionPoint[]
  riskReasons: FundAnalysisDecisionPoint[]
  methodNotes: string[]
  topSignal?: FundAnalysisDecisionPoint
}

const RESULT_META: Record<FundAnalysisRecommendationCode, { label: string; tone: FundAnalysisDecisionTone }> = {
  increase: { label: '结构偏积极', tone: 'positive' },
  hold: { label: '适合观察', tone: 'neutral' },
  decrease: { label: '风险偏高', tone: 'negative' },
}

export function buildFundAnalysisDecision(analysis?: FundAnalysis | null): FundAnalysisDecisionView | null {
  if (!analysis) {
    return null
  }

  const resultCode = dominantAnalysisRecommendation(analysis)
  const resultMeta = RESULT_META[resultCode]
  const resultPercent = recommendationPercent(analysis, resultCode)
  const mainReasons = buildMainReasons(analysis, resultCode)
  const riskReasons = buildRiskReasons(analysis)
  const methodNotes = [
    ...(analysis.confidence_deductions || []),
    ...(analysis.ai_explanation?.limitations || []),
  ].filter(Boolean).slice(0, 3)

  return {
    result: {
      code: resultCode,
      label: resultMeta.label,
      summary: analysis.summary || fallbackSummary(resultCode),
      percent: resultPercent,
      percentLabel: formatAnalysisPercent(resultPercent),
      tone: resultMeta.tone,
    },
    scoreLabel: formatAnalysisScore(analysis.total_score),
    riskLabel: riskLevelLabel(analysis.risk_level),
    confidenceLabel: confidenceLevelLabel(analysis.confidence),
    basisLabel: analysis.analysis_basis || '分析口径待定',
    holdingPeriodLabel: analysis.latest_holding_period || '披露期待补',
    evidenceCountLabel: `${(analysis.primary_evidence || []).length} 主证据 / ${(analysis.counter_evidence || []).length} 限制`,
    sourceBadges: buildSourceBadges(analysis),
    mainReasons,
    riskReasons,
    methodNotes,
    topSignal: mainReasons[0] || riskReasons[0],
  }
}

function recommendationPercent(analysis: FundAnalysis, code: FundAnalysisRecommendationCode) {
  if (code === 'increase') {
    return parseAnalysisNumber(analysis.increase_percent)
  }
  if (code === 'decrease') {
    return parseAnalysisNumber(analysis.decrease_percent)
  }
  return parseAnalysisNumber(analysis.hold_percent)
}

function fallbackSummary(code: FundAnalysisRecommendationCode) {
  switch (code) {
    case 'increase':
      return '当前结构偏积极，但仍需要结合证据强度分批观察。'
    case 'decrease':
      return '当前风险暴露偏高，先等待风险信号重新收敛。'
    default:
      return '当前更适合观察，等待趋势、事件和风险信号进一步拉开差距。'
  }
}

function buildSourceBadges(analysis: FundAnalysis) {
  return [
    analysis.analysis_basis,
    confidenceLevelLabel(analysis.confidence),
    riskLevelLabel(analysis.risk_level),
    analysis.latest_holding_period ? `披露期 ${analysis.latest_holding_period}` : '',
  ].filter(Boolean).slice(0, 4)
}

function buildMainReasons(analysis: FundAnalysis, resultCode: FundAnalysisRecommendationCode) {
  const points: FundAnalysisDecisionPoint[] = []

  for (const item of analysis.primary_evidence || []) {
    if (points.length >= 3) break
    addUniquePoint(points, pointFromEvidence(item, 'primary'))
  }

  for (const event of analysis.event_impacts || []) {
    if (points.length >= 3) break
    if (!eventSupportsResult(event, resultCode)) continue
    addUniquePoint(points, pointFromEvent(event, 'support'))
  }

  for (const scoreModule of strongestModules(analysis.module_scores || [])) {
    if (points.length >= 3) break
    addUniquePoint(points, pointFromModule(scoreModule))
  }

  for (const reason of analysis.reasons || []) {
    if (points.length >= 3) break
    addUniquePoint(points, pointFromText(reason, 'support', '规则原因'))
  }

  if (points.length === 0) {
    addUniquePoint(points, {
      id: 'summary-fallback',
      role: 'support',
      tone: 'neutral',
      title: '结论等待确认',
      summary: analysis.summary || fallbackSummary(resultCode),
      sourceLabel: '规则汇总',
    })
  }

  return points.slice(0, 3)
}

function buildRiskReasons(analysis: FundAnalysis) {
  const points: FundAnalysisDecisionPoint[] = []

  for (const item of analysis.counter_evidence || []) {
    if (points.length >= 3) break
    addUniquePoint(points, pointFromEvidence(item, 'risk'))
  }

  for (const event of analysis.event_impacts || []) {
    if (points.length >= 3) break
    if (event.impact !== 'negative') continue
    addUniquePoint(points, pointFromEvent(event, 'risk'))
  }

  for (const warning of analysis.warnings || []) {
    if (points.length >= 3) break
    addUniquePoint(points, pointFromText(warning, 'risk', '风险提示'))
  }

  for (const deduction of analysis.confidence_deductions || []) {
    if (points.length >= 3) break
    addUniquePoint(points, pointFromText(deduction, 'data', '数据限制'))
  }

  if (points.length === 0) {
    addUniquePoint(points, {
      id: 'risk-fallback',
      role: 'risk',
      tone: 'warning',
      title: '暂无强反方证据',
      summary: '当前未识别到明显反向事件，但仍需结合自身仓位和数据口径复核。',
      sourceLabel: '风险兜底',
    })
  }

  return points.slice(0, 3)
}

function pointFromEvidence(item: FundAnalysisEvidenceItem, role: FundAnalysisDecisionPointRole): FundAnalysisDecisionPoint {
  return {
    id: `evidence-${item.code || item.title}`,
    role,
    tone: toneFromImpact(item.impact, role),
    title: item.title || evidenceTypeLabel(item.evidence_type),
    summary: item.summary || item.title,
    sourceLabel: sourceScopeLabel(item.source_scope) || evidenceTypeLabel(item.evidence_type),
    metaLabel: [item.strength ? eventStrengthLabel(item.strength) : '', item.horizon ? eventHorizonLabel(item.horizon) : ''].filter(Boolean).join(' / '),
    trace: item,
  }
}

function pointFromEvent(event: FundAnalysisEventImpact, role: FundAnalysisDecisionPointRole): FundAnalysisDecisionPoint {
  return {
    id: `event-${event.code || event.title}`,
    role,
    tone: toneFromImpact(event.impact, role),
    title: event.title || eventScopeLabel(event.target_scope),
    summary: event.summary || event.title,
    sourceLabel: eventScopeLabel(event.target_scope),
    metaLabel: [eventImpactLabel(event.impact), event.strength ? eventStrengthLabel(event.strength) : '', event.horizon ? eventHorizonLabel(event.horizon) : ''].filter(Boolean).join(' / '),
    trace: event,
  }
}

function pointFromModule(scoreModule: FundAnalysisModuleScore): FundAnalysisDecisionPoint {
  const score = parseAnalysisNumber(scoreModule.score)
  return {
    id: `module-${scoreModule.code || scoreModule.name}`,
    role: 'support',
    tone: scoreModule.code === 'risk' || score < 45 ? 'warning' : 'neutral',
    title: `${scoreModule.name || '模块'}得分 ${formatAnalysisScore(scoreModule.score)}`,
    summary: scoreModule.summary || '该模块对当前结论有辅助贡献。',
    sourceLabel: '模块评分',
    metaLabel: '规则模块',
  }
}

function pointFromText(summary: string, role: FundAnalysisDecisionPointRole, sourceLabel: string): FundAnalysisDecisionPoint {
  return {
    id: `${sourceLabel}-${summary}`,
    role,
    tone: role === 'risk' || role === 'data' ? 'warning' : 'neutral',
    title: sourceLabel,
    summary,
    sourceLabel,
  }
}

function strongestModules(scoreModules: FundAnalysisModuleScore[]) {
  return scoreModules
    .filter((scoreModule) => parseAnalysisNumber(scoreModule.score) >= 60)
    .sort((left, right) => parseAnalysisNumber(right.score) - parseAnalysisNumber(left.score))
    .slice(0, 2)
}

function eventSupportsResult(event: FundAnalysisEventImpact, code: FundAnalysisRecommendationCode) {
  if (event.target_scope === 'methodology' || event.target_scope === 'disclosure') {
    return false
  }
  if (code === 'increase') {
    return event.impact === 'positive'
  }
  if (code === 'decrease') {
    return event.impact === 'negative'
  }
  return event.impact !== 'negative'
}

function toneFromImpact(impact: string | undefined, role: FundAnalysisDecisionPointRole): FundAnalysisDecisionTone {
  if (role === 'risk' || role === 'data') {
    return 'warning'
  }
  if (impact === 'positive') {
    return 'positive'
  }
  if (impact === 'negative') {
    return 'negative'
  }
  return 'neutral'
}

function addUniquePoint(points: FundAnalysisDecisionPoint[], point: FundAnalysisDecisionPoint) {
  if (!point.summary && !point.title) {
    return
  }
  const fingerprint = normalizeText(`${point.title}${point.summary}`)
  const exists = points.some((item) => normalizeText(`${item.title}${item.summary}`) === fingerprint)
  if (!exists) {
    points.push(point)
  }
}

function normalizeText(value: string) {
  return value.replace(/\s+/g, '').slice(0, 80)
}

function sourceScopeLabel(scope?: string) {
  switch (scope) {
    case 'holding':
      return '持仓'
    case 'exposure':
      return '暴露'
    case 'macro':
      return '宏观'
    case 'fund':
      return '基金事件'
    case 'index':
      return '指数'
    case 'disclosure':
      return '披露'
    case 'methodology':
      return '方法'
    default:
      return ''
  }
}

function eventScopeLabel(scope?: string) {
  switch (scope) {
    case 'macro':
      return '实时宏观'
    case 'holding':
      return '持仓事件'
    case 'exposure':
      return '主线暴露'
    case 'fund':
      return '基金事件'
    case 'index':
      return '指数事件'
    case 'disclosure':
      return '披露口径'
    case 'methodology':
      return '方法限制'
    default:
      return '事件'
  }
}

function evidenceTypeLabel(type?: string) {
  switch (type) {
    case 'event':
      return '事件证据'
    case 'exposure':
      return '暴露证据'
    case 'market':
      return '行情证据'
    case 'confidence':
      return '可信度证据'
    default:
      return '证据'
  }
}
