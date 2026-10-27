import { expect, test, type Page, type Route } from '@playwright/test'
import type { WorkspaceConfig, WorkspaceResult } from '../../src/lib/market-workspace-types'
import type { ResearchInstrument } from '../../src/hooks/use-market-research'

const key = 'fund-live.market-workspaces.v1'
const meta = { source: '测试同源日线', fetched_at: '2026-10-09T16:00:00+08:00', stale: false }
const instruments: ResearchInstrument[] = [{ symbol: 'sh512480', name: '半导体ETF', group: '半导体' }, { symbol: 'sh000688', name: '科创50', group: '大盘' }, { symbol: 'sh512890', name: '红利低波ETF', group: '红利' }]

function result(config: WorkspaceConfig): WorkspaceResult {
  return {
    generated_at: meta.fetched_at, sources: config.sources.map(source => ({ alias: source.alias, symbol: source.symbol, meta })), warnings: [],
    rows: ['2026-10-08', '2026-10-09'].map((date, index) => ({ date, state: index === 1, signal: index === 1, values: Object.fromEntries(config.curves.map((curve, i) => [curve.id, 100 + index + i])) })),
    study: { rule: 'workspace', symbol: config.target_symbol, direction: config.event.direction, holding: config.holding, fee_bps: config.fee_bps, sample_count: 1, pending_count: 2, direction_win_rate: 100, mean_return_pct: 2, mean_net_return_pct: config.event.direction === 'down' ? null : 1.9, worst_drawdown_pct: 4, baseline_mean_return_pct: 1, last_completed_date: '2026-10-09', events: [{ signal_date: '2026-09-01', entry_date: '2026-09-02', exit_date: '2026-09-09', return_pct: 2, direction_pct: 2, net_return_pct: 1.9, drawdown_pct: 4 }], history_meta: meta, method: '信号后下一交易日收盘进入；样本不是未来概率。' },
  }
}
async function fulfill(route: Route, data: unknown) { await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ success: true, data }) }) }
async function setup(page: Page) {
  const state = { requests: [] as WorkspaceConfig[], fail: false, empty: false, delay: 0 }
  await page.route('**/api/v1/**', async route => {
    const url = new URL(route.request().url())
    if (url.pathname.endsWith('/auth/me')) { await route.fulfill({ status: 401, contentType: 'application/json', body: '{}' }); return }
    if (url.pathname.endsWith('/auth/config')) { await fulfill(route, { google_login_enabled: false, email_code_login_enabled: true }); return }
    if (url.pathname.endsWith('/market/status')) { await fulfill(route, { is_trading: false, is_trading_day: true, session: 'after_hours', current_time: meta.fetched_at, current_date: '2026-10-09', timezone: 'Asia/Shanghai' }); return }
    if (url.pathname.endsWith('/research/overview')) {
      await fulfill(route, { generated_at: meta.fetched_at, market_phase: 'after_hours', instruments, quotes: instruments.map(instrument => ({ ...instrument, price: 1, prev_close: 1, open: 1, high: 1, low: 1, change_pct: 0, timestamp: meta.fetched_at, delayed: false })), quote_meta: meta, breadth: null, breadth_meta: meta, news: [], news_meta: meta }); return
    }
    if (url.pathname.endsWith('/research/instrument')) {
      const instrument = instruments.find(item => item.symbol === url.searchParams.get('symbol')) || instruments[0]
      await fulfill(route, { instrument, quote: null, quote_meta: meta, bars: [], history_meta: meta, technical: null, minute: null, minute_meta: meta }); return
    }
    if (url.pathname.endsWith('/research/study')) { await fulfill(route, { ...result({ sources: [], curves: [], target_symbol: url.searchParams.get('symbol') || 'sh512480', event: { direction: 'up', mode: 'all', conditions: [] }, holding: 10, fee_bps: 10, limit: 120, name: 'test' }).study, rule: 'ma20_up', events: [] }); return }
    if (url.pathname.endsWith('/market/workspace')) {
      const config = route.request().postDataJSON() as WorkspaceConfig
      state.requests.push(config)
      if (state.delay) await new Promise(resolve => setTimeout(resolve, state.delay))
      if (state.fail) { await route.fulfill({ status: 503, contentType: 'application/json', body: JSON.stringify({ success: false, error: { message: '测试公式服务离线' } }) }); return }
      const data = result(config)
      if (state.empty) { data.rows = []; data.study.events = []; data.study.sample_count = 0; data.study.direction_win_rate = null }
      await fulfill(route, data); return
    }
    await fulfill(route, { generated_at: meta.fetched_at, sectors: [], instruments: [], meta, method: '数据商估算' })
  })
  return state
}
function workspace(page: Page) { return page.getByRole('region', { name: '自定义公式工作区' }) }

test.describe('formula workspace', () => {
  test('manual calculation applies sources, curves, event and research parameters; errors retain result', async ({ page }) => {
    const state = await setup(page)
    await page.goto('/research')
    const area = workspace(page)
    await expect(area).toBeVisible()
    expect(state.requests).toHaveLength(0)
    await area.getByLabel('数据源A标的').selectOption('sh512890')
    await area.getByLabel('曲线1公式').fill('A/MA(A,20)*100')
    await area.getByLabel('条件1比较方式').selectOption('cross_down')
    await area.getByLabel('条件组合').selectOption('any')
    await area.getByLabel('研究方向').selectOption('down')
    await area.getByLabel('公式持有期').fill('7')
    await area.getByLabel('公式总交易费用').fill('25')
    await area.getByLabel('公式显示范围').selectOption('60')
    await area.getByRole('button', { name: '应用参数并计算' }).click()
    await expect(area.getByRole('table').filter({ has: page.getByText('最新事件状态', { exact: true }) }).getByText('已触发', { exact: true })).toBeVisible()
    await expect(area.locator('[aria-label="公式原始走势"]')).toBeVisible()
    expect(state.requests[0]).toMatchObject({ target_symbol: 'sh512890', sources: [{ alias: 'A', symbol: 'sh512890', field: 'close' }, { alias: 'B', symbol: 'sh000688', field: 'close' }], event: { mode: 'any', direction: 'down', conditions: [{ left: 'A', operator: 'cross_down', right: 'MA(A,5)' }, { left: 'B', operator: 'gt', right: 'MA(B,5)' }] }, holding: 7, fee_bps: 25, limit: 60 })
    await expect(area.getByText('不适用', { exact: true })).toBeVisible()
    await area.getByLabel('公式显示方式').selectOption('normalized')
    await expect(area.locator('[aria-label="公式归一化走势"]')).toBeVisible()
    await area.getByLabel('曲线1公式').fill('ROC(A,5)')
    await expect(area.getByText(/参数已修改/)).toBeVisible()
    state.fail = true
    await area.getByRole('button', { name: '应用参数并计算' }).click()
    await expect(area.getByRole('alert')).toContainText('测试公式服务离线')
    await expect(area.getByRole('alert')).toContainText('已保留上次成功结果')
    await expect(area.locator('[aria-label="公式归一化走势"]')).toBeVisible()
    expect(state.requests).toHaveLength(2)
  })

  test('local named workspaces survive reload; loading precedes update and deleting targets the selected ID', async ({ page }) => {
    await setup(page)
    await page.goto('/research')
    let area = workspace(page)
    await area.getByLabel('方案名称').fill('半导体观察一')
    await area.getByRole('button', { name: '保存为新方案' }).click()
    const firstId = await area.getByLabel('已保存方案').inputValue()
    await area.getByLabel('方案名称').fill('红利观察二')
    await area.getByLabel('曲线1公式').fill('ROC(A,5)')
    await area.getByRole('button', { name: '保存为新方案' }).click()
    const secondId = await area.getByLabel('已保存方案').inputValue()
    expect(secondId).not.toBe(firstId)
    await area.getByLabel('已保存方案').selectOption(firstId)
    await expect(area.getByRole('button', { name: '更新当前方案' })).toBeDisabled()
    await page.reload()
    area = workspace(page)
    await expect(area.getByLabel('已保存方案').locator('option')).toHaveCount(3)
    await area.getByLabel('已保存方案').selectOption(firstId)
    await area.getByRole('button', { name: '载入方案' }).click()
    await expect(area.getByLabel('方案名称')).toHaveValue('半导体观察一')
    await expect(area.getByLabel('曲线1公式')).toHaveValue('A/MA(A,20)*100')
    await area.getByLabel('公式持有期').fill('9')
    await area.getByRole('button', { name: '更新当前方案' }).click()
    const stored = await page.evaluate(storageKey => JSON.parse(localStorage.getItem(storageKey)!), key)
    expect(stored.workspaces.find((entry: { id: string }) => entry.id === firstId).config.holding).toBe(9)
    expect(stored.workspaces.find((entry: { id: string }) => entry.id === secondId).config.holding).toBe(5)
    await area.getByLabel('已保存方案').selectOption(secondId)
    await area.getByRole('button', { name: '删除方案' }).click()
    await expect(area.getByLabel('已保存方案').locator('option')).toHaveCount(2)
    const remaining = await page.evaluate(storageKey => JSON.parse(localStorage.getItem(storageKey)!).workspaces, key)
    expect(remaining.map((entry: { id: string }) => entry.id)).toEqual([firstId])
  })

  test('invalid local storage and JSON import are rejected gracefully; browser quota failure is visible', async ({ page }) => {
    await page.addInitScript(storageKey => localStorage.setItem(storageKey, '{bad'), key)
    await setup(page)
    await page.goto('/research')
    const area = workspace(page)
    await expect(area.getByText(/本地方案无法读取/)).toBeVisible()
    await area.getByText('导入JSON配置', { exact: true }).click()
    await area.getByLabel('导入JSON配置').fill(JSON.stringify({ version: 1, config: { name: 'bad', sources: [{ alias: 'A', symbol: 'untrusted', field: 'close' }] } }))
    await area.getByRole('button', { name: '校验并导入' }).click()
    await expect(area.getByText(/导入失败/)).toBeVisible()
    await expect(area.getByLabel('方案名称')).toHaveValue('半导体趋势观察')
    await page.evaluate(() => { Storage.prototype.setItem = () => { throw new DOMException('QuotaExceeded', 'QuotaExceededError') } })
    await area.getByRole('button', { name: '保存为新方案' }).click()
    await expect(area.getByText(/保存失败/)).toBeVisible()
    await area.getByRole('button', { name: '应用参数并计算' }).click()
    await expect(area.getByRole('heading', { name: '自定义事件历史研究' })).toBeVisible()
  })


  test('automatic refresh uses only submitted parameters and stops when disabled', async ({ page }) => {
    await page.clock.install()
    const state = await setup(page)
    await page.goto('/research')
    const area = workspace(page)
    await area.getByRole('button', { name: '应用参数并计算' }).click()
    await expect(area.getByRole('heading', { name: '自定义事件历史研究' })).toBeVisible()
    await area.getByLabel('公式持有期').fill('8')
    await area.getByRole('checkbox', { name: /自动刷新已应用方案/ }).check()
    await page.clock.fastForward(300_100)
    await expect.poll(() => state.requests.length).toBe(2)
    expect(state.requests[1].holding).toBe(5)
    await expect(area.getByText(/参数已修改/)).toBeVisible()
    await area.getByRole('checkbox', { name: /自动刷新已应用方案/ }).uncheck()
    await page.clock.fastForward(300_100)
    expect(state.requests).toHaveLength(2)
    await area.getByRole('button', { name: '应用参数并计算' }).click()
    await expect.poll(() => state.requests.length).toBe(3)
    expect(state.requests[2].holding).toBe(8)
  })

  test('mobile workspace charts and empty samples remain usable without document overflow', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 })
    const state = await setup(page)
    await page.goto('/research')
    const area = workspace(page)
    await area.getByRole('button', { name: '应用参数并计算' }).click()
    await expect(area.locator('[aria-label="公式原始走势"]')).toBeVisible()
    await area.getByLabel('公式显示方式').selectOption('normalized')
    await expect(area.locator('[aria-label="公式归一化走势"]')).toBeVisible()
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth)
    expect(overflow).toBe(false)
    state.empty = true
    await area.getByRole('button', { name: '应用参数并计算' }).click()
    await expect(area.getByText('暂无公式结果')).toBeVisible()
    await expect(area.getByText('暂无已完成的历史事件样本')).toBeVisible()
    await expect(area.getByRole('table').filter({ has: page.getByText('最新事件状态', { exact: true }) }).getByText('数据不足')).toBeVisible()
  })
})
