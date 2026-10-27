import { expect, test } from '@playwright/test'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { createMarketResearchSources, researchInstruments, parseResearchFlows } from '../../src/lib/market-research-sources'
import { beijingDate, dataPointWarning, dayCompleted, historyAsOf, isTradingDay, marketPhase, previousTradingDate } from '../../src/lib/market-research-calendar'
import holidays from '../../src/lib/market-research-holidays.json'

const at = (time: string) => new Date(`2026-10-09T${time}+08:00`)

function quotes(price = '102', timestamp = '20261009100000') {
  return researchInstruments.map((instrument) => {
    const fields = Array.from({ length: 40 }, () => '0')
    fields[1] = 'Instrument'; fields[2] = instrument.symbol.slice(2)
    fields[3] = price; fields[4] = '100'; fields[5] = '101'
    fields[30] = timestamp; fields[32] = '2'; fields[33] = '104'; fields[34] = '99'
    return `v_${instrument.symbol}="${fields.join('~')}";`
  }).join('\n')
}

const history = (symbol: string, date = '2026-10-08', field = 'qfqday') => ({ code: 0, data: { [symbol]: { [field]: [[date, '100', '102', '104', '99', '1000']] } } })
const minute = (symbol: string, date = '20261009', point = '0959 102 100') => ({ code: 0, data: { [symbol]: { data: { date, data: [point] } } } })
const breadth = { data: { diff: [{ f12: '000001', f6: 100, f104: 2, f105: 3, f106: 1 }, { f12: '399001', f6: 200, f104: 4, f105: 5, f106: 2 }] } }
const news = { code: '1', data: { fastNewsList: [{ code: '20261009123456', title: '半导体今日公开消息', showTime: '2026-10-09 09:59:00' }] } }

function fakeFetch(handler?: (url: URL) => Response | Promise<Response>) {
  const requests: URL[] = []
  const mocked = (async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = new URL(String(input))
    requests.push(url)
    expect(init?.cache).toBe('no-store')
    expect(init?.signal).toBeTruthy()
    if (handler) return handler(url)
    if (url.hostname === 'qt.gtimg.cn') return new Response(quotes())
    if (url.pathname.includes('/fqkline/')) return Response.json(history(url.searchParams.get('param')!.split(',')[0]))
    if (url.pathname.includes('/minute/')) return Response.json(minute(url.searchParams.get('code')!))
    if (url.pathname.includes('/ulist.')) return Response.json(breadth)
    return Response.json(news)
  }) as typeof fetch
  return { mocked, requests }
}

test('source calendar preserves embedded SSE holidays and Beijing boundaries', () => {
  const goCalendar = readFileSync(resolve(process.cwd(), '../internal/trading/holiday_data.go'), 'utf8')
  const goDates = [...goCalendar.matchAll(/"(\d{4}-\d{2}-\d{2})"/g)].map((match) => match[1]).sort()
  expect(Object.values(holidays.years).flat().sort()).toEqual(goDates)
  expect(beijingDate(new Date('2026-10-08T23:00:00Z'))).toBe('2026-10-09')
  expect(marketPhase(new Date('2026-10-01T10:00:00+08:00'))).toBe('holiday')
  expect(isTradingDay('2026-10-01')).toBe(false)
  expect(isTradingDay('2026-10-10')).toBe(false)
  expect(previousTradingDate(new Date('2026-10-08T10:00:00+08:00'))).toBe('2026-09-30')
  expect(dayCompleted('2026-10-09', at('15:04:59'))).toBe(false)
  expect(dayCompleted('2026-10-09', at('15:05:00'))).toBe(true)
  expect(dayCompleted('2026-10-10', at('16:00:00'))).toBe(false)
})

test('sources keep Shanghai prefix, upstream timestamps, structured news and typed empty arrays', async () => {
  const { mocked, requests } = fakeFetch()
  const sources = createMarketResearchSources({ fetch: mocked, now: () => at('10:00:00') })
  const overview = await sources.overview()
  expect(overview.quotes).toHaveLength(researchInstruments.length)
  expect(overview.quotes[0].symbol).toBe('sh000001')
  expect(overview.quotes[0].timestamp).toBe('2026-10-09T10:00:00+08:00')
  expect(overview.quotes[0].name).toBe('上证指数')
  expect(overview.breadth?.turnover_yuan).toBe(300)
  expect(overview.news[0].related).toBe(true)
  expect(overview.news[0].url).toBe('https://finance.eastmoney.com/a/20261009123456.html')
  const detail = await sources.instrument('sh000001')
  expect(detail.bars[0].date).toBe('2026-10-08')
  expect(detail.minute?.points[0].time).toBe('09:59')
  expect(requests.find((url) => url.hostname === 'qt.gtimg.cn')?.searchParams.get('q')).toContain('sh000001')
  expect(requests.find((url) => url.hostname === 'qt.gtimg.cn')?.searchParams.get('q')).not.toContain('sz000001')
  const before = requests.length
  await expect(sources.instrument('000001')).rejects.toThrow('白名单')
  await expect(sources.history('https://evil.example')).rejects.toThrow('白名单')
  expect(requests).toHaveLength(before)
})

test('concurrent refresh deduplicates inflight public requests', async () => {
  const { mocked, requests } = fakeFetch()
  const sources = createMarketResearchSources({ fetch: mocked, now: () => at('10:00:00') })
  await Promise.all([sources.overview(), sources.overview(), sources.instrument('sz159994'), sources.instrument('sz159994')])
  expect(requests.filter((url) => url.hostname === 'qt.gtimg.cn')).toHaveLength(1)
  expect(requests.filter((url) => url.pathname.includes('/fqkline/'))).toHaveLength(1)
  expect(requests.filter((url) => url.pathname.includes('/minute/'))).toHaveLength(1)
  expect(requests.filter((url) => url.pathname.includes('/ulist.'))).toHaveLength(1)
})

test('failed refresh retains last good timestamp, backs off retries and recomputes quote delay', async () => {
  let now = at('10:00:00'), failed = false
  const { mocked, requests } = fakeFetch((url) => url.hostname === 'qt.gtimg.cn' ? new Response(failed ? '' : quotes(), { status: failed ? 502 : 200 }) : Response.json(url.pathname.includes('/ulist.') ? breadth : news))
  const sources = createMarketResearchSources({ fetch: mocked, now: () => now })
  await sources.overview()
  now = at('10:04:00'); failed = true
  const stale = await sources.overview()
  expect(stale.quotes).toHaveLength(researchInstruments.length)
  expect(stale.quotes[0].delayed).toBe(true)
  expect(stale.quote_meta).toMatchObject({ stale: true, fetched_at: '2026-10-09T10:00:00+08:00' })
  expect(stale.quote_meta.error).toContain('502')
  const calls = requests.length
  await sources.overview()
  expect(requests).toHaveLength(calls)
})

test('partial first-source failures report missing quotes without inventing values', async () => {
  const { mocked } = fakeFetch((url) => url.hostname === 'qt.gtimg.cn' ? new Response(quotes().split('\n')[0]) : new Response('', { status: 503 }))
  const overview = await createMarketResearchSources({ fetch: mocked, now: () => at('10:00:00') }).overview()
  expect(overview.quotes).toHaveLength(1)
  expect(overview.quote_meta.stale).toBe(true)
  expect(overview.quote_meta.fetched_at).toBe('')
  expect(overview.quote_meta.error).toContain('sz399001')
  expect(overview.news).toEqual([])
  expect(overview.news_meta.stale).toBe(true)
  expect(overview.breadth).toBeNull()
})

test('missing breadth values and nonfinite quote rows become explicit source failures', async () => {
  const missing = { data: { diff: [{ f12: '000001', f6: 100, f104: 2, f105: 3, f106: 1 }, { f12: '399001', f6: 200, f104: 4, f105: 5 }] } }
  const { mocked } = fakeFetch((url) => url.hostname === 'qt.gtimg.cn' ? new Response(quotes('NaN')) : Response.json(url.pathname.includes('/ulist.') ? missing : news))
  const overview = await createMarketResearchSources({ fetch: mocked, now: () => at('10:00:00') }).overview()
  expect(overview.quotes).toEqual([])
  expect(overview.quote_meta.stale).toBe(true)
  expect(overview.breadth).toBeNull()
  expect(overview.breadth_meta.error).toContain('缺值')
})

test('ETF history requires adjustment and sorts validated dates without quote mixing', async () => {
  const { mocked } = fakeFetch((url) => {
    const symbol = url.searchParams.get('param')?.split(',')[0]
    if (symbol === 'sz159994') return Response.json(history(symbol, '2026-10-08', 'day'))
    return Response.json({ code: 0, data: { sh000001: { day: [['2026-10-08', '100', '101', '103', '99', '100'], ['2026-09-30', '99', '100', '102', '98', '100']] } } })
  })
  const sources = createMarketResearchSources({ fetch: mocked, now: () => at('10:00:00') })
  const etf = await sources.history('sz159994')
  expect(etf.bars).toEqual([])
  expect(etf.meta.error).toContain('前复权')
  const index = await sources.history('sh000001')
  expect(index.bars.map((bar) => bar.date)).toEqual(['2026-09-30', '2026-10-08'])
  expect(index.meta.stale).toBe(false)
})

test('minute date and final point freshness are recomputed from cached data', async () => {
  let now = at('10:00:00')
  const { mocked } = fakeFetch((url) => {
    if (url.hostname === 'qt.gtimg.cn') return new Response(quotes())
    if (url.pathname.includes('/fqkline/')) return Response.json(history('sz159994'))
    return Response.json(minute('sz159994', '20261008', '1500 102 100'))
  })
  const sources = createMarketResearchSources({ fetch: mocked, now: () => now })
  const old = await sources.instrument('sz159994')
  expect(old.minute_meta.stale).toBe(true)
  expect(old.minute_meta.error).toContain('应观察 2026-10-09')
  expect(old.minute?.date).toBe('2026-10-08')
  now = new Date('2026-10-10T10:00:00+08:00')
  expect(dataPointWarning('2026-10-09', '15:00', now)).toBe('')
  expect(dataPointWarning('2026-10-09', '10:00', at('16:00:00'))).toContain('延迟')
  expect(dataPointWarning('2026-10-09', '10:02', at('10:00:00'))).toContain('未来')
})

test('pre-close cache forces reload and remains provisional after failed refresh', async () => {
  let now = at('15:03:00'), failed = false
  const { mocked, requests } = fakeFetch(() => failed ? new Response('', { status: 502 }) : Response.json(history('sz159994', '2026-10-09')))
  const sources = createMarketResearchSources({ fetch: mocked, now: () => now })
  await sources.history('sz159994')
  now = at('15:05:00'); failed = true
  const snapshot = await sources.history('sz159994')
  expect(requests).toHaveLength(2)
  expect(snapshot.meta.stale).toBe(true)
  expect(snapshot.asOf.toISOString()).toBe(at('15:03:00').toISOString())
  expect(dayCompleted('2026-10-09', snapshot.asOf)).toBe(false)
  expect(historyAsOf(snapshot.meta, new Date('2026-10-10T10:00:00+08:00')).toISOString()).toBe(at('15:03:00').toISOString())
})

test('structured news sanitizes titles, drops future records and deduplicates article codes', async () => {
  const rows = [{ code: '20261009123456', title: '<b>半导体今日公开消息</b>', showTime: '2026-10-09 09:59:00' }, { code: '20261009123456', title: '重复新闻标题', showTime: '2026-10-09 09:59:00' }, { code: '20261010123456', title: '未来新闻标题不展示', showTime: '2026-10-10 09:59:00' }, { code: 'javascript:alert(1)', title: '恶意代码不生成链接', showTime: '2026-10-09 09:59:00' }]
  const { mocked } = fakeFetch((url) => url.hostname === 'qt.gtimg.cn' ? new Response(quotes()) : Response.json(url.pathname.includes('/ulist.') ? breadth : { code: '1', data: { fastNewsList: rows } }))
  const overview = await createMarketResearchSources({ fetch: mocked, now: () => at('10:00:00') }).overview()
  expect(overview.news).toHaveLength(1)
  expect(overview.news[0].title).toBe('半导体今日公开消息')
})

test('upstream response size cap produces partial overview instead of an unbounded read', async () => {
  const { mocked } = fakeFetch(() => new Response('oversized', { headers: { 'content-length': String(4 * 1024 * 1024) } }))
  const overview = await createMarketResearchSources({ fetch: mocked, now: () => at('10:00:00') }).overview()
  expect(overview.quote_meta.error).toContain('大小限制')
  expect(overview.quotes).toEqual([])
  expect(overview.news).toEqual([])
})


test('sector flow parsing rejects missing data and future timestamps without fabricating estimates', () => {
  const row = { f12: 'BK0001', f14: '半导体', f3: -1, f62: -100000000, f124: at('10:00:00').getTime() / 1000 }
  expect(parseResearchFlows({ data: { diff: [row, row] } }, at('10:00:00'))).toEqual([{ code: 'BK0001', name: '半导体', change_pct: -1, net_inflow_yuan: -100000000, timestamp: '2026-10-09T10:00:00+08:00' }])
  expect(() => parseResearchFlows({ data: { diff: [{ ...row, f62: undefined }] } }, at('10:00:00'))).toThrow('缺值')
  expect(() => parseResearchFlows({ data: { diff: [{ ...row, f124: at('11:00:00').getTime() / 1000 }] } }, at('10:00:00'))).toThrow('时间')
})

test('sector flow refresh preserves the last complete ranking when one source fails', async () => {
  let failed = false, now = at('10:00:00')
  const { mocked } = fakeFetch((url) => {
    if (failed && url.searchParams.get('po') === '0') return new Response('', { status: 503 })
    return Response.json({ data: { diff: [{ f12: url.searchParams.get('po') === '1' ? 'BK0001' : 'BK0002', f14: '测试板块', f3: 1, f62: url.searchParams.get('po') === '1' ? 100 : -100, f124: at('10:00:00').getTime() / 1000 }] } })
  })
  const sources = createMarketResearchSources({ fetch: mocked, now: () => now })
  const first = await sources.flows()
  expect(first.rows.map(row => row.net_inflow_yuan)).toEqual([100, -100])
  expect(first.meta.stale).toBe(false)
  failed = true; now = at('10:01:05')
  const second = await sources.flows()
  expect(second.rows).toEqual(first.rows)
  expect(second.meta.stale).toBe(true)
  expect(second.meta.error).toContain('503')
  expect(second.method).toContain('不代表')
})



test('public history bounds upstream extra current bar to the newest 640 observations', async () => {
  const rows = Array.from({ length: 641 }, (_, i) => [new Date(Date.UTC(2023, 0, 1 + i)).toISOString().slice(0, 10), '100', '102', '104', '99', '1000'])
  const { mocked } = fakeFetch(() => Response.json({ code: 0, data: { sh512480: { qfqday: rows } } }))
  const result = await createMarketResearchSources({ fetch: mocked, now: () => at('16:00:00') }).history('sh512480')
  expect(result.bars).toHaveLength(640)
  expect(result.bars[0].date).toBe(rows[1][0])
  expect(result.bars.at(-1)!.date).toBe(rows.at(-1)![0])
})
