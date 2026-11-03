import { expect, test, type Page } from '@playwright/test'

const names: Record<string, string> = { '005827': '易方达蓝筹精选混合', '003095': '中欧医疗健康混合' }

async function mockAuction(page: Page, options: { minimal?: boolean; authenticated?: boolean; session?: 'call_auction' | 'morning'; historyError?: boolean; emptyHistory?: boolean; delayedSecondFund?: boolean } = {}) {
  const requested: string[] = []
  let historyError = options.historyError
  await page.addInitScript(minimal => localStorage.setItem('fund-ui-preferences-v1', JSON.stringify({ themeType: 'classic', viewMode: minimal ? 'minimal' : 'professional' })), options.minimal)
  const fund = (id: string) => ({ id, name: names[id], type: '混合型', manager: '测试经理', company: '测试基金公司', net_asset_val: '1.5', scale: '0', updated_at: '2026-11-02T15:00:00Z' })
  await page.route('**/api/v1/**', async route => {
    const url = new URL(route.request().url())
    requested.push(url.pathname)
    const ok = (data: unknown) => route.fulfill({ json: { success: true, data } })
    if (url.pathname.endsWith('/auth/me')) return options.minimal || options.authenticated
      ? ok({ user: { id: 'test-user', email: 'test@example.com', display_name: '测试用户' }, expires_at: '2099-01-01T00:00:00Z' })
      : route.fulfill({ status: 401, json: { success: false } })
    if (url.pathname.endsWith('/auth/config')) return ok({ email_code_login_enabled: true, google_login_enabled: false })
    if (url.pathname.includes('/announcements')) return ok([])
    if (url.pathname.endsWith('/market/status')) return ok({ is_trading: options.session === 'morning', is_trading_day: true, session: options.session || 'call_auction', current_time: '2026-11-03T09:20:00+08:00', current_date: '2026-11-03', display_date: '2026-11-03', previous_trading_day: '2026-11-02', last_trading_day: '2026-11-02', next_trading_day: '2026-11-04', time_until_next_session_seconds: 600, time_until_next_transition_seconds: 600, timezone: 'Asia/Shanghai', covered_years: [2026] })
    if (url.pathname.endsWith('/user/watchlist/groups')) return ok([{ id: 'group', name: '自选基金', funds: Object.keys(names).map(id => ({ fund_id: id, fund: fund(id) })) }])
    if (url.pathname.endsWith('/fund/search')) return ok(Object.keys(names).map(fund))
    const match = url.pathname.match(/\/fund\/(\d{6})(?:\/(.+))?$/)
    if (match) {
      const [, id, endpoint] = match
      if (endpoint === 'holdings') return ok({ fund: fund(id), holdings: [], display_items: [] })
      if (endpoint === 'history') {
        if (options.delayedSecondFund && id === '003095') await new Promise(resolve => setTimeout(resolve, 1400))
        if (historyError) return route.fulfill({ status: 503, json: { success: false, error: { message: '历史数据服务暂不可用' } } })
        return ok({ fund_id: id, days: 30, points: options.emptyHistory ? [] : [
          { fund_id: id, date: '2026-10-30', net_asset_val: id === '005827' ? '1.4' : '2.1', daily_return: '1' },
          { fund_id: id, date: '2026-11-02', net_asset_val: id === '005827' ? '1.5' : '2.2', daily_return: '2' },
        ] })
      }
      if (endpoint === 'dashboard') return ok({ fund: fund(id), estimate: { fund_id: id, fund_name: names[id], change_percent: '9.99', estimate_nav: '1.6', prev_nav: '1.5' }, time_series: [], session: 'call_auction' })
      if (endpoint === 'timeseries') return ok({ points: [] })
      if (!endpoint) return ok(fund(id))
    }
    return ok({})
  })
  return { requested, recover: () => { historyError = false } }
}

test('auction keeps fund name and code while hiding intraday estimates', async ({ page }) => {
  await mockAuction(page)
  await page.goto('/?fund=005827')
  await expect(page.getByRole('heading', { name: names['005827'], exact: true })).toBeVisible()
  await expect(page.getByText('基金代码：005827', { exact: true })).toBeVisible()
  await expect(page.getByText('等待开盘', { exact: true })).toBeVisible()
  await expect(page.getByText('+9.99%', { exact: true })).toHaveCount(0)
})

test('auction still requests and displays published official NAV history', async ({ page }) => {
  const { requested } = await mockAuction(page)
  await page.goto('/?fund=005827')
  await expect(page.getByText('近 30 日官方净值', { exact: true })).toBeVisible()
  await expect(page.getByText('区间变化', { exact: true })).toBeVisible()
  expect(requested).toContain('/api/v1/fund/005827/history')
  await expect(page.getByText('官方净值公布后会自动补上。', { exact: true })).toHaveCount(0)
})

test('minimal auction cards retain distinct fund identities', async ({ page }) => {
  await mockAuction(page, { minimal: true })
  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto('/')
  for (const [id, name] of Object.entries(names)) {
    await expect(page.getByRole('heading', { name, exact: true })).toBeVisible()
    await expect(page.getByText('基金代码：' + id, { exact: true })).toBeVisible()
  }
  expect(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth)).toBe(false)
})

test('history failure is recoverable and never described as unpublished NAV', async ({ page }) => {
  const pageErrors: string[] = []
  page.on('pageerror', error => pageErrors.push(error.message))
  const mock = await mockAuction(page, { historyError: true })
  await page.goto('/?fund=005827')
  await expect(page.getByText('历史净值暂时无法读取', { exact: true })).toBeVisible()
  await expect(page.getByText('官方净值公布后会自动补上。', { exact: true })).toHaveCount(0)
  const failedRetry = page.waitForResponse(response => response.url().includes('/history?') && response.status() === 503)
  await page.getByRole('button', { name: '重试历史净值' }).click()
  await failedRetry
  await expect(page.getByText('历史净值暂时无法读取', { exact: true })).toBeVisible()
  mock.recover()
  await page.getByRole('button', { name: '重试历史净值' }).click()
  await expect(page.getByText('区间变化', { exact: true })).toBeVisible()
  expect(pageErrors).toEqual([])
})

test('empty history describes missing collected records without implying no publication', async ({ page }) => {
  await mockAuction(page, { emptyHistory: true })
  await page.goto('/?fund=005827')
  await expect(page.getByText('暂无已收录的官方净值记录', { exact: true })).toBeVisible()
  await expect(page.getByText('近 30 日官方净值', { exact: true })).toBeVisible()
})

test('switching funds during auction cannot show the previous fund history', async ({ page }) => {
  await mockAuction(page, { authenticated: true, delayedSecondFund: true })
  await page.goto('/?fund=005827')
  await expect(page.getByText('+7.14%', { exact: true })).toBeVisible()
  await page.getByRole('button', { name: '下一只', exact: true }).click()
  await expect(page.getByRole('heading', { name: names['003095'], exact: true })).toBeVisible()
  await expect(page.getByText('历史净值正在同步', { exact: true })).toBeVisible()
  await expect(page.getByText('+7.14%', { exact: true })).toHaveCount(0)
  await expect(page.getByText('+4.76%', { exact: true })).toBeVisible()
})

test('open trading still shows intraday estimates alongside official history', async ({ page }) => {
  await mockAuction(page, { session: 'morning' })
  await page.goto('/?fund=005827')
  await expect(page.getByRole('heading', { name: names['005827'], exact: true })).toBeVisible()
  await expect(page.getByText('+9.99%', { exact: true })).toBeVisible()
  await expect(page.getByText('近 30 日官方净值', { exact: true })).toBeVisible()
})

test('auction identity and history remain readable on desktop and mobile', async ({ page }, testInfo) => {
  await mockAuction(page)
  await page.emulateMedia({ reducedMotion: 'reduce' })
  for (const [width, theme] of [[1440, 'classic'], [390, 'dark']] as const) {
    await page.addInitScript(value => localStorage.setItem('fund-ui-preferences-v1', JSON.stringify({ themeType: value, viewMode: 'professional' })), theme)
    await page.setViewportSize({ width, height: 900 })
    await page.goto('/?fund=005827')
    await expect(page.getByRole('heading', { name: names['005827'], exact: true })).toBeVisible()
    await expect(page.getByText('区间变化', { exact: true })).toBeVisible()
    expect(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth)).toBe(false)
    await page.getByText('近 30 日官方净值', { exact: true }).scrollIntoViewIfNeeded()
    const historyPanel = page.getByText('近 30 日官方净值', { exact: true }).locator('xpath=ancestor::section[1]')
    await expect(historyPanel.locator('.recharts-area-curve')).toBeVisible()
    // Recharts draws the curve over its configured 700 ms animation.
    await page.waitForTimeout(800)
    await historyPanel.screenshot({ path: testInfo.outputPath(`history-${theme}-${width}.png`), animations: 'disabled' })
    await page.evaluate(() => window.scrollTo(0, 0))
    await page.screenshot({ path: testInfo.outputPath(`auction-${theme}-${width}.png`), fullPage: true, animations: 'disabled' })
  }
})
