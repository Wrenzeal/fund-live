'use client'

import { useState, useCallback } from 'react'
import { useFundSearch } from '@/hooks/use-fund-data'
import { clearRecentSearches, recordFundSelection, toSearchSnapshot, useSearchPreferences } from '@/hooks/use-search-preferences'
import { cn } from '@/lib/utils'
import { Search, X, Loader2, RotateCcw } from 'lucide-react'
import { useDebounce } from '@/hooks/use-debounce'

interface FundSearchProps {
    onSelect: (fundId: string) => void
    currentFundId?: string
    className?: string
}

const SAMPLE_FUNDS = [
    { id: '005827', name: '易方达蓝筹' },
    { id: '003095', name: '中欧医疗' },
    { id: '320007', name: '诺安成长' },
]

export function FundSearch({ onSelect, currentFundId, className }: FundSearchProps) {
    const [inputValue, setInputValue] = useState('')
    const [isOpen, setIsOpen] = useState(false)
    const { recentFunds, quickSelectFunds } = useSearchPreferences()

    const debouncedQuery = useDebounce(inputValue, 300)
    const { results, isLoading } = useFundSearch(debouncedQuery)

    const handleSelect = useCallback((fundId: string) => {
        onSelect(fundId)
        setInputValue('')
        setIsOpen(false)
    }, [onSelect])

    const handleTrackedSelect = useCallback((fund: { id: string; name: string; type?: string; manager?: string; company?: string }) => {
        recordFundSelection(toSearchSnapshot(fund))
        handleSelect(fund.id)
    }, [handleSelect])

    const fallbackFunds = quickSelectFunds.length > 0 ? quickSelectFunds : SAMPLE_FUNDS

    const handleClear = () => {
        setInputValue('')
        setIsOpen(false)
    }

    const handleClearRecent = () => {
        clearRecentSearches()
    }

    return (
        <div className={cn('relative', className)}>
            <div className="relative">
                <Search className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-theme-muted" />
                <input
                    type="text"
                    value={inputValue}
                    onChange={(e) => {
                        setInputValue(e.target.value)
                        setIsOpen(true)
                    }}
                    onFocus={() => setIsOpen(true)}
                    placeholder="搜索基金代码或名称..."
                    className={cn(
                        'w-full pl-12 pr-12 py-3 rounded-xl',
                        'search-input border border-[var(--input-border)]',
                        'text-theme-primary placeholder:text-theme-muted',
                        'focus:outline-none focus:ring-2 focus:ring-[var(--input-focus)] focus:border-transparent',
                        'transition-all duration-200'
                    )}
                />
                {inputValue && !isLoading && (
                    <button
                        onClick={handleClear}
                        className="absolute right-4 top-1/2 -translate-y-1/2 text-theme-muted hover:text-theme-primary transition-colors"
                    >
                        <X className="w-5 h-5" />
                    </button>
                )}
                {isLoading && (
                    <Loader2 className="absolute right-4 top-1/2 -translate-y-1/2 w-5 h-5 text-cyan-400 animate-spin" />
                )}
            </div>

            {isOpen && (inputValue || results.length > 0 || !inputValue) && (
                <div className="absolute top-full left-0 right-0 mt-2 z-50">
                    <div className="glass search-dropdown-panel rounded-xl shadow-2xl overflow-hidden">
                        {results.length > 0 ? (
                            <ul className="max-h-64 overflow-y-auto">
                                {results.map((fund) => (
                                    <li key={fund.id}>
                                        <button
                                            onClick={() => handleTrackedSelect(fund)}
                                            className={cn(
                                                'w-full px-4 py-3 text-left transition-colors',
                                                'hover:bg-[var(--input-bg)]',
                                                'flex items-center justify-between',
                                                currentFundId === fund.id && 'bg-cyan-500/20'
                                            )}
                                        >
                                            <div>
                                                <div className="font-medium text-theme-primary">{fund.name}</div>
                                                <div className="text-xs text-theme-muted mt-0.5">
                                                    {fund.id} · {fund.manager} · {fund.company}
                                                </div>
                                            </div>
                                            <div className="text-xs text-cyan-400 font-mono">
                                                {fund.type}
                                            </div>
                                        </button>
                                    </li>
                                ))}
                            </ul>
                        ) : inputValue && debouncedQuery && !isLoading ? (
                            <div className="px-4 py-6 text-center text-theme-muted">
                                未找到匹配的基金
                            </div>
                        ) : (
                            <div className="px-4 py-4 space-y-4">
                                {recentFunds.length > 0 && (
                                    <div>
                                        <div className="mb-3 flex items-center justify-between gap-3">
                                            <div className="text-sm text-theme-muted">历史搜索</div>
                                            <button
                                                type="button"
                                                onClick={handleClearRecent}
                                                className="inline-flex items-center gap-1 text-xs text-theme-muted transition-colors hover:text-theme-primary"
                                            >
                                                <RotateCcw className="h-3.5 w-3.5" />
                                                清空
                                            </button>
                                        </div>
                                        <div className="flex flex-wrap gap-2">
                                            {recentFunds.map((fund) => (
                                                <button
                                                    key={fund.id}
                                                    onClick={() => handleTrackedSelect(fund)}
                                                    className={cn(
                                                        'px-3 py-1.5 rounded-lg text-sm transition-colors',
                                                        'bg-[var(--input-bg)] border border-[var(--input-border)]',
                                                        'text-theme-secondary hover:text-theme-primary hover:border-[var(--accent-primary)]',
                                                        currentFundId === fund.id && 'border-cyan-500 text-cyan-400'
                                                    )}
                                                >
                                                    {fund.name}
                                                </button>
                                            ))}
                                        </div>
                                    </div>
                                )}

                                <div>
                                    <div className="text-sm text-theme-muted mb-3">快速选择</div>
                                    <div className="space-y-2">
                                        {fallbackFunds.map((fund) => {
                                            const count = typeof (fund as { count?: number }).count === 'number'
                                                ? (fund as { count?: number }).count ?? 0
                                                : 0
                                            return (
                                                <button
                                                    key={fund.id}
                                                    onClick={() => handleTrackedSelect(fund)}
                                                    className={cn(
                                                        'w-full rounded-xl border px-3 py-2 text-left transition-colors',
                                                        'bg-[var(--input-bg)] border-[var(--input-border)] hover:border-[var(--accent-primary)]',
                                                        currentFundId === fund.id && 'border-cyan-500 bg-cyan-500/10'
                                                    )}
                                                >
                                                    <div className="flex items-center justify-between gap-3">
                                                        <div className="min-w-0">
                                                            <div className="truncate text-sm font-medium text-theme-primary">{fund.name}</div>
                                                            <div className="mt-0.5 text-xs text-theme-muted">{fund.id}</div>
                                                        </div>
                                                        {count > 0 && (
                                                            <span className="shrink-0 rounded-full border border-[var(--input-border)] px-2 py-0.5 text-xs text-theme-secondary">
                                                                搜索 {count} 次
                                                            </span>
                                                        )}
                                                    </div>
                                                </button>
                                            )
                                        })}
                                    </div>
                                </div>
                            </div>
                        )}
                    </div>
                </div>
            )}

            {isOpen && (
                <div
                    className="fixed inset-0 z-40"
                    onClick={() => setIsOpen(false)}
                />
            )}
        </div>
    )
}
