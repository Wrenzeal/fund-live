'use client'

import { useEffect, useRef } from 'react'

interface GoogleSignInButtonProps {
  onCredential: (credential: string) => void | Promise<void>
}

const GOOGLE_SCRIPT_ID = 'google-identity-services'
const GOOGLE_CLIENT_ID = process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID || ''

export function GoogleSignInButton({ onCredential }: GoogleSignInButtonProps) {
  const containerRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    if (!GOOGLE_CLIENT_ID || !containerRef.current) {
      return
    }

    const initialize = () => {
      if (!containerRef.current || !window.google?.accounts?.id) {
        return
      }

      const container = containerRef.current
      container.innerHTML = ''

      window.google.accounts.id.initialize({
        client_id: GOOGLE_CLIENT_ID,
        callback: (response) => {
          if (response.credential) {
            void onCredential(response.credential)
          }
        },
      })

      window.google.accounts.id.renderButton(container, {
        type: 'standard',
        theme: 'outline',
        size: 'large',
        shape: 'pill',
        text: 'signin_with',
        width: 320,
        logo_alignment: 'left',
      })
    }

    const existing = document.getElementById(GOOGLE_SCRIPT_ID) as HTMLScriptElement | null
    if (existing) {
      if (window.google?.accounts?.id) {
        initialize()
      } else {
        existing.addEventListener('load', initialize, { once: true })
      }
      return () => {
        existing.removeEventListener('load', initialize)
      }
    }

    const script = document.createElement('script')
    script.id = GOOGLE_SCRIPT_ID
    script.src = 'https://accounts.google.com/gsi/client'
    script.async = true
    script.defer = true
    script.onload = initialize
    document.head.appendChild(script)

    return () => {
      script.onload = null
    }
  }, [onCredential])

  if (!GOOGLE_CLIENT_ID) {
    return (
      <div className="rounded-2xl border border-dashed border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-4 text-sm text-theme-secondary">
        未配置 `NEXT_PUBLIC_GOOGLE_CLIENT_ID`，Google 登录按钮暂不可用。
      </div>
    )
  }

  return (
    <div className="space-y-3">
      <div
        ref={containerRef}
        className="flex min-h-11 items-center justify-center rounded-2xl border border-[var(--input-border)] bg-[var(--card-bg)] px-3 py-2 shadow-[var(--card-shadow)]"
      />
      <p className="text-xs leading-5 text-theme-muted">
        使用 Google 首次登录时，系统会自动创建本地账户并绑定当前 Google 身份。
      </p>
    </div>
  )
}
