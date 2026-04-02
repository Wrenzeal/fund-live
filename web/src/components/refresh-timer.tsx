'use client'

import { useState, useEffect } from 'react'
import { useFundStore } from '@/store/fund-store'
import { calculateCountdown } from '@/lib/utils'
import { useMarketTradingState } from '@/hooks/use-market-status'
import { cn } from '@/lib/utils'
import { RefreshCw, Clock, AlertCircle } from 'lucide-react'

interface RefreshTimerProps {
    className?: string
}

export function RefreshTimer({ className }: RefreshTimerProps) {
    const {
        lastUpdated,
        refreshInterval,
        currentFundId,
        isLoading,
        fetchFundEstimate,
        fetchTimeSeries
    } = useFundStore()
    const { isTrading } = useMarketTradingState()

    const [countdown, setCountdown] = useState(refreshInterval)

    // Update countdown every second
    useEffect(() => {
        const timer = setInterval(() => {
            setCountdown(calculateCountdown(lastUpdated, refreshInterval))
        }, 1000)

        return () => clearInterval(timer)
    }, [lastUpdated, refreshInterval])

    // Auto-refresh when countdown reaches 0
    useEffect(() => {
        if (countdown === 0 && currentFundId && !isLoading) {
            fetchFundEstimate(currentFundId)
            fetchTimeSeries(currentFundId)
        }
    }, [countdown, currentFundId, isLoading, fetchFundEstimate, fetchTimeSeries])

    const handleManualRefresh = () => {
        if (currentFundId && !isLoading) {
            fetchFundEstimate(currentFundId)
            fetchTimeSeries(currentFundId)
        }
    }

    const progress = ((refreshInterval - countdown) / refreshInterval) * 100

    return (
        <div className={cn('flex items-center gap-4', className)}>
            {/* Trading hours indicator */}
            <div className="flex items-center gap-2 text-sm">
                {isTrading ? (
                    <>
                        <span className="inline-block w-2 h-2 rounded-full bg-green-500 animate-pulse" />
                        <span className="text-green-400">交易时段</span>
                    </>
                ) : (
                    <>
                        <AlertCircle className="w-4 h-4 text-yellow-500" />
                        <span className="text-yellow-500">非交易时间</span>
                    </>
                )}
            </div>

            {/* Countdown display */}
            <div className="flex items-center gap-2">
                <Clock className="w-4 h-4 text-white/60" />
                <div className="relative w-24 h-2 bg-white/10 rounded-full overflow-hidden">
                    <div
                        className="absolute inset-y-0 left-0 bg-gradient-to-r from-cyan-500 to-blue-500 transition-all duration-1000"
                        style={{ width: `${progress}%` }}
                    />
                </div>
                <span className="text-white/60 text-sm w-8">{countdown}s</span>
            </div>

            {/* Manual refresh button */}
            <button
                onClick={handleManualRefresh}
                disabled={isLoading || !currentFundId}
                className={cn(
                    'p-2 rounded-lg transition-all',
                    'bg-white/10 hover:bg-white/20',
                    'disabled:opacity-50 disabled:cursor-not-allowed',
                    isLoading && 'animate-spin'
                )}
            >
                <RefreshCw className="w-4 h-4 text-white/80" />
            </button>
        </div>
    )
}
