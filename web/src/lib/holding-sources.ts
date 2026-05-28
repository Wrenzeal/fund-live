export type HoldingSourcePlatform = 'alipay' | 'wechat' | 'eastmoney' | 'bank' | 'manual' | 'other'
export type HoldingSourceFilter = HoldingSourcePlatform | 'all'

export const HOLDING_SOURCE_OPTIONS: Array<{
  value: HoldingSourcePlatform
  label: string
  description: string
}> = [
  { value: 'alipay', label: '支付宝', description: '按支付宝持仓详情校正' },
  { value: 'wechat', label: '微信', description: '按微信理财通等持仓详情校正' },
  { value: 'eastmoney', label: '天天基金', description: '按天天基金账户口径校正' },
  { value: 'bank', label: '银行 App', description: '按银行代销账户口径校正' },
  { value: 'manual', label: '手工迁移', description: '历史数据搬迁或批量补录' },
  { value: 'other', label: '其他来源', description: '其他平台或人工核对结果' },
]

const SOURCE_LABELS = new Map(HOLDING_SOURCE_OPTIONS.map((option) => [option.value, option.label]))

export function isHoldingSourcePlatform(value?: string): value is HoldingSourcePlatform {
  return Boolean(value && SOURCE_LABELS.has(value as HoldingSourcePlatform))
}

export function resolveHoldingSourceLabel(platform?: string, label?: string) {
  const normalizedLabel = label?.trim()
  if (normalizedLabel) {
    return normalizedLabel
  }
  if (isHoldingSourcePlatform(platform)) {
    return SOURCE_LABELS.get(platform) ?? platform
  }
  return ''
}
