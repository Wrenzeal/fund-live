import Link from 'next/link'

import { BrandMark } from '@/components/brand-mark'
import { cn } from '@/lib/utils'

interface SiteFooterProps {
  className?: string
  compact?: boolean
}

export function SiteFooter({ className, compact = false }: SiteFooterProps) {
  return (
    <footer className={cn('border-t border-[var(--card-border)]', className)}>
      <div className={cn('container mx-auto px-4', compact ? 'py-5' : 'py-7')}>
        <div className="flex flex-col gap-5 md:flex-row md:items-center md:justify-between">
          <BrandMark subtitle="数据观察工具" className="shrink-0" />

          <div className="flex flex-col gap-3 text-sm text-theme-muted md:items-end">
            <nav className="flex flex-wrap gap-x-4 gap-y-2" aria-label="页脚导航">
              <Link className="transition-colors hover:text-theme-primary focus-visible:text-theme-primary" href="/privacy">
                隐私政策
              </Link>
              <Link className="transition-colors hover:text-theme-primary focus-visible:text-theme-primary" href="/terms">
                服务条款
              </Link>
              <Link className="transition-colors hover:text-theme-primary focus-visible:text-theme-primary" href="/announcements">
                更新公告
              </Link>
              <Link className="transition-colors hover:text-theme-primary focus-visible:text-theme-primary" href="/auth/login">
                账户登录
              </Link>
            </nav>
            <p className="max-w-2xl text-xs leading-5 text-theme-muted md:text-right">
              © 2024 - 2026 涨了多少 · FundLive。页面数据仅用于整理与观察，不构成投资建议或收益承诺。
            </p>
          </div>
        </div>
      </div>
    </footer>
  )
}
