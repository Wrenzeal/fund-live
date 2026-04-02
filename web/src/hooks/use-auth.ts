'use client'

import useSWR from 'swr'

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || ''

export interface AuthUser {
  id: string
  email: string
  display_name: string
  avatar_url: string
  provider: 'password' | 'google' | 'hybrid'
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
  }
}

interface PasswordAuthPayload {
  email: string
  password: string
  display_name?: string
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

async function postAuth<T>(path: string, payload?: object, method: 'POST' | 'GET' = 'POST'): Promise<T> {
  const res = await fetch(`${API_BASE_URL}${path}`, {
    method,
    credentials: 'include',
    headers: payload ? { 'Content-Type': 'application/json' } : undefined,
    body: payload ? JSON.stringify(payload) : undefined,
  })

  const json = await res.json() as ApiEnvelope<T>
  if (!res.ok || !json.success || !json.data) {
    throw new Error(json.error?.message || 'Authentication request failed')
  }

  return json.data
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

export async function logout() {
  await fetch(`${API_BASE_URL}/api/v1/auth/logout`, {
    method: 'POST',
    credentials: 'include',
  })
}
