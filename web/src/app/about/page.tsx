import type { Metadata } from 'next'
import Link from 'next/link'

import { StaticInfoHero, StaticInfoShell } from '@/components/static-info-page'
import { Surface } from '@/components/ui/surface'
import { publicPageMetadata } from '@/lib/site-metadata'

export const metadata: Metadata = publicPageMetadata('关于 FundLive 与资料更正', '了解涨了多少 FundLive 的产品范围、开源维护方、数据来源、更正流程和金融信息使用边界。', '/about')

export default function AboutPage() {
  return <StaticInfoShell subtitle="关于 FundLive">
    <div className="mx-auto max-w-4xl space-y-6">
      <StaticInfoHero eyebrow="关于与维护" title="涨了多少 · FundLive" description="FundLive 是开源的基金数据观察项目，整理盘中估值、公开披露和市场研究，帮助使用者核对数字、时间和来源。" />
      <Surface as="section" padding="md">
        <h2 className="text-xl font-semibold text-theme-primary">谁维护，提供什么？</h2>
        <p className="mt-4 text-sm leading-7 text-theme-secondary">项目通过公开源码与变更记录说明计算方法和产品更新，维护入口为 <a href="https://github.com/Wrenzeal/fund-live" className="text-[var(--accent-primary)] underline underline-offset-4">GitHub 项目仓库</a>。本站没有声称由基金公司、交易所或持牌投资顾问运营，也没有把开源身份当作金融资质或数据准确性保证。</p>
        <p className="mt-3 text-sm leading-7 text-theme-secondary">估值、持仓记录和公告阅读各有独立用途，市场研究描述历史条件。本站不代客交易、不托管资金，不提供确定性收益承诺。<Link href="/methodology" className="ml-1 text-[var(--accent-primary)] underline underline-offset-4">方法与数据来源</Link>列出估算公式与使用限制。</p>
      </Surface>
      <Surface as="section" padding="md">
        <h2 className="text-xl font-semibold text-theme-primary">资料如何追溯与更正？</h2>
        <ol className="mt-4 list-decimal space-y-3 pl-5 text-sm leading-7 text-theme-secondary">
          <li>先区分官方净值、盘中预估、持仓报告期和系统读取时间，打开相应条目的来源页面核对。</li>
          <li>通过<Link href="/issues" className="mx-1 text-[var(--accent-primary)] underline underline-offset-4">站内反馈</Link>或<a href="https://github.com/Wrenzeal/fund-live/issues" className="mx-1 text-[var(--accent-primary)] underline underline-offset-4">GitHub Issues</a>提供基金代码、涉及日期、可复现步骤和公开披露链接。公开反馈中不要附账户密码、验证码或完整个人持仓截图。</li>
          <li>维护者依据证据核查后修正数据链路或说明；处理进度以反馈记录和<Link href="/announcements" className="mx-1 text-[var(--accent-primary)] underline underline-offset-4">产品更新公告</Link>为准。来源暂不可用时不会据此断定“没有公告”。</li>
        </ol>
      </Surface>
      <Surface as="section" padding="md">
        <h2 className="text-xl font-semibold text-theme-primary">怎样使用这些资料？</h2>
        <p className="mt-4 text-sm leading-7 text-theme-secondary">引用时保留基金代码、报告期或发布日期以及原始来源链接。公开行情与公告入口的存在，不代表第三方授权任意再分发；具体披露仍以发布主体为准。个人自选和持仓工作区需要登录，不进入公开基金目录。</p>
        <p className="mt-3 text-sm leading-7 text-theme-secondary"><Link href="/privacy" className="text-[var(--accent-primary)] underline underline-offset-4">隐私政策</Link> · <Link href="/terms" className="text-[var(--accent-primary)] underline underline-offset-4">服务条款</Link></p>
      </Surface>
    </div>
  </StaticInfoShell>
}
