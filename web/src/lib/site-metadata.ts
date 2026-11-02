import type { Metadata } from 'next'

export const SITE_URL = 'https://fund.wrenzeal.top'
export const SITE_NAME = '涨了多少 · FundLive'
export const SITE_DESCRIPTION = 'FundLive（涨了多少）提供基金盘中估值、自选与持仓管理、公开公告观察和市场研究；区分实时预估与官方净值，并展示数据来源和覆盖范围。页面数据仅供观察，不构成投资建议。'

export function isSearchIndexingEnabled() {
  return process.env.SEO_NOINDEX !== 'true' && (!process.env.VERCEL_ENV || process.env.VERCEL_ENV === 'production')
}

export const SOCIAL_IMAGE = { url: '/opengraph-image', width: 1200, height: 630, alt: 'FundLive — fund estimates, disclosures and research' }

// Next.js replaces nested metadata objects rather than deeply merging them.
export function publicPageMetadata(title: string, description: string, path: string): Metadata {
  return {
    title: { absolute: `${title} | 涨了多少 FundLive` },
    description,
    alternates: { canonical: path },
    robots: { index: isSearchIndexingEnabled(), follow: true },
    openGraph: {
      title: `${title} | FundLive`,
      description,
      url: path,
      type: 'website',
      siteName: SITE_NAME,
      locale: 'zh_CN',
      images: [SOCIAL_IMAGE],
    },
    twitter: { card: 'summary_large_image', title: `${title} | FundLive`, description, images: [SOCIAL_IMAGE.url] },
  }
}
