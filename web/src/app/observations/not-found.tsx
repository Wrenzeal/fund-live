import type { Metadata } from 'next'

import { StaticInfoHero, StaticInfoShell } from '@/components/static-info-page'

export const metadata: Metadata = {
  title: '基金页面未找到',
  description: '无法识别此基金代码。',
  alternates: { canonical: null },
  robots: { index: false, follow: true },
  openGraph: { title: '基金页面未找到 | FundLive', url: null },
}

export default function ObservationNotFound() {
  return (
    <StaticInfoShell subtitle="基金观察" backHref="/observations" backLabel="返回基金观察">
      <StaticInfoHero
        eyebrow="404 / 基金未找到"
        title="无法识别此基金代码"
        description="请使用有效的六位基金代码，或返回基金观察入口重新搜索。"
      />
    </StaticInfoShell>
  )
}
