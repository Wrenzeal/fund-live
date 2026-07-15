'use client'

import useSWR from 'swr'
import { API_BASE_URL } from '@/lib/api-base-url'


export type QuoteSource = 'sina' | 'tencent'

export interface AuthUser {
  id: string
  email: string
  display_name: string
  avatar_url: string
  is_admin: boolean
  preferred_quote_source: QuoteSource
  provider: 'password' | 'google' | 'hybrid' | 'email_code'
  email_verified: boolean
  last_login_at?: string
  created_at: string
  updated_at: string
}

export interface AuthSessionData {
  user: AuthUser
  expires_at: string
}

interface ApiEnvelope<T> {
  success: boolean
  data?: T
  error?: {
    code: string
    message: string
    retry_after_seconds?: number
  }
}

export interface PublicAuthConfig {
  google_client_id: string
  google_login_enabled: boolean
  email_code_login_enabled: boolean
}

export interface EmailCodeStartResult {
  email: string
  dev_code?: string
  expires_in_seconds: number
  resend_after_seconds: number
}

export class AuthApiError extends Error {
  code: string
  retryAfterSeconds?: number

  constructor(message: string, code = 'AUTH_REQUEST_FAILED', retryAfterSeconds?: number) {
    super(message)
    this.name = 'AuthApiError'
    this.code = code
    this.retryAfterSeconds = retryAfterSeconds
  }
}

interface PasswordAuthPayload {
  email: string
  password: string
  display_name?: string
}

interface QuoteSourcePreferenceResponse {
  preferred_quote_source: QuoteSource
  effective_quote_source: QuoteSource
}

async function fetchAuth<T>(url: string): Promise<T | null> {
  const res = await fetch(url, {
    credentials: 'include',
  })

  if (res.status === 401) {
    return null
  }

  const json = await res.json() as ApiEnvelope<T>
  if (!res.ok || !json.success) {
    throw new Error(json.error?.message || 'Authentication request failed')
  }

  return json.data ?? null
}

async function postAuth<T>(path: string, payload?: object, method: 'POST' | 'PUT' | 'GET' = 'POST'): Promise<T> {
  const res = await fetch(`${API_BASE_URL}${path}`, {
    method,
    credentials: 'include',
    headers: payload ? { 'Content-Type': 'application/json' } : undefined,
    body: payload ? JSON.stringify(payload) : undefined,
  })

  const json = await res.json() as ApiEnvelope<T>
  if (!res.ok || !json.success || !json.data) {
    throw new AuthApiError(
      json.error?.message || 'Authentication request failed',
      json.error?.code,
      json.error?.retry_after_seconds
    )
  }

  return json.data
}

export function useAuthConfig() {
  const { data, error, isLoading, mutate } = useSWR<PublicAuthConfig | null>(
    `${API_BASE_URL}/api/v1/auth/config`,
    fetchAuth,
    {
      revalidateOnFocus: false,
      shouldRetryOnError: false,
    }
  )

  return {
    config: data,
    isLoading,
    error,
    mutate,
  }
}

export function useCurrentUser() {
  const { data, error, isLoading, mutate } = useSWR<AuthSessionData | null>(
    `${API_BASE_URL}/api/v1/auth/me`,
    fetchAuth,
    {
      revalidateOnFocus: false,
      shouldRetryOnError: false,
    }
  )

  return {
    session: data,
    user: data?.user ?? null,
    expiresAt: data?.expires_at ?? null,
    isLoading,
    isAuthenticated: Boolean(data?.user),
    error,
    mutate,
  }
}

export function registerWithPassword(payload: PasswordAuthPayload) {
  return postAuth<AuthSessionData>('/api/v1/auth/register', payload)
}

export function loginWithPassword(payload: PasswordAuthPayload) {
  return postAuth<AuthSessionData>('/api/v1/auth/login', payload)
}

export function loginWithGoogle(idToken: string) {
  return postAuth<AuthSessionData>('/api/v1/auth/google', {
    id_token: idToken,
  })
}

export function startEmailCodeLogin(email: string) {
  return postAuth<EmailCodeStartResult>('/api/v1/auth/email/start', { email })
}

export function loginWithEmailCode(email: string, code: string) {
  return postAuth<AuthSessionData>('/api/v1/auth/email/verify', { email, code })
}

export async function logout() {
  await fetch(`${API_BASE_URL}/api/v1/auth/logout`, {
    method: 'POST',
    credentials: 'include',
  })
}

export function updateQuoteSourcePreference(quoteSource: QuoteSource) {
  return postAuth<QuoteSourcePreferenceResponse>('/api/v1/user/quote-source', {
    quote_source: quoteSource,
  }, 'PUT')
}
