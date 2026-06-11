'use client'

import { useEffect, useRef, useState } from 'react'

interface GoogleSignInButtonProps {
  onCredential: (credential: string) => void | Promise<void>
}

interface AuthConfigResponse {
  google_client_id: string
  google_login_enabled: boolean
}

interface ApiEnvelope<T> {
  success: boolean
  data?: T
  error?: {
    code: string
    message: string
  }
}

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || ''
const GOOGLE_SCRIPT_ID = 'google-identity-services'
const GOOGLE_CLIENT_ID = process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID || ''

async function fetchGoogleClientId() {
  const response = await fetch(`${API_BASE_URL}/api/v1/auth/config`, {
    credentials: 'include',
    cache: 'no-store',
  })
  const json = await response.json() as ApiEnvelope<AuthConfigResponse>
  if (!response.ok || !json.success) {
    throw new Error(json.error?.message || 'Google 登录配置读取失败')
  }
  return json.data?.google_client_id?.trim() || ''
}

export function GoogleSignInButton({ onCredential }: GoogleSignInButtonProps) {
  const containerRef = useRef<HTMLDivElement | null>(null)
  const [googleClientId, setGoogleClientId] = useState(GOOGLE_CLIENT_ID)
  const [isLoadingConfig, setIsLoadingConfig] = useState(!GOOGLE_CLIENT_ID)
  const [configError, setConfigError] = useState<string | null>(null)

  useEffect(() => {
    if (GOOGLE_CLIENT_ID) {
      return
    }

    let cancelled = false

    void fetchGoogleClientId()
      .then((clientId) => {
        if (!cancelled) {
          setGoogleClientId(clientId)
        }
      })
      .catch((error) => {
        if (!cancelled) {
          setConfigError(error instanceof Error ? error.message : 'Google 登录配置读取失败')
        }
      })
      .finally(() => {
        if (!cancelled) {
          setIsLoadingConfig(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [])

  useEffect(() => {
    if (!googleClientId || !containerRef.current) {
      return
    }

    const initialize = () => {
      if (!containerRef.current || !window.google?.accounts?.id) {
        return
      }

      const container = containerRef.current
      container.innerHTML = ''

      window.google.accounts.id.initialize({
        client_id: googleClientId,
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
  }, [googleClientId, onCredential])

  if (isLoadingConfig) {
    return (
      <div className="rounded-2xl border border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-4 text-sm text-theme-secondary">
        正在读取 Google 登录配置...
      </div>
    )
  }

  if (!googleClientId) {
    return (
      <div className="rounded-2xl border border-dashed border-[var(--input-border)] bg-[var(--input-bg)] px-4 py-4 text-sm text-theme-secondary">
        {configError || 'Google 登录暂不可用，请先使用邮箱登录。'}
      </div>
    )
  }

  return (
    <div className="space-y-3">
      <div
        ref={containerRef}
        className="flex justify-center"
      />
      <p className="text-xs leading-5 text-theme-muted">
        使用 Google 快速登录，继续查看你的自选和持仓。
      </p>
    </div>
  )
}
