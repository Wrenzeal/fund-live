'use client'

import { cn } from '@/lib/utils'

interface LoadingSpinnerProps {
    size?: 'sm' | 'md' | 'lg'
    className?: string
    text?: string
}

export function LoadingSpinner({ size = 'md', className, text }: LoadingSpinnerProps) {
    const sizeClasses = {
        sm: 'w-4 h-4',
        md: 'w-8 h-8',
        lg: 'w-12 h-12'
    }

    return (
        <div className={cn('flex flex-col items-center justify-center gap-3', className)}>
            <div className={cn(
                'relative',
                sizeClasses[size]
            )}>
                {/* Outer ring */}
                <div className={cn(
                    'absolute inset-0 rounded-full border-2 border-[var(--card-border)]',
                    sizeClasses[size]
                )} />
                {/* Spinning gradient */}
                <div className={cn(
                    'absolute inset-0 rounded-full border-2 border-transparent animate-spin',
                    'border-t-cyan-500 border-r-blue-500',
                    sizeClasses[size]
                )} />
                {/* Inner glow */}
                <div className={cn(
                    'absolute inset-1 rounded-full bg-gradient-to-br from-cyan-500/20 to-blue-500/20 animate-pulse',
                )} />
            </div>
            {text && (
                <span className="text-sm text-theme-secondary animate-pulse">{text}</span>
            )}
        </div>
    )
}

interface LoadingOverlayProps {
    isLoading: boolean
    text?: string
    children: React.ReactNode
}

export function LoadingOverlay({ isLoading, text = '加载中...', children }: LoadingOverlayProps) {
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
            <div className="glass rounded-3xl p-8 max-w-md w-full mx-4 text-center">
                {/* Animated logo */}
                <div className="relative w-20 h-20 mx-auto mb-6">
                    {/* Rotating outer rings */}
                    <div className="absolute inset-0 rounded-full border-4 border-cyan-500/30 animate-ping" />
                    <div className="absolute inset-2 rounded-full border-4 border-blue-500/30 animate-ping animation-delay-200" />

                    {/* Main spinner */}
                    <div className="absolute inset-0 rounded-full border-4 border-transparent border-t-cyan-500 border-r-blue-500 animate-spin" />

                    {/* Center icon */}
                    <div className="absolute inset-4 rounded-full bg-gradient-to-br from-cyan-500 to-blue-600 flex items-center justify-center">
                        <svg
                            className="w-6 h-6 text-white"
                            fill="none"
                            viewBox="0 0 24 24"
                            stroke="currentColor"
                        >
                            <path
                                strokeLinecap="round"
                                strokeLinejoin="round"
                                strokeWidth={2}
                                d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6"
                            />
                        </svg>
                    </div>
                </div>

                {/* Loading text */}
                <h3 className="text-xl font-semibold text-theme-primary mb-2">
                    {fundName ? `正在加载 ${fundName}` : '正在加载基金数据'}
                </h3>
                <p className="text-sm text-theme-secondary mb-4">
                    {detailText || '正在获取实时估值和持仓数据...'}
                </p>

                {/* Progress dots */}
                <div className="flex items-center justify-center gap-2">
                    <div className="w-2 h-2 rounded-full bg-cyan-500 animate-bounce" style={{ animationDelay: '0ms' }} />
                    <div className="w-2 h-2 rounded-full bg-cyan-400 animate-bounce" style={{ animationDelay: '150ms' }} />
                    <div className="w-2 h-2 rounded-full bg-blue-500 animate-bounce" style={{ animationDelay: '300ms' }} />
                </div>
            </div>
        </div>
    )
}
