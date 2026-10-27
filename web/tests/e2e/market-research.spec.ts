import { expect, test, type Page, type Route } from '@playwright/test'
import type { ResearchBar, ResearchInstrument, ResearchInstrumentDetail, ResearchOverview, ResearchQuote, ResearchStudy } from '../../src/hooks/use-market-research'

const instruments: ResearchInstrument[] = [
  { symbol: 'sh000001', name: '上证指数', group: '大盘' },
  { symbol: 'sz159994', name: '通信ETF银华', group: '通信/CPO' },
  { symbol: 'sh561980', name: '半导体设备ETF招商', group: '半导体' },
  { symbol: 'sz159363', name: '创业板人工智能ETF华宝', group: '人工智能' },
]
const meta = { source: '测试公开源', fetched_at: '2026-10-09T10:10:00+08:00', stale: false }

function quote(instrument: ResearchInstrument): ResearchQuote {
  return { ...instrument, price: instrument.group === '大盘' ? 3780 : 1.09, prev_close: instrument.group === '大盘' ? 3810 : 1.1, open: 1.1, high: 1.12, low: 1.08, change_pct: -0.91, timestamp: '2026-10-09T10:09:00+08:00', delayed: false }
}

function bars(dates = ['2026-09-28', '2026-09-29', '2026-09-30', '2026-10-08']): ResearchBar[] {
  return dates.map((date, index) => ({ date, open: 1 + index / 100, close: 1.02 + index / 100, high: 1.04 + index / 100, low: 0.98 + index / 100, volume: 1000 }))
}

function instrumentDetail(symbol: string): ResearchInstrumentDetail {
  const instrument = instruments.find((item) => item.symbol === symbol)!
  return {
    instrument, quote: quote(instrument), quote_meta: meta, bars: bars(), history_meta: meta,
    technical: { ma5: 1.08, ma20: 1.1, ma60: 1.12, rsi14: 28, previous_low: 1.04, provisional: false, conditions: [{ label: '低于20日均线', detail: '同源前复权日线', tone: 'negative' }] },
    minute: { date: '2026-10-09', points: [{ time: '09:30', price: 1.1 }, { time: '09:35', price: 1.11 }, { time: '10:00', price: 1.09 }] }, minute_meta: meta,
  }
}

function study(url: URL): ResearchStudy {
  const rule = (url.searchParams.get('rule') || 'ma20_up') as ResearchStudy['rule']
  const down = ['ma20_down', 'previous_low_break'].includes(rule)
  return {
    symbol: url.searchParams.get('symbol')!, rule, direction: down ? 'down' : 'up', holding: Number(url.searchParams.get('holding')), fee_bps: Number(url.searchParams.get('fee_bps')),
    sample_count: 1, pending_count: 2, direction_win_rate: 100, mean_return_pct: down ? -2 : 2, mean_net_return_pct: down ? null : 1.9,
    worst_drawdown_pct: 4.5, baseline_mean_return_pct: 0.8, last_completed_date: '2026-10-08', method: '测试方法：信号后下一交易日收盘进入；排除重叠及未完成样本。', history_meta: meta,
    events: [{ signal_date: '2026-09-01', entry_date: '2026-09-02', exit_date: '2026-09-16', return_pct: down ? -2 : 2, direction_pct: 2, net_return_pct: down ? null : 1.9, drawdown_pct: 4.5 }],
  }
}

async function fulfill(route: Route, data: unknown) {
  await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ success: true, data }) })
}

async function mockResearch(page: Page) {
  const state = {
    failOverview: false,
    priceRevision: 0,
    failInstrument: false,
    empty: false,
    stale: false,
    mismatch: false,
    alignMinute: false,
    requestOrigins: [] as string[],
    studies: [] as URL[],
  }
  await page.route('**/api/v1/**', async (route) => {
    const url = new URL(route.request().url())
    if (url.pathname.includes('/market/research/')) state.requestOrigins.push(url.origin)
    if (url.pathname.endsWith('/auth/me')) {
      await route.fulfill({ status: 401, contentType: 'application/json', body: '{}' })
    } else if (url.pathname.endsWith('/auth/config')) {
      await fulfill(route, { google_login_enabled: false, email_code_login_enabled: true })
    } else if (url.pathname.endsWith('/market/status')) {
      await fulfill(route, { is_trading: true, is_trading_day: true, session: 'morning', current_time: '2026-10-09T10:10:00+08:00', current_date: '2026-10-09', display_date: '2026-10-09', previous_trading_day: '2026-10-08', last_trading_day: '2026-10-08', next_trading_day: '2026-10-12', timezone: 'Asia/Shanghai', covered_years: [2026] })
    } else if (url.pathname.endsWith('/research/overview')) {
      if (state.failOverview) {
        await route.fulfill({ status: 503, contentType: 'application/json', body: JSON.stringify({ success: false, error: { message: '测试行情服务离线' } }) })
      } else {
        const staleMeta = state.stale ? { ...meta, stale: true, error: '上游超时，保留历史缓存' } : meta
        const overview: ResearchOverview = {
          generated_at: meta.fetched_at, market_phase: 'morning', instruments, quotes: instruments.map(quote), quote_meta: staleMeta,
          breadth: state.empty ? null : { advancing: 900, declining: 4100, unchanged: 100, turnover_yuan: 560_000_000_000 }, breadth_meta: staleMeta,
          news: state.empty ? [] : [{ title: '测试科技新闻原文', url: 'https://example.com/research-news', date: '2026-10-09', related: true }], news_meta: staleMeta,
        }
        await fulfill(route, overview)
      }
    } else if (url.pathname.endsWith('/research/instrument')) {
      if (state.failInstrument) {
        await route.fulfill({ status: 503, contentType: 'application/json', body: JSON.stringify({ success: false, error: { message: '测试日线服务离线' } }) })
      } else {
        const detail = instrumentDetail(url.searchParams.get('symbol')!)
        if (state.priceRevision && detail.bars.length) {
          const last = detail.bars[detail.bars.length - 1]
          last.close += state.priceRevision
          last.high = Math.max(last.high, last.close)
        }
        if (state.empty) {
          detail.bars = []
          detail.technical = null
          detail.minute = null
        }
        if (state.mismatch) {
          detail.quote!.timestamp = '2026-10-08T23:30:00Z'
          detail.minute!.date = state.alignMinute ? '2026-10-09' : '2026-10-08'
          detail.bars = bars(detail.instrument.symbol === 'sz159994' ? ['2026-09-29', '2026-09-30', '2026-10-08'] : detail.instrument.symbol === 'sh561980' ? ['2026-09-30', '2026-10-08', '2026-10-09'] : ['2026-09-28', '2026-09-30', '2026-10-08'])
        }
        await fulfill(route, detail)
      }
    } else if (url.pathname.endsWith('/research/flows')) {
      await fulfill(route, { rows: [], meta, method: '测试成交分类估算，不代表ETF申赎' })
    } else if (url.pathname.endsWith('/research/study')) {
      state.studies.push(url)
      await fulfill(route, study(url))
    } else {
      await fulfill(route, [])
    }
  })
  return state
}

function section(page: Page, name: string) {
  return page.locator('section').filter({ has: page.getByRole('heading', { name, exact: true }) })
}

function trackConsoleErrors(page: Page) {
  const errors: string[] = []
  page.on('console', (message) => {
    if (message.type() === 'error' && !message.text().startsWith('Failed to load resource:')) errors.push(message.text())
    if (message.type() === 'warning' && /width\(-1\)|height\(-1\)/.test(message.text())) errors.push(message.text())
  })
  return errors
}

test.describe('market research workspace', () => {
  test('works without login and applies instrument, chart and study parameters', async ({ page }) => {
    const consoleErrors = trackConsoleErrors(page)
    const state = await mockResearch(page)
    await page.goto('/research')
    await expect(page.getByRole('heading', { name: '市场研究', exact: true })).toBeVisible()
    await expect(page.getByRole('link', { name: '研究', exact: true })).toHaveAttribute('aria-current', 'page')
    await expect(page.getByText('自动刷新 30 秒')).toBeVisible()
    await expect(page.getByText('正在读取市场状态')).toHaveCount(0)
    await expect(page.getByRole('button', { name: '搜索基金代码或名称' })).toHaveCount(0)
    await expect(section(page, '通信ETF银华').getByRole('heading', { name: '技术指标 · 日线截至 2026-10-08' })).toBeVisible()

    await page.getByLabel('标的分组').selectOption('半导体')
    const quotes = section(page, '标的行情').getByRole('table')
    await expect(quotes.getByRole('button', { name: /半导体设备ETF招商/ })).toBeVisible()
    await expect(quotes.getByRole('button', { name: /通信ETF银华/ })).toHaveCount(0)
    await page.getByLabel('当前研究标的').selectOption('sh561980')
    const detail = section(page, '半导体设备ETF招商')
    await expect(detail).toBeVisible()
    await detail.getByRole('tab', { name: '分时', exact: true }).click()
    await expect(detail.locator('[aria-label="2026-10-09 分时走势"]')).toBeVisible()
    await expect(detail.locator('.market-minute-previous-close')).toHaveCount(1)

    await page.getByLabel('事件规则').selectOption('ma20_down')
    await page.getByLabel('持有期', { exact: true }).fill('5')
    await page.getByLabel('总交易费用', { exact: true }).fill('25')
    await page.getByRole('button', { name: '计算', exact: true }).click()
    await expect.poll(() => state.studies.at(-1)?.searchParams.toString()).toBe('symbol=sh561980&rule=ma20_down&holding=5&fee_bps=25')
    const studySection = section(page, '规则事件研究')
    await expect(studySection.getByText('不适用', { exact: true })).toBeVisible()
    const eventRow = studySection.getByRole('row').filter({ hasText: '2026-09-01' })
    await expect(eventRow.getByRole('cell').nth(5)).toHaveText('--')
    await expect(eventRow.getByRole('cell').nth(6)).toHaveText('4.50%')
    await expect(eventRow.getByRole('cell').nth(6)).toHaveClass(/text-down/)
    await studySection.getByText('研究方法与限制', { exact: true }).click()
    await expect(studySection.getByText(/日线无法确定同一天高低价发生顺序/)).toBeVisible()
    await expect(page.getByRole('link', { name: /测试科技新闻原文/ })).toHaveAttribute('href', 'https://example.com/research-news')
    expect(state.requestOrigins.length).toBeGreaterThan(0)
    expect(new Set(state.requestOrigins)).toEqual(new Set([new URL(page.url()).origin]))
    expect(consoleErrors).toEqual([])
  })

  test('retains successful data and reports overview and instrument refresh failures', async ({ page }) => {
    const consoleErrors = trackConsoleErrors(page)
    const state = await mockResearch(page)
    await page.goto('/research')
    await expect(section(page, '通信ETF银华').getByText('1.090', { exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: '刷新研究数据', exact: true })).toBeEnabled()
    state.failOverview = true
    state.failInstrument = true
    await page.getByRole('button', { name: '刷新研究数据', exact: true }).click()
    await expect(page.getByText(/测试行情服务离线；当前保留上次成功数据/)).toBeVisible()
    await expect(section(page, '通信ETF银华').getByText(/测试日线服务离线；保留上次成功数据/)).toBeVisible()
    await expect(section(page, '通信ETF银华').getByText('1.090', { exact: true })).toBeVisible()
    await expect(page.getByRole('heading', { name: '大盘概览', exact: true })).toBeVisible()
    expect(consoleErrors).toEqual([])
  })

  test('exposes expired source metadata and meaningful empty states', async ({ page }) => {
    const consoleErrors = trackConsoleErrors(page)
    const state = await mockResearch(page)
    state.empty = true
    state.stale = true
    await page.goto('/research')
    await expect(page.getByText(/部分上游数据已过期/)).toBeVisible()
    await expect(page.getByText('缓存已过期').first()).toBeVisible()
    await expect(page.getByText('上游超时，保留历史缓存').first()).toBeVisible()
    await expect(page.getByText('市场宽度暂不可用')).toBeVisible()
    await expect(page.getByText('暂无日线数据')).toBeVisible()
    await expect(page.getByText('暂无可读取的消息原文')).toBeVisible()
    await page.getByRole('tab', { name: '分时', exact: true }).click()
    await expect(page.getByText('暂无分时数据')).toBeVisible()
    expect(consoleErrors).toEqual([])
  })

  test('keeps date-mismatched prices separate and comparison dates aligned', async ({ page }) => {
    const consoleErrors = trackConsoleErrors(page)
    const state = await mockResearch(page)
    state.mismatch = true
    await page.goto('/research')
    const detail = section(page, '通信ETF银华')
    await expect(detail.getByRole('heading', { name: '技术指标 · 日线截至 2026-10-08' })).toBeVisible()
    await expect(detail.getByText('早于最新报价日期')).toBeVisible()
    await expect(section(page, '相对走势').getByText('2026-09-30 — 2026-10-08 · 基期100')).toBeVisible()
    await detail.getByRole('tab', { name: '分时', exact: true }).click()
    await expect(detail.getByText('分时日期与报价日期不一致，昨收参考线暂不显示')).toBeVisible()
    await expect(detail.locator('.market-minute-previous-close')).toHaveCount(0)

    state.alignMinute = true
    await detail.getByRole('button', { name: '刷新标的', exact: true }).click()
    await expect(detail.getByText('分时日期与报价日期不一致，昨收参考线暂不显示')).toHaveCount(0)
    await expect(detail.locator('.market-minute-previous-close')).toHaveCount(1)
    expect(consoleErrors).toEqual([])
  })

  test('keeps research navigation and timestamps usable at mobile width', async ({ page }) => {
    const consoleErrors = trackConsoleErrors(page)
    await page.setViewportSize({ width: 390, height: 844 })
    await mockResearch(page)
    await page.goto('/research')
    await expect(page.getByRole('link', { name: '研究', exact: true })).toBeVisible()
    await expect(page.getByRole('heading', { name: '规则事件研究' })).toBeVisible()
    const quoteRow = section(page, '标的行情').getByRole('row').filter({ hasText: 'sz159994' })
    await expect(quoteRow.getByRole('button')).toContainText('10/09 10:09:00')
    expect(await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth)).toBe(false)
    expect(consoleErrors).toEqual([])
  })

  test('chart waits for viewport, draws then preserves identical refreshes and transitions changed prices', async ({ page }) => {
    const state = await mockResearch(page)
    await page.goto('/research')
    const detail = section(page, '通信ETF银华')
    const frame = detail.locator('.research-chart-frame')
    await expect(frame).toHaveAttribute('data-motion-state', 'waiting')
    await frame.scrollIntoViewIfNeeded()
    await expect(frame.locator('.recharts-line-curve').first()).toBeVisible()
    await expect(frame).toHaveAttribute('data-motion-state', 'settled')
    const curve = frame.locator('.recharts-line-curve').first()
    const initial = await curve.getAttribute('d')
    const instance = await frame.evaluate(el => {
      el.setAttribute('data-test-instance', 'original')
      el.setAttribute('data-path-updates', '0')
      const observer = new MutationObserver(records => {
        const changes = records.filter(record => record.attributeName === 'd').length
        if (changes) el.setAttribute('data-path-updates', String(Number(el.getAttribute('data-path-updates')) + changes))
      })
      observer.observe(el, { attributes: true, subtree: true, attributeFilter: ['d'] })
      return el.getAttribute('data-test-instance')
    })
    await detail.getByRole('button', { name: '刷新标的', exact: true }).click()
    await expect(detail.getByRole('button', { name: '刷新标的', exact: true })).toBeEnabled()
    await expect(curve).toHaveAttribute('d', initial!)
    await expect(frame).toHaveAttribute('data-test-instance', instance!)
    await expect(frame).toHaveAttribute('data-path-updates', '0')
    state.priceRevision = 0.03
    await detail.getByRole('button', { name: '刷新标的', exact: true }).click()
    await expect.poll(() => curve.getAttribute('d')).not.toBe(initial)
    await expect(frame).toHaveAttribute('data-motion-state', 'settled')
    // Multiple intermediate SVG paths prove the price update was interpolated, not swapped.
    await expect.poll(async () => Number(await frame.getAttribute('data-path-updates'))).toBeGreaterThan(2)
    await expect(frame.locator('.research-chart-end-dot')).toHaveCount(1)
    await frame.locator('svg').hover({ position: { x: 200, y: 100 } })
    await expect(frame.locator('.recharts-tooltip-wrapper')).toBeVisible()
  })

  test('reduced-motion charts render immediately at final coordinates without a drawing animation', async ({ page }) => {
    await page.emulateMedia({ reducedMotion: 'reduce' })
    await mockResearch(page)
    await page.goto('/research')
    const frame = section(page, '通信ETF银华').locator('.research-chart-frame')
    await frame.scrollIntoViewIfNeeded()
    await expect(frame).toHaveAttribute('data-motion-state', 'reduced')
    await expect(frame.locator('.recharts-line-curve').first()).toBeVisible()
    await expect(frame.locator('.research-chart-end-dot')).toHaveCount(1)
    await expect.poll(() => frame.locator('.research-chart-surface').evaluate(el => getComputedStyle(el).animationName)).toBe('none')
  })

})
