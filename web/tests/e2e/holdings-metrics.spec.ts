import { expect, test } from '@playwright/test'
import {
  aggregateProfitValue, detailProfitValue, matchesAggregateFilter,
  matchesHoldingFilter, resolveAggregateMetrics, resolveHoldingMetrics,
} from '../../src/lib/holding-display'
import type { HoldingAggregateEntry, HoldingEntry, HoldingEstimateAggregateMetrics } from '../../src/hooks/use-user-portfolio'

function holding(overrides: Partial<HoldingEntry> = {}): HoldingEntry {
  return { id: 'lot', fund_id: '000001', amount: '1000', shares: '500', confirmed_nav_date: '2026-10-15', as_of_date: '2026-10-15', real_metrics_ready: true, current_market_value: '1100', today_profit: '-20', today_change_percent: '-2', note: '', created_at: '', updated_at: '', ...overrides }
}
function aggregate(overrides: Partial<HoldingAggregateEntry> = {}): HoldingAggregateEntry {
  return { fund_id: '000001', holding_count: 2, confirmed_holding_count: 1, real_metrics_ready_count: 1, incomplete_holdings_count: 1, total_principal: '2000', ready_principal: '1000', confirmed_principal: '1000', confirmed_shares: '500', official_current_market_value: '1100', official_today_profit: '-20', official_today_change_percent: '-2', metrics_scope: 'partial', real_metrics_ready: false, ...overrides }
}
const estimate = { estimate_nav: '2.2', prev_nav: '2', change_percent: '10' }
const preview: HoldingEstimateAggregateMetrics = { fund_id: '000001', preview_ready: true, fallback_ready: true, preview_current_market_value: '1100', preview_today_profit: '100', preview_today_change_percent: '10', confirmed_principal: '1000', confirmed_shares: '500', fallback_today_profit: '200' }

test('detail rows and sorting share the same share-based preview calculation', () => {
  const a = holding({ shares: '600', amount: '1000' })
  const b = holding({ shares: '400', amount: '1100' })
  const viewA = resolveHoldingMetrics(a, estimate, 'estimate')
  const viewB = resolveHoldingMetrics(b, estimate, 'estimate')
  expect(viewA).toMatchObject({ marketValue: 1320, principal: 1000, holdingProfit: 320, todayProfit: 120, change: 10, ready: true, partial: false })
  expect(viewB.todayProfit).toBe(80)
  expect(detailProfitValue(a, estimate, 'estimate')).toBe(viewA.todayProfit)
  expect(detailProfitValue(b, estimate, 'estimate')).toBe(viewB.todayProfit)
  expect([b, a].sort((left, right) => detailProfitValue(right, estimate, 'estimate')! - detailProfitValue(left, estimate, 'estimate')!).map(item => item.shares)).toEqual(['600', '400'])
})

test('official selection never substitutes preview and partial official aggregates retain subtotals', () => {
  expect(resolveHoldingMetrics(holding({ real_metrics_ready: false }), estimate, 'official')).toMatchObject({ marketValue: null, todayProfit: null, ready: false })
  const partial = resolveAggregateMetrics(aggregate(), preview, 'official')
  expect(partial).toMatchObject({ marketValue: 1100, principal: 1000, holdingProfit: 100, todayProfit: -20, change: -2, ready: true, partial: true })
  expect(aggregateProfitValue(aggregate(), preview, 'official')).toBe(-20)
  expect(matchesAggregateFilter(aggregate(), preview, 'official', 'loss')).toBe(true)
  expect(matchesAggregateFilter(aggregate(), preview, 'official', 'ready')).toBe(false)
  expect(matchesAggregateFilter(aggregate(), preview, 'official', 'partial')).toBe(true)
})

test('principal-only fallback is never a priced holding or selectable profit', () => {
  const pending = holding({ shares: undefined, confirmed_nav_date: undefined, real_metrics_ready: false })
  expect(resolveHoldingMetrics(pending, estimate, 'estimate')).toMatchObject({ marketValue: null, todayProfit: null, ready: false })
  expect(detailProfitValue(pending, estimate, 'estimate')).toBeNull()
  expect(matchesHoldingFilter(pending, estimate, 'estimate', 'ready')).toBe(false)
  expect(matchesHoldingFilter(pending, estimate, 'estimate', 'profit')).toBe(false)
  expect(aggregateProfitValue(aggregate(), { ...preview, preview_ready: false }, 'estimate')).toBeNull()
})

test('preview coverage includes unconfirmed lots within an otherwise priced fund', () => {
  expect(resolveAggregateMetrics(aggregate(), preview, 'estimate')).toMatchObject({ marketValue: 1100, principal: 1000, todayProfit: 100, ready: true, partial: true })
  expect(matchesAggregateFilter(aggregate(), preview, 'estimate', 'ready')).toBe(false)
  expect(matchesAggregateFilter(aggregate(), preview, 'estimate', 'partial')).toBe(true)
})

test('missing and nonfinite data stay unavailable, while zero P&L remains valid', () => {
  expect(resolveHoldingMetrics(holding(), { ...estimate, estimate_nav: 'Infinity' }, 'estimate').marketValue).toBeNull()
  expect(resolveHoldingMetrics(holding(), { ...estimate, prev_nav: '0' }, 'estimate').todayProfit).toBeNull()
  expect(resolveHoldingMetrics(holding({ amount: 'NaN' }), estimate, 'estimate').holdingProfit).toBeNull()
  expect(resolveHoldingMetrics(holding({ today_profit: '0', today_change_percent: '0' }), estimate, 'official')).toMatchObject({ todayProfit: 0, change: 0 })
})
