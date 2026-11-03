import { expect, test } from '@playwright/test'
import { parseHoldingImport } from '../../src/lib/holding-import'

test('existing-position import keeps shares/cost and does not invent a buy time', () => {
  const parsed = parseHoldingImport('基金代码,持有份额,剩余成本,登记日期,备注\n014662,500,900,2026-11-02,"平台迁移,保留份额"', 'opening', 'shares', 'alipay')
  expect(parsed.errors).toEqual([])
  expect(parsed.rows[0].errors).toEqual([])
  expect(parsed.rows[0].payload).toEqual({ fund_id: '014662', entry_type: 'opening', shares: '500', amount: '900', snapshot_date: '2026-11-02', note: '平台迁移,保留份额', source_platform: 'alipay' })
})
test('value/profit imports preserve loss signs and their NAV date', () => {
  const parsed = parseHoldingImport('基金代码,持仓市值,持有盈亏,净值日期\n014662,900,-100,2026-11-02', 'opening', 'value', 'manual')
  expect(parsed.rows[0].payload).toMatchObject({ market_value: '900', holding_profit: '-100', snapshot_date: '2026-11-02' })
  expect(parsed.rows[0].payload.amount).toBeUndefined()
})
test('historical buy import preserves explicit clock time and cutoff', () => {
  const parsed = parseHoldingImport('基金代码,买入金额,交易日期,提交时段,备注\n014662,1000,2025-03-03,15:00后,晚间提交', 'buy', 'shares', 'alipay')
  expect(parsed.rows[0].payload.trade_at).toBe('2025-03-03T15:01:00+08:00')
  const precise = parseHoldingImport('014662,1000,2025-03-03T14:20:00+08:00,原时间', 'buy', 'shares', 'manual')
  expect(precise.rows[0].payload.trade_at).toBe('2025-03-03T14:20:00+08:00')
})
test('invalid dates, missing shares and over-limit input are explicit errors', () => {
  expect(parseHoldingImport('014662,,900,2026-02-30', 'opening', 'shares', 'manual').rows[0].errors.length).toBeGreaterThan(0)
  const large = parseHoldingImport(Array(21).fill('014662,500,900,2026-11-02').join('\n'), 'opening', 'shares', 'manual')
  expect(large.errors).toContain('每次最多导入 20 行，请拆分后导入。')
})
