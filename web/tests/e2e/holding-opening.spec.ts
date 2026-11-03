import { expect, test, type Page } from '@playwright/test'

async function mockOpening(page: Page, failNAV = false, partialImport = false) {
  const writes: { path: string; body: Record<string, unknown> }[] = []
  const holdings: Record<string, unknown>[] = []
  const fund = { id: '014662', name: '测试黄金ETF联接C', type: 'feeder' }
  const create = (payload: Record<string, unknown>) => ({ id: 'lot-' + holdings.length, fund_id: fund.id, fund, entry_type: payload.entry_type || 'buy', amount: payload.entry_type === 'opening' ? payload.amount || '900' : payload.amount, shares: payload.shares || '500', as_of_date: payload.snapshot_date || '2025-03-04', trade_at: payload.trade_at || '', confirmed_nav_date: payload.entry_type === 'opening' ? undefined : '2025-03-04', current_market_value: '1000', today_profit: '5', real_metrics_ready: true, real_metrics_stale: true, actual_date: '2026-11-02', manual_confirmation: payload.entry_type === 'opening', note: '' })
  await page.route('**/api/v1/**', async route => {
    const request = route.request(), url = new URL(request.url()), path = url.pathname
    const ok = (data: unknown) => route.fulfill({ json: { success: true, data } })
    if (request.method() === 'POST') {
      const payload = request.postDataJSON(); writes.push({ path, body: payload })
      if (failNAV && payload.market_value) return route.fulfill({ status: 409, json: { success: false, error: { message: '该净值日暂无可用官方净值，请选择已公布日期或改用持有份额录入' } } })
      if (path.endsWith('/batch')) {
        const items = payload.items as Record<string, unknown>[]
        const partial = partialImport && writes.filter(write => write.path.endsWith('/batch')).length === 1
        const created = items.filter((_, index) => !partial || index !== 1).map(item => { const holding = create(item); holdings.push(holding); return holding })
        return ok({ total: items.length, created_count: created.length, failed_count: partial ? 1 : 0, failed: partial ? [{ index: 1, fund_id: '014662', message: '该行净值暂未就绪' }] : [], created })
      }
      const holding = create(payload); holdings.push(holding); return ok(holding)
    }
    if (path.endsWith('/auth/me')) return ok({ user: { id: 'opening-owner', email: 'test@example.com', display_name: '测试用户' }, expires_at: '2099-01-01T00:00:00Z' })
    if (path.includes('/announcements') || path.endsWith('/user/watchlist/groups') || path.includes('/holdings/transactions')) return ok([])
    if (path.endsWith('/auth/config')) return ok({ email_code_login_enabled: true })
    if (path.endsWith('/market/status')) return ok({ is_trading: false, is_trading_day: true, session: 'after_hours', current_time: '2026-11-03T16:00:00+08:00', current_date: '2026-11-03', previous_trading_day: '2026-11-02', last_trading_day: '2026-11-03', next_trading_day: '2026-11-04', timezone: 'Asia/Shanghai' })
    if (path.endsWith('/market/pricing-date')) return ok({ trade_date: '2025-03-03', pricing_date: '2025-03-04', rule: 'next_trading_day', after_cutoff: true, is_trading_day: true, timezone: 'Asia/Shanghai' })
    if (path.endsWith('/fund/search')) return ok([fund])
    if (path.endsWith('/user/holdings')) return ok({ items: holdings, aggregates: holdings.length ? [{ fund_id: fund.id, fund, holding_count: holdings.length, confirmed_holding_count: holdings.length, real_metrics_ready_count: holdings.length, real_metrics_ready: true, metrics_scope: 'full', total_principal: String(holdings.reduce((sum, item) => sum + Number(item.amount), 0)), ready_principal: String(holdings.reduce((sum, item) => sum + Number(item.amount), 0)), confirmed_shares: String(holdings.length * 500), official_current_market_value: String(holdings.length * 1000), official_today_profit: String(holdings.length * 5) }] : [], summary: { total_principal: String(holdings.reduce((sum, item) => sum + Number(item.amount), 0)) } })
    if (path.endsWith('/history')) return ok({ fund_id: fund.id, days: 1, points: [{ fund_id: fund.id, date: '2026-11-02', net_asset_val: '2', daily_return: '0.5' }] })
    if (path.endsWith('/estimate')) return ok({ fund_id: fund.id, estimate_nav: '2', prev_nav: '2', change_percent: '0' })
    return ok({})
  })
  return writes
}

async function chooseOpeningFund(page: Page) {
  await page.getByRole('button', { name: '录入已有持仓', exact: true }).click()
  await page.getByPlaceholder('输入基金代码或名称', { exact: true }).fill('014662')
  await page.getByRole('list', { name: '已有持仓基金搜索结果' }).getByRole('button').click()
}

test('shares and cost create an existing position with value without inventing a purchase', async ({ page }) => {
  const writes = await mockOpening(page)
  await page.goto('/holdings'); await chooseOpeningFund(page)
  await page.getByLabel('持有份额', { exact: true }).fill('500')
  await page.getByLabel('剩余成本（元）', { exact: true }).fill('900')
  await page.getByLabel('持仓登记日期').fill('2026-11-02')
  await page.getByRole('button', { name: '保存已有持仓' }).click()
  await expect(page.getByTestId('holdings-summary').getByText('¥1,000.00', { exact: true })).toBeVisible()
  await expect(page.getByTestId('holdings-summary').getByText('+¥100.00', { exact: true })).toBeVisible()
  await expect(page.getByTestId('holdings-summary').getByText(/部分净值待更新/)).toBeVisible()
  expect(writes).toHaveLength(1)
  expect(writes[0].body).toMatchObject({ entry_type: 'opening', shares: '500', amount: '900', snapshot_date: '2026-11-02' })
  expect(writes[0].body.trade_at).toBeUndefined()
})

test('market value and profit use an explicit official NAV date', async ({ page }) => {
  const writes = await mockOpening(page)
  await page.goto('/holdings'); await chooseOpeningFund(page)
  await page.getByRole('button', { name: '市值与持有盈亏', exact: true }).click()
  await page.getByLabel('持仓市值（元）', { exact: true }).fill('1000')
  await page.getByLabel('持有盈亏（元）', { exact: true }).fill('100')
  await page.getByLabel('净值对应日期').fill('2026-11-02')
  await page.getByRole('button', { name: '保存已有持仓' }).click()
  await expect(page.getByTestId('holdings-summary')).toBeVisible()
  expect(writes[0].body).toMatchObject({ entry_type: 'opening', market_value: '1000', holding_profit: '100', snapshot_date: '2026-11-02' })
  expect(writes[0].body.amount).toBeUndefined()
})

test('existing-position import saves shares and cost and returns to a priced portfolio', async ({ page }) => {
  const writes = await mockOpening(page)
  await page.goto('/holdings')
  await page.getByRole('button', { name: '导入历史记录', exact: true }).click()
  await page.getByLabel('CSV 数据').fill('基金代码,持有份额,剩余成本,登记日期\n014662,500,900,2026-11-02')
  await page.getByRole('button', { name: '确认导入', exact: true }).click()
  await expect(page.getByTestId('holdings-summary').getByText('¥1,000.00', { exact: true })).toBeVisible()
  expect(writes[0].body.items).toEqual([{ fund_id: '014662', entry_type: 'opening', shares: '500', amount: '900', snapshot_date: '2026-11-02', note: '', source_platform: 'alipay' }])
})

test('importing historical purchase draft retains its original timestamp on save', async ({ page }) => {
  const writes = await mockOpening(page)
  await page.goto('/holdings')
  await page.getByRole('button', { name: '导入历史记录', exact: true }).click()
  await page.getByRole('button', { name: '导入买入记录', exact: true }).click()
  await page.getByLabel('CSV 数据').fill('基金代码,买入金额,交易时间,备注\n014662,1000,2025-03-03T16:20:00+08:00,历史买入')
  await page.getByRole('button', { name: '带入买入表单' }).click()
  await expect(page.getByLabel('交易日期', { exact: true })).toHaveValue('2025-03-03')
  await page.getByRole('button', { name: '保存买入记录', exact: true }).click()
  await expect(page.getByTestId('holdings-summary')).toBeVisible()
  expect(writes[0].body.trade_at).toBe('2025-03-03T16:20:00+08:00')
})

test('unavailable NAV preserves the existing-position form for correction', async ({ page }) => {
  await mockOpening(page, true)
  await page.goto('/holdings'); await chooseOpeningFund(page)
  await page.getByRole('button', { name: '市值与持有盈亏', exact: true }).click()
  await page.getByLabel('持仓市值（元）', { exact: true }).fill('1000')
  await page.getByLabel('持有盈亏（元）', { exact: true }).fill('100')
  await page.getByRole('button', { name: '保存已有持仓' }).click()
  await expect(page.getByRole('alert').filter({ hasText: '该净值日暂无可用官方净值' })).toBeVisible()
  await expect(page.getByLabel('持仓市值（元）', { exact: true })).toHaveValue('1000')
})

test('opening entry is readable on mobile in both input modes', async ({ page }, testInfo) => {
  await mockOpening(page)
  await page.setViewportSize({ width: 390, height: 900 })
  await page.goto('/holdings'); await chooseOpeningFund(page)
  for (const mode of ['份额与成本', '市值与持有盈亏']) {
    await page.getByRole('button', { name: mode, exact: true }).click()
    expect(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth)).toBe(false)
    await page.evaluate(() => window.scrollTo({ top: 0, behavior: 'instant' }))
    await page.screenshot({ path: testInfo.outputPath(mode + '.png'), fullPage: true })
  }
})


test('partial import retries only failed rows instead of adding successful positions again', async ({ page }) => {
  const writes = await mockOpening(page, false, true)
  await page.goto('/holdings')
  await page.getByRole('button', { name: '导入历史记录', exact: true }).click()
  await page.getByLabel('CSV 数据').fill('基金代码,持有份额,剩余成本,登记日期,备注\n014662,500,900,2026-11-02,第一笔\n014662,500,900,2026-11-02,第二笔')
  await page.getByRole('button', { name: '确认导入', exact: true }).click()
  await expect(page.getByText('已导入 1 行，失败 1 行。', { exact: true })).toBeVisible()
  await page.getByRole('button', { name: '确认导入', exact: true }).click()
  await expect(page.getByTestId('holdings-summary')).toBeVisible()
  expect(writes).toHaveLength(2)
  expect(writes[1].body.items).toEqual([expect.objectContaining({ note: '第二笔' })])
})
