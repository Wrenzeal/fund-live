import type { Metadata } from 'next'
import { notFound } from 'next/navigation'
import { publicPageMetadata, SITE_URL, isSearchIndexingEnabled } from '@/lib/site-metadata'
import { getPublicObservation } from '@/lib/public-observation.server'
import { StructuredData } from '@/components/structured-data'
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

  const result = await getPublicObservation(fundId)
  const name = result.status === 'available' ? result.data.fund.name : `基金 ${fundId}`
  const title = `${name}（${fundId}）公告与持仓披露`
  const description = result.status === 'available'
    ? `查看${name}（${fundId}）近 ${result.data.observation.window_days} 天的公开公告、持仓报告期和已披露资产占比；标注来源及资料限制，不提供买卖建议。`
    : `基金 ${fundId} 的观察资料暂不可用，请稍后重试。`

  return {
    ...publicPageMetadata(title, description, `/observations/${fundId}`),
    robots: { index: isSearchIndexingEnabled() && result.status === 'available' && result.indexable, follow: true },
  }
}

export default async function FundObservationPage({ params }: { params: Promise<{ fundId: string }> }) {
  const { fundId } = await params
  if (!/^\d{6}$/.test(fundId)) notFound()
  const result = await getPublicObservation(fundId)
  if (result.status === 'missing') notFound()
  const data = result.status === 'available' ? result.data : undefined
  const url = `${SITE_URL}/observations/${fundId}`
  return <>
    {data && <StructuredData id="observation-structured-data" data={{
      '@context': 'https://schema.org', '@graph': [
        { '@type': 'WebPage', '@id': `${url}#page`, url, name: `${data.fund.name}（${fundId}）公告与持仓披露`, inLanguage: 'zh-CN', isPartOf: { '@id': `${SITE_URL}/#website` },
          description: '公开基金与部分已披露持仓的公告来源整理，非实时全量持仓或投资建议。',
          about: { '@type': 'Thing', name: data.fund.name, identifier: fundId },
          citation: [new URL('/methodology', SITE_URL).href, ...new Set(data.observation.notices.filter(notice => notice.source_url).map(notice => notice.source_url))],
        },
        { '@type': 'BreadcrumbList', itemListElement: [
          { '@type': 'ListItem', position: 1, name: '首页', item: SITE_URL },
          { '@type': 'ListItem', position: 2, name: '基金观察', item: `${SITE_URL}/observations` },
          { '@type': 'ListItem', position: 3, name: data.fund.name, item: url },
        ] },
      ],
    }} />}
    <FundObservationPageClient key={fundId} fundId={fundId} initialData={data} serverUnavailable={result.status === 'unavailable'} />
  </>
}
