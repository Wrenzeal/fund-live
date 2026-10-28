import type { Metadata } from 'next'

export async function generateMetadata({ params }: { params: Promise<{ id: string }> }): Promise<Metadata> {
  const { id } = await params
  return {
    title: /^[a-zA-Z0-9_-]{1,80}$/.test(id) ? '产品更新公告详情' : '公告详情',
    alternates: { canonical: null },
    // Client-only details are not indexing-ready until the entity is validated on the server.
    robots: { index: false, follow: true },
  }
}

export default function AnnouncementDetailLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return children
}
