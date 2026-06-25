'use client'

import { useState, useCallback, useEffect, useId, useRef } from 'react'
import { createPortal } from 'react-dom'
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

type SearchableFund = {
    id: string
    name: string
    type?: string
    manager?: string
    company?: string
    category_name?: string
    count?: number
}

const SAMPLE_FUNDS = [
    { id: '005827', name: '易方达蓝筹' },
    { id: '003095', name: '中欧医疗' },
    { id: '320007', name: '诺安成长' },
]

export function FundSearch({ onSelect, currentFundId, className }: FundSearchProps) {
    const [inputValue, setInputValue] = useState('')
    const [isOpen, setIsOpen] = useState(false)
    const [overlayEntered, setOverlayEntered] = useState(false)
    const searchTitleId = useId()
    const expandedInputRef = useRef<HTMLInputElement | null>(null)
    const { recentFunds, quickSelectFunds } = useSearchPreferences()

    const debouncedQuery = useDebounce(inputValue, 300)
    const { results, isLoading } = useFundSearch(debouncedQuery)

    const openSearch = useCallback(() => {
        setOverlayEntered(false)
        setIsOpen(true)
    }, [])

    const closeSearch = useCallback(() => {
        setOverlayEntered(false)
        setIsOpen(false)
    }, [])

    const handleSelect = useCallback((fundId: string) => {
        onSelect(fundId)
        setInputValue('')
        closeSearch()
    }, [closeSearch, onSelect])

    const handleTrackedSelect = useCallback((fund: SearchableFund) => {
        recordFundSelection(toSearchSnapshot(fund))
        handleSelect(fund.id)
    }, [handleSelect])

    const fallbackFunds = quickSelectFunds.length > 0 ? quickSelectFunds : SAMPLE_FUNDS

    const handleClear = () => {
        setInputValue('')
        expandedInputRef.current?.focus()
    }

    const handleClearRecent = () => {
        clearRecentSearches()
    }

    useEffect(() => {
        if (!isOpen) {
            return
        }

        const previousOverflow = document.body.style.overflow
        document.body.style.overflow = 'hidden'

        const frame = window.requestAnimationFrame(() => {
            setOverlayEntered(true)
            expandedInputRef.current?.focus({ preventScroll: true })
        })

        return () => {
            window.cancelAnimationFrame(frame)
            document.body.style.overflow = previousOverflow
        }
    }, [isOpen])

    useEffect(() => {
        if (!isOpen) {
            return
        }

        const handleKeyDown = (event: KeyboardEvent) => {
            if (event.key === 'Escape') {
                closeSearch()
            }
        }

        window.addEventListener('keydown', handleKeyDown)
        return () => window.removeEventListener('keydown', handleKeyDown)
    }, [closeSearch, isOpen])

    const showResults = results.length > 0
    const showEmpty = Boolean(inputValue && debouncedQuery && !isLoading && !showResults)
    const showDiscovery = !showResults && !showEmpty

    return (
        <div className={cn('relative', className)}>
            <div className="relative">
                <Search className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-theme-muted" />
                <input
                    type="text"
                    value={inputValue}
                    onChange={(e) => {
                        setInputValue(e.target.value)
                        openSearch()
                    }}
                    onFocus={openSearch}
                    placeholder="搜索基金代码或名称..."
                    className={cn(
                        'w-full pl-12 pr-12 py-3 rounded-xl',
                        'search-input border border-[var(--input-border)]',
                        'text-theme-primary placeholder:text-theme-muted',
                        'focus:outline-none focus:ring-2 focus:ring-[var(--input-focus)] focus:border-transparent',
                        'transition-all duration-200 hover:border-[var(--input-focus)]',
                        isOpen && 'ring-2 ring-[var(--input-focus)]/35'
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

            {isOpen && typeof document !== 'undefined' && createPortal(
                <>
                    <div
                        className={cn(
                            'fixed inset-0 z-[80] transition-opacity duration-300',
                            overlayEntered ? 'opacity-100' : 'opacity-0'
                        )}
                        style={{
                            background: 'color-mix(in srgb, var(--background) 72%, rgba(2, 8, 23, 0.62))',
                            backdropFilter: 'blur(9px)',
                            WebkitBackdropFilter: 'blur(9px)',
                        }}
                        onClick={closeSearch}
                    />

                    <section
                        role="dialog"
                        aria-modal="true"
                        aria-labelledby={searchTitleId}
                        className={cn(
                            'fixed left-3 right-3 top-4 z-[90] max-h-[calc(100dvh-2rem)] overflow-hidden rounded-[28px] border border-[var(--card-border)]',
                            'search-expanded-panel shadow-[0_32px_90px_rgba(2,8,23,0.42)] transition-all duration-300 ease-out',
                            'sm:left-1/2 sm:right-auto sm:top-20 sm:w-[min(48rem,calc(100vw-2rem))] sm:-translate-x-1/2',
                            overlayEntered ? 'translate-y-0 scale-100 opacity-100' : 'translate-y-5 scale-[0.97] opacity-0'
                        )}
                    >
                        <div className="border-b border-[var(--card-border)] px-4 py-4 sm:px-5">
                            <div className="mb-3 flex items-center justify-between gap-4">
                                <div>
                                    <h2 id={searchTitleId} className="text-base font-bold text-theme-primary sm:text-lg">
                                        搜索基金
                                    </h2>
                                    <p className="mt-1 text-xs text-theme-muted">
                                        输入代码、名称或公司，快速切换当前观察对象。
                                    </p>
                                </div>
                                <button
                                    type="button"
                                    onClick={closeSearch}
                                    className="rounded-full border border-[var(--card-border)] bg-[var(--input-bg)]/80 p-2 text-theme-muted transition-colors hover:text-theme-primary"
                                    aria-label="关闭基金搜索"
                                >
                                    <X className="h-4 w-4" />
                                </button>
                            </div>

                            <div className="relative">
                                <Search className="absolute left-4 top-1/2 h-5 w-5 -translate-y-1/2 text-theme-muted" />
                                <input
                                    ref={expandedInputRef}
                                    type="text"
                                    value={inputValue}
                                    onChange={(event) => setInputValue(event.target.value)}
                                    placeholder="例如 005827、蓝筹、易方达"
                                    className={cn(
                                        'w-full rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] py-4 pl-12 pr-12 text-base font-medium',
                                        'text-theme-primary placeholder:text-theme-muted focus:border-transparent focus:outline-none focus:ring-2 focus:ring-[var(--input-focus)]'
                                    )}
                                />
                                {inputValue && !isLoading && (
                                    <button
                                        type="button"
                                        onClick={handleClear}
                                        className="absolute right-4 top-1/2 -translate-y-1/2 text-theme-muted transition-colors hover:text-theme-primary"
                                        aria-label="清空搜索内容"
                                    >
                                        <X className="h-5 w-5" />
                                    </button>
                                )}
                                {isLoading && (
                                    <Loader2 className="absolute right-4 top-1/2 h-5 w-5 -translate-y-1/2 animate-spin text-cyan-400" />
                                )}
                            </div>
                        </div>

                        <div className="max-h-[calc(100dvh-12rem)] overflow-y-auto px-4 py-4 sm:max-h-[30rem] sm:px-5">
                            {showResults && (
                                <ul className="space-y-2">
                                    {results.map((fund) => (
                                        <li key={fund.id}>
                                            <FundSearchOption
                                                fund={fund}
                                                currentFundId={currentFundId}
                                                onSelect={handleTrackedSelect}
                                            />
                                        </li>
                                    ))}
                                </ul>
                            )}

                            {showEmpty && (
                                <div className="rounded-3xl border border-dashed border-[var(--card-border)] bg-[var(--input-bg)]/45 px-4 py-8 text-center">
                                    <div className="text-sm font-semibold text-theme-primary">没有找到匹配基金</div>
                                    <p className="mt-2 text-xs leading-5 text-theme-muted">
                                        可以换成 6 位基金代码、基金简称或基金公司再试。
                                    </p>
                                </div>
                            )}

                            {showDiscovery && (
                                <SearchDiscovery
                                    recentFunds={recentFunds}
                                    fallbackFunds={fallbackFunds}
                                    currentFundId={currentFundId}
                                    onSelect={handleTrackedSelect}
                                    onClearRecent={handleClearRecent}
                                />
                            )}
                        </div>
                    </section>
                </>,
                document.body
            )}
        </div>
    )
}

function FundSearchOption({
    fund,
    currentFundId,
    onSelect,
}: {
    fund: SearchableFund
    currentFundId?: string
    onSelect: (fund: SearchableFund) => void
}) {
    const meta = [fund.id, fund.manager, fund.company].filter(Boolean).join(' · ')

    return (
        <button
            type="button"
            onClick={() => onSelect(fund)}
            className={cn(
                'group w-full rounded-2xl border px-4 py-3 text-left transition-all duration-200',
                'border-[var(--card-border)] bg-[var(--input-bg)]/62 hover:-translate-y-0.5 hover:border-[var(--accent-primary)]',
                'hover:shadow-[0_18px_36px_rgba(34,211,238,0.12)]',
                currentFundId === fund.id && 'border-cyan-500 bg-cyan-500/10'
            )}
        >
            <div className="flex items-start justify-between gap-4">
                <div className="min-w-0">
                    <div className="truncate text-sm font-semibold text-theme-primary sm:text-base">
                        {fund.name}
                    </div>
                    <div className="mt-1 text-xs text-theme-muted">
                        {meta}
                    </div>
                </div>
                <div className="shrink-0 text-right">
                    {fund.type && (
                        <div className="rounded-full border border-cyan-500/22 bg-cyan-500/10 px-2.5 py-1 text-[11px] font-semibold text-theme-primary">
                            {fund.type}
                        </div>
                    )}
                    {fund.category_name && (
                        <div className="mt-1 text-[11px] text-theme-muted">
                            {fund.category_name}
                        </div>
                    )}
                </div>
            </div>
        </button>
    )
}

function SearchDiscovery({
    recentFunds,
    fallbackFunds,
    currentFundId,
    onSelect,
    onClearRecent,
}: {
    recentFunds: SearchableFund[]
    fallbackFunds: SearchableFund[]
    currentFundId?: string
    onSelect: (fund: SearchableFund) => void
    onClearRecent: () => void
}) {
    return (
        <div className="space-y-5">
            {recentFunds.length > 0 && (
                <div>
                    <div className="mb-3 flex items-center justify-between gap-3">
                        <div>
                            <div className="text-sm font-semibold text-theme-primary">历史搜索</div>
                            <div className="mt-1 text-xs text-theme-muted">最近切换过的基金</div>
                        </div>
                        <button
                            type="button"
                            onClick={onClearRecent}
                            className="inline-flex items-center gap-1 rounded-full border border-[var(--card-border)] bg-[var(--input-bg)]/60 px-3 py-1.5 text-xs text-theme-muted transition-colors hover:text-theme-primary"
                        >
                            <RotateCcw className="h-3.5 w-3.5" />
                            清空
                        </button>
                    </div>
                    <div className="grid gap-2 sm:grid-cols-3">
                        {recentFunds.map((fund) => (
                            <CompactFundButton
                                key={fund.id}
                                fund={fund}
                                currentFundId={currentFundId}
                                onSelect={onSelect}
                            />
                        ))}
                    </div>
                </div>
            )}

            <div>
                <div className="mb-3">
                    <div className="text-sm font-semibold text-theme-primary">快速选择</div>
                    <div className="mt-1 text-xs text-theme-muted">常用基金会自动排在前面</div>
                </div>
                <div className="grid gap-2 sm:grid-cols-3">
                    {fallbackFunds.map((fund) => (
                        <CompactFundButton
                            key={fund.id}
                            fund={fund}
                            currentFundId={currentFundId}
                            onSelect={onSelect}
                        />
                    ))}
                </div>
            </div>
        </div>
    )
}

function CompactFundButton({
    fund,
    currentFundId,
    onSelect,
}: {
    fund: SearchableFund
    currentFundId?: string
    onSelect: (fund: SearchableFund) => void
}) {
    return (
        <button
            type="button"
            onClick={() => onSelect(fund)}
            className={cn(
                'rounded-2xl border border-[var(--card-border)] bg-[var(--input-bg)]/55 px-3 py-3 text-left transition-all duration-200',
                'hover:-translate-y-0.5 hover:border-[var(--accent-primary)] hover:text-theme-primary',
                currentFundId === fund.id && 'border-cyan-500 bg-cyan-500/10'
            )}
        >
            <div className="truncate text-sm font-semibold text-theme-primary">{fund.name}</div>
            <div className="mt-1 flex items-center justify-between gap-2 text-xs text-theme-muted">
                <span className="font-mono">{fund.id}</span>
                {typeof fund.count === 'number' && fund.count > 0 && (
                    <span className="rounded-full border border-[var(--card-border)] px-2 py-0.5">
                        {fund.count} 次
                    </span>
                )}
            </div>
        </button>
    )
}
