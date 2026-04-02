'use client'

import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useEffect, useState } from 'react'
import { ArrowRight, Loader2, LockKeyhole, Mail } from 'lucide-react'
import { AuthShell } from '@/components/auth-shell'
import { GoogleSignInButton } from '@/components/google-sign-in-button'
import { loginWithGoogle, loginWithPassword, useCurrentUser } from '@/hooks/use-auth'

export default function LoginPage() {
  const router = useRouter()
  const { user, mutate } = useCurrentUser()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [isGoogleSubmitting, setIsGoogleSubmitting] = useState(false)
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
        await loginWithPassword({ email, password })
        void mutate()
        router.replace('/')
      } catch (err) {
        setError(err instanceof Error ? err.message : '登录失败')
        setIsSubmitting(false)
      }
    })()
  }

  const handleGoogleLogin = async (credential: string) => {
    setError(null)
    setIsGoogleSubmitting(true)

    try {
      await loginWithGoogle(credential)
      void mutate()
      router.replace('/')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Google 登录失败')
    } finally {
      setIsGoogleSubmitting(false)
    }
  }

  return (
    <AuthShell
      eyebrow="密码登录"
      title="登录账户"
      description="先接通邮箱密码登录闭环，后续会在同一入口追加 Google 登录和自动注册。"
      footer={(
        <div className="flex items-center justify-between gap-4">
          <span>还没有账号？</span>
          <Link href="/auth/register" className="font-medium text-cyan-400 transition-colors hover:text-cyan-300">
            去注册
          </Link>
        </div>
      )}
    >
      <form onSubmit={handleSubmit} className="space-y-5">
        <GoogleSignInButton onCredential={handleGoogleLogin} />

        <div className="flex items-center gap-3 text-xs uppercase tracking-[0.25em] text-theme-muted">
          <span className="h-px flex-1 bg-[var(--card-border)]" />
          或继续使用邮箱
          <span className="h-px flex-1 bg-[var(--card-border)]" />
        </div>

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
              autoComplete="current-password"
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
          disabled={isSubmitting || isGoogleSubmitting}
          className="inline-flex w-full items-center justify-center gap-2 rounded-2xl bg-gradient-to-r from-cyan-500 via-sky-500 to-blue-600 px-5 py-3.5 text-sm font-semibold text-white transition-transform hover:-translate-y-0.5 disabled:cursor-not-allowed disabled:opacity-60"
        >
          {isSubmitting || isGoogleSubmitting ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin" />
              {isGoogleSubmitting ? 'Google 登录中...' : '登录中...'}
            </>
          ) : (
            <>
              登录并进入首页
              <ArrowRight className="h-4 w-4" />
            </>
          )}
        </button>
      </form>
    </AuthShell>
  )
}
