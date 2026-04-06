'use client'

import useSWR, { mutate as globalMutate } from 'swr'

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || ''

export type AnnouncementSourceType = 'manual' | 'changelog'

export interface Announcement {
  id: string
  title: string
  summary: string
  content: string
  source_type: AnnouncementSourceType
  source_ref: string
  published_at: string
  created_at: string
  updated_at: string
  read?: boolean
}

interface ApiEnvelope<T> {
  success: boolean
  data?: T
  error?: {
    code?: string
    message?: string
  }
}

export class AnnouncementRequestError extends Error {
  code: string

  constructor(message: string, code?: string) {
    super(message)
    this.name = 'AnnouncementRequestError'
    this.code = code || 'ANNOUNCEMENT_REQUEST_FAILED'
  }
}

async function fetchAnnouncements<T>(url: string): Promise<T> {
  const res = await fetch(url, {
    credentials: 'include',
  })
  const json = await res.json() as ApiEnvelope<T>
  if (!res.ok || !json.success || typeof json.data === 'undefined') {
    throw new AnnouncementRequestError(json.error?.message || 'Announcement request failed', json.error?.code)
  }
  return json.data
}

async function requestAnnouncements<T>(path: string, init?: RequestInit): Promise<T | null> {
  const res = await fetch(`${API_BASE_URL}${path}`, {
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
    ...init,
  })

  const json = await res.json() as ApiEnvelope<T>
  if (!res.ok || !json.success) {
    throw new AnnouncementRequestError(json.error?.message || 'Announcement request failed', json.error?.code)
  }
  return json.data ?? null
}

export function useAnnouncements() {
  const { data, error, isLoading, mutate } = useSWR<Announcement[]>(
    `${API_BASE_URL}/api/v1/announcements`,
    fetchAnnouncements,
    {
      revalidateOnFocus: false,
      dedupingInterval: 1000,
    }
  )

  return {
    announcements: data ?? [],
    error,
    isLoading,
    refresh: mutate,
  }
}

export function useAnnouncement(announcementID: string | null) {
  const { data, error, isLoading, mutate } = useSWR<Announcement>(
    announcementID ? `${API_BASE_URL}/api/v1/announcements/${announcementID}` : null,
    fetchAnnouncements,
    {
      revalidateOnFocus: false,
      dedupingInterval: 1000,
    }
  )

  return {
    announcement: data ?? null,
    error,
    isLoading,
    refresh: mutate,
  }
}

export function useUnreadAnnouncements(enabled: boolean) {
  const { data, error, isLoading, mutate } = useSWR<Announcement[]>(
    enabled ? `${API_BASE_URL}/api/v1/announcements/unread` : null,
    fetchAnnouncements,
    {
      revalidateOnFocus: false,
      dedupingInterval: 1000,
    }
  )

  return {
    unreadAnnouncements: data ?? [],
    error,
    isLoading,
    refresh: mutate,
  }
}

export async function markAnnouncementRead(announcementID: string) {
  const result = await requestAnnouncements(`/api/v1/announcements/${announcementID}/read`, {
    method: 'POST',
  })
  void globalMutate(`${API_BASE_URL}/api/v1/announcements/unread`)
  return result
}

export async function createAnnouncement(input: {
  title: string
  summary: string
  content: string
}) {
  const result = await requestAnnouncements<Announcement>('/api/v1/admin/announcements', {
    method: 'POST',
    body: JSON.stringify(input),
  })
  void globalMutate(`${API_BASE_URL}/api/v1/announcements`)
  return result
}

export async function importAnnouncementsFromChangelog() {
  const result = await requestAnnouncements<{ imported: number }>('/api/v1/admin/announcements/import-changelog', {
    method: 'POST',
  })
  void globalMutate(`${API_BASE_URL}/api/v1/announcements`)
  void globalMutate(`${API_BASE_URL}/api/v1/announcements/unread`)
  return result
}
