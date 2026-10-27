import { expect, test } from '@playwright/test'
import { calculateWorkspace, evaluateWorkspaceExpression as evaluate, parseWorkspaceExpression as parse, runWorkspace, validateWorkspaceConfig, readWorkspaceBody } from '../../src/lib/market-workspace-engine'
import { isTradingDay } from '../../src/lib/market-research-calendar'
import type { ResearchBar } from '../../src/hooks/use-market-research'
import type { ResearchHistory } from '../../src/lib/market-research-sources'
import type { WorkspaceConfig } from '../../src/lib/market-workspace-types'

const NOW = new Date('2026-10-09T16:00:00+08:00')
function bars(closes: number[], start = '2026-06-01'): ResearchBar[] {
  const date = new Date(`${start}T00:00:00Z`)
  return closes.map(close => {
    while (!isTradingDay(date.toISOString().slice(0, 10))) date.setUTCDate(date.getUTCDate() + 1)
    const bar = { date: date.toISOString().slice(0, 10), close, open: close, high: close, low: close, volume: 10 }
    date.setUTCDate(date.getUTCDate() + 1)
    return bar
  })
}
function history(input: ResearchBar[], asOf = NOW): ResearchHistory { return { bars: input, meta: { source: 'fixture', fetched_at: asOf.toISOString(), stale: false }, asOf } }
function config(): WorkspaceConfig {
  return { name: '测试工作区', sources: [{ alias: 'A', symbol: 'sh561980', field: 'close' }], curves: [{ id: 'curve', name: '曲线', expression: 'A' }], event: { mode: 'all', direction: 'up', conditions: [{ left: 'A', operator: 'gt', right: '100' }] }, target_symbol: 'sh561980', holding: 1, fee_bps: 100, limit: 20 }
}
function run(input: ResearchBar[], configuration = config()) { return calculateWorkspace(configuration, { sh561980: history(input) }, NOW) }

test('formula precedence, causal rolling indicators, null warmup and divide by zero', () => {
  const variables = { A: [1, 2, 3, 4, 5] }
  expect(evaluate('-A + 2 * (A + 1)', variables)).toEqual([3, 4, 5, 6, 7])
  expect(evaluate('MA(A,3)', variables)).toEqual([null, null, 2, 3, 4])
  expect(evaluate('EMA(A,3)', variables)).toEqual([null, null, 2, 3, 4])
  expect(evaluate('RSI(A,2)', variables)).toEqual([null, null, 100, 100, 100])
  expect(evaluate('REF(A,2)', variables)).toEqual([null, null, 1, 2, 3])
  expect(evaluate('REF(A,0)', variables)).toEqual(variables.A)
  expect(evaluate('ROC(A,2)', variables)).toEqual([null, null, 200, 100, (5 / 3 - 1) * 100])
  expect(evaluate('MIN(A,2)+MAX(A,2)', variables)).toEqual([null, 3, 5, 7, 9])
  expect(evaluate('STD(A,2)', variables)).toEqual([null, 0.5, 0.5, 0.5, 0.5])
  expect(evaluate('ABS(-A)', variables)).toEqual(variables.A)
  expect(evaluate('A/(A-A)', variables)).toEqual([null, null, null, null, null])
  expect(evaluate('EMA(A,2)', { A: [1, 3, null, 5, 7, 9] })).toEqual([null, 2, null, null, 6, 8])
})

test('rejects injection, unknown syntax, variable periods, future references and excessive complexity', () => {
  for (const expression of ['globalThis.process.exit()', 'A.constructor', 'A[0]', '(() => 1)()', 'fetch(A)', 'MA(A,B)', 'MA(A,0)', 'MA(A,201)', 'REF(A,-1)', 'MA(A,1.5)', 'MA(A,2+1)', 'A;A', 'A ** 2', 'B', 'ABS(A,2)']) expect(() => parse(expression, ['A'])).toThrow()
  expect(() => parse('('.repeat(26) + 'A' + ')'.repeat(26), ['A'])).toThrow(/嵌套/)
  expect(() => parse(Array(130).fill('A').join('+'), ['A'])).toThrow(/词元/)
  expect(() => parse(' '.repeat(1025), ['A'])).toThrow()
  expect(() => parse('MA(A,200)', ['A'])).not.toThrow()
})

test('rejects invalid config before network load', async () => {
  let loads = 0
  await expect(runWorkspace({ ...config(), sources: [{ alias: 'A', symbol: 'https://evil.example', field: 'close' }] }, async () => { loads++; return history([]) })).rejects.toThrow()
  expect(loads).toBe(0)
  for (const override of [{ holding: 0 }, { fee_bps: 201 }, { limit: 19 }, { limit: 641 }, { target_symbol: 'sz159994' }, { sources: [...config().sources, ...config().sources] }, { curves: [{ id: 'constructor', name: 'x', expression: 'A' }] }]) expect(() => validateWorkspaceConfig({ ...config(), ...override })).toThrow()
})

test('only transitions trigger; next target close enters; fees, pending and overlap are accounted', () => {
  const input = bars([90, 110, 120, 132, 140, 90, 110, 120, 108, 90, 110])
  const result = run(input)
  expect(result.rows.map(row => row.signal)).toEqual([false, true, false, false, false, false, true, false, false, false, true])
  expect(result.study.sample_count).toBe(2); expect(result.study.pending_count).toBe(1)
  expect(result.study.events[0]).toMatchObject({ signal_date: input[1].date, entry_date: input[2].date, exit_date: input[3].date })
  expect(result.study.events[0].return_pct).toBeCloseTo(10); expect(result.study.events[0].net_return_pct).toBeCloseTo(9)
  expect(result.study.direction_win_rate).toBe(50); expect(result.study.events[1].return_pct).toBeCloseTo(-10)
  const overlap = run(bars([90, 110, 90, 110, 90, 110, 120, 130]), { ...config(), holding: 3 })
  expect(overlap.study.sample_count).toBe(1); expect(overlap.study.events[0].exit_date).toBe(overlap.rows[5].date)
  const down = config(); down.event.direction = 'down'
  expect(run(input, down).study.events.every(event => event.net_return_pct === null)).toBe(true)
  expect(run(input, down).study.mean_net_return_pct).toBeNull()
})

test('strict date intersection excludes unfinished cached bars; crosses cannot leap source gaps', () => {
  const a = bars([90, 90, 110, 120, 130]), b = a.filter((_, i) => i !== 1)
  const cfg = config(); cfg.sources.push({ alias: 'B', symbol: 'sz159994', field: 'close' }); cfg.curves[0].expression = 'MA(A,2)+B'; cfg.event.conditions[0].operator = 'cross_up'
  const result = calculateWorkspace(cfg, { sh561980: history(a), sz159994: history(b) }, NOW)
  expect(result.rows.map(row => row.date)).toEqual([a[0].date, a[2].date, a[3].date, a[4].date])
  expect(result.rows[1].values.curve).toBe(210); expect(result.rows.every(row => !row.signal)).toBe(true)
  expect(result.warnings.join('')).toMatch(/缺失/)
  const cached = history([...a, { ...a[4], date: '2026-10-09', close: 10000, high: 10000 }], new Date('2026-10-09T15:04:00+08:00')); cached.meta.stale = true
  const filtered = calculateWorkspace(config(), { sh561980: cached }, NOW)
  expect(filtered.rows.at(-1)!.date).toBe(a[4].date); expect(filtered.warnings.join('')).toMatch(/滞后/)
})

test('execution uses target native timeline even if secondary source misses entry date', () => {
  const a = bars([90, 110, 120, 132, 140, 150]), b = a.filter((_, i) => i !== 2)
  const cfg = config(); cfg.sources.push({ alias: 'B', symbol: 'sz159994', field: 'close' })
  const result = calculateWorkspace(cfg, { sh561980: history(a), sz159994: history(b) }, NOW)
  expect(result.study.events[0].entry_date).toBe(a[2].date); expect(result.study.events[0].exit_date).toBe(a[3].date)
})

test('future mutation cannot change earlier curves or signals and limit does not truncate study', () => {
  const input = bars(Array.from({ length: 80 }, (_, i) => i % 4 === 0 ? 90 : 110 + i))
  const cfg = config(); cfg.curves[0].expression = 'MA(A,3)+EMA(A,4)+RSI(A,5)+REF(A,1)'
  const original = run(input, { ...cfg, limit: 640 })
  const mutated = input.map((bar, i) => i < 50 ? bar : { ...bar, close: bar.close * 2, open: bar.open * 2, high: bar.high * 2, low: bar.low * 2 })
  expect(run(mutated, { ...cfg, limit: 640 }).rows.slice(0, 50)).toEqual(original.rows.slice(0, 50))
  expect(run(input, cfg).rows).toHaveLength(20); expect(run(input, cfg).study).toEqual(original.study)
})

test('empty, malformed and unknown-time histories do not invent observations', () => {
  expect(() => run([])).toThrow(/没有已完成/)
  const bad = history(bars([1, 2])); bad.meta.fetched_at = ''
  expect(() => calculateWorkspace(config(), { sh561980: bad }, NOW)).toThrow(/观察时间/)
  expect(() => run([bars([1])[0], bars([1])[0]])).toThrow(/重复/)
  expect(() => run([{ ...bars([1])[0], close: Number.NaN }])).toThrow(/无效/)
})

test('request limits actual streamed bytes despite misleading Content-Length', async () => {
  const encoder = new TextEncoder()
  const body = new ReadableStream<Uint8Array>({ start(controller) { controller.enqueue(encoder.encode(' '.repeat(20000))); controller.enqueue(encoder.encode(' '.repeat(20000))); controller.close() } })
  const request = new Request('http://localhost/api', { method: 'POST', body, headers: { 'content-length': '1' }, duplex: 'half' } as RequestInit)
  await expect(readWorkspaceBody(request)).rejects.toThrow(/32KB/)
  await expect(readWorkspaceBody(new Request('http://localhost/api', { method: 'POST', body: '{' }))).rejects.toThrow(/JSON/)
  expect(await readWorkspaceBody(new Request('http://localhost/api', { method: 'POST', body: JSON.stringify(config()) }))).toEqual(config())
})

test('compound all/any condition modes respect unknown warmup and intersection', () => {
  const cfg = config()
  cfg.event.conditions = [{ left: 'A', operator: 'gt', right: '100' }, { left: 'MA(A,2)', operator: 'gt', right: '105' }]
  const input = bars([90, 110, 120, 90, 120, 130])
  expect(run(input, cfg).rows.map(row => row.signal)).toEqual([false, false, true, false, false, true])
  cfg.event.mode = 'any'
  expect(run(input, cfg).rows.map(row => row.signal)).toEqual([false, false, false, false, true, false])
  cfg.event.conditions = [{ left: 'A', operator: 'lt', right: 'MA(A,3)' }]
  expect(run(bars([120, 100, 90]), cfg).rows.every(row => !row.signal)).toBe(true)
})

test('bounded source history and finite metadata reject excessive or future inputs', () => {
  const cfg = config()
  const futureBars = bars(Array(641).fill(100))
  expect(() => run(futureBars)).toThrow(/640/)
  const future = history(bars([90, 110]), new Date('2026-10-10T16:00:00+08:00'))
  expect(() => calculateWorkspace(cfg, { sh561980: future }, NOW)).toThrow(/未来/)
  const badDate = history([{ ...bars([100])[0], date: '2026-06-07' }])
  expect(() => calculateWorkspace(cfg, { sh561980: badDate }, NOW)).toThrow(/非交易日/)
})


test('mixed cross and threshold fires causal first crossing after warmup without prior known composite', () => {
  const cfg = config()
  cfg.sources.push({ alias: 'B', symbol: 'sz159994', field: 'close' })
  cfg.event.conditions = [{ left: 'A', operator: 'cross_up', right: 'MA(A,2)' }, { left: 'B', operator: 'gt', right: '10' }]
  const a = bars([110, 90, 120, 130, 140]), b = bars([20, 20, 20, 20, 20])
  const result = calculateWorkspace(cfg, { sh561980: history(a), sz159994: history(b) }, NOW)
  expect(result.rows.map(row => row.state)).toEqual([null, null, true, false, false])
  expect(result.rows.map(row => row.signal)).toEqual([false, false, true, false, false])
  expect(result.study.events[0].signal_date).toBe(a[2].date)
})

test('OR continuous branch does not invent transition from null crossing; true crossing remains a pulse', () => {
  const cfg = config()
  cfg.event.mode = 'any'
  cfg.event.conditions = [{ left: 'A', operator: 'cross_up', right: 'MA(A,2)' }, { left: 'A', operator: 'gt', right: '100' }]
  const result = run(bars([90, 110, 90, 120, 130, 140]), cfg)
  expect(result.rows.map(row => row.state)).toEqual([null, true, false, true, true, true])
  expect(result.rows.map(row => row.signal)).toEqual([false, false, false, true, false, false])
  expect(result.study.sample_count).toBe(1)
  // A fresh causal crossing still triggers when the continuous OR branch has stayed true.
  cfg.event.conditions[1].right = '10'
  expect(run(bars([110, 90, 120, 130, 140]), cfg).rows.map(row => row.signal)).toEqual([false, false, true, false, false])
})

test('all sources missing the same trading day still cannot invent a crossing across the gap', () => {
  const cfg = config(); cfg.event.conditions[0].operator = 'cross_up'
  const input = bars([90, 90, 110, 120]).filter((_, i) => i !== 1)
  expect(run(input, cfg).rows.map(row => row.signal)).toEqual([false, false, false])
})

test('overflow formulas stay null and overflow baseline statistics are rejected rather than serialized as null', () => {
  const hugeLiteral = '9'.repeat(200)
  expect(evaluate(`${hugeLiteral}*${hugeLiteral}`, { A: [1, 2] })).toEqual([null, null])
  const cfg = config(); cfg.event.conditions[0].right = '0'
  expect(() => run(bars([1, 1e-300, 1e300, 1]), cfg)).toThrow(/有限数值范围/)
})
