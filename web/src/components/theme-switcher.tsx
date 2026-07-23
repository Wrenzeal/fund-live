'use client'

import { useState } from 'react'
import { Check, ChevronDown, Eye, Moon, Palette, Settings2, Sparkles, Sun } from 'lucide-react'
import type { ThemeType, ViewMode } from '@/hooks/use-ui-preferences'
import { cn } from '@/lib/utils'

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
    name: '暖白',
    icon: <Sun className="h-4 w-4" />,
    description: '暖白纸面，低饱和涨跌色',
  },
  {
    id: 'dark',
    name: '深色',
    icon: <Moon className="h-4 w-4" />,
    description: '深色背景，适合长时间查看',
  },
  {
    id: 'cyber',
    name: '深色高对比',
    icon: <Sparkles className="h-4 w-4" />,
    description: '高对比边界，减少背景干扰',
  },
]

const viewModes: { id: ViewMode; name: string; description: string }[] = [
  { id: 'minimal', name: '简洁模式', description: '突出核心数字' },
  { id: 'professional', name: '专业模式', description: '显示分时图和持仓' },
]

const desktopToggleClass = cn(
  'group relative flex items-center gap-2 overflow-hidden rounded-lg px-3 py-2',
  'glass switcher-toggle text-sm text-theme-secondary transition-all duration-200',
  'hover:bg-[var(--input-bg)] hover:text-theme-primary',
)

const desktopOptionClass = cn(
  'flex w-full items-center justify-between px-4 py-3 text-left transition-colors',
  'hover:bg-[var(--input-bg)]',
)

export function ThemeSwitcher({
  themeType,
  setThemeType,
  viewMode,
  setViewMode,
  hideViewMode,
  className,
}: ThemeSwitcherProps) {
  const [isMobileOpen, setIsMobileOpen] = useState(false)
  const [isThemeOpen, setIsThemeOpen] = useState(false)
  const [isViewOpen, setIsViewOpen] = useState(false)

  const currentTheme = themes.find((theme) => theme.id === themeType)
  const currentView = viewModes.find((mode) => mode.id === viewMode)

  return (
    <div className={cn('relative flex items-center', className)}>
      <div className="md:hidden">
        <button
          type="button"
          onClick={() => setIsMobileOpen((open) => !open)}
          className="switcher-toggle glass inline-flex h-11 w-11 items-center justify-center rounded-xl text-theme-secondary transition-colors hover:bg-[var(--input-bg)] hover:text-theme-primary"
          aria-label="界面设置"
          aria-expanded={isMobileOpen}
          title="界面设置"
        >
          <Settings2 className="h-5 w-5" />
        </button>

        {isMobileOpen && (
          <>
            <div className="fixed inset-0 z-40" onClick={() => setIsMobileOpen(false)} />
            <div
              role="dialog"
              aria-modal="true"
              aria-label="界面设置"
              className="switcher-dropdown-panel glass absolute right-0 top-full z-50 mt-2 w-[min(20rem,calc(100vw-1.5rem))] overflow-hidden rounded-2xl border border-[var(--card-border)] shadow-2xl"
            >
              <div className="border-b border-[var(--card-border)] px-4 py-3">
                <div className="flex items-center gap-2 text-sm font-semibold text-theme-primary">
                  <Palette className="h-4 w-4 text-[var(--accent-primary)]" />
                  主题
                </div>
              </div>
              <div className="grid grid-cols-3 gap-2 p-3">
                {themes.map((theme) => (
                  <button
                    key={theme.id}
                    type="button"
                    onClick={() => setThemeType(theme.id)}
                    className={cn(
                      'flex min-h-11 items-center justify-center gap-1.5 rounded-xl border px-2 text-xs font-medium transition-colors',
                      themeType === theme.id
                        ? 'border-cyan-400/35 bg-cyan-500/15 text-theme-primary'
                        : 'border-[var(--card-border)] bg-[var(--input-bg)]/60 text-theme-secondary',
                    )}
                  >
                    {theme.icon}
                    <span className="truncate">{theme.name}</span>
                  </button>
                ))}
              </div>

              {!hideViewMode && (
                <>
                  <div className="border-y border-[var(--card-border)] px-4 py-3">
                    <div className="flex items-center gap-2 text-sm font-semibold text-theme-primary">
                      <Eye className="h-4 w-4 text-[var(--accent-primary)]" />
                      视图模式
                    </div>
                  </div>
                  <div className="grid grid-cols-2 gap-2 p-3">
                    {viewModes.map((mode) => (
                      <button
                        key={mode.id}
                        type="button"
                        onClick={() => setViewMode(mode.id)}
                        className={cn(
                          'min-h-11 rounded-xl border px-3 text-sm font-medium transition-colors',
                          viewMode === mode.id
                            ? 'border-cyan-400/35 bg-cyan-500/15 text-theme-primary'
                            : 'border-[var(--card-border)] bg-[var(--input-bg)]/60 text-theme-secondary',
                        )}
                      >
                        {mode.name}
                      </button>
                    ))}
                  </div>
                </>
              )}
            </div>
          </>
        )}
      </div>

      <div className="hidden items-center gap-2 md:flex">
        {!hideViewMode && (
          <div className="relative">
            <button
              type="button"
              onClick={() => setIsViewOpen((open) => !open)}
              className={desktopToggleClass}
              aria-expanded={isViewOpen}
            >
              <Eye className="h-4 w-4" />
              <span className="hidden sm:inline">{currentView?.name}</span>
              <ChevronDown className={cn('h-4 w-4 transition-transform', isViewOpen && 'rotate-180')} />
            </button>

            {isViewOpen && (
              <>
                <div className="fixed inset-0 z-40" onClick={() => setIsViewOpen(false)} />
                <div className="switcher-dropdown-panel glass absolute right-0 top-full z-50 mt-2 w-48 overflow-hidden rounded-xl shadow-2xl">
                  {viewModes.map((mode) => (
                    <button
                      key={mode.id}
                      type="button"
                      onClick={() => {
                        setViewMode(mode.id)
                        setIsViewOpen(false)
                      }}
                      className={cn(desktopOptionClass, viewMode === mode.id && 'bg-cyan-500/20')}
                    >
                      <div>
                        <div className="text-sm font-medium text-theme-primary">{mode.name}</div>
                        <div className="text-xs text-theme-muted">{mode.description}</div>
                      </div>
                      {viewMode === mode.id && <Check className="h-4 w-4 text-cyan-400" />}
                    </button>
                  ))}
                </div>
              </>
            )}
          </div>
        )}

        <div className="relative">
          <button
            type="button"
            onClick={() => setIsThemeOpen((open) => !open)}
            className={desktopToggleClass}
            aria-expanded={isThemeOpen}
          >
            <Palette className="h-4 w-4" />
            <span className="hidden sm:inline">{currentTheme?.name}</span>
            <ChevronDown className={cn('h-4 w-4 transition-transform', isThemeOpen && 'rotate-180')} />
          </button>

          {isThemeOpen && (
            <>
              <div className="fixed inset-0 z-40" onClick={() => setIsThemeOpen(false)} />
              <div className="switcher-dropdown-panel glass absolute right-0 top-full z-50 mt-2 w-56 overflow-hidden rounded-xl shadow-2xl">
                {themes.map((theme) => (
                  <button
                    key={theme.id}
                    type="button"
                    onClick={() => {
                      setThemeType(theme.id)
                      setIsThemeOpen(false)
                    }}
                    className={cn(desktopOptionClass, themeType === theme.id && 'bg-cyan-500/20')}
                  >
                    <div className="flex items-center gap-3">
                      <div className={cn(
                        'rounded-lg p-2',
                        theme.id === 'classic' && 'border border-stone-300 bg-[#fff8ea] text-stone-800',
                        theme.id === 'dark' && 'bg-slate-800 text-white',
                        theme.id === 'cyber' && 'bg-slate-700 text-slate-100',
                      )}>
                        {theme.icon}
                      </div>
                      <div>
                        <div className="text-sm font-medium text-theme-primary">{theme.name}</div>
                        <div className="text-xs text-theme-muted">{theme.description}</div>
                      </div>
                    </div>
                    {themeType === theme.id && <Check className="h-4 w-4 text-cyan-400" />}
                  </button>
                ))}
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  )
}
