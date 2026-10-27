import { marketResearchSources } from '@/lib/market-research-sources'
import { runWorkspace, readWorkspaceBody } from '@/lib/market-workspace-engine'

export const runtime = 'nodejs'
export const dynamic = 'force-dynamic'
export const maxDuration = 30
export async function POST(request: Request) {
  try {
    const input = await readWorkspaceBody(request)
    const data = await runWorkspace(input, symbol => marketResearchSources.history(symbol))
    return Response.json({ success: true, data }, { headers: { 'Cache-Control': 'no-store' } })
  } catch (error) {
    return Response.json({ success: false, error: { code: 'INVALID_WORKSPACE_REQUEST', message: error instanceof Error ? error.message : '工作区研究失败' } }, { status: 400, headers: { 'Cache-Control': 'no-store' } })
  }
}
