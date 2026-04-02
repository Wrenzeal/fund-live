'use client'

import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useEffect, useState } from 'react'
import { ArrowRight, Loader2, LockKeyhole, Mail, UserRound } from 'lucide-react'
import { AuthShell } from '@/components/auth-shell'
import { registerWithPassword, useCurrentUser } from '@/hooks/use-auth'

export default function RegisterPage() {
  const router = useRouter()
  const { user, mutate } = useCurrentUser()
  const [email, setEmail] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)

  useEffect(() => {
    if (user) {
      router.replace('/')
    }
  }, [router, user])

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setError(null)
    setIsSubmitting(true)

    void (async () => {
      try {
        await registerWithPassword({ email, display_name: displayName, password })
        void mutate()
        router.replace('/')
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
      description="账户创建后会立即建立登录态，方便后续接入自选基金、持仓修正和 Google 自动注册。"
      footer={(
        <div className="flex items-center justify-between gap-4">
          <span>已经有账号？</span>
          <Link href="/auth/login" className="font-medium text-cyan-400 transition-colors hover:text-cyan-300">
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
              placeholder="至少 8 位密码"
              required
            />
          </div>
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
