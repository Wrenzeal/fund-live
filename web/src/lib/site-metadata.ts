import type { Metadata } from 'next'

export const SITE_URL = 'https://fund.wrenzeal.top'
export const SITE_NAME = '涨了多少 · FundLive'
export const SITE_DESCRIPTION = 'FundLive（涨了多少）提供基金盘中估值、自选与持仓管理、公开公告观察和市场研究；区分实时预估与官方净值，并展示数据来源和覆盖范围。页面数据仅供观察，不构成投资建议。'

// Next.js replaces nested metadata objects rather than deeply merging them.
export function publicPageMetadata(title: string, description: string, path: string): Metadata {
  return {
    title: { absolute: `${title} | 涨了多少 FundLive` },
    description,
    alternates: { canonical: path },
    openGraph: {
      title: `${title} | FundLive`,
      description,
      url: path,
      type: 'website',
      siteName: SITE_NAME,
      locale: 'zh_CN',
    },
  }
}
