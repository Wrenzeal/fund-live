import { getPublicAnnouncements } from '@/lib/public-announcements.server'
import { isSearchIndexingEnabled } from '@/lib/site-metadata'
import AnnouncementsPageClient from './page-client'

export const revalidate = 300

export async function generateMetadata() {
  const announcements = await getPublicAnnouncements()
  return { robots: { index: isSearchIndexingEnabled() && announcements !== undefined, follow: true } }
}

export default async function AnnouncementsPage() {
  const initialData = await getPublicAnnouncements()
  return <AnnouncementsPageClient initialData={initialData} serverUnavailable={initialData === undefined} />
}
