import type { MetadataRoute } from 'next'
import { SITE_URL } from '@/lib/site-metadata'

export default function sitemap(): MetadataRoute.Sitemap {
  return [
    { url: SITE_URL },
    { url: `${SITE_URL}/observations` },
    { url: `${SITE_URL}/research` },
    { url: `${SITE_URL}/methodology` },
    { url: `${SITE_URL}/announcements` },
    { url: `${SITE_URL}/privacy` },
    { url: `${SITE_URL}/terms` },
  ]
}
