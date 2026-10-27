import { expect, test } from '@playwright/test'
import {
  calculateResearchStudy,
  calculateResearchTechnical,
} from '../../src/lib/market-research-math'
import { isTradingDay } from '../../src/lib/market-research-calendar'
import type { ResearchBar, ResearchRule } from '../../src/hooks/use-market-research'

const NOW = new Date('2026-10-09T10:00:00+08:00')

function bars(closes: number[], start = '2026-06-01'): ResearchBar[] {
  const date = new Date(`${start}T00:00:00Z`)
  return closes.map((close) => {
    while (!isTradingDay(date.toISOString().slice(0, 10))) date.setUTCDate(date.getUTCDate() + 1)
    const bar = { date: date.toISOString().slice(0, 10), open: close, close, high: close, low: close, volume: 1 }
    date.setUTCDate(date.getUTCDate() + 1)
    return bar
  })
}

function study(input: ResearchBar[], rule: ResearchRule, holding = 2, feeBps = 10, now = NOW) {
  return calculateResearchStudy(input, 'sz159994', rule, holding, feeBps, now)
}

test.describe('market research indicator arithmetic', () => {
  test('calculates moving averages and requires sufficient observations', () => {
    const technical = calculateResearchTechnical(bars(Array.from({ length: 60 }, (_, i) => i + 1)), NOW)
    expect(technical).not.toBeNull()
    expect(technical!.ma5).toBe(58)
    expect(technical!.ma20).toBe(50.5)
    expect(technical!.ma60).toBe(30.5)
    expect(technical!.rsi14).toBe(100)
    const short = calculateResearchTechnical(bars([1, 2]), NOW)
    expect(short!.ma5).toBeNull()
    expect(short!.ma20).toBeNull()
    expect(short!.ma60).toBeNull()
    expect(short!.rsi14).toBeNull()
    expect(calculateResearchTechnical([], NOW)).toBeNull()
  })

  test('matches known Wilder RSI values rather than a rolling simple average', () => {
    const closes = [44.34, 44.09, 44.15, 43.61, 44.33, 44.83, 45.10, 45.42, 45.84, 46.08, 45.89, 46.03, 45.61, 46.28, 46.28, 46.00, 46.03]
    expect(calculateResearchTechnical(bars(closes.slice(0, 14)), NOW)!.rsi14).toBeNull()
    expect(calculateResearchTechnical(bars(closes.slice(0, 15)), NOW)!.rsi14).toBeCloseTo(70.4641350211, 8)
    expect(calculateResearchTechnical(bars(closes.slice(0, 16)), NOW)!.rsi14).toBeCloseTo(66.2496185536, 8)
    expect(calculateResearchTechnical(bars(closes), NOW)!.rsi14).toBeCloseTo(66.4809418347, 8)
    expect(calculateResearchTechnical(bars(Array(15).fill(10)), NOW)!.rsi14).toBe(50)
    expect(calculateResearchTechnical(bars(Array.from({ length: 15 }, (_, i) => 20 - i)), NOW)!.rsi14).toBe(0)
  })

  test('uses Shanghai time for provisional values and omits future price bars', () => {
    const input: ResearchBar[] = [
      { date: '2026-10-08', open: 10, close: 10, high: 11, low: 9, volume: 1 },
      { date: '2026-10-09', open: 8, close: 8, high: 9, low: 7, volume: 1 },
      { date: '2026-10-12', open: 100, close: 100, high: 100, low: 100, volume: 1 },
    ]
    const before = new Date('2026-10-09T07:04:59Z')
    const after = new Date('2026-10-09T07:05:00Z')
    const technical = calculateResearchTechnical(input, before)
    expect(technical!.provisional).toBe(true)
    expect(technical!.previous_low).toBe(9)
    expect(calculateResearchTechnical(input, after)!.provisional).toBe(false)
    expect(calculateResearchTechnical(input, before)).toEqual(calculateResearchTechnical(input.slice(0, 2), before))
    expect(study(input, 'previous_low_break', 1, 10, before).last_completed_date).toBe('2026-10-08')
    expect(study(input, 'previous_low_break', 1, 10, after).last_completed_date).toBe('2026-10-09')
  })

  test('does not mutate adjusted history or infer an extra unadjusted snapshot', () => {
    const input = bars(Array.from({ length: 60 }, (_, i) => (i + 1) / 10))
    const original = structuredClone(input)
    const technical = calculateResearchTechnical(input, NOW)
    expect(technical!.ma5).toBeCloseTo(5.8, 10)
    expect(technical!.ma20).toBeCloseTo(5.05, 10)
    expect(technical!.ma60).toBeCloseTo(3.05, 10)
    study(input, 'ma20_up')
    expect(input).toEqual(original)
    expect(calculateResearchTechnical(input, NOW)).toEqual(technical)
    const scaled = calculateResearchTechnical(input.map((bar) => ({ ...bar, open: bar.open * 10, close: bar.close * 10, high: bar.high * 10, low: bar.low * 10 })), NOW)
    expect(scaled!.ma5).toBeCloseTo(technical!.ma5! * 10, 10)
    expect(scaled!.rsi14).toBe(technical!.rsi14)
  })
})

test.describe('market research event study timing and accounting', () => {
  test('enters at the next observed trading close, skips overlap and excludes pending outcomes', () => {
    const input = bars([100, 90, 80, 70, 60, 50, 40, 30, 20, 10], '2026-09-14')
    const result = study(input, 'previous_low_break')
    expect(result.sample_count).toBe(2)
    expect(result.pending_count).toBe(1)
    expect(result.events).toHaveLength(2)
    expect(result.events[0]).toMatchObject({
      signal_date: '2026-09-15', entry_date: '2026-09-16', exit_date: '2026-09-18',
      return_pct: -25, direction_pct: 25, net_return_pct: null, drawdown_pct: 25,
    })
    expect(result.events[1].signal_date).toBe('2026-09-21')
    expect(result.events[1].entry_date).toBe('2026-09-22')
    expect(result.direction).toBe('down')
    expect(result.direction_win_rate).toBe(100)
    expect(result.mean_net_return_pct).toBeNull()
  })

  test('applies total basis-point costs once and leaves completed outcomes independent of later prices', () => {
    const input = bars([...Array(20).fill(100), 110, 120, 126, 132, 133])
    const result = study(input, 'ma20_up')
    expect(result.sample_count).toBe(1)
    expect(result.events[0].signal_date).toBe(input[20].date)
    expect(result.events[0].entry_date).toBe(input[21].date)
    expect(result.events[0].exit_date).toBe(input[23].date)
    expect(result.events[0].return_pct).toBeCloseTo(10, 10)
    expect(result.events[0].net_return_pct).toBeCloseTo(9.9, 10)
    expect(result.mean_net_return_pct).toBeCloseTo(9.9, 10)
    expect(result.baseline_mean_return_pct).toBeCloseTo((10 + (133 / 126 - 1) * 100) / 2, 10)
    const changed = structuredClone(input)
    changed[24] = { ...changed[24], open: 1000, close: 1000, high: 1000, low: 1000 }
    expect(study(changed, 'ma20_up').events).toEqual(result.events)
    expect(study(input, 'ma20_up', 2, 100).events[0].net_return_pct).toBeCloseTo(9, 10)
  })

  test('direction correctness counts completed neutral events without converting down signals into shorts', () => {
    const input = bars([100, 90, 80, 80, 90, 100, 110, 100, 90, 80, 100, 110, 100, 100, 110])
    const result = study(input, 'previous_low_break', 1)
    expect(result.sample_count).toBe(3)
    expect(result.events[0].return_pct).toBe(0)
    expect(result.events[0].direction_pct).toBeCloseTo(0, 10)
    expect(result.events[1].return_pct).toBeCloseTo((80 / 90 - 1) * 100, 10)
    expect(result.events[1].direction_pct).toBeCloseTo((1 - 80 / 90) * 100, 10)
    expect(result.events[2].return_pct).toBeCloseTo(10, 10)
    expect(result.events[2].direction_pct).toBeCloseTo(-10, 10)
    expect(result.direction_win_rate).toBeCloseTo(100 / 3, 10)
    expect(result.mean_return_pct).toBeCloseTo(((80 / 90 - 1) * 100 + 10) / 3, 10)
    expect(result.events.every((event) => event.net_return_pct === null)).toBe(true)
    expect(result.mean_net_return_pct).toBeNull()
  })

  test('keeps zero samples undefined and removes unfinished bars from signal and baseline calculations', () => {
    const empty = study([], 'ma20_up')
    expect(empty.sample_count).toBe(0)
    expect(empty.direction_win_rate).toBeNull()
    expect(empty.mean_return_pct).toBeNull()
    expect(empty.mean_net_return_pct).toBeNull()
    expect(empty.worst_drawdown_pct).toBeNull()
    expect(empty.baseline_mean_return_pct).toBeNull()
    const input = bars([100, 90, 80, 70, 60, 50], '2026-09-14')
    const original = study(input, 'previous_low_break')
    const appended = [
      ...input,
      { date: '2026-10-09', open: 999, close: 999, high: 999, low: 999, volume: 1 },
      { date: '2026-10-12', open: 1, close: 1, high: 1, low: 1, volume: 1 },
    ]
    expect(study(appended, 'previous_low_break')).toEqual(original)
  })

  test('a preclose cache remains provisional when supplied its original observation time', () => {
    const dates = ['2026-09-24', '2026-09-28', '2026-09-29', '2026-09-30', '2026-10-08', '2026-10-09']
    const input = bars([100, 100, 90, 80, 70, 60]).map((bar, index) => ({ ...bar, date: dates[index] }))
    const cachedAsOf = new Date('2026-10-09T15:04:00+08:00')
    const freshAsOf = new Date('2026-10-09T15:06:00+08:00')
    const cached = study(input, 'previous_low_break', 2, 10, cachedAsOf)
    expect(cached.last_completed_date).toBe('2026-10-08')
    expect(cached.sample_count).toBe(0)
    expect(cached.pending_count).toBe(1)
    expect(cached.direction_win_rate).toBeNull()
    expect(calculateResearchTechnical(input, cachedAsOf)!.provisional).toBe(true)
    const fresh = study(input, 'previous_low_break', 2, 10, freshAsOf)
    expect(fresh.last_completed_date).toBe('2026-10-09')
    expect(fresh.sample_count).toBe(1)
    expect(fresh.events[0].return_pct).toBeCloseTo(-25, 10)
    expect(calculateResearchTechnical(input, freshAsOf)!.provisional).toBe(false)
  })

  test('rejects invalid rule, holding period and nonfinite transaction costs', () => {
    const input = bars([100, 90, 80, 70])
    expect(() => study(input, 'invalid' as ResearchRule)).toThrow()
    for (const holding of [0, 61, 1.5, Number.NaN]) {
      expect(() => study(input, 'previous_low_break', holding)).toThrow()
    }
    for (const feeBps of [-1, 201, Number.NaN, Number.POSITIVE_INFINITY]) {
      expect(() => study(input, 'previous_low_break', 1, feeBps)).toThrow()
    }
    expect(() => study(input, 'previous_low_break', 60, 200)).not.toThrow()
  })

  test('crossing rules include equality and require RSI warmup', () => {
    const maInput = bars([...Array(20).fill(100), 90, 80, 70])
    const maDown = study(maInput, 'ma20_down', 1)
    expect(maDown.sample_count).toBe(1)
    expect(maDown.events[0].signal_date).toBe(maInput[20].date)
    expect(maDown.events[0].return_pct).toBeCloseTo(-12.5, 10)
    const rsiInput = bars([...Array.from({ length: 15 }, (_, i) => 100 - i), 200, 210, 220])
    const rsiRecovery = study(rsiInput, 'rsi_recovery', 1)
    expect(rsiRecovery.sample_count).toBe(1)
    expect(rsiRecovery.events[0].signal_date).toBe(rsiInput[15].date)
    expect(rsiRecovery.events[0].entry_date).toBe(rsiInput[16].date)
    expect(rsiRecovery.events[0].net_return_pct).toBeCloseTo((220 / 210 - 1) * 100 - 0.1, 10)
  })

  test('drawdown starts after entry close and reports a conservative same-day high-low bound', () => {
    const input = bars([...Array(20).fill(100), 110, 120, 125, 130])
    input[21] = { ...input[21], high: 1000, low: 1 }
    input[22] = { ...input[22], high: 150, low: 90 }
    const result = study(input, 'ma20_up')
    expect(result.events[0].drawdown_pct).toBeCloseTo(40, 10)
    expect(result.worst_drawdown_pct).toBeCloseTo(40, 10)
    expect(result.method).toMatch(/高低|顺序|上界/)
  })
})
