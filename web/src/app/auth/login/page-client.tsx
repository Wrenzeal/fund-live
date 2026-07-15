'use client'

import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { ArrowRight, CheckCircle2, KeyRound, Loader2, LockKeyhole, Mail, PencilLine, RefreshCw } from 'lucide-react'
import { AuthShell } from '@/components/auth-shell'
import { GoogleSignInButton } from '@/components/google-sign-in-button'
import {
  AuthApiError,
  loginWithEmailCode,
  loginWithGoogle,
  loginWithPassword,
  startEmailCodeLogin,
  useAuthConfig,
  useCurrentUser,
} from '@/hooks/use-auth'
import { authRouteWithReturnTo } from '@/lib/auth-return-to'
import { cn } from '@/lib/utils'

type LoginMode = 'email_code' | 'password'
type EmailCodeStage = 'email' | 'code'

function secondsRemaining(availableAt: number, now: number) {
  return Math.max(0, Math.ceil((availableAt - now) / 1000))
}

export function LoginPageClient({ returnTo }: { returnTo: string }) {
  const router = useRouter()
  const { user, mutate } = useCurrentUser()
  const { config, error: configError, isLoading: isLoadingConfig } = useAuthConfig()
  const [mode, setMode] = useState<LoginMode>('email_code')
  const [emailStage, setEmailStage] = useState<EmailCodeStage>('email')
  const [email, setEmail] = useState('')
  const [sentEmail, setSentEmail] = useState('')
  const [password, setPassword] = useState('')
  const [code, setCode] = useState('')
  const [devCode, setDevCode] = useState('')
  const [expiresInSeconds, setExpiresInSeconds] = useState(0)
  const [resendAvailableAt, setResendAvailableAt] = useState(0)
  const [clock, setClock] = useState(() => Date.now())
  const [error, setError] = useState<string | null>(null)
  const [availabilityNotice, setAvailabilityNotice] = useState<string | null>(null)
  const [isGoogleSubmitting, setIsGoogleSubmitting] = useState(false)
  const [isSendingCode, setIsSendingCode] = useState(false)
  const [isVerifyingCode, setIsVerifyingCode] = useState(false)
  const [isPasswordSubmitting, setIsPasswordSubmitting] = useState(false)
  const codeInputRef = useRef<HTMLInputElement | null>(null)

  const emailCodeEnabled = Boolean(config?.email_code_login_enabled)
  const resendSeconds = secondsRemaining(resendAvailableAt, clock)
  const isBusy = isGoogleSubmitting || isSendingCode || isVerifyingCode || isPasswordSubmitting
  const registerHref = useMemo(() => authRouteWithReturnTo('/auth/register', returnTo), [returnTo])

  useEffect(() => {
    if (user) {
      router.replace(returnTo)
    }
  }, [returnTo, router, user])

  useEffect(() => {
    if (isLoadingConfig) {
      return
    }
    if (!emailCodeEnabled) {
      setMode('password')
      setAvailabilityNotice('验证码登录暂不可用，已为你切换到密码登录。')
    } else {
      setAvailabilityNotice(null)
    }
  }, [emailCodeEnabled, isLoadingConfig])

  useEffect(() => {
    if (!resendAvailableAt || resendAvailableAt <= Date.now()) {
      return
    }
    const interval = window.setInterval(() => {
      const now = Date.now()
      setClock(now)
      if (now >= resendAvailableAt) {
        window.clearInterval(interval)
      }
    }, 250)
    return () => window.clearInterval(interval)
  }, [resendAvailableAt])

  const completeLogin = useCallback(async () => {
    await mutate()
    router.replace(returnTo)
  }, [mutate, returnTo, router])

  const handlePasswordSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setError(null)
    setIsPasswordSubmitting(true)

    void (async () => {
      try {
        await loginWithPassword({ email, password })
        await completeLogin()
      } catch (err) {
        setError(err instanceof Error ? err.message : '登录失败')
      } finally {
        setIsPasswordSubmitting(false)
      }
    })()
  }

  const sendCode = useCallback(async (targetEmail: string) => {
    setError(null)
    setIsSendingCode(true)
    try {
      const result = await startEmailCodeLogin(targetEmail)
      setEmail(result.email)
      setSentEmail(result.email)
      setCode('')
      setDevCode(result.dev_code || '')
      setExpiresInSeconds(result.expires_in_seconds)
      setResendAvailableAt(Date.now() + result.resend_after_seconds * 1000)
      setClock(Date.now())
      setEmailStage('code')
      window.requestAnimationFrame(() => codeInputRef.current?.focus())
    } catch (err) {
      if (err instanceof AuthApiError && err.code === 'EMAIL_CODE_LOGIN_UNAVAILABLE') {
        setMode('password')
        setAvailabilityNotice('验证码登录暂不可用，已为你切换到密码登录。')
      }
      if (err instanceof AuthApiError && err.retryAfterSeconds) {
        setResendAvailableAt(Date.now() + err.retryAfterSeconds * 1000)
        setClock(Date.now())
      }
      setError(err instanceof Error ? err.message : '验证码发送失败')
    } finally {
      setIsSendingCode(false)
    }
  }, [])

  const handleStartEmailCode = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    void sendCode(email)
  }

  const handleResend = () => {
    if (!sentEmail || resendSeconds > 0 || isSendingCode) {
      return
    }
    void sendCode(sentEmail)
  }

  const handleVerifyCode = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setError(null)
    setIsVerifyingCode(true)

    void (async () => {
      try {
        await loginWithEmailCode(sentEmail, code)
        await completeLogin()
      } catch (err) {
        setError(err instanceof Error ? err.message : '验证码登录失败')
        if (err instanceof AuthApiError && err.code === 'INVALID_VERIFICATION_CODE') {
          setCode('')
          window.requestAnimationFrame(() => codeInputRef.current?.focus())
        }
      } finally {
        setIsVerifyingCode(false)
      }
    })()
  }

  const handleGoogleLogin = useCallback(async (credential: string) => {
    setError(null)
    setIsGoogleSubmitting(true)
    try {
      await loginWithGoogle(credential)
      await completeLogin()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Google 登录失败')
    } finally {
      setIsGoogleSubmitting(false)
    }
  }, [completeLogin])

  const changeMode = (nextMode: LoginMode) => {
    if (nextMode === 'email_code' && !emailCodeEnabled) {
      return
    }
    setMode(nextMode)
    setError(null)
  }

  const changeEmail = () => {
    setEmailStage('email')
    setSentEmail('')
    setCode('')
    setDevCode('')
    setError(null)
  }

  return (
    <AuthShell
      eyebrow="安全登录"
      title="登录账户"
      description="用邮箱验证码、密码或 Google 登录，继续查看已保存的基金和持仓。"
      footer={(
        <div className="space-y-2">
          <div className="flex items-center justify-between gap-4">
            <span>还没有账号？</span>
            <Link href={registerHref} className="font-medium text-cyan-400 transition-colors hover:text-cyan-300">
              使用密码注册
            </Link>
          </div>
          <p className="text-xs leading-5 text-theme-muted">首次完成邮箱验证码验证也会自动创建账户。</p>
        </div>
      )}
    >
      <div className="space-y-5">
        <div className="grid grid-cols-2 gap-2 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] p-1.5" aria-label="选择登录方式">
          <button
            type="button"
            disabled={!emailCodeEnabled || isLoadingConfig || isBusy}
            onClick={() => changeMode('email_code')}
            className={cn(
              'rounded-xl px-3 py-2.5 text-sm font-semibold transition-colors disabled:cursor-not-allowed disabled:opacity-45',
              mode === 'email_code' ? 'bg-cyan-500/15 text-cyan-300' : 'text-theme-secondary hover:text-theme-primary'
            )}
          >
            验证码登录
          </button>
          <button
            type="button"
            disabled={isBusy}
            onClick={() => changeMode('password')}
            className={cn(
              'rounded-xl px-3 py-2.5 text-sm font-semibold transition-colors disabled:cursor-not-allowed disabled:opacity-45',
              mode === 'password' ? 'bg-cyan-500/15 text-cyan-300' : 'text-theme-secondary hover:text-theme-primary'
            )}
          >
            密码登录
          </button>
        </div>

        {isLoadingConfig && (
          <div role="status" className="rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3 text-sm text-theme-secondary">
            正在准备验证码登录…
          </div>
        )}

        {(availabilityNotice || configError) && (
          <div role="status" className="rounded-2xl border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-100">
            {availabilityNotice || (configError instanceof Error ? configError.message : '登录配置读取失败')}
          </div>
        )}

        {mode === 'email_code' && emailStage === 'email' && (
          <form onSubmit={handleStartEmailCode} className="space-y-5">
            <label className="block space-y-2">
              <span className="text-sm text-theme-secondary">邮箱</span>
              <div className="auth-input-shell flex items-center gap-3 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3">
                <Mail className="h-4 w-4 text-theme-muted" />
                <input
                  type="email"
                  autoComplete="email"
                  value={email}
                  onChange={(event) => {
                    setEmail(event.target.value)
                    setError(null)
                  }}
                  className="auth-input w-full bg-transparent text-theme-primary outline-none placeholder:text-theme-muted"
                  placeholder="name@example.com"
                  disabled={isBusy}
                  required
                />
              </div>
            </label>
            <p className="text-sm leading-6 text-theme-muted">验证码 10 分钟内有效；首次验证会自动创建账户。</p>
            <button
              type="submit"
              disabled={isBusy || !emailCodeEnabled}
              className="inline-flex w-full items-center justify-center gap-2 rounded-2xl bg-[var(--accent-primary)] px-5 py-3.5 text-sm font-semibold text-white transition-[filter] hover:brightness-105 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {isSendingCode ? <><Loader2 className="h-4 w-4 animate-spin" /> 正在发送验证码...</> : <><Mail className="h-4 w-4" /> 发送验证码</>}
            </button>
          </form>
        )}

        {mode === 'email_code' && emailStage === 'code' && (
          <div className="space-y-5">
            <div className="rounded-2xl border border-cyan-500/25 bg-cyan-500/10 p-4">
              <div className="flex items-start justify-between gap-4">
                <div className="min-w-0">
                  <p className="inline-flex items-center gap-2 text-sm font-semibold text-cyan-200">
                    <CheckCircle2 className="h-4 w-4" /> 验证码已发送
                  </p>
                  <p className="mt-1 truncate text-xs text-theme-secondary">{sentEmail}</p>
                </div>
                <button
                  type="button"
                  onClick={handleResend}
                  disabled={isBusy || resendSeconds > 0}
                  className="inline-flex shrink-0 items-center gap-1.5 rounded-xl border border-cyan-500/25 bg-[var(--input-bg)] px-3 py-2 text-xs font-semibold text-cyan-200 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {isSendingCode ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RefreshCw className="h-3.5 w-3.5" />}
                  {resendSeconds > 0 ? `${resendSeconds}s 后重发` : '重新发送'}
                </button>
              </div>
              <button
                type="button"
                onClick={changeEmail}
                disabled={isBusy}
                className="mt-3 inline-flex items-center gap-1.5 text-xs font-medium text-theme-muted transition-colors hover:text-theme-primary disabled:opacity-50"
              >
                <PencilLine className="h-3.5 w-3.5" /> 更换邮箱
              </button>
            </div>

            {devCode && (
              <div className="rounded-2xl border border-emerald-500/25 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-100">
                开发验证码：<span className="font-mono font-bold tracking-[0.18em]">{devCode}</span>
              </div>
            )}

            <form onSubmit={handleVerifyCode} className="space-y-5">
              <label className="block space-y-2">
                <span className="text-sm text-theme-secondary">6 位验证码</span>
                <div className="auth-input-shell flex items-center gap-3 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3">
                  <KeyRound className="h-4 w-4 text-theme-muted" />
                  <input
                    ref={codeInputRef}
                    type="text"
                    inputMode="numeric"
                    autoComplete="one-time-code"
                    maxLength={6}
                    value={code}
                    onChange={(event) => {
                      setCode(event.target.value.replace(/\D/g, '').slice(0, 6))
                      setError(null)
                    }}
                    className="auth-input w-full bg-transparent font-mono text-lg tracking-[0.24em] text-theme-primary outline-none placeholder:text-sm placeholder:tracking-normal placeholder:text-theme-muted"
                    placeholder="输入邮件中的验证码"
                    disabled={isBusy}
                    aria-invalid={Boolean(error)}
                    required
                  />
                </div>
              </label>
              <p className="text-sm leading-6 text-theme-muted">验证码 {Math.max(1, Math.ceil(expiresInSeconds / 60))} 分钟内有效，请勿转发。</p>
              <button
                type="submit"
                disabled={isBusy || code.length !== 6}
                className="inline-flex w-full items-center justify-center gap-2 rounded-2xl bg-[var(--accent-primary)] px-5 py-3.5 text-sm font-semibold text-white transition-[filter] hover:brightness-105 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {isVerifyingCode ? <><Loader2 className="h-4 w-4 animate-spin" /> 正在验证...</> : <>验证并登录 <ArrowRight className="h-4 w-4" /></>}
              </button>
            </form>
          </div>
        )}

        {mode === 'password' && (
          <form onSubmit={handlePasswordSubmit} className="space-y-5">
            <label className="block space-y-2">
              <span className="text-sm text-theme-secondary">邮箱</span>
              <div className="auth-input-shell flex items-center gap-3 rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-3">
                <Mail className="h-4 w-4 text-theme-muted" />
                <input
                  type="email"
                  autoComplete="email"
                  value={email}
                  onChange={(event) => {
                    setEmail(event.target.value)
                    setError(null)
                  }}
                  className="auth-input w-full bg-transparent text-theme-primary outline-none placeholder:text-theme-muted"
                  placeholder="name@example.com"
                  disabled={isBusy}
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
                  onChange={(event) => {
                    setPassword(event.target.value)
                    setError(null)
                  }}
                  className="auth-input w-full bg-transparent text-theme-primary outline-none placeholder:text-theme-muted"
                  placeholder="请输入账户密码"
                  disabled={isBusy}
                  required
                />
              </div>
            </label>
            <button
              type="submit"
              disabled={isBusy}
              className="inline-flex w-full items-center justify-center gap-2 rounded-2xl bg-[var(--accent-primary)] px-5 py-3.5 text-sm font-semibold text-white transition-[filter] hover:brightness-105 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {isPasswordSubmitting ? <><Loader2 className="h-4 w-4 animate-spin" /> 登录中...</> : <>登录并继续 <ArrowRight className="h-4 w-4" /></>}
            </button>
          </form>
        )}

        {error && (
          <div role="alert" className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
            {error}
          </div>
        )}

        <div className="flex items-center gap-3 text-sm text-theme-muted">
          <span className="h-px flex-1 bg-[var(--card-border)]" />
          其他登录方式
          <span className="h-px flex-1 bg-[var(--card-border)]" />
        </div>
        <GoogleSignInButton onCredential={handleGoogleLogin} />
      </div>
    </AuthShell>
  )
}
