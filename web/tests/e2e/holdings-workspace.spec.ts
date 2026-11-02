import { expect, test, type Page } from '@playwright/test'

async function mockHoldings(page: Page, options: { empty?: boolean; pending?: boolean; partial?: boolean; missingPrevious?: boolean; fail?: boolean; delay?: number } = {}) {
  const writes: { path: string; body: unknown }[] = []
  let added = false
  const fund = (id: string) => ({ id, name: id === '000001' ? '测试稳健基金' : '测试成长基金', type: 'hybrid', company: '测试基金公司', manager: '测试经理', nav: '2.4', scale: '0', updated_at: '2026-11-02T08:00:00Z' })
  await page.route('**/api/v1/**', async route => {
    const request = route.request()
    const url = new URL(request.url())
    const path = url.pathname
    const ok = (data: unknown) => route.fulfill({ json: { success: true, data } })
    if (request.method() !== 'GET') {
      writes.push({ path, body: request.postDataJSON() })
      added = true
      return ok({ id: 'new-lot' })
    }
    if (path.endsWith('/auth/me')) return ok({ user: { id: 'test-user', email: 'test@example.com', display_name: '测试用户', is_admin: false, preferred_quote_source: 'sina', provider: 'email_code', email_verified: true }, expires_at: '2099-01-01T00:00:00Z' })
    if (path.endsWith('/auth/config')) return ok({ email_code_login_enabled: true, google_login_enabled: false })
    if (path.endsWith('/market/status')) return ok({ is_trading: false, is_trading_day: true, session: 'after_hours', current_time: '2026-11-02T16:00:00+08:00', current_date: '2026-11-02', display_date: '2026-11-02', previous_trading_day: '2026-10-30', last_trading_day: '2026-11-02', next_trading_day: '2026-11-03', time_until_next_session_seconds: 0, time_until_next_transition_seconds: 0, timezone: 'Asia/Shanghai', covered_years: [2026] })
    if (path.endsWith('/market/pricing-date')) return ok({ trade_at: '2026-11-02T15:01:00+08:00', trade_date: '2026-11-02', pricing_date: '2026-11-03', rule: 'next_trading_day', is_trading_day: true, after_cutoff: true, message: '将按下一交易日净值确认。', timezone: 'Asia/Shanghai' })
    if (path.endsWith('/user/holdings')) {
      if (options.delay) await new Promise(resolve => setTimeout(resolve, options.delay))
      if (options.fail) return route.fulfill({ status: 503, json: { success: false, error: { message: '测试持仓服务暂不可用' } } })
      if (options.empty && !added) return ok({ items: [], aggregates: [], summary: { total_principal: '0', real_metrics_ready_count: 0, total_holdings: 0 } })
      const items = ['000001', '000002'].map((id, index) => ({ id: `lot-${id}`, fund_id: id, amount: index ? '1100' : '1000', shares: index ? '400' : '500', confirmed_nav: index ? '2.75' : '2', confirmed_nav_date: '2026-10-15', as_of_date: '2026-10-15', actual_date: '2026-11-02', real_metrics_ready: !options.pending && !(options.partial && index === 1), current_market_value: options.pending ? undefined : index ? '800' : '1200', today_profit: options.pending ? undefined : index ? '-40' : '100', today_change_percent: index ? '-4.7619' : '9.0909', fund: fund(id), note: '', created_at: '2026-10-15T08:00:00Z', updated_at: '2026-11-02T08:00:00Z' }))
      const aggregates = items.map(item => ({ fund_id: item.fund_id, fund: item.fund, holding_count: 1, confirmed_holding_count: 1, real_metrics_ready_count: item.real_metrics_ready ? 1 : 0, incomplete_holdings_count: item.real_metrics_ready ? 0 : 1, real_metrics_ready: item.real_metrics_ready, metrics_scope: item.real_metrics_ready ? 'full' : 'none', total_principal: item.amount, confirmed_principal: item.amount, ready_principal: item.real_metrics_ready ? item.amount : undefined, confirmed_shares: item.shares, official_current_market_value: item.current_market_value, official_today_profit: item.today_profit, official_today_change_percent: item.real_metrics_ready ? item.today_change_percent : undefined }))
      return ok({ items, aggregates, summary: { total_principal: '2100', ready_principal: options.pending ? '0' : '2100', total_current_market_value: options.pending ? undefined : '2000', total_today_profit: options.pending ? undefined : '60', total_today_change_percent: '3.09', real_metrics_ready_count: options.pending ? 0 : 2, total_holdings: 2, metrics_scope: options.pending ? 'none' : 'full' } })
    }
    if (path.endsWith('/user/watchlist/groups') || path.includes('/holdings/transactions') || path.includes('/announcements')) return ok([])
    if (path.endsWith('/fund/search')) return ok([fund('000001')])
    const match = path.match(/\/fund\/(\d{6})(?:\/(\w+))?$/)
    if (match) {
      const [, id, endpoint] = match
      if (endpoint === 'estimate') return ok({ fund_id: id, fund_name: fund(id).name, change_percent: id === '000001' ? '9.0909' : '10', estimate_nav: id === '000001' ? '2.4' : '2.2', prev_nav: options.missingPrevious ? undefined : id === '000001' ? '2.2' : '2', calculated_at: '2026-11-02T08:00:00Z' })
      if (endpoint === 'holdings') return ok({ fund: fund(id), holdings: [] })
      if (endpoint === 'dashboard') return ok({ fund: fund(id) })
      if (!endpoint) return ok(fund(id))
    }
    return ok({})
  })
  return writes
}

test('holds market value and fund list together and keeps pricing choice explicit', async ({ page }) => {
  await mockHoldings(page)
  await page.goto('/holdings')
  const summary = page.getByTestId('holdings-summary')
  await expect(summary.getByText('¥2,000.00', { exact: true })).toBeVisible()
  await expect(summary.getByText('-¥100.00', { exact: true })).toBeVisible()
  await expect(page.getByTestId('holding-aggregate-row')).toHaveCount(2)
  await page.getByRole('button', { name: '盘中预估', exact: true }).click()
  await expect(summary.getByText('¥2,080.00', { exact: true })).toBeVisible()
  await page.getByRole('button', { name: '官方净值', exact: true }).click()
  await expect(summary.getByText('¥2,000.00', { exact: true })).toBeVisible()
})

test('zero official coverage cannot silently display estimate figures', async ({ page }) => {
  await mockHoldings(page, { pending: true })
  await page.goto('/holdings')
  const summary = page.getByTestId('holdings-summary')
  await expect(summary.getByText('等待净值更新')).toBeVisible()
  await expect(summary.getByText('¥2,080.00', { exact: true })).toHaveCount(0)
  await page.getByRole('button', { name: '盘中预估', exact: true }).click()
  await expect(summary.getByText('¥2,080.00', { exact: true })).toBeVisible()
})

test('loading and failure do not look like an empty portfolio', async ({ page }) => {
  await mockHoldings(page, { fail: true })
  await page.goto('/holdings')
  await expect(page.getByText('测试持仓服务暂不可用')).toBeVisible()
  await expect(page.getByRole('button', { name: '记录第一笔' })).toHaveCount(0)
})

test('first record does not seed demo data and saves the original buy contract', async ({ page }) => {
  const writes = await mockHoldings(page, { empty: true })
  await page.goto('/holdings')
  await page.getByRole('button', { name: '记录第一笔' }).click()
  await page.getByLabel('基金', { exact: true }).fill('000001')
  await page.getByRole('list', { name: '基金搜索结果' }).getByRole('button', { name: /测试稳健基金/ }).click()
  await page.getByLabel('买入金额（元）').fill('1000')
  await page.getByRole('button', { name: '保存买入记录' }).click()
  await expect(page.getByTestId('holdings-summary')).toBeVisible()
  expect(writes).toHaveLength(1)
  expect(writes[0]).toMatchObject({ path: '/api/v1/user/holdings', body: { fund_id: '000001', amount: '1000' } })
})

test('mobile money, controls and secondary actions remain usable in all themes', async ({ page }) => {
  await mockHoldings(page)
  await page.setViewportSize({ width: 390, height: 844 })
  for (const theme of ['classic', 'dark', 'cyber']) {
    await page.addInitScript(value => localStorage.setItem('fund-ui-preferences-v1', JSON.stringify({ themeType: value, viewMode: 'professional' })), theme)
    await page.goto('/holdings')
    await expect(page.getByTestId('holdings-summary')).toBeVisible()
    await expect(page.getByTestId('holding-aggregate-row')).toHaveCount(2)
    expect(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth)).toBe(false)
    await page.getByRole('button', { name: '交易记录', exact: true }).click()
    await expect(page.getByRole('button', { name: '持仓', exact: true })).toBeVisible()
  }
})


test('partial official coverage compares only the priced market value and its cost', async ({ page }) => {
  await mockHoldings(page, { partial: true })
  await page.goto('/holdings')
  const summary = page.getByTestId('holdings-summary')
  await expect(summary.getByText('已计价持仓市值', { exact: true })).toBeVisible()
  await expect(summary.getByText('¥1,200.00', { exact: true })).toBeVisible()
  await expect(summary.getByText('+¥200.00', { exact: true })).toBeVisible()
  await expect(summary.getByText(/已计价 1\/2 笔/)).toBeVisible()
  await page.getByRole('button', { name: '分笔明细' }).click()
  await expect(page.getByTestId('holding-detail-row').nth(1).getByText('¥880.00', { exact: true })).toHaveCount(0)
})

test('waiting for holdings never shows first-record onboarding', async ({ page }) => {
  await mockHoldings(page, { delay: 1500 })
  await page.goto('/holdings')
  await expect(page.getByRole('status', { name: '' }).filter({ hasText: '正在读取持仓' })).toBeVisible()
  await expect(page.getByRole('button', { name: '记录第一笔' })).toHaveCount(0)
  await expect(page.getByTestId('holdings-summary')).toBeVisible()
})

test('sale keeps share and cost contracts explicit and correction dividend adjustment remain reachable', async ({ page }) => {
  const writes = await mockHoldings(page)
  await page.goto('/holdings')
  await page.getByRole('button', { name: '查看 测试稳健基金 明细' }).click()
  const row = page.getByTestId('holding-detail-row')
  await row.getByRole('button', { name: '校正 测试稳健基金 持仓' }).click()
  await expect(row.getByText('校正持仓口径', { exact: true })).toBeVisible()
  await row.getByRole('button', { name: '记录 测试稳健基金 分红' }).click()
  await expect(row.getByText('记录分红 / 红利再投', { exact: true })).toBeVisible()
  await row.getByRole('button', { name: '调整 测试稳健基金 份额' }).click()
  await expect(row.getByText('记录份额调整', { exact: true })).toBeVisible()
  await row.getByRole('button', { name: '卖出/减仓 测试稳健基金 持仓' }).click()
  const cost = row.getByLabel('减少成本本金（非到账金额）')
  const shares = row.getByLabel('卖出份额（优先填写）')
  await cost.fill('200')
  await shares.fill('50')
  await expect(cost).toHaveValue('')
  await row.getByRole('button', { name: '确认减仓', exact: true }).click()
  expect(writes).toHaveLength(1)
  expect(writes[0].path).toBe('/api/v1/user/holdings/lot-000001/sell')
  expect(writes[0].body).toMatchObject({ shares: '50', sell_all: false })
})

test('desktop and narrow mobile reference captures', async ({ page }, testInfo) => {
  await mockHoldings(page)
  for (const theme of ['classic', 'dark', 'cyber']) {
    for (const width of [1440, 360]) {
      await page.setViewportSize({ width, height: width > 1000 ? 1000 : 900 })
      await page.addInitScript(value => localStorage.setItem('fund-ui-preferences-v1', JSON.stringify({ themeType: value, viewMode: 'professional' })), theme)
      await page.goto('/holdings')
      await expect(page.locator('html')).toHaveAttribute('data-theme', theme)
      await expect(page.getByTestId('holdings-summary')).toBeVisible()
      await expect(page.getByTestId('holding-aggregate-row')).toHaveCount(2)
      expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(width)
      await page.screenshot({ path: testInfo.outputPath(theme + '-' + width + '.png'), fullPage: true })
    }
  }
})


test('missing previous NAV preserves current valuation and leaves daily movement unknown', async ({ page }) => {
  await mockHoldings(page, { missingPrevious: true })
  await page.goto('/holdings')
  await page.getByRole('button', { name: '盘中预估', exact: true }).click()
  await expect(page.getByTestId('holdings-summary').getByText('¥2,080.00', { exact: true })).toBeVisible()
  const first = page.getByTestId('holding-aggregate-row').first()
  await expect(first.getByText('¥1,200.00', { exact: true })).toBeVisible()
  await expect(first.getByText('--', { exact: true })).toBeVisible()
})
