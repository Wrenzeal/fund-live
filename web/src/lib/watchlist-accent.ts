export const GROUP_ACCENT_OPTIONS = [
  { value: 'cyan', label: '清爽蓝', shell: 'border-cyan-400/35 bg-cyan-500/12 text-theme-primary', dot: 'bg-cyan-300' },
  { value: 'emerald', label: '自然绿', shell: 'border-emerald-400/35 bg-emerald-500/12 text-theme-primary', dot: 'bg-emerald-300' },
  { value: 'amber', label: '暖金橙', shell: 'border-amber-400/35 bg-amber-500/12 text-theme-primary', dot: 'bg-amber-300' },
  { value: 'fuchsia', label: '紫红调', shell: 'border-fuchsia-400/35 bg-fuchsia-500/12 text-theme-primary', dot: 'bg-fuchsia-300' },
] as const

export type WatchlistAccent = (typeof GROUP_ACCENT_OPTIONS)[number]['value']

export function watchlistAccentToClass(accent: string) {
  switch (accent) {
    case 'emerald':
      return 'from-emerald-500/30 via-teal-500/15 to-transparent'
    case 'amber':
      return 'from-amber-500/25 via-orange-500/15 to-transparent'
    case 'fuchsia':
      return 'from-fuchsia-500/25 via-violet-500/15 to-transparent'
    default:
      return 'from-cyan-500/30 via-sky-500/20 to-transparent'
  }
}

export function watchlistAccentLabel(accent: string) {
  switch (accent) {
    case 'emerald':
      return '自然绿'
    case 'amber':
      return '暖金橙'
    case 'fuchsia':
      return '紫红调'
    default:
      return '清爽蓝'
  }
}

export function watchlistAccentBadgeClass(accent: string) {
  switch (accent) {
    case 'emerald':
      return 'border-emerald-400/35 bg-emerald-500/12 text-theme-primary'
    case 'amber':
      return 'border-amber-400/35 bg-amber-500/12 text-theme-primary'
    case 'fuchsia':
      return 'border-fuchsia-400/35 bg-fuchsia-500/12 text-theme-primary'
    default:
      return 'border-cyan-400/35 bg-cyan-500/12 text-theme-primary'
  }
}
