import type { ResearchBar } from '../hooks/use-market-research'
import { beijingDate, dayCompleted, previousTradingDate, validDate, isTradingDay } from './market-research-calendar'
import { researchRsi, validateStudyParameters } from './market-research-math'
import { findResearchInstrument } from './market-research-sources'
import type { ResearchHistory } from './market-research-sources'
import type { WorkspaceConfig, WorkspaceResult, WorkspaceStudy } from './market-workspace-types'

export const WORKSPACE_FORMULA_HELP = '变量 A/B/C；+ - * /、负号、括号；MA/EMA/RSI/ROC/STD/MIN/MAX(表达式,整数周期1..200)，REF(表达式,整数滞后0..200)，ABS(表达式)。MIN/MAX为滚动最小/最大，STD为总体标准差，ROC为百分比；EMA以首个完整周期均值初始化，RSI为Wilder平滑。缺失、预热、除零返回空值。周期按各来源已完成日线的共同日期计算，严格取交集，不补值；交叉不会跨越来源缺失日期。持续条件要求组合由已知假转真；组合当日成立且含有效交叉脉冲时，即使前日组合未知或已成立也触发。'
const MAX_TOKENS = 256
const MAX_DEPTH = 24
const PERIOD_FUNCTIONS = new Set(['MA', 'EMA', 'RSI', 'ROC', 'STD', 'MIN', 'MAX', 'REF'])
type Node = { kind: 'number'; value: number } | { kind: 'variable'; name: string } | { kind: 'unary'; child: Node; sign: number } | { kind: 'binary'; op: string; left: Node; right: Node } | { kind: 'call'; name: string; child: Node; period?: number }
type Series = (number | null)[]

export function parseWorkspaceExpression(expression: string, aliases: readonly string[]): Node {
  if (typeof expression !== 'string' || !expression.trim() || expression.length > 1024) throw new Error('公式不能为空且最多1024字符')
  const tokens: string[] = []
  const pattern = /\s*(?:(\d+(?:\.\d*)?|\.\d+)|([A-Za-z][A-Za-z0-9_]*)|([+\-*/(),]))/y
  let offset = 0
  while (offset < expression.trimEnd().length) {
    pattern.lastIndex = offset
    const match = pattern.exec(expression)
    if (!match) throw new Error(`公式包含不支持的字符（位置${offset + 1}）`)
    tokens.push(match[1] || match[2] || match[3])
    offset = pattern.lastIndex
    if (tokens.length > MAX_TOKENS) throw new Error('公式最多256个词元')
  }
  let cursor = 0
  const take = (token: string) => { if (tokens[cursor] !== token) throw new Error(`公式缺少 ${token}`); cursor++ }
  function primary(depth: number): Node {
    if (depth > MAX_DEPTH) throw new Error('公式嵌套最多24层')
    const token = tokens[cursor++]
    if (token === '+' || token === '-') return { kind: 'unary', sign: token === '-' ? -1 : 1, child: primary(depth + 1) }
    if (token === '(') { const child = expressionNode(depth + 1); take(')'); return child }
    if (token && /^(?:\d|\.)/.test(token)) {
      const value = Number(token)
      if (!Number.isFinite(value)) throw new Error('数字必须有限')
      return { kind: 'number', value }
    }
    if (token && /^[A-Za-z]/.test(token)) {
      if (tokens[cursor] !== '(') {
        if (!aliases.includes(token)) throw new Error(`未知变量 ${token}`)
        return { kind: 'variable', name: token }
      }
      if (token !== 'ABS' && !PERIOD_FUNCTIONS.has(token)) throw new Error(`未知函数 ${token}`)
      take('(')
      const child = expressionNode(depth + 1)
      let period: number | undefined
      if (token !== 'ABS') {
        take(',')
        const raw = tokens[cursor++]
        if (!raw || !/^\d+$/.test(raw)) throw new Error('周期必须为整数字面量')
        period = Number(raw)
        if (period < (token === 'REF' ? 0 : 1) || period > 200) throw new Error('周期范围为1至200，REF允许0')
      }
      take(')')
      return { kind: 'call', name: token, child, period }
    }
    throw new Error('公式缺少操作数')
  }
  function product(depth: number): Node {
    let left = primary(depth)
    while (tokens[cursor] === '*' || tokens[cursor] === '/') { const op = tokens[cursor++]; left = { kind: 'binary', op, left, right: primary(depth) } }
    return left
  }
  function expressionNode(depth: number): Node {
    let left = product(depth)
    while (tokens[cursor] === '+' || tokens[cursor] === '-') { const op = tokens[cursor++]; left = { kind: 'binary', op, left, right: product(depth) } }
    return left
  }
  const node = expressionNode(0)
  if (cursor !== tokens.length) throw new Error('公式末尾有多余内容')
  return node
}

function finite(value: number): number | null { return Number.isFinite(value) ? value : null }
export function evaluateWorkspaceExpression(expression: string, variables: Record<string, Series>): Series {
  const lengths = Object.values(variables).map(value => value.length)
  if (!lengths.length || lengths.some(length => length !== lengths[0])) throw new Error('公式变量长度不一致')
  const size = lengths[0]
  const ast = parseWorkspaceExpression(expression, Object.keys(variables))
  function evaluate(node: Node): Series {
    if (node.kind === 'number') return Array(size).fill(node.value)
    if (node.kind === 'variable') return variables[node.name].map(value => value === null ? null : finite(value))
    if (node.kind === 'unary') return evaluate(node.child).map(value => value === null ? null : finite(value * node.sign))
    if (node.kind === 'binary') {
      const left = evaluate(node.left), right = evaluate(node.right)
      return left.map((a, index) => {
        const b = right[index]
        if (a === null || b === null || node.op === '/' && b === 0) return null
        return finite(node.op === '+' ? a + b : node.op === '-' ? a - b : node.op === '*' ? a * b : a / b)
      })
    }
    const input = evaluate(node.child), n = node.period ?? 1
    if (node.name === 'ABS') return input.map(value => value === null ? null : Math.abs(value))
    if (node.name === 'REF' || node.name === 'ROC') return input.map((value, i) => {
      const previous = i < n ? null : input[i - n]
      if (node.name === 'REF') return previous
      return value === null || previous === null || previous === 0 ? null : finite((value / previous - 1) * 100)
    })
    if (node.name === 'RSI') {
      const output: Series = Array(size).fill(null)
      let start = 0
      while (start < size) {
        while (start < size && input[start] === null) start++
        let end = start
        while (end < size && input[end] !== null) end++
        if (end > start) researchRsi(input.slice(start, end) as number[], n).forEach((value, i) => { output[start + i] = value === null ? null : finite(value) })
        start = end + 1
      }
      return output
    }
    const output: Series = Array(size).fill(null)
    let ema: number | null = null
    for (let i = 0; i < size; i++) {
      if (input[i] === null) { ema = null; continue }
      const window = input.slice(Math.max(0, i - n + 1), i + 1)
      if (window.length !== n || window.some(value => value === null)) continue
      const numbers = window as number[]
      const mean = numbers.reduce((sum, value) => sum + value, 0) / n
      if (node.name === 'EMA') { ema = ema === null ? mean : input[i]! * (2 / (n + 1)) + ema * (1 - 2 / (n + 1)); output[i] = finite(ema) }
      else if (node.name === 'MA') output[i] = finite(mean)
      else if (node.name === 'MIN') output[i] = Math.min(...numbers)
      else if (node.name === 'MAX') output[i] = Math.max(...numbers)
      else output[i] = finite(Math.sqrt(numbers.reduce((sum, value) => sum + (value - mean) ** 2, 0) / n))
    }
    return output
  }
  return evaluate(ast)
}

function record(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error('研究配置必须为对象')
  return value as Record<string, unknown>
}
function text(value: unknown, max: number, label: string): string {
  if (typeof value !== 'string' || !value.trim() || value.length > max) throw new Error(`${label}不能为空且最多${max}字符`)
  return value.trim()
}
function list(value: unknown, label: string): unknown[] {
  if (!Array.isArray(value) || value.length < 1 || value.length > 3) throw new Error(`${label}必须为1至3项`)
  return value
}
export function validateWorkspaceConfig(input: unknown): WorkspaceConfig {
  const value = record(input)
  const sources = list(value.sources, '数据源').map(item => {
    const source = record(item)
    const alias = text(source.alias, 1, '别名')
    if (!['A', 'B', 'C'].includes(alias)) throw new Error('别名只能为A/B/C')
    const symbol = text(source.symbol, 20, '标的')
    findResearchInstrument(symbol)
    if (!['close', 'open', 'high', 'low', 'volume'].includes(String(source.field))) throw new Error('未知数据字段')
    return { alias, symbol, field: source.field as WorkspaceConfig['sources'][number]['field'] }
  })
  if (new Set(sources.map(source => source.alias)).size !== sources.length) throw new Error('数据源别名不能重复')
  const aliases = sources.map(source => source.alias)
  const curves = list(value.curves, '曲线').map(item => {
    const curve = record(item)
    const id = text(curve.id, 24, '曲线ID')
    if (!/^[A-Za-z][A-Za-z0-9_]*$/.test(id) || ['__proto__', 'prototype', 'constructor'].includes(id)) throw new Error('曲线ID必须为安全字母数字标识')
    const expression = text(curve.expression, 1024, '公式')
    parseWorkspaceExpression(expression, aliases)
    return { id, name: text(curve.name, 40, '曲线名'), expression }
  })
  if (new Set(curves.map(curve => curve.id)).size !== curves.length) throw new Error('曲线ID不能重复')
  const event = record(value.event)
  if (!['all', 'any'].includes(String(event.mode)) || !['up', 'down'].includes(String(event.direction))) throw new Error('未知事件组合或方向')
  const conditions = list(event.conditions, '条件').map(item => {
    const condition = record(item)
    if (!['gt', 'lt', 'cross_up', 'cross_down'].includes(String(condition.operator))) throw new Error('未知条件操作符')
    const left = text(condition.left, 1024, '条件左公式'), right = text(condition.right, 1024, '条件右公式')
    parseWorkspaceExpression(left, aliases); parseWorkspaceExpression(right, aliases)
    return { left, right, operator: condition.operator as WorkspaceConfig['event']['conditions'][number]['operator'] }
  })
  const target = text(value.target_symbol, 20, '研究标的')
  if (!sources.some(source => source.symbol === target)) throw new Error('研究标的必须属于已选数据源')
  if (typeof value.holding !== 'number' || typeof value.fee_bps !== 'number') throw new Error('持有期与费用必须为数字')
  validateStudyParameters('ma20_up', value.holding, value.fee_bps)
  if (!Number.isInteger(value.limit) || Number(value.limit) < 20 || Number(value.limit) > 640) throw new Error('显示条数必须为20至640的整数')
  return { name: text(value.name, 60, '工作区名称'), sources, curves, event: { mode: event.mode as 'all' | 'any', conditions, direction: event.direction as 'up' | 'down' }, target_symbol: target, holding: value.holding, fee_bps: value.fee_bps, limit: value.limit as number }
}

export type WorkspaceHistories = Record<string, ResearchHistory>
function completedHistory(history: ResearchHistory, now: Date): ResearchBar[] {
  if (!history || !(history.asOf instanceof Date) || !Number.isFinite(history.asOf.getTime()) || !Number.isFinite(new Date(history.meta.fetched_at).getTime())) throw new Error('历史数据观察时间未知，无法进行研究')
  if (history.bars.length > 640) throw new Error('单个数据源最多640根日线')
  if (history.bars.some(bar => !validDate(bar.date) || !isTradingDay(bar.date))) throw new Error('历史日线日期无效或非交易日')
  if (history.asOf.getTime() > now.getTime() + 60_000 || new Date(history.meta.fetched_at).getTime() > now.getTime() + 60_000) throw new Error('历史源观察时间在未来')
  const asOf = new Date(Math.min(now.getTime(), history.asOf.getTime(), new Date(history.meta.fetched_at).getTime()))
  const bars = history.bars.filter(bar => dayCompleted(bar.date, asOf)).sort((a, b) => a.date.localeCompare(b.date))
  if (!bars.length) throw new Error('数据源没有已完成的日线')
  if (bars.length > 640) throw new Error('单个数据源最多640根日线')
  if (new Set(bars.map(bar => bar.date)).size !== bars.length) throw new Error('历史日期重复，无法对齐')
  if (bars.some(bar => !validDate(bar.date) || ![bar.open, bar.high, bar.low, bar.close, bar.volume].every(Number.isFinite) || Math.min(bar.open, bar.high, bar.low, bar.close) <= 0 || bar.volume < 0 || bar.high < Math.max(bar.open, bar.close, bar.low) || bar.low > Math.min(bar.open, bar.close))) throw new Error('历史日线包含无效价格或成交量')
  return bars
}

function calculateStudy(bars: ResearchBar[], dates: string[], signals: boolean[], config: WorkspaceConfig, meta: ResearchHistory['meta'], firstEligibleDate?: string): WorkspaceStudy {
  const down = config.event.direction === 'down'
  const result: WorkspaceStudy = {
    rule: 'workspace', symbol: config.target_symbol, direction: config.event.direction, holding: config.holding, fee_bps: config.fee_bps,
    sample_count: 0, pending_count: 0, direction_win_rate: null, mean_return_pct: null, mean_net_return_pct: null, worst_drawdown_pct: null, baseline_mean_return_pct: null,
    events: [], last_completed_date: bars.at(-1)?.date || '', history_meta: meta,
    method: `${WORKSPACE_FORMULA_HELP} 数据缺失或预热不制造事件。持续条件要求组合由已知假转真；混合组合满足且至少一个有效交叉当日成立时直接触发，即使前日组合未知或已成立。信号后标的下一交易日收盘进入，进入后第N个标的交易日收盘退出；标的时间轴保留，不强制退出日期在共同交集内。排除持有期间重叠信号，未完成事件记为待完成。向上收益减一次总费用（基点/100百分点），向下仅统计规避下跌方向，净收益为空。回撤从进入收盘后计算日线高点到低点保守上界，同日高低顺序未知。基准为首个全部条件可评价日期后全部可完成的N日滚动窗口（可重叠、不扣费）。历史复权可被修订，未模拟滑点或未输入费用；历史结果不是未来承诺。`,
  }
  const indexes = new Map(bars.map((bar, i) => [bar.date, i]))
  let lastExit = -1
  for (let i = 0; i < dates.length; i++) {
    const index = indexes.get(dates[i])!
    if (!signals[i] || index <= lastExit) continue
    const entry = index + 1, exit = entry + config.holding
    lastExit = exit
    if (exit >= bars.length) { result.pending_count++; continue }
    const change = (bars[exit].close / bars[entry].close - 1) * 100
    if (!Number.isFinite(change)) throw new Error('历史收益超出有限数值范围')
    let peak = bars[entry].close, drawdown = 0
    for (let j = entry + 1; j <= exit; j++) { peak = Math.max(peak, bars[j].high); drawdown = Math.max(drawdown, (1 - bars[j].low / peak) * 100) }
    result.events.push({ signal_date: bars[index].date, entry_date: bars[entry].date, exit_date: bars[exit].date, return_pct: change, direction_pct: down ? -change : change, net_return_pct: down ? null : change - config.fee_bps / 100, drawdown_pct: drawdown })
  }
  const count = result.events.length
  result.sample_count = count
  if (count) {
    result.direction_win_rate = result.events.filter(event => event.direction_pct > 0).length / count * 100
    result.mean_return_pct = result.events.reduce((sum, event) => sum + event.return_pct, 0) / count
    result.worst_drawdown_pct = Math.max(...result.events.map(event => event.drawdown_pct))
    if (!down) result.mean_net_return_pct = result.events.reduce((sum, event) => sum + event.net_return_pct!, 0) / count
  }
  const first = firstEligibleDate ? indexes.get(firstEligibleDate)! + 1 : bars.length
  let baselineTotal = 0, baselineCount = 0
  for (let entry = first; entry + config.holding < bars.length; entry++) { const change = (bars[entry + config.holding].close / bars[entry].close - 1) * 100; if (!Number.isFinite(change)) throw new Error('历史基准收益超出有限数值范围'); baselineTotal += change; baselineCount++ }
  if (baselineCount) result.baseline_mean_return_pct = baselineTotal / baselineCount
  if ([result.mean_return_pct, result.mean_net_return_pct, result.baseline_mean_return_pct].some(value => value !== null && !Number.isFinite(value))) throw new Error('历史统计超出有限数值范围')
  return result
}

export function calculateWorkspace(input: unknown, histories: WorkspaceHistories, now = new Date()): WorkspaceResult {
  const config = validateWorkspaceConfig(input)
  if (!Number.isFinite(now.getTime())) throw new Error('研究时间无效')
  const completed = new Map(config.sources.map(source => [source.symbol, completedHistory(histories[source.symbol], now)]))
  const maps = config.sources.map(source => new Map(completed.get(source.symbol)!.map((bar, index) => [bar.date, { bar, index }])))
  const dates = [...maps[0].keys()].filter(date => maps.every(map => map.has(date)))
  if (!dates.length) throw new Error('数据源没有共同的已完成日期')
  const continuous = dates.map((date, i) => i > 0 && previousTradingDate(new Date(date + 'T00:00:00+08:00')) === dates[i - 1] && maps.every(map => map.get(date)!.index === map.get(dates[i - 1])!.index + 1))
  const variables: Record<string, Series> = Object.fromEntries(config.sources.map((source, i) => [source.alias, dates.map(date => maps[i].get(date)!.bar[source.field])]))
  const curves = config.curves.map(curve => ({ id: curve.id, values: evaluateWorkspaceExpression(curve.expression, variables) }))
  const conditions = config.event.conditions.map(condition => {
    const left = evaluateWorkspaceExpression(condition.left, variables), right = evaluateWorkspaceExpression(condition.right, variables)
    return dates.map((_, i): boolean | null => {
      const a = left[i], b = right[i]
      if (a === null || b === null) return null
      if (condition.operator === 'gt') return a > b
      if (condition.operator === 'lt') return a < b
      if (!continuous[i] || left[i - 1] === null || right[i - 1] === null) return null
      return condition.operator === 'cross_up' ? left[i - 1]! <= right[i - 1]! && a > b : left[i - 1]! >= right[i - 1]! && a < b
    })
  })
  const truths = dates.map((_, i): boolean | null => {
    const values = conditions.map(condition => condition[i])
    if (config.event.mode === 'all') return values.includes(false) ? false : values.includes(null) ? null : true
    return values.includes(true) ? true : values.includes(null) ? null : false
  })
  // A causally established crossing is itself a pulse, even if the prior composite was unknown.
  // A continuous OR branch cannot manufacture a transition from an unknown prior state.
  const signals = truths.map((truth, i) => {
    const crossingPulse = config.event.conditions.some((condition, j) => condition.operator.startsWith('cross_') && conditions[j][i] === true)
    return truth === true && (crossingPulse || i > 0 && continuous[i] && truths[i - 1] === false)
  })
  const target = histories[config.target_symbol]
  const warnings = new Set<string>(['多源指标周期在共同日期交集上计算，不等同于各标的原生交易日指标；请选择时间轴相容的来源。'])
  const expected = isTradingDay(beijingDate(now)) && dayCompleted(beijingDate(now), now) ? beijingDate(now) : previousTradingDate(now)
  for (const source of config.sources) {
    const history = histories[source.symbol]
    if (history.meta.stale || history.meta.error) warnings.add(`${source.alias}（${source.symbol}）历史源滞后或降级：${history.meta.error || '使用缓存'}；观察时间${history.meta.fetched_at}`)
    if (completed.get(source.symbol)!.at(-1)!.date < expected) warnings.add(`${source.alias}最后已完成日线为${completed.get(source.symbol)!.at(-1)!.date}，早于应观察日期${expected}`)
  }
  if (maps.some(map => map.size !== dates.length)) warnings.add('已严格按共同日期对齐，排除未完成或缺失日期，不填充价格；交叉不跨缺失日期。')
  return {
    generated_at: now.toISOString(),
    rows: dates.map((date, i) => ({ date, values: Object.fromEntries(curves.map(curve => [curve.id, curve.values[i]])), signal: signals[i], state: truths[i] })).slice(-config.limit),
    sources: config.sources.map(source => ({ alias: source.alias, symbol: source.symbol, meta: histories[source.symbol].meta })),
    study: calculateStudy(completed.get(config.target_symbol)!, dates, signals, config, target.meta, dates.find((_, i) => conditions.every(condition => condition[i] !== null))), warnings: [...warnings],
  }
}

export async function runWorkspace(input: unknown, loadHistory: (symbol: string) => Promise<ResearchHistory>, now = new Date()): Promise<WorkspaceResult> {
  const config = validateWorkspaceConfig(input)
  const symbols = [...new Set(config.sources.map(source => source.symbol))]
  const histories = await Promise.all(symbols.map(async symbol => [symbol, await loadHistory(symbol)] as const))
  return calculateWorkspace(config, Object.fromEntries(histories), now)
}
const BODY_LIMIT = 32 * 1024

export async function readWorkspaceBody(request: Request): Promise<unknown> {
  const declared = request.headers.get('content-length')
  if (declared !== null && (!/^\d+$/.test(declared) || Number(declared) > BODY_LIMIT)) throw new Error('请求体最多32KB')
  if (!request.body) throw new Error('请求体不能为空')
  const reader = request.body.getReader()
  const chunks: Uint8Array[] = []
  let total = 0
  try {
    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      total += value.byteLength
      if (total > BODY_LIMIT) { await reader.cancel(); throw new Error('请求体最多32KB') }
      chunks.push(value)
    }
  } finally { reader.releaseLock() }
  const bytes = new Uint8Array(total)
  let offset = 0
  for (const chunk of chunks) { bytes.set(chunk, offset); offset += chunk.byteLength }
  try { return JSON.parse(new TextDecoder('utf-8', { fatal: true }).decode(bytes)) }
  catch { throw new Error('请求体必须为有效JSON') }
}
