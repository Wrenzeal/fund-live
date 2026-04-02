'use client'

import { type MarketStatus, useMarketStatus, getSessionLabel, formatTimeUntil } from '@/hooks/use-market-status'
import { cn } from '@/lib/utils'
import { Clock, TrendingUp, Coffee, Moon, Calendar } from 'lucide-react'

interface MarketStatusIndicatorProps {
    className?: string
    showDetails?: boolean
    status?: MarketStatus & { mounted?: boolean }
}

function MarketStatusIndicatorBody({
    className,
    showDetails,
    status,
    mounted,
}: {
    className?: string
    showDetails: boolean
    status: MarketStatus
    mounted: boolean
}) {

    const getSessionIcon = () => {
        switch (status.session) {
            case 'morning':
            case 'afternoon':
                return <TrendingUp className="w-4 h-4" />
            case 'lunch_break':
                return <Coffee className="w-4 h-4" />
            case 'pre_market':
            case 'after_hours':
                return <Moon className="w-4 h-4" />
            case 'weekend':
            case 'holiday':
                return <Calendar className="w-4 h-4" />
            default:
                return <Clock className="w-4 h-4" />
        }
    }

    // 在客户端挂载前显示加载占位符，避免 Hydration 错误
    if (!mounted) {
        return (
            <div className={cn('flex items-center gap-3', className)}>
                <div className="flex items-center gap-2">
                    <span className="inline-block w-2.5 h-2.5 rounded-full bg-gray-400 animate-pulse" />
                    <span className="text-sm font-medium text-theme-muted">加载中...</span>
                </div>
            </div>
        )
    }

    return (
        <div className={cn('flex items-center gap-3', className)}>
            {/* Status indicator dot */}
            <div className="flex items-center gap-2">
                <span
                    className={cn(
                        'inline-block w-2.5 h-2.5 rounded-full',
                        status.isTrading ? 'bg-green-500 live-indicator' : 'bg-amber-500'
                    )}
                />
                <span className={cn(
                    'text-sm font-medium',
                    status.isTrading ? 'market-open' : 'market-closed'
                )}>
                    {status.isTrading ? '交易中' : getSessionLabel(status.session)}
                </span>
            </div>

            {/* Session icon */}
            <div className="text-theme-muted">
                {getSessionIcon()}
            </div>

            {/* Details: time until next session */}
            {showDetails && !status.isTrading && status.timeUntilNextSession > 0 && (
                <div className="text-xs text-theme-muted">
                    距开盘: {formatTimeUntil(status.timeUntilNextSession)}
                </div>
            )}

            {/* Current time */}
            {showDetails && (
                <div className="text-xs text-theme-muted hidden sm:block">
                    <Clock className="w-3 h-3 inline mr-1" />
                    {status.currentTime.toLocaleTimeString('zh-CN', {
                        hour: '2-digit',
                        minute: '2-digit',
                        timeZone: 'Asia/Shanghai',
                    })}
                </div>
            )}
        </div>
    )
}

function ConnectedMarketStatusIndicator({ className, showDetails }: Omit<MarketStatusIndicatorProps, 'status'>) {
    const { mounted, ...status } = useMarketStatus()

    return (
        <MarketStatusIndicatorBody
            className={className}
            showDetails={showDetails ?? false}
            status={status}
            mounted={mounted}
        />
    )
}

export function MarketStatusIndicator({ className, showDetails = false, status }: MarketStatusIndicatorProps) {
    if (status) {
        return (
            <MarketStatusIndicatorBody
                className={className}
                showDetails={showDetails}
                status={status}
                mounted={status.mounted ?? true}
            />
        )
    }

    return <ConnectedMarketStatusIndicator className={className} showDetails={showDetails} />
}
