import type { MetadataRoute } from 'next'
import { SITE_URL, isSearchIndexingEnabled } from '@/lib/site-metadata'
import { getPublicObservationDirectory } from '@/lib/public-observation.server'
import { getPublicAnnouncements } from '@/lib/public-announcements.server'

export const revalidate = 600

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  if (!isSearchIndexingEnabled()) return []
  const [funds, announcements] = await Promise.all([getPublicObservationDirectory(), getPublicAnnouncements()])
  return [
    { url: SITE_URL },
    { url: `${SITE_URL}/observations` },
    { url: `${SITE_URL}/research` },
    { url: `${SITE_URL}/methodology` },
    { url: `${SITE_URL}/about` },
    { url: `${SITE_URL}/announcements` },
    { url: `${SITE_URL}/privacy` },
    { url: `${SITE_URL}/terms` },
    ...funds.map(({ fund }) => ({ url: `${SITE_URL}/observations/${fund.id}` })),
    ...(announcements || []).map(item => ({ url: `${SITE_URL}/announcements/${item.id}`, lastModified: item.updated_at })),
  ]
}
