import type { Metadata } from 'next'
import { publicPageMetadata } from '@/lib/site-metadata'

const description = '查看 FundLive 已发布的功能更新和产品公告。'

export const metadata: Metadata = publicPageMetadata('产品更新公告', description, '/announcements')

export default function AnnouncementsLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return children
}
