import type { Metadata } from 'next'
import { notFound } from 'next/navigation'
import { publicPageMetadata } from '@/lib/site-metadata'
import { FundObservationPageClient } from './page-client'

export async function generateMetadata({ params }: { params: Promise<{ fundId: string }> }): Promise<Metadata> {
  const { fundId } = await params
  if (!/^\d{6}$/.test(fundId)) {
    return {
      title: '基金页面未找到',
      description: '无法识别此基金代码。',
      alternates: { canonical: null },
      robots: { index: false, follow: true },
      openGraph: { title: '基金页面未找到 | FundLive', url: null },
    }
  }

  const title = `${fundId} 基金公告与持仓披露`
  const description = `查看基金 ${fundId} 的近期公告、已披露持仓范围、报告期和来源状态。估值与持仓数据仅供观察，不构成投资建议。`

  return {
    ...publicPageMetadata(title, description, `/observations/${fundId}`),
    // Reopen indexing when a server-rendered, entity-validated snapshot exists.
    robots: { index: false, follow: true },
  }
}

export default async function FundObservationPage({ params }: { params: Promise<{ fundId: string }> }) {
  const { fundId } = await params
  if (!/^\d{6}$/.test(fundId)) notFound()
  return <FundObservationPageClient key={fundId} fundId={fundId} />
}
