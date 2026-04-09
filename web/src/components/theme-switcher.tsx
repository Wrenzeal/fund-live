'use client'

import { useState } from 'react'
import { cn } from '@/lib/utils'
import { Palette, Eye, ChevronDown, Check, Moon, Sun, Sparkles } from 'lucide-react'
import type { ThemeType, ViewMode } from '@/hooks/use-ui-preferences'

interface ThemeSwitcherProps {
    themeType: ThemeType
    setThemeType: (theme: ThemeType) => void
    viewMode: ViewMode
    setViewMode: (mode: ViewMode) => void
    hideViewMode?: boolean
    className?: string
}

const themes: { id: ThemeType; name: string; icon: React.ReactNode; description: string }[] = [
    {
        id: 'classic',
        name: 'Classic',
        icon: <Sun className="w-4 h-4" />,
        description: '经典白色背景，红涨绿跌'
    },
    {
        id: 'dark',
        name: 'Dark',
        icon: <Moon className="w-4 h-4" />,
        description: '深色模式，护眼首选'
    },
    {
        id: 'cyber',
        name: 'Cyber',
        icon: <Sparkles className="w-4 h-4" />,
        description: '赛博朋克霓虹风格'
    },
]

const viewModes: { id: ViewMode; name: string; description: string }[] = [
    { id: 'minimal', name: '极简模式', description: '只显示核心数字' },
    { id: 'professional', name: '专业模式', description: '完整分时图和持仓' },
]

export function ThemeSwitcher({
    themeType,
    setThemeType,
    viewMode,
    setViewMode,
    hideViewMode,
    className
}: ThemeSwitcherProps) {
    const [isThemeOpen, setIsThemeOpen] = useState(false)
    const [isViewOpen, setIsViewOpen] = useState(false)

    const currentTheme = themes.find(t => t.id === themeType)
    const currentView = viewModes.find(v => v.id === viewMode)

    return (
        <div className={cn('flex items-center gap-2', className)}>
            {!hideViewMode && (
                <div className="relative">
                    <button
                        onClick={() => setIsViewOpen(!isViewOpen)}
                        className={cn(
                            'group relative flex items-center gap-2 overflow-hidden px-3 py-2 rounded-lg',
                            'glass switcher-toggle text-theme-secondary text-sm transition-all duration-200',
                            'hover:-translate-y-0.5 hover:bg-[var(--input-bg)] hover:text-theme-primary hover:shadow-[0_12px_24px_rgba(34,211,238,0.10)]',
                            'active:scale-[0.97]'
                        )}
                    >
                        <span className="action-button-shine" />
                        <Eye className="relative z-10 w-4 h-4 transition-transform duration-300 group-hover:scale-110" />
                        <span className="relative z-10 hidden sm:inline">{currentView?.name}</span>
                        <ChevronDown className={cn('relative z-10 w-4 h-4 transition-transform', isViewOpen && 'rotate-180')} />
                    </button>

                    {isViewOpen && (
                        <>
                            <div className="fixed inset-0 z-40" onClick={() => setIsViewOpen(false)} />
                            <div className="absolute right-0 top-full mt-2 z-50 w-48">
                                <div className="glass switcher-dropdown-panel rounded-xl shadow-2xl overflow-hidden">
                                    {viewModes.map((mode) => (
                                        <button
                                            key={mode.id}
                                            onClick={() => {
                                                setViewMode(mode.id)
                                                setIsViewOpen(false)
                                            }}
                                            className={cn(
                                                'w-full px-4 py-3 text-left transition-colors',
                                                'hover:bg-[var(--input-bg)]',
                                                'flex items-center justify-between',
                                                viewMode === mode.id && 'bg-cyan-500/20'
                                            )}
                                        >
                                            <div>
                                                <div className="font-medium text-theme-primary text-sm">{mode.name}</div>
                                                <div className="text-xs text-theme-muted">{mode.description}</div>
                                            </div>
                                            {viewMode === mode.id && <Check className="w-4 h-4 text-cyan-400" />}
                                        </button>
                                    ))}
                                </div>
                            </div>
                        </>
                    )}
                </div>
            )}

            {/* Theme Switcher */}
            <div className="relative">
                <button
                    onClick={() => setIsThemeOpen(!isThemeOpen)}
                    className={cn(
                        'group relative flex items-center gap-2 overflow-hidden px-3 py-2 rounded-lg',
                        'glass switcher-toggle text-theme-secondary text-sm transition-all duration-200',
                        'hover:-translate-y-0.5 hover:bg-[var(--input-bg)] hover:text-theme-primary hover:shadow-[0_12px_24px_rgba(34,211,238,0.10)]',
                        'active:scale-[0.97]'
                    )}
                >
                    <span className="action-button-shine" />
                    <Palette className="relative z-10 w-4 h-4 transition-transform duration-300 group-hover:scale-110 group-hover:-rotate-6" />
                    <span className="relative z-10 hidden sm:inline">{currentTheme?.name}</span>
                    <ChevronDown className={cn('relative z-10 w-4 h-4 transition-transform', isThemeOpen && 'rotate-180')} />
                </button>

                {isThemeOpen && (
                    <>
                        <div className="fixed inset-0 z-40" onClick={() => setIsThemeOpen(false)} />
                        <div className="absolute right-0 top-full mt-2 z-50 w-56">
                            <div className="glass switcher-dropdown-panel rounded-xl shadow-2xl overflow-hidden">
                                {themes.map((theme) => (
                                    <button
                                        key={theme.id}
                                        onClick={() => {
                                            setThemeType(theme.id)
                                            setIsThemeOpen(false)
                                        }}
                                        className={cn(
                                            'w-full px-4 py-3 text-left transition-colors',
                                            'hover:bg-[var(--input-bg)]',
                                            'flex items-center justify-between',
                                            themeType === theme.id && 'bg-cyan-500/20'
                                        )}
                                    >
                                        <div className="flex items-center gap-3">
                                            <div className={cn(
                                                'p-2 rounded-lg',
                                                theme.id === 'classic' && 'border border-slate-200 bg-slate-50 text-slate-900',
                                                theme.id === 'dark' && 'bg-slate-800 text-white',
                                                theme.id === 'cyber' && 'bg-gradient-to-r from-cyan-500 to-purple-500 text-white'
                                            )}>
                                                {theme.icon}
                                            </div>
                                            <div>
                                                <div className="font-medium text-theme-primary text-sm">{theme.name}</div>
                                                <div className="text-xs text-theme-muted">{theme.description}</div>
                                            </div>
                                        </div>
                                        {themeType === theme.id && <Check className="w-4 h-4 text-cyan-400" />}
                                    </button>
                                ))}
                            </div>
                        </div>
                    </>
                )}
            </div>
        </div>
    )
}
