'use client'

import { useMemo, memo } from 'react'
import { cn } from '@/lib/utils'
import { useMarketTradingState } from '@/hooks/use-market-status'
import type { TimeSeriesPoint, FundEstimate } from '@/hooks/use-fund-data'
import {
    ResponsiveContainer,
    ComposedChart,
    Line,
    XAxis,
    YAxis,
    Tooltip,
    ReferenceLine,
    CartesianGrid,
} from 'recharts'

// ====================
// Constants & Types
// ====================

// A-Share Trading Hours (in minutes from midnight)
const TRADING_TIMES = {
    MORNING_START: 9 * 60 + 30,   // 09:30
    MORNING_END: 11 * 60 + 30,    // 11:30
    AFTERNOON_START: 13 * 60,      // 13:00
    AFTERNOON_END: 15 * 60,        // 15:00
} as const

// Fixed X-Axis ticks for A-Share trading day
const X_AXIS_TICKS = ['09:30', '10:00', '10:30', '11:00', '11:30', '13:00', '13:30', '14:00', '14:30', '15:00']

interface ChartDataPoint {
    time: string        // Format: HH:mm
    timeMinutes: number // Minutes from midnight (for sorting)
    change: number | null
    nav: number | null
    isMorning?: boolean
    isAfternoon?: boolean
}

interface IntradayChartProps {
    timeSeries: TimeSeriesPoint[]
    estimate?: FundEstimate
    isLoading?: boolean
    isCallAuction?: boolean
    displayDate?: string
    isHistorical?: boolean
    session?: string
    className?: string
}

// ====================
// Utility Functions
// ====================

/**
 * Format minutes to HH:mm string
 */
function formatMinutesToTime(minutes: number): string {
    const h = Math.floor(minutes / 60)
    const m = minutes % 60
    return `${h.toString().padStart(2, '0')}:${m.toString().padStart(2, '0')}`
}

/**
 * Generate base chart data with all trading time slots (empty initially)
 * This ensures the X-axis always shows 09:30-15:00 regardless of data availability
 */
function generateEmptyTradingDayData(): ChartDataPoint[] {
    const slots: ChartDataPoint[] = []

    // Morning session: 09:30 - 11:30 (every 5 minutes)
    for (let min = TRADING_TIMES.MORNING_START; min <= TRADING_TIMES.MORNING_END; min += 5) {
        slots.push({
            time: formatMinutesToTime(min),
            timeMinutes: min,
            change: null,
            nav: null,
            isMorning: true,
        })
    }

    // Afternoon session: 13:00 - 15:00 (every 5 minutes)
    for (let min = TRADING_TIMES.AFTERNOON_START; min <= TRADING_TIMES.AFTERNOON_END; min += 5) {
        slots.push({
            time: formatMinutesToTime(min),
            timeMinutes: min,
            change: null,
            nav: null,
            isAfternoon: true,
        })
    }

    return slots
}

/**
 * Round minutes to nearest 5-minute slot
 * e.g., 09:51 -> 09:50, 09:53 -> 09:55
 */
function roundToNearestFiveMinutes(hours: number, minutes: number): string {
    const roundedMinutes = Math.round(minutes / 5) * 5
    const adjustedHours = roundedMinutes === 60 ? hours + 1 : hours
    const adjustedMinutes = roundedMinutes === 60 ? 0 : roundedMinutes
    return `${adjustedHours.toString().padStart(2, '0')}:${adjustedMinutes.toString().padStart(2, '0')}`
}

/**
 * Merge real data into the empty trading day template
 */
function mergeDataIntoTemplate(
    template: ChartDataPoint[],
    realData: TimeSeriesPoint[]
): ChartDataPoint[] {
    // Create a lookup map for real data by time slot (HH:mm format, rounded to 5 min)
    const dataMap = new Map<string, { change: number; nav: number }>()

    for (const point of realData) {
        const date = new Date(point.timestamp)
        const hours = date.getHours()
        const minutes = date.getMinutes()

        // Round to nearest 5-minute slot
        const timeSlot = roundToNearestFiveMinutes(hours, minutes)

        // Update with latest value for each slot (later data overwrites earlier)
        dataMap.set(timeSlot, {
            change: parseFloat(point.change_percent),
            nav: parseFloat(point.estimate_nav),
        })
    }

    // Fill in the template with real data where available
    return template.map(slot => {
        const realPoint = dataMap.get(slot.time)
        if (realPoint) {
            return {
                ...slot,
                change: realPoint.change,
                nav: realPoint.nav,
            }
        }
        return slot
    })
}

/**
 * Prepare chart data with separate fields for morning/afternoon sessions
 * This allows proper rendering in ComposedChart with gap during lunch break
 */
function prepareChartDataWithSessions(data: ChartDataPoint[]): ChartDataPoint[] {
    return data.map(d => ({
        ...d,
        // Split change into morning and afternoon fields
        morningChange: d.isMorning ? d.change : null,
        afternoonChange: d.isAfternoon ? d.change : null,
    }))
}

// ====================
// Chart Components
// ====================

interface PreparedChartDataPoint extends ChartDataPoint {
    morningChange: number | null
    afternoonChange: number | null
}

const ChartContent = memo(function ChartContent({
    chartData,
    isPositive,
    yDomain,
}: {
    chartData: ChartDataPoint[]
    isPositive: boolean
    yDomain: [number, number]
}) {
    // Prepare data with separate morning/afternoon fields
    const preparedData = useMemo(
        () => prepareChartDataWithSessions(chartData) as PreparedChartDataPoint[],
        [chartData]
    )
    const strokeColor = isPositive ? 'var(--accent-up, #f43f5e)' : 'var(--accent-down, #10b981)'

    return (
        <ResponsiveContainer width="100%" height="100%">
            <ComposedChart
                data={preparedData}
                margin={{ top: 10, right: 10, left: 0, bottom: 0 }}
            >
                <defs>
                    <linearGradient id="positiveGradient" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="var(--accent-up, #f43f5e)" stopOpacity={0.3} />
                        <stop offset="95%" stopColor="var(--accent-up, #f43f5e)" stopOpacity={0} />
                    </linearGradient>
                    <linearGradient id="negativeGradient" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="var(--accent-down, #10b981)" stopOpacity={0.3} />
                        <stop offset="95%" stopColor="var(--accent-down, #10b981)" stopOpacity={0} />
                    </linearGradient>
                </defs>

                <CartesianGrid strokeDasharray="3 3" stroke="var(--card-border)" />

                {/* Fixed X-Axis: Always 09:30 - 15:00 */}
                <XAxis
                    dataKey="time"
                    axisLine={false}
                    tickLine={false}
                    tick={{ fill: 'var(--text-muted)', fontSize: 11 }}
                    ticks={X_AXIS_TICKS}
                    domain={['09:30', '15:00']}
                    type="category"
                    allowDuplicatedCategory={false}
                    interval={0}
                    tickFormatter={(value) => {
                        // Show only key ticks to avoid overlap
                        if (['09:30', '11:30', '13:00', '15:00'].includes(value)) {
                            return value
                        }
                        return ''
                    }}
                />

                <YAxis
                    domain={yDomain}
                    axisLine={false}
                    tickLine={false}
                    tick={{ fill: 'var(--text-muted)', fontSize: 11 }}
                    tickFormatter={(value) => `${value > 0 ? '+' : ''}${value.toFixed(1)}%`}
                    width={50}
                />

                <ReferenceLine
                    y={0}
                    stroke="var(--text-muted)"
                    strokeDasharray="5 5"
                />

                <Tooltip
                    contentStyle={{
                        backgroundColor: 'var(--card-bg)',
                        border: '1px solid var(--card-border)',
                        borderRadius: '8px',
                        color: 'var(--text-primary)',
                    }}
                    formatter={(value) => {
                        if (value === null || value === undefined) return null
                        const numValue = Number(value)
                        if (isNaN(numValue)) return null
                        return [
                            `${numValue >= 0 ? '+' : ''}${numValue.toFixed(2)}%`,
                            '涨跌幅'
                        ]
                    }}
                    labelFormatter={(label) => `时间: ${label}`}
                />

                {/* Morning Session Line (09:30 - 11:30) */}
                <Line
                    type="monotone"
                    dataKey="morningChange"
                    stroke={strokeColor}
                    strokeWidth={2}
                    dot={false}
                    animationDuration={300}
                    connectNulls={false}
                    activeDot={{ r: 4, strokeWidth: 2, fill: 'var(--background)' }}
                />

                {/* Afternoon Session Line (13:00 - 15:00) */}
                <Line
                    type="monotone"
                    dataKey="afternoonChange"
                    stroke={strokeColor}
                    strokeWidth={2}
                    dot={false}
                    animationDuration={300}
                    connectNulls={false}
                    activeDot={{ r: 4, strokeWidth: 2, fill: 'var(--background)' }}
                />
            </ComposedChart>
        </ResponsiveContainer>
    )
})

// ====================
// Main Component
// ====================

export function IntradayChart({
    timeSeries,
    estimate,
    isLoading,
    isCallAuction = false,
    displayDate,
    isHistorical = false,
    className
}: IntradayChartProps) {
    const { isTrading } = useMarketTradingState()

    // Generate chart data with fixed X-axis domain
    const { chartData, hasData } = useMemo(() => {
        // Start with empty trading day template
        const template = generateEmptyTradingDayData()

        // If we have real data, merge it in
        if (timeSeries.length > 0) {
            const merged = mergeDataIntoTemplate(template, timeSeries)
            return { chartData: merged, hasData: true }
        }

        return { chartData: template, hasData: false }
    }, [timeSeries])

    // Determine overall trend direction
    const isPositive = useMemo(() => {
        if (estimate) {
            return parseFloat(estimate.change_percent) >= 0
        }
        // Find the last non-null data point
        for (let i = chartData.length - 1; i >= 0; i--) {
            const changeVal = chartData[i].change
            if (changeVal !== null) {
                return changeVal >= 0
            }
        }
        return true
    }, [estimate, chartData])

    // Calculate Y-axis domain
    const yDomain = useMemo((): [number, number] => {
        const changes = chartData
            .map(d => d.change)
            .filter((c): c is number => c !== null)

        if (changes.length === 0) {
            // No data: show symmetric default range
            return [-1, 1]
        }

        const minChange = Math.min(...changes, 0)
        const maxChange = Math.max(...changes, 0)

        // Add 20% padding and round to tenths
        return [
            Math.floor(minChange * 1.2 * 10) / 10,
            Math.ceil(maxChange * 1.2 * 10) / 10
        ]
    }, [chartData])

    // Format display date for UI
    const formattedDate = useMemo(() => {
        if (!displayDate) return ''
        try {
            const date = new Date(`${displayDate}T12:00:00+08:00`)
            return date.toLocaleDateString('zh-CN', {
                month: 'numeric',
                day: 'numeric',
                weekday: 'short',
                timeZone: 'Asia/Shanghai',
            })
        } catch {
            return displayDate
        }
    }, [displayDate])

    return (
        <div className={cn('glass rounded-2xl p-6', className)}>
            <div className="flex items-center justify-between mb-4">
                <div className="flex items-center gap-2">
                    <h3 className="text-lg font-semibold text-theme-primary">分时走势</h3>
                    {isHistorical && (
                        <span className="px-2 py-0.5 text-xs rounded-full bg-amber-500/20 text-amber-400 border border-amber-500/30">
                            上一交易日
                        </span>
                    )}
                    {displayDate && (
                        <span className="text-xs text-theme-muted">
                            {formattedDate}
                        </span>
                    )}
                </div>
                <div className="flex items-center gap-2 text-xs text-theme-muted">
                    {isCallAuction ? (
                        <span>集合竞价中</span>
                    ) : isTrading ? (
                        <>
                            <span className="inline-block w-2 h-2 rounded-full bg-green-500 live-indicator" />
                            <span>实时更新</span>
                        </>
                    ) : (
                        <span>休市中</span>
                    )}
                </div>
            </div>

            <div className="h-64">
                {isCallAuction ? (
                    <div className="flex h-full items-center justify-center rounded-2xl border border-dashed border-[var(--card-border)] bg-[var(--input-bg)]/40 text-center">
                        <div>
                            <div className="text-lg font-semibold text-theme-primary">集合竞价中</div>
                            <div className="mt-2 text-sm text-theme-secondary">等待 09:30 开盘后更新分时走势图</div>
                        </div>
                    </div>
                ) : isLoading && !hasData ? (
                    <div className="h-full flex items-center justify-center">
                        <div className="skeleton w-full h-full rounded-lg" />
                    </div>
                ) : (
                    <ChartContent
                        chartData={chartData}
                        isPositive={isPositive}
                        yDomain={yDomain}
                    />
                )}
            </div>

            {/* Trading hours indicator */}
            <div className="mt-4 flex justify-between text-xs text-theme-muted">
                <span>09:30</span>
                <span className="text-amber-500/70">— 午休 —</span>
                <span>15:00</span>
            </div>
        </div>
    )
}
