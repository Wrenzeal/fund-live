'use client'

import { useMarketResearchFlows } from '@/hooks/use-market-research'
import { RefreshCw } from 'lucide-react'

export function MarketSectorFlows({ phase }: { phase?: string }) {
  const flows = useMarketResearchFlows(phase)
  const rows = flows.data?.rows || []
  const inflows = rows.filter(row => row.net_inflow_yuan > 0).slice(0, 6)
  const outflows = rows.filter(row => row.net_inflow_yuan < 0).sort((a, b) => a.net_inflow_yuan - b.net_inflow_yuan).slice(0, 6)
  const formatTime = (value: string) => new Date(value).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false })
  return <section className="border-t border-[var(--card-border)] py-6" aria-label="板块资金观察">
    <div className="mb-4 flex items-center justify-between gap-3"><h2 className="text-base font-semibold text-theme-primary">板块资金观察</h2><button type="button" aria-label="刷新板块资金" disabled={flows.isValidating} onClick={() => void flows.mutate()} className="rounded-md border border-[var(--input-border)] p-2 text-theme-secondary"><RefreshCw className={`h-4 w-4 ${flows.isValidating ? 'animate-spin' : ''}`} /></button></div>
    {flows.error && <p role="alert" className="mb-3 text-sm text-amber-500">{flows.error.message}{flows.data && '；保留上次成功数据'}</p>}
    {flows.isLoading && <p role="status" className="py-4 text-sm text-theme-muted">正在读取板块资金…</p>}
    {!flows.isLoading && !rows.length && <p className="py-4 text-sm text-theme-muted">板块资金暂不可用</p>}
    {!!rows.length && <div className="grid gap-5 md:grid-cols-2">{[{ title: '净流入居前', values: inflows }, { title: '净流出居前', values: outflows }].map(group => <div key={group.title}><h3 className="mb-2 text-sm font-medium text-theme-secondary">{group.title}</h3><table className="w-full text-xs"><thead className="text-theme-muted"><tr><th className="py-2 text-left font-normal">板块</th><th className="text-right font-normal">涨跌</th><th className="text-right font-normal">净流入（亿元）</th></tr></thead><tbody>{group.values.map(row => <tr key={row.code} className="border-t border-[var(--card-border)]"><td className="py-3 text-theme-secondary">{row.name}<span className="mt-1 block text-[10px] text-theme-muted">数据 {formatTime(row.timestamp)}</span></td><td className={`text-right tabular-nums ${row.change_pct > 0 ? 'text-up' : row.change_pct < 0 ? 'text-down' : 'text-theme-muted'}`}>{row.change_pct > 0 ? '+' : ''}{row.change_pct.toFixed(2)}%</td><td className={`text-right tabular-nums ${row.net_inflow_yuan > 0 ? 'text-up' : 'text-down'}`}>{row.net_inflow_yuan > 0 ? '+' : ''}{(row.net_inflow_yuan / 1e8).toFixed(2)}</td></tr>)}</tbody></table></div>)}</div>}
    {flows.data && <div className="mt-3 space-y-1 text-xs leading-5 text-theme-muted"><p>{flows.data.meta.source} · 读取 {flows.data.meta.fetched_at ? formatTime(flows.data.meta.fetched_at) : '--'}{flows.data.meta.stale && <span className="ml-2 text-amber-500">缓存已过期</span>}</p>{flows.data.meta.error && <p className="text-amber-500">{flows.data.meta.error}</p>}<p>{flows.data.method}</p></div>}
  </section>
}
