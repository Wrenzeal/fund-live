import { expect, test } from '@playwright/test'
import type { HoldingAggregateEntry, HoldingEntry, HoldingEstimateAggregateMetrics } from '../../src/hooks/use-user-portfolio'
import {
  aggregateChangeValue, aggregateProfitValue, compareOptionalNumbers,
  detailProfitValue, incompleteAggregateChildren, matchesAggregateFilter,
} from '../../src/lib/holding-display'
import { buildHoldingReconciliationSummary, buildHoldingReminderSummary, buildPortfolioHealthSummary } from '../../src/lib/holding-insights'

function holding(overrides: Partial<HoldingEntry> = {}): HoldingEntry {
  return { id: 'lot-a', fund_id: '000001', amount: '1000', shares: '500', confirmed_nav: '2', confirmed_nav_date: '2026-10-15', as_of_date: '2026-10-15', real_metrics_ready: true, note: '', created_at: '2026-10-15T10:00:00Z', updated_at: '2026-10-15T10:00:00Z', ...overrides }
}
function aggregate(overrides: Partial<HoldingAggregateEntry> = {}): HoldingAggregateEntry {
  return { fund_id: '000001', holding_count: 1, confirmed_holding_count: 1, real_metrics_ready_count: 1, incomplete_holdings_count: 0, total_principal: '1000', official_today_profit: '-20', official_today_change_percent: '-2', real_metrics_ready: true, metrics_scope: 'full', ...overrides }
}
function metrics(overrides: Partial<HoldingEstimateAggregateMetrics> = {}): HoldingEstimateAggregateMetrics {
  return { fund_id: '000001', preview_ready: true, fallback_ready: true, preview_today_profit: '30', preview_today_change_percent: '3', fallback_today_profit: '10', ...overrides }
}

test('holdings keep official, share-based preview and principal fallback accounting separate', () => {
  expect(aggregateProfitValue(aggregate(), metrics(), 'official')).toBe(-20)
  expect(aggregateProfitValue(aggregate(), metrics(), 'estimate')).toBe(30)
  expect(aggregateProfitValue(aggregate(), metrics({ preview_ready: false }), 'estimate')).toBe(10)
  expect(aggregateProfitValue(aggregate(), undefined, 'estimate')).toBeNull()
  expect(aggregateChangeValue(aggregate(), metrics(), 'official')).toBe(-2)
  expect(aggregateChangeValue(aggregate(), metrics(), 'estimate')).toBe(3)
  expect(detailProfitValue(holding({ today_profit: '-20' }), { change_percent: '3' }, 'official')).toBe(-20)
  expect(detailProfitValue(holding(), { change_percent: '3' }, 'estimate')).toBe(30)
})

test('holdings filters use selected accounting scope and retain missing values last', () => {
  expect(matchesAggregateFilter(aggregate(), metrics(), 'official', 'loss')).toBe(true)
  expect(matchesAggregateFilter(aggregate(), metrics(), 'estimate', 'profit')).toBe(true)
  expect(matchesAggregateFilter(aggregate(), undefined, 'estimate', 'profit')).toBe(false)
  const complete = holding(), pending = holding({ id: 'pending', shares: undefined, real_metrics_ready: false })
  expect(incompleteAggregateChildren([complete, pending], true)).toEqual([pending])
  expect(incompleteAggregateChildren([complete, pending], false)).toEqual([complete, pending])
  expect([null, -20, 30].sort((a, b) => compareOptionalNumbers(a, b, 'desc'))).toEqual([30, -20, null])
})

test('reconciliation keeps share NAV identity and flags missing or inconsistent confirmations', () => {
  const result = buildHoldingReconciliationSummary([holding(), holding({ id: 'mismatch', amount: '1500' }), holding({ id: 'pending', shares: undefined })])
  expect(result).toMatchObject({ total: 3, okCount: 1, reviewCount: 1, pendingCount: 1 })
  expect(result.items[0]).toMatchObject({ id: 'mismatch', impliedPrincipal: 1000, difference: 500 })
})

test('portfolio review retains actual concentration and disclosed underlying overlap', () => {
  const result = buildPortfolioHealthSummary({
    holdings: [holding(), holding({ id: 'lot-b', fund_id: '000002' })],
    aggregates: [aggregate({ total_principal: '6000' }), aggregate({ fund_id: '000002', total_principal: '4000' })],
    aggregateMetrics: {}, metricScope: 'official',
    topHoldingsByFundID: {
      '000001': [{ stock_code: '600000', stock_name: '测试公司', holding_ratio: '10' }],
      '000002': [{ stock_code: '600000', stock_name: '测试公司', holding_ratio: '20' }],
    },
  })
  expect(result.signals.find(item => item.id === 'largest-position-danger')).toMatchObject({ metric: '60.0%', tone: 'danger' })
  expect(result.signals.find(item => item.id === 'top-holding-overlap')?.description).toContain('测试公司')
  expect(result.signals.find(item => item.id === 'complete-data')).toBeDefined()
  expect(result.signals.filter(item => item.tone === 'warning' || item.tone === 'danger').map(item => item.id).sort()).toEqual([
    'largest-position-danger', 'profit-driver', 'top-holding-overlap',
  ])
  expect(result.reviewCount).toBe(3)
  expect(result).not.toHaveProperty('score')
})

test('reminders preserve pending data and scope-specific drawdown alerts', () => {
  const base = { holdings: [holding({ shares: undefined, real_metrics_ready: false })], aggregates: [aggregate({ official_today_change_percent: '-6' })], aggregateMetrics: { '000001': metrics() } }
  const official = buildHoldingReminderSummary({ ...base, metricScope: 'official' })
  expect(official.reminders.find(item => item.id === 'official-nav-pending')).toMatchObject({ tone: 'danger' })
  expect(official.reminders.find(item => item.id === 'large-drop')).toMatchObject({ tone: 'danger', metric: '-6.00%' })
  const preview = buildHoldingReminderSummary({ ...base, metricScope: 'estimate' })
  expect(preview.reminders.find(item => item.id === 'large-drop')).toBeUndefined()
})

test('every counted portfolio review remains visible even with many simultaneous facts', () => {
  const ids = ['000001', '000002', '000003']
  const result = buildPortfolioHealthSummary({
    holdings: [
      holding({ manual_confirmation: true }),
      holding({ id: 'pending', shares: undefined, real_metrics_ready: false }),
      holding({ id: 'second', fund_id: ids[1] }), holding({ id: 'third', fund_id: ids[2] }),
    ],
    aggregates: ids.map((id, index) => aggregate({ fund_id: id, total_principal: ['6000', '3000', '1000'][index], holding_count: index === 0 ? 2 : 1 })),
    metricScope: 'official', aggregateMetrics: {},
    exposureSnapshots: Object.fromEntries(ids.map(id => [id, {
      sectorSnapshot: { fund_id: id, as_of_date: '2026-06-30', primary_sector_code: 'test', primary_sector_name: '共同板块', source: 'holdings', confidence: 'high', breakdown: [{ sector_code: 'test', sector_name: '共同板块', weight_percent: '100', rank: 1 }] },
      themeSnapshot: { fund_id: id, as_of_date: '2026-06-30', primary_theme_code: 'test', primary_theme_name: '共同主题', source: 'holdings', confidence: 'high', breakdown: [{ theme_code: 'test', theme_name: '共同主题', weight_percent: '100', rank: 1 }] },
    }])),
    topHoldingsByFundID: Object.fromEntries(ids.map(id => [id, [{ stock_code: '600000', stock_name: '共同持仓', holding_ratio: '20' }]])),
  })
  expect(result.signals.length).toBeGreaterThan(8)
  expect(result.signals.some(item => item.id === 'duplicate-exposure')).toBe(true)
  expect(result.signals.some(item => item.id === 'top-holding-overlap')).toBe(true)
  expect(result.reviewCount).toBe(result.signals.filter(item => item.tone === 'danger' || item.tone === 'warning').length)
})
