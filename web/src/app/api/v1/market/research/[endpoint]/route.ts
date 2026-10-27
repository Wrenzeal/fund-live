import type { NextRequest } from 'next/server'
import { marketResearchSources } from '@/lib/market-research-sources'
import { historyAsOf } from '@/lib/market-research-calendar'
import { calculateResearchStudy, calculateResearchTechnical, validateStudyParameters } from '@/lib/market-research-math'
import type { ResearchRule } from '@/hooks/use-market-research'

export const runtime = 'nodejs'
export const dynamic = 'force-dynamic'
export const maxDuration = 30

function response(data: unknown) {
  return Response.json({ success: true, data }, { headers: { 'Cache-Control': 'no-store' } })
}

export async function GET(request: NextRequest, context: { params: Promise<{ endpoint: string }> }) {
  const { endpoint } = await context.params
  const query = request.nextUrl.searchParams
  try {
    if (endpoint === 'overview') return response(await marketResearchSources.overview())
    if (endpoint === 'flows') return response(await marketResearchSources.flows())
    const symbol = (query.get('symbol') ?? 'sz159994').trim()
    if (endpoint === 'instrument') {
      const detail = await marketResearchSources.instrument(symbol)
      return response({ ...detail, technical: calculateResearchTechnical(detail.bars, historyAsOf(detail.history_meta, new Date())) })
    }
    if (endpoint === 'study') {
      const rule = (query.get('rule') ?? 'ma20_up').trim()
      const holdingText = query.get('holding') ?? '10'
      const feeText = query.get('fee_bps') ?? '10'
      if (!holdingText.trim() || !feeText.trim()) throw new Error('持有期与总交易费用不能为空')
      const holding = Number(holdingText)
      const fee = Number(feeText)
      validateStudyParameters(rule, holding, fee)
      const history = await marketResearchSources.history(symbol)
      return response({ ...calculateResearchStudy(history.bars, symbol, rule as ResearchRule, holding, fee, history.asOf), history_meta: history.meta })
    }
    return Response.json({ success: false, error: { code: 'NOT_FOUND', message: '未知研究接口' } }, { status: 404 })
  } catch (error) {
    return Response.json({ success: false, error: { code: 'INVALID_RESEARCH_REQUEST', message: error instanceof Error ? error.message : '研究请求失败' } }, { status: 400 })
  }
}
