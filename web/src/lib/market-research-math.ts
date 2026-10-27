import type { ResearchBar, ResearchInstrumentDetail, ResearchRule, ResearchStudy } from '../hooks/use-market-research'
import { beijingDate, dayCompleted } from './market-research-calendar'

type Technical = ResearchInstrumentDetail['technical']

export const RESEARCH_RULES: ResearchRule[] = ['ma20_up', 'ma20_down', 'rsi_recovery', 'previous_low_break']

export function validateStudyParameters(rule: string, holding: number, feeBps: number) {
  if (!RESEARCH_RULES.includes(rule as ResearchRule)) throw new Error('规则必须为 ma20_up、ma20_down、rsi_recovery 或 previous_low_break')
  if (!Number.isInteger(holding) || holding < 1 || holding > 60) throw new Error('持有期必须为 1 至 60 个交易日')
  if (!Number.isFinite(feeBps) || feeBps < 0 || feeBps > 200) throw new Error('总交易费用必须为 0 至 200 基点')
}

function average(closes: number[], index: number, period: number) {
  if (index < period - 1) return null
  return closes.slice(index - period + 1, index + 1).reduce((sum, value) => sum + value, 0) / period
}

export function researchRsi(closes: number[], period = 14): (number | null)[] {
  const values: (number | null)[] = closes.map(() => null)
  if (closes.length <= period) return values
  let gain = 0
  let loss = 0
  for (let i = 1; i <= period; i++) {
    const delta = closes[i] - closes[i - 1]
    gain += Math.max(delta, 0)
    loss += Math.max(-delta, 0)
  }
  gain /= period
  loss /= period
  for (let i = period; i < closes.length; i++) {
    if (i > period) {
      const delta = closes[i] - closes[i - 1]
      gain = (gain * (period - 1) + Math.max(delta, 0)) / period
      loss = (loss * (period - 1) + Math.max(-delta, 0)) / period
    }
    values[i] = gain === 0 && loss === 0 ? 50 : loss === 0 ? 100 : 100 - 100 / (1 + gain / loss)
  }
  return values
}

export function calculateResearchTechnical(input: ResearchBar[], asOf: Date): Technical {
  const bars = input.filter(bar => bar.date <= beijingDate(asOf)).sort((a, b) => a.date.localeCompare(b.date))
  if (!bars.length) return null
  const closes = bars.map(bar => bar.close)
  const index = bars.length - 1
  const technical: NonNullable<Technical> = {
    ma5: average(closes, index, 5), ma20: average(closes, index, 20), ma60: average(closes, index, 60),
    rsi14: researchRsi(closes)[index], previous_low: index > 0 ? bars[index - 1].low : null,
    provisional: !dayCompleted(bars[index].date, asOf), conditions: [],
  }
  technical.conditions.push({
    label: '数据口径', tone: 'neutral',
    detail: `日线日期 ${bars[index].date}；同源前复权日线计算${technical.provisional ? '，盘中日线可能滞后于报价，收盘前会变化' : ''}`,
  })
  if (technical.ma20 !== null) {
    const above = closes[index] >= technical.ma20
    technical.conditions.push({ label: above ? '位于20日均线上方' : '低于20日均线', tone: above ? 'positive' : 'negative', detail: '反映历史趋势位置，不代表后续必然延续' })
  }
  if (technical.previous_low !== null) {
    const below = closes[index] < technical.previous_low
    technical.conditions.push({ label: below ? '日线收盘参考价跌破前日低点' : '日线收盘参考价守住前日低点', tone: below ? 'negative' : 'neutral', detail: '比较同源日线；盘中报价和最终收盘价可能不同' })
  }
  if (technical.rsi14 !== null) {
    technical.conditions.push({ label: technical.rsi14 < 30 ? 'RSI低于30' : technical.rsi14 > 70 ? 'RSI高于70' : 'RSI处于中间区间', tone: 'neutral', detail: '超买或超卖不等于反转；RSI采用14期Wilder平滑' })
  }
  return technical
}

function triggered(bars: ResearchBar[], closes: number[], rsi: (number | null)[], index: number, rule: ResearchRule) {
  if (!index) return false
  if (rule === 'ma20_up' || rule === 'ma20_down') {
    const previous = average(closes, index - 1, 20)
    const current = average(closes, index, 20)
    if (previous === null || current === null) return false
    return rule === 'ma20_up' ? closes[index - 1] <= previous && closes[index] > current : closes[index - 1] >= previous && closes[index] < current
  }
  if (rule === 'rsi_recovery') {
    const previous = rsi[index - 1]
    const current = rsi[index]
    return previous !== null && current !== null && previous <= 30 && current > 30
  }
  return bars[index].close < bars[index - 1].low
}

const STUDY_METHOD = '固定规则；仅使用已完成同源前复权日线（指数为原始日线），当日15:05前排除当日；信号后下一交易日收盘进入，进入后第N个交易日收盘退出；排除持有期间重叠信号与未完成样本。收益为标的价格变化，向上规则减去一次总交易费用（基点/100为百分点）；向下规则仅统计规避下跌的方向正确性，不模拟做空，净收益为空。回撤为进入收盘后至退出日的日线高点到低点最大下降幅度，数值为正；同日日内高低点顺序未知，采用先高后低的保守上界，可能高估实际回撤。基准为同一可观察期间所有可完成的N日滚动窗口平均涨跌（窗口可重叠，未扣费）。复权历史可能被源修订，费用不含未输入的基金赎回费、跟踪误差与滑点；历史结果不是未来承诺。'

export function calculateResearchStudy(input: ResearchBar[], symbol: string, rule: ResearchRule, holding: number, feeBps: number, asOf: Date): Omit<ResearchStudy, 'history_meta'> {
  validateStudyParameters(rule, holding, feeBps)
  const bars = input.filter(bar => dayCompleted(bar.date, asOf)).sort((a, b) => a.date.localeCompare(b.date))
  const down = rule === 'ma20_down' || rule === 'previous_low_break'
  const result: Omit<ResearchStudy, 'history_meta'> = {
    symbol, rule, direction: down ? 'down' : 'up', holding, fee_bps: feeBps, sample_count: 0, pending_count: 0,
    direction_win_rate: null, mean_return_pct: null, mean_net_return_pct: null, worst_drawdown_pct: null,
    baseline_mean_return_pct: null, events: [], last_completed_date: bars.at(-1)?.date || '', method: STUDY_METHOD,
  }
  if (!bars.length) return result
  const closes = bars.map(bar => bar.close)
  const rsi = researchRsi(closes)
  const firstEligible = rule === 'ma20_up' || rule === 'ma20_down' ? 20 : rule === 'rsi_recovery' ? 15 : 1
  let lastExit = -1
  for (let i = firstEligible; i < bars.length; i++) {
    if (i <= lastExit || !triggered(bars, closes, rsi, i, rule)) continue
    const entry = i + 1
    const exit = entry + holding
    lastExit = exit
    if (exit >= bars.length) {
      result.pending_count++
      continue
    }
    const change = (bars[exit].close / bars[entry].close - 1) * 100
    let peak = bars[entry].close
    let drawdown = 0
    for (let j = entry + 1; j <= exit; j++) {
      // Daily OHLC cannot establish order; assume the day's high precedes its low.
      peak = Math.max(peak, bars[j].high)
      drawdown = Math.max(drawdown, (1 - bars[j].low / peak) * 100)
    }
    result.events.push({ signal_date: bars[i].date, entry_date: bars[entry].date, exit_date: bars[exit].date, return_pct: change, direction_pct: down ? -change : change, net_return_pct: down ? null : change - feeBps / 100, drawdown_pct: drawdown })
  }
  result.sample_count = result.events.length
  if (result.sample_count) {
    result.direction_win_rate = result.events.filter(event => event.direction_pct > 0).length / result.sample_count * 100
    result.mean_return_pct = result.events.reduce((sum, event) => sum + event.return_pct, 0) / result.sample_count
    result.worst_drawdown_pct = Math.max(...result.events.map(event => event.drawdown_pct))
    if (!down) result.mean_net_return_pct = result.events.reduce((sum, event) => sum + event.net_return_pct!, 0) / result.sample_count
  }
  let total = 0
  let count = 0
  for (let entry = firstEligible + 1; entry + holding < bars.length; entry++) {
    total += (bars[entry + holding].close / bars[entry].close - 1) * 100
    count++
  }
  if (count) result.baseline_mean_return_pct = total / count
  return result
}
