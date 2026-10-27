# 市场研究工作台第一版

目标：在 FundLive 内提供近期行情、大盘宽度、通信/CPO、半导体与 AI 的走势、技术条件和可复核事件研究。不复用 EasyK 的付费因子或宣传胜率。无需数据库或登录即可使用。Next.js Node.js Route Handler 独立获取数据，可部署于 Vercel，不依赖 Go 服务器。

## API

现有 `APIResponse` 包装 `{success:true,data:T}`；路径 `/api/v1/market/research`。

GET `/overview` → `Overview`:

```ts
type SourceMeta = { source: string; fetched_at: string; stale: boolean; error?: string }
type Instrument = { symbol: string; name: string; group: string }
type Quote = Instrument & { price: number; prev_close: number; open: number; high: number; low: number; change_pct: number; timestamp: string; delayed: boolean }
type Breadth = { advancing: number; declining: number; unchanged: number; turnover_yuan: number }
type News = { title: string; url: string; date: string; related: boolean }
type Overview = { generated_at: string; instruments: Instrument[]; quotes: Quote[]; quote_meta: SourceMeta; breadth: Breadth | null; breadth_meta: SourceMeta; news: News[]; news_meta: SourceMeta; market_phase: string }
```

GET `/instrument?symbol=sz159994` → `InstrumentDetail`:

```ts
type Bar = { date: string; open: number; close: number; high: number; low: number; volume: number }
type Technical = { ma5: number | null; ma20: number | null; ma60: number | null; rsi14: number | null; previous_low: number | null; provisional: boolean; conditions: { label: string; detail: string; tone: string }[] }
type InstrumentDetail = { instrument: Instrument; quote: Quote | null; quote_meta: SourceMeta; bars: Bar[]; history_meta: SourceMeta; technical: Technical | null; minute: { date: string; points: { time: string; price: number }[] } | null; minute_meta: SourceMeta }
```

GET `/study?symbol=sz159994&rule=ma20_up&holding=10&fee_bps=10` → `Study`:

第一版采用固定透明规则，避免未经验证的自由模型：`ma20_up` 收盘价上穿20日均线、`ma20_down` 下穿、`rsi_recovery` RSI14上穿30、`previous_low_break` 收盘价跌破前日低点。只用已完成交易日日线；信号日后下一交易日收盘为进入参考，持有 N 个交易日，排除重叠及未完成样本。向下规则衡量规避下跌的方向正确性，不计算做空收益。模型固定，历史验证不能承诺未来结果。

`drawdown_pct` / `worst_drawdown_pct` 是正数损失幅度，表示持有期间高点至低点的保守回撤上界；日内顺序未知时按先高后低计算，可能高估回撤。UI 使用下跌色且不加正号。

```ts
type StudyEvent = { signal_date: string; entry_date: string; exit_date: string; return_pct: number; direction_pct: number; net_return_pct: number | null; drawdown_pct: number }
type Study = { symbol: string; rule: string; direction: string; holding: number; fee_bps: number; sample_count: number; pending_count: number; direction_win_rate: number | null; mean_return_pct: number | null; mean_net_return_pct: number | null; worst_drawdown_pct: number | null; baseline_mean_return_pct: number | null; events: StudyEvent[]; last_completed_date: string; method: string; history_meta: SourceMeta }
```

只允许白名单标的，保持交易所前缀（上证 sh000001 不能误解析成深市股票）。公开接口超时或返回异常时保留缓存且标注 stale，不造数据。报价时间采用上游时间，刷新时间不能代替行情时间。历史ETF使用前复权；研究避免混用前复权历史价与未复权快照跨除权日形成伪信号。全部最新数据来自公开源，不能保证零延迟。

## 页面与验证

新增 `/research`，顶栏入口“研究”。市场概览、可选择标的、日线/分时图、归一化对比、技术状态、新闻原文入口、事件历史统计与样本明细。无需将用户此前卖出前的金额写入公开仓库；持仓管理继续使用原模块。

不引入新的依赖。单元测试覆盖交易所识别、缺值/陈旧缓存、均线与RSI、未完成日线排除、回测进入时间/去重/费用。前端验证刷新、切换标的、研究参数与移动端排版。交付时本机运行，并给出可复制的启动步骤。


## 可保存的自定义研究

第二阶段增加 `POST /api/v1/market/workspace`，类型见 `web/src/lib/market-workspace-types.ts`。最多3个来源、3条曲线、3个组合条件，支持受限算术与MA/EMA/RSI/REF/ROC/STD/MIN/MAX/ABS。返回共同日期的曲线值、当前组合状态（true/false/null）、新增事件脉冲、来源元数据、历史研究和警告。JSON正文上限32KB，表达式上限1024字符/256词元/24层；历史最多最近640条。相邻完整交易日上验证穿越；持续阈值只在已知假转真触发。研究使用目标原生交易日执行，不把多源缺失日期填充为价格。

`GET /api/v1/market/research/flows` 返回板块资金排名，独立缓存且独立失败。客户端本地方案使用版本1的浏览器存储；不宣称云端同步。完整操作、单位和统计限制见 `docs/easyk-workspace.md`。
