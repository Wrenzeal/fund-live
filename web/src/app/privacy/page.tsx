import type { Metadata } from 'next'
import { Database, LockKeyhole, ShieldCheck } from 'lucide-react'

import { StaticInfoCardGrid, StaticInfoHero, StaticInfoShell } from '@/components/static-info-page'

export const metadata: Metadata = {
  title: '隐私政策 | FundLive',
  description: '了解 FundLive 为提供基金观察、自选、持仓和账号安全能力而保存和使用的信息。',
}

const sections = [
  {
    title: '我们保存哪些信息',
    icon: Database,
    body: '账户邮箱、显示名称、自选分组、持仓记录、偏好设置和必要的登录会话会用于提供同步、估值观察和个人工作台能力。',
  },
  {
    title: '我们如何使用信息',
    icon: ShieldCheck,
    body: '数据只用于基金搜索、持仓估算、量化观察、报告生成和账号安全校验；页面展示的数据不用于承诺收益或替代个人投资判断。',
  },
  {
    title: '账号与安全',
    icon: LockKeyhole,
    body: '登录会话使用 HttpOnly Cookie。请不要在备注、导入内容或反馈中填写银行卡号、身份证号、真实交易密码等敏感信息。',
  },
]

export default function PrivacyPage() {
  return (
    <StaticInfoShell subtitle="隐私政策">
      <StaticInfoHero
        eyebrow="隐私与数据"
        title="隐私政策"
        description="FundLive 是面向个人基金观察的工具。我们尽量只收集提供功能所必需的数据，并把敏感配置和登录状态与前端展示隔离。"
      />
      <StaticInfoCardGrid cards={sections} />
    </StaticInfoShell>
  )
}
