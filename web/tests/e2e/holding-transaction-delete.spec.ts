import { expect, test, type Page } from '@playwright/test'
import { readabilityIssues } from '../helpers/readability'

async function mockLedger(page: Page, options: { failDelete?: boolean; delayDelete?: boolean } = {}) {
  const deletes: string[] = []
  let fail = options.failDelete
  const fund = { id: '005827', name: '测试基金', type: 'hybrid' }
  let records = ['active', 'voided'].map((status, index) => ({ id: 'tx-' + status, fund_id: fund.id, fund, type: 'buy', amount: index ? '200' : '1000', shares: '500', voided: status === 'voided', note: index ? '需要删除的作废流水' : '需要删除的买入流水', created_at: '2026-11-03T08:00:00Z', metadata: {} }))
  await page.route('**/api/v1/**', async route => {
    const url = new URL(route.request().url()), path = url.pathname
    const ok = (data: unknown) => route.fulfill({ json: { success: true, data } })
    if (route.request().method() === 'DELETE') {
      deletes.push(path)
      if (options.delayDelete) await new Promise(resolve => setTimeout(resolve, 800))
      if (fail) return route.fulfill({ status: 503, json: { success: false, error: { message: '测试删除失败，请重试' } } })
      records = records.filter(record => !path.endsWith('/' + record.id))
      return ok({ deleted: true })
    }
    if (path.endsWith('/auth/me')) return ok({ user: { id: 'test-owner', email: 'test@example.com', display_name: '测试用户', is_admin: false }, expires_at: '2099-01-01T00:00:00Z' })
    if (path.endsWith('/auth/config')) return ok({ email_code_login_enabled: true })
    if (path.includes('/announcements') || path.endsWith('/user/watchlist/groups')) return ok([])
    if (path.endsWith('/market/status')) return ok({ is_trading: false, is_trading_day: true, session: 'after_hours', current_time: '2026-11-03T16:00:00+08:00', current_date: '2026-11-03', timezone: 'Asia/Shanghai' })
    if (path.endsWith('/user/holdings')) return ok({ items: [{ id: 'holding', fund_id: fund.id, fund, amount: '1000', shares: '500', real_metrics_ready: true, current_market_value: '1000', today_profit: '0', note: '' }], aggregates: [{ fund_id: fund.id, fund, holding_count: 1, confirmed_holding_count: 1, real_metrics_ready_count: 1, real_metrics_ready: true, metrics_scope: 'full', total_principal: '1000', ready_principal: '1000', official_current_market_value: '1000', official_today_profit: '0' }], summary: { total_principal: '1000' } })
    if (path.endsWith('/holdings/transactions')) return ok(records)
    const detailID = path.match(/\/holdings\/transactions\/([^/]+)$/)?.[1]
    if (detailID) {
      const transaction = records.find(record => record.id === detailID)
      return transaction ? ok({ transaction, related_transactions: [], subsequent_transactions: [], impact_chain: [], rollback_preview: { title: '流水影响', summary: '测试只读预览', affected_fields: [], can_apply_automatically: false } }) : route.fulfill({ status: 404, json: { success: false, error: { message: '流水不存在' } } })
    }
    if (path.endsWith('/estimate')) return ok({ fund_id: fund.id, estimate_nav: '2', prev_nav: '2', change_percent: '0' })
    return ok({})
  })
  return { deletes, recover: () => { fail = false } }
}

async function openLedger(page: Page) {
  await page.goto('/holdings')
  await page.getByRole('button', { name: '交易记录', exact: true }).click()
}

test('delete is explicit, cancellation makes no request, and confirming preserves current holdings', async ({ page }) => {
  const { deletes } = await mockLedger(page)
  await openLedger(page)
  const row = page.getByTestId('holding-transaction-row').filter({ hasText: '需要删除的买入流水' })
  await row.getByRole('button', { name: /删除.*流水/ }).click()
  const dialog = page.getByRole('dialog', { name: '删除这条流水？' })
  await expect(dialog.getByText(/当前持仓金额和份额不会改变/)).toBeVisible()
  await dialog.getByRole('button', { name: '取消', exact: true }).click()
  expect(deletes).toEqual([])
  await row.getByRole('button', { name: /删除.*流水/ }).click()
  await dialog.getByRole('button', { name: '确认删除', exact: true }).click()
  await expect(page.getByText('需要删除的买入流水', { exact: true })).toHaveCount(0)
  expect(deletes).toEqual(['/api/v1/user/holdings/transactions/tx-active'])
  await page.getByRole('button', { name: '持仓', exact: true }).click()
  await expect(page.getByTestId('holdings-summary').getByText('¥1,000.00', { exact: true })).toBeVisible()
})

test('voided activity can also be deleted and remains absent after reload', async ({ page }) => {
  const { deletes } = await mockLedger(page)
  await openLedger(page)
  await page.getByTestId('holding-transaction-row').filter({ hasText: '需要删除的作废流水' }).getByRole('button', { name: /删除.*流水/ }).click()
  await page.getByRole('dialog').getByRole('button', { name: '确认删除', exact: true }).click()
  await expect(page.getByText('需要删除的作废流水', { exact: true })).toHaveCount(0)
  await page.reload()
  await page.getByRole('button', { name: '交易记录', exact: true }).click()
  await expect(page.getByText('需要删除的作废流水', { exact: true })).toHaveCount(0)
  expect(deletes).toEqual(['/api/v1/user/holdings/transactions/tx-voided'])
})

test('failed deletion keeps the record and supports retry without closing the confirmation', async ({ page }) => {
  const mock = await mockLedger(page, { failDelete: true })
  await openLedger(page)
  await page.getByTestId('holding-transaction-row').first().getByRole('button', { name: /删除.*流水/ }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByRole('button', { name: '确认删除', exact: true }).click()
  await expect(dialog.getByRole('alert')).toHaveText('测试删除失败，请重试')
  expect(mock.deletes).toHaveLength(1)
  mock.recover()
  await dialog.getByRole('button', { name: '确认删除', exact: true }).click()
  await expect(dialog).toHaveCount(0)
})

test('detail has delete entry and returns to transaction records after deletion', async ({ page }) => {
  const { deletes } = await mockLedger(page)
  await page.goto('/holdings/transactions/tx-active')
  await page.getByRole('button', { name: /删除.*流水/ }).click()
  await page.getByRole('dialog').getByRole('button', { name: '确认删除', exact: true }).click()
  await expect(page).toHaveURL(/\/holdings\?tab=ledger$/)
  await expect(page.getByText('需要删除的作废流水', { exact: true })).toBeVisible()
  expect(deletes).toHaveLength(1)
})

test('mobile confirmation is readable, Escape cancels and pending deletion cannot be repeated', async ({ page }, testInfo) => {
  const { deletes } = await mockLedger(page, { delayDelete: true })
  await page.setViewportSize({ width: 390, height: 844 })
  await openLedger(page)
  await page.getByTestId('holding-transaction-row').first().getByRole('button', { name: /删除.*流水/ }).click()
  await page.keyboard.press('Escape')
  expect(deletes).toHaveLength(0)
  await page.getByTestId('holding-transaction-row').first().getByRole('button', { name: /删除.*流水/ }).click()
  const dialog = page.getByRole('dialog')
  expect(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth)).toBe(false)
  await expect.poll(() => readabilityIssues(page, 'dialog h2, dialog p, dialog dd, dialog button')).toEqual([])
  await dialog.screenshot({ path: testInfo.outputPath('delete-confirmation-mobile.png') })
  await dialog.getByRole('button', { name: '确认删除', exact: true }).click()
  await expect(dialog.getByRole('button', { name: '正在删除…', exact: true })).toBeDisabled()
  await expect(dialog).toHaveCount(0)
  expect(deletes).toHaveLength(1)
})
