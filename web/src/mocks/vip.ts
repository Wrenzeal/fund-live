export type VIPBillingCycle = 'monthly' | 'yearly'
export type VIPTaskType = 'sector_analysis' | 'portfolio_analysis'
export type VIPTargetType = 'watchlist_group' | 'watchlist_all' | 'holdings_all'
export type VIPTaskStatus = 'queued' | 'running' | 'completed' | 'failed'
export type VIPRiskLevel = 'low' | 'medium' | 'high'
export type VIPSourceType = 'news' | 'policy' | 'earnings' | 'market'

export interface VIPPlan {
  code: 'vip'
  name: string
  subtitle: string
  billingOptions: Array<{
    cycle: VIPBillingCycle
    label: string
    priceLabel: string
    dailyCostLabel: string
    badge?: string
  }>
  highlights: string[]
  rights: string[]
  disclaimer: string
}

export interface VIPMembershipState {
  isVip: boolean
  planCode: 'vip'
  planName: string
  billingCycle: VIPBillingCycle
  activatedAt: string
  expiresAt: string
  usageByDate: Record<string, { sectorAnalysis: number; portfolioAnalysis: number }>
}

export interface VIPTaskRecord {
  id: string
  type: VIPTaskType
  targetType: VIPTargetType
  targetId: string
  targetName: string
  createdAt: string
  templateReportId: string
}

export interface VIPTaskView extends VIPTaskRecord {
  status: VIPTaskStatus
  startedAt?: string
  completedAt?: string
  progressText: string
  reportId?: string
}

export interface VIPReportSource {
  id: string
  title: string
  type: VIPSourceType
  publisher: string
  publishedAt: string
  url: string
  snippet: string
}

export interface VIPAdvice {
  action: '建仓' | '观望' | '低吸' | '减仓' | '止盈'
  positionRange: string
  conditions: string[]
}

export interface VIPNarrativeSection {
  title: string
  content: string
  bullets: string[]
}

export interface VIPEarningsSection {
  title: string
  companies: Array<{
    name: string
    note: string
  }>
}

export interface VIPReport {
  id: string
  type: VIPTaskType
  title: string
  targetName: string
  generatedAt: string
  coverageWindow: string
  riskLevel: VIPRiskLevel
  summary: {
    headline: string
    bullets: string[]
  }
  advice: VIPAdvice
  macro: VIPNarrativeSection
  policy: VIPNarrativeSection
  earnings: VIPEarningsSection
  market: VIPNarrativeSection
  risks: string[]
  sources: VIPReportSource[]
  footerDisclaimer: string
}

export const VIP_PLAN: VIPPlan = {
  code: 'vip',
  name: 'FundLive VIP',
  subtitle: '面向基金投资者的智能投研服务',
  billingOptions: [
    {
      cycle: 'monthly',
      label: '月度 VIP',
      priceLabel: '¥39 / 月',
      dailyCostLabel: '约 ¥1.3 / 天',
    },
    {
      cycle: 'yearly',
      label: '年度 VIP',
      priceLabel: '¥399 / 年',
      dailyCostLabel: '约 ¥1.1 / 天',
      badge: '推荐',
    },
  ],
  highlights: [
    '每交易日 2 次板块分析',
    '每交易日 2 次组合分析',
    '带引用来源的结构化分析报告',
    '异步生成，不阻塞自选与持仓主流程',
  ],
  rights: [
    '从宏观、政策、财报、市场走势四个维度给出整理后的研究结论',
    '对自选分组识别主导板块并生成板块分析',
    '对持仓组合输出结构化结论、风险等级与操作建议',
    '支持在任务中心查看报告生成进度与历史任务状态',
  ],
  disclaimer: '投资有风险，以下能力与报告内容仅供参考，不构成任何投资建议。',
}

export const VIP_DAILY_QUOTA = {
  sectorAnalysis: 2,
  portfolioAnalysis: 2,
} as const

const defaultDisclaimer = '投资有风险，报告内容基于公开信息与模拟数据整理，仅供参考，不构成投资建议。'

export const VIP_SAMPLE_REPORTS: VIPReport[] = [
  {
    id: 'sector-ai-manufacturing',
    type: 'sector_analysis',
    title: 'AI 产业链主导板块分析',
    targetName: '智能制造主题自选分组',
    generatedAt: '2026-04-04T10:35:00+08:00',
    coverageWindow: '近 24 小时市场与公开资讯',
    riskLevel: 'medium',
    summary: {
      headline: 'AI 算力与高端制造链条热度仍在，但短线交易拥挤度提升，适合偏低吸而非追高。',
      bullets: [
        '外围科技资产维持偏强，风险偏好并未明显退潮。',
        '国内政策层面对先进制造和算力基础设施的支持方向仍然清晰。',
        '板块内个股财报分化扩大，强阿尔法公司与普通主题标的拉开差距。',
      ],
    },
    advice: {
      action: '低吸',
      positionRange: '5%-10%',
      conditions: [
        '若板块回撤后量能未明显失真，可分批低吸。',
        '若单日涨幅过大且北向资金流入放缓，优先观望而非追涨。',
      ],
    },
    macro: {
      title: '世界局势与宏观环境',
      content: '全球风险偏好仍围绕科技成长与制造升级展开，海外长端利率对高估值成长资产的压制边际减弱。',
      bullets: [
        '美国核心科技资产维持强势，情绪外溢仍利好本地 AI 产业链。',
        '地缘摩擦尚未出现新的系统性冲击，但仍会影响高端制造出口预期。',
      ],
    },
    policy: {
      title: '政策信息',
      content: '国内政策口径仍偏向“新质生产力、先进制造、算力基础设施”，方向层面没有逆风。',
      bullets: [
        '科技创新和工业升级仍是中期政策主线。',
        '若后续产业扶持细则落地，算力、设备与工业软件板块弹性可能更强。',
      ],
    },
    earnings: {
      title: '财报与公司基本面',
      companies: [
        { name: '中际旭创', note: '订单预期稳定，但短线市场预期已经较高。' },
        { name: '工业富联', note: '算力服务器业务仍是核心观察点，资金偏好较高。' },
        { name: '海光信息', note: '国产算力替代逻辑持续，但估值波动也更大。' },
      ],
    },
    market: {
      title: '大盘与板块走势',
      content: '市场风格仍偏成长，AI 与高端制造仍是高弹性方向，但短线拥挤度需要警惕。',
      bullets: [
        '大盘成交额维持在高位区间，有利于主题方向延续。',
        '板块轮动速度加快，分化中更强调龙头质量而非泛主题覆盖。',
      ],
    },
    risks: [
      '若外围科技股回撤放大，高估值主题会先承压。',
      '若政策层面短期没有增量催化，板块可能进入高位震荡。',
      '若核心公司财报不及预期，板块整体风险偏好会快速降温。',
    ],
    sources: [
      {
        id: 'source-sector-1',
        title: '沪深两市成交额与板块强度日报',
        type: 'market',
        publisher: 'FundLive Market Feed',
        publishedAt: '2026-04-04T10:20:00+08:00',
        url: 'https://example.com/market-daily',
        snippet: '成长风格继续占优，AI 与高端制造链维持高活跃度。',
      },
      {
        id: 'source-sector-2',
        title: '先进制造与算力建设政策跟踪',
        type: 'policy',
        publisher: '公开政策汇编',
        publishedAt: '2026-04-03T20:00:00+08:00',
        url: 'https://example.com/policy-ai',
        snippet: '政策方向延续对先进制造与算力基础设施的支持表述。',
      },
      {
        id: 'source-sector-3',
        title: '算力产业链核心公司季报摘要',
        type: 'earnings',
        publisher: '上市公司公告整理',
        publishedAt: '2026-04-03T18:30:00+08:00',
        url: 'https://example.com/earnings-ai',
        snippet: '板块龙头订单和利润率表现分化，市场更重视兑现能力。',
      },
    ],
    footerDisclaimer: defaultDisclaimer,
  },
  {
    id: 'sector-medical-innovation',
    type: 'sector_analysis',
    title: '医药创新主导板块分析',
    targetName: '医药成长观察分组',
    generatedAt: '2026-04-04T11:05:00+08:00',
    coverageWindow: '近 24 小时公开资讯与板块走势',
    riskLevel: 'medium',
    summary: {
      headline: '医药板块仍处于修复区间，政策与估值具备支撑，但财报兑现决定反弹持续性。',
      bullets: [
        '创新药与器械方向存在资金回流，但节奏偏慢。',
        '行业政策环境较前期更温和，估值压制有所缓解。',
        '板块更适合中期跟踪，短线不宜过度激进。',
      ],
    },
    advice: {
      action: '观望',
      positionRange: '0%-5%',
      conditions: [
        '若后续财报持续改善，可逐步提升跟踪仓位。',
        '若板块出现放量突破，再考虑从观望转向低吸。',
      ],
    },
    macro: {
      title: '世界局势与宏观环境',
      content: '全球风险偏好修复对医药成长有边际利好，但医药仍更多取决于自身产业与政策预期。',
      bullets: [
        '外部宏观压力缓和后，成长行业估值修复空间略有释放。',
      ],
    },
    policy: {
      title: '政策信息',
      content: '政策环境已从极度压缩估值的阶段转向更稳定的观察期，行业情绪修复仍需时间。',
      bullets: [
        '集采与监管预期边际稳定。',
        '创新支持政策仍是中长期重要变量。',
      ],
    },
    earnings: {
      title: '财报与公司基本面',
      companies: [
        { name: '药明康德', note: '订单与外部预期仍是市场焦点。' },
        { name: '迈瑞医疗', note: '稳健基本面仍是板块重要压舱石。' },
        { name: '恒瑞医药', note: '创新药管线兑现将影响估值修复斜率。' },
      ],
    },
    market: {
      title: '大盘与板块走势',
      content: '医药板块处于修复但未完全转强状态，短线更适合等待确认信号。',
      bullets: [
        '板块量能仍弱于高热度科技赛道。',
        '低位反弹后的持续性仍需等待验证。',
      ],
    },
    risks: [
      '若医药财报继续弱于预期，反弹节奏会受阻。',
      '若市场风险偏好重新集中到高弹性赛道，医药可能再度边缘化。',
    ],
    sources: [
      {
        id: 'source-med-1',
        title: '医药板块交易热度追踪',
        type: 'market',
        publisher: 'FundLive Market Feed',
        publishedAt: '2026-04-04T10:50:00+08:00',
        url: 'https://example.com/market-med',
        snippet: '板块回暖但量能尚未充分放大，风格偏修复。',
      },
      {
        id: 'source-med-2',
        title: '创新药政策环境观察',
        type: 'policy',
        publisher: '公开政策汇编',
        publishedAt: '2026-04-03T19:15:00+08:00',
        url: 'https://example.com/policy-med',
        snippet: '行业政策环境趋稳，创新支持方向仍是长期变量。',
      },
    ],
    footerDisclaimer: defaultDisclaimer,
  },
  {
    id: 'portfolio-core-balance',
    type: 'portfolio_analysis',
    title: '核心组合平衡分析报告',
    targetName: '全部持仓组合',
    generatedAt: '2026-04-04T13:45:00+08:00',
    coverageWindow: '近 24 小时市场、财报与公开资讯',
    riskLevel: 'medium',
    summary: {
      headline: '组合当前处于“成长驱动 + 医药修复”并存阶段，适合维持平衡偏积极而非极端押注。',
      bullets: [
        '成长基金贡献弹性，但波动也更大。',
        '医药仓位为组合提供一定风格平衡，但修复节奏仍偏慢。',
        '大盘成交活跃有利于组合维持弹性，但短线仍需警惕高位回撤。',
      ],
    },
    advice: {
      action: '低吸',
      positionRange: '5%-10%',
      conditions: [
        '若市场维持高成交且成长方向未明显退潮，可分批增配。',
        '若核心基金连续快速拉升，则优先等待回撤后再补仓。',
      ],
    },
    macro: {
      title: '世界局势与宏观环境',
      content: '全球风险资产情绪对成长方向依然友好，但组合层面不宜忽视突发风险事件对波动的放大。',
      bullets: [
        '外围成长资产偏强是组合风险偏好的正向变量。',
        '突发地缘与利率波动仍是组合回撤的主要外部触发器。',
      ],
    },
    policy: {
      title: '政策信息',
      content: '国内政策主线仍围绕先进制造、科技创新和稳增长，组合中的成长资产仍具备政策环境支撑。',
      bullets: [
        '科技与制造方向中期逻辑未破坏。',
        '医药和消费方向更多依赖盈利修复与风格轮动。',
      ],
    },
    earnings: {
      title: '财报与公司基本面',
      companies: [
        { name: '工业富联', note: '成长仓位的关键景气风向标。' },
        { name: '药明康德', note: '医药仓位的修复预期核心样本。' },
        { name: '贵州茅台', note: '若组合含消费核心资产，其稳健性有助于平衡成长波动。' },
      ],
    },
    market: {
      title: '大盘与板块走势',
      content: '市场当前偏成长风格，组合层面应接受“收益弹性提升但短线回撤变大”的现实。',
      bullets: [
        '指数层面偏强，但板块分化明显。',
        '组合更适合通过结构平衡而非单赛道重仓提升体验。',
      ],
    },
    risks: [
      '若组合成长仓位过高，会放大日内波动。',
      '若医药修复不及预期，组合平衡作用会弱化。',
      '若市场成交额回落，组合弹性会明显下降。',
    ],
    sources: [
      {
        id: 'source-port-1',
        title: '组合持仓风格拆解',
        type: 'market',
        publisher: 'FundLive Portfolio Feed',
        publishedAt: '2026-04-04T13:20:00+08:00',
        url: 'https://example.com/portfolio-style',
        snippet: '组合当前成长暴露高于防御暴露，医药承担风格平衡作用。',
      },
      {
        id: 'source-port-2',
        title: '宏观与政策环境周观察',
        type: 'policy',
        publisher: '公开政策汇编',
        publishedAt: '2026-04-03T21:00:00+08:00',
        url: 'https://example.com/policy-weekly',
        snippet: '政策仍偏支持科技与制造升级，稳增长环境未发生明显逆转。',
      },
      {
        id: 'source-port-3',
        title: '核心持仓公司财报摘要',
        type: 'earnings',
        publisher: '上市公司公告整理',
        publishedAt: '2026-04-03T18:00:00+08:00',
        url: 'https://example.com/portfolio-earnings',
        snippet: '组合中的核心样本公司基本面分化，景气判断仍需逐项跟踪。',
      },
    ],
    footerDisclaimer: defaultDisclaimer,
  },
]

export const VIP_SAMPLE_REPORT_IDS = {
  defaultSector: 'sector-ai-manufacturing',
  defaultSectorMedical: 'sector-medical-innovation',
  defaultPortfolio: 'portfolio-core-balance',
} as const

export function getVIPSampleReportByID(reportID: string) {
  return VIP_SAMPLE_REPORTS.find((report) => report.id === reportID) ?? null
}
