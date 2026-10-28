import type { Metadata } from 'next'
import { publicPageMetadata } from '@/lib/site-metadata'

const description =
  '按基金代码查看近期基金公告、已披露持仓范围、报告期、数据来源和覆盖情况。'

export const metadata: Metadata = publicPageMetadata('基金公告与披露观察', description, '/observations')

export default function ObservationsLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return children
}
