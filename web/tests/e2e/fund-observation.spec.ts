import { expect, test, type Page } from '@playwright/test'
import path from 'node:path'
import type { FundNotice } from '../../src/hooks/use-fund-observation'

async function captureObservation(page: Page, name: string) {
  if (process.env.OBSERVATION_SCREENSHOT_DIR) {
    await page.screenshot({ path: path.join(process.env.OBSERVATION_SCREENSHOT_DIR, `${name}.png`), fullPage: true })
  }
}

const fullTitle = '贵州茅台：2026年第三季度报告及董事会审议事项的公告，含完整披露说明'

function observation(id = '005827') {
  return {
    fund: { id, name: id === '005827' ? '易方达蓝筹精选混合' : '中欧医疗健康混合', type: '混合型', manager: '', company: '测试基金公司' },
    observation: {
      checked_at: '2026-10-16T02:50:00Z', window_days: 45,
      latest_holding_period: '2026-06-30', holding_age_days: 108 as number | null,
      disclosed_weight_percent: '39.12', holdings_count: 10,
      holdings_source: '基金定期报告', holdings_status: 'available',
      notices: [
        { id: 'fund-1', scope: 'fund', title: '关于基金经理变更的公告', published_at: '2026-10-15T16:30:00Z', source_name: '基金公司', source_url: 'https://example.com/fund-notice' },
        { id: 'holding-1', scope: 'holding', title: fullTitle, published_at: '2026-10-15T00:00:00Z', source_name: '交易所', source_url: 'https://example.com/holding-notice', related_symbol: '600519', related_name: '贵州茅台' },
      ] as FundNotice[],
      sources: [
        { code: 'fund_notices', status: 'ok', checked_at: '2026-10-16T02:50:00Z' },
        { code: 'holding_notices', status: 'ok', checked_at: '2026-10-16T02:50:00Z' },
      ],
      notes: ['持仓来自定期披露，不代表基金当前全部持仓。'],
    },
  }
}

type ObservationPayload = ReturnType<typeof observation>

async function mockApi(page: Page, getObservation: (id: string) => ObservationPayload | null | Promise<ObservationPayload | null>) {
  const requestedPaths: string[] = []
  await page.route('**/api/v1/**', async (route) => {
    const url = new URL(route.request().url())
    requestedPaths.push(url.pathname)
    const match = url.pathname.match(/\/fund\/(\d+)\/observation$/)
    if (match) {
      const payload = await getObservation(match[1])
      await route.fulfill({ status: payload ? 200 : 503, json: payload ? { success: true, data: payload } : { success: false, error: { message: '公告服务暂时不可用' } } })
      return
    }
    if (url.pathname.endsWith('/fund/search')) {
      await route.fulfill({ json: { success: true, data: [{ id: '003095', name: '中欧医疗健康混合' }] } })
      return
    }
    if (url.pathname.endsWith('/auth/me')) {
      await route.fulfill({ status: 401, json: { success: false } })
      return
    }
    await route.fulfill({ json: { success: true, data: null } })
  })
  return requestedPaths
}

test.describe('fund observation', () => {
  test('shows complete dated notices, source links and disclosure context without ratings', async ({ page }) => {
    const paths = await mockApi(page, () => observation())
    await page.goto('/observations/005827')
    const main = page.getByRole('main')
    await expect(main.getByRole('heading', { name: '易方达蓝筹精选混合' })).toBeVisible()
    await expect(main.getByRole('link', { name: fullTitle })).toHaveAttribute('href', 'https://example.com/holding-notice')
    await expect(main.getByRole('link', { name: '关于基金经理变更的公告' })).toBeVisible()
    await expect(main.getByText('2026-10-16', { exact: true })).toBeVisible()
    await expect(main.getByText('2026-06-30', { exact: true })).toBeVisible()
    await expect(main.getByText('39.12%', { exact: true })).toBeVisible()
    await expect(main.getByText('10 项', { exact: true })).toBeVisible()
    await expect(main.getByText('基金定期报告', { exact: true })).toBeVisible()
    await expect(main.getByRole('link', { name: '返回基金详情' })).toHaveAttribute('href', '/?fund=005827')
    await captureObservation(page, 'detail-desktop')
    await main.getByRole('button', { name: '基金公告', exact: true }).click()
    await expect(main.getByRole('link', { name: fullTitle })).toHaveCount(0)
    await expect(main.getByRole('link', { name: '关于基金经理变更的公告' })).toBeVisible()
    await main.getByRole('button', { name: '持仓公告', exact: true }).click()
    await expect(main.getByRole('link', { name: fullTitle })).toBeVisible()
    await expect(main.getByText(/综合评分|胜率|增持|减持|AI 解读|雷达/)).toHaveCount(0)
    expect(paths.some((path) => /analysis|rankings|dashboard/.test(path))).toBe(false)
  })

  test('separates no recent notices from unavailable sources and missing holdings', async ({ page }) => {
    const payload = observation()
    payload.observation.notices = []
    payload.observation.holdings_status = 'missing'
    payload.observation.holdings_count = 0
    payload.observation.holding_age_days = null
    payload.observation.latest_holding_period = ''
    payload.observation.sources[1].status = 'not_applicable'
    await mockApi(page, () => payload)
    await page.goto('/observations/005827')
    await expect(page.getByText('近 45 天暂无公告', { exact: true })).toBeVisible()
    await expect(page.getByText('尚无公开持仓披露，仍可查看基金公告。')).toBeVisible()
    await page.getByRole('button', { name: '持仓公告', exact: true }).click()
    await expect(page.getByText('当前披露范围没有可关联的公告来源', { exact: true })).toBeVisible()
    await expect(page.getByText('近 45 天暂无持仓公告', { exact: true })).toHaveCount(0)
    await page.getByRole('button', { name: '全部', exact: true }).click()
    payload.observation.sources[0].status = 'unavailable'
    await page.getByRole('button', { name: '刷新公告', exact: true }).click()
    await expect(page.getByText('公告暂时无法完整读取', { exact: true })).toBeVisible()
    await expect(page.getByText('近 45 天暂无公告', { exact: true })).toHaveCount(0)
  })

  test('identifies target ETF codes and labels list links as source pages', async ({ page }) => {
    const payload = observation()
    payload.observation.notices = [{
      id: 'target-etf-1', scope: 'target_etf', title: '目标 ETF 公告列表',
      published_at: '2026-10-15', source_name: '东方财富基金公告',
      source_url: 'https://fundf10.eastmoney.com/jjgg_510300.html', related_symbol: '510300',
    }]
    await mockApi(page, () => payload)
    await page.goto('/observations/005827')
    const main = page.getByRole('main')
    await expect(main.getByText('510300', { exact: true })).toBeVisible()
    await expect(main.getByText('近 45 天 · 按发布日期排列 · 点击标题查看来源页面')).toBeVisible()
    await expect(main.getByRole('link', { name: '目标 ETF 公告列表' })).toHaveAttribute('href', 'https://fundf10.eastmoney.com/jjgg_510300.html')
    await expect(main.getByText(/打开原文|查看原文/)).toHaveCount(0)
  })

  test('labels quarter disclosures and target ETF coverage without leaking source enums', async ({ page }) => {
    const payload = observation()
    payload.observation.latest_holding_period = '2026Q2'
    payload.observation.holdings_source = 'target_etf_fallback'
    await mockApi(page, () => payload)
    await page.goto('/observations/005827')
    await expect(page.getByText('2026 年第 2 季度', { exact: true })).toBeVisible()
    await expect(page.getByText('目标 ETF 公开持仓', { exact: true })).toBeVisible()
    await expect(page.getByText('target_etf_fallback')).toHaveCount(0)
    payload.observation.latest_holding_period = '2026-Q3'
    payload.observation.holdings_source = 'direct_holdings'
    await page.getByRole('button', { name: '刷新公告', exact: true }).click()
    await expect(page.getByText('2026 年第 3 季度', { exact: true })).toBeVisible()
    await expect(page.getByText('基金公开持仓', { exact: true })).toBeVisible()
    payload.observation.notices = []
    payload.observation.sources = []
    await page.getByRole('button', { name: '刷新公告', exact: true }).click()
    await expect(page.getByText('公告暂时无法完整读取', { exact: true })).toBeVisible()
    await expect(page.getByText('近 45 天暂无公告', { exact: true })).toHaveCount(0)
  })

  test('makes partial data explicit and retries failed requests', async ({ page }) => {
    let fail = true
    const payload = observation()
    payload.observation.sources[1].status = 'partial'
    await mockApi(page, () => fail ? null : payload)
    await page.goto('/observations/005827')
    await expect(page.getByText('公告服务暂时不可用', { exact: true })).toBeVisible()
    fail = false
    await page.getByRole('button', { name: '重试', exact: true }).click()
    await expect(page.getByRole('link', { name: fullTitle })).toBeVisible()
    await expect(page.getByText('部分可用', { exact: true })).toBeVisible()
  })

  test('switches funds without showing the previous fund notices while loading', async ({ page }) => {
    let resolveSecond!: (value: ObservationPayload) => void
    const second = new Promise<ObservationPayload>((resolve) => { resolveSecond = resolve })
    await mockApi(page, (id) => id === '003095' ? second : observation())
    await page.goto('/observations/005827')
    await expect(page.getByRole('link', { name: fullTitle })).toBeVisible()
    await page.getByRole('button', { name: '搜索基金代码或名称' }).first().click()
    const search = page.getByRole('dialog', { name: '搜索基金' })
    await search.getByRole('textbox').fill('中欧')
    await search.getByRole('button', { name: /中欧医疗健康混合/ }).click()
    await expect(page).toHaveURL('/observations/003095')
    await expect(page.getByRole('link', { name: fullTitle })).toHaveCount(0)
    resolveSecond({ ...observation('003095'), observation: { ...observation('003095').observation, notices: [] } })
    await expect(page.getByRole('heading', { name: '中欧医疗健康混合' })).toBeVisible()
    await expect(page.getByText('近 45 天暂无公告', { exact: true })).toBeVisible()
  })

  test('starts with search instead of sample rankings and retains legacy links', async ({ page }) => {
    const paths = await mockApi(page, () => observation())
    await page.goto('/analysis/rankings')
    await expect(page).toHaveURL('/observations')
    await expect(page.getByRole('heading', { name: '基金观察', exact: true })).toBeVisible()
    await expect(page.getByRole('heading', { name: '选择要观察的基金' })).toBeVisible()
    await captureObservation(page, 'index-desktop')
    expect(paths.filter((path) => path.endsWith('/observation'))).toEqual([])
    await page.goto('/analysis/005827')
    await expect(page).toHaveURL('/observations/005827')
    await expect(page.getByRole('link', { name: fullTitle })).toBeVisible()
  })

  test('keeps mobile navigation, complete titles and filters usable in every theme', async ({ page }) => {
    await mockApi(page, () => observation())
    await page.setViewportSize({ width: 360, height: 800 })
    for (const themeType of ['classic', 'dark', 'cyber']) {
      await page.addInitScript((theme) => localStorage.setItem('fund-ui-preferences-v1', JSON.stringify({ themeType: theme, viewMode: 'professional' })), themeType)
      await page.goto('/observations/005827')
      await expect(page.locator('html')).toHaveAttribute('data-theme', themeType)
      await expect(page.getByRole('link', { name: fullTitle })).toBeVisible()
      const navigation = page.getByRole('navigation', { name: '主要页面' })
      await expect(navigation.getByRole('link', { name: '观察', exact: true })).toHaveAttribute('aria-current', 'page')
      await expect(page.getByRole('button', { name: '持仓公告', exact: true })).toBeVisible()
      expect(await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth)).toBe(false)
      await captureObservation(page, `detail-mobile-${themeType}`)
      const height = await page.getByRole('button', { name: '持仓公告', exact: true }).evaluate((element) => element.getBoundingClientRect().height)
      expect(height).toBeGreaterThanOrEqual(44)
    }
  })
})
