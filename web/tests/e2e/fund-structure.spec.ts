import { expect, test, type Page } from '@playwright/test'
import { readabilityIssues } from '../helpers/readability'

async function mockStructure(page: Page, options: { admin?: boolean; empty?: boolean; fail?: boolean; saveError?: boolean } = {}) {
  const saves: unknown[] = []
  const fund = { id: '005827', name: '测试均衡成长混合', type: 'hybrid', category_name: '混合型', manager: '测试经理', company: '测试公司', net_asset_val: '1.25', updated_at: '2026-11-02T15:00:00Z' }
  const sector = { fund_id: fund.id, as_of_date: '2026-09-30', primary_sector_code: 'manual', primary_sector_name: '人工消费标注', source: 'target_etf_fallback', confidence: 'low', breakdown: [{ sector_code: 'food', sector_name: '食品饮料', weight_percent: '18.6', rank: 1 }, { sector_code: 'internet', sector_name: '互联网平台', weight_percent: '12.4', rank: 2 }, { sector_code: 'health', sector_name: '医疗服务', weight_percent: '8.1', rank: 3 }] }
  const theme = { fund_id: fund.id, as_of_date: '2026-06-30', primary_theme_code: 'consumer', primary_theme_name: '消费', source: 'direct_holdings', confidence: 'medium', breakdown: [{ theme_code: 'consumer', theme_name: '消费', weight_percent: '14', rank: 1 }, { theme_code: 'health', theme_name: '医疗健康', weight_percent: '8', rank: 2 }] }
  await page.route('**/api/v1/**', async route => {
    const url = new URL(route.request().url())
    const path = url.pathname
    const ok = (data: unknown) => route.fulfill({ json: { success: true, data } })
    if (path.endsWith('/auth/me')) return ok({ user: { id: 'structure-user', display_name: '测试用户', email: 'test@example.com', is_admin: options.admin ?? false }, expires_at: '2099-01-01T00:00:00Z' })
    if (path.endsWith('/auth/config')) return ok({ email_code_login_enabled: true })
    if (path.includes('/announcements') || path.endsWith('/user/watchlist/groups')) return ok([])
    if (path.endsWith('/market/status')) return ok({ is_trading: false, is_trading_day: true, session: 'after_hours', current_time: '2026-11-03T16:00:00+08:00', current_date: '2026-11-03', timezone: 'Asia/Shanghai' })
    if (path.endsWith('/classification-options')) return ok({ categories: [{ code: 'hybrid', name: '混合型' }], sectors: [{ code: 'manual', name: '人工消费标注' }, { code: 'food', name: '食品饮料' }], themes: [{ code: 'consumer', name: '消费' }] })
    if (path.endsWith('/classification') && route.request().method() === 'PUT') {
      saves.push(route.request().postDataJSON())
      if (options.saveError) return route.fulfill({ status: 503, json: { success: false, error: { message: '保存分类失败，请重试' } } })
      return ok({ fund_id: fund.id })
    }
    if (path.endsWith('/dashboard')) {
      if (options.fail) return route.fulfill({ status: 503, json: { success: false, error: { message: '资料读取失败' } } })
      return ok({ fund, estimate: { fund_id: fund.id, fund_name: fund.name, change_percent: '1', estimate_nav: '1.26', prev_nav: '1.25', holdings: [] }, sector_snapshot: options.empty ? undefined : sector, theme_snapshot: options.empty ? undefined : theme, classification_override: options.empty ? undefined : { fund_id: fund.id, primary_sector_code: 'manual', primary_sector_name: '人工消费标注', manual_tags: ['人工关注'], note: '跨行业的补充说明', updated_at: '2026-11-02T08:00:00Z' }, time_series: [] })
    }
    if (path.endsWith('/holdings')) return ok({ fund, holdings: [] })
    if (path.endsWith('/history')) return ok({ fund_id: fund.id, days: 30, points: [] })
    if (path.endsWith('/timeseries')) return ok({ points: [] })
    if (path.endsWith('/fund/005827')) return ok(fund)
    return ok({})
  })
  await page.addInitScript(() => localStorage.setItem('fund-ui-preferences-v1', JSON.stringify({ themeType: 'classic', viewMode: 'professional' })))
  return saves
}

test('industry rows retain original percentages and target ETF source without manual label substitution', async ({ page }) => {
  await mockStructure(page)
  await page.goto('/?fund=005827')
  const structure = page.getByRole('region', { name: '基金持仓结构', exact: true })
  await expect(structure).toBeVisible()
  await expect(structure.getByText('18.6%', { exact: true })).toBeVisible()
  await expect(structure.getByText('39.1%', { exact: true })).toBeVisible()
  await expect(structure.getByText(/目标 ETF 披露口径/)).toBeVisible()
  await expect(structure.getByText('人工消费标注', { exact: true })).toHaveCount(0)
  await expect(structure.getByTestId('industry-bar').first()).toHaveAttribute('style', /width: 18.6%/)
  await expect(structure.getByRole('button', { name: '编辑人工标注' })).toHaveCount(0)
})

test('theme and manual views are keyboard reachable with independent snapshot dates', async ({ page }) => {
  await mockStructure(page)
  await page.goto('/?fund=005827')
  const structure = page.getByRole('region', { name: '基金持仓结构', exact: true })
  await structure.getByRole('tab', { name: '行业分布' }).focus()
  await page.keyboard.press('ArrowRight')
  await expect(structure.getByRole('tab', { name: '相关主题' })).toHaveAttribute('aria-selected', 'true')
  await expect(structure.getByText(/主题快照 2026-06-30/)).toBeVisible()
  await structure.getByText('查看主题权重与规则', { exact: true }).click()
  await expect(structure.getByRole('tabpanel').getByText(/分摊/)).toBeVisible()
  await structure.getByRole('tab', { name: '相关主题' }).focus()
  await page.keyboard.press('End')
  await expect(structure.getByText('人工消费标注', { exact: true })).toBeVisible()
  await expect(structure.getByText('跨行业的补充说明', { exact: true })).toBeVisible()
  await expect(structure.getByText('食品饮料', { exact: true })).toBeVisible()
})

test('admin editing retains update contract, drafts and server save failures', async ({ page }) => {
  const saves = await mockStructure(page, { admin: true, saveError: true })
  await page.goto('/?fund=005827')
  const structure = page.getByRole('region', { name: '基金持仓结构', exact: true })
  await structure.getByRole('tab', { name: '人工标注' }).click()
  await structure.getByRole('button', { name: '编辑人工标注', exact: true }).click()
  await structure.getByLabel('人工标签（逗号分隔）').fill('消费，长期观察')
  await structure.getByRole('button', { name: '保存人工标注', exact: true }).click()
  await expect(structure.getByText('保存分类失败，请重试', { exact: true })).toBeVisible()
  await expect(structure.getByLabel('人工标签（逗号分隔）')).toHaveValue('消费，长期观察')
  expect(saves).toEqual([{ category_code: '', primary_sector_code: 'manual', primary_theme_code: '', manual_tags: ['消费', '长期观察'], note: '跨行业的补充说明' }])
})

test('no structure and unavailable sources do not silently disappear', async ({ page }) => {
  await mockStructure(page, { empty: true })
  await page.goto('/?fund=005827')
  await expect(page.getByRole('region', { name: '基金持仓结构', exact: true })).toBeVisible()
  await expect(page.getByText('暂无已归类的行业持仓', { exact: true })).toBeVisible()
})

test('classification sources failures are separate from an empty disclosure', async ({ page }) => {
  await mockStructure(page, { fail: true })
  await page.goto('/?fund=005827')
  await expect(page.getByRole('region', { name: '基金持仓结构', exact: true }).getByRole('alert')).toContainText('持仓结构暂时无法读取')
})

test('paper and dark structures fit mobile and preserve readable weights', async ({ page }, testInfo) => {
  await mockStructure(page)
  await page.emulateMedia({ reducedMotion: 'reduce' })
  for (const theme of ['classic', 'dark', 'cyber']) {
    for (const width of [1440, 390]) {
      await page.addInitScript(value => localStorage.setItem('fund-ui-preferences-v1', JSON.stringify({ themeType: value, viewMode: 'professional' })), theme)
      await page.setViewportSize({ width, height: 1000 })
      await page.goto('/?fund=005827')
      const structure = page.getByRole('region', { name: '基金持仓结构', exact: true })
      await expect(structure.getByText('18.6%', { exact: true })).toBeVisible()
      expect(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth)).toBe(false)
      await structure.screenshot({ path: testInfo.outputPath(`structure-${theme}-${width}.png`) })
      await expect.poll(() => readabilityIssues(page, '.fund-structure h2, .fund-structure p, .fund-structure dt, .fund-structure dd, .fund-structure button')).toEqual([])
      await page.evaluate(() => window.scrollTo({ top: 0, behavior: 'instant' }))
      await page.screenshot({ path: testInfo.outputPath(`home-${theme}-${width}.png`), fullPage: true })
    }
  }
})


test('successful admin save closes editing and keeps raw industry weights intact', async ({ page }) => {
  const saves = await mockStructure(page, { admin: true })
  await page.goto('/?fund=005827')
  const structure = page.getByRole('region', { name: '基金持仓结构', exact: true })
  await structure.getByRole('tab', { name: '人工标注' }).click()
  await structure.getByRole('button', { name: '编辑人工标注', exact: true }).click()
  await expect(structure.getByRole('button', { name: '保存人工标注', exact: true })).toBeEnabled()
  await structure.getByRole('button', { name: '保存人工标注', exact: true }).click()
  await expect(structure.getByLabel('人工标签（逗号分隔）')).toHaveCount(0)
  expect(saves).toHaveLength(1)
  await structure.getByRole('tab', { name: '行业分布' }).click()
  await expect(structure.getByText('18.6%', { exact: true })).toBeVisible()
})
