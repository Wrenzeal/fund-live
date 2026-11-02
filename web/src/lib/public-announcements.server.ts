import 'server-only'
import { cache } from 'react'
import { PRODUCTION_API_BASE_URL, resolveBackendBaseUrl } from '@/lib/api-base-url'
import type { Announcement } from '@/lib/announcement'

function record(value: unknown): value is Record<string, unknown> { return typeof value === 'object' && value !== null && !Array.isArray(value) }
function validDate(value: unknown) { return typeof value === 'string' && /^\d{4}-\d{2}-\d{2}T/.test(value) && Number.isFinite(new Date(value).getTime()) }
function announcement(value: unknown): Announcement | undefined {
  if (!record(value) || typeof value.id !== 'string' || !/^[a-zA-Z0-9_-]{1,100}$/.test(value.id)
    || typeof value.title !== 'string' || !value.title.trim() || value.title.length > 500
    || typeof value.summary !== 'string' || typeof value.content !== 'string' || !value.content.trim() || value.content.length > 500000
    || !['manual', 'changelog'].includes(String(value.source_type)) || typeof value.source_ref !== 'string'
    || !validDate(value.published_at) || !validDate(value.updated_at) || !validDate(value.created_at)
    || new Date(String(value.published_at)).getTime() > Date.now() + 60000) return
  // Explicitly omit user-specific read state from shared anonymous caches.
  return { id: value.id, title: value.title.trim(), summary: value.summary, content: value.content, source_type: value.source_type as Announcement['source_type'], source_ref: value.source_ref, published_at: String(value.published_at), created_at: String(value.created_at), updated_at: String(value.updated_at) }
}

async function fetchPublic(path: string) {
  const base = resolveBackendBaseUrl(process.env.BACKEND_URL, process.env.VERCEL ? PRODUCTION_API_BASE_URL : 'http://127.0.0.1:8080')
  try {
    const response = await fetch(`${base}/api/v1/announcements${path}`, { headers: { Accept: 'application/json' }, credentials: 'omit', redirect: 'error', signal: AbortSignal.timeout(5000), next: { revalidate: 300 } })
    const value: unknown = await response.json()
    if (!record(value)) return { status: 'unavailable' as const }
    if (response.status === 404 && value.success === false && record(value.error) && value.error.code === 'ANNOUNCEMENT_NOT_FOUND') return { status: 'missing' as const }
    return response.ok && value.success === true ? { status: 'available' as const, data: value.data } : { status: 'unavailable' as const }
  } catch { return { status: 'unavailable' as const } }
}

export const getPublicAnnouncements = cache(async () => {
  const result = await fetchPublic('')
  if (result.status !== 'available' || !Array.isArray(result.data)) return undefined
  return result.data.map(announcement).filter((item): item is Announcement => Boolean(item)).sort((a, b) => b.published_at.localeCompare(a.published_at)).slice(0, 100)
})

export const getPublicAnnouncement = cache(async (id: string) => {
  if (!/^[a-zA-Z0-9_-]{1,100}$/.test(id)) return { status: 'missing' as const }
  const result = await fetchPublic(`/${encodeURIComponent(id)}`)
  if (result.status !== 'available') return result
  const data = announcement(result.data)
  return data?.id === id ? { status: 'available' as const, data } : { status: 'unavailable' as const }
})
