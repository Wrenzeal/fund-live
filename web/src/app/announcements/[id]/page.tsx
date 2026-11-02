import type { Metadata } from 'next'
import { notFound } from 'next/navigation'
import { StructuredData } from '@/components/structured-data'
import { getPublicAnnouncement } from '@/lib/public-announcements.server'
import { publicPageMetadata, SITE_URL, isSearchIndexingEnabled } from '@/lib/site-metadata'
import AnnouncementDetailPageClient from './page-client'

export async function generateMetadata({ params }: { params: Promise<{ id: string }> }): Promise<Metadata> {
  const { id } = await params
  const result = await getPublicAnnouncement(id)
  if (result.status !== 'available') return { title: '公告资料暂不可用', alternates: { canonical: null }, robots: { index: false, follow: true } }
  const item = result.data
  const metadata = publicPageMetadata(item.title, item.summary.slice(0, 180) || item.title, `/announcements/${item.id}`)
  return { ...metadata, robots: { index: isSearchIndexingEnabled(), follow: true }, openGraph: { ...metadata.openGraph, type: 'article', publishedTime: item.published_at, modifiedTime: item.updated_at } }
}

export default async function AnnouncementPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params
  const result = await getPublicAnnouncement(id)
  if (result.status === 'missing') notFound()
  const item = result.status === 'available' ? result.data : undefined
  const url = `${SITE_URL}/announcements/${encodeURIComponent(id)}`
  return <>
    {item && <StructuredData id="announcement-structured-data" data={{ '@context': 'https://schema.org', '@type': 'Article', headline: item.title, description: item.summary, datePublished: item.published_at, dateModified: item.updated_at, mainEntityOfPage: url, author: { '@type': 'Organization', name: 'FundLive 项目', url: `${SITE_URL}/about` } }} />}
    <AnnouncementDetailPageClient initialData={item} serverUnavailable={result.status === 'unavailable'} />
  </>
}
