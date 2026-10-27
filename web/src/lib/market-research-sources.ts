import type { ResearchBar, ResearchInstrument, ResearchInstrumentDetail, ResearchOverview, ResearchQuote, ResearchSourceMeta, ResearchFlows } from '../hooks/use-market-research'
import { beijingDate, beijingTimestamp, dataPointWarning, dayCompleted, expectedDataDate, historyAsOf, marketPhase, previousTradingDate, validDate } from './market-research-calendar'

export const researchInstruments: ResearchInstrument[] = [
  { symbol: 'sh000001', name: '上证指数', group: '大盘' },
  { symbol: 'sz399001', name: '深证成指', group: '大盘' },
  { symbol: 'sz399006', name: '创业板指', group: '大盘' },
  { symbol: 'sh000688', name: '科创50', group: '大盘' },
  { symbol: 'sh000300', name: '沪深300', group: '大盘' },
  { symbol: 'sz159994', name: '通信ETF银华', group: '通信/CPO' },
  { symbol: 'sh561980', name: '半导体设备ETF招商', group: '半导体' },
  { symbol: 'sh512480', name: '半导体ETF国联安', group: '半导体' },
  { symbol: 'sz159995', name: '芯片ETF华夏', group: '半导体' },
  { symbol: 'sh588200', name: '科创芯片ETF嘉实', group: '半导体' },
  { symbol: 'sh512890', name: '红利低波ETF华泰柏瑞', group: '配置观察' },
  { symbol: 'sz159611', name: '电力ETF广发', group: '配置观察' },
  { symbol: 'sh518880', name: '黄金ETF华安', group: '配置观察' },
  { symbol: 'sh511360', name: '短融ETF海富通', group: '配置观察' },
  { symbol: 'sz159363', name: '创业板人工智能ETF华宝', group: '人工智能' },
  { symbol: 'sz300308', name: '中际旭创', group: '通信/CPO' },
  { symbol: 'sz300502', name: '新易盛', group: '通信/CPO' },
  { symbol: 'sz300394', name: '天孚通信', group: '通信/CPO' },
  { symbol: 'sh688981', name: '中芯国际', group: '半导体' },
  { symbol: 'sz002371', name: '北方华创', group: '半导体' },
]

export function findResearchInstrument(symbol: string): ResearchInstrument {
  const instrument = researchInstruments.find((item) => item.symbol === symbol)
  if (!instrument) throw new Error('标的不在研究白名单内')
  return { ...instrument }
}

type Minute = NonNullable<ResearchInstrumentDetail['minute']>
type Breadth = NonNullable<ResearchOverview['breadth']>
type News = ResearchOverview['news']
export type ResearchHistory = { bars: ResearchBar[]; meta: ResearchSourceMeta; asOf: Date }
type SourceResult<T> = { value: T; meta: ResearchSourceMeta }
type CacheEntry<T> = { value: T; fetchedAt: Date; hasValue: boolean; retryAt: number; error?: string; inflight?: Promise<SourceResult<T>> }
type SourceOptions = { fetch?: typeof fetch; now?: () => Date }

const endpoints = {
  quotes: 'https://qt.gtimg.cn/',
  history: 'https://web.ifzq.gtimg.cn/appstock/app/fqkline/get',
  minute: 'https://ifzq.gtimg.cn/appstock/app/minute/query',
  breadth: 'https://push2.eastmoney.com/api/qt/ulist.np/get',
  flows: 'https://push2.eastmoney.com/api/qt/clist/get',
  news: 'https://np-listapi.eastmoney.com/comm/web/getFastNewsList',
}

function object(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error('公开源数据结构异常')
  return value as Record<string, unknown>
}

function finite(value: unknown): number {
  if (typeof value !== 'string' && typeof value !== 'number' || value === '') throw new Error('公开源存在缺值')
  const result = Number(value)
  if (!Number.isFinite(result)) throw new Error('公开源数值无效')
  return result
}

function warn(meta: ResearchSourceMeta, warning: string): ResearchSourceMeta {
  return warning ? { ...meta, stale: true, error: [meta.error, warning].filter(Boolean).join('；') } : meta
}

function quoteDelayed(quote: ResearchQuote, now: Date): boolean {
  const stamp = quote.timestamp.match(/^(\d{4}-\d{2}-\d{2})T(\d{2}:\d{2}:\d{2})\+08:00$/)
  return !stamp || Boolean(dataPointWarning(stamp[1], stamp[2], now))
}

function parseQuotes(text: string): ResearchQuote[] {
  const bySymbol = new Map<string, ResearchQuote>()
  for (const match of text.matchAll(/v_((?:sh|sz)\d{6})="([^"]*)";/g)) {
    const instrument = researchInstruments.find((item) => item.symbol === match[1])
    if (!instrument) continue
    const fields = match[2].split('~')
    try {
      if (fields.length < 35 || !/^\d{14}$/.test(fields[30])) continue
      const stamp = fields[30]
      const date = `${stamp.slice(0, 4)}-${stamp.slice(4, 6)}-${stamp.slice(6, 8)}`
      const time = `${stamp.slice(8, 10)}:${stamp.slice(10, 12)}:${stamp.slice(12, 14)}`
      if (!validDate(date) || !/^([01]\d|2[0-3]):[0-5]\d:[0-5]\d$/.test(time)) continue
      const quote: ResearchQuote = { ...instrument, price: finite(fields[3]), prev_close: finite(fields[4]), open: finite(fields[5]), high: finite(fields[33]), low: finite(fields[34]), change_pct: finite(fields[32]), timestamp: `${date}T${time}+08:00`, delayed: false }
      if (quote.price <= 0 || quote.prev_close <= 0) continue
      bySymbol.set(quote.symbol, quote)
    } catch { /* Invalid upstream rows stay missing and are reported by the caller. */ }
  }
  return researchInstruments.flatMap((item) => bySymbol.has(item.symbol) ? [bySymbol.get(item.symbol)!] : [])
}

function parseHistory(json: unknown, symbol: string): ResearchBar[] {
  const root = object(json)
  if (finite(root.code) !== 0) throw new Error('日线源返回异常状态')
  const entry = object(object(root.data)[symbol])
  let rows = entry.qfqday
  if (!Array.isArray(rows) || rows.length === 0) {
    if (findResearchInstrument(symbol).group !== '大盘') throw new Error('日线源缺少前复权数据')
    rows = entry.day
  }
  if (!Array.isArray(rows) || rows.length === 0) throw new Error('日线源返回空数据')
  const byDate = new Map<string, ResearchBar>()
  for (const row of rows) {
    if (!Array.isArray(row) || row.length < 6 || typeof row[0] !== 'string' || !validDate(row[0])) throw new Error('日线记录不完整')
    const bar = { date: row[0], open: finite(row[1]), close: finite(row[2]), high: finite(row[3]), low: finite(row[4]), volume: finite(row[5]) }
    if (bar.open <= 0 || bar.close <= 0 || bar.low <= 0 || bar.high < bar.low || bar.volume < 0) throw new Error('日线价格或成交量无效')
    byDate.set(bar.date, bar)
  }
  // The upstream can return 640 completed bars plus a current bar despite the requested limit.
  return [...byDate.values()].sort((a, b) => a.date.localeCompare(b.date)).slice(-640)
}

function parseMinute(json: unknown, symbol: string): Minute {
  const root = object(json)
  if (finite(root.code) !== 0) throw new Error('分时源返回异常状态')
  const entry = object(object(object(root.data)[symbol]).data)
  if (typeof entry.date !== 'string' || !/^\d{8}$/.test(entry.date)) throw new Error('分时源没有有效交易日期')
  const date = `${entry.date.slice(0, 4)}-${entry.date.slice(4, 6)}-${entry.date.slice(6, 8)}`
  if (!validDate(date) || !Array.isArray(entry.data)) throw new Error('分时源记录异常')
  const byTime = new Map<string, { time: string; price: number }>()
  for (const row of entry.data) {
    if (typeof row !== 'string') continue
    const fields = row.trim().split(/\s+/)
    if (!/^([01]\d|2[0-3])[0-5]\d$/.test(fields[0]) || fields.length < 2) continue
    const price = Number(fields[1])
    if (!Number.isFinite(price) || price <= 0) continue
    const time = `${fields[0].slice(0, 2)}:${fields[0].slice(2, 4)}`
    byTime.set(time, { time, price })
  }
  const points = [...byTime.values()].sort((a, b) => a.time.localeCompare(b.time))
  if (!points.length) throw new Error('分时源返回空数据')
  return { date, points }
}

function parseBreadth(json: unknown): Breadth {
  const rows = object(object(json).data).diff
  if (!Array.isArray(rows)) throw new Error('沪深宽度源返回空数据')
  const result: Breadth = { advancing: 0, declining: 0, unchanged: 0, turnover_yuan: 0 }
  const seen = new Set<string>()
  for (const raw of rows) {
    const row = object(raw)
    if (row.f12 !== '000001' && row.f12 !== '399001' || seen.has(String(row.f12))) continue
    const advancing = finite(row.f104), declining = finite(row.f105), unchanged = finite(row.f106), amount = finite(row.f6)
    if (![advancing, declining, unchanged].every((value) => Number.isInteger(value) && value >= 0) || amount < 0) throw new Error('沪深宽度源数值无效')
    seen.add(row.f12)
    result.advancing += advancing; result.declining += declining; result.unchanged += unchanged; result.turnover_yuan += amount
  }
  if (seen.size !== 2 || !result.advancing && !result.declining && !result.unchanged) throw new Error('沪深宽度源未完整返回两市数据')
  return result
}

function parseNews(json: unknown, now: Date): News {
  const root = object(json)
  if (String(root.code) !== '1') throw new Error('新闻源返回异常状态')
  const rows = object(root.data).fastNewsList
  if (!Array.isArray(rows)) throw new Error('新闻源返回空数据')
  const seen = new Set<string>()
  const result: News = []
  for (const raw of rows) {
    const row = object(raw)
    if (typeof row.code !== 'string' || !/^\d{8}\d+$/.test(row.code) || seen.has(row.code) || typeof row.title !== 'string' || typeof row.showTime !== 'string') continue
    const stamp = row.showTime.match(/^(\d{4}-\d{2}-\d{2}) (\d{2}:\d{2}:\d{2})$/)
    if (!stamp || !validDate(stamp[1]) || !/^([01]\d|2[0-3]):[0-5]\d:[0-5]\d$/.test(stamp[2]) || new Date(`${stamp[1]}T${stamp[2]}+08:00`).getTime() > now.getTime() + 60_000) continue
    const title = row.title.replace(/<[^>]*>/g, '').replace(/\s+/g, ' ').trim()
    if (title.length < 6) continue
    seen.add(row.code)
    result.push({ title, url: `https://finance.eastmoney.com/a/${row.code}.html`, date: stamp[1], related: /AI|CPO|半导体|光模块|光通信|人工智能|芯片|算力|通信|科技|OPENAI|英伟达/i.test(title) })
  }
  if (!result.length) throw new Error('新闻源未找到有效原文链接')
  return result.slice(0, 24)
}

export function parseResearchFlows(json: unknown, now: Date): ResearchFlows['rows'] {
  const rows = object(object(json).data).diff
  if (!Array.isArray(rows) || !rows.length) throw new Error('板块资金源返回空数据')
  const seen = new Set<string>()
  return rows.map((raw) => {
    const row = object(raw)
    if (typeof row.f12 !== 'string' || !/^BK\d+$/.test(row.f12) || typeof row.f14 !== 'string' || !row.f14.trim() || row.f14.length > 60) throw new Error('板块资金标识无效')
    const seconds = finite(row.f124)
    if (!Number.isInteger(seconds) || seconds <= 0 || seconds * 1000 > now.getTime() + 60_000) throw new Error('板块资金时间无效')
    const timestamp = beijingTimestamp(new Date(seconds * 1000))
    return { code: row.f12, name: row.f14.trim(), change_pct: finite(row.f3), net_inflow_yuan: finite(row.f62), timestamp }
  }).filter((row) => { if (seen.has(row.code)) return false; seen.add(row.code); return true })
}

export function createMarketResearchSources(options: SourceOptions = {}) {
  const fetcher = options.fetch ?? fetch
  const clock = options.now ?? (() => new Date())
  // Module-instance cache only: serverless instances/redeploys do not share or persist it.
  const cache = new Map<string, CacheEntry<unknown>>()

  async function request(endpoint: string, params: Record<string, string>, encoding = 'utf-8'): Promise<string> {
    const url = new URL(endpoint)
    for (const [key, value] of Object.entries(params)) url.searchParams.set(key, value)
    const response = await fetcher(url, { signal: AbortSignal.timeout(8_000), cache: 'no-store', headers: { 'User-Agent': 'Mozilla/5.0', Referer: 'https://finance.eastmoney.com/' } })
    if (!response.ok) throw new Error(`公开源返回 HTTP ${response.status}`)
    const limit = 3 * 1024 * 1024
    if (Number(response.headers.get('content-length')) > limit) { await response.body?.cancel(); throw new Error('公开源响应超过大小限制') }
    const reader = response.body?.getReader()
    if (!reader) throw new Error('公开源返回空响应')
    const chunks: Uint8Array[] = []
    let size = 0
    try {
      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        size += value.length
        if (size > limit) { await reader.cancel(); throw new Error('公开源响应超过大小限制') }
        chunks.push(value)
      }
    } finally { reader.releaseLock() }
    const bytes = new Uint8Array(size)
    let offset = 0
    for (const chunk of chunks) { bytes.set(chunk, offset); offset += chunk.length }
    return new TextDecoder(encoding).decode(bytes)
  }

  async function load<T>(key: string, source: string, ttl: number, empty: T, loader: () => Promise<T>, force = false): Promise<SourceResult<T>> {
    const now = clock()
    let entry = cache.get(key) as CacheEntry<T> | undefined
    if (!entry) { entry = { value: empty, fetchedAt: new Date(0), hasValue: false, retryAt: 0 }; cache.set(key, entry) }
    const meta = (): ResearchSourceMeta => ({ source, fetched_at: entry!.hasValue ? beijingTimestamp(entry!.fetchedAt) : '', stale: Boolean(entry!.error), ...(entry!.error ? { error: entry!.error } : {}) })
    if (entry.inflight) return entry.inflight
    if (entry.hasValue && !entry.error && !force && now.getTime() - entry.fetchedAt.getTime() < ttl) return { value: entry.value, meta: meta() }
    if (now.getTime() < entry.retryAt) return { value: entry.value, meta: { ...meta(), stale: true } }
    const current = entry
    current.inflight = (async () => {
      try {
        const value = await loader()
        current.value = value; current.fetchedAt = now; current.hasValue = true; current.retryAt = 0; current.error = undefined
      } catch (error) {
        current.error = error instanceof Error ? error.message : '公开源读取失败'
        current.retryAt = clock().getTime() + 10_000
      }
      return { value: current.value, meta: { ...meta(), stale: Boolean(current.error) } }
    })()
    try { return await current.inflight } finally { current.inflight = undefined }
  }

  async function quotes(): Promise<SourceResult<ResearchQuote[]>> {
    const loaded = await load('quotes', '腾讯公开行情', 20_000, [] as ResearchQuote[], async () => {
      const values = parseQuotes(await request(endpoints.quotes, { q: researchInstruments.map((item) => item.symbol).join(',') }, 'gb18030'))
      const missing = researchInstruments.filter((item) => !values.some((quote) => quote.symbol === item.symbol)).map((item) => item.symbol)
      if (missing.length) {
        const previous = cache.get('quotes') as CacheEntry<ResearchQuote[]>
        if (!previous.hasValue) previous.value = values
        throw new Error(`部分行情缺失或字段无效: ${missing.join(', ')}`)
      }
      return values
    })
    const now = clock()
    return { ...loaded, value: loaded.value.map((quote) => ({ ...quote, delayed: quoteDelayed(quote, now) })) }
  }

  async function history(symbol: string): Promise<ResearchHistory> {
    findResearchInstrument(symbol)
    const now = clock(), today = beijingDate(now)
    const entry = cache.get(`history:${symbol}`) as CacheEntry<ResearchBar[]> | undefined
    const force = Boolean(entry?.hasValue && dayCompleted(today, now) && !dayCompleted(today, entry.fetchedAt) && entry.value.at(-1)?.date === today)
    const loaded = await load(`history:${symbol}`, '腾讯前复权日线（指数为原始日线）', 300_000, [] as ResearchBar[], async () => parseHistory(JSON.parse(await request(endpoints.history, { param: `${symbol},day,,,640,qfq` })), symbol), force)
    let expected = expectedDataDate(now)
    if (!dayCompleted(today, now) && expected === today) expected = previousTradingDate(now)
    const last = loaded.value.at(-1)?.date
    const meta = last && last < expected ? warn(loaded.meta, `日线仅更新到 ${last}，已完成交易日应为 ${expected}`) : loaded.meta
    return { bars: loaded.value.slice(-640).map((bar) => ({ ...bar })), meta, asOf: historyAsOf(meta, now) }
  }

  async function minute(symbol: string): Promise<SourceResult<Minute | null>> {
    const loaded = await load(`minute:${symbol}`, '腾讯公开分时', 30_000, null as Minute | null, async () => parseMinute(JSON.parse(await request(endpoints.minute, { code: symbol })), symbol))
    const last = loaded.value?.points.at(-1)
    const warning = loaded.value ? dataPointWarning(loaded.value.date, last?.time ?? '', clock()) : ''
    return { ...loaded, meta: warn(loaded.meta, warning) }
  }

  async function overview(): Promise<ResearchOverview> {
    const [quote, breadth, news] = await Promise.all([
      quotes(),
      load('breadth', '东方财富沪深市场统计', 30_000, null as Breadth | null, async () => parseBreadth(JSON.parse(await request(endpoints.breadth, { secids: '1.000001,0.399001', fields: 'f12,f6,f104,f105,f106', fltt: '2' })))),
      load('news', '东方财富公开快讯（原文入口）', 300_000, [] as News, async () => parseNews(JSON.parse(await request(endpoints.news, { client: 'web', biz: 'web_724', fastColumn: '102', sortEnd: '', pageSize: '24', req_trace: String(clock().getTime()) })), clock())),
    ])
    const now = clock()
    return { generated_at: beijingTimestamp(now), instruments: researchInstruments.map((item) => ({ ...item })), quotes: quote.value, quote_meta: quote.meta, breadth: breadth.value, breadth_meta: breadth.meta, news: news.value, news_meta: news.meta, market_phase: marketPhase(now) }
  }

  async function flows(): Promise<ResearchFlows> {
    const loaded = await load('flows', '东方财富板块资金估算', 60_000, [] as ResearchFlows['rows'], async () => {
      const params = { pn: '1', pz: '50', np: '1', fltt: '2', invt: '2', fid: 'f62', fs: 'm:90+t:2', fields: 'f12,f14,f3,f62,f124' }
      const [inflows, outflows] = await Promise.all([request(endpoints.flows, { ...params, po: '1' }), request(endpoints.flows, { ...params, po: '0' })])
      const rows = [...parseResearchFlows(JSON.parse(inflows), clock()), ...parseResearchFlows(JSON.parse(outflows), clock())]
      return [...new Map(rows.map((row) => [row.code, row])).values()].sort((a, b) => b.net_inflow_yuan - a.net_inflow_yuan)
    })
    const now = clock()
    const warnings = loaded.value.map((row) => dataPointWarning(row.timestamp.slice(0, 10), row.timestamp.slice(11, 16), now)).filter(Boolean)
    return { rows: loaded.value, meta: warn(loaded.meta, [...new Set(warnings)].join('；')), method: '按数据商成交分类估算的板块主力净流入；取净流入和净流出排名各前50项，非完整板块列表。板块可能重叠，不可相加；不代表机构实际持仓、北向净买入或ETF份额净申购。' }
  }

  async function instrument(symbol: string): Promise<Omit<ResearchInstrumentDetail, 'technical'>> {
    const selected = findResearchInstrument(symbol)
    const [quote, bars, minutes] = await Promise.all([quotes(), history(symbol), minute(symbol)])
    return { instrument: selected, quote: quote.value.find((item) => item.symbol === symbol) ?? null, quote_meta: quote.meta, bars: bars.bars, history_meta: bars.meta, minute: minutes.value, minute_meta: minutes.meta }
  }

  return { overview, instrument, history, flows }
}

export const marketResearchSources = createMarketResearchSources()
