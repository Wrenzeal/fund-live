'use client'

import { useMemo, useState } from 'react'
import {
  AlertTriangle,
  BellRing,
  CheckCircle2,
  Info,
  ShieldAlert,
} from 'lucide-react'
import type { FundAnalysis } from '@/hooks/use-fund-data'
import type {
  HoldingAggregateEntry,
  HoldingEntry,
  HoldingEstimateAggregateMetrics,
} from '@/hooks/use-user-portfolio'
import {
  buildHoldingReminderSummary,
  type HoldingExposureSnapshot,
  type InsightTone,
} from '@/lib/holding-insights'
import type { HoldingMetricScope } from '@/lib/holding-display'
import { cn } from '@/lib/utils'

interface HoldingReminderPanelProps {
  compact?: boolean
  holdings: HoldingEntry[]
  aggregates: HoldingAggregateEntry[]
  analysesByFundID: Record<string, FundAnalysis | null>
  aggregateMetrics: Record<string, HoldingEstimateAggregateMetrics>
  metricScope: HoldingMetricScope
  exposureSnapshots?: Record<string, HoldingExposureSnapshot>
}

const REMINDER_STORAGE_KEY = 'fundlive.holding.reminder.state.v1'
const REMINDER_SNOOZE_MS = 24 * 60 * 60 * 1000

interface StoredReminderState {
  muted: string[]
  snoozed: Record<string, number>
}

interface ReminderState {
  muted: Set<string>
  snoozed: Record<string, number>
}

function loadReminderState(): ReminderState {
  if (typeof window === 'undefined') {
    return { muted: new Set(), snoozed: {} }
  }

  try {
    const raw = window.localStorage.getItem(REMINDER_STORAGE_KEY)
    if (!raw) {
      return { muted: new Set(), snoozed: {} }
    }
    const parsed = JSON.parse(raw) as Partial<StoredReminderState>
    const now = Date.now()
    const snoozed = Object.fromEntries(
      Object.entries(parsed.snoozed ?? {}).filter(
        ([, expiresAt]) => typeof expiresAt === 'number' && expiresAt > now,
      ),
    )
    return {
      muted: new Set((parsed.muted ?? []).filter(Boolean)),
      snoozed,
    }
  } catch {
    return { muted: new Set(), snoozed: {} }
  }
}

function persistReminderState(state: ReminderState) {
  if (typeof window === 'undefined') {
    return
  }

  const payload: StoredReminderState = {
    muted: Array.from(state.muted),
    snoozed: state.snoozed,
  }
  window.localStorage.setItem(REMINDER_STORAGE_KEY, JSON.stringify(payload))
}

function toneMeta(tone: InsightTone) {
  switch (tone) {
    case 'danger':
      return {
        className: 'border-rose-400/28 bg-rose-500/10 text-rose-100',
        icon: ShieldAlert,
      }
    case 'warning':
      return {
        className: 'border-amber-400/28 bg-amber-400/10 text-amber-100',
        icon: AlertTriangle,
      }
    case 'good':
      return {
        className: 'border-emerald-400/24 bg-emerald-400/10 text-emerald-100',
        icon: CheckCircle2,
      }
    case 'info':
    default:
      return {
        className: 'border-cyan-400/24 bg-cyan-400/10 text-cyan-100',
        icon: Info,
      }
  }
}

export function HoldingReminderPanel({
  compact = false,
  holdings,
  aggregates,
  analysesByFundID,
  aggregateMetrics,
  metricScope,
  exposureSnapshots = {},
}: HoldingReminderPanelProps) {
  const [reminderState, setReminderState] = useState<ReminderState>(() =>
    loadReminderState(),
  )
  const summary = buildHoldingReminderSummary({
    holdings,
    aggregates,
    analysesByFundID,
    aggregateMetrics,
    metricScope,
    exposureSnapshots,
  })
  const visibleReminders = useMemo(
    () =>
      summary.reminders.filter(
        (reminder) =>
          !reminderState.muted.has(reminder.id) &&
          !(reminder.id in reminderState.snoozed),
      ),
    [reminderState, summary.reminders],
  )
  const hiddenCount = summary.reminders.length - visibleReminders.length
  const visibleUrgentCount = visibleReminders.filter(
    (reminder) => reminder.tone === 'danger' || reminder.tone === 'warning',
  ).length

  const updateReminderState = (
    updater: (current: ReminderState) => ReminderState,
  ) => {
    setReminderState((current) => {
      const next = updater(current)
      persistReminderState(next)
      return next
    })
  }

  const muteReminder = (reminderID: string) => {
    updateReminderState((current) => {
      const muted = new Set(current.muted)
      muted.add(reminderID)
      return { muted, snoozed: { ...current.snoozed } }
    })
  }

  const snoozeReminder = (reminderID: string) => {
    updateReminderState((current) => ({
      muted: new Set(current.muted),
      snoozed: {
        ...current.snoozed,
        [reminderID]: Date.now() + REMINDER_SNOOZE_MS,
      },
    }))
  }

  const restoreReminders = () => {
    updateReminderState(() => ({ muted: new Set(), snoozed: {} }))
  }

  if (holdings.length === 0) {
    return null
  }

  return (
    <section className="mb-6 overflow-hidden rounded-[30px] border border-[var(--card-border)] bg-[var(--card-bg)]/84 p-5 glass">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <div className="inline-flex items-center gap-2 rounded-full border border-rose-400/25 bg-rose-400/10 px-3 py-1 text-[11px] font-medium tracking-[0.2em] text-rose-100">
            <BellRing className="h-3.5 w-3.5" />
            持仓提醒
          </div>
          <h3 className="mt-3 text-2xl font-black text-theme-primary">
            风险提醒
          </h3>
          {!compact && (
            <p className="mt-2 max-w-3xl text-sm leading-6 text-theme-secondary">
              展示待补齐、跌幅、风险升高和主题事件。
            </p>
          )}
          {hiddenCount > 0 && (
            <div className="mt-3 flex flex-wrap items-center gap-2 text-xs text-theme-muted">
              已读或稍后隐藏 {hiddenCount} 条
              <button
                type="button"
                onClick={restoreReminders}
                className="rounded-full border border-rose-300/22 bg-rose-400/10 px-3 py-1 font-semibold text-rose-100 transition hover:border-rose-200/45"
              >
                恢复全部提醒
              </button>
            </div>
          )}
        </div>
        <div className="rounded-[24px] border border-rose-300/20 bg-rose-400/10 px-4 py-3 text-center">
          <div className="text-xs text-theme-muted">需优先关注</div>
          <div className="mt-1 text-3xl font-black text-theme-primary">
            {visibleUrgentCount}
          </div>
        </div>
      </div>

      <div className="mt-5 grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        {visibleReminders.length === 0 ? (
          <div className="rounded-[24px] border border-emerald-400/22 bg-emerald-400/10 p-4 text-sm text-emerald-100 md:col-span-2 xl:col-span-3">
            <div className="flex items-center gap-2 font-semibold">
              <CheckCircle2 className="h-4 w-4" />
              {summary.reminders.length === 0
                ? '暂无需要提醒的异常项'
                : '当前提醒已处理'}
            </div>
            <p className="mt-2 text-xs leading-5 text-theme-secondary">
              {summary.reminders.length === 0
                ? '组合数据齐备且没有明显风险提醒时，这里会保持安静。'
                : '已读或稍后提醒会在本机保留；需要重新查看时可恢复全部提醒。'}
            </p>
          </div>
        ) : (
          visibleReminders.map((reminder) => {
            const meta = toneMeta(reminder.tone)
            const Icon = meta.icon
            return (
              <div
                key={reminder.id}
                className={cn('rounded-[24px] border p-4', meta.className)}
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="flex min-w-0 items-center gap-2">
                    <Icon className="h-4 w-4 shrink-0" />
                    <div className="truncate text-sm font-semibold text-theme-primary">
                      {reminder.title}
                    </div>
                  </div>
                  {reminder.metric && (
                    <span className="shrink-0 rounded-full border border-white/15 bg-white/8 px-2 py-0.5 text-[11px]">
                      {reminder.metric}
                    </span>
                  )}
                </div>
                <p className="mt-2 text-xs leading-5 text-theme-secondary">
                  {reminder.description}
                </p>
                {reminder.action && (
                  <div className="mt-3 text-xs font-medium text-theme-primary">
                    {reminder.action}
                  </div>
                )}
                <div className="mt-4 flex flex-wrap gap-2">
                  <button
                    type="button"
                    onClick={() => snoozeReminder(reminder.id)}
                    className="rounded-2xl border border-white/15 bg-white/8 px-3 py-1.5 text-[11px] font-semibold transition hover:border-white/35 hover:bg-white/12"
                  >
                    稍后提醒
                  </button>
                  <button
                    type="button"
                    onClick={() => muteReminder(reminder.id)}
                    className="rounded-2xl border border-white/15 bg-white/8 px-3 py-1.5 text-[11px] font-semibold transition hover:border-white/35 hover:bg-white/12"
                  >
                    已读
                  </button>
                </div>
              </div>
            )
          })
        )}
      </div>
    </section>
  )
}
