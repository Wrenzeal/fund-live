'use client'

import { cn } from '@/lib/utils'

interface LoadingSpinnerProps {
    size?: 'sm' | 'md' | 'lg'
    className?: string
    text?: string
}

export function LoadingSpinner({ size = 'md', className, text }: LoadingSpinnerProps) {
    const sizeClasses = {
        sm: 'h-4 w-4',
        md: 'h-8 w-8',
        lg: 'h-12 w-12'
    }

    return (
        <div className={cn('flex flex-col items-center justify-center gap-3', className)}>
            <div className={cn('relative', sizeClasses[size])} aria-hidden="true">
                <div className={cn('absolute inset-0 rounded-full border-2 border-[var(--card-border)]', sizeClasses[size])} />
                <div
                    className={cn(
                        'absolute inset-0 rounded-full border-2 border-transparent animate-spin',
                        'border-t-cyan-400 border-r-sky-300',
                        sizeClasses[size]
                    )}
                />
            </div>
            {text && (
                <span className="text-sm text-theme-secondary">{text}</span>
            )}
        </div>
    )
}

interface LoadingOverlayProps {
    isLoading: boolean
    text?: string
    children: React.ReactNode
}

export function LoadingOverlay({ isLoading, text = '正在加载…', children }: LoadingOverlayProps) {
    return (
        <div className="relative">
            {children}
            {isLoading && (
                <div
                    className="absolute inset-0 z-20 flex items-center justify-center rounded-2xl backdrop-blur-sm"
                    style={{ backgroundColor: 'color-mix(in srgb, var(--background) 82%, transparent)' }}
                >
                    <LoadingSpinner size="lg" text={text} />
                </div>
            )}
        </div>
    )
}

interface LayoutSkeletonProps {
    className?: string
    rows?: number
    dense?: boolean
}

export function LayoutSkeleton({ className, rows = 4, dense = false }: LayoutSkeletonProps) {
    return (
        <div className={cn('rounded-[28px] border border-[var(--card-border)] p-5 glass', className)} aria-hidden="true">
            <div className="flex items-start justify-between gap-4">
                <div className="space-y-3">
                    <div className="skeleton-line h-3 w-28 rounded-full" />
                    <div className="skeleton-line h-7 w-48 rounded-full" />
                </div>
                <div className="skeleton-block h-12 w-12 rounded-2xl" />
            </div>
            <div className={cn('mt-6 grid gap-3', dense ? 'grid-cols-2' : 'sm:grid-cols-3')}>
                {Array.from({ length: rows }).map((_, index) => (
                    <div key={index} className="rounded-2xl border border-[var(--card-border)] bg-[var(--input-bg)]/55 p-4">
                        <div className="skeleton-line h-3 w-20 rounded-full" />
                        <div className="skeleton-line mt-4 h-6 w-24 rounded-full" />
                    </div>
                ))}
            </div>
            <div className="mt-6 space-y-3">
                <div className="skeleton-line h-3 w-full rounded-full" />
                <div className="skeleton-line h-3 w-10/12 rounded-full" />
                <div className="skeleton-line h-3 w-7/12 rounded-full" />
            </div>
        </div>
    )
}

interface FundLoadingIndicatorProps {
    isVisible: boolean
    fundName?: string
    detailText?: string
}

export function FundLoadingIndicator({ isVisible, fundName, detailText }: FundLoadingIndicatorProps) {
    if (!isVisible) return null

    return (
        <div
            className="fixed inset-0 z-50 flex items-center justify-center backdrop-blur-md"
            style={{ backgroundColor: 'color-mix(in srgb, var(--background) 90%, transparent)' }}
        >
            <div className="mx-4 w-full max-w-3xl rounded-[36px] border border-[var(--card-border)] p-6 glass-strong">
            <div className="mb-6 flex items-start justify-between gap-4">
                <div>
                        <div className="text-sm font-medium text-theme-muted">正在准备基金数据</div>
                        <h3 className="mt-3 text-2xl font-semibold text-theme-primary">
                            {fundName ? `正在读取 ${fundName}` : '正在读取基金数据'}
                        </h3>
                        <p className="mt-2 text-sm leading-6 text-theme-secondary">
                            {detailText || '估值、持仓和分时数据会依次出现。'}
                        </p>
                    </div>
                    <LoadingSpinner size="md" />
                </div>
                <LayoutSkeleton rows={3} dense />
            </div>
        </div>
    )
}
