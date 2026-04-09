'use client'

import useSWR from 'swr'

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || ''

export type IssueType = 'bug' | 'feature' | 'improvement'
export type IssueStatus = 'pending' | 'accepted' | 'completed'

export interface IssueOfficialReply {
  body: string
  replied_by_user_id: string
  replied_by_display_name: string
  created_at: string
  updated_at: string
}

export interface Issue {
  id: string
  title: string
  body: string
  type: IssueType
  status: IssueStatus
  official_reply?: IssueOfficialReply
  created_by_user_id: string
  created_by_display_name: string
  created_at: string
  updated_at: string
}

interface ApiEnvelope<T> {
  success: boolean
  data?: T
  error?: {
    code?: string
    message?: string
  }
}

export class IssueRequestError extends Error {
  code: string

  constructor(message: string, code?: string) {
    super(message)
    this.name = 'IssueRequestError'
    this.code = code || 'ISSUE_REQUEST_FAILED'
  }
}

async function fetchIssues<T>(url: string): Promise<T> {
  const res = await fetch(url, {
    credentials: 'include',
  })
  const json = await res.json() as ApiEnvelope<T>
  if (!res.ok || !json.success || typeof json.data === 'undefined') {
    throw new IssueRequestError(json.error?.message || 'Issue request failed', json.error?.code)
  }
  return json.data
}

async function requestIssues<T>(path: string, init?: RequestInit): Promise<T | null> {
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
    throw new IssueRequestError(json.error?.message || 'Issue request failed', json.error?.code)
  }
  return json.data ?? null
}

export function useIssues(filters: {
  query: string
  type: '' | IssueType
  status: '' | IssueStatus
}) {
  const params = new URLSearchParams()
  if (filters.query.trim()) {
    params.set('q', filters.query.trim())
  }
  if (filters.type) {
    params.set('type', filters.type)
  }
  if (filters.status) {
    params.set('status', filters.status)
  }

  const queryString = params.toString()
  const key = `${API_BASE_URL}/api/v1/issues${queryString ? `?${queryString}` : ''}`
  const { data, error, isLoading, mutate } = useSWR<Issue[]>(key, fetchIssues, {
    revalidateOnFocus: false,
    dedupingInterval: 1000,
  })

  return {
    issues: data ?? [],
    error,
    isLoading,
    refresh: mutate,
  }
}

export function useIssue(issueID: string | null) {
  const { data, error, isLoading, mutate } = useSWR<Issue>(
    issueID ? `${API_BASE_URL}/api/v1/issues/${issueID}` : null,
    fetchIssues,
    {
      revalidateOnFocus: false,
      dedupingInterval: 1000,
    }
  )

  return {
    issue: data ?? null,
    error,
    isLoading,
    refresh: mutate,
  }
}

export async function createIssue(input: {
  title: string
  body: string
  type: IssueType
}) {
  return requestIssues<Issue>('/api/v1/issues', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export async function updateIssueStatus(issueID: string, status: IssueStatus) {
  return requestIssues<Issue>(`/api/v1/admin/issues/${issueID}/status`, {
    method: 'PUT',
    body: JSON.stringify({ status }),
  })
}

export async function updateIssueReply(issueID: string, body: string) {
  return requestIssues<Issue>(`/api/v1/admin/issues/${issueID}/reply`, {
    method: 'PUT',
    body: JSON.stringify({ body }),
  })
}
