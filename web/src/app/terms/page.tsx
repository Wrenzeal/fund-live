import type { Metadata } from 'next'
import { FileText, Scale, TriangleAlert } from 'lucide-react'

import { StaticInfoCardGrid, StaticInfoHero, StaticInfoShell } from '@/components/static-info-page'

export const metadata: Metadata = {
  title: '服务条款 | FundLive',
  description: '了解 FundLive 作为基金数据观察工具的使用边界、数据限制和账号使用规则。',
}

const sections = [
  {
    title: '仅作数据观察',
    icon: TriangleAlert,
    body: '估值、量化分数、报告摘要和风险提示都只是基于公开数据与本地规则整理的观察结果，不构成投资建议。',
  },
  {
    title: '数据可能延迟或缺失',
    icon: FileText,
    body: '基金净值、持仓、公告、行情和海外数据可能存在延迟、缓存、缺失或来源不可用，最终结果应以官方披露为准。',
  },
  {
    title: '合理使用账号',
    icon: Scale,
    body: '请勿批量抓取、绕过限制、攻击接口、提交违法内容或把本服务生成的内容包装成确定性收益承诺。',
  },
]

export default function TermsPage() {
  return (
    <StaticInfoShell subtitle="服务条款">
      <StaticInfoHero
        eyebrow="使用边界"
        title="服务条款"
        description="使用 FundLive 即表示你理解本产品是数据整理与个人观察工具。你仍需要独立判断风险，并以官方披露和自身情况为准。"
      />
      <StaticInfoCardGrid cards={sections} />
    </StaticInfoShell>
  )
}
