import type { CreateHoldingBatchItem } from '@/hooks/use-user-portfolio'
import { buildTradeAtValue, type TradeTiming } from '@/lib/holding-display'
import type { HoldingSourceFilter } from '@/lib/holding-sources'

export type HoldingImportKind = 'buy' | 'opening'
export type HoldingImportBasis = 'shares' | 'value'
export interface HoldingImportRow { index: number; payload: CreateHoldingBatchItem; errors: string[] }

function splitCSV(text: string) {
  const rows: string[][] = []; let row: string[] = [], field = '', quoted = false
  for (let index = 0; index < text.length; index++) {
    const char = text[index]
    if (char === '"') { if (quoted && text[index + 1] === '"') { field += '"'; index++ } else quoted = !quoted }
    else if (!quoted && [',', '\t', '，'].includes(char)) { row.push(field.trim()); field = '' }
    else if (!quoted && (char === '\n' || char === '\r')) { if (char === '\r' && text[index + 1] === '\n') index++; row.push(field.trim()); if (row.some(Boolean)) rows.push(row); row = []; field = '' }
    else field += char
  }
  row.push(field.trim()); if (row.some(Boolean)) rows.push(row)
  return { rows, quoted }
}
const aliases: Record<string, string[]> = {
  fund: ['基金代码', 'fund_id', 'fundcode'], cost: ['剩余成本', '成本', '金额', '买入金额', 'amount'], shares: ['持有份额', '份额', 'shares'],
  market: ['持仓市值', '持仓金额', '市值', 'market_value'], profit: ['持有盈亏', '持有收益', 'holding_profit'],
  snapshot: ['登记日期', '净值日期', 'snapshot_date'], trade: ['交易日期', '交易时间', 'trade_at', 'trade_date'], timing: ['提交时段', 'timing'], note: ['备注', 'note'],
}
function validDate(value: string) { return /^\d{4}-\d{2}-\d{2}$/.test(value) && Number.isFinite(Date.parse(value + 'T00:00:00Z')) && new Date(value + 'T00:00:00Z').toISOString().slice(0, 10) === value }
const clean = (value: string) => value.replace(/,/g, '').trim()
const positive = (value: string) => !!value.trim() && Number.isFinite(Number(value)) && Number(value) > 0

export function parseHoldingImport(raw: string, kind: HoldingImportKind, basis: HoldingImportBasis, source: HoldingSourceFilter, defaultTiming: TradeTiming = 'before_close'): { rows: HoldingImportRow[]; errors: string[] } {
  const parsed = splitCSV(raw.replace(/^\uFEFF/, ''))
  const errors: string[] = []
  if (parsed.quoted) errors.push('CSV 引号未闭合，请检查备注或金额。')
  const first = parsed.rows[0] || []
  const hasHeader = first.some(field => aliases.fund.includes(field.toLowerCase()))
  const header = hasHeader ? first.map(field => field.toLowerCase()) : null
  const data = hasHeader ? parsed.rows.slice(1) : parsed.rows
  if (data.length > 20) errors.push('每次最多导入 20 行，请拆分后导入。')
  const rows = data.slice(0, 20).map((fields, index): HoldingImportRow => {
    const get = (key: string, fallback: number) => header ? fields[header.findIndex(value => aliases[key].includes(value))] || '' : fields[fallback] || ''
    const rowErrors: string[] = []
    const fundID = get('fund', 0)
    if (!/^\d{6}$/.test(fundID)) rowErrors.push('基金代码需要 6 位数字')
    const payload: CreateHoldingBatchItem = { fund_id: fundID, entry_type: kind, note: get('note', kind === 'buy' ? 3 : 4), source_platform: source === 'all' ? undefined : source }
    if (kind === 'opening') {
      payload.snapshot_date = get('snapshot', 3)
      if (!validDate(payload.snapshot_date)) rowErrors.push('登记或净值日期需要有效的 YYYY-MM-DD')
      if (basis === 'shares') {
        payload.shares = clean(get('shares', 1)); payload.amount = clean(get('cost', 2))
        if (!positive(payload.shares)) rowErrors.push('持有份额需要大于 0')
        if (!positive(payload.amount)) rowErrors.push('剩余成本需要大于 0')
      } else {
        payload.market_value = clean(get('market', 1)); payload.holding_profit = clean(get('profit', 2))
        if (!positive(payload.market_value)) rowErrors.push('市值需要大于 0')
        if (!payload.holding_profit || !Number.isFinite(Number(payload.holding_profit)) || Number(payload.market_value) - Number(payload.holding_profit) <= 0) rowErrors.push('持有盈亏无效或推算成本不大于 0')
      }
    } else {
      payload.amount = clean(get('cost', 1)); if (!positive(payload.amount)) rowErrors.push('买入金额需要大于 0')
      const trade = get('trade', 2)
      if (!validDate(trade.slice(0, 10))) rowErrors.push('交易日期无效')
      if (trade.length > 10) {
        const timestamp = /^\d{4}-\d{2}-\d{2} \d{2}:\d{2}(?::\d{2})?$/.test(trade) ? trade.replace(' ', 'T') + (trade.length === 16 ? ':00' : '') + '+08:00' : trade
        payload.trade_at = timestamp
        if (!Number.isFinite(Date.parse(timestamp))) rowErrors.push('交易时间无效')
      } else {
        const timing = get('timing', -1).replace(/\s/g, '')
        if (timing && !['before_close', 'after_close', '15:00前', '15:00后', '前', '后'].includes(timing)) rowErrors.push('提交时段请填写 15:00前 或 15:00后')
        payload.trade_at = buildTradeAtValue(trade, ['after_close', '15:00后', '后'].includes(timing) ? 'after_close' : timing ? 'before_close' : defaultTiming)
      }
    }
    return { index: index + 1, payload, errors: rowErrors }
  })
  return { rows, errors }
}
