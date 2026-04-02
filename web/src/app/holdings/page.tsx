'use client'

import Link from 'next/link'
import { useEffect, useMemo, useRef, useState } from 'react'
import { AlertTriangle, BarChart4, CalendarDays, CheckCircle2, Clock3, LoaderCircle, Plus, Wallet } from 'lucide-react'
import { AccountAreaShell } from '@/components/account-area-shell'
import { HoldingFundRow } from '@/components/holding-fund-row'
import { VIPAnalysisEntry } from '@/components/vip-analysis-entry'
import { useCurrentUser } from '@/hooks/use-auth'
import { useFundSearch } from '@/hooks/use-fund-data'
import { useMarketStatus, usePricingDatePreview } from '@/hooks/use-market-status'
import { useUserPortfolio } from '@/hooks/use-user-portfolio'
import { cn } from '@/lib/utils'

const BEIJING_OFFSET = '+08:00'
type TradeTiming = 'before_close' | 'after_close'

function buildTradeAtValue(date: string, tradeTiming: TradeTiming) {
  if (!date) {
    return ''
  }

  const marker = tradeTiming === 'before_close' ? '14:59:00' : '15:01:00'
  return `${date}T${marker}${BEIJING_OFFSET}`
}

function formatTradeDateLabel(date: string) {
  if (!date) {
    return '选择交易日期'
  }

  const parsed = new Date(`${date}T12:00:00${BEIJING_OFFSET}`)
  if (Number.isNaN(parsed.getTime())) {
    return date
  }

  return new Intl.DateTimeFormat('zh-CN', {
    timeZone: 'Asia/Shanghai',
    month: 'long',
    day: 'numeric',
    weekday: 'short',
  }).format(parsed)
}

function formatTradeTimingLabel(tradeTiming: TradeTiming) {
  if (tradeTiming === 'after_close') {
    return '15:00 后'
  }

  return '15:00 前'
}

function resolveTradeTimingFromServerClock(currentTime: Date) {
  const beijingTime = currentTime.toLocaleTimeString('en-GB', {
    timeZone: 'Asia/Shanghai',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  })

  return beijingTime >= '15:00' ? 'after_close' : 'before_close'
}

export default function HoldingsPage() {
  const { user, isLoading } = useCurrentUser()
  const marketStatus = useMarketStatus()
  const { holdings, seedDemoData, addHolding, removeHolding } = useUserPortfolio(user?.id ?? null)
  const [query, setQuery] = useState('')
  const [selectedFundID, setSelectedFundID] = useState('')
  const [selectedFundName, setSelectedFundName] = useState('')
  const [amount, setAmount] = useState('')
  const [tradeDate, setTradeDate] = useState('')
  const [tradeTiming, setTradeTiming] = useState<TradeTiming>('before_close')
  const [note, setNote] = useState('')
  const [feedback, setFeedback] = useState<{ type: 'success' | 'error'; message: string } | null>(null)
  const [isSeedingDemo, setIsSeedingDemo] = useState(false)
  const [isAddingHolding, setIsAddingHolding] = useState(false)
  const defaultsInitializedRef = useRef(false)
  const { results } = useFundSearch(query)
  const normalizedQuery = query.trim()

  const autoMatchedFund = useMemo(() => {
    if (!normalizedQuery) {
      return null
    }

    const exactMatch = results.find((fund) => fund.id === normalizedQuery || fund.name === normalizedQuery)
    if (exactMatch) {
      return exactMatch
    }

    if (results.length === 1) {
      return results[0]
    }

    return null
  }, [normalizedQuery, results])

  useEffect(() => {
    if (defaultsInitializedRef.current || !marketStatus.currentDate || marketStatus.currentTime.getTime() === 0) {
      return
    }

    setTradeDate(marketStatus.currentDate)
    setTradeTiming(resolveTradeTimingFromServerClock(marketStatus.currentTime))
    defaultsInitializedRef.current = true
  }, [marketStatus.currentDate, marketStatus.currentTime])

  const resolvedFundID = selectedFundID || autoMatchedFund?.id || ''
  const resolvedFundName = selectedFundName || autoMatchedFund?.name || ''
  const tradeAtPayload = buildTradeAtValue(tradeDate, tradeTiming)
  const {
    preview,
    isLoading: isPricingPreviewLoading,
    error: pricingPreviewError,
  } = usePricingDatePreview(tradeAtPayload || null)
  const pricingDatePreview = preview?.pricingDate || ''
  const tradeDateLabel = formatTradeDateLabel(tradeDate)
  const tradeTimingLabel = formatTradeTimingLabel(tradeTiming)
  const todayTradeDate = marketStatus.currentDate || tradeDate
  const previousTradeDate = marketStatus.previousTradingDay || ''
  const nextTradeDate = marketStatus.nextTradingDay || ''
  const pricingRuleLabel = !tradeDate
    ? '选择交易日期和提交时段后，会自动预览确认净值日。'
    : pricingPreviewError
      ? pricingPreviewError.message
    : isPricingPreviewLoading
      ? '正在按后端交易日历校验确认净值日...'
      : preview?.message || '正在按后端交易日历校验确认净值日...'

  const handleSeedDemo = async () => {
    setFeedback(null)
    setIsSeedingDemo(true)

    try {
      await seedDemoData()
      setFeedback({
        type: 'success',
        message: '已载入演示持仓。若你没有自选分组，也会一并创建演示分组。',
      })
    } catch (error) {
      setFeedback({
        type: 'error',
        message: error instanceof Error ? error.message : '载入演示持仓失败，请稍后重试。',
      })
    } finally {
      setIsSeedingDemo(false)
    }
  }

  const handleAddHolding = async () => {
    setFeedback(null)

    if (!resolvedFundID) {
      setFeedback({
        type: 'error',
        message: '请先从搜索结果中选择基金，或输入能唯一匹配的基金代码/名称。',
      })
      return
    }

    if (!amount.trim()) {
      setFeedback({
        type: 'error',
        message: '请输入有效的持仓金额。',
      })
      return
    }

    if (!tradeDate.trim()) {
      setFeedback({
        type: 'error',
        message: '请选择交易日期。',
      })
      return
    }

    setIsAddingHolding(true)

    try {
      await addHolding(resolvedFundID, amount, tradeAtPayload, note)
      setSelectedFundID('')
      setSelectedFundName('')
      setQuery('')
      setAmount('')
      setTradeDate(marketStatus.currentDate || tradeDate)
      setTradeTiming(
        marketStatus.currentTime.getTime() === 0
          ? 'before_close'
          : resolveTradeTimingFromServerClock(marketStatus.currentTime)
      )
      setNote('')
      setFeedback({
        type: 'success',
        message: pricingDatePreview
          ? `已加入 ${resolvedFundName || resolvedFundID} 的持仓记录，将按 ${pricingDatePreview} 收盘净值确认。`
          : `已加入 ${resolvedFundName || resolvedFundID} 的持仓记录，确认净值日已按服务端交易日历计算。`,
      })
    } catch (error) {
      setFeedback({
        type: 'error',
        message: error instanceof Error ? error.message : '加入持仓失败，请稍后重试。',
      })
    } finally {
      setIsAddingHolding(false)
    }
  }

  if (isLoading) {
    return (
      <AccountAreaShell title="持仓明细" description="按基金记录你的持仓金额，并实时查看预估涨跌额。">
        <div className="glass h-64 rounded-[32px] border border-[var(--card-border)]" />
      </AccountAreaShell>
    )
  }

  if (!user) {
    return (
      <AccountAreaShell title="持仓明细" description="按基金记录你的持仓金额，并实时查看预估涨跌额。">
        <div className="glass rounded-[32px] border border-[var(--card-border)] p-8 text-center">
          <div className="mb-3 text-2xl font-bold text-theme-primary">登录后可查看持仓明细</div>
          <p className="mx-auto max-w-xl text-sm leading-6 text-theme-secondary">
            持仓明细现在已经绑定到账户，可以在服务端持久化保存你的基金持仓记录。
          </p>
          <div className="mt-6 flex justify-center gap-3">
            <Link href="/auth/login" className="rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-sm text-theme-secondary">
              去登录
            </Link>
            <Link href="/auth/register" className="rounded-2xl bg-gradient-to-r from-cyan-500 via-sky-500 to-blue-600 px-4 py-3 text-sm font-medium text-white">
              去注册
            </Link>
          </div>
        </div>
      </AccountAreaShell>
    )
  }

  return (
    <AccountAreaShell title="持仓明细" description="记录你的持仓金额、日期与备注，快速看到每只基金的实时预估涨跌额。数据已改为服务端存储。">
      <div className="space-y-8">
        {feedback && (
          <div
            className={cn(
              'flex items-start gap-3 rounded-[28px] border px-4 py-4 text-sm',
              feedback.type === 'success'
                ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-50'
                : 'border-amber-500/30 bg-amber-500/10 text-amber-100'
            )}
          >
            {feedback.type === 'success' ? (
              <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0" />
            ) : (
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
            )}
            <span>{feedback.message}</span>
          </div>
        )}

        <div className="grid gap-6 xl:grid-cols-[1.65fr_0.85fr]">
          <section className="rounded-[32px] border border-[var(--card-border)] p-6 glass">
            <div className="mb-6 flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
              <div>
                <div className="text-sm text-theme-muted">持仓总览</div>
                <div className="mt-1 text-3xl font-black text-theme-primary">{holdings.length} 条持仓记录</div>
              </div>

              {holdings.length === 0 && (
                <button
                  type="button"
                  onClick={() => void handleSeedDemo()}
                  disabled={isSeedingDemo}
                  className={cn(
                    'group relative inline-flex items-center gap-2 overflow-hidden rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-sm text-theme-secondary transition-all duration-200',
                    'hover:-translate-y-0.5 hover:border-cyan-400/45 hover:text-theme-primary hover:shadow-[0_14px_30px_rgba(34,211,238,0.12)]',
                    'active:scale-[0.985] disabled:cursor-not-allowed disabled:opacity-80',
                    isSeedingDemo && 'holding-action-button'
                  )}
                >
                  <span className="holding-action-shine" />
                  {isSeedingDemo ? (
                    <LoaderCircle className="relative z-10 h-4 w-4 animate-spin" />
                  ) : (
                    <BarChart4 className="relative z-10 h-4 w-4 transition-transform duration-300 group-hover:-rotate-6 group-hover:scale-110" />
                  )}
                  <span className="relative z-10">{isSeedingDemo ? '载入中...' : '载入演示持仓'}</span>
                </button>
              )}
            </div>

            <div className="space-y-5">
              <div className="grid gap-4 lg:grid-cols-3">
                <div className="space-y-2">
                  <div className="text-sm text-theme-secondary">选择基金</div>
                  <input
                    value={query}
                    onChange={(event) => {
                      setQuery(event.target.value)
                      setSelectedFundID('')
                      setSelectedFundName('')
                    }}
                    placeholder="搜索基金代码或名称"
                    className="auth-input w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-theme-primary outline-none placeholder:text-theme-muted"
                  />
                  <div className="truncate text-xs text-theme-muted">
                    {resolvedFundID ? `${resolvedFundName || resolvedFundID} · ${resolvedFundID}` : '先选择或唯一匹配一只基金'}
                  </div>
                </div>

                <div className="space-y-2">
                  <div className="text-sm text-theme-secondary">持仓金额</div>
                  <input
                    value={amount}
                    onChange={(event) => setAmount(event.target.value)}
                    placeholder="例如 30000"
                    className="auth-input w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-theme-primary outline-none placeholder:text-theme-muted"
                  />
                  <div className="text-xs text-theme-muted">录入基金申购或持仓金额，单位为人民币。</div>
                </div>

                <div className="space-y-2">
                  <div className="text-sm text-theme-secondary">备注</div>
                  <input
                    value={note}
                    onChange={(event) => setNote(event.target.value)}
                    placeholder="例如：长期底仓"
                    className="auth-input w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-theme-primary outline-none placeholder:text-theme-muted"
                  />
                  <div className="text-xs text-theme-muted">可选，用于标记策略、来源或补仓背景。</div>
                </div>
              </div>

              <div className="grid gap-5 lg:grid-cols-[0.96fr_1.04fr]">
                <div className="space-y-4">
                  <div className="rounded-[28px] border border-[var(--card-border)] bg-[var(--card-bg)]/76 p-4">
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <div className="text-sm font-medium text-theme-primary">基金搜索结果</div>
                        <div className="mt-1 text-xs leading-5 text-theme-muted">
                          选择后会自动填入右侧确认信息和新增卡片。
                        </div>
                      </div>
                      <div className="rounded-full border border-cyan-400/20 bg-cyan-400/10 px-3 py-1 text-[11px] tracking-[0.18em] text-cyan-200">
                        TOP 5
                      </div>
                    </div>

                    <div className="mt-4 space-y-2">
                      {results.slice(0, 5).map((fund) => (
                        <button
                          key={fund.id}
                          type="button"
                          onClick={() => {
                            setSelectedFundID(fund.id)
                            setSelectedFundName(fund.name)
                            setQuery(fund.name)
                            setFeedback(null)
                          }}
                          className="flex w-full items-center justify-between rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-left transition-colors hover:border-cyan-500/40"
                        >
                          <div className="min-w-0">
                            <div className="truncate text-sm font-medium text-theme-primary">{fund.name}</div>
                            <div className="mt-1 text-xs text-theme-muted">{fund.id}</div>
                          </div>
                          <Plus className="h-4 w-4 shrink-0 text-cyan-300" />
                        </button>
                      ))}

                      {results.length === 0 && (
                        <div className="rounded-2xl border border-dashed border-[var(--card-border)] px-4 py-8 text-center text-sm text-theme-secondary">
                          输入基金代码或名称后，这里会展示可选结果。
                        </div>
                      )}
                    </div>
                  </div>

                  <div className="rounded-[28px] border border-[var(--card-border)] bg-[var(--input-bg)]/70 p-5">
                    <div className="text-sm text-theme-muted">准备新增</div>
                    <div className="mt-2 text-lg font-bold text-theme-primary">{resolvedFundID || '未选择基金'}</div>
                    <div className="mt-1 text-xs text-theme-muted">
                      {resolvedFundName || (resolvedFundID ? '已解析到基金代码' : '请先选择或唯一匹配一只基金')}
                    </div>
                    <div className="mt-3 rounded-2xl border border-[var(--card-border)] bg-[var(--card-bg)]/70 px-4 py-3">
                      <div className="text-xs text-theme-muted">将按以下净值日确认</div>
                      <div className="mt-1 text-base font-semibold text-theme-primary">{pricingDatePreview || '--'}</div>
                      <div className="mt-1 text-xs text-theme-secondary">{pricingRuleLabel}</div>
                    </div>
                    <p className="mt-2 text-sm leading-6 text-theme-secondary">
                      当前版本会把交易日期和提交时段一并保存，后端自动计算确认净值日，便于后续收益分析和持仓回溯。
                    </p>
                    <button
                      type="button"
                      onClick={() => void handleAddHolding()}
                      disabled={isAddingHolding}
                      className={cn(
                        'group relative mt-5 inline-flex w-full items-center justify-center gap-2 overflow-hidden rounded-2xl bg-gradient-to-r from-cyan-500 via-sky-500 to-blue-600 px-4 py-3 text-sm font-medium text-white transition-all duration-200',
                        'hover:-translate-y-0.5 hover:shadow-[0_18px_35px_rgba(14,165,233,0.28)]',
                        'active:scale-[0.985] disabled:cursor-not-allowed disabled:opacity-85',
                        isAddingHolding && 'holding-action-button'
                      )}
                      aria-busy={isAddingHolding}
                    >
                      <span className="holding-action-shine" />
                      {isAddingHolding ? (
                        <LoaderCircle className="relative z-10 h-4 w-4 animate-spin" />
                      ) : (
                        <Wallet className="relative z-10 h-4 w-4 transition-transform duration-300 group-hover:-rotate-6 group-hover:scale-110" />
                      )}
                      <span className="relative z-10">{isAddingHolding ? '提交中...' : '加入持仓'}</span>
                    </button>
                  </div>
                </div>

                <div className="rounded-[30px] border border-[var(--card-border)] bg-[var(--card-bg)]/88 p-5">
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <div className="text-sm font-medium text-theme-primary">交易时间</div>
                      <div className="mt-1 text-xs leading-5 text-theme-muted">
                        按北京时间 15:00 截止规则自动推导确认净值日。
                      </div>
                    </div>
                    <div className="rounded-full border border-cyan-400/30 bg-cyan-400/10 px-3 py-1 text-[11px] font-medium tracking-[0.18em] text-cyan-200">
                      T+1
                    </div>
                  </div>

                  <div className="mt-5 grid gap-4 xl:grid-cols-[1.08fr_0.92fr]">
                    <label className="holding-picker-shell relative overflow-hidden rounded-[24px] border border-[var(--input-border)] bg-[var(--input-bg)]/85 px-4 py-4 transition-all duration-200 hover:border-cyan-400/40 hover:bg-[var(--input-bg)] focus-within:border-cyan-400/60 focus-within:bg-[var(--input-bg)] focus-within:shadow-[0_14px_30px_rgba(34,211,238,0.12)]">
                      <span className="holding-picker-shine" />
                      <span className="relative z-10 flex items-center gap-2 text-xs font-medium text-theme-secondary">
                        <CalendarDays className="h-3.5 w-3.5 text-cyan-300" />
                        交易日期
                      </span>
                      <div className="relative z-10 mt-4 rounded-[20px] border border-[var(--input-border)] bg-[var(--card-bg)]/85 px-4 py-3">
                        <input
                          type="date"
                          value={tradeDate}
                          onChange={(event) => setTradeDate(event.target.value)}
                          className="holding-datetime-input w-full text-sm font-medium text-theme-primary outline-none"
                        />
                      </div>
                      <div className="relative z-10 mt-3 text-xs text-theme-secondary">
                        当前选择：<span className="font-medium text-theme-primary">{tradeDateLabel}</span>
                      </div>
                    </label>

                    <div className="rounded-[24px] border border-[var(--input-border)] bg-[var(--input-bg)]/78 px-4 py-4">
                      <div className="flex items-center gap-2 text-xs font-medium text-theme-secondary">
                        <Clock3 className="h-3.5 w-3.5 text-cyan-300" />
                        提交时段
                      </div>
                      <div className="mt-4 grid gap-3">
                        {([
                          {
                            id: 'before_close',
                            title: '15:00 前',
                            description: '按当日收盘净值确认，适合当日交易提交',
                          },
                          {
                            id: 'after_close',
                            title: '15:00 后',
                            description: '顺延至下个交易日确认，适合收盘后录入',
                          },
                        ] as const).map((option) => (
                          <button
                            key={option.id}
                            type="button"
                            onClick={() => setTradeTiming(option.id)}
                            className={cn(
                              'rounded-[20px] border px-4 py-3 text-left transition-all duration-200',
                              tradeTiming === option.id
                                ? 'border-cyan-400/55 bg-cyan-400/14 text-cyan-100 shadow-[0_12px_26px_rgba(34,211,238,0.12)]'
                                : 'border-[var(--input-border)] bg-[var(--card-bg)]/72 text-theme-secondary hover:border-cyan-400/35 hover:text-theme-primary'
                            )}
                            aria-pressed={tradeTiming === option.id}
                          >
                            <div className="text-sm font-semibold">{option.title}</div>
                            <div className="mt-1 text-xs leading-5 text-theme-muted">{option.description}</div>
                          </button>
                        ))}
                      </div>
                    </div>
                  </div>

                  <div className="mt-4 flex flex-wrap gap-2">
                    {['今天', '上个交易日', '下个交易日'].map((shortcut) => (
                      <button
                        key={shortcut}
                        type="button"
                        onClick={() => {
                          if (shortcut === '今天') {
                            setTradeDate(todayTradeDate)
                            return
                          }

                          if (shortcut === '上个交易日') {
                            setTradeDate(previousTradeDate)
                            return
                          }

                          setTradeDate(nextTradeDate)
                        }}
                        className={cn(
                          'rounded-full border px-3 py-1.5 text-xs transition-all duration-200',
                          (shortcut === '今天' && tradeDate === todayTradeDate) ||
                          (shortcut === '上个交易日' && tradeDate === previousTradeDate) ||
                          (shortcut === '下个交易日' && tradeDate === nextTradeDate)
                            ? 'border-cyan-400/50 bg-cyan-400/15 text-cyan-100 shadow-[0_10px_22px_rgba(34,211,238,0.12)]'
                            : 'border-[var(--input-border)] bg-[var(--input-bg)]/70 text-theme-secondary hover:border-cyan-400/35 hover:text-theme-primary'
                        )}
                      >
                        {shortcut}
                      </button>
                    ))}
                  </div>

                  <div className="mt-4 rounded-[22px] border border-cyan-400/18 bg-cyan-400/8 px-4 py-4">
                      <div className="text-[11px] font-medium tracking-[0.18em] text-cyan-200">净值确认预览</div>
                      <div className="mt-3 flex items-start justify-between gap-4">
                        <div className="min-w-0">
                        <div className="text-sm font-medium text-theme-primary">{tradeDate ? `${tradeDate} · ${tradeTimingLabel}` : '请选择交易日期'}</div>
                        <div className="mt-1 text-xs leading-5 text-theme-secondary">{pricingRuleLabel}</div>
                      </div>
                      <div className="shrink-0 text-right">
                        <div className="text-xs text-theme-muted">确认净值日</div>
                        <div className="mt-1 text-lg font-semibold text-cyan-100">{pricingDatePreview || '--'}</div>
                        </div>
                      </div>
                    </div>
                </div>
              </div>
            </div>
          </section>
        </div>

        {holdings.length === 0 ? (
          <div className="rounded-[32px] border border-dashed border-[var(--card-border)] p-10 text-center glass">
            <Wallet className="mx-auto h-10 w-10 text-theme-muted" />
            <div className="mt-4 text-xl font-semibold text-theme-primary">还没有持仓记录</div>
            <p className="mt-2 text-sm leading-6 text-theme-secondary">
              你可以在上方选择基金、录入持仓金额和日期。当前版本会把数据保存到服务端。
            </p>
          </div>
        ) : (
          <div className="space-y-4">
            {holdings.map((holding) => (
              <HoldingFundRow
                key={holding.id}
                holding={holding}
                onRemove={() => removeHolding(holding.id)}
              />
            ))}
          </div>
        )}

        <VIPAnalysisEntry
          title="AI 持仓分析入口"
          description="这里预留给后续的 VIP 功能，结合你的持仓结构、基金风格和大盘走势生成行情分析与建议。当前版本只保留可见入口，不接实际 AI 能力。"
          accent="amber"
        />
      </div>
    </AccountAreaShell>
  )
}
