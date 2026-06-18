import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
    return twMerge(clsx(inputs))
}

// Format a decimal string as percentage with color class
export function formatPercent(value: string | undefined): { text: string; isPositive: boolean } {
    if (value === undefined || value === null || value === '') return { text: '--', isPositive: false }

    const num = parseFloat(value)
    if (!Number.isFinite(num)) return { text: '--', isPositive: false }

    const isPositive = num >= 0
    const prefix = isPositive ? '+' : ''

    return {
        text: `${prefix}${num.toFixed(2)}%`,
        isPositive
    }
}

// Format a decimal string as currency (CNY). Missing financial values must stay unknown, not zero.
export function formatCurrency(value: string | undefined): string {
    if (value === undefined || value === null || value === '') return '--'

    const num = parseFloat(value)
    if (!Number.isFinite(num)) return '--'

    return `¥${num.toFixed(4)}`
}

// Format stock holding ratio
export function formatRatio(value: string | undefined): string {
    if (value === undefined || value === null || value === '') return '--'

    const num = parseFloat(value)
    if (!Number.isFinite(num)) return '--'

    return `${num.toFixed(2)}%`
}

// Format timestamp to readable time
export function formatTime(timestamp: string | Date | undefined): string {
    if (!timestamp) return '--:--'

    const date = typeof timestamp === 'string' ? new Date(timestamp) : timestamp
    return date.toLocaleTimeString('zh-CN', {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
    })
}

// Calculate countdown to next refresh
export function calculateCountdown(lastUpdated: Date | null, intervalSeconds: number): number {
    if (!lastUpdated) return intervalSeconds

    const elapsed = Math.floor((Date.now() - lastUpdated.getTime()) / 1000)
    const remaining = intervalSeconds - elapsed

    return remaining > 0 ? remaining : 0
}
