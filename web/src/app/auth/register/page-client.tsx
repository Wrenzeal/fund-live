'use client'

import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useEffect, useState } from 'react'
import { ArrowRight, Loader2, LockKeyhole, Mail, UserRound } from 'lucide-react'
import { AuthShell } from '@/components/auth-shell'
import { registerWithPassword, useCurrentUser } from '@/hooks/use-auth'
import { authRouteWithReturnTo } from '@/lib/auth-return-to'

export function RegisterPageClient({ returnTo }: { returnTo: string }) {
  const router = useRouter()
  const { user, mutate } = useCurrentUser()
  const [email, setEmail] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)

  useEffect(() => {
    if (user) {
      router.replace(returnTo)
    }
  }, [returnTo, router, user])

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setError(null)
    if ([...password].length < 10 || !/\p{L}/u.test(password) || !/\p{N}/u.test(password) || /\s/u.test(password)) {
      setError('密码至少 10 位，并且需要同时包含字母和数字，不能包含空格')
      return
    }
    setIsSubmitting(true)

    void (async () => {
      try {
        await registerWithPassword({ email, display_name: displayName, password })
        void mutate()
        router.replace(returnTo)
      } catch (err) {
        setError(err instanceof Error ? err.message : '注册失败')
        setIsSubmitting(false)
      }
    })()
  }

  return (
    <AuthShell
      eyebrow="创建账户"
      title="注册账户"
      description="创建账户后即可保存自选基金、持仓记录和个人设置。"
      footer={(
        <div className="flex items-center justify-between gap-4">
          <span>已经有账号？</span>
          <Link href={authRouteWithReturnTo('/auth/login', returnTo)} className="font-medium text-cyan-400 transition-colors hover:text-cyan-300">
            去登录
          </Link>
        </div>
      )}
    >
      <form onSubmit={handleSubmit} className="space-y-5">
        <label className="block space-y-2">
          <span className="text-sm text-theme-secondary">显示名称</span>
          <div className="auth-input-shell flex items-center gap-3 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3">
            <UserRound className="h-4 w-4 text-theme-muted" />
            <input
              type="text"
              autoComplete="nickname"
              value={displayName}
              onChange={(event) => setDisplayName(event.target.value)}
              className="auth-input w-full bg-transparent text-theme-primary outline-none placeholder:text-theme-muted"
              placeholder="给这个账户起个名字"
            />
          </div>
        </label>

        <label className="block space-y-2">
          <span className="text-sm text-theme-secondary">邮箱</span>
          <div className="auth-input-shell flex items-center gap-3 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3">
            <Mail className="h-4 w-4 text-theme-muted" />
            <input
              type="email"
              autoComplete="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              className="auth-input w-full bg-transparent text-theme-primary outline-none placeholder:text-theme-muted"
              placeholder="name@example.com"
              required
            />
          </div>
        </label>

        <label className="block space-y-2">
          <span className="text-sm text-theme-secondary">密码</span>
          <div className="auth-input-shell flex items-center gap-3 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3">
            <LockKeyhole className="h-4 w-4 text-theme-muted" />
            <input
              type="password"
              autoComplete="new-password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              className="auth-input w-full bg-transparent text-theme-primary outline-none placeholder:text-theme-muted"
              placeholder="至少 10 位，包含字母和数字"
              required
            />
          </div>
          <span className="text-xs text-theme-muted">用于保护持仓与自选数据，建议不要复用其他网站密码。</span>
        </label>

        {error && (
          <div className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
            {error}
          </div>
        )}

        <button
          type="submit"
          disabled={isSubmitting}
          className="inline-flex w-full items-center justify-center gap-2 rounded-2xl bg-gradient-to-r from-cyan-500 via-sky-500 to-blue-600 px-5 py-3.5 text-sm font-semibold text-white transition-transform hover:-translate-y-0.5 disabled:cursor-not-allowed disabled:opacity-60"
        >
          {isSubmitting ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin" />
              注册中...
            </>
          ) : (
            <>
              注册并进入首页
              <ArrowRight className="h-4 w-4" />
            </>
          )}
        </button>
      </form>
    </AuthShell>
  )
}
