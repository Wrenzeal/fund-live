import type { Metadata } from 'next'
import { publicPageMetadata } from '@/lib/site-metadata'

const description =
  '查看公开市场指数、基金与 ETF 行情、技术指标和历史事件研究。行情可能延迟；历史条件不代表未来收益。'

export const metadata: Metadata = publicPageMetadata('市场行情与历史研究', description, '/research')

export default function ResearchLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return children
}
