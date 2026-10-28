import type { Metadata } from 'next'
import Link from 'next/link'

import { StaticInfoHero, StaticInfoShell } from '@/components/static-info-page'
import { Surface } from '@/components/ui/surface'
import { publicPageMetadata } from '@/lib/site-metadata'

export const metadata: Metadata = publicPageMetadata(
  '基金估值方法与数据来源',
  '了解 FundLive 如何使用已披露持仓与公开行情估算基金涨跌，覆盖率的含义，以及官方净值、目标 ETF、公告和历史研究的限制。',
  '/methodology',
)

export default function MethodologyPage() {
  return (
    <StaticInfoShell subtitle="方法与数据来源">
      <div className="mx-auto max-w-4xl space-y-6">
        <StaticInfoHero
          eyebrow="方法与数据来源"
          title="基金估值如何计算，数据能说明什么？"
          description="FundLive（涨了多少）是基金数据观察工具。盘中估值是根据公开资料计算的参考值，官方净值是基金管理人公布的结果；两者不相同。"
        />
        <p className="text-sm text-theme-secondary">维护方：FundLive 项目 · 说明更新：<time dateTime="2026-10-28">2026-10-28</time>。此日期是说明文档更新日期，不是行情或持仓披露日期。</p>

        <Surface as="section" aria-labelledby="estimation-method" padding="md">
          <h2 id="estimation-method" className="text-xl font-semibold text-theme-primary">盘中估值和官方净值有什么区别？</h2>
          <div className="mt-4 space-y-3 text-sm leading-7 text-theme-secondary">
            <p>普通持仓估算链路使用已披露持仓及可获取的行情，对参与计算的股票涨跌幅按持仓权重归一化。它不能观察基金经理未披露的调仓，也不能完整反映未披露资产、费用、现金和全部汇率影响。</p>
            <p className="break-words rounded-xl border border-[var(--card-border)] p-4 font-mono text-theme-primary">预估涨跌幅 = Σ（参与计算的持仓权重 × 对应涨跌幅）÷ Σ（参与计算的持仓权重）</p>
            <p>预估净值以系统当前使用的基准净值乘以「1 + 预估涨跌幅」计算。页面的「昨日净值」是估算使用的净值基准，可能因公告时间、节假日或上游更新延迟而早于上一自然日，应核对其实际净值日期。</p>
            <p>单只股票的「贡献」按该股票持仓权重与涨跌幅相乘计算，未经归一化的贡献合计不一定等于最终预估涨跌幅。官方收盘净值公布后独立展示，不把盘中预估改称官方结果。</p>
            <p><Link href="/" className="text-[var(--accent-primary)] underline underline-offset-4">查看基金估值</Link>，或核对<Link href="https://github.com/Wrenzeal/fund-live/blob/main/internal/service/valuation_snapshot.go" className="ml-1 text-[var(--accent-primary)] underline underline-offset-4">当前开源计算实现</Link>。此公式说明普通持仓链路，不代表所有基金使用同一模型。</p>
          </div>
        </Surface>

        <Surface as="section" aria-labelledby="coverage-meaning" padding="md">
          <h2 id="coverage-meaning" className="text-xl font-semibold text-theme-primary">覆盖率高，就代表估值一定准确吗？</h2>
          <div className="mt-4 space-y-3 text-sm leading-7 text-theme-secondary">
            <p>不是。普通股票持仓估算中的覆盖率表示参与计算的已披露持仓权重合计；「10 / 10 只」只是股票数量，不等于覆盖基金全部资产。</p>
            <p>持仓来自历史定期披露。即使披露完整，当前实际持仓仍可能改变；覆盖率不等于准确率、置信概率或收益保证。观察页的「已披露资产占比」描述披露范围，也不能替代盘中行情的可用范围。</p>
            <p>核对数据时应分别查看持仓报告期、行情时间、系统读取时间和净值日期。系统刚读取一条数据，不意味着数据对应的事件刚刚发生。</p>
          </div>
        </Surface>

        <Surface as="section" aria-labelledby="other-fund-types" padding="md">
          <h2 id="other-fund-types" className="text-xl font-semibold text-theme-primary">联接基金、QDII 和特殊基金如何处理？</h2>
          <div className="mt-4 space-y-3 text-sm leading-7 text-theme-secondary">
            <p>联接基金可能使用目标 ETF 的市场行情，或在目标行情不可用时使用目标 ETF 的公开持仓。ETF 场内价格、折溢价和联接基金最终净值并不相同；目标 ETF 的持仓覆盖也不代表联接基金的实时权益仓位。</p>
            <p>QDII 等海外基金还受到市场时区、报价延迟、净值公布时间和汇率变化影响。商品、期货等基金依赖适配的估值配置，不能简单套用股票重仓股公式。缺少可用配置或行情时，系统可能无法给出估值。</p>
          </div>
        </Surface>

        <Surface as="section" aria-labelledby="sources" padding="md">
          <h2 id="sources" className="text-xl font-semibold text-theme-primary">公告和行情数据从哪里来？</h2>
          <dl className="mt-4 space-y-4 text-sm leading-7 text-theme-secondary">
            <div><dt className="font-semibold text-theme-primary">公开资料与行情</dt><dd>基金资料与持仓依赖公开数据和缓存快照；股票行情可使用新浪、腾讯等已接入来源，具体以页面标注为准。来源可用性、字段更新和海外覆盖可能变化。</dd></div>
            <div><dt className="font-semibold text-theme-primary">基金公告</dt><dd>基金公告读取东方财富公开公告入口，部分沪深持仓公告读取巨潮资讯。页面保留标题、发布日期和来源链接；点击后可能进入文档或公告列表页。最终请核对基金公司或上市公司的正式披露。</dd></div>
            <div><dt className="font-semibold text-theme-primary">市场研究</dt><dd>市场行情与日线主要读取腾讯公开数据；市场宽度、板块资金和消息入口读取东方财富公开数据。板块资金是数据商按成交分类估算的结果，板块可能重叠，不代表机构实际持仓或 ETF 份额净申购。</dd></div>
          </dl>
          <p className="mt-4 text-sm leading-7 text-theme-secondary"><a href="https://fund.eastmoney.com/" className="text-[var(--accent-primary)] underline underline-offset-4">东方财富基金资料入口</a> · <a href="https://www.cninfo.com.cn/" className="text-[var(--accent-primary)] underline underline-offset-4">巨潮资讯披露入口</a></p>
        </Surface>

        <Surface as="section" aria-labelledby="notice-limitations" padding="md">
          <h2 id="notice-limitations" className="text-xl font-semibold text-theme-primary">没有公告，或者历史方向正确率较高，意味着什么？</h2>
          <div className="mt-4 space-y-3 text-sm leading-7 text-theme-secondary">
            <p><Link href="/observations" className="text-[var(--accent-primary)] underline underline-offset-4">基金观察</Link>展示最近 45 天内、当前来源可返回的公告及部分已披露持仓公告，不是全量新闻搜索。「暂无公告」只描述已成功读取的范围；来源暂不可用、部分可用和没有关联来源是不同状态，不能互相替代。</p>
            <p><Link href="/research" className="text-[var(--accent-primary)] underline underline-offset-4">市场研究</Link>中的技术条件和事件统计描述历史价格，不代表未来获利概率。样本重叠、研究区间、费用设置和日线数据限制都会影响统计，ETF 市场价格结果也不等于场外基金可实现收益。</p>
            <p>本产品不提供综合投资评分、收益承诺或自动交易。发现数据问题时，请附上基金代码、涉及日期和官方来源，通过<Link href="/issues" className="mx-1 text-[var(--accent-primary)] underline underline-offset-4">反馈入口</Link>提交，避免提供账户密码或其他敏感信息。</p>
          </div>
        </Surface>
      </div>
    </StaticInfoShell>
  )
}
